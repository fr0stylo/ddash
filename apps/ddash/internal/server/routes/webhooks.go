package routes

import (
	"net/http"

	"github.com/labstack/echo/v4"

	"github.com/fr0stylo/ddash/apps/ddash/internal/app/ports"
	appingestion "github.com/fr0stylo/ddash/apps/ddash/internal/application/ingestion"
	customwebhook "github.com/fr0stylo/ddash/apps/ddash/internal/webhooks/custom"
)

// WebhookRoutes registers webhook endpoints.
type WebhookRoutes struct {
	custom    *customwebhook.Handler
	githubApp *customwebhook.GitHubAppHandler
	gitlabApp *customwebhook.GitLabAppHandler
}

// NewWebhookRoutes constructs webhook routes.
func NewWebhookRoutes(storeFactory ports.IngestionStoreFactory, batchConfig appingestion.BatchConfig, installations ports.GitHubInstallationStore, gitlabProjects ports.GitLabProjectStore, ingestorToken string) *WebhookRoutes {
	return &WebhookRoutes{
		custom:    customwebhook.NewHandler(storeFactory, batchConfig),
		githubApp: customwebhook.NewGitHubAppHandler(storeFactory, batchConfig, installations, ingestorToken),
		gitlabApp: customwebhook.NewGitLabAppHandler(storeFactory, batchConfig, gitlabProjects, ingestorToken),
	}
}

// RegisterRoutes registers webhook endpoints.
func (w *WebhookRoutes) RegisterRoutes(s *echo.Echo) {
	s.POST("/webhooks/custom", w.handleLegacyCustomWebhook)
	s.POST("/webhooks/cdevents", w.handleCustomWebhook)
	s.POST("/webhooks/github-app", w.handleGitHubAppWebhook)
	s.POST("/webhooks/gitlab-app", w.handleGitLabAppWebhook)
}

func (w *WebhookRoutes) handleCustomWebhook(c echo.Context) error {
	return w.custom.Handle(c.Response(), c.Request())
}

func (w *WebhookRoutes) handleLegacyCustomWebhook(c echo.Context) error {
	return c.JSON(http.StatusGone, map[string]string{
		"error": "legacy custom payload ingestion removed; send CDEvents delivery events to /webhooks/cdevents",
	})
}

func (w *WebhookRoutes) handleGitHubAppWebhook(c echo.Context) error {
	return w.githubApp.Handle(c.Response(), c.Request())
}

func (w *WebhookRoutes) handleGitLabAppWebhook(c echo.Context) error {
	return w.gitlabApp.Handle(c.Response(), c.Request())
}
