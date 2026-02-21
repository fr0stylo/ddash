package routes

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/labstack/echo/v4"

	"github.com/fr0stylo/ddash/views/components"
	"github.com/fr0stylo/ddash/views/pages"
)

func (v *ViewRoutes) handleGitHubIntegration(c echo.Context) error {
	ctx := c.Request().Context()
	orgID, err := v.currentOrganizationID(c)
	if err != nil {
		return err
	}

	mappings := make([]components.GitHubInstallationMapping, 0)
	if v.github != nil && v.github.Enabled() {
		rows, listErr := v.github.ListMappings(ctx, orgID)
		if listErr != nil {
			return listErr
		}
		for _, row := range rows {
			status := "disabled"
			if row.Enabled {
				status = "enabled"
			}
			mappings = append(mappings, components.GitHubInstallationMapping{
				InstallationID:     row.InstallationID,
				OrganizationLabel:  row.OrganizationLabel,
				Endpoint:           row.DDashEndpoint,
				DefaultEnvironment: row.DefaultEnvironment,
				Status:             status,
			})
		}
	}

	return c.Render(http.StatusOK, "", pages.GitHubIntegrationPage(v.github != nil && v.github.Enabled(), mappings, csrfToken(c)))
}

func (v *ViewRoutes) handleGitHubIntegrationLink(c echo.Context) error {
	ctx := c.Request().Context()
	orgID, err := v.currentOrganizationID(c)
	if err != nil {
		return err
	}
	if v.github == nil || !v.github.Enabled() {
		return c.NoContent(http.StatusServiceUnavailable)
	}

	settings, err := v.config.GetSettings(ctx, orgID)
	if err != nil {
		return err
	}
	org, err := v.orgs.GetOrganizationByID(ctx, orgID)
	if err != nil {
		return err
	}
	defaultEnvironment := strings.TrimSpace(c.FormValue("default_environment"))
	if defaultEnvironment == "" {
		defaultEnvironment = "production"
	}

	redirectURL, err := v.github.StartInstall(ctx, orgID, org.Name, settings.AuthToken, settings.WebhookSecret, defaultEnvironment)
	if err != nil {
		return err
	}
	return c.Redirect(http.StatusFound, redirectURL)
}

func (v *ViewRoutes) handleGitHubIntegrationDelete(c echo.Context) error {
	ctx := c.Request().Context()
	orgID, err := v.currentOrganizationID(c)
	if err != nil {
		return err
	}
	if v.github == nil || !v.github.Enabled() {
		return c.NoContent(http.StatusServiceUnavailable)
	}
	installationID, err := strconv.ParseInt(strings.TrimSpace(c.FormValue("installation_id")), 10, 64)
	if err != nil || installationID <= 0 {
		return c.NoContent(http.StatusBadRequest)
	}
	if err := v.github.DeleteMapping(ctx, installationID, orgID); err != nil {
		return err
	}
	return c.Redirect(http.StatusFound, "/settings/integrations/github")
}
