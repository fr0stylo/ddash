package routes

import (
	"fmt"
	"net/http"
	"time"

	"github.com/labstack/echo/v4"

	"github.com/fr0stylo/ddash/internal/renderer"
	"github.com/fr0stylo/ddash/views/components"
	"github.com/fr0stylo/ddash/views/pages"
)

func (v *ViewRoutes) handleDeployments(c echo.Context) error {
	ctx := c.Request().Context()
	orgID, err := v.currentOrganizationID(c)
	if err != nil {
		return err
	}
	deployments, metadataOptions, err := v.read.GetDeployments(ctx, orgID, "", "")
	if err != nil {
		return err
	}
	settings, err := v.loadDashboardSettings(ctx, orgID)
	if err != nil {
		return err
	}
	return c.Render(http.StatusOK, "", pages.DeploymentsPage(mapDomainDeployments(deployments), mapDomainMetadataOptions(metadataOptions), settings.ShowSyncStatus, settings.ShowEnvironmentColumn, settings.ShowMetadataFilters, settings.EnableSSELiveUpdates, settings.StatusSemanticsMode))
}

func (v *ViewRoutes) handleDeploymentFilter(c echo.Context) error {
	ctx := c.Request().Context()
	env := c.QueryParam("env")
	service := c.QueryParam("service")
	orgID, err := v.currentOrganizationID(c)
	if err != nil {
		return err
	}
	deployments, _, err := v.read.GetDeployments(ctx, orgID, env, service)
	if err != nil {
		return err
	}
	settings, err := v.loadDashboardSettings(ctx, orgID)
	if err != nil {
		return err
	}
	return c.Render(http.StatusOK, "", pages.DeploymentResults(mapDomainDeployments(deployments), env, service, settings.ShowSyncStatus, settings.ShowEnvironmentColumn, settings.EnableSSELiveUpdates, settings.StatusSemanticsMode))
}

func (v *ViewRoutes) handleDeploymentStream(c echo.Context) error {
	w := c.Response().Writer
	c.Response().Header().Set("Content-Type", "text/event-stream")
	c.Response().Header().Set("Cache-Control", "no-cache")
	c.Response().Header().Set("Connection", "keep-alive")

	flusher, ok := w.(http.Flusher)
	if !ok {
		return fmt.Errorf("streaming unsupported")
	}

	ctx := c.Request().Context()
	orgID, err := v.currentOrganizationID(c)
	if err != nil {
		return err
	}
	settings, err := v.loadDashboardSettings(ctx, orgID)
	if err != nil {
		return err
	}
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	envFilter := c.QueryParam("env")
	serviceFilter := c.QueryParam("service")

	seen := map[string]bool{}
	initialRows, _, err := v.read.GetDeployments(ctx, orgID, envFilter, serviceFilter)
	if err != nil {
		return err
	}
	for _, row := range mapDomainDeployments(initialRows) {
		seen[deploymentEventKey(row)] = true
	}

	for {
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
			rows, _, err := v.read.GetDeployments(ctx, orgID, envFilter, serviceFilter)
			if err != nil {
				return err
			}
			uiRows := mapDomainDeployments(rows)
			newRows := make([]components.DeploymentRow, 0, len(uiRows))
			for _, row := range uiRows {
				key := deploymentEventKey(row)
				if seen[key] {
					continue
				}
				seen[key] = true
				newRows = append(newRows, row)
			}

			for i := len(newRows) - 1; i >= 0; i-- {
				payload, err := renderer.DeploymentRow(ctx, newRows[i], settings.ShowSyncStatus, settings.ShowEnvironmentColumn, settings.StatusSemanticsMode)
				if err != nil {
					return err
				}
				fmt.Fprintf(w, "event: deployment-new\ndata: %s\n\n", payload)
			}
			if len(newRows) > 0 {
				flusher.Flush()
			}
		}
	}
}

func deploymentEventKey(row components.DeploymentRow) string {
	return fmt.Sprintf("%s|%s|%s|%s", row.DeployedAt, row.Service, row.Environment, row.Status)
}
