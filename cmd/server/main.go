package main

import (
	"log/slog"
	"os"

	_ "modernc.org/sqlite"

	"github.com/joho/godotenv"

	"github.com/fr0stylo/ddash"
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

	customWebhookBase := os.Getenv("DDASH_WEBHOOK_DB_BASE")
	if customWebhookBase == "" {
		customWebhookBase = "data"
	}

	authSecret := os.Getenv("DDASH_SESSION_SECRET")
	if authSecret == "" {
		authSecret = "ddash-local-dev"
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

	srv.RegisterRouter(routes.NewAuthRoutes())
	srv.RegisterRouter(routes.NewViewRoutes(database))
	srv.RegisterRouter(&routes.APIRoutes{})
	srv.RegisterRouter(routes.NewWebhookRoutes(
		[]byte(os.Getenv("GITHUB_WEBHOOK_SECRET")),
		customWebhookBase,
	))

	slog.Info("Starting server", "port", 8080)
	slog.Error("Closing server", "error", srv.Start(":8080"))
}
