package db

import (
	"context"
	"database/sql"
	"errors"
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

// UpsertGitHubInstallationMapping upserts installation mapping for an organization.
func (c *Database) UpsertGitHubInstallationMapping(ctx context.Context, params queries.UpsertGitHubInstallationMappingParams) error {
	return c.Queries.UpsertGitHubInstallationMapping(ctx, params)
}

// ListGitHubInstallationMappings lists GitHub installation mappings for an organization.
func (c *Database) ListGitHubInstallationMappings(ctx context.Context, organizationID int64) ([]queries.ListGitHubInstallationMappingsRow, error) {
	return c.Queries.ListGitHubInstallationMappings(ctx, organizationID)
}

// DeleteGitHubInstallationMapping deletes one installation mapping for an organization.
func (c *Database) DeleteGitHubInstallationMapping(ctx context.Context, params queries.DeleteGitHubInstallationMappingParams) (int64, error) {
	return c.Queries.DeleteGitHubInstallationMapping(ctx, params)
}

// GetOrganizationByGitHubInstallationID resolves organization for GitHub installation.
func (c *Database) GetOrganizationByGitHubInstallationID(ctx context.Context, installationID int64) (queries.Organization, error) {
	return c.Queries.GetOrganizationByGitHubInstallationID(ctx, installationID)
}

// CreateGitHubSetupIntent stores setup intent state.
func (c *Database) CreateGitHubSetupIntent(ctx context.Context, params queries.CreateGitHubSetupIntentParams) error {
	return c.Queries.CreateGitHubSetupIntent(ctx, params)
}

// GetGitHubSetupIntentByState resolves setup intent by state.
func (c *Database) GetGitHubSetupIntentByState(ctx context.Context, state string) (queries.GetGitHubSetupIntentByStateRow, error) {
	return c.Queries.GetGitHubSetupIntentByState(ctx, state)
}

// DeleteGitHubSetupIntent removes setup intent state.
func (c *Database) DeleteGitHubSetupIntent(ctx context.Context, state string) error {
	return c.Queries.DeleteGitHubSetupIntent(ctx, state)
}

// UpsertGitLabProjectMapping upserts GitLab project mapping for an organization.
func (c *Database) UpsertGitLabProjectMapping(ctx context.Context, params queries.UpsertGitLabProjectMappingParams) error {
	return c.Queries.UpsertGitLabProjectMapping(ctx, params)
}

// GetOrganizationByGitLabProjectID resolves organization for GitLab project id.
func (c *Database) GetOrganizationByGitLabProjectID(ctx context.Context, projectID int64) (queries.Organization, error) {
	return c.Queries.GetOrganizationByGitLabProjectID(ctx, projectID)
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

// GetOrganizationRenderVersion returns coarse UI render version for one organization.
func (c *Database) GetOrganizationRenderVersion(ctx context.Context, orgID int64) (interface{}, error) {
	return c.Queries.GetOrganizationRenderVersion(ctx, orgID)
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

// ListServiceDependencies returns dependencies for one service.
func (c *Database) ListServiceDependencies(ctx context.Context, params queries.ListServiceDependenciesParams) ([]string, error) {
	return c.Queries.ListServiceDependencies(ctx, params)
}

// ListServiceDependants returns dependants for one service.
func (c *Database) ListServiceDependants(ctx context.Context, params queries.ListServiceDependantsParams) ([]string, error) {
	return c.Queries.ListServiceDependants(ctx, params)
}

// UpsertServiceDependency creates one dependency edge.
func (c *Database) UpsertServiceDependency(ctx context.Context, params queries.UpsertServiceDependencyParams) error {
	return c.Queries.UpsertServiceDependency(ctx, params)
}

// DeleteServiceDependency removes one dependency edge.
func (c *Database) DeleteServiceDependency(ctx context.Context, params queries.DeleteServiceDependencyParams) error {
	return c.Queries.DeleteServiceDependency(ctx, params)
}

// GetServiceCurrentState returns current projected status metrics for one service.
func (c *Database) GetServiceCurrentState(ctx context.Context, params queries.GetServiceCurrentStateParams) (queries.GetServiceCurrentStateRow, error) {
	return c.Queries.GetServiceCurrentState(ctx, params)
}

// GetServiceDeliveryStats30d returns 30d delivery counters for one service.
func (c *Database) GetServiceDeliveryStats30d(ctx context.Context, params queries.GetServiceDeliveryStats30dParams) (queries.GetServiceDeliveryStats30dRow, error) {
	return c.Queries.GetServiceDeliveryStats30d(ctx, params)
}

// ListServiceChangeLinksRecent returns recent change link rows for one service.
func (c *Database) ListServiceChangeLinksRecent(ctx context.Context, params queries.ListServiceChangeLinksRecentParams) ([]queries.ListServiceChangeLinksRecentRow, error) {
	return c.Queries.ListServiceChangeLinksRecent(ctx, params)
}

// ListServiceLeadTimeSamplesFromEvents returns lead-time samples derived from change->deploy ordering.
func (c *Database) ListServiceLeadTimeSamplesFromEvents(ctx context.Context, params queries.ListServiceLeadTimeSamplesFromEventsParams) ([]queries.ListServiceLeadTimeSamplesFromEventsRow, error) {
	return c.Queries.ListServiceLeadTimeSamplesFromEvents(ctx, params)
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
	return c.WithTx(ctx, func(q *queries.Queries) error {
		_, err := appendEventWithProjections(ctx, q, params)
		return err
	})
}

// AppendEventStoreBatch stores multiple CDEvents in one transaction.
func (c *Database) AppendEventStoreBatch(ctx context.Context, params []queries.AppendEventStoreParams) error {
	if len(params) == 0 {
		return nil
	}
	return c.WithTx(ctx, func(q *queries.Queries) error {
		for _, item := range params {
			if _, err := appendEventWithProjections(ctx, q, item); err != nil {
				return err
			}
		}
		return nil
	})
}

func appendEventWithProjections(ctx context.Context, q *queries.Queries, params queries.AppendEventStoreParams) (bool, error) {
	seq, err := q.AppendEventStore(ctx, params)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return false, nil
		}
		return false, err
	}
	if strings.TrimSpace(strings.ToLower(params.SubjectType)) != "service" {
		return true, nil
	}

	projectionParams := queries.UpsertServiceEnvStateFromEventSeqParams{
		OrganizationID: params.OrganizationID,
		Seq:            seq,
	}
	if err := q.UpsertServiceEnvStateFromEventSeq(ctx, projectionParams); err != nil {
		return false, err
	}
	if err := q.UpsertServiceDeliveryStatsDailyFromEventSeq(ctx, queries.UpsertServiceDeliveryStatsDailyFromEventSeqParams{
		OrganizationID: params.OrganizationID,
		Seq:            seq,
	}); err != nil {
		return false, err
	}
	if err := q.UpsertServiceChangeLinkFromEventSeq(ctx, queries.UpsertServiceChangeLinkFromEventSeqParams{
		OrganizationID: params.OrganizationID,
		Seq:            seq,
	}); err != nil {
		return false, err
	}

	serviceName := serviceNameFromSubjectID(params.SubjectID)
	if serviceName == "" {
		return true, nil
	}
	if err := q.UpsertServiceCurrentStateByService(ctx, queries.UpsertServiceCurrentStateByServiceParams{
		OrganizationID: params.OrganizationID,
		ServiceName:    serviceName,
	}); err != nil {
		return false, err
	}
	return true, nil
}

func serviceNameFromSubjectID(subjectID string) string {
	subjectID = strings.TrimSpace(subjectID)
	if subjectID == "" {
		return ""
	}
	if idx := strings.LastIndex(subjectID, "/"); idx >= 0 && idx+1 < len(subjectID) {
		return strings.TrimSpace(subjectID[idx+1:])
	}
	return subjectID
}

// ProjectionRebuildStats reports row counts after rebuild.
type ProjectionRebuildStats struct {
	CurrentStateRows int64
	EnvStateRows     int64
	DailyStatsRows   int64
	ChangeLinkRows   int64
}

// RebuildServiceProjections rebuilds projection tables from event_store.
func (c *Database) RebuildServiceProjections(ctx context.Context, organizationID int64) (ProjectionRebuildStats, error) {
	stats := ProjectionRebuildStats{}
	if err := c.execProjectionRebuild(ctx, organizationID); err != nil {
		return ProjectionRebuildStats{}, err
	}

	var err error
	stats.CurrentStateRows, err = c.countProjectionRows(ctx, "service_current_state", organizationID)
	if err != nil {
		return ProjectionRebuildStats{}, err
	}
	stats.EnvStateRows, err = c.countProjectionRows(ctx, "service_env_state", organizationID)
	if err != nil {
		return ProjectionRebuildStats{}, err
	}
	stats.DailyStatsRows, err = c.countProjectionRows(ctx, "service_delivery_stats_daily", organizationID)
	if err != nil {
		return ProjectionRebuildStats{}, err
	}
	stats.ChangeLinkRows, err = c.countProjectionRows(ctx, "service_change_links", organizationID)
	if err != nil {
		return ProjectionRebuildStats{}, err
	}

	return stats, nil
}

func (c *Database) execProjectionRebuild(ctx context.Context, _ int64) error {
	statements := []string{
		"DELETE FROM service_current_state",
		"DELETE FROM service_env_state",
		"DELETE FROM service_delivery_stats_daily",
		"DELETE FROM service_change_links",
		`INSERT INTO service_env_state (
			organization_id, service_name, environment,
			latest_event_seq, latest_event_type, latest_event_ts_ms,
			latest_status, latest_artifact_id
		)
		WITH ranked AS (
			SELECT
				es.organization_id,
				CASE WHEN instr(es.subject_id, '/') > 0 THEN substr(es.subject_id, instr(es.subject_id, '/') + 1) ELSE es.subject_id END AS service_name,
				COALESCE(NULLIF(json_extract(es.raw_event_json, '$.subject.content.environment.id'), ''), 'unknown') AS environment,
				es.seq,
				es.event_type,
				es.event_ts_ms,
				COALESCE(json_extract(es.raw_event_json, '$.subject.content.artifactId'), '') AS artifact_id,
				CASE
					WHEN es.event_type LIKE 'dev.cdevents.service.deployed.%' THEN 'synced'
					WHEN es.event_type LIKE 'dev.cdevents.service.upgraded.%' THEN 'synced'
					WHEN es.event_type LIKE 'dev.cdevents.service.published.%' THEN 'synced'
					WHEN es.event_type LIKE 'dev.cdevents.service.rolledback.%' THEN 'warning'
					WHEN es.event_type LIKE 'dev.cdevents.service.removed.%' THEN 'out-of-sync'
					ELSE 'unknown'
				END AS status,
				row_number() OVER (
					PARTITION BY es.organization_id,
					CASE WHEN instr(es.subject_id, '/') > 0 THEN substr(es.subject_id, instr(es.subject_id, '/') + 1) ELSE es.subject_id END,
					COALESCE(NULLIF(json_extract(es.raw_event_json, '$.subject.content.environment.id'), ''), 'unknown')
					ORDER BY es.event_ts_ms DESC, es.seq DESC
				) AS rn
			FROM event_store es
			WHERE es.subject_type = 'service'
		)
		SELECT
			organization_id,
			service_name,
			environment,
			seq,
			event_type,
			event_ts_ms,
			status,
			artifact_id
		FROM ranked
		WHERE rn = 1`,
		`INSERT INTO service_current_state (
			organization_id, service_name,
			latest_event_seq, latest_event_type, latest_event_ts_ms,
			latest_status, latest_artifact_id, latest_environment,
			drift_count, failed_streak
		)
		WITH latest AS (
			SELECT
				ses.organization_id,
				ses.service_name,
				ses.latest_event_seq,
				ses.latest_event_type,
				ses.latest_event_ts_ms,
				ses.latest_status,
				ses.latest_artifact_id,
				ses.environment,
				row_number() OVER (PARTITION BY ses.organization_id, ses.service_name ORDER BY ses.latest_event_ts_ms DESC, ses.latest_event_seq DESC) AS rn
			FROM service_env_state ses
		), drift AS (
			SELECT
				organization_id,
				service_name,
				COUNT(DISTINCT NULLIF(latest_artifact_id, '')) AS drift_count,
				SUM(CASE WHEN latest_status IN ('warning', 'out-of-sync') THEN 1 ELSE 0 END) AS failed_streak
			FROM service_env_state
			GROUP BY organization_id, service_name
		)
		SELECT
			l.organization_id,
			l.service_name,
			l.latest_event_seq,
			l.latest_event_type,
			l.latest_event_ts_ms,
			l.latest_status,
			l.latest_artifact_id,
			l.environment,
			COALESCE(d.drift_count, 0),
			COALESCE(d.failed_streak, 0)
		FROM latest l
		LEFT JOIN drift d ON d.organization_id = l.organization_id AND d.service_name = l.service_name
		WHERE l.rn = 1`,
		`INSERT INTO service_delivery_stats_daily (
			organization_id, service_name, day_utc,
			deploy_success_count, deploy_failure_count, rollback_count
		)
		SELECT
			es.organization_id,
			CASE WHEN instr(es.subject_id, '/') > 0 THEN substr(es.subject_id, instr(es.subject_id, '/') + 1) ELSE es.subject_id END AS service_name,
			date(datetime(es.event_ts_ms / 1000, 'unixepoch')) AS day_utc,
			SUM(CASE WHEN es.event_type LIKE 'dev.cdevents.service.deployed.%' OR es.event_type LIKE 'dev.cdevents.service.upgraded.%' OR es.event_type LIKE 'dev.cdevents.service.published.%' THEN 1 ELSE 0 END),
			SUM(CASE WHEN es.event_type LIKE 'dev.cdevents.service.removed.%' THEN 1 ELSE 0 END),
			SUM(CASE WHEN es.event_type LIKE 'dev.cdevents.service.rolledback.%' THEN 1 ELSE 0 END)
		FROM event_store es
		WHERE es.subject_type = 'service'
		GROUP BY es.organization_id, service_name, day_utc`,
		`INSERT INTO service_change_links (
			organization_id, service_name, event_seq, event_ts_ms,
			chain_id, environment, artifact_id, pipeline_run_id,
			run_url, actor_name
		)
		SELECT
			es.organization_id,
			CASE WHEN instr(es.subject_id, '/') > 0 THEN substr(es.subject_id, instr(es.subject_id, '/') + 1) ELSE es.subject_id END AS service_name,
			es.seq,
			es.event_ts_ms,
			es.chain_id,
			COALESCE(NULLIF(json_extract(es.raw_event_json, '$.subject.content.environment.id'), ''), 'unknown') AS environment,
			COALESCE(json_extract(es.raw_event_json, '$.subject.content.artifactId'), '') AS artifact_id,
			COALESCE(json_extract(es.raw_event_json, '$.subject.content.pipeline.runId'), '') AS pipeline_run_id,
			COALESCE(json_extract(es.raw_event_json, '$.subject.content.pipeline.url'), '') AS run_url,
			COALESCE(json_extract(es.raw_event_json, '$.subject.content.actor.name'), '') AS actor_name
		FROM event_store es
		WHERE es.subject_type = 'service'`,
	}

	for _, statement := range statements {
		if _, err := c.db.ExecContext(ctx, statement); err != nil {
			return err
		}
	}
	return nil
}

func (c *Database) countProjectionRows(ctx context.Context, table string, organizationID int64) (int64, error) {
	query := "SELECT COUNT(*) FROM " + table
	args := []interface{}{}
	if organizationID > 0 {
		query += " WHERE organization_id = ?"
		args = append(args, organizationID)
	}
	row := c.db.QueryRowContext(ctx, query, args...)
	var count int64
	err := row.Scan(&count)
	return count, err
}

// WithTx runs a function within a transaction.
func (c *Database) WithTx(ctx context.Context, fn func(*queries.Queries) error) error {
	tx, err := c.db.BeginTx(ctx, &sql.TxOptions{})
	if err != nil {
		return err
	}
	q := queries.New(newInstrumentedDBTX(tx, c.tracker))
	if err := fn(q); err != nil {
		if rollbackErr := tx.Rollback(); rollbackErr != nil {
			return rollbackErr
		}
		return err
	}
	return tx.Commit()
}
