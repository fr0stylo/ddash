package sqlite

import (
	"context"
	"database/sql"
	"strconv"
	"strings"

	"github.com/fr0stylo/ddash/apps/ddash/internal/app/ports"
	"github.com/fr0stylo/ddash/internal/db/queries"
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
	featureShowServiceDependencies     = "show_service_dependencies"

	prefDeploymentRetentionDays = "deployment_retention_days"
	prefDefaultDashboardView    = "default_dashboard_view"
	prefStatusSemanticsMode     = "status_semantics_mode"
)

// Store is the sqlite/sqlc-backed implementation of AppStore.
// Future adapters (gRPC, ClickHouse) should implement the same port.
type Store struct {
	database storeDatabase
}

func mapUser(row queries.User) ports.User {
	return ports.User{
		ID:        row.ID,
		GitHubID:  row.GithubID.String,
		Email:     row.Email,
		Nickname:  row.Nickname,
		Name:      row.Name.String,
		AvatarURL: row.AvatarUrl.String,
	}
}

func nullString(value string) sql.NullString {
	value = strings.TrimSpace(value)
	if value == "" {
		return sql.NullString{}
	}
	return sql.NullString{String: value, Valid: true}
}

func nullInt64(value int64) sql.NullInt64 {
	if value <= 0 {
		return sql.NullInt64{}
	}
	return sql.NullInt64{Int64: value, Valid: true}
}

// NewStore constructs a sqlite adapter around the existing database implementation.
func NewStore(database storeDatabase) *Store {
	return &Store{database: database}
}

var _ ports.AppStore = (*Store)(nil)

// GetDefaultOrganization loads the default organization.
func (s *Store) GetDefaultOrganization(ctx context.Context) (ports.Organization, error) {
	org, err := s.database.GetDefaultOrganization(ctx)
	if err != nil {
		return ports.Organization{}, err
	}
	return mapOrganization(org), nil
}

// GetOrganizationByID returns organization by id.
func (s *Store) GetOrganizationByID(ctx context.Context, id int64) (ports.Organization, error) {
	org, err := s.database.GetOrganizationByID(ctx, id)
	if err != nil {
		return ports.Organization{}, err
	}
	return mapOrganization(org), nil
}

// GetOrganizationByJoinCode returns organization by join code.
func (s *Store) GetOrganizationByJoinCode(ctx context.Context, joinCode string) (ports.Organization, error) {
	org, err := s.database.GetOrganizationByJoinCode(ctx, nullString(joinCode))
	if err != nil {
		return ports.Organization{}, err
	}
	return mapOrganization(org), nil
}

// ListOrganizations returns all organizations.
func (s *Store) ListOrganizations(ctx context.Context) ([]ports.Organization, error) {
	rows, err := s.database.ListOrganizations(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]ports.Organization, 0, len(rows))
	for _, row := range rows {
		out = append(out, mapOrganization(row))
	}
	return out, nil
}

// CreateOrganization creates a new organization.
func (s *Store) CreateOrganization(ctx context.Context, params ports.CreateOrganizationInput) (ports.Organization, error) {
	enabled := int64(0)
	if params.Enabled {
		enabled = 1
	}

	org, err := s.database.CreateOrganization(ctx, queries.CreateOrganizationParams{
		Name:          strings.TrimSpace(params.Name),
		AuthToken:     strings.TrimSpace(params.AuthToken),
		JoinCode:      nullString(params.JoinCode),
		WebhookSecret: strings.TrimSpace(params.WebhookSecret),
		Enabled:       enabled,
	})
	if err != nil {
		return ports.Organization{}, err
	}
	return mapOrganization(org), nil
}

// UpdateOrganizationName updates one organization name.
func (s *Store) UpdateOrganizationName(ctx context.Context, organizationID int64, name string) error {
	name = strings.TrimSpace(name)
	if organizationID <= 0 || name == "" {
		return nil
	}
	return s.database.UpdateOrganizationName(ctx, organizationID, name)
}

// UpdateOrganizationEnabled updates one organization enabled state.
func (s *Store) UpdateOrganizationEnabled(ctx context.Context, organizationID int64, enabled bool) error {
	if organizationID <= 0 {
		return nil
	}
	return s.database.UpdateOrganizationEnabled(ctx, organizationID, enabled)
}

// DeleteOrganization removes one organization and cascading tenant data.
func (s *Store) DeleteOrganization(ctx context.Context, organizationID int64) error {
	if organizationID <= 0 {
		return nil
	}
	return s.database.DeleteOrganization(ctx, organizationID)
}

