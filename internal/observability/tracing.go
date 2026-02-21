package observability

import (
	"context"
	"strings"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

const dbTracerName = "ddash/db"

type contextKey string

const (
	userIDContextKey contextKey = "observability.user_id"
	orgIDContextKey  contextKey = "observability.org_id"
	requestIDKey     contextKey = "observability.request_id"
	routeKey         contextKey = "observability.route"
)

// Span is the application-level tracing span contract.
type Span interface {
	End()
	RecordError(error)
}

type otelSpan struct {
	inner trace.Span
}

// StartDBSpan starts a database tracing span for one query operation.
func StartDBSpan(ctx context.Context, queryName, operation string) (context.Context, Span) {
	queryName = strings.TrimSpace(queryName)
	if queryName == "" {
		queryName = "unknown"
	}
	attrs := []attribute.KeyValue{
		attribute.String("db.system.name", "sqlite"),
		attribute.String("db.query_name", queryName),
		attribute.String("db.operation", strings.TrimSpace(operation)),
	}
	if userID, ok := UserIDFromContext(ctx); ok {
		attrs = append(attrs, attribute.Int64("enduser.id", userID))
	}
	if orgID, ok := OrgIDFromContext(ctx); ok {
		attrs = append(attrs, attribute.Int64("ddash.org_id", orgID))
	}

	ctx, span := otel.Tracer(dbTracerName).Start(ctx, "db."+queryName,
		trace.WithSpanKind(trace.SpanKindClient),
		trace.WithAttributes(attrs...),
	)

	return ctx, otelSpan{inner: span}
}

// WithRequestIdentity enriches context and current span with user/org attributes.
func WithRequestIdentity(ctx context.Context, userID, orgID int64) context.Context {
	if userID > 0 {
		ctx = context.WithValue(ctx, userIDContextKey, userID)
	}
	if orgID > 0 {
		ctx = context.WithValue(ctx, orgIDContextKey, orgID)
	}
	setSpanIdentityAttributes(ctx, userID, orgID)
	return ctx
}

// WithRequestMetadata enriches context and current span with request metadata.
func WithRequestMetadata(ctx context.Context, requestID, route string) context.Context {
	requestID = strings.TrimSpace(requestID)
	route = strings.TrimSpace(route)
	if requestID != "" {
		ctx = context.WithValue(ctx, requestIDKey, requestID)
	}
	if route != "" {
		ctx = context.WithValue(ctx, routeKey, route)
	}
	setSpanRequestAttributes(ctx, requestID, route)
	return ctx
}

// UserIDFromContext extracts request user id.
func UserIDFromContext(ctx context.Context) (int64, bool) {
	value, ok := ctx.Value(userIDContextKey).(int64)
	return value, ok && value > 0
}

// OrgIDFromContext extracts active organization id.
func OrgIDFromContext(ctx context.Context) (int64, bool) {
	value, ok := ctx.Value(orgIDContextKey).(int64)
	return value, ok && value > 0
}

// RequestIDFromContext extracts request id.
func RequestIDFromContext(ctx context.Context) (string, bool) {
	value, ok := ctx.Value(requestIDKey).(string)
	value = strings.TrimSpace(value)
	return value, ok && value != ""
}

// RouteFromContext extracts normalized route path.
func RouteFromContext(ctx context.Context) (string, bool) {
	value, ok := ctx.Value(routeKey).(string)
	value = strings.TrimSpace(value)
	return value, ok && value != ""
}

func setSpanIdentityAttributes(ctx context.Context, userID, orgID int64) {
	span := trace.SpanFromContext(ctx)
	if span == nil {
		return
	}
	attrs := make([]attribute.KeyValue, 0, 2)
	if userID > 0 {
		attrs = append(attrs, attribute.Int64("enduser.id", userID))
	}
	if orgID > 0 {
		attrs = append(attrs, attribute.Int64("ddash.org_id", orgID))
	}
	if len(attrs) > 0 {
		span.SetAttributes(attrs...)
	}
}

func setSpanRequestAttributes(ctx context.Context, requestID, route string) {
	span := trace.SpanFromContext(ctx)
	if span == nil {
		return
	}
	attrs := make([]attribute.KeyValue, 0, 2)
	if requestID != "" {
		attrs = append(attrs, attribute.String("request.id", requestID))
	}
	if route != "" {
		attrs = append(attrs, attribute.String("http.route", route))
	}
	if len(attrs) > 0 {
		span.SetAttributes(attrs...)
	}
}

func (s otelSpan) End() {
	if s.inner == nil {
		return
	}
	s.inner.End()
}

func (s otelSpan) RecordError(err error) {
	if s.inner == nil || err == nil {
		return
	}
	s.inner.RecordError(err)
	s.inner.SetStatus(codes.Error, err.Error())
}
