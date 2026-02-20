package domain

// ServiceStatus is a read-model status for service views.
type ServiceStatus string

const (
	// ServiceStatusSynced indicates service is synchronized.
	ServiceStatusSynced ServiceStatus = "synced"
	// ServiceStatusProgressing indicates service is progressing.
	ServiceStatusProgressing ServiceStatus = "progressing"
	// ServiceStatusOutOfSync indicates service drift.
	ServiceStatusOutOfSync ServiceStatus = "out-of-sync"
	// ServiceStatusWarning indicates warning state.
	ServiceStatusWarning ServiceStatus = "warning"
	// ServiceStatusUnknown indicates unknown state.
	ServiceStatusUnknown ServiceStatus = "unknown"
	// ServiceStatusAll is synthetic filter status.
	ServiceStatusAll ServiceStatus = "all"
)

// DeploymentStatus is a read-model status for deployment views.
type DeploymentStatus string

const (
	// DeploymentStatusQueued indicates queued deployment.
	DeploymentStatusQueued DeploymentStatus = "queued"
	// DeploymentStatusProcessing indicates in-progress deployment.
	DeploymentStatusProcessing DeploymentStatus = "processing"
	// DeploymentStatusSuccess indicates successful deployment.
	DeploymentStatusSuccess DeploymentStatus = "success"
	// DeploymentStatusError indicates failed deployment.
	DeploymentStatusError DeploymentStatus = "error"
)

// MetadataFilterOption is one metadata filter dropdown option.
type MetadataFilterOption struct {
	Value string
	Label string
}

// MetadataField represents metadata key/value configuration or value.
type MetadataField struct {
	Label      string
	Value      string
	Filterable bool
}

// Service is one service row/card projection.
type Service struct {
	Title           string
	Environment     string
	Status          ServiceStatus
	LastDeploy      string
	Revision        string
	CommitSHA       string
	DeployDuration  string
	MissingMetadata int
	MetadataTags    string
}

// DeploymentRow is one deployment projection row.
type DeploymentRow struct {
	Service      string
	Environment  string
	DeployedAt   string
	Status       DeploymentStatus
	MetadataTags string
}

// ServiceEnvironment is one environment row in service details.
type ServiceEnvironment struct {
	Name            string
	LastDeploy      string
	LastDeployedAgo string
	Ref             string
	DeployCount7d   int
	DeployCount30d  int
	DailyRate30d    string
}

// DeploymentRecord is one deployment history item.
type DeploymentRecord struct {
	Ref         string
	PreviousRef string
	ChangeLog   string
	Commits     int
	DeployedAt  string
	Environment string
	DeployedAgo string
}

// ServiceDetail is composed service details page view model.
type ServiceDetail struct {
	Title             string
	Description       string
	IntegrationType   string
	MissingMetadata   int
	MetadataSaveURL   string
	MetadataFields    []MetadataField
	OrgRequiredFields []MetadataField
	Environments      []ServiceEnvironment
	DeploymentHistory []DeploymentRecord
}
