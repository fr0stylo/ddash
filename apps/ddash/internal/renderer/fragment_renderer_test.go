package renderer

import (
	"context"
	"io"
	"testing"
	"time"

	"github.com/a-h/templ"
)

type staticComponent string

func (c staticComponent) Render(_ context.Context, w io.Writer) error {
	_, err := io.WriteString(w, string(c))
	return err
}

func TestFragmentRendererCachesByVersion(t *testing.T) {
	t.Parallel()

	r := NewFragmentRenderer(8, time.Minute)
	buildCount := 0
	builder := func() (templ.Component, error) {
		buildCount++
		return staticComponent("<div>ok</div>"), nil
	}

	body, hit, err := r.RenderCached(context.Background(), "k1", 10, builder)
	if err != nil {
		t.Fatalf("RenderCached first call error: %v", err)
	}
	if hit {
		t.Fatalf("expected miss on first call")
	}
	if string(body) != "<div>ok</div>" {
		t.Fatalf("unexpected body: %s", string(body))
	}

	_, hit, err = r.RenderCached(context.Background(), "k1", 10, builder)
	if err != nil {
		t.Fatalf("RenderCached second call error: %v", err)
	}
	if !hit {
		t.Fatalf("expected hit on same key/version")
	}
	if buildCount != 1 {
		t.Fatalf("expected single build, got %d", buildCount)
	}

	_, hit, err = r.RenderCached(context.Background(), "k1", 11, builder)
	if err != nil {
		t.Fatalf("RenderCached version bump error: %v", err)
	}
	if hit {
		t.Fatalf("expected miss after version bump")
	}
	if buildCount != 2 {
		t.Fatalf("expected rebuild after version bump, got %d", buildCount)
	}

}

func TestFragmentRendererTTLEviction(t *testing.T) {
	t.Parallel()

	r := NewFragmentRenderer(8, 20*time.Millisecond)
	buildCount := 0
	builder := func() (templ.Component, error) {
		buildCount++
		return staticComponent("<div>x</div>"), nil
	}

	if _, _, err := r.RenderCached(context.Background(), "k1", 1, builder); err != nil {
		t.Fatalf("initial render error: %v", err)
	}
	time.Sleep(30 * time.Millisecond)
	if _, hit, err := r.RenderCached(context.Background(), "k1", 1, builder); err != nil {
		t.Fatalf("post-ttl render error: %v", err)
	} else if hit {
		t.Fatalf("expected miss after TTL expiry")
	}
	if buildCount != 2 {
		t.Fatalf("expected rebuild after expiry, got %d", buildCount)
	}
}

func TestFragmentRendererBypassMetric(t *testing.T) {
	t.Parallel()

	r := NewFragmentRenderer(8, time.Minute)
	r.RecordBypass(context.Background())
	r.RecordBypass(context.Background())
}
