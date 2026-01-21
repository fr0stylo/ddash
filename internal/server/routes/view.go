package routes

import (
	"database/sql"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/labstack/echo/v4"

	"github.com/fr0stylo/ddash/internal/db"
	"github.com/fr0stylo/ddash/internal/renderer"
	"github.com/fr0stylo/ddash/views/components"
	"github.com/fr0stylo/ddash/views/pages"
)

// ViewRoutes wires view routes with database access.
type ViewRoutes struct {
	db *db.Database
}

// NewViewRoutes constructs view routes.
func NewViewRoutes(database *db.Database) *ViewRoutes {
	return &ViewRoutes{db: database}
}

// RegisterRoutes registers view routes.
func (v *ViewRoutes) RegisterRoutes(s *echo.Echo) {
	s.GET("/", v.handleHome)
	s.GET("/s/:name", v.handleServiceDetails)
	s.GET("/settings", v.handleSettings)
	s.POST("/settings", v.handleSettingsUpdate)

	s.GET("/deployments", v.handleDeployments)
	s.GET("/deployments/filter", v.handleDeploymentFilter)
	s.GET("/services/stream", v.handleServiceStream)
	s.GET("/services/grid", v.handleServiceGrid)
	s.GET("/services/table", v.handleServiceTable)
	s.GET("/services/filter", v.handleServiceFilter)
	s.GET("/deployments/stream", v.handleDeploymentStream)
}

func (v *ViewRoutes) handleHome(c echo.Context) error {
	ctx := c.Request().Context()
	rows, err := v.db.ListServiceInstances(ctx)
	if err != nil {
		return err
	}
	return c.Render(http.StatusOK, "", pages.HomePage(mapServiceInstances(rows)))
}

func (v *ViewRoutes) handleDeployments(c echo.Context) error {
	ctx := c.Request().Context()
	rows, err := v.db.ListDeployments(ctx, db.ListDeploymentsParams{
		Env:     "",
		Service: "",
	})
	if err != nil {
		return err
	}
	return c.Render(http.StatusOK, "", pages.DeploymentsPage(mapDeploymentRows(rows)))
}

func (v *ViewRoutes) handleServiceDetails(c echo.Context) error {
	ctx := c.Request().Context()
	name := c.Param("name")
	if name == "" {
		name = "payments-api"
	}

	service, err := v.db.GetServiceByName(ctx, name)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return c.NoContent(http.StatusNotFound)
		}
		return err
	}

	fields, err := v.db.ListServiceFields(ctx, service.ID)
	if err != nil {
		return err
	}
	org, err := v.db.GetDefaultOrganization(ctx)
	if err != nil {
		return err
	}
	orgFields, err := v.db.ListOrganizationRequiredFields(ctx, org.ID)
	if err != nil {
		return err
	}
	serviceEnvs, err := v.db.ListServiceEnvironments(ctx, service.ID)
	if err != nil {
		return err
	}
	commits, err := v.db.ListPendingCommitsNotInProd(ctx, db.ListPendingCommitsNotInProdParams{
		ServiceID: service.ID,
		Limit:     6,
	})
	if err != nil {
		return err
	}
	history, err := v.db.ListDeploymentHistoryByService(ctx, db.ListDeploymentHistoryByServiceParams{
		ServiceID: service.ID,
		Limit:     10,
	})
	if err != nil {
		return err
	}

	detail := components.ServiceDetail{
		Title:             service.Name,
		Description:       nullString(service.Description),
		Context:           nullString(service.Context),
		Team:              nullString(service.Team),
		IntegrationType:   service.IntegrationType,
		CustomFields:      mapServiceFields(fields),
		OrgRequiredFields: mapOrganizationRequiredFields(orgFields),
		Environments:      mapServiceEnvironments(serviceEnvs),
		PendingCommits:    mapPendingCommits(commits),
		DeploymentHistory: mapDeploymentHistory(history),
	}

	return c.Render(http.StatusOK, "", pages.ServicePage(detail))
}

func (v *ViewRoutes) handleDeploymentFilter(c echo.Context) error {
	ctx := c.Request().Context()
	env := c.QueryParam("env")
	service := c.QueryParam("service")

	rows, err := v.db.ListDeployments(ctx, db.ListDeploymentsParams{
		Env:     env,
		Service: service,
	})
	if err != nil {
		return err
	}

	return c.Render(http.StatusOK, "", pages.DeploymentResults(mapDeploymentRows(rows), env, service))
}

