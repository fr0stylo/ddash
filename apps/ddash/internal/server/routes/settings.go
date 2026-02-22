package routes

import (
	"net/http"

	"github.com/labstack/echo/v4"

	apporgconfig "github.com/fr0stylo/ddash/apps/ddash/internal/application/orgconfig"
	"github.com/fr0stylo/ddash/views/pages"
)

type settingsPayload struct {
	AuthToken                   string               `json:"authToken"`
	WebhookSecret               string               `json:"webhookSecret"`
	Enabled                     bool                 `json:"enabled"`
	ShowSyncStatus              bool                 `json:"showSyncStatus"`
	ShowMetadataBadges          bool                 `json:"showMetadataBadges"`
	ShowEnvironmentColumn       bool                 `json:"showEnvironmentColumn"`
	EnableSSELiveUpdates        bool                 `json:"enableSSELiveUpdates"`
	ShowDeploymentHistory       bool                 `json:"showDeploymentHistory"`
	ShowMetadataFilters         bool                 `json:"showMetadataFilters"`
	StrictMetadataEnforcement   bool                 `json:"strictMetadataEnforcement"`
	MaskSensitiveMetadataValues bool                 `json:"maskSensitiveMetadataValues"`
	AllowServiceMetadataEditing bool                 `json:"allowServiceMetadataEditing"`
	ShowOnboardingHints         bool                 `json:"showOnboardingHints"`
	ShowIntegrationTypeBadges   bool                 `json:"showIntegrationTypeBadges"`
	ShowServiceDetailInsights   bool                 `json:"showServiceDetailInsights"`
	ShowServiceDependencies     bool                 `json:"showServiceDependencies"`
	ShowServiceDeliveryMetrics  bool                 `json:"showServiceDeliveryMetrics"`
	DeploymentRetentionDays     int                  `json:"deploymentRetentionDays"`
	DefaultDashboardView        string               `json:"defaultDashboardView"`
	StatusSemanticsMode         string               `json:"statusSemanticsMode"`
	RequiredFields              []settingsFieldInput `json:"requiredFields"`
	EnvironmentOrder            []string             `json:"environmentOrder"`
}

type settingsFieldInput struct {
	Label      string `json:"label"`
	Type       string `json:"type"`
	Filterable bool   `json:"filterable"`
}

func (v *ViewRoutes) handleSettings(c echo.Context) error {
	ctx := c.Request().Context()
	orgID, err := v.currentOrganizationID(c)
	if err != nil {
		return err
	}
	settings, err := v.config.GetSettings(ctx, orgID)
	if err != nil {
		return err
	}

	return c.Render(http.StatusOK, "", pages.SettingsPage(
		mapDomainMetadataFields(settings.RequiredFields),
		settings.EnvironmentOrder,
		settings.AuthToken,
		settings.WebhookSecret,
		settings.Enabled,
		settings.ShowSyncStatus,
		settings.ShowMetadataBadges,
		settings.ShowEnvironmentColumn,
		settings.EnableSSELiveUpdates,
		settings.ShowDeploymentHistory,
		settings.ShowMetadataFilters,
		settings.StrictMetadataEnforcement,
		settings.MaskSensitiveMetadataValues,
		settings.AllowServiceMetadataEditing,
		settings.ShowOnboardingHints,
		settings.ShowIntegrationTypeBadges,
		settings.ShowServiceDetailInsights,
		settings.ShowServiceDeliveryMetrics,
		settings.ShowServiceDependencies,
		settings.DeploymentRetentionDays,
		settings.DefaultDashboardView,
		settings.StatusSemanticsMode,
		csrfToken(c),
	))
}

func (v *ViewRoutes) handleSettingsUpdate(c echo.Context) error {
	ctx := c.Request().Context()
	orgID, err := v.currentOrganizationID(c)
	if err != nil {
		return err
	}
	payload := settingsPayload{}
	if err := c.Bind(&payload); err != nil {
		return err
	}

	update := apporgconfig.OrganizationSettingsUpdate{
		AuthToken:                   payload.AuthToken,
		WebhookSecret:               payload.WebhookSecret,
		Enabled:                     payload.Enabled,
		ShowSyncStatus:              payload.ShowSyncStatus,
		ShowMetadataBadges:          payload.ShowMetadataBadges,
		ShowEnvironmentColumn:       payload.ShowEnvironmentColumn,
		EnableSSELiveUpdates:        payload.EnableSSELiveUpdates,
		ShowDeploymentHistory:       payload.ShowDeploymentHistory,
		ShowMetadataFilters:         payload.ShowMetadataFilters,
		StrictMetadataEnforcement:   payload.StrictMetadataEnforcement,
		MaskSensitiveMetadataValues: payload.MaskSensitiveMetadataValues,
		AllowServiceMetadataEditing: payload.AllowServiceMetadataEditing,
		ShowOnboardingHints:         payload.ShowOnboardingHints,
		ShowIntegrationTypeBadges:   payload.ShowIntegrationTypeBadges,
		ShowServiceDetailInsights:   payload.ShowServiceDetailInsights,
		ShowServiceDependencies:     payload.ShowServiceDependencies,
		ShowServiceDeliveryMetrics:  payload.ShowServiceDeliveryMetrics,
		DeploymentRetentionDays:     payload.DeploymentRetentionDays,
		DefaultDashboardView:        payload.DefaultDashboardView,
		StatusSemanticsMode:         payload.StatusSemanticsMode,
		EnvironmentOrder:            payload.EnvironmentOrder,
		RequiredFields:              make([]apporgconfig.RequiredFieldInput, 0, len(payload.RequiredFields)),
	}
	for _, field := range payload.RequiredFields {
		update.RequiredFields = append(update.RequiredFields, apporgconfig.RequiredFieldInput{
			Label:      field.Label,
			Type:       field.Type,
			Filterable: field.Filterable,
		})
	}

	err = v.config.UpdateSettings(ctx, orgID, update)
	if err != nil {
		return err
	}

	return c.NoContent(http.StatusNoContent)
}
