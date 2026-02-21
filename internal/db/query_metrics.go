package db

import (
	"context"
	"database/sql"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/fr0stylo/ddash/internal/db/queries"
	"github.com/fr0stylo/ddash/internal/observability"
)

const maxSamplesPerQuery = 512

type queryLatencyStats struct {
	Name  string
	Count int
	P50   time.Duration
	P95   time.Duration
	Max   time.Duration
}

type queryLatencyTracker struct {
	mu      sync.Mutex
	samples map[string][]time.Duration
}

func newQueryLatencyTracker() *queryLatencyTracker {
	return &queryLatencyTracker{samples: make(map[string][]time.Duration)}
}

func (t *queryLatencyTracker) observe(name string, duration time.Duration) {
	if t == nil {
		return
	}
	name = strings.TrimSpace(name)
	if name == "" {
		name = "unknown"
	}

	t.mu.Lock()
	defer t.mu.Unlock()

	window := append(t.samples[name], duration)
	if len(window) > maxSamplesPerQuery {
		window = window[len(window)-maxSamplesPerQuery:]
	}
	t.samples[name] = window
}

func (t *queryLatencyTracker) snapshot() []queryLatencyStats {
	if t == nil {
		return nil
	}

	t.mu.Lock()
	defer t.mu.Unlock()

	stats := make([]queryLatencyStats, 0, len(t.samples))
	for name, durations := range t.samples {
		if len(durations) == 0 {
			continue
		}
		sorted := make([]time.Duration, len(durations))
		copy(sorted, durations)
		sort.Slice(sorted, func(i, j int) bool { return sorted[i] < sorted[j] })

		stats = append(stats, queryLatencyStats{
			Name:  name,
			Count: len(sorted),
			P50:   sorted[(len(sorted)-1)/2],
			P95:   sorted[int(float64(len(sorted)-1)*0.95)],
			Max:   sorted[len(sorted)-1],
		})
	}

	sort.Slice(stats, func(i, j int) bool {
		if stats[i].P95 == stats[j].P95 {
			return stats[i].Name < stats[j].Name
		}
		return stats[i].P95 > stats[j].P95
	})

	return stats
}

type instrumentedDBTX struct {
	inner   queries.DBTX
	tracker *queryLatencyTracker
}

func newInstrumentedDBTX(inner queries.DBTX, tracker *queryLatencyTracker) queries.DBTX {
	if tracker == nil {
		return inner
	}
	return &instrumentedDBTX{inner: inner, tracker: tracker}
}

func (d *instrumentedDBTX) ExecContext(ctx context.Context, query string, args ...interface{}) (sql.Result, error) {
	name := queryName(query)
	ctx, span := observability.StartDBSpan(ctx, name, "exec")
	defer span.End()

	start := time.Now()
	result, err := d.inner.ExecContext(ctx, query, args...)
	d.tracker.observe(name, time.Since(start))
	span.RecordError(err)
	return result, err
}

func (d *instrumentedDBTX) PrepareContext(ctx context.Context, query string) (*sql.Stmt, error) {
	name := queryName(query)
	ctx, span := observability.StartDBSpan(ctx, name, "prepare")
	defer span.End()

	start := time.Now()
	stmt, err := d.inner.PrepareContext(ctx, query)
	d.tracker.observe(name, time.Since(start))
	span.RecordError(err)
	return stmt, err
}

func (d *instrumentedDBTX) QueryContext(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error) {
	name := queryName(query)
	ctx, span := observability.StartDBSpan(ctx, name, "query")
	defer span.End()

	start := time.Now()
	rows, err := d.inner.QueryContext(ctx, query, args...)
	d.tracker.observe(name, time.Since(start))
	span.RecordError(err)
	return rows, err
}

func (d *instrumentedDBTX) QueryRowContext(ctx context.Context, query string, args ...interface{}) *sql.Row {
	name := queryName(query)
	ctx, span := observability.StartDBSpan(ctx, name, "query_row")
	start := time.Now()
	row := d.inner.QueryRowContext(ctx, query, args...)
	d.tracker.observe(name, time.Since(start))
	span.End()
	return row
}

func queryName(query string) string {
	lines := strings.Split(strings.TrimSpace(query), "\n")
	if len(lines) == 0 {
		return "unknown"
	}
	first := strings.TrimSpace(lines[0])
	if !strings.HasPrefix(first, "-- name:") {
		return "unknown"
	}
	parts := strings.Fields(first)
	if len(parts) < 3 {
		return "unknown"
	}
	return strings.TrimSpace(parts[2])
}
