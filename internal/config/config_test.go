package config

import "testing"

func TestLoadDefaultsForLocalDevelopment(t *testing.T) {
	t.Setenv("DDASH_ENV", "dev")
	t.Setenv("DDASH_SESSION_SECRET", "")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	if cfg.Auth.SessionSecret != "ddash-local-dev" {
		t.Fatalf("expected local fallback secret, got %q", cfg.Auth.SessionSecret)
	}
	if cfg.Server.Port != 8080 {
		t.Fatalf("expected default port 8080, got %d", cfg.Server.Port)
	}
}

func TestLoadRequiresSessionSecretOutsideLocal(t *testing.T) {
	t.Setenv("DDASH_ENV", "production")
	t.Setenv("DDASH_SESSION_SECRET", "")

	_, err := Load()
	if err == nil {
		t.Fatal("expected error for missing session secret in production")
	}
}

func TestLoadForToolAllowsMissingSessionSecretOutsideLocal(t *testing.T) {
	t.Setenv("DDASH_ENV", "production")
	t.Setenv("DDASH_SESSION_SECRET", "")

	cfg, err := LoadForTool()
	if err != nil {
		t.Fatalf("expected no error for tool config load, got %v", err)
	}
	if cfg.Auth.SessionSecret != "" {
		t.Fatalf("expected empty session secret for tool load, got %q", cfg.Auth.SessionSecret)
	}
}

func TestLoadParsesOTLPHeadersAndMetricsConsole(t *testing.T) {
	t.Setenv("DDASH_ENV", "dev")
	t.Setenv("DDASH_SESSION_SECRET", "")
	t.Setenv("OTEL_EXPORTER_OTLP_HEADERS", "authorization=Bearer common,x-org=abc")
	t.Setenv("OTEL_EXPORTER_OTLP_TRACES_HEADERS", "x-trace=trace-only")
	t.Setenv("OTEL_EXPORTER_OTLP_METRICS_HEADERS", "x-metric=metric-only")
	t.Setenv("DDASH_OTEL_METRICS_CONSOLE", "true")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	if !cfg.Observability.Enabled {
		t.Fatal("expected observability enabled when console metrics is true")
	}
	if !cfg.Observability.MetricsConsole {
		t.Fatal("expected metrics console enabled")
	}
	if cfg.Observability.OTLPTraceHeaders["authorization"] != "Bearer common" {
		t.Fatalf("expected common header to be in trace headers, got %#v", cfg.Observability.OTLPTraceHeaders)
	}
	if cfg.Observability.OTLPTraceHeaders["x-trace"] != "trace-only" {
		t.Fatalf("expected trace-specific header, got %#v", cfg.Observability.OTLPTraceHeaders)
	}
	if cfg.Observability.OTLPMetricHeaders["authorization"] != "Bearer common" {
		t.Fatalf("expected common header to be in metric headers, got %#v", cfg.Observability.OTLPMetricHeaders)
	}
	if cfg.Observability.OTLPMetricHeaders["x-metric"] != "metric-only" {
		t.Fatalf("expected metric-specific header, got %#v", cfg.Observability.OTLPMetricHeaders)
	}
}
