package main

import (
	"log/slog"
	"os"
	"strings"

	"github.com/joho/godotenv"
	_ "modernc.org/sqlite"

	"github.com/fr0stylo/ddash"
	"github.com/fr0stylo/ddash/internal/adapters/sqlite"
	"github.com/fr0stylo/ddash/internal/db"
	"github.com/fr0stylo/ddash/internal/server"
	"github.com/fr0stylo/ddash/internal/server/routes"
)

var publicFS = ddash.PublicFS

func main() {
	log := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
	slog.SetDefault(log)

	if err := godotenv.Load(); err != nil {
		slog.Debug("No .env file loaded", "error", err)
	}

	srv := server.New(log, publicFS)
	isLocalEnv := isLocalDevelopmentEnv()

	defaultDBPath := os.Getenv("DDASH_DB_PATH")
	database, err := db.New(defaultDBPath)
	if err != nil {
		slog.Error("Failed to open database", "error", err)
		return
	}
	defer func() {
		if err := database.Close(); err != nil {
			slog.Error("Failed to close database", "error", err)
		}
	}()

	authSecret := os.Getenv("DDASH_SESSION_SECRET")
	if authSecret == "" {
		if isLocalEnv {
			authSecret = "ddash-local-dev"
			slog.Warn("DDASH_SESSION_SECRET not set, using local development fallback")
		} else {
			slog.Error("DDASH_SESSION_SECRET is required outside local/dev environments")
			return
		}
	}

	callbackURL := os.Getenv("GITHUB_CALLBACK_URL")
	if callbackURL == "" {
		callbackURL = "http://localhost:8080/auth/github/callback"
	}

	routes.ConfigureAuth(routes.AuthConfig{
		SessionKey:         authSecret,
		GitHubClientID:     os.Getenv("GITHUB_CLIENT_ID"),
		GitHubClientSecret: os.Getenv("GITHUB_CLIENT_SECRET"),
		GitHubCallbackURL:  callbackURL,
		SecureCookies:      os.Getenv("DDASH_SECURE_COOKIE") == "true",
	})

	store := sqlite.NewStore(database)

	srv.RegisterRouter(routes.NewAuthRoutes(store, isLocalEnv))
	srv.RegisterRouter(routes.NewViewRoutes(store, store))
	srv.RegisterRouter(routes.NewWebhookRoutes(sqlite.NewSharedIngestionStoreFactory(database)))

	slog.Info("Starting server", "port", 8080)
	slog.Error("Closing server", "error", srv.Start(":8080"))
}

func isLocalDevelopmentEnv() bool {
	env := strings.ToLower(strings.TrimSpace(os.Getenv("DDASH_ENV")))
	if env == "" {
		env = strings.ToLower(strings.TrimSpace(os.Getenv("APP_ENV")))
	}
	if env == "" {
		env = strings.ToLower(strings.TrimSpace(os.Getenv("GO_ENV")))
	}
	switch env {
	case "", "local", "dev", "development", "test":
		return true
	default:
		return false
	}
}
