package routes

import (
	"github.com/labstack/echo/v4"

	customwebhook "github.com/fr0stylo/ddash/internal/webhooks/custom"
	githubwebhook "github.com/fr0stylo/ddash/internal/webhooks/github"
)

// WebhookRoutes registers webhook endpoints.
type WebhookRoutes struct {
	custom *customwebhook.Handler
	github *githubwebhook.Handler
}

// NewWebhookRoutes constructs webhook routes.
func NewWebhookRoutes(githubSecret []byte, baseDir string) *WebhookRoutes {
	return &WebhookRoutes{
		custom: customwebhook.NewHandler(baseDir),
		github: githubwebhook.NewHandler(githubSecret),
	}
}

// RegisterRoutes registers webhook endpoints.
func (w *WebhookRoutes) RegisterRoutes(s *echo.Echo) {
	s.POST("/webhooks/github", w.handleGitHubWebhook)
	s.POST("/webhooks/custom", w.handleCustomWebhook)
}

func (w *WebhookRoutes) handleGitHubWebhook(c echo.Context) error {
	return w.github.Handle(c.Response(), c.Request())
}

func (w *WebhookRoutes) handleCustomWebhook(c echo.Context) error {
	return w.custom.Handle(c.Response(), c.Request())
}
