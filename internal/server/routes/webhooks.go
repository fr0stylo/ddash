package routes

import (
	"net/http"

	"github.com/labstack/echo/v4"

	"github.com/fr0stylo/ddash/internal/app/ports"
	appservices "github.com/fr0stylo/ddash/internal/app/services"
	customwebhook "github.com/fr0stylo/ddash/internal/webhooks/custom"
)

// WebhookRoutes registers webhook endpoints.
type WebhookRoutes struct {
	custom *customwebhook.Handler
}

// NewWebhookRoutes constructs webhook routes.
func NewWebhookRoutes(storeFactory ports.IngestionStoreFactory, batchConfig appservices.IngestBatchConfig) *WebhookRoutes {
	return &WebhookRoutes{
		custom: customwebhook.NewHandler(storeFactory, batchConfig),
	}
}

// RegisterRoutes registers webhook endpoints.
func (w *WebhookRoutes) RegisterRoutes(s *echo.Echo) {
	s.POST("/webhooks/custom", w.handleLegacyCustomWebhook)
	s.POST("/webhooks/cdevents", w.handleCustomWebhook)
}

func (w *WebhookRoutes) handleCustomWebhook(c echo.Context) error {
	return w.custom.Handle(c.Response(), c.Request())
}

func (w *WebhookRoutes) handleLegacyCustomWebhook(c echo.Context) error {
	return c.JSON(http.StatusGone, map[string]string{
		"error": "legacy custom payload ingestion removed; send CDEvents delivery events to /webhooks/cdevents",
	})
}
