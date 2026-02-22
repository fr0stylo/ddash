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
