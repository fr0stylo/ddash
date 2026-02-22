package routes

import (
	"net/http"

	"github.com/labstack/echo/v4"

	"github.com/fr0stylo/ddash/views/pages"
)

func (v *ViewRoutes) handleServiceGraphPage(c echo.Context) error {
	ctx := c.Request().Context()
	orgID, err := v.currentOrganizationID(c)
	if err != nil {
		return err
	}
	settings, err := v.loadDashboardSettings(ctx, orgID)
	if err != nil {
		return err
	}
	if !settings.ShowServiceDependencies {
		return c.NoContent(http.StatusForbidden)
	}
	return c.Render(http.StatusOK, "", pages.ServiceGraphPage())
}

func (v *ViewRoutes) handleServiceGraphData(c echo.Context) error {
	ctx := c.Request().Context()
	orgID, err := v.currentOrganizationID(c)
	if err != nil {
		return err
	}
	settings, err := v.loadDashboardSettings(ctx, orgID)
	if err != nil {
		return err
	}
	if !settings.ShowServiceDependencies {
		return c.NoContent(http.StatusForbidden)
	}

	graph, err := v.read.BuildDependencyGraph(ctx, orgID)
	if err != nil {
		return err
	}
	return c.JSON(http.StatusOK, graph)
}
