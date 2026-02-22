package main

import (
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"strings"

	"github.com/joho/godotenv"
	"github.com/spf13/viper"

	"github.com/fr0stylo/ddash/apps/gitlabingestor/internal/gitlabbridge"
)

func Run() error {
	_ = godotenv.Load()
	v := viper.New()
	v.AutomaticEnv()

	listenAddr := valueOrDefault(v.GetString("GITLAB_INGESTOR_ADDR"), ":8082")
	webhookToken := strings.TrimSpace(v.GetString("GITLAB_WEBHOOK_SECRET"))
	if webhookToken == "" {
		return fmt.Errorf("GITLAB_WEBHOOK_SECRET is required")
	}

	runtime := ingestorRuntime{
		ingestorToken: strings.TrimSpace(v.GetString("GITHUB_APP_INGESTOR_SETUP_TOKEN")),
		webhookPath:   valueOrDefault(v.GetString("GITLAB_INGESTOR_PATH"), "/webhooks/gitlab"),
		webhookToken:  webhookToken,
		ddashEndpoint: strings.TrimSpace(v.GetString("DDASH_ENDPOINT")),
		defaultConvertCfg: gitlabbridge.ConvertConfig{
			DefaultEnvironment: strings.TrimSpace(v.GetString("GITLAB_INGESTOR_DEFAULT_ENV")),
			Source:             valueOrDefault(v.GetString("GITLAB_INGESTOR_SOURCE"), "gitlab/webhook"),
		},
	}

	mux := http.NewServeMux()
	runtime.registerWebhookRoute(mux)

	slog.Info("GitLab ingestor listening", "addr", listenAddr, "webhook_path", runtime.webhookPath)
	if err := http.ListenAndServe(listenAddr, mux); err != nil {
		return fmt.Errorf("gitlab ingestor stopped: %w", err)
	}
	return nil
}

func main() {
	if err := Run(); err != nil {
		slog.Error("gitlab ingestor exited", "error", err)
		os.Exit(1)
	}
}