func (v *ViewRoutes) handleServiceStream(c echo.Context) error {
	w := c.Response().Writer
	c.Response().Header().Set("Content-Type", "text/event-stream")
	c.Response().Header().Set("Cache-Control", "no-cache")
	c.Response().Header().Set("Connection", "keep-alive")

	flusher, ok := w.(http.Flusher)
	if !ok {
		return fmt.Errorf("streaming unsupported")
	}

	ctx := c.Request().Context()
	view := c.QueryParam("view")
	ticker := time.NewTicker(3 * time.Second)
	defer ticker.Stop()

	rows, err := v.db.ListServiceInstances(ctx)
	if err != nil {
		return err
	}
	services := mapServiceInstances(rows)
	indexes := make([]int, len(services))
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
			service := services[idx]
			service.Status = nextServiceStatus(service.Status)
			if service.Status == components.StatusSynced {
				service.LastDeploy = time.Now().Format("2006-01-02 15:04")
			}
			services[idx] = service

			switch view {
			case "table":
				rowPayload, err := renderer.ServiceRow(ctx, services, service)
				if err != nil {
					return err
				}
				fmt.Fprintf(w, "event: %s\ndata: %s\n\n", components.ServiceRowEventName(service), rowPayload)
			case "grid":
				payload, err := renderer.ServiceCard(ctx, services, service)
				if err != nil {
					return err
				}
				fmt.Fprintf(w, "event: %s\ndata: %s\n\n", components.ServiceCardEventName(service), payload)
			default:
				payload, err := renderer.ServiceCard(ctx, services, service)
				if err != nil {
					return err
				}
				fmt.Fprintf(w, "event: %s\ndata: %s\n\n", components.ServiceCardEventName(service), payload)
				rowPayload, err := renderer.ServiceRow(ctx, services, service)
				if err != nil {
					return err
				}
				fmt.Fprintf(w, "event: %s\ndata: %s\n\n", components.ServiceRowEventName(service), rowPayload)
			}
			flusher.Flush()
			step++
		}
	}
}

func (v *ViewRoutes) handleServiceGrid(c echo.Context) error {
	ctx := c.Request().Context()
	env := c.QueryParam("env")
	if env == "" || env == "all" {
		rows, err := v.db.ListServiceInstances(ctx)
		if err != nil {
			return err
		}
		return c.Render(http.StatusOK, "", pages.ServiceGridFragment(mapServiceInstances(rows)))
	}

	rows, err := v.db.ListServiceInstancesByEnv(ctx, env)
	if err != nil {
		return err
	}
	return c.Render(http.StatusOK, "", pages.ServiceGridFragment(mapServiceInstancesByEnv(rows)))
}

func (v *ViewRoutes) handleServiceTable(c echo.Context) error {
	ctx := c.Request().Context()
	env := c.QueryParam("env")
	if env == "" || env == "all" {
		rows, err := v.db.ListServiceInstances(ctx)
		if err != nil {
			return err
		}
		return c.Render(http.StatusOK, "", pages.ServiceTableFragment(mapServiceInstances(rows)))
	}

	rows, err := v.db.ListServiceInstancesByEnv(ctx, env)
	if err != nil {
		return err
	}
	return c.Render(http.StatusOK, "", pages.ServiceTableFragment(mapServiceInstancesByEnv(rows)))
}

func (v *ViewRoutes) handleServiceFilter(c echo.Context) error {
	ctx := c.Request().Context()
	env := c.QueryParam("env")
	view := c.QueryParam("view")

	var services []components.Service
	if env == "" || env == "all" {
		rows, err := v.db.ListServiceInstances(ctx)
		if err != nil {
			return err
		}
		services = mapServiceInstances(rows)
	} else {
		rows, err := v.db.ListServiceInstancesByEnv(ctx, env)
		if err != nil {
			return err
		}
		services = mapServiceInstancesByEnv(rows)
	}

	if view == "table" {
		return c.Render(http.StatusOK, "", pages.ServiceTableFragment(services))
	}
	return c.Render(http.StatusOK, "", pages.ServiceGridFragment(services))
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
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	type pendingUpdate struct {
		at     time.Time
		row    components.DeploymentRow
		status components.DeploymentStatus
	}

	envFilter := c.QueryParam("env")
	serviceFilter := c.QueryParam("service")

	pending := []pendingUpdate{}
	counter := 0
	for {
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
			now := time.Now()
			if len(pending) > 0 {
				next := []pendingUpdate{}
				for _, item := range pending {
					if now.Before(item.at) {
						next = append(next, item)
						continue
					}
					item.row.Status = item.status
					update, err := renderer.DeploymentRow(ctx, item.row)
					if err != nil {
						return err
					}
					fmt.Fprintf(w, "event: %s\ndata: %s\n\n", components.DeploymentRowEventName(item.row), update)
				}
				pending = next
				flusher.Flush()
			}

			counter++
			row := components.DeploymentRow{
				Service:     fmt.Sprintf("jobs-worker-%d", counter),
				Environment: "staging",
				DeployedAt:  now.Format("2006-01-02 15:04:05"),
				Status:      components.DeploymentProcessing,
				JobURL:      fmt.Sprintf("https://ci.example.com/jobs/%d", 1900+counter),
			}
			if deploymentMatchesFilter(row, envFilter, serviceFilter) {
				payload, err := renderer.DeploymentRow(ctx, row)
				if err != nil {
					return err
				}
				fmt.Fprintf(w, "event: deployment-new\ndata: %s\n\n", payload)
				flusher.Flush()

				nextStatus := components.DeploymentSuccess
				if counter%3 == 0 {
					nextStatus = components.DeploymentError
				}
				pending = append(pending, pendingUpdate{
					at:     now.Add(4 * time.Second),
					row:    row,
					status: nextStatus,
				})
			}
		}
	}
}
