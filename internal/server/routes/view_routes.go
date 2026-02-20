package routes

import (
	"context"

	"github.com/labstack/echo/v4"

	"github.com/fr0stylo/ddash/internal/app/ports"
	appservices "github.com/fr0stylo/ddash/internal/app/services"
)

// ViewRoutes wires view routes with database access.
type ViewRoutes struct {
	read     *appservices.ServiceReadService
	metadata *appservices.MetadataService
	config   *appservices.OrganizationConfigService
	orgs     *appservices.OrganizationManagementService
}

// NewViewRoutes constructs view routes.
func NewViewRoutes(configStore ports.AppStore, readStore ports.ServiceReadStore) *ViewRoutes {
	return &ViewRoutes{
		read:     appservices.NewServiceReadService(readStore),
		metadata: appservices.NewMetadataService(configStore),
		config:   appservices.NewOrganizationConfigService(configStore),
		orgs:     appservices.NewOrganizationManagementService(configStore),
	}
}

// RegisterRoutes registers view routes.
func (v *ViewRoutes) RegisterRoutes(s *echo.Echo) {
	authed := s.Group("", RequireAuth)

	authed.GET("/", v.handleHome)
	authed.GET("/s/:name", v.handleServiceDetails)
	authed.POST("/s/:name/metadata", v.handleServiceMetadataUpdate)
	authed.GET("/settings", v.handleSettings)
	authed.POST("/settings", v.handleSettingsUpdate)
	authed.GET("/organizations", v.handleOrganizations)
	authed.GET("/organizations/current", v.handleOrganizationCurrent)
	authed.POST("/organizations", v.handleOrganizationCreate)
	authed.POST("/organizations/rename", v.handleOrganizationRename)
	authed.POST("/organizations/toggle", v.handleOrganizationToggle)
	authed.POST("/organizations/delete", v.handleOrganizationDelete)
	authed.POST("/organizations/switch", v.handleOrganizationSwitch)
	authed.GET("/organizations/members", v.handleOrganizationMembers)
	authed.POST("/organizations/members/add", v.handleOrganizationMemberAdd)
	authed.POST("/organizations/members/role", v.handleOrganizationMemberRole)
	authed.POST("/organizations/members/remove", v.handleOrganizationMemberRemove)
	authed.GET("/onboarding", v.handleOnboarding)

	authed.GET("/deployments", v.handleDeployments)
	authed.GET("/deployments/filter", v.handleDeploymentFilter)
	authed.GET("/services/stream", v.handleServiceStream)

	authed.GET("/services/grid", v.handleServiceGrid)
	authed.GET("/services/table", v.handleServiceTable)
	authed.GET("/services/filter", v.handleServiceFilter)
	authed.GET("/deployments/stream", v.handleDeploymentStream)
}

func (v *ViewRoutes) loadDashboardSettings(ctx context.Context, organizationID int64) (appservices.OrganizationSettings, error) {
	settings, err := v.config.GetSettings(ctx, organizationID)
	if err != nil {
		return appservices.OrganizationSettings{}, err
	}
	return settings, nil
}
