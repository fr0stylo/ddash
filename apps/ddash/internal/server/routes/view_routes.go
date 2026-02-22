package routes

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/labstack/echo/v4"

	"github.com/fr0stylo/ddash/apps/ddash/internal/app/ports"
	appservices "github.com/fr0stylo/ddash/apps/ddash/internal/app/services"
	appgithub "github.com/fr0stylo/ddash/apps/ddash/internal/application/githubintegration"
	appidentity "github.com/fr0stylo/ddash/apps/ddash/internal/application/identity"
	apporgconfig "github.com/fr0stylo/ddash/apps/ddash/internal/application/orgconfig"
	appcatalog "github.com/fr0stylo/ddash/apps/ddash/internal/application/servicecatalog"
	"github.com/fr0stylo/ddash/apps/ddash/internal/renderer"
)

// ViewRoutes wires view routes with database access.
type ViewRoutes struct {
	read              *appcatalog.Service
	metadata          *appservices.MetadataService
	config            *apporgconfig.Service
	orgs              *appidentity.Service
	githubIntegration *appgithub.Service
	fragments         *renderer.FragmentRenderer
}

type ViewExternalConfig struct {
	PublicURL           string
	GitHubAppInstallURL string
	GitHubIngestorToken string
}

// NewViewRoutes constructs view routes.
func NewViewRoutes(configStore ports.AppStore, readStore ports.ServiceReadStore, installStore ports.GitHubInstallationStore, external ViewExternalConfig) *ViewRoutes {
	return &ViewRoutes{
		read:              appcatalog.NewService(readStore),
		metadata:          appservices.NewMetadataService(configStore),
		config:            apporgconfig.NewService(configStore),
		orgs:              appidentity.NewService(configStore),
		githubIntegration: appgithub.NewService(installStore, NewGitHubIngestorClient(external.GitHubAppInstallURL, external.GitHubIngestorToken, external.PublicURL)),
		fragments:         renderer.NewFragmentRenderer(512, 5*time.Second),
	}
}

// RegisterRoutes registers view routes.
func (v *ViewRoutes) RegisterRoutes(s *echo.Echo) {
	s.GET("/settings/integrations/github/callback", v.handleGitHubIntegrationCallback)

	authed := s.Group("", RequireAuth)
	authed.GET("/welcome", v.handleWelcome)
	authed.POST("/welcome/create", v.handleWelcomeCreateOrganization)
	authed.POST("/welcome/join", v.handleWelcomeJoinOrganization)

	orgAuthed := authed.Group("", v.requireOrganizationMembership)

	orgAuthed.GET("/", v.handleHome)
	orgAuthed.GET("/s/:name", v.handleServiceDetails)
	orgAuthed.GET("/services/graph", v.handleServiceGraphPage)
	orgAuthed.GET("/api/services/graph", v.handleServiceGraphData)
	orgAuthed.GET("/api/metrics/lead-time", v.handleLeadTimeMetrics)
	orgAuthed.POST("/s/:name/metadata", v.handleServiceMetadataUpdate)
	orgAuthed.POST("/s/:name/dependencies", v.handleServiceDependencyUpsert)
	orgAuthed.POST("/s/:name/dependencies/delete", v.handleServiceDependencyDelete)
	orgAuthed.GET("/settings", v.handleSettings)
	orgAuthed.POST("/settings", v.handleSettingsUpdate)
	orgAuthed.GET("/settings/integrations/github", v.handleGitHubIntegration)
	orgAuthed.POST("/settings/integrations/github/link", v.handleGitHubIntegrationLink)
	orgAuthed.POST("/settings/integrations/github/delete", v.handleGitHubIntegrationDelete)
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
	orgAuthed.GET("/deployments/filter", v.handleDeploymentFilter, v.htmxFragmentCacheMiddleware("deployments-filter"))
	orgAuthed.GET("/services/stream", v.handleServiceStream)

	orgAuthed.GET("/services/grid", v.handleServiceGrid, v.htmxFragmentCacheMiddleware("services-grid"))
	orgAuthed.GET("/services/table", v.handleServiceTable, v.htmxFragmentCacheMiddleware("services-table"))
	orgAuthed.GET("/services/filter", v.handleServiceFilter, v.htmxFragmentCacheMiddleware("services-filter"))
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
			if errors.Is(err, appidentity.ErrOrganizationMembershipRequired) {
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
