package ports

import (
	"context"
)

// AppStore defines storage operations used by route/application layer.
// It is intentionally backend-agnostic: sqlc-backed DB implements this today,
// but gRPC/clickhouse adapters can implement it later.
type AppStore interface {
	GetDefaultOrganization(ctx context.Context) (Organization, error)
	GetOrganizationByID(ctx context.Context, id int64) (Organization, error)
	GetOrganizationByJoinCode(ctx context.Context, joinCode string) (Organization, error)
	ListOrganizations(ctx context.Context) ([]Organization, error)
	CreateOrganization(ctx context.Context, params CreateOrganizationInput) (Organization, error)
	UpdateOrganizationName(ctx context.Context, organizationID int64, name string) error
	UpdateOrganizationEnabled(ctx context.Context, organizationID int64, enabled bool) error
	DeleteOrganization(ctx context.Context, organizationID int64) error

	UpsertUser(ctx context.Context, input UpsertUserInput) (User, error)
	GetUserByID(ctx context.Context, id int64) (User, error)
	GetUserByEmailOrNickname(ctx context.Context, email, nickname string) (User, error)
	ListOrganizationsByUser(ctx context.Context, userID int64) ([]Organization, error)
	GetOrganizationMemberRole(ctx context.Context, organizationID, userID int64) (string, error)
	UpsertOrganizationMember(ctx context.Context, organizationID, userID int64, role string) error
	DeleteOrganizationMember(ctx context.Context, organizationID, userID int64) error
	CountOrganizationOwners(ctx context.Context, organizationID int64) (int64, error)
	ListOrganizationMembers(ctx context.Context, organizationID int64) ([]OrganizationMember, error)
	UpsertOrganizationJoinRequest(ctx context.Context, organizationID, userID int64, requestCode string) error
	ListPendingOrganizationJoinRequests(ctx context.Context, organizationID int64) ([]OrganizationJoinRequest, error)
	SetOrganizationJoinRequestStatus(ctx context.Context, organizationID, userID int64, status string, reviewedBy int64) error

	ListOrganizationRequiredFields(ctx context.Context, organizationID int64) ([]RequiredField, error)
	ListOrganizationEnvironmentPriorities(ctx context.Context, organizationID int64) ([]string, error)
	ListOrganizationFeatures(ctx context.Context, organizationID int64) ([]OrganizationFeature, error)
	ListOrganizationPreferences(ctx context.Context, organizationID int64) ([]OrganizationPreference, error)
	ListDistinctServiceEnvironmentsFromEvents(ctx context.Context, organizationID int64) ([]string, error)

	UpdateOrganizationSettings(ctx context.Context, organizationID int64, params OrganizationSettingsUpdate) error
	ReplaceServiceMetadata(ctx context.Context, organizationID int64, serviceName string, values []MetadataValue) error
}

// CreateOrganizationInput represents organization creation fields.
type CreateOrganizationInput struct {
	Name          string
	AuthToken     string
	JoinCode      string
	WebhookSecret string
	Enabled       bool
}

// OrganizationJoinRequest is one pending/processed join request for an organization.
type OrganizationJoinRequest struct {
	OrganizationID int64
	UserID         int64
	RequestCode    string
	Status         string
	Email          string
	Nickname       string
	Name           string
}

// OrganizationSettingsUpdate contains values persisted from settings page.
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
	DeploymentRetentionDays     int
	DefaultDashboardView        string
	StatusSemanticsMode         string
	RequiredFields              []RequiredField
	EnvironmentOrder            []string
}

// OrganizationFeature is one organization feature flag.
type OrganizationFeature struct {
	Key     string
	Enabled bool
}

// OrganizationPreference is one organization preference key-value setting.
type OrganizationPreference struct {
	Key   string
	Value string
}

// UpsertUserInput contains user identity fields captured during auth callback.
type UpsertUserInput struct {
	GitHubID  string
	Email     string
	Nickname  string
	Name      string
	AvatarURL string
}

// User is a local authenticated identity record.
type User struct {
	ID        int64
	GitHubID  string
	Email     string
	Nickname  string
	Name      string
	AvatarURL string
}

// OrganizationMember contains one organization member and role.
type OrganizationMember struct {
	UserID    int64
	Email     string
	Nickname  string
	Name      string
	AvatarURL string
	Role      string
}
