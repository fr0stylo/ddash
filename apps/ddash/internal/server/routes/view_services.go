package routes

import (
	"database/sql"
	"errors"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/labstack/echo/v4"

	appdomain "github.com/fr0stylo/ddash/apps/ddash/internal/app/domain"
	appservices "github.com/fr0stylo/ddash/apps/ddash/internal/app/services"
	"github.com/fr0stylo/ddash/apps/ddash/internal/renderer"
	"github.com/fr0stylo/ddash/views/components"
	"github.com/fr0stylo/ddash/views/pages"
)

func serviceDetailsRedirectURL(serviceName, message, level string) string {
	values := url.Values{}
	message = strings.TrimSpace(message)
	if message != "" {
		values.Set("msg", message)
	}
	if level == "error" {
		values.Set("level", "error")
	} else {
		values.Set("level", "success")
	}
	path := "/s/" + url.PathEscape(strings.TrimSpace(serviceName))
	query := values.Encode()
	if query == "" {
		return path
	}
	return path + "?" + query
}

func (v *ViewRoutes) handleServiceDetails(c echo.Context) error {
	ctx := c.Request().Context()
	name := c.Param("name")
	orgID, err := v.currentOrganizationID(c)
	if err != nil {
		return err
	}
	if name == "" {
		return c.NoContent(http.StatusNotFound)
	}

	detail, err := v.read.GetServiceDetail(ctx, orgID, name)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return c.NoContent(http.StatusNotFound)
		}
		return err
	}

	settings, err := v.loadDashboardSettings(ctx, orgID)
	if err != nil {
		return err
	}
	if settings.DeploymentRetentionDays > 0 {
		detail.DeploymentHistory = trimDeploymentHistoryByDays(detail.DeploymentHistory, settings.DeploymentRetentionDays)
	}
	if settings.MaskSensitiveMetadataValues {
		detail.MetadataFields = maskSensitiveFields(detail.MetadataFields)
	}
	flashMessage := strings.TrimSpace(c.QueryParam("msg"))
	flashLevel := strings.TrimSpace(c.QueryParam("level"))
	if flashLevel != "error" {
		flashLevel = "success"
	}
	return c.Render(http.StatusOK, "", pages.ServicePage(mapDomainServiceDetail(detail), settings.ShowMetadataBadges, settings.ShowDeploymentHistory, settings.AllowServiceMetadataEditing, settings.ShowIntegrationTypeBadges, settings.ShowServiceDetailInsights, settings.ShowServiceDependencies, flashMessage, flashLevel, csrfToken(c)))
}

func (v *ViewRoutes) handleServiceGrid(c echo.Context) error {
	ctx := c.Request().Context()
	env := c.QueryParam("env")
	orgID, err := v.currentOrganizationID(c)
	if err != nil {
		return err
	}
	services, err := v.read.GetServicesByEnv(ctx, orgID, env)
	if err != nil {
		return err
	}
	settings, err := v.loadDashboardSettings(ctx, orgID)
	if err != nil {
		return err
	}
	return c.Render(http.StatusOK, "", pages.ServiceGridFragment(mapDomainServices(services), settings.ShowSyncStatus, settings.ShowMetadataBadges, settings.ShowEnvironmentColumn, settings.EnableSSELiveUpdates, settings.StatusSemanticsMode))
}

func (v *ViewRoutes) handleServiceTable(c echo.Context) error {
	ctx := c.Request().Context()
	env := c.QueryParam("env")
	orgID, err := v.currentOrganizationID(c)
	if err != nil {
		return err
	}
	services, err := v.read.GetServicesByEnv(ctx, orgID, env)
	if err != nil {
		return err
	}
	settings, err := v.loadDashboardSettings(ctx, orgID)
	if err != nil {
		return err
	}
	return c.Render(http.StatusOK, "", pages.ServiceTableFragment(mapDomainServices(services), settings.ShowSyncStatus, settings.ShowMetadataBadges, settings.ShowEnvironmentColumn, settings.EnableSSELiveUpdates, settings.StatusSemanticsMode))
}