// UpsertUser inserts/updates local user identity.
func (s *Store) UpsertUser(ctx context.Context, input ports.UpsertUserInput) (ports.User, error) {
	row, err := s.database.UpsertUser(ctx, queries.UpsertUserParams{
		GithubID:  nullString(input.GitHubID),
		Email:     strings.TrimSpace(input.Email),
		Nickname:  strings.TrimSpace(input.Nickname),
		Name:      nullString(input.Name),
		AvatarUrl: nullString(input.AvatarURL),
	})
	if err != nil {
		return ports.User{}, err
	}
	return mapUser(row), nil
}

// GetUserByID returns one user by id.
func (s *Store) GetUserByID(ctx context.Context, id int64) (ports.User, error) {
	row, err := s.database.GetUserByID(ctx, id)
	if err != nil {
		return ports.User{}, err
	}
	return mapUser(row), nil
}

// GetUserByEmailOrNickname returns one user by email or nickname.
func (s *Store) GetUserByEmailOrNickname(ctx context.Context, email, nickname string) (ports.User, error) {
	row, err := s.database.GetUserByEmailOrNickname(ctx, strings.TrimSpace(email), strings.TrimSpace(nickname))
	if err != nil {
		return ports.User{}, err
	}
	return mapUser(row), nil
}

// ListOrganizationsByUser lists organizations for a user.
func (s *Store) ListOrganizationsByUser(ctx context.Context, userID int64) ([]ports.Organization, error) {
	rows, err := s.database.ListOrganizationsByUser(ctx, userID)
	if err != nil {
		return nil, err
	}
	out := make([]ports.Organization, 0, len(rows))
	for _, row := range rows {
		out = append(out, mapOrganization(row))
	}
	return out, nil
}

// GetOrganizationMemberRole returns one member role.
func (s *Store) GetOrganizationMemberRole(ctx context.Context, organizationID, userID int64) (string, error) {
	return s.database.GetOrganizationMemberRole(ctx, organizationID, userID)
}

// UpsertOrganizationMember upserts member role.
func (s *Store) UpsertOrganizationMember(ctx context.Context, organizationID, userID int64, role string) error {
	return s.database.UpsertOrganizationMember(ctx, organizationID, userID, strings.TrimSpace(role))
}

// DeleteOrganizationMember removes one member.
func (s *Store) DeleteOrganizationMember(ctx context.Context, organizationID, userID int64) error {
	return s.database.DeleteOrganizationMember(ctx, organizationID, userID)
}

// CountOrganizationOwners returns owner count.
func (s *Store) CountOrganizationOwners(ctx context.Context, organizationID int64) (int64, error) {
	return s.database.CountOrganizationOwners(ctx, organizationID)
}

// ListOrganizationMembers lists members with profile data.
func (s *Store) ListOrganizationMembers(ctx context.Context, organizationID int64) ([]ports.OrganizationMember, error) {
	rows, err := s.database.ListOrganizationMembers(ctx, organizationID)
	if err != nil {
		return nil, err
	}
	out := make([]ports.OrganizationMember, 0, len(rows))
	for _, row := range rows {
		out = append(out, ports.OrganizationMember{
			UserID:    row.UserID,
			Email:     row.Email,
			Nickname:  row.Nickname,
			Name:      row.Name.String,
			AvatarURL: row.AvatarUrl.String,
			Role:      row.Role,
		})
	}
	return out, nil
}

// UpsertGitHubInstallationMapping upserts GitHub installation mapping in DDash.
func (s *Store) UpsertGitHubInstallationMapping(ctx context.Context, mapping ports.GitHubInstallationMapping) error {
	enabled := int64(0)
	if mapping.Enabled {
		enabled = 1
	}
	return s.database.UpsertGitHubInstallationMapping(ctx, queries.UpsertGitHubInstallationMappingParams{
		InstallationID:     mapping.InstallationID,
		OrganizationID:     mapping.OrganizationID,
		OrganizationLabel:  strings.TrimSpace(mapping.OrganizationLabel),
		DefaultEnvironment: strings.TrimSpace(mapping.DefaultEnvironment),
		Enabled:            enabled,
	})
}

