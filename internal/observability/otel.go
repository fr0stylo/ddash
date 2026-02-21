package observability

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetrichttp"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/exporters/stdout/stdoutmetric"
	"go.opentelemetry.io/otel/propagation"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.37.0"
)

type OpenTelemetryConfig struct {
	Enabled           bool
	OTLPEndpoint      string
	OTLPTraceHeaders  map[string]string
	OTLPMetricHeaders map[string]string
	ServiceName       string
	ServiceVer        string
	SamplingRatio     float64
	MetricsConsole    bool
}

// SetupOpenTelemetry configures global tracing and propagation.
func SetupOpenTelemetry(ctx context.Context, log *slog.Logger, cfg OpenTelemetryConfig) (func(context.Context) error, error) {
	if !cfg.Enabled {
		return func(context.Context) error { return nil }, nil
	}

	res, err := resource.New(ctx,
		resource.WithFromEnv(),
		resource.WithProcess(),
		resource.WithTelemetrySDK(),
		resource.WithAttributes(
			semconv.ServiceName(cfg.ServiceName),
			semconv.ServiceVersion(cfg.ServiceVer),
		),
	)
	if err != nil {
		return nil, fmt.Errorf("create otel resource: %w", err)
	}
	shutdownFns := make([]func(context.Context) error, 0, 2)
	tracesEnabled := false
	if cfg.OTLPEndpoint != "" || len(cfg.OTLPTraceHeaders) > 0 {
		options := make([]otlptracehttp.Option, 0, 2)
		if cfg.OTLPEndpoint != "" {
			options = append(options, otlptracehttp.WithEndpointURL(cfg.OTLPEndpoint))
		}
		if len(cfg.OTLPTraceHeaders) > 0 {
			options = append(options, otlptracehttp.WithHeaders(cfg.OTLPTraceHeaders))
		}
		exporter, exporterErr := otlptracehttp.New(ctx, options...)
		if exporterErr != nil {
			return nil, fmt.Errorf("create otlp trace exporter: %w", exporterErr)
		}

		provider := sdktrace.NewTracerProvider(
			sdktrace.WithSampler(configuredSampler(cfg.SamplingRatio)),
			sdktrace.WithBatcher(exporter),
			sdktrace.WithResource(res),
		)
		otel.SetTracerProvider(provider)
		shutdownFns = append(shutdownFns, provider.Shutdown)
		tracesEnabled = true
	}

	metricReaders := make([]sdkmetric.Reader, 0, 2)
	if cfg.OTLPEndpoint != "" {
		metricOptions := make([]otlpmetrichttp.Option, 0, 2)
		metricOptions = append(metricOptions, otlpmetrichttp.WithEndpointURL(cfg.OTLPEndpoint))
		if len(cfg.OTLPMetricHeaders) > 0 {
			metricOptions = append(metricOptions, otlpmetrichttp.WithHeaders(cfg.OTLPMetricHeaders))
		}
		exporter, exporterErr := otlpmetrichttp.New(ctx, metricOptions...)
		if exporterErr != nil {
			return nil, fmt.Errorf("create otlp metric exporter: %w", exporterErr)
		}
		metricReaders = append(metricReaders, sdkmetric.NewPeriodicReader(exporter, sdkmetric.WithInterval(10*time.Second)))
	}
	if cfg.MetricsConsole {
		exporter, exporterErr := stdoutmetric.New(stdoutmetric.WithPrettyPrint())
		if exporterErr != nil {
			return nil, fmt.Errorf("create stdout metric exporter: %w", exporterErr)
		}
		metricReaders = append(metricReaders, sdkmetric.NewPeriodicReader(exporter, sdkmetric.WithInterval(10*time.Second)))
	}
	if len(metricReaders) > 0 {
		meterOptions := []sdkmetric.Option{sdkmetric.WithResource(res)}
		for _, reader := range metricReaders {
			meterOptions = append(meterOptions, sdkmetric.WithReader(reader))
		}
		meterProvider := sdkmetric.NewMeterProvider(meterOptions...)
		otel.SetMeterProvider(meterProvider)
		shutdownFns = append(shutdownFns, meterProvider.Shutdown)
	}

	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	))
	instrumentDefaultHTTPClient()

	log.Info("OpenTelemetry enabled",
		"service", cfg.ServiceName,
		"version", cfg.ServiceVer,
		"traces_enabled", tracesEnabled,
		"metrics_console", cfg.MetricsConsole,
		"metrics_otlp", cfg.OTLPEndpoint != "",
	)

	return func(shutdownCtx context.Context) error {
		var firstErr error
		for i := len(shutdownFns) - 1; i >= 0; i-- {
			if err := shutdownFns[i](shutdownCtx); err != nil && firstErr == nil {
				firstErr = err
			}
		}
		return firstErr
	}, nil
}

func instrumentDefaultHTTPClient() {
	http.DefaultTransport = otelhttp.NewTransport(http.DefaultTransport)
	http.DefaultClient.Transport = http.DefaultTransport
}

func configuredSampler(ratio float64) sdktrace.Sampler {
	if ratio >= 1 {
		return sdktrace.ParentBased(sdktrace.AlwaysSample())
	}
	if ratio <= 0 {
		return sdktrace.ParentBased(sdktrace.NeverSample())
	}
	return sdktrace.ParentBased(sdktrace.TraceIDRatioBased(ratio))
}
