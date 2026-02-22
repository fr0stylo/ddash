package routes

import (
	"net/http"

	"github.com/labstack/echo/v4"

	"github.com/fr0stylo/ddash/views/pages"
)

func (v *ViewRoutes) handleHome(c echo.Context) error {
	ctx := c.Request().Context()
	orgID, err := v.currentOrganizationID(c)
	if err != nil {
		return err
	}
	services, metadataOptions, err := v.read.GetHomeData(ctx, orgID)
	if err != nil {
		return err
	}
	if len(services) == 0 {
		return c.Redirect(http.StatusFound, "/onboarding")
	}
	settings, err := v.loadDashboardSettings(ctx, orgID)
	if err != nil {
		return err
	}
	return c.Render(http.StatusOK, "", pages.HomePage(mapDomainServices(services), mapDomainMetadataOptions(metadataOptions), settings.ShowSyncStatus, settings.ShowMetadataBadges, settings.ShowEnvironmentColumn, settings.ShowMetadataFilters, settings.EnableSSELiveUpdates, settings.DefaultDashboardView, settings.StatusSemanticsMode, settings.ShowOnboardingHints))
}

func (v *ViewRoutes) handleOnboarding(c echo.Context) error {
	ctx := c.Request().Context()
	orgID, err := v.currentOrganizationID(c)
	if err != nil {
		return err
	}
	settings, err := v.loadDashboardSettings(ctx, orgID)
	if err != nil {
		return err
	}
	return c.Render(http.StatusOK, "", pages.OnboardingPage(settings.ShowOnboardingHints))
}