// ListGitHubInstallationMappings lists mappings for one organization.
func (s *Store) ListGitHubInstallationMappings(ctx context.Context, organizationID int64) ([]ports.GitHubInstallationMapping, error) {
	rows, err := s.database.ListGitHubInstallationMappings(ctx, organizationID)
	if err != nil {
		return nil, err
	}
	out := make([]ports.GitHubInstallationMapping, 0, len(rows))
	for _, row := range rows {
		out = append(out, ports.GitHubInstallationMapping{
			InstallationID:     row.InstallationID,
			OrganizationID:     row.OrganizationID,
			OrganizationLabel:  row.OrganizationLabel,
			DefaultEnvironment: row.DefaultEnvironment,
			Enabled:            row.Enabled != 0,
		})
	}
	return out, nil
}

// DeleteGitHubInstallationMapping deletes one mapping for one organization.
func (s *Store) DeleteGitHubInstallationMapping(ctx context.Context, installationID, organizationID int64) error {
	deleted, err := s.database.DeleteGitHubInstallationMapping(ctx, queries.DeleteGitHubInstallationMappingParams{
		InstallationID: installationID,
		OrganizationID: organizationID,
	})
	if err != nil {
		return err
	}
	if deleted == 0 {
		return sql.ErrNoRows
	}
	return nil
}

// GetOrganizationByGitHubInstallationID resolves organization from installation ID.
func (s *Store) GetOrganizationByGitHubInstallationID(ctx context.Context, installationID int64) (ports.Organization, error) {
	row, err := s.database.GetOrganizationByGitHubInstallationID(ctx, installationID)
	if err != nil {
		return ports.Organization{}, err
	}
	return ports.Organization{
		ID:            row.ID,
		Name:          row.Name,
		AuthToken:     row.AuthToken,
		JoinCode:      row.JoinCode.String,
		WebhookSecret: row.WebhookSecret,
		Enabled:       row.Enabled != 0,
	}, nil
}

// CreateGitHubSetupIntent stores GitHub setup state.
func (s *Store) CreateGitHubSetupIntent(ctx context.Context, intent ports.GitHubSetupIntent) error {
	return s.database.CreateGitHubSetupIntent(ctx, queries.CreateGitHubSetupIntentParams{
		State:              strings.TrimSpace(intent.State),
		OrganizationID:     intent.OrganizationID,
		OrganizationLabel:  strings.TrimSpace(intent.OrganizationLabel),
		DefaultEnvironment: strings.TrimSpace(intent.DefaultEnvironment),
		ExpiresAt:          intent.ExpiresAt.UTC(),
	})
}

// GetGitHubSetupIntentByState resolves setup state.
func (s *Store) GetGitHubSetupIntentByState(ctx context.Context, state string) (ports.GitHubSetupIntent, error) {
	row, err := s.database.GetGitHubSetupIntentByState(ctx, strings.TrimSpace(state))
	if err != nil {
		return ports.GitHubSetupIntent{}, err
	}
	return ports.GitHubSetupIntent{
		State:              row.State,
		OrganizationID:     row.OrganizationID,
		OrganizationLabel:  row.OrganizationLabel,
		DefaultEnvironment: row.DefaultEnvironment,
		ExpiresAt:          row.ExpiresAt,
	}, nil
}

// DeleteGitHubSetupIntent removes setup state.
func (s *Store) DeleteGitHubSetupIntent(ctx context.Context, state string) error {
	return s.database.DeleteGitHubSetupIntent(ctx, strings.TrimSpace(state))
}

// UpsertOrganizationJoinRequest creates or refreshes a pending join request.
func (s *Store) UpsertOrganizationJoinRequest(ctx context.Context, organizationID, userID int64, requestCode string) error {
	if organizationID <= 0 || userID <= 0 {
		return nil
	}
	return s.database.UpsertOrganizationJoinRequest(ctx, queries.UpsertOrganizationJoinRequestParams{
		OrganizationID: organizationID,
		UserID:         userID,
		RequestCode:    strings.TrimSpace(requestCode),
	})
}

// ListPendingOrganizationJoinRequests returns pending join requests.
func (s *Store) ListPendingOrganizationJoinRequests(ctx context.Context, organizationID int64) ([]ports.OrganizationJoinRequest, error) {
	rows, err := s.database.ListPendingOrganizationJoinRequests(ctx, organizationID)
	if err != nil {
		return nil, err
	}
	out := make([]ports.OrganizationJoinRequest, 0, len(rows))
	for _, row := range rows {
		name := ""
		if row.Name.Valid {
			name = row.Name.String
		}
		out = append(out, ports.OrganizationJoinRequest{
			OrganizationID: row.OrganizationID,
			UserID:         row.UserID,
			RequestCode:    row.RequestCode,
			Status:         row.Status,
			Email:          row.Email,
			Nickname:       row.Nickname,
			Name:           name,
		})
	}
	return out, nil
}

