package main

import (
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"strings"

	"github.com/joho/godotenv"
	"github.com/spf13/viper"

	"github.com/fr0stylo/ddash/apps/githubappingestor/internal/githubbridge"
)

func Run() error {
	_ = godotenv.Load()
	v := viper.New()
	v.AutomaticEnv()

	listenAddr := valueOrDefault(v.GetString("GITHUB_APP_INGESTOR_ADDR"), ":8081")

	githubSecret := strings.TrimSpace(v.GetString("GITHUB_WEBHOOK_SECRET"))
	if githubSecret == "" {
		return fmt.Errorf("GITHUB_WEBHOOK_SECRET is required")
	}

	ddashEndpoint := strings.TrimSpace(v.GetString("DDASH_ENDPOINT"))
	runtime := ingestorRuntime{
		ingestorToken: strings.TrimSpace(v.GetString("GITHUB_APP_INGESTOR_SETUP_TOKEN")),
		webhookPath:   valueOrDefault(v.GetString("GITHUB_APP_INGESTOR_PATH"), "/webhooks/github"),
		githubSecret:  githubSecret,
		ddashEndpoint: ddashEndpoint,
		defaultConvertCfg: githubbridge.ConvertConfig{
			DefaultEnvironment: strings.TrimSpace(v.GetString("GITHUB_APP_INGESTOR_DEFAULT_ENV")),
			Source:             valueOrDefault(v.GetString("GITHUB_APP_INGESTOR_SOURCE"), "github/app"),
		},
	}

	mux := http.NewServeMux()
	runtime.registerWebhookRoute(mux)

	slog.Info("GitHub App ingestor listening", "addr", listenAddr, "webhook_path", runtime.webhookPath)
	if err := http.ListenAndServe(listenAddr, mux); err != nil {
		return fmt.Errorf("github app ingestor stopped: %w", err)
	}
	return nil
}

func main() {
	if err := Run(); err != nil {
		slog.Error("github app ingestor exited", "error", err)
		os.Exit(1)
	}
}
