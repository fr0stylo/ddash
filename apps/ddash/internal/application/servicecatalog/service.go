package servicecatalog

import (
	"context"
	"math"
	"sort"
	"time"

	"github.com/fr0stylo/ddash/apps/ddash/internal/app/domain"
	"github.com/fr0stylo/ddash/apps/ddash/internal/app/ports"
	appservices "github.com/fr0stylo/ddash/apps/ddash/internal/app/services"
	domaincatalog "github.com/fr0stylo/ddash/apps/ddash/internal/domains/servicecatalog"
)

type Service struct {
	read  *appservices.ServiceReadService
	store ports.ServiceReadStore
}

type LeadTimeSummary struct {
	Samples    int   `json:"samples"`
	AvgSeconds int64 `json:"avg_seconds"`
	P50Seconds int64 `json:"p50_seconds"`
	P95Seconds int64 `json:"p95_seconds"`
}

type LeadTimeByService struct {
	Service string          `json:"service"`
	Stats   LeadTimeSummary `json:"stats"`
}

type LeadTimeByDay struct {
	Day   string          `json:"day"`
	Stats LeadTimeSummary `json:"stats"`
}

type LeadTimeReport struct {
	Overall   LeadTimeSummary     `json:"overall"`
	ByService []LeadTimeByService `json:"by_service"`
	ByDay     []LeadTimeByDay     `json:"by_day"`
}

type PipelineStats struct {
	PipelineStartedCount   int64   `json:"pipeline_started_count"`
	PipelineSucceededCount int64   `json:"pipeline_succeeded_count"`
	PipelineFailedCount    int64   `json:"pipeline_failed_count"`
	TotalDurationSeconds   int64   `json:"total_duration_seconds"`
	AvgDurationSeconds     float64 `json:"avg_duration_seconds"`
}

type DeploymentDurationStats struct {
	SampleCount         int64   `json:"sample_count"`
	AvgDurationSeconds  float64 `json:"avg_duration_seconds"`
	MinDurationSeconds  int64   `json:"min_duration_seconds"`
	MaxDurationSeconds  int64   `json:"max_duration_seconds"`
	LastDurationSeconds int64   `json:"last_duration_seconds"`
}

type EnvironmentDrift struct {
	EnvironmentFrom string `json:"environment_from"`
	EnvironmentTo   string `json:"environment_to"`
	ArtifactIDFrom  string `json:"artifact_id_from"`
	ArtifactIDTo    string `json:"artifact_id_to"`
	DriftDetectedAt int64  `json:"drift_detected_at"`
}

type RedeploymentRate struct {
	RedeployCount int64   `json:"redeploy_count"`
	DeployDays    int64   `json:"deploy_days"`
	RedeployRate  float64 `json:"redeploy_rate"`
}

type WeeklyThroughput struct {
	WeekStart        string `json:"week_start"`
	ChangesCount     int64  `json:"changes_count"`
	DeploymentsCount int64  `json:"deployments_count"`
}

type ArtifactAge struct {
	Environment   string `json:"environment"`
	ArtifactID    string `json:"artifact_id"`
	AgeSeconds    int64  `json:"age_seconds"`
	LastEventTsMs int64  `json:"last_event_ts_ms"`
}

type MTTRStats struct {
	IncidentCount int64   `json:"incident_count"`
	MTTRSeconds   float64 `json:"mttr_seconds"`
	MTTDSeconds   float64 `json:"mttd_seconds"`
	MTTESeconds   float64 `json:"mtte_seconds"`
}

type IncidentLink struct {
	IncidentID         string `json:"incident_id"`
	IncidentType       string `json:"incident_type"`
	LinkedAt           int64  `json:"linked_at"`
	DeploymentEventSeq int64  `json:"deployment_event_seq"`
}

type ComprehensiveDeliveryMetrics struct {
	LeadTimeSeconds              float64 `json:"lead_time_seconds"`
	DeploymentFrequency30d       int64   `json:"deployment_frequency_30d"`
	ChangeFailureRate            float64 `json:"change_failure_rate"`
	AvgDeploymentDurationSeconds float64 `json:"avg_deployment_duration_seconds"`
	PipelineSuccessCount30d      int64   `json:"pipeline_success_count_30d"`
	PipelineFailureCount30d      int64   `json:"pipeline_failure_count_30d"`
	ActiveDeployDays30d          int64   `json:"active_deploy_days_30d"`
}