// SetOrganizationJoinRequestStatus updates join request status.
func (s *Store) SetOrganizationJoinRequestStatus(ctx context.Context, organizationID, userID int64, status string, reviewedBy int64) error {
	if organizationID <= 0 || userID <= 0 || reviewedBy <= 0 {
		return nil
	}
	return s.database.SetOrganizationJoinRequestStatus(ctx, queries.SetOrganizationJoinRequestStatusParams{
		Status:         strings.TrimSpace(status),
		ReviewedBy:     nullInt64(reviewedBy),
		OrganizationID: organizationID,
		UserID:         userID,
	})
}

// ListOrganizationRequiredFields returns configured required metadata fields.
func (s *Store) ListOrganizationRequiredFields(ctx context.Context, organizationID int64) ([]ports.RequiredField, error) {
	rows, err := s.database.ListOrganizationRequiredFields(ctx, organizationID)
	if err != nil {
		return nil, err
	}
	out := make([]ports.RequiredField, 0, len(rows))
	for _, row := range rows {
		out = append(out, ports.RequiredField{
			Label:      row.Label,
			Type:       row.FieldType,
			Filterable: row.IsFilterable != 0,
		})
	}
	return out, nil
}

// ListOrganizationEnvironmentPriorities returns ordered environment names.
func (s *Store) ListOrganizationEnvironmentPriorities(ctx context.Context, organizationID int64) ([]string, error) {
	rows, err := s.database.ListOrganizationEnvironmentPriorities(ctx, organizationID)
	if err != nil {
		return nil, err
	}
	out := make([]string, 0, len(rows))
	for _, row := range rows {
		value := strings.TrimSpace(row.Environment)
		if value != "" {
			out = append(out, value)
		}
	}
	return out, nil
}

// ListOrganizationFeatures returns feature flags for one organization.
func (s *Store) ListOrganizationFeatures(ctx context.Context, organizationID int64) ([]ports.OrganizationFeature, error) {
	rows, err := s.database.ListOrganizationFeatures(ctx, organizationID)
	if err != nil {
		return nil, err
	}
	out := make([]ports.OrganizationFeature, 0, len(rows))
	for _, row := range rows {
		out = append(out, ports.OrganizationFeature{Key: row.FeatureKey, Enabled: row.IsEnabled != 0})
	}
	return out, nil
}

// ListOrganizationPreferences returns preference key-values for one organization.
func (s *Store) ListOrganizationPreferences(ctx context.Context, organizationID int64) ([]ports.OrganizationPreference, error) {
	rows, err := s.database.ListOrganizationPreferences(ctx, organizationID)
	if err != nil {
		return nil, err
	}
	out := make([]ports.OrganizationPreference, 0, len(rows))
	for _, row := range rows {
		out = append(out, ports.OrganizationPreference{Key: row.PreferenceKey, Value: row.PreferenceValue})
	}
	return out, nil
}

// ListDistinctServiceEnvironmentsFromEvents returns discovered environment names.
func (s *Store) ListDistinctServiceEnvironmentsFromEvents(ctx context.Context, organizationID int64) ([]string, error) {
	return s.database.ListDistinctServiceEnvironmentsFromEvents(ctx, organizationID)
}

