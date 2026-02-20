package db

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/fr0stylo/ddash/internal/db/queries"
)

// GetOrganizationByAuthToken fetches an org by auth token.
func (c *Database) GetOrganizationByAuthToken(ctx context.Context, authToken string) (queries.Organization, error) {
	return c.Queries.GetOrganizationByAuthToken(ctx, authToken)
}

// GetOrganizationByID fetches an org by id.
func (c *Database) GetOrganizationByID(ctx context.Context, id int64) (queries.Organization, error) {
	return c.Queries.GetOrganizationByID(ctx, id)
}

// ListOrganizations returns all organizations.
func (c *Database) ListOrganizations(ctx context.Context) ([]queries.Organization, error) {
	return c.Queries.ListOrganizations(ctx)
}

// UpdateOrganizationName updates organization name.
func (c *Database) UpdateOrganizationName(ctx context.Context, organizationID int64, name string) error {
	return c.Queries.UpdateOrganizationName(ctx, queries.UpdateOrganizationNameParams{Name: name, ID: organizationID})
}

// UpdateOrganizationEnabled updates organization enabled state.
func (c *Database) UpdateOrganizationEnabled(ctx context.Context, organizationID int64, enabled bool) error {
	enabledValue := int64(0)
	if enabled {
		enabledValue = 1
	}
	return c.Queries.UpdateOrganizationEnabled(ctx, queries.UpdateOrganizationEnabledParams{Enabled: enabledValue, ID: organizationID})
}

// DeleteOrganization removes an organization and cascading tenant data.
func (c *Database) DeleteOrganization(ctx context.Context, organizationID int64) error {
	return c.Queries.DeleteOrganization(ctx, organizationID)
}

// UpsertUser inserts/updates local user identity from OAuth profile.
func (c *Database) UpsertUser(ctx context.Context, params queries.UpsertUserParams) (queries.User, error) {
	return c.Queries.UpsertUser(ctx, params)
}

// GetUserByID fetches a user by id.
func (c *Database) GetUserByID(ctx context.Context, id int64) (queries.User, error) {
	return c.Queries.GetUserByID(ctx, id)
}

// GetUserByEmailOrNickname fetches a user by email/nickname.
func (c *Database) GetUserByEmailOrNickname(ctx context.Context, email, nickname string) (queries.User, error) {
	return c.Queries.GetUserByEmailOrNickname(ctx, queries.GetUserByEmailOrNicknameParams{Email: email, Nickname: nickname})
}

// ListOrganizationsByUser lists orgs where user is a member.
func (c *Database) ListOrganizationsByUser(ctx context.Context, userID int64) ([]queries.Organization, error) {
	return c.Queries.ListOrganizationsByUser(ctx, userID)
}

// GetOrganizationMemberRole returns one user's role in org.
func (c *Database) GetOrganizationMemberRole(ctx context.Context, organizationID, userID int64) (string, error) {
	return c.Queries.GetOrganizationMemberRole(ctx, queries.GetOrganizationMemberRoleParams{OrganizationID: organizationID, UserID: userID})
}

// UpsertOrganizationMember upserts org membership and role.
func (c *Database) UpsertOrganizationMember(ctx context.Context, organizationID, userID int64, role string) error {
	return c.Queries.UpsertOrganizationMember(ctx, queries.UpsertOrganizationMemberParams{OrganizationID: organizationID, UserID: userID, Role: role})
}

// DeleteOrganizationMember removes user from org membership.
func (c *Database) DeleteOrganizationMember(ctx context.Context, organizationID, userID int64) error {
	return c.Queries.DeleteOrganizationMember(ctx, queries.DeleteOrganizationMemberParams{OrganizationID: organizationID, UserID: userID})
}

// CountOrganizationOwners returns owner count in org.
func (c *Database) CountOrganizationOwners(ctx context.Context, organizationID int64) (int64, error) {
	return c.Queries.CountOrganizationOwners(ctx, organizationID)
}

// ListOrganizationMembers lists members joined with user profile.
func (c *Database) ListOrganizationMembers(ctx context.Context, organizationID int64) ([]queries.ListOrganizationMembersRow, error) {
	return c.Queries.ListOrganizationMembers(ctx, organizationID)
}

// GetDefaultOrganization returns the first organization.
func (c *Database) GetDefaultOrganization(ctx context.Context) (queries.Organization, error) {
	return c.Queries.GetDefaultOrganization(ctx)
}