func (v *ViewRoutes) handleServiceFilter(c echo.Context) error {
	ctx := c.Request().Context()
	env := c.QueryParam("env")
	view := c.QueryParam("view")
	orgID, err := v.currentOrganizationID(c)
	if err != nil {
		return err
	}
	services, err := v.read.GetServicesByEnv(ctx, orgID, env)
	if err != nil {
		return err
	}
	settings, err := v.loadDashboardSettings(ctx, orgID)
	if err != nil {
		return err
	}

	if view == "table" {
		return c.Render(http.StatusOK, "", pages.ServiceTableFragment(mapDomainServices(services), settings.ShowSyncStatus, settings.ShowMetadataBadges, settings.ShowEnvironmentColumn, settings.EnableSSELiveUpdates, settings.StatusSemanticsMode))
	}
	return c.Render(http.StatusOK, "", pages.ServiceGridFragment(mapDomainServices(services), settings.ShowSyncStatus, settings.ShowMetadataBadges, settings.ShowEnvironmentColumn, settings.EnableSSELiveUpdates, settings.StatusSemanticsMode))
}

func (v *ViewRoutes) handleServiceStream(c echo.Context) error {
	w := c.Response().Writer
	c.Response().Header().Set("Content-Type", "text/event-stream")
	c.Response().Header().Set("Cache-Control", "no-cache")
	c.Response().Header().Set("Connection", "keep-alive")

	flusher, ok := w.(http.Flusher)
	if !ok {
		return errors.New("streaming unsupported")
	}

	ctx := c.Request().Context()
	orgID, err := v.currentOrganizationID(c)
	if err != nil {
		return err
	}
	view := c.QueryParam("view")
	settings, err := v.loadDashboardSettings(ctx, orgID)
	if err != nil {
		return err
	}
	ticker := time.NewTicker(3 * time.Second)
	defer ticker.Stop()

	services, err := v.read.GetServicesByEnv(ctx, orgID, "all")
	if err != nil {
		return err
	}
	uiServices := mapDomainServices(services)
	indexes := make([]int, len(uiServices))
	for i := range services {
		indexes[i] = i
	}
	if len(indexes) == 0 {
		return nil
	}

	step := 0
	for {
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
			idx := indexes[step%len(indexes)]
			service := uiServices[idx]
			service.Status = nextServiceStatus(service.Status)
			if service.Status == components.StatusSynced {
				service.LastDeploy = time.Now().Format("2006-01-02 15:04")
			}
			uiServices[idx] = service

			switch view {
			case "table":
				rowPayload, err := renderer.ServiceRow(ctx, uiServices, service, settings.ShowSyncStatus, settings.ShowMetadataBadges, settings.ShowEnvironmentColumn, settings.StatusSemanticsMode)
				if err != nil {
					return err
				}
				_, _ = w.Write([]byte("event: " + components.ServiceRowEventName(service) + "\ndata: " + rowPayload + "\n\n"))
			case "grid":
				payload, err := renderer.ServiceCard(ctx, uiServices, service, settings.ShowSyncStatus, settings.ShowMetadataBadges, settings.ShowEnvironmentColumn, settings.StatusSemanticsMode)
				if err != nil {
					return err
				}
				_, _ = w.Write([]byte("event: " + components.ServiceCardEventName(service) + "\ndata: " + payload + "\n\n"))
			default:
				payload, err := renderer.ServiceCard(ctx, uiServices, service, settings.ShowSyncStatus, settings.ShowMetadataBadges, settings.ShowEnvironmentColumn, settings.StatusSemanticsMode)
				if err != nil {
					return err
				}
				_, _ = w.Write([]byte("event: " + components.ServiceCardEventName(service) + "\ndata: " + payload + "\n\n"))
				rowPayload, err := renderer.ServiceRow(ctx, uiServices, service, settings.ShowSyncStatus, settings.ShowMetadataBadges, settings.ShowEnvironmentColumn, settings.StatusSemanticsMode)
				if err != nil {
					return err
				}
				_, _ = w.Write([]byte("event: " + components.ServiceRowEventName(service) + "\ndata: " + rowPayload + "\n\n"))
			}
			flusher.Flush()
			step++
		}
	}
}