func NewService(store ports.ServiceReadStore) *Service {
	return &Service{read: appservices.NewServiceReadServiceFromStore(store), store: store}
}

func (s *Service) GetHomeData(ctx context.Context, organizationID int64) ([]domain.Service, []domain.MetadataFilterOption, error) {
	return s.read.GetHomeData(ctx, organizationID)
}

func (s *Service) GetServicesByEnv(ctx context.Context, organizationID int64, env string) ([]domain.Service, error) {
	return s.read.GetServicesByEnv(ctx, organizationID, env)
}

func (s *Service) GetDeployments(ctx context.Context, organizationID int64, env, service string) ([]domain.DeploymentRow, []domain.MetadataFilterOption, error) {
	return s.read.GetDeployments(ctx, organizationID, env, service)
}

func (s *Service) GetOrganizationRenderVersion(ctx context.Context, organizationID int64) (int64, error) {
	return s.read.GetOrganizationRenderVersion(ctx, organizationID)
}

func (s *Service) GetServiceDetail(ctx context.Context, organizationID int64, name string) (domain.ServiceDetail, error) {
	return s.read.GetServiceDetail(ctx, organizationID, name)
}

func (s *Service) UpsertServiceDependency(ctx context.Context, organizationID int64, serviceName, dependsOn string) error {
	serviceName, dependsOn, ok := domaincatalog.NormalizeDependencyInput(serviceName, dependsOn)
	if !ok {
		return nil
	}
	return s.read.UpsertServiceDependency(ctx, organizationID, serviceName, dependsOn)
}

func (s *Service) UpsertServiceDependencies(ctx context.Context, organizationID int64, serviceName, rawDependsOn string) (int, error) {
	added := 0
	for _, dependsOn := range domaincatalog.ParseDependencyInputs(serviceName, rawDependsOn) {
		if err := s.read.UpsertServiceDependency(ctx, organizationID, serviceName, dependsOn); err != nil {
			return added, err
		}
		added++
	}
	return added, nil
}

func (s *Service) DeleteServiceDependency(ctx context.Context, organizationID int64, serviceName, dependsOn string) error {
	serviceName, dependsOn, ok := domaincatalog.NormalizeDependencyInput(serviceName, dependsOn)
	if !ok {
		return nil
	}
	return s.read.DeleteServiceDependency(ctx, organizationID, serviceName, dependsOn)
}

func (s *Service) BuildDependencyGraph(ctx context.Context, organizationID int64) (domaincatalog.DependencyGraph, error) {
	services, err := s.store.ListServiceInstances(ctx, organizationID, "")
	if err != nil {
		return domaincatalog.DependencyGraph{}, err
	}

	nodeMap := map[string]domaincatalog.GraphNode{}
	edges := make([]domaincatalog.GraphEdge, 0)
	seenEdges := map[string]bool{}

	for _, item := range services {
		name := item.Title
		if name == "" {
			continue
		}
		nodeMap[name] = domaincatalog.GraphNode{ID: name, Name: name}
		deps, depErr := s.store.ListServiceDependencies(ctx, organizationID, name)
		if depErr != nil {
			return domaincatalog.DependencyGraph{}, depErr
		}
		for _, dep := range deps {
			if dep == "" {
				continue
			}
			nodeMap[dep] = domaincatalog.GraphNode{ID: dep, Name: dep}
			key := name + "->" + dep
			if seenEdges[key] {
				continue
			}
			seenEdges[key] = true
			edges = append(edges, domaincatalog.GraphEdge{From: name, To: dep})
		}
	}

	nodes := make([]domaincatalog.GraphNode, 0, len(nodeMap))
	for _, node := range nodeMap {
		nodes = append(nodes, node)
	}
	sort.Slice(nodes, func(i, j int) bool { return nodes[i].Name < nodes[j].Name })
	sort.Slice(edges, func(i, j int) bool {
		if edges[i].From == edges[j].From {
			return edges[i].To < edges[j].To
		}
		return edges[i].From < edges[j].From
	})

	return domaincatalog.DependencyGraph{Nodes: nodes, Edges: edges}, nil
}

