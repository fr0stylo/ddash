package renderer

import (
	"container/list"
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/a-h/templ"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
)

type FragmentRenderer struct {
	mu       sync.Mutex
	capacity int
	ttl      time.Duration
	ll       *list.List
	entries  map[string]*list.Element
	metrics  fragmentRendererMetrics
}

type fragmentRendererMetrics struct {
	requests  metric.Int64Counter
	hits      metric.Int64Counter
	misses    metric.Int64Counter
	rerenders metric.Int64Counter
	bypasses  metric.Int64Counter
	errors    metric.Int64Counter
}

func newFragmentRendererMetrics() fragmentRendererMetrics {
	meter := otel.Meter("github.com/fr0stylo/ddash/apps/ddash/internal/renderer")
	requests, _ := meter.Int64Counter("ddash.fragment_renderer.requests")
	hits, _ := meter.Int64Counter("ddash.fragment_renderer.hits")
	misses, _ := meter.Int64Counter("ddash.fragment_renderer.misses")
	rerenders, _ := meter.Int64Counter("ddash.fragment_renderer.rerenders")
	bypasses, _ := meter.Int64Counter("ddash.fragment_renderer.bypasses")
	errors, _ := meter.Int64Counter("ddash.fragment_renderer.errors")
	return fragmentRendererMetrics{
		requests:  requests,
		hits:      hits,
		misses:    misses,
		rerenders: rerenders,
		bypasses:  bypasses,
		errors:    errors,
	}
}

type fragmentEntry struct {
	cacheKey  string
	body      []byte
	expiresAt time.Time
}

func NewFragmentRenderer(capacity int, ttl time.Duration) *FragmentRenderer {
	if capacity <= 0 {
		capacity = 512
	}
	if ttl <= 0 {
		ttl = 5 * time.Second
	}
	return &FragmentRenderer{
		capacity: capacity,
		ttl:      ttl,
		ll:       list.New(),
		entries:  make(map[string]*list.Element, capacity),
		metrics:  newFragmentRendererMetrics(),
	}
}

func (r *FragmentRenderer) RenderCached(ctx context.Context, key string, version int64, build func() (templ.Component, error), attrs ...attribute.KeyValue) ([]byte, bool, error) {
	r.metrics.requests.Add(ctx, 1, metric.WithAttributes(attrs...))
	cacheKey := fmt.Sprintf("%s|v=%d", key, version)
	return r.renderCachedByKey(ctx, cacheKey, build, attrs...)
}

// RenderCachedTTL caches by logical key only; no version lookup is performed.
func (r *FragmentRenderer) RenderCachedTTL(ctx context.Context, key string, build func() (templ.Component, error), attrs ...attribute.KeyValue) ([]byte, bool, error) {
	r.metrics.requests.Add(ctx, 1, metric.WithAttributes(attrs...))
	cacheKey := fmt.Sprintf("%s|ttl", key)
	return r.renderCachedByKey(ctx, cacheKey, build, attrs...)
}

func (r *FragmentRenderer) TryGetTTL(ctx context.Context, key string, attrs ...attribute.KeyValue) ([]byte, bool) {
	r.metrics.requests.Add(ctx, 1, metric.WithAttributes(attrs...))
	cacheKey := fmt.Sprintf("%s|ttl", key)
	body, ok := r.get(cacheKey)
	if ok {
		r.metrics.hits.Add(ctx, 1, metric.WithAttributes(attrs...))
		return body, true
	}
	r.metrics.misses.Add(ctx, 1, metric.WithAttributes(attrs...))
	return nil, false
}

func (r *FragmentRenderer) StoreTTL(ctx context.Context, key string, body []byte, attrs ...attribute.KeyValue) {
	if r == nil || key == "" || len(body) == 0 {
		return
	}
	r.metrics.rerenders.Add(ctx, 1, metric.WithAttributes(attrs...))
	cacheKey := fmt.Sprintf("%s|ttl", key)
	r.set(cacheKey, body)
}

func (r *FragmentRenderer) renderCachedByKey(ctx context.Context, cacheKey string, build func() (templ.Component, error), attrs ...attribute.KeyValue) ([]byte, bool, error) {
	if body, ok := r.get(cacheKey); ok {
		r.metrics.hits.Add(ctx, 1, metric.WithAttributes(attrs...))
		return body, true, nil
	}
	r.metrics.misses.Add(ctx, 1, metric.WithAttributes(attrs...))
	r.metrics.rerenders.Add(ctx, 1, metric.WithAttributes(attrs...))

	component, err := build()
	if err != nil {
		r.metrics.errors.Add(ctx, 1, metric.WithAttributes(attrs...))
		return nil, false, err
	}
	body, err := RenderComponent(ctx, component)
	if err != nil {
		r.metrics.errors.Add(ctx, 1, metric.WithAttributes(attrs...))
		return nil, false, err
	}
	r.set(cacheKey, body)
	return body, false, nil
}

func (r *FragmentRenderer) RecordBypass(ctx context.Context, attrs ...attribute.KeyValue) {
	if r == nil {
		return
	}
	r.metrics.bypasses.Add(ctx, 1, metric.WithAttributes(attrs...))
}

func (r *FragmentRenderer) RecordError(ctx context.Context, attrs ...attribute.KeyValue) {
	if r == nil {
		return
	}
	r.metrics.errors.Add(ctx, 1, metric.WithAttributes(attrs...))
}

func (r *FragmentRenderer) get(cacheKey string) ([]byte, bool) {
	r.mu.Lock()
	defer r.mu.Unlock()

	element, ok := r.entries[cacheKey]
	if !ok {
		return nil, false
	}
	entry := element.Value.(*fragmentEntry)
	if time.Now().After(entry.expiresAt) {
		r.ll.Remove(element)
		delete(r.entries, cacheKey)
		return nil, false
	}
	r.ll.MoveToFront(element)
	copyBody := append([]byte(nil), entry.body...)
	return copyBody, true
}

func (r *FragmentRenderer) set(cacheKey string, body []byte) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if element, ok := r.entries[cacheKey]; ok {
		entry := element.Value.(*fragmentEntry)
		entry.body = append(entry.body[:0], body...)
		entry.expiresAt = time.Now().Add(r.ttl)
		r.ll.MoveToFront(element)
		return
	}

	entry := &fragmentEntry{
		cacheKey:  cacheKey,
		body:      append([]byte(nil), body...),
		expiresAt: time.Now().Add(r.ttl),
	}
	element := r.ll.PushFront(entry)
	r.entries[cacheKey] = element

	if r.ll.Len() > r.capacity {
		tail := r.ll.Back()
		if tail == nil {
			return
		}
		r.ll.Remove(tail)
		victim := tail.Value.(*fragmentEntry)
		delete(r.entries, victim.cacheKey)
	}
}
