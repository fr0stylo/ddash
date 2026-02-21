package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/joho/godotenv"
	_ "modernc.org/sqlite"

	"github.com/fr0stylo/ddash"
	"github.com/fr0stylo/ddash/internal/adapters/sqlite"
	"github.com/fr0stylo/ddash/internal/app/services"
	"github.com/fr0stylo/ddash/internal/config"
	"github.com/fr0stylo/ddash/internal/db"
	"github.com/fr0stylo/ddash/internal/observability"
	"github.com/fr0stylo/ddash/internal/server"
	"github.com/fr0stylo/ddash/internal/server/routes"
)

var publicFS = ddash.PublicFS

func main() {
	baseHandler := slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo})
	log := slog.New(observability.WrapSlogHandler(baseHandler))
	slog.SetDefault(log)

	if err := godotenv.Load(); err != nil {
		slog.Debug("No .env file loaded", "error", err)
	}

	cfg, err := config.Load()
	if err != nil {
		slog.Error("Failed to load configuration", "error", err)
		return
	}
	if cfg.IsLocalDevelopment() && cfg.Auth.SessionSecret == "ddash-local-dev" {
		slog.Warn("DDASH_SESSION_SECRET not set, using local development fallback")
	}

	shutdownTelemetry, err := observability.SetupOpenTelemetry(context.Background(), log, observability.OpenTelemetryConfig{
		Enabled:           cfg.Observability.Enabled,
		OTLPEndpoint:      cfg.Observability.OTLPEndpoint,
		OTLPTraceHeaders:  cfg.Observability.OTLPTraceHeaders,
		OTLPMetricHeaders: cfg.Observability.OTLPMetricHeaders,
		ServiceName:       cfg.Observability.ServiceName,
		ServiceVer:        cfg.Observability.ServiceVer,
		SamplingRatio:     cfg.Observability.SamplingRatio,
		MetricsConsole:    cfg.Observability.MetricsConsole,
	})
	if err != nil {
		slog.Error("Failed to initialize OpenTelemetry", "error", err)
		return
	}
	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := shutdownTelemetry(ctx); err != nil {
			slog.Error("Failed to shutdown OpenTelemetry", "error", err)
		}
	}()

	srv := server.New(log, publicFS)

	database, err := db.New(cfg.Database.Path)
	if err != nil {
		slog.Error("Failed to open database", "error", err)
		return
	}
	defer func() {
		if err := database.Close(); err != nil {
			slog.Error("Failed to close database", "error", err)
		}
	}()

	if cfg.Database.LogTiming {
		go logDBLatencyStats(log, database)
	}

	routes.ConfigureAuth(routes.AuthConfig{
		SessionKey:         cfg.Auth.SessionSecret,
		GitHubClientID:     cfg.Auth.GitHubClientID,
		GitHubClientSecret: cfg.Auth.GitHubClientSecret,
		GitHubCallbackURL:  cfg.Auth.GitHubCallbackURL,
		SecureCookies:      cfg.Auth.SecureCookie,
	})

	store := sqlite.NewStore(database)

	srv.RegisterRouter(routes.NewAuthRoutes(store, cfg.IsLocalDevelopment()))
	srv.RegisterRouter(routes.NewViewRoutes(store, store, routes.ViewExternalConfig{
		PublicURL:           cfg.Integrations.PublicURL,
		GitHubIngestorURL:   cfg.Integrations.GitHubIngestorURL,
		GitHubIngestorToken: cfg.Integrations.GitHubIngestorToken,
	}))
	srv.RegisterRouter(routes.NewWebhookRoutes(sqlite.NewSharedIngestionStoreFactory(database), services.IngestBatchConfig{
		Enabled:       cfg.Ingestion.BatchEnabled,
		Size:          cfg.Ingestion.BatchSize,
		FlushInterval: cfg.IngestionBatchFlushInterval(),
	}))

	addr := fmt.Sprintf(":%d", cfg.Server.Port)
	slog.Info("Starting server", "port", cfg.Server.Port)
	slog.Error("Closing server", "error", srv.Start(addr))
}

func logDBLatencyStats(log *slog.Logger, database *db.Database) {
	ticker := time.NewTicker(60 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		stats := database.QueryLatencyStats()
		if len(stats) == 0 {
			continue
		}
		limit := 5
		if len(stats) < limit {
			limit = len(stats)
		}
		for index := 0; index < limit; index++ {
			entry := stats[index]
			log.Info("db_query_latency",
				"query", entry.Name,
				"count", entry.Count,
				"p50_ms", entry.P50.Milliseconds(),
				"p95_ms", entry.P95.Milliseconds(),
				"max_ms", entry.Max.Milliseconds(),
			)
		}
	}
}
