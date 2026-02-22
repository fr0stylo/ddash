package config

import (
	"fmt"
	"strings"
	"time"

	"github.com/spf13/viper"
)

type Config struct {
	Environment   string
	Server        ServerConfig
	Database      DatabaseConfig
	Auth          AuthConfig
	Observability ObservabilityConfig
	Ingestion     IngestionConfig
	Integrations  IntegrationsConfig
}

type ServerConfig struct {
	Port int
}

type DatabaseConfig struct {
	Path      string
	LogTiming bool
}

type AuthConfig struct {
	SessionSecret      string
	GitHubClientID     string
	GitHubClientSecret string
	GitHubCallbackURL  string
	SecureCookie       bool
}

type ObservabilityConfig struct {
	Enabled           bool
	OTLPEndpoint      string
	OTLPTraceHeaders  map[string]string
	OTLPMetricHeaders map[string]string
	ServiceName       string
	ServiceVer        string
	SamplingRatio     float64
	MetricsConsole    bool
}

type IngestionConfig struct {
	BatchEnabled bool
	BatchSize    int
	BatchFlushMS int
}

type IntegrationsConfig struct {
	PublicURL           string
	GitHubAppInstallURL string
	GitHubIngestorToken string
}

func Load() (Config, error) {
	return load(true)
}

// LoadForTool loads config for CLI tools that do not require auth session secrets.
func LoadForTool() (Config, error) {
	return load(false)
}

