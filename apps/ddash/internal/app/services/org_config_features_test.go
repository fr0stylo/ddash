package services

import (
	"context"
	"testing"

	"github.com/fr0stylo/ddash/apps/ddash/internal/app/ports"
)

type orgConfigStoreFake struct {
	org          ports.Organization
	features     []ports.OrganizationFeature
	prefs        []ports.OrganizationPreference
	updateParams ports.OrganizationSettingsUpdate
}

func (f *orgConfigStoreFake) GetDefaultOrganization(context.Context) (ports.Organization, error) {
	return f.org, nil
}

func (f *orgConfigStoreFake) GetOrganizationByID(context.Context, int64) (ports.Organization, error) {
	return f.org, nil
}

func (f *orgConfigStoreFake) GetOrganizationByJoinCode(context.Context, string) (ports.Organization, error) {
	return ports.Organization{}, nil
}

func (f *orgConfigStoreFake) ListOrganizations(context.Context) ([]ports.Organization, error) {
	return []ports.Organization{f.org}, nil
}

func (f *orgConfigStoreFake) CreateOrganization(context.Context, ports.CreateOrganizationInput) (ports.Organization, error) {
	return f.org, nil
}

func (f *orgConfigStoreFake) UpdateOrganizationName(context.Context, int64, string) error {
	return nil
}

func (f *orgConfigStoreFake) UpdateOrganizationEnabled(context.Context, int64, bool) error {
	return nil
}

func (f *orgConfigStoreFake) DeleteOrganization(context.Context, int64) error {
	return nil
}

func (f *orgConfigStoreFake) UpsertUser(context.Context, ports.UpsertUserInput) (ports.User, error) {
	return ports.User{}, nil
}

func (f *orgConfigStoreFake) GetUserByID(context.Context, int64) (ports.User, error) {
	return ports.User{}, nil
}

func (f *orgConfigStoreFake) GetUserByEmailOrNickname(context.Context, string, string) (ports.User, error) {
	return ports.User{}, nil
}

func (f *orgConfigStoreFake) ListOrganizationsByUser(context.Context, int64) ([]ports.Organization, error) {
	return []ports.Organization{f.org}, nil
}

func (f *orgConfigStoreFake) GetOrganizationMemberRole(context.Context, int64, int64) (string, error) {
	return "owner", nil
}

func (f *orgConfigStoreFake) UpsertOrganizationMember(context.Context, int64, int64, string) error {
	return nil
}

func (f *orgConfigStoreFake) DeleteOrganizationMember(context.Context, int64, int64) error {
	return nil
}

func (f *orgConfigStoreFake) CountOrganizationOwners(context.Context, int64) (int64, error) {
	return 1, nil
}

func (f *orgConfigStoreFake) ListOrganizationMembers(context.Context, int64) ([]ports.OrganizationMember, error) {
	return nil, nil
}

func (f *orgConfigStoreFake) UpsertOrganizationJoinRequest(context.Context, int64, int64, string) error {
	return nil
}

func (f *orgConfigStoreFake) ListPendingOrganizationJoinRequests(context.Context, int64) ([]ports.OrganizationJoinRequest, error) {
	return nil, nil
}

func (f *orgConfigStoreFake) SetOrganizationJoinRequestStatus(context.Context, int64, int64, string, int64) error {
	return nil
}

func (f *orgConfigStoreFake) ListOrganizationRequiredFields(context.Context, int64) ([]ports.RequiredField, error) {
	return nil, nil
}

func (f *orgConfigStoreFake) ListOrganizationEnvironmentPriorities(context.Context, int64) ([]string, error) {
	return nil, nil
}

func (f *orgConfigStoreFake) ListOrganizationFeatures(context.Context, int64) ([]ports.OrganizationFeature, error) {
	return f.features, nil
}

func (f *orgConfigStoreFake) ListOrganizationPreferences(context.Context, int64) ([]ports.OrganizationPreference, error) {
	return f.prefs, nil
}