// CreateOrganization inserts a new organization.
func (c *Database) CreateOrganization(ctx context.Context, params queries.CreateOrganizationParams) (queries.Organization, error) {
	return c.Queries.CreateOrganization(ctx, params)
}

// ListDeploymentsFromEventsParams configures event-stream deployment query.
type ListDeploymentsFromEventsParams = queries.ListDeploymentsFromEventsParams

// ListDeploymentHistoryByServiceFromEventsParams configures event-stream deployment history query.
type ListDeploymentHistoryByServiceFromEventsParams = queries.ListDeploymentHistoryByServiceFromEventsParams

// ListServiceInstancesFromEvents returns current service states from event stream.
func (c *Database) ListServiceInstancesFromEvents(ctx context.Context, organizationID int64) ([]queries.ListServiceInstancesFromEventsRow, error) {
	return c.Queries.ListServiceInstancesFromEvents(ctx, organizationID)
}

// ListServiceInstancesByEnvFromEvents returns current service states by environment from event stream.
func (c *Database) ListServiceInstancesByEnvFromEvents(ctx context.Context, params queries.ListServiceInstancesByEnvFromEventsParams) ([]queries.ListServiceInstancesByEnvFromEventsRow, error) {
	return c.Queries.ListServiceInstancesByEnvFromEvents(ctx, params)
}

// ListDeploymentsFromEvents returns deployment rows from event stream.
func (c *Database) ListDeploymentsFromEvents(ctx context.Context, params queries.ListDeploymentsFromEventsParams) ([]queries.ListDeploymentsFromEventsRow, error) {
	return c.Queries.ListDeploymentsFromEvents(ctx, params)
}

// GetServiceLatestFromEvents returns latest event for a service.
func (c *Database) GetServiceLatestFromEvents(ctx context.Context, params queries.GetServiceLatestFromEventsParams) (queries.GetServiceLatestFromEventsRow, error) {
	return c.Queries.GetServiceLatestFromEvents(ctx, params)
}

// ListServiceEnvironmentsFromEvents returns latest service state per environment from event stream.
func (c *Database) ListServiceEnvironmentsFromEvents(ctx context.Context, params queries.ListServiceEnvironmentsFromEventsParams) ([]queries.ListServiceEnvironmentsFromEventsRow, error) {
	return c.Queries.ListServiceEnvironmentsFromEvents(ctx, params)
}

// ListDeploymentHistoryByServiceFromEvents returns service deployment history from event stream.
func (c *Database) ListDeploymentHistoryByServiceFromEvents(ctx context.Context, params queries.ListDeploymentHistoryByServiceFromEventsParams) ([]queries.ListDeploymentHistoryByServiceFromEventsRow, error) {
	return c.Queries.ListDeploymentHistoryByServiceFromEvents(ctx, params)
}

// ListOrganizationRequiredFields returns required fields for an org.
func (c *Database) ListOrganizationRequiredFields(ctx context.Context, organizationID int64) ([]queries.ListOrganizationRequiredFieldsRow, error) {
	return c.Queries.ListOrganizationRequiredFields(ctx, organizationID)
}

// ListOrganizationEnvironmentPriorities returns environment ordering for an org.
func (c *Database) ListOrganizationEnvironmentPriorities(ctx context.Context, organizationID int64) ([]queries.ListOrganizationEnvironmentPrioritiesRow, error) {
	return c.Queries.ListOrganizationEnvironmentPriorities(ctx, organizationID)
}

// ListOrganizationFeatures returns feature flags for an org.
func (c *Database) ListOrganizationFeatures(ctx context.Context, organizationID int64) ([]queries.ListOrganizationFeaturesRow, error) {
	return c.Queries.ListOrganizationFeatures(ctx, organizationID)
}

// ListOrganizationPreferences returns preferences for an org.
func (c *Database) ListOrganizationPreferences(ctx context.Context, organizationID int64) ([]queries.ListOrganizationPreferencesRow, error) {
	return c.Queries.ListOrganizationPreferences(ctx, organizationID)
}

// UpsertOrganizationFeature upserts one feature flag for an org.
func (c *Database) UpsertOrganizationFeature(ctx context.Context, organizationID int64, featureKey string, isEnabled bool) error {
	enabled := int64(0)
	if isEnabled {
		enabled = 1
	}
	return c.Queries.UpsertOrganizationFeature(ctx, queries.UpsertOrganizationFeatureParams{
		OrganizationID: organizationID,
		FeatureKey:     featureKey,
		IsEnabled:      enabled,
	})
}

