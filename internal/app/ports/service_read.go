package ports

import (
	"context"

	"github.com/fr0stylo/ddash/internal/app/domain"
)

// Organization is an app-level organization model.
type Organization struct {
	ID            int64
	Name          string
	AuthToken     string
	JoinCode      string
	WebhookSecret string
	Enabled       bool
}

// RequiredField is an app-level required metadata field definition.
type RequiredField struct {
	Label      string
	Type       string
	Filterable bool
}

// MetadataValue is one metadata label/value pair.
type MetadataValue struct {
	Label string
	Value string
}

// ServiceMetadataValue is metadata value associated with a service.
type ServiceMetadataValue struct {
	ServiceName string
	Label       string
	Value       string
}

// ServiceLatest contains latest service identity and integration info.
type ServiceLatest struct {
	Name            string
	IntegrationType string
}

// ServiceReadStore is a backend-agnostic read model store.
type ServiceReadStore interface {
	ListServiceInstances(ctx context.Context, organizationID int64, env string) ([]domain.Service, error)
	ListDeployments(ctx context.Context, organizationID int64, env, service string) ([]domain.DeploymentRow, error)
	GetServiceLatest(ctx context.Context, organizationID int64, name string) (ServiceLatest, error)
	ListServiceEnvironments(ctx context.Context, organizationID int64, service string) ([]domain.ServiceEnvironment, error)
	ListDeploymentHistory(ctx context.Context, organizationID int64, service string, limit int64) ([]domain.DeploymentRecord, error)
	ListRequiredFields(ctx context.Context, organizationID int64) ([]RequiredField, error)
	ListServiceMetadata(ctx context.Context, organizationID int64, service string) ([]MetadataValue, error)
	ListServiceMetadataValuesByOrganization(ctx context.Context, organizationID int64) ([]ServiceMetadataValue, error)
	ListEnvironmentPriorities(ctx context.Context, organizationID int64) ([]string, error)
	ListDiscoveredEnvironments(ctx context.Context, organizationID int64) ([]string, error)
}
