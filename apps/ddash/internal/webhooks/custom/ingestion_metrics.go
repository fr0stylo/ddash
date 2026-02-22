package custom

import (
	"context"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
)

type providerIngestionMetrics struct {
	requests metric.Int64Counter
	accepted metric.Int64Counter
	rejected metric.Int64Counter
	mapping  metric.Int64Counter
}

func newProviderIngestionMetrics() providerIngestionMetrics {
	meter := otel.Meter("github.com/fr0stylo/ddash/apps/ddash/internal/webhooks/custom")
	requests, _ := meter.Int64Counter("ddash.ingestion.provider.requests")
	accepted, _ := meter.Int64Counter("ddash.ingestion.provider.accepted")
	rejected, _ := meter.Int64Counter("ddash.ingestion.provider.rejected")
	mapping, _ := meter.Int64Counter("ddash.ingestion.provider.mapping")
	return providerIngestionMetrics{
		requests: requests,
		accepted: accepted,
		rejected: rejected,
		mapping:  mapping,
	}
}

func (m providerIngestionMetrics) recordRequest(ctx context.Context, provider string) {
	m.requests.Add(ctx, 1, metric.WithAttributes(attribute.String("provider", provider)))
}

func (m providerIngestionMetrics) recordAccepted(ctx context.Context, provider string) {
	m.accepted.Add(ctx, 1, metric.WithAttributes(attribute.String("provider", provider)))
}

func (m providerIngestionMetrics) recordRejected(ctx context.Context, provider, reason string) {
	m.rejected.Add(ctx, 1, metric.WithAttributes(
		attribute.String("provider", provider),
		attribute.String("reason", reason),
	))
}

func (m providerIngestionMetrics) recordMapping(ctx context.Context, provider, outcome string) {
	m.mapping.Add(ctx, 1, metric.WithAttributes(
		attribute.String("provider", provider),
		attribute.String("outcome", outcome),
	))
}