func (s *Service) BuildLeadTimeReport(ctx context.Context, organizationID int64, days int) (LeadTimeReport, error) {
	if days <= 0 {
		days = 30
	}
	if days > 365 {
		days = 365
	}
	sinceMs := time.Now().UTC().Add(-time.Duration(days) * 24 * time.Hour).UnixMilli()
	samples, err := s.store.ListServiceLeadTimeSamples(ctx, organizationID, sinceMs)
	if err != nil {
		return LeadTimeReport{}, err
	}

	overallVals := make([]int64, 0, len(samples))
	byService := map[string][]int64{}
	byDay := map[string][]int64{}
	for _, sample := range samples {
		overallVals = append(overallVals, sample.LeadSeconds)
		if sample.ServiceName != "" {
			byService[sample.ServiceName] = append(byService[sample.ServiceName], sample.LeadSeconds)
		}
		if sample.DayUTC != "" {
			byDay[sample.DayUTC] = append(byDay[sample.DayUTC], sample.LeadSeconds)
		}
	}

	serviceRows := make([]LeadTimeByService, 0, len(byService))
	for service, values := range byService {
		serviceRows = append(serviceRows, LeadTimeByService{Service: service, Stats: summarizeLeadTimes(values)})
	}
	sort.Slice(serviceRows, func(i, j int) bool { return serviceRows[i].Service < serviceRows[j].Service })

	dayRows := make([]LeadTimeByDay, 0, len(byDay))
	for day, values := range byDay {
		dayRows = append(dayRows, LeadTimeByDay{Day: day, Stats: summarizeLeadTimes(values)})
	}
	sort.Slice(dayRows, func(i, j int) bool { return dayRows[i].Day < dayRows[j].Day })

	return LeadTimeReport{
		Overall:   summarizeLeadTimes(overallVals),
		ByService: serviceRows,
		ByDay:     dayRows,
	}, nil
}

func summarizeLeadTimes(values []int64) LeadTimeSummary {
	if len(values) == 0 {
		return LeadTimeSummary{}
	}
	ordered := append([]int64(nil), values...)
	sort.Slice(ordered, func(i, j int) bool { return ordered[i] < ordered[j] })
	var sum int64
	for _, value := range ordered {
		sum += value
	}
	return LeadTimeSummary{
		Samples:    len(ordered),
		AvgSeconds: sum / int64(len(ordered)),
		P50Seconds: percentile(ordered, 0.50),
		P95Seconds: percentile(ordered, 0.95),
	}
}

func percentile(sorted []int64, p float64) int64 {
	if len(sorted) == 0 {
		return 0
	}
	if p <= 0 {
		return sorted[0]
	}
	if p >= 1 {
		return sorted[len(sorted)-1]
	}
	idx := int(math.Ceil(p*float64(len(sorted)))) - 1
	if idx < 0 {
		idx = 0
	}
	if idx >= len(sorted) {
		idx = len(sorted) - 1
	}
	return sorted[idx]
}

type ServiceMetricsResponse struct {
	ServiceName         string                             `json:"service_name"`
	PipelineStats       ports.PipelineStats                `json:"pipeline_stats"`
	DeploymentDurations ports.DeploymentDurationStats      `json:"deployment_durations"`
	DriftCount          int64                              `json:"drift_count"`
	RedeploymentRate    ports.RedeploymentRate             `json:"redeployment_rate"`
	Throughput          ports.WeeklyThroughput             `json:"throughput"`
	ArtifactAges        []ports.ArtifactAge                `json:"artifact_ages"`
	Comprehensive       ports.ComprehensiveDeliveryMetrics `json:"comprehensive"`
}