// UpsertOrganizationPreference upserts one preference value for an org.
func (c *Database) UpsertOrganizationPreference(ctx context.Context, organizationID int64, preferenceKey, preferenceValue string) error {
	return c.Queries.UpsertOrganizationPreference(ctx, queries.UpsertOrganizationPreferenceParams{
		OrganizationID:  organizationID,
		PreferenceKey:   preferenceKey,
		PreferenceValue: preferenceValue,
	})
}

// DeleteOrganizationEnvironmentPriorities removes environment priorities for an org.
func (c *Database) DeleteOrganizationEnvironmentPriorities(ctx context.Context, organizationID int64) error {
	return c.Queries.DeleteOrganizationEnvironmentPriorities(ctx, organizationID)
}

// CreateOrganizationEnvironmentPriority inserts one environment priority row.
func (c *Database) CreateOrganizationEnvironmentPriority(ctx context.Context, params queries.CreateOrganizationEnvironmentPriorityParams) (queries.OrganizationEnvironmentPriority, error) {
	return c.Queries.CreateOrganizationEnvironmentPriority(ctx, params)
}

// ListServiceMetadataByService returns metadata values for a service.
func (c *Database) ListServiceMetadataByService(ctx context.Context, params queries.ListServiceMetadataByServiceParams) ([]queries.ListServiceMetadataByServiceRow, error) {
	return c.Queries.ListServiceMetadataByService(ctx, params)
}

// ListServiceMetadataByOrganization returns metadata values for all services in an org.
func (c *Database) ListServiceMetadataByOrganization(ctx context.Context, organizationID int64) ([]queries.ListServiceMetadataByOrganizationRow, error) {
	return c.Queries.ListServiceMetadataByOrganization(ctx, organizationID)
}

// ListDistinctServiceEnvironmentsFromEvents returns discovered environments from service events.
func (c *Database) ListDistinctServiceEnvironmentsFromEvents(ctx context.Context, organizationID int64) ([]string, error) {
	rows, err := c.Queries.ListDistinctServiceEnvironmentsFromEvents(ctx, organizationID)
	if err != nil {
		return nil, err
	}
	result := make([]string, 0, len(rows))
	for _, row := range rows {
		switch value := row.(type) {
		case nil:
			continue
		case string:
			value = strings.TrimSpace(value)
			if value != "" {
				result = append(result, value)
			}
		case []byte:
			text := strings.TrimSpace(string(value))
			if text != "" {
				result = append(result, text)
			}
		default:
			text := strings.TrimSpace(fmt.Sprint(value))
			if text != "" {
				result = append(result, text)
			}
		}
	}
	return result, nil
}

// DeleteOrganizationRequiredFields removes required fields for an org.
func (c *Database) DeleteOrganizationRequiredFields(ctx context.Context, organizationID int64) error {
	return c.Queries.DeleteOrganizationRequiredFields(ctx, organizationID)
}

// CreateOrganizationRequiredField inserts a required field.
func (c *Database) CreateOrganizationRequiredField(ctx context.Context, params queries.CreateOrganizationRequiredFieldParams) (queries.OrganizationRequiredField, error) {
	return c.Queries.CreateOrganizationRequiredField(ctx, params)
}

// UpdateOrganizationSecrets updates auth token and webhook secret.
func (c *Database) UpdateOrganizationSecrets(ctx context.Context, authToken, webhookSecret string, enabled bool, organizationID int64) error {
	enabledValue := int64(0)
	if enabled {
		enabledValue = 1
	}
	return c.Queries.UpdateOrganizationSecrets(ctx, queries.UpdateOrganizationSecretsParams{
		AuthToken:     authToken,
		WebhookSecret: webhookSecret,
		Enabled:       enabledValue,
		ID:            organizationID,
	})
}

// AppendEventStore stores a CDEvent in the append-only event store.
func (c *Database) AppendEventStore(ctx context.Context, params queries.AppendEventStoreParams) error {
	return c.Queries.AppendEventStore(ctx, params)
}

// WithTx runs a function within a transaction.
func (c *Database) WithTx(ctx context.Context, fn func(*queries.Queries) error) error {
	tx, err := c.db.BeginTx(ctx, &sql.TxOptions{})
	if err != nil {
		return err
	}
	if err := fn(c.Queries.WithTx(tx)); err != nil {
		if rollbackErr := tx.Rollback(); rollbackErr != nil {
			return rollbackErr
		}
		return err
	}
	return tx.Commit()
}
