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

type PipelineStats struct {
	PipelineStartedCount   int64
	PipelineSucceededCount int64
	PipelineFailedCount    int64
	TotalDurationSeconds   int64
	AvgDurationSeconds     float64
}

type DeploymentDurationStats struct {
	SampleCount         int64
	AvgDurationSeconds  float64
	MinDurationSeconds  int64
	MaxDurationSeconds  int64
	LastDurationSeconds int64
}

type EnvironmentDrift struct {
	EnvironmentFrom string
	EnvironmentTo   string
	ArtifactIDFrom  string
	ArtifactIDTo    string
	DriftDetectedAt int64
}

type RedeploymentRate struct {
	RedeployCount int64
	DeployDays    int64
	RedeployRate  float64
}

type WeeklyThroughput struct {
	WeekStart        string
	ChangesCount     int64
	DeploymentsCount int64
}

type ArtifactAge struct {
	Environment   string
	ArtifactID    string
	AgeSeconds    int64
	LastEventTsMs int64
}

type MTTRStats struct {
	IncidentCount int64
	MTTRSeconds   float64
	MTTDSeconds   float64
	MTTESeconds   float64
}

type IncidentLink struct {
	IncidentID         string
	IncidentType       string
	LinkedAt           int64
	DeploymentEventSeq int64
}

type ComprehensiveDeliveryMetrics struct {
	LeadTimeSeconds              float64
	DeploymentFrequency30d       int64
	ChangeFailureRate            float64
	AvgDeploymentDurationSeconds float64
	PipelineSuccessCount30d      int64
	PipelineFailureCount30d      int64
	ActiveDeployDays30d          int64
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
	GetPipelineStats30d(ctx context.Context, organizationID int64, service string) (PipelineStats, error)
	GetDeploymentDurationStats(ctx context.Context, organizationID int64, service string, environment string, sinceMs int64) (DeploymentDurationStats, error)
	GetEnvironmentDriftCount(ctx context.Context, organizationID int64, service string, sinceMs int64) (int64, error)
	ListEnvironmentDrifts(ctx context.Context, organizationID int64, service string, limit int64) ([]EnvironmentDrift, error)
	GetRedeploymentRate30d(ctx context.Context, organizationID int64, service string) (RedeploymentRate, error)
	GetThroughputStats(ctx context.Context, organizationID int64, service string) (WeeklyThroughput, error)
	ListWeeklyThroughput(ctx context.Context, organizationID int64, service string, limit int64) ([]WeeklyThroughput, error)
	GetArtifactAgeByEnvironment(ctx context.Context, organizationID int64, service string) ([]ArtifactAge, error)
	GetMTTR(ctx context.Context, organizationID int64, sinceMs int64) (MTTRStats, error)
	ListIncidentLinks(ctx context.Context, organizationID int64, service string, limit int64) ([]IncidentLink, error)
	GetComprehensiveDeliveryMetrics(ctx context.Context, organizationID int64, sinceMs int64) (ComprehensiveDeliveryMetrics, error)
}

// ServiceReadStore is a convenience aggregate for callsites using one store.
type ServiceReadStore interface {
	ServiceQueryStore
	ServiceMetadataStore
	ServiceAnalyticsStore
}