func (f *orgConfigStoreFake) ListDistinctServiceEnvironmentsFromEvents(context.Context, int64) ([]string, error) {
	return nil, nil
}

func (f *orgConfigStoreFake) UpdateOrganizationSettings(_ context.Context, _ int64, params ports.OrganizationSettingsUpdate) error {
	f.updateParams = params
	return nil
}

func (f *orgConfigStoreFake) ReplaceServiceMetadata(context.Context, int64, string, []ports.MetadataValue) error {
	return nil
}

func TestOrganizationConfigGetSettingsReadsFeaturesAndPreferences(t *testing.T) {
	store := &orgConfigStoreFake{
		org: ports.Organization{ID: 10, Enabled: true},
		features: []ports.OrganizationFeature{
			{Key: "show_sync_status", Enabled: false},
			{Key: "show_metadata_badges", Enabled: false},
			{Key: "show_environment_column", Enabled: false},
			{Key: "enable_sse_live_updates", Enabled: false},
			{Key: "show_onboarding_hints", Enabled: false},
			{Key: "show_service_detail_insights", Enabled: false},
			{Key: "show_service_dependencies", Enabled: false},
		},
		prefs: []ports.OrganizationPreference{
			{Key: "deployment_retention_days", Value: "14"},
			{Key: "default_dashboard_view", Value: "table"},
			{Key: "status_semantics_mode", Value: "plain"},
		},
	}

	svc := NewOrganizationConfigService(store)
	settings, err := svc.GetSettings(context.Background(), 10)
	if err != nil {
		t.Fatalf("GetSettings error: %v", err)
	}

	if settings.ShowSyncStatus || settings.ShowMetadataBadges || settings.ShowEnvironmentColumn || settings.EnableSSELiveUpdates || settings.ShowOnboardingHints || settings.ShowServiceDetailInsights || settings.ShowServiceDependencies {
		t.Fatalf("expected disabled features from store, got %+v", settings)
	}
	if settings.DeploymentRetentionDays != 14 || settings.DefaultDashboardView != "table" || settings.StatusSemanticsMode != "plain" {
		t.Fatalf("unexpected preferences: days=%d view=%s mode=%s", settings.DeploymentRetentionDays, settings.DefaultDashboardView, settings.StatusSemanticsMode)
	}
}

func TestOrganizationConfigUpdateSettingsForwardsFeatureFields(t *testing.T) {
	store := &orgConfigStoreFake{org: ports.Organization{ID: 10, Enabled: true}}
	svc := NewOrganizationConfigService(store)

	err := svc.UpdateSettings(context.Background(), 10, OrganizationSettingsUpdate{
		AuthToken:                   "a",
		WebhookSecret:               "b",
		Enabled:                     true,
		ShowSyncStatus:              false,
		ShowMetadataBadges:          false,
		ShowEnvironmentColumn:       false,
		EnableSSELiveUpdates:        false,
		ShowDeploymentHistory:       false,
		ShowMetadataFilters:         false,
		StrictMetadataEnforcement:   true,
		MaskSensitiveMetadataValues: true,
		AllowServiceMetadataEditing: false,
		ShowOnboardingHints:         false,
		ShowIntegrationTypeBadges:   false,
		ShowServiceDetailInsights:   false,
		ShowServiceDependencies:     false,
		DeploymentRetentionDays:     7,
		DefaultDashboardView:        "table",
		StatusSemanticsMode:         "plain",
	})
	if err != nil {
		t.Fatalf("UpdateSettings error: %v", err)
	}

	if store.updateParams.ShowSyncStatus || store.updateParams.ShowMetadataBadges || store.updateParams.ShowEnvironmentColumn || store.updateParams.EnableSSELiveUpdates {
		t.Fatalf("expected forwarded disabled flags, got %+v", store.updateParams)
	}
	if store.updateParams.DeploymentRetentionDays != 7 || store.updateParams.DefaultDashboardView != "table" || store.updateParams.StatusSemanticsMode != "plain" {
		t.Fatalf("unexpected forwarded preferences: %+v", store.updateParams)
	}
}
