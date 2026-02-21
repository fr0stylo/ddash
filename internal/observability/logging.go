package observability

import (
	"context"
	"io"
	"log/slog"

	"go.opentelemetry.io/otel/trace"
)

type traceAwareHandler struct {
	next slog.Handler
}

// WrapSlogHandler adds trace and request context fields to structured logs.
func WrapSlogHandler(next slog.Handler) slog.Handler {
	if next == nil {
		next = slog.NewTextHandler(io.Discard, nil)
	}
	return &traceAwareHandler{next: next}
}

func (h *traceAwareHandler) Enabled(ctx context.Context, level slog.Level) bool {
	return h.next.Enabled(ctx, level)
}

func (h *traceAwareHandler) Handle(ctx context.Context, record slog.Record) error {
	if requestID, ok := RequestIDFromContext(ctx); ok {
		record.AddAttrs(slog.String("request_id", requestID))
	}
	if route, ok := RouteFromContext(ctx); ok {
		record.AddAttrs(slog.String("route", route))
	}

	span := trace.SpanFromContext(ctx)
	if span != nil {
		sc := span.SpanContext()
		if sc.IsValid() {
			record.AddAttrs(
				slog.String("trace_id", sc.TraceID().String()),
				slog.String("span_id", sc.SpanID().String()),
			)
		}
	}

	return h.next.Handle(ctx, record)
}

func (h *traceAwareHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return &traceAwareHandler{next: h.next.WithAttrs(attrs)}
}

func (h *traceAwareHandler) WithGroup(name string) slog.Handler {
	return &traceAwareHandler{next: h.next.WithGroup(name)}
}