func load(requireSessionSecret bool) (Config, error) {
	v := viper.New()
	v.AutomaticEnv()

	v.SetDefault("ddash_env", "")
	v.SetDefault("app_env", "")
	v.SetDefault("go_env", "")
	v.SetDefault("ddash_port", 8080)
	v.SetDefault("ddash_db_path", "data/default")
	v.SetDefault("ddash_db_timing", false)
	v.SetDefault("ddash_secure_cookie", false)
	v.SetDefault("ddash_otel_enabled", false)
	v.SetDefault("otel_exporter_otlp_endpoint", "")
	v.SetDefault("otel_exporter_otlp_headers", "")
	v.SetDefault("otel_exporter_otlp_traces_headers", "")
	v.SetDefault("otel_exporter_otlp_metrics_headers", "")
	v.SetDefault("otel_service_name", "ddash")
	v.SetDefault("ddash_service_name", "ddash")
	v.SetDefault("ddash_version", "dev")
	v.SetDefault("otel_service_version", "")
	v.SetDefault("ddash_otel_sampling_ratio", 1.0)
	v.SetDefault("ddash_otel_metrics_console", false)
	v.SetDefault("ddash_ingest_batch_enabled", true)
	v.SetDefault("ddash_ingest_batch_size", 100)
	v.SetDefault("ddash_ingest_batch_flush_ms", 50)
	v.SetDefault("ddash_public_url", "")
	v.SetDefault("github_app_install_url", "")
	v.SetDefault("github_app_ingestor_setup_token", "")

	env := resolveEnvironment(v)
	port := v.GetInt("ddash_port")
	if port <= 0 || port > 65535 {
		return Config{}, fmt.Errorf("invalid DDASH_PORT: %d", port)
	}

	samplingRatio := v.GetFloat64("ddash_otel_sampling_ratio")
	if samplingRatio < 0 {
		samplingRatio = 0
	}
	if samplingRatio > 1 {
		samplingRatio = 1
	}

	batchSize := v.GetInt("ddash_ingest_batch_size")
	if batchSize <= 0 {
		batchSize = 100
	}
	if batchSize > 2000 {
		batchSize = 2000
	}

	batchFlush := v.GetInt("ddash_ingest_batch_flush_ms")
	if batchFlush <= 0 {
		batchFlush = 50
	}
	if batchFlush > 5000 {
		batchFlush = 5000
	}

	callbackURL := strings.TrimSpace(v.GetString("github_callback_url"))
	if callbackURL == "" {
		callbackURL = fmt.Sprintf("http://localhost:%d/auth/github/callback", port)
	}

	serviceName := strings.TrimSpace(v.GetString("otel_service_name"))
	if serviceName == "" {
		serviceName = strings.TrimSpace(v.GetString("ddash_service_name"))
	}
	if serviceName == "" {
		serviceName = "ddash"
	}

	serviceVersion := strings.TrimSpace(v.GetString("ddash_version"))
	if serviceVersion == "" {
		serviceVersion = strings.TrimSpace(v.GetString("otel_service_version"))
	}
	if serviceVersion == "" {
		serviceVersion = "dev"
	}

	otlpEndpoint := strings.TrimSpace(v.GetString("otel_exporter_otlp_endpoint"))
	otlpCommonHeaders := parseOTLPHeaders(v.GetString("otel_exporter_otlp_headers"))
	otlpTraceHeaders := parseOTLPHeaders(v.GetString("otel_exporter_otlp_traces_headers"))
	otlpMetricHeaders := parseOTLPHeaders(v.GetString("otel_exporter_otlp_metrics_headers"))
	metricsConsole := v.GetBool("ddash_otel_metrics_console")
	otelEnabled := v.GetBool("ddash_otel_enabled") || otlpEndpoint != "" || metricsConsole
	traceHeaders := mergeHeaderMaps(otlpCommonHeaders, otlpTraceHeaders)
	metricHeaders := mergeHeaderMaps(otlpCommonHeaders, otlpMetricHeaders)

	cfg := Config{
		Environment: env,
		Server:      ServerConfig{Port: port},
		Database: DatabaseConfig{
			Path:      strings.TrimSpace(v.GetString("ddash_db_path")),
			LogTiming: v.GetBool("ddash_db_timing"),
		},
		Auth: AuthConfig{
			SessionSecret:      strings.TrimSpace(v.GetString("ddash_session_secret")),
			GitHubClientID:     strings.TrimSpace(v.GetString("github_client_id")),
			GitHubClientSecret: strings.TrimSpace(v.GetString("github_client_secret")),
			GitHubCallbackURL:  callbackURL,
			SecureCookie:       v.GetBool("ddash_secure_cookie"),
		},
		Observability: ObservabilityConfig{
			Enabled:           otelEnabled,
			OTLPEndpoint:      otlpEndpoint,
			OTLPTraceHeaders:  traceHeaders,
			OTLPMetricHeaders: metricHeaders,
			ServiceName:       serviceName,
			ServiceVer:        serviceVersion,
			SamplingRatio:     samplingRatio,
			MetricsConsole:    metricsConsole,
		},
		Ingestion: IngestionConfig{
			BatchEnabled: v.GetBool("ddash_ingest_batch_enabled"),
			BatchSize:    batchSize,
			BatchFlushMS: batchFlush,
		},
		Integrations: IntegrationsConfig{
			PublicURL:           strings.TrimSpace(v.GetString("ddash_public_url")),
			GitHubAppInstallURL: strings.TrimSpace(v.GetString("github_app_install_url")),
			GitHubIngestorToken: strings.TrimSpace(v.GetString("github_app_ingestor_setup_token")),
		},
	}

	if strings.TrimSpace(cfg.Database.Path) == "" {
		cfg.Database.Path = "data/default"
	}
	if requireSessionSecret && !cfg.IsLocalDevelopment() && cfg.Auth.SessionSecret == "" {
		return Config{}, fmt.Errorf("DDASH_SESSION_SECRET is required outside local/dev environments")
	}
	if cfg.IsLocalDevelopment() && cfg.Auth.SessionSecret == "" {
		cfg.Auth.SessionSecret = "ddash-local-dev"
	}

	return cfg, nil
}

func parseOTLPHeaders(raw string) map[string]string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil
	}
	out := make(map[string]string)
	for _, part := range strings.Split(raw, ",") {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		pair := strings.SplitN(part, "=", 2)
		if len(pair) != 2 {
			continue
		}
		key := strings.TrimSpace(pair[0])
		value := strings.TrimSpace(pair[1])
		if key == "" || value == "" {
			continue
		}
		out[key] = value
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

func mergeHeaderMaps(base, override map[string]string) map[string]string {
	if len(base) == 0 && len(override) == 0 {
		return nil
	}
	out := make(map[string]string, len(base)+len(override))
	for k, v := range base {
		out[k] = v
	}
	for k, v := range override {
		out[k] = v
	}
	return out
}

func (c Config) IsLocalDevelopment() bool {
	switch strings.ToLower(strings.TrimSpace(c.Environment)) {
	case "", "local", "dev", "development", "test":
		return true
	default:
		return false
	}
}

func (c Config) IngestionBatchFlushInterval() time.Duration {
	return time.Duration(c.Ingestion.BatchFlushMS) * time.Millisecond
}

func resolveEnvironment(v *viper.Viper) string {
	for _, key := range []string{"ddash_env", "app_env", "go_env"} {
		value := strings.TrimSpace(v.GetString(key))
		if value != "" {
			return strings.ToLower(value)
		}
	}
	return ""
}
