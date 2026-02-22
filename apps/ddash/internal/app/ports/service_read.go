package ports

import (
	"context"

	"github.com/fr0stylo/ddash/apps/ddash/internal/app/domain"
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

// ServiceCurrentState holds latest projected state for one service.
type ServiceCurrentState struct {
	LastStatus    string
	LastEventTSMs int64
	DriftCount    int
	FailedStreak  int
}

// ServiceDeliveryStats summarizes 30-day delivery outcomes.
type ServiceDeliveryStats struct {
	Success30d   int
	Failures30d  int
	Rollbacks30d int
}

// ServiceChangeLink is one recent chain/audit linkage row.
type ServiceChangeLink struct {
	EventTSMs     int64
	ChainID       string
	Environment   string
	ArtifactID    string
	PipelineRunID string
	RunURL        string
	ActorName     string
}

type ServiceLeadTimeSample struct {
	DayUTC      string
	ServiceName string
	LeadSeconds int64
}

// ServiceDependency represents one dependency relationship edge.
type ServiceDependency struct {
	ServiceName   string
	DependsOnName string
}

// ServiceQueryStore exposes non-analytical service/deployment reads.
type ServiceQueryStore interface {
	ListServiceInstances(ctx context.Context, organizationID int64, env string) ([]domain.Service, error)
	ListDeployments(ctx context.Context, organizationID int64, env, service string) ([]domain.DeploymentRow, error)
	GetServiceLatest(ctx context.Context, organizationID int64, name string) (ServiceLatest, error)
	ListServiceEnvironments(ctx context.Context, organizationID int64, service string) ([]domain.ServiceEnvironment, error)
	ListDeploymentHistory(ctx context.Context, organizationID int64, service string, limit int64) ([]domain.DeploymentRecord, error)
	ListServiceDependencies(ctx context.Context, organizationID int64, service string) ([]string, error)
	ListServiceDependants(ctx context.Context, organizationID int64, service string) ([]string, error)
	UpsertServiceDependency(ctx context.Context, organizationID int64, serviceName, dependsOnServiceName string) error
	DeleteServiceDependency(ctx context.Context, organizationID int64, serviceName, dependsOnServiceName string) error
	GetOrganizationRenderVersion(ctx context.Context, organizationID int64) (int64, error)
}

// ServiceMetadataStore exposes metadata/settings reads for service views.
type ServiceMetadataStore interface {
	ListRequiredFields(ctx context.Context, organizationID int64) ([]RequiredField, error)
	ListServiceMetadata(ctx context.Context, organizationID int64, service string) ([]MetadataValue, error)
	ListServiceMetadataValuesByOrganization(ctx context.Context, organizationID int64) ([]ServiceMetadataValue, error)
	ListEnvironmentPriorities(ctx context.Context, organizationID int64) ([]string, error)
	ListDiscoveredEnvironments(ctx context.Context, organizationID int64) ([]string, error)
}

// ServiceAnalyticsStore exposes analytical projection reads.
type ServiceAnalyticsStore interface {
	GetServiceCurrentState(ctx context.Context, organizationID int64, service string) (ServiceCurrentState, error)
	GetServiceDeliveryStats30d(ctx context.Context, organizationID int64, service string) (ServiceDeliveryStats, error)
	ListServiceChangeLinksRecent(ctx context.Context, organizationID int64, service string, limit int64) ([]ServiceChangeLink, error)
	ListServiceLeadTimeSamples(ctx context.Context, organizationID int64, sinceMs int64) ([]ServiceLeadTimeSample, error)
}

// ServiceReadStore is a convenience aggregate for callsites using one store.
type ServiceReadStore interface {
	ServiceQueryStore
	ServiceMetadataStore
	ServiceAnalyticsStore
}
