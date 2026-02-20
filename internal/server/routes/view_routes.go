package routes

import (
	"context"
	"errors"
	"net/http"

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
	authed.GET("/welcome", v.handleWelcome)
	authed.POST("/welcome/create", v.handleWelcomeCreateOrganization)
	authed.POST("/welcome/join", v.handleWelcomeJoinOrganization)

	orgAuthed := authed.Group("", v.requireOrganizationMembership)

	orgAuthed.GET("/", v.handleHome)
	orgAuthed.GET("/s/:name", v.handleServiceDetails)
	orgAuthed.POST("/s/:name/metadata", v.handleServiceMetadataUpdate)
	orgAuthed.GET("/settings", v.handleSettings)
	orgAuthed.POST("/settings", v.handleSettingsUpdate)
	orgAuthed.GET("/organizations", v.handleOrganizations)
	orgAuthed.GET("/organizations/current", v.handleOrganizationCurrent)
	orgAuthed.POST("/organizations", v.handleOrganizationCreate)
	orgAuthed.POST("/organizations/rename", v.handleOrganizationRename)
	orgAuthed.POST("/organizations/toggle", v.handleOrganizationToggle)
	orgAuthed.POST("/organizations/delete", v.handleOrganizationDelete)
	orgAuthed.POST("/organizations/switch", v.handleOrganizationSwitch)
	orgAuthed.POST("/organizations/members/add", v.handleOrganizationMemberAdd)
	orgAuthed.POST("/organizations/members/role", v.handleOrganizationMemberRole)
	orgAuthed.POST("/organizations/members/remove", v.handleOrganizationMemberRemove)
	orgAuthed.POST("/organizations/join-requests/approve", v.handleOrganizationJoinRequestApprove)
	orgAuthed.POST("/organizations/join-requests/reject", v.handleOrganizationJoinRequestReject)
	orgAuthed.GET("/onboarding", v.handleOnboarding)

	orgAuthed.GET("/deployments", v.handleDeployments)
	orgAuthed.GET("/deployments/filter", v.handleDeploymentFilter)
	orgAuthed.GET("/services/stream", v.handleServiceStream)

	orgAuthed.GET("/services/grid", v.handleServiceGrid)
	orgAuthed.GET("/services/table", v.handleServiceTable)
	orgAuthed.GET("/services/filter", v.handleServiceFilter)
	orgAuthed.GET("/deployments/stream", v.handleDeploymentStream)
}

func (v *ViewRoutes) requireOrganizationMembership(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		ctx := c.Request().Context()
		userID, ok := GetAuthUserID(c)
		if !ok || userID <= 0 {
			return c.Redirect(http.StatusFound, "/login")
		}
		activeID, _ := GetActiveOrganizationID(c)
		org, err := v.orgs.GetActiveOrDefaultOrganizationForUser(ctx, userID, activeID)
		if err != nil {
			if errors.Is(err, appservices.ErrOrganizationMembershipRequired) {
				return c.Redirect(http.StatusFound, "/welcome")
			}
			return err
		}
		if err := SetActiveOrganizationID(c, org.ID); err != nil {
			return err
		}
		return next(c)
	}
}

func (v *ViewRoutes) loadDashboardSettings(ctx context.Context, organizationID int64) (appservices.OrganizationSettings, error) {
	settings, err := v.config.GetSettings(ctx, organizationID)
	if err != nil {
		return appservices.OrganizationSettings{}, err
	}
	return settings, nil
}
