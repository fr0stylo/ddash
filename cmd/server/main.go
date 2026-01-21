package main

import (
	"log/slog"
	"os"

	_ "modernc.org/sqlite"

	"github.com/fr0stylo/ddash"
	"github.com/fr0stylo/ddash/internal/db"
	"github.com/fr0stylo/ddash/internal/server"
	"github.com/fr0stylo/ddash/internal/server/routes"
)

var publicFS = ddash.PublicFS

func main() {
	log := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
	slog.SetDefault(log)

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

	srv.RegisterRouter(routes.NewViewRoutes(database))
	srv.RegisterRouter(&routes.APIRoutes{})
	srv.RegisterRouter(routes.NewWebhookRoutes(
		[]byte(os.Getenv("GITHUB_WEBHOOK_SECRET")),
		customWebhookBase,
	))

	slog.Info("Starting server", "port", 8080)
	slog.Error("Closing server", "error", srv.Start(":8080"))
}
