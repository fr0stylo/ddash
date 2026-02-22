package routes

import (
	"net/http"
	"strconv"

	"github.com/labstack/echo/v4"
)

func (v *ViewRoutes) handleLeadTimeMetrics(c echo.Context) error {
	ctx := c.Request().Context()
	orgID, err := v.currentOrganizationID(c)
	if err != nil {
		return err
	}
	days := 30
	if raw := c.QueryParam("days"); raw != "" {
		if parsed, parseErr := strconv.Atoi(raw); parseErr == nil {
			days = parsed
		}
	}
	report, err := v.read.BuildLeadTimeReport(ctx, orgID, days)
	if err != nil {
		return err
	}
	return c.JSON(http.StatusOK, report)
}