type metadataPayload struct {
	Fields []metadataFieldInput `json:"fields"`
}

type metadataFieldInput struct {
	Label string `json:"label"`
	Value string `json:"value"`
}

func (v *ViewRoutes) handleServiceMetadataUpdate(c echo.Context) error {
	ctx := c.Request().Context()
	orgID, err := v.currentOrganizationID(c)
	if err != nil {
		return err
	}
	serviceName := strings.TrimSpace(c.Param("name"))
	if serviceName == "" {
		return c.NoContent(http.StatusBadRequest)
	}

	payload := metadataPayload{}
	if err := c.Bind(&payload); err != nil {
		return err
	}

	updates := make([]appservices.MetadataFieldUpdate, 0, len(payload.Fields))
	for _, field := range payload.Fields {
		updates = append(updates, appservices.MetadataFieldUpdate{Label: field.Label, Value: field.Value})
	}

	settings, err := v.loadDashboardSettings(ctx, orgID)
	if err != nil {
		return err
	}
	if !settings.AllowServiceMetadataEditing {
		return c.NoContent(http.StatusForbidden)
	}
	if err := v.metadata.UpdateServiceMetadata(ctx, orgID, serviceName, updates, settings.StrictMetadataEnforcement); err != nil {
		return err
	}

	return c.NoContent(http.StatusNoContent)
}

func (v *ViewRoutes) handleServiceDependencyUpsert(c echo.Context) error {
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

	serviceName := strings.TrimSpace(c.Param("name"))
	dependsOn := strings.TrimSpace(c.FormValue("depends_on"))
	if err := v.read.UpsertServiceDependency(ctx, orgID, serviceName, dependsOn); err != nil {
		return err
	}
	return c.Redirect(http.StatusFound, serviceDetailsRedirectURL(serviceName, "Dependency added", "success"))
}

func (v *ViewRoutes) handleServiceDependencyDelete(c echo.Context) error {
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

	serviceName := strings.TrimSpace(c.Param("name"))
	dependsOn := strings.TrimSpace(c.FormValue("depends_on"))
	if err := v.read.DeleteServiceDependency(ctx, orgID, serviceName, dependsOn); err != nil {
		return err
	}
	return c.Redirect(http.StatusFound, serviceDetailsRedirectURL(serviceName, "Dependency removed", "success"))
}

func trimDeploymentHistoryByDays(rows []appdomain.DeploymentRecord, days int) []appdomain.DeploymentRecord {
	if days <= 0 {
		return rows
	}
	cutoff := time.Now().AddDate(0, 0, -days)
	out := make([]appdomain.DeploymentRecord, 0, len(rows))
	for _, row := range rows {
		parsed, err := time.Parse("2006-01-02 15:04", row.DeployedAt)
		if err == nil && parsed.Before(cutoff) {
			continue
		}
		out = append(out, row)
	}
	return out
}

func maskSensitiveFields(fields []appdomain.MetadataField) []appdomain.MetadataField {
	out := make([]appdomain.MetadataField, 0, len(fields))
	for _, field := range fields {
		item := field
		label := strings.ToLower(strings.TrimSpace(field.Label))
		if strings.Contains(label, "secret") || strings.Contains(label, "token") || strings.Contains(label, "password") || strings.Contains(label, "key") {
			if strings.TrimSpace(item.Value) != "" {
				item.Value = "***"
			}
		}
		out = append(out, item)
	}
	return out
}
