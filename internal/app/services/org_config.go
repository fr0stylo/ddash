package services

import (
	"context"
	"sort"
	"strconv"
	"strings"

	"github.com/fr0stylo/ddash/internal/app/domain"
	"github.com/fr0stylo/ddash/internal/app/ports"
)

const (
	featureShowSyncStatus              = "show_sync_status"
	featureShowMetadataBadges          = "show_metadata_badges"
	featureShowEnvironmentColumn       = "show_environment_column"
	featureEnableSSELiveUpdates        = "enable_sse_live_updates"
	featureShowDeploymentHistory       = "show_deployment_history"
	featureShowMetadataFilters         = "show_metadata_filters"
	featureStrictMetadataEnforcement   = "strict_metadata_enforcement"
	featureMaskSensitiveMetadataValues = "mask_sensitive_metadata_values"
	featureAllowServiceMetadataEditing = "allow_service_metadata_editing"
	featureShowOnboardingHints         = "show_onboarding_hints"
	featureShowIntegrationTypeBadges   = "show_integration_type_badges"
	featureShowServiceDetailInsights   = "show_service_detail_insights"

	prefDeploymentRetentionDays = "deployment_retention_days"
	prefDefaultDashboardView    = "default_dashboard_view"
	prefStatusSemanticsMode     = "status_semantics_mode"
)

// OrganizationConfigService provides org-level settings read/write operations.
type OrganizationConfigService struct {
	store ports.AppStore
}

// NewOrganizationConfigService constructs organization config service.
func NewOrganizationConfigService(store ports.AppStore) *OrganizationConfigService {
	return &OrganizationConfigService{store: store}
}

// OrganizationSettings contains values rendered on settings page.
type OrganizationSettings struct {
	AuthToken                   string
	WebhookSecret               string
	Enabled                     bool
	ShowSyncStatus              bool
	ShowMetadataBadges          bool
	ShowEnvironmentColumn       bool
	EnableSSELiveUpdates        bool
	ShowDeploymentHistory       bool
	ShowMetadataFilters         bool
	StrictMetadataEnforcement   bool
	MaskSensitiveMetadataValues bool
	AllowServiceMetadataEditing bool
	ShowOnboardingHints         bool
	ShowIntegrationTypeBadges   bool
	ShowServiceDetailInsights   bool
	DeploymentRetentionDays     int
	DefaultDashboardView        string
	StatusSemanticsMode         string
	RequiredFields              []domain.MetadataField
	EnvironmentOrder            []string
}

// RequiredFieldInput is one required metadata field definition.
type RequiredFieldInput struct {
	Label      string
	Type       string
	Filterable bool
}

// OrganizationSettingsUpdate contains settings update payload.
type OrganizationSettingsUpdate struct {
	AuthToken                   string
	WebhookSecret               string
	Enabled                     bool
	ShowSyncStatus              bool
	ShowMetadataBadges          bool
	ShowEnvironmentColumn       bool
	EnableSSELiveUpdates        bool
	ShowDeploymentHistory       bool
	ShowMetadataFilters         bool
	StrictMetadataEnforcement   bool
	MaskSensitiveMetadataValues bool
	AllowServiceMetadataEditing bool
	ShowOnboardingHints         bool
	ShowIntegrationTypeBadges   bool
	ShowServiceDetailInsights   bool
	DeploymentRetentionDays     int
	DefaultDashboardView        string
	StatusSemanticsMode         string
	RequiredFields              []RequiredFieldInput
	EnvironmentOrder            []string
}

// GetSettings returns organization settings view model.
func (s *OrganizationConfigService) GetSettings(ctx context.Context, organizationID int64) (OrganizationSettings, error) {
	org, err := s.store.GetOrganizationByID(ctx, organizationID)
	if err != nil {
		return OrganizationSettings{}, err
	}
	fields, err := s.store.ListOrganizationRequiredFields(ctx, org.ID)
	if err != nil {
		return OrganizationSettings{}, err
	}
	envPriorities, err := s.store.ListOrganizationEnvironmentPriorities(ctx, org.ID)
	if err != nil && !isMissingEnvPriorityTableErr(err) {
		return OrganizationSettings{}, err
	}
	if isMissingEnvPriorityTableErr(err) {
		envPriorities = nil
	}
	discoveredEnvs, err := s.store.ListDistinctServiceEnvironmentsFromEvents(ctx, org.ID)
	if err != nil {
		return OrganizationSettings{}, err
	}
	features, err := s.store.ListOrganizationFeatures(ctx, org.ID)
	if err != nil {
		return OrganizationSettings{}, err
	}
	featureFlags := map[string]bool{
		featureShowSyncStatus:              true,
		featureShowMetadataBadges:          true,
		featureShowEnvironmentColumn:       true,
		featureEnableSSELiveUpdates:        true,
		featureShowDeploymentHistory:       true,
		featureShowMetadataFilters:         true,
		featureStrictMetadataEnforcement:   false,
		featureMaskSensitiveMetadataValues: false,
		featureAllowServiceMetadataEditing: true,
		featureShowOnboardingHints:         true,
		featureShowIntegrationTypeBadges:   true,
		featureShowServiceDetailInsights:   true,
	}
	for _, feature := range features {
		key := strings.ToLower(strings.TrimSpace(feature.Key))
		if _, ok := featureFlags[key]; ok {
			featureFlags[key] = feature.Enabled
		}
	}

	prefs, err := s.store.ListOrganizationPreferences(ctx, org.ID)
	if err != nil {
		return OrganizationSettings{}, err
	}
	deploymentRetentionDays := 30
	defaultDashboardView := "grid"
	statusSemanticsMode := "technical"
	for _, preference := range prefs {
		key := strings.ToLower(strings.TrimSpace(preference.Key))
		value := strings.TrimSpace(preference.Value)
		switch key {
		case prefDeploymentRetentionDays:
			if parsed, convErr := strconv.Atoi(value); convErr == nil && parsed > 0 {
				deploymentRetentionDays = parsed
			}
		case prefDefaultDashboardView:
			if value == "table" || value == "grid" {
				defaultDashboardView = value
			}
		case prefStatusSemanticsMode:
			if value == "plain" || value == "technical" {
				statusSemanticsMode = value
			}
		}
	}

	return OrganizationSettings{
		AuthToken:                   org.AuthToken,
		WebhookSecret:               org.WebhookSecret,
		Enabled:                     org.Enabled,
		ShowSyncStatus:              featureFlags[featureShowSyncStatus],
		ShowMetadataBadges:          featureFlags[featureShowMetadataBadges],
		ShowEnvironmentColumn:       featureFlags[featureShowEnvironmentColumn],
		EnableSSELiveUpdates:        featureFlags[featureEnableSSELiveUpdates],
		ShowDeploymentHistory:       featureFlags[featureShowDeploymentHistory],
		ShowMetadataFilters:         featureFlags[featureShowMetadataFilters],
		StrictMetadataEnforcement:   featureFlags[featureStrictMetadataEnforcement],
		MaskSensitiveMetadataValues: featureFlags[featureMaskSensitiveMetadataValues],
		AllowServiceMetadataEditing: featureFlags[featureAllowServiceMetadataEditing],
		ShowOnboardingHints:         featureFlags[featureShowOnboardingHints],
		ShowIntegrationTypeBadges:   featureFlags[featureShowIntegrationTypeBadges],
		ShowServiceDetailInsights:   featureFlags[featureShowServiceDetailInsights],
		DeploymentRetentionDays:     deploymentRetentionDays,
		DefaultDashboardView:        defaultDashboardView,
		StatusSemanticsMode:         statusSemanticsMode,
		RequiredFields:              normalizeSettingsFields(fields),
		EnvironmentOrder:            mergeEnvironmentOrder(normalizeEnvironmentOrder(envPriorities), normalizeEnvironmentOrderInput(discoveredEnvs)),
	}, nil
}

