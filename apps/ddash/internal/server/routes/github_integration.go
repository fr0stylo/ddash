package routes

import (
	"errors"
	"net/http"
	"strconv"
	"strings"

	appgithub "github.com/fr0stylo/ddash/apps/ddash/internal/application/githubintegration"
	"github.com/fr0stylo/ddash/views/components"
	"github.com/fr0stylo/ddash/views/pages"
	"github.com/labstack/echo/v4"
)

func (v *ViewRoutes) handleGitHubIntegration(c echo.Context) error {
	ctx := c.Request().Context()
	orgID, err := v.currentOrganizationID(c)
	if err != nil {
		return err
	}

	mappings := make([]components.GitHubInstallationMapping, 0)
	if v.githubIntegration != nil {
		rows, err := v.githubIntegration.ListMappings(ctx, orgID)
		if err != nil {
			return err
		}
		for _, row := range rows {
			status := "disabled"
			if row.Enabled {
				status = "enabled"
			}
			mappings = append(mappings, components.GitHubInstallationMapping{
				InstallationID:     row.InstallationID,
				OrganizationLabel:  row.OrganizationLabel,
				Endpoint:           "ddash://internal",
				DefaultEnvironment: row.DefaultEnvironment,
				Status:             status,
			})
		}
	}

	return c.Render(http.StatusOK, "", pages.GitHubIntegrationPage(v.githubIntegration != nil && v.githubIntegration.Enabled(), mappings, csrfToken(c)))
}

func (v *ViewRoutes) handleGitHubIntegrationLink(c echo.Context) error {
	ctx := c.Request().Context()
	orgID, err := v.currentOrganizationID(c)
	if err != nil {
		return err
	}
	if v.githubIntegration == nil || !v.githubIntegration.Enabled() {
		return c.NoContent(http.StatusServiceUnavailable)
	}

	org, err := v.orgs.GetOrganizationByID(ctx, orgID)
	if err != nil {
		return err
	}
	defaultEnvironment := strings.TrimSpace(c.FormValue("default_environment"))
	if defaultEnvironment == "" {
		defaultEnvironment = "production"
	}

	redirectURL, err := v.githubIntegration.StartInstall(ctx, orgID, org.Name, defaultEnvironment)
	if err != nil {
		return err
	}
	return c.Redirect(http.StatusFound, redirectURL)
}

func (v *ViewRoutes) handleGitHubIntegrationCallback(c echo.Context) error {
	ctx := c.Request().Context()
	if v.githubIntegration == nil {
		return c.NoContent(http.StatusServiceUnavailable)
	}
	state := strings.TrimSpace(c.QueryParam("state"))
	installationID, err := strconv.ParseInt(strings.TrimSpace(firstNonEmpty(c.QueryParam("installation_id"), c.QueryParam("installationId"))), 10, 64)
	if state == "" || err != nil || installationID <= 0 {
		return c.NoContent(http.StatusBadRequest)
	}

	if err := v.githubIntegration.CompleteInstall(ctx, state, installationID); err != nil {
		if errors.Is(err, appgithub.ErrSetupIntentNotFound) {
			return c.NoContent(http.StatusNotFound)
		}
		if errors.Is(err, appgithub.ErrSetupIntentExpired) {
			return c.NoContent(http.StatusGone)
		}
		return err
	}

	return c.HTML(http.StatusOK, "<html><body><h2>GitHub installation mapped successfully.</h2><p>You can close this window.</p><p><a href='/settings/integrations/github'>Open integration settings</a></p></body></html>")
}

func (v *ViewRoutes) handleGitHubIntegrationDelete(c echo.Context) error {
	ctx := c.Request().Context()
	orgID, err := v.currentOrganizationID(c)
	if err != nil {
		return err
	}
	if v.githubIntegration == nil || !v.githubIntegration.Enabled() {
		return c.NoContent(http.StatusServiceUnavailable)
	}
	installationID, err := strconv.ParseInt(strings.TrimSpace(c.FormValue("installation_id")), 10, 64)
	if err != nil || installationID <= 0 {
		return c.NoContent(http.StatusBadRequest)
	}
	if err := v.githubIntegration.DeleteMapping(ctx, orgID, installationID); err != nil {
		return err
	}
	return c.Redirect(http.StatusFound, "/settings/integrations/github")
}