func (s *Service) GetServiceMetrics(ctx context.Context, organizationID int64, serviceName string, days int) (ServiceMetricsResponse, error) {
	if days <= 0 {
		days = 30
	}
	if days > 365 {
		days = 365
	}
	sinceMs := time.Now().UTC().Add(-time.Duration(days) * 24 * time.Hour).UnixMilli()

	pipelineStats, err := s.store.GetPipelineStats30d(ctx, organizationID, serviceName)
	if err != nil {
		return ServiceMetricsResponse{}, err
	}

	durationStats, err := s.store.GetDeploymentDurationStats(ctx, organizationID, serviceName, "", sinceMs)
	if err != nil {
		return ServiceMetricsResponse{}, err
	}

	driftCount, err := s.store.GetEnvironmentDriftCount(ctx, organizationID, serviceName, sinceMs)
	if err != nil {
		return ServiceMetricsResponse{}, err
	}

	redeployRate, err := s.store.GetRedeploymentRate30d(ctx, organizationID, serviceName)
	if err != nil {
		return ServiceMetricsResponse{}, err
	}

	throughput, err := s.store.GetThroughputStats(ctx, organizationID, serviceName)
	if err != nil {
		return ServiceMetricsResponse{}, err
	}

	artifactAges, err := s.store.GetArtifactAgeByEnvironment(ctx, organizationID, serviceName)
	if err != nil {
		return ServiceMetricsResponse{}, err
	}

	comprehensive, err := s.store.GetComprehensiveDeliveryMetrics(ctx, organizationID, sinceMs)
	if err != nil {
		return ServiceMetricsResponse{}, err
	}

	return ServiceMetricsResponse{
		ServiceName:         serviceName,
		PipelineStats:       pipelineStats,
		DeploymentDurations: durationStats,
		DriftCount:          driftCount,
		RedeploymentRate:    redeployRate,
		Throughput:          throughput,
		ArtifactAges:        artifactAges,
		Comprehensive:       comprehensive,
	}, nil
}

type OrgMetricsResponse struct {
	TotalServices       int64                              `json:"total_services"`
	PipelineStats       ports.PipelineStats                `json:"pipeline_stats"`
	Comprehensive       ports.ComprehensiveDeliveryMetrics `json:"comprehensive"`
	LeadTimeReport      LeadTimeReport                     `json:"lead_time"`
	WeeklyThroughput    []ports.WeeklyThroughput           `json:"weekly_throughput"`
	TopRedeployServices []ports.RedeploymentRate           `json:"top_redeploy_services"`
}

func (s *Service) GetOrgMetrics(ctx context.Context, organizationID int64, days int) (OrgMetricsResponse, error) {
	if days <= 0 {
		days = 30
	}
	if days > 365 {
		days = 365
	}
	sinceMs := time.Now().UTC().Add(-time.Duration(days) * 24 * time.Hour).UnixMilli()

	services, err := s.store.ListServiceInstances(ctx, organizationID, "")
	if err != nil {
		return OrgMetricsResponse{}, err
	}

	serviceNames := make(map[string]bool)
	for _, svc := range services {
		if svc.Title != "" {
			serviceNames[svc.Title] = true
		}
	}

	var totalPipelineStats ports.PipelineStats
	var comprehensive ports.ComprehensiveDeliveryMetrics

	for name := range serviceNames {
		ps, err := s.store.GetPipelineStats30d(ctx, organizationID, name)
		if err != nil {
			continue
		}
		totalPipelineStats.PipelineStartedCount += ps.PipelineStartedCount
		totalPipelineStats.PipelineSucceededCount += ps.PipelineSucceededCount
		totalPipelineStats.PipelineFailedCount += ps.PipelineFailedCount
		totalPipelineStats.TotalDurationSeconds += ps.TotalDurationSeconds
	}

	comp, err := s.store.GetComprehensiveDeliveryMetrics(ctx, organizationID, sinceMs)
	if err == nil {
		comprehensive = comp
	}

	leadTimeReport, err := s.BuildLeadTimeReport(ctx, organizationID, days)
	if err != nil {
		leadTimeReport = LeadTimeReport{}
	}

	return OrgMetricsResponse{
		TotalServices:    int64(len(serviceNames)),
		PipelineStats:    totalPipelineStats,
		Comprehensive:    comprehensive,
		LeadTimeReport:   leadTimeReport,
		WeeklyThroughput: []ports.WeeklyThroughput{},
	}, nil
}