// UpdateOrganizationSettings persists organization secrets and metadata settings.
func (s *Store) UpdateOrganizationSettings(ctx context.Context, organizationID int64, params ports.OrganizationSettingsUpdate) error {
	params.AuthToken = strings.TrimSpace(params.AuthToken)
	params.WebhookSecret = strings.TrimSpace(params.WebhookSecret)

	return s.database.WithTx(ctx, func(q *queries.Queries) error {
		enabled := int64(0)
		if params.Enabled {
			enabled = 1
		}

		if err := q.UpdateOrganizationSecrets(ctx, queries.UpdateOrganizationSecretsParams{
			AuthToken:     params.AuthToken,
			WebhookSecret: params.WebhookSecret,
			Enabled:       enabled,
			ID:            organizationID,
		}); err != nil {
			return err
		}

		features := []struct {
			key     string
			enabled bool
		}{
			{featureShowSyncStatus, params.ShowSyncStatus},
			{featureShowMetadataBadges, params.ShowMetadataBadges},
			{featureShowEnvironmentColumn, params.ShowEnvironmentColumn},
			{featureEnableSSELiveUpdates, params.EnableSSELiveUpdates},
			{featureShowDeploymentHistory, params.ShowDeploymentHistory},
			{featureShowMetadataFilters, params.ShowMetadataFilters},
			{featureStrictMetadataEnforcement, params.StrictMetadataEnforcement},
			{featureMaskSensitiveMetadataValues, params.MaskSensitiveMetadataValues},
			{featureAllowServiceMetadataEditing, params.AllowServiceMetadataEditing},
			{featureShowOnboardingHints, params.ShowOnboardingHints},
			{featureShowIntegrationTypeBadges, params.ShowIntegrationTypeBadges},
			{featureShowServiceDetailInsights, params.ShowServiceDetailInsights},
			{featureShowServiceDependencies, params.ShowServiceDependencies},
		}
		for _, feature := range features {
			if err := q.UpsertOrganizationFeature(ctx, queries.UpsertOrganizationFeatureParams{
				OrganizationID: organizationID,
				FeatureKey:     feature.key,
				IsEnabled:      boolToInt64(feature.enabled),
			}); err != nil {
				return err
			}
		}

		preferences := []struct {
			key   string
			value string
		}{
			{prefDeploymentRetentionDays, strings.TrimSpace(strconv.Itoa(params.DeploymentRetentionDays))},
			{prefDefaultDashboardView, strings.TrimSpace(params.DefaultDashboardView)},
			{prefStatusSemanticsMode, strings.TrimSpace(params.StatusSemanticsMode)},
		}
		for _, preference := range preferences {
			if preference.value == "" {
				continue
			}
			if err := q.UpsertOrganizationPreference(ctx, queries.UpsertOrganizationPreferenceParams{
				OrganizationID:  organizationID,
				PreferenceKey:   preference.key,
				PreferenceValue: preference.value,
			}); err != nil {
				return err
			}
		}

		if err := q.DeleteOrganizationRequiredFields(ctx, organizationID); err != nil {
			return err
		}
		for index, field := range params.RequiredFields {
			label := strings.TrimSpace(field.Label)
			fieldType := strings.TrimSpace(field.Type)
			if label == "" || fieldType == "" {
				continue
			}
			filterable := int64(0)
			if field.Filterable {
				filterable = 1
			}
			if _, err := q.CreateOrganizationRequiredField(ctx, queries.CreateOrganizationRequiredFieldParams{
				OrganizationID: organizationID,
				Label:          label,
				FieldType:      fieldType,
				SortOrder:      int64(index),
				IsFilterable:   filterable,
			}); err != nil {
				return err
			}
		}

		if err := q.DeleteOrganizationEnvironmentPriorities(ctx, organizationID); err != nil {
			if !isMissingEnvPriorityTableErr(err) {
				return err
			}
			return nil
		}

		for index, environment := range params.EnvironmentOrder {
			value := strings.TrimSpace(environment)
			if value == "" {
				continue
			}
			if _, err := q.CreateOrganizationEnvironmentPriority(ctx, queries.CreateOrganizationEnvironmentPriorityParams{
				OrganizationID: organizationID,
				Environment:    value,
				SortOrder:      int64(index),
			}); err != nil {
				if isMissingEnvPriorityTableErr(err) {
					return nil
				}
				return err
			}
		}

		return nil
	})
}

func boolToInt64(value bool) int64 {
	if value {
		return 1
	}
	return 0
}

// ReplaceServiceMetadata replaces all metadata values for one service.
func (s *Store) ReplaceServiceMetadata(ctx context.Context, organizationID int64, serviceName string, values []ports.MetadataValue) error {
	serviceName = strings.TrimSpace(serviceName)
	if serviceName == "" {
		return nil
	}

	return s.database.WithTx(ctx, func(q *queries.Queries) error {
		if err := q.DeleteServiceMetadataByService(ctx, queries.DeleteServiceMetadataByServiceParams{
			OrganizationID: organizationID,
			ServiceName:    serviceName,
		}); err != nil {
			return err
		}

		for _, value := range values {
			label := strings.TrimSpace(value.Label)
			fieldValue := strings.TrimSpace(value.Value)
			if label == "" || fieldValue == "" {
				continue
			}
			if err := q.UpsertServiceMetadata(ctx, queries.UpsertServiceMetadataParams{
				OrganizationID: organizationID,
				ServiceName:    serviceName,
				Label:          label,
				Value:          fieldValue,
			}); err != nil {
				return err
			}
		}
		return nil
	})
}
