package sqlite

import (
	"context"
	"database/sql"

	"github.com/fr0stylo/ddash/internal/db/queries"
)

type storeDatabase interface {
	GetDefaultOrganization(ctx context.Context) (queries.Organization, error)
	GetOrganizationByID(ctx context.Context, id int64) (queries.Organization, error)
	GetOrganizationByJoinCode(ctx context.Context, joinCode sql.NullString) (queries.Organization, error)
	ListOrganizations(ctx context.Context) ([]queries.Organization, error)
	UpdateOrganizationName(ctx context.Context, organizationID int64, name string) error
	UpdateOrganizationEnabled(ctx context.Context, organizationID int64, enabled bool) error
	DeleteOrganization(ctx context.Context, organizationID int64) error
	CreateOrganization(ctx context.Context, params queries.CreateOrganizationParams) (queries.Organization, error)
	UpsertUser(ctx context.Context, params queries.UpsertUserParams) (queries.User, error)
	GetUserByID(ctx context.Context, id int64) (queries.User, error)
	GetUserByEmailOrNickname(ctx context.Context, email, nickname string) (queries.User, error)
	ListOrganizationsByUser(ctx context.Context, userID int64) ([]queries.Organization, error)
	GetOrganizationMemberRole(ctx context.Context, organizationID, userID int64) (string, error)
	UpsertOrganizationMember(ctx context.Context, organizationID, userID int64, role string) error
	DeleteOrganizationMember(ctx context.Context, organizationID, userID int64) error
	CountOrganizationOwners(ctx context.Context, organizationID int64) (int64, error)
	ListOrganizationMembers(ctx context.Context, organizationID int64) ([]queries.ListOrganizationMembersRow, error)
	UpsertGitHubInstallationMapping(ctx context.Context, params queries.UpsertGitHubInstallationMappingParams) error
	ListGitHubInstallationMappings(ctx context.Context, organizationID int64) ([]queries.ListGitHubInstallationMappingsRow, error)
	DeleteGitHubInstallationMapping(ctx context.Context, params queries.DeleteGitHubInstallationMappingParams) (int64, error)
	GetOrganizationByGitHubInstallationID(ctx context.Context, installationID int64) (queries.Organization, error)
	CreateGitHubSetupIntent(ctx context.Context, params queries.CreateGitHubSetupIntentParams) error
	GetGitHubSetupIntentByState(ctx context.Context, state string) (queries.GetGitHubSetupIntentByStateRow, error)
	DeleteGitHubSetupIntent(ctx context.Context, state string) error
	UpsertGitLabProjectMapping(ctx context.Context, params queries.UpsertGitLabProjectMappingParams) error
	GetOrganizationByGitLabProjectID(ctx context.Context, projectID int64) (queries.Organization, error)
	UpsertOrganizationJoinRequest(ctx context.Context, params queries.UpsertOrganizationJoinRequestParams) error
	ListPendingOrganizationJoinRequests(ctx context.Context, organizationID int64) ([]queries.ListPendingOrganizationJoinRequestsRow, error)
	SetOrganizationJoinRequestStatus(ctx context.Context, params queries.SetOrganizationJoinRequestStatusParams) error
	ListOrganizationRequiredFields(ctx context.Context, organizationID int64) ([]queries.ListOrganizationRequiredFieldsRow, error)
	ListOrganizationEnvironmentPriorities(ctx context.Context, organizationID int64) ([]queries.ListOrganizationEnvironmentPrioritiesRow, error)
	ListOrganizationFeatures(ctx context.Context, organizationID int64) ([]queries.ListOrganizationFeaturesRow, error)
	UpsertOrganizationFeature(ctx context.Context, organizationID int64, featureKey string, isEnabled bool) error
	ListOrganizationPreferences(ctx context.Context, organizationID int64) ([]queries.ListOrganizationPreferencesRow, error)
	UpsertOrganizationPreference(ctx context.Context, organizationID int64, preferenceKey, preferenceValue string) error
	ListDistinctServiceEnvironmentsFromEvents(ctx context.Context, organizationID int64) ([]string, error)

	ListServiceInstancesFromEvents(ctx context.Context, organizationID int64) ([]queries.ListServiceInstancesFromEventsRow, error)
	ListServiceInstancesByEnvFromEvents(ctx context.Context, params queries.ListServiceInstancesByEnvFromEventsParams) ([]queries.ListServiceInstancesByEnvFromEventsRow, error)
	ListDeploymentsFromEvents(ctx context.Context, params queries.ListDeploymentsFromEventsParams) ([]queries.ListDeploymentsFromEventsRow, error)
	GetOrganizationRenderVersion(ctx context.Context, orgID int64) (interface{}, error)
	GetServiceLatestFromEvents(ctx context.Context, params queries.GetServiceLatestFromEventsParams) (queries.GetServiceLatestFromEventsRow, error)
	ListServiceEnvironmentsFromEvents(ctx context.Context, params queries.ListServiceEnvironmentsFromEventsParams) ([]queries.ListServiceEnvironmentsFromEventsRow, error)
	ListDeploymentHistoryByServiceFromEvents(ctx context.Context, params queries.ListDeploymentHistoryByServiceFromEventsParams) ([]queries.ListDeploymentHistoryByServiceFromEventsRow, error)
	ListServiceDependencies(ctx context.Context, params queries.ListServiceDependenciesParams) ([]string, error)
	ListServiceDependants(ctx context.Context, params queries.ListServiceDependantsParams) ([]string, error)
	UpsertServiceDependency(ctx context.Context, params queries.UpsertServiceDependencyParams) error
	DeleteServiceDependency(ctx context.Context, params queries.DeleteServiceDependencyParams) error
	GetServiceCurrentState(ctx context.Context, params queries.GetServiceCurrentStateParams) (queries.GetServiceCurrentStateRow, error)
	GetServiceDeliveryStats30d(ctx context.Context, params queries.GetServiceDeliveryStats30dParams) (queries.GetServiceDeliveryStats30dRow, error)
	ListServiceChangeLinksRecent(ctx context.Context, params queries.ListServiceChangeLinksRecentParams) ([]queries.ListServiceChangeLinksRecentRow, error)
	ListServiceLeadTimeSamplesFromEvents(ctx context.Context, params queries.ListServiceLeadTimeSamplesFromEventsParams) ([]queries.ListServiceLeadTimeSamplesFromEventsRow, error)

	GetPipelineStats30d(ctx context.Context, params queries.GetPipelineStats30dParams) (queries.GetPipelineStats30dRow, error)
	GetDeploymentDurationStats(ctx context.Context, params queries.GetDeploymentDurationStatsParams) (queries.GetDeploymentDurationStatsRow, error)
	GetEnvironmentDriftCount(ctx context.Context, params queries.GetEnvironmentDriftCountParams) (int64, error)
	ListEnvironmentDrifts(ctx context.Context, params queries.ListEnvironmentDriftsParams) ([]queries.ListEnvironmentDriftsRow, error)
	GetRedeploymentRate30d(ctx context.Context, params queries.GetRedeploymentRate30dParams) (queries.GetRedeploymentRate30dRow, error)
	GetThroughputStats(ctx context.Context, params queries.GetThroughputStatsParams) (queries.GetThroughputStatsRow, error)
	ListWeeklyThroughput(ctx context.Context, params queries.ListWeeklyThroughputParams) ([]queries.ListWeeklyThroughputRow, error)
	GetArtifactAgeByEnvironment(ctx context.Context, params queries.GetArtifactAgeByEnvironmentParams) ([]queries.GetArtifactAgeByEnvironmentRow, error)
	GetMTTR(ctx context.Context, params queries.GetMTTRParams) (queries.GetMTTRRow, error)
	ListIncidentLinks(ctx context.Context, params queries.ListIncidentLinksParams) ([]queries.ListIncidentLinksRow, error)
	GetComprehensiveDeliveryMetrics(ctx context.Context, params queries.GetComprehensiveDeliveryMetricsParams) (queries.GetComprehensiveDeliveryMetricsRow, error)

	ListServiceMetadataByService(ctx context.Context, params queries.ListServiceMetadataByServiceParams) ([]queries.ListServiceMetadataByServiceRow, error)
	ListServiceMetadataByOrganization(ctx context.Context, organizationID int64) ([]queries.ListServiceMetadataByOrganizationRow, error)

	WithTx(ctx context.Context, fn func(*queries.Queries) error) error
}