// UpdateSettings updates one organization settings.
func (s *OrganizationConfigService) UpdateSettings(ctx context.Context, organizationID int64, update OrganizationSettingsUpdate) error {
	update.AuthToken = strings.TrimSpace(update.AuthToken)
	update.WebhookSecret = strings.TrimSpace(update.WebhookSecret)
	update.EnvironmentOrder = normalizeEnvironmentOrderInput(update.EnvironmentOrder)

	requiredFields := make([]ports.RequiredField, 0, len(update.RequiredFields))
	for _, field := range update.RequiredFields {
		label := strings.TrimSpace(field.Label)
		fieldType := strings.TrimSpace(field.Type)
		if label == "" || fieldType == "" {
			continue
		}
		requiredFields = append(requiredFields, ports.RequiredField{
			Label:      label,
			Type:       fieldType,
			Filterable: field.Filterable,
		})
	}

	return s.store.UpdateOrganizationSettings(ctx, organizationID, ports.OrganizationSettingsUpdate{
		AuthToken:                   update.AuthToken,
		WebhookSecret:               update.WebhookSecret,
		Enabled:                     update.Enabled,
		ShowSyncStatus:              update.ShowSyncStatus,
		ShowMetadataBadges:          update.ShowMetadataBadges,
		ShowEnvironmentColumn:       update.ShowEnvironmentColumn,
		EnableSSELiveUpdates:        update.EnableSSELiveUpdates,
		ShowDeploymentHistory:       update.ShowDeploymentHistory,
		ShowMetadataFilters:         update.ShowMetadataFilters,
		StrictMetadataEnforcement:   update.StrictMetadataEnforcement,
		MaskSensitiveMetadataValues: update.MaskSensitiveMetadataValues,
		AllowServiceMetadataEditing: update.AllowServiceMetadataEditing,
		ShowOnboardingHints:         update.ShowOnboardingHints,
		ShowIntegrationTypeBadges:   update.ShowIntegrationTypeBadges,
		ShowServiceDetailInsights:   update.ShowServiceDetailInsights,
		DeploymentRetentionDays:     update.DeploymentRetentionDays,
		DefaultDashboardView:        update.DefaultDashboardView,
		StatusSemanticsMode:         update.StatusSemanticsMode,
		RequiredFields:              requiredFields,
		EnvironmentOrder:            update.EnvironmentOrder,
	})
}

func normalizeSettingsFields(rows []ports.RequiredField) []domain.MetadataField {
	fields := make([]domain.MetadataField, 0, len(rows))
	for _, row := range rows {
		fields = append(fields, domain.MetadataField{
			Label:      row.Label,
			Value:      row.Type,
			Filterable: row.Filterable,
		})
	}
	return fields
}

func mergeEnvironmentOrder(prioritized, discovered []string) []string {
	merged := normalizeEnvironmentOrderInput(prioritized)
	seen := map[string]bool{}
	for _, value := range merged {
		seen[strings.ToLower(value)] = true
	}

	rest := make([]string, 0, len(discovered))
	for _, value := range normalizeEnvironmentOrderInput(discovered) {
		key := strings.ToLower(value)
		if seen[key] {
			continue
		}
		seen[key] = true
		rest = append(rest, value)
	}
	sort.Strings(rest)

	return append(merged, rest...)
}

func isMissingEnvPriorityTableErr(err error) bool {
	if err == nil {
		return false
	}
	message := strings.ToLower(err.Error())
	return strings.Contains(message, "no such table") && strings.Contains(message, "organization_environment_priorities")
}
