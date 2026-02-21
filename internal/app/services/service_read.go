package services

import (
	"context"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/fr0stylo/ddash/internal/app/domain"
	"github.com/fr0stylo/ddash/internal/app/ports"
)

// ServiceReadService provides read-side projections for service/deployment views.
type ServiceReadService struct {
	serviceStore   ports.ServiceQueryStore
	metadataStore  ports.ServiceMetadataStore
	analyticsStore ports.ServiceAnalyticsStore
}

// NewServiceReadService constructs a read-side service with separated dependencies.
func NewServiceReadService(serviceStore ports.ServiceQueryStore, metadataStore ports.ServiceMetadataStore, analyticsStore ports.ServiceAnalyticsStore) *ServiceReadService {
	return &ServiceReadService{
		serviceStore:   serviceStore,
		metadataStore:  metadataStore,
		analyticsStore: analyticsStore,
	}
}

// NewServiceReadServiceFromStore constructs from a single aggregate store.
func NewServiceReadServiceFromStore(store ports.ServiceReadStore) *ServiceReadService {
	return NewServiceReadService(store, store, store)
}

// GetHomeData returns service cards/table data and metadata filter options.
func (s *ServiceReadService) GetHomeData(ctx context.Context, organizationID int64) ([]domain.Service, []domain.MetadataFilterOption, error) {
	services, err := s.GetServicesByEnv(ctx, organizationID, "all")
	if err != nil {
		return nil, nil, err
	}
	metadata, err := s.loadMetadataFilterData(ctx, organizationID)
	if err != nil {
		return nil, nil, err
	}
	return services, metadata.Options, nil
}

// GetServicesByEnv returns service rows enriched with metadata badges/tags.
func (s *ServiceReadService) GetServicesByEnv(ctx context.Context, organizationID int64, env string) ([]domain.Service, error) {
	services, err := s.serviceStore.ListServiceInstances(ctx, organizationID, env)
	if err != nil {
		return nil, err
	}
	metadata, err := s.loadMetadataFilterData(ctx, organizationID)
	if err != nil {
		return nil, err
	}
	return applyMetadataToServices(services, metadata), nil
}

// GetDeployments returns deployment rows enriched with metadata tags.
func (s *ServiceReadService) GetDeployments(ctx context.Context, organizationID int64, env, service string) ([]domain.DeploymentRow, []domain.MetadataFilterOption, error) {
	rows, err := s.serviceStore.ListDeployments(ctx, organizationID, env, service)
	if err != nil {
		return nil, nil, err
	}
	metadata, err := s.loadMetadataFilterData(ctx, organizationID)
	if err != nil {
		return nil, nil, err
	}
	return applyMetadataToDeployments(rows, metadata), metadata.Options, nil
}

// GetOrganizationRenderVersion returns a lightweight version stamp for UI fragments.
func (s *ServiceReadService) GetOrganizationRenderVersion(ctx context.Context, organizationID int64) (int64, error) {
	return s.serviceStore.GetOrganizationRenderVersion(ctx, organizationID)
}

// GetServiceDetail returns a fully composed service details view model.
func (s *ServiceReadService) GetServiceDetail(ctx context.Context, organizationID int64, name string) (domain.ServiceDetail, error) {
	service, err := s.serviceStore.GetServiceLatest(ctx, organizationID, name)
	if err != nil {
		return domain.ServiceDetail{}, err
	}

	requiredFields, err := s.metadataStore.ListRequiredFields(ctx, organizationID)
	if err != nil {
		return domain.ServiceDetail{}, err
	}
	metadataRows, err := s.metadataStore.ListServiceMetadata(ctx, organizationID, name)
	if err != nil {
		return domain.ServiceDetail{}, err
	}
	serviceEnvs, err := s.serviceStore.ListServiceEnvironments(ctx, organizationID, name)
	if err != nil {
		return domain.ServiceDetail{}, err
	}
	envPriorities, err := s.metadataStore.ListEnvironmentPriorities(ctx, organizationID)
	if err != nil {
		return domain.ServiceDetail{}, err
	}
	historyRows, err := s.serviceStore.ListDeploymentHistory(ctx, organizationID, name, 200)
	if err != nil {
		return domain.ServiceDetail{}, err
	}
	currentState, err := s.analyticsStore.GetServiceCurrentState(ctx, organizationID, name)
	if err != nil {
		return domain.ServiceDetail{}, err
	}
	stats30d, err := s.analyticsStore.GetServiceDeliveryStats30d(ctx, organizationID, name)
	if err != nil {
		return domain.ServiceDetail{}, err
	}
	changeLinks, err := s.analyticsStore.ListServiceChangeLinksRecent(ctx, organizationID, name, 20)
	if err != nil {
		return domain.ServiceDetail{}, err
	}

	metadataFields := buildServiceMetadataFields(requiredFields, metadataRows)
	missingMetadata := 0
	for _, field := range metadataFields {
		if strings.TrimSpace(field.Value) == "" {
			missingMetadata++
		}
	}

	applyEnvironmentPriorityOrder(serviceEnvs, envPriorities)
	enrichReleaseChangeLogs(historyRows)
	enrichEnvironmentDeploymentStats(serviceEnvs, historyRows)

	return domain.ServiceDetail{
		Title:             service.Name,
		Description:       "",
		IntegrationType:   service.IntegrationType,
		MissingMetadata:   missingMetadata,
		MetadataSaveURL:   "/s/" + url.PathEscape(name) + "/metadata",
		MetadataFields:    metadataFields,
		OrgRequiredFields: mapRequiredFields(requiredFields),
		Environments:      serviceEnvs,
		DeploymentHistory: historyRows,
		LastStatus:        strings.TrimSpace(currentState.LastStatus),
		DriftCount:        currentState.DriftCount,
		FailedStreak:      currentState.FailedStreak,
		Success30d:        stats30d.Success30d,
		Failures30d:       stats30d.Failures30d,
		Rollbacks30d:      stats30d.Rollbacks30d,
		ChangeFailureRate: formatChangeFailureRate(stats30d.Success30d, stats30d.Failures30d, stats30d.Rollbacks30d),
		RiskEvents:        mapServiceRiskEvents(changeLinks),
	}, nil
}

func formatChangeFailureRate(successes, failures, rollbacks int) string {
	total := successes + failures + rollbacks
	if total <= 0 {
		return "0%"
	}
	rate := (float64(failures+rollbacks) / float64(total)) * 100
	return strings.TrimRight(strings.TrimRight(fmtFloat(rate), "0"), ".") + "%"
}

func mapServiceRiskEvents(rows []ports.ServiceChangeLink) []domain.ServiceRiskEvent {
	out := make([]domain.ServiceRiskEvent, 0, len(rows))
	for _, row := range rows {
		out = append(out, domain.ServiceRiskEvent{
			When:          formatTimestampFromUnixMs(row.EventTSMs),
			Environment:   strings.TrimSpace(row.Environment),
			Artifact:      strings.TrimSpace(row.ArtifactID),
			ChainID:       strings.TrimSpace(row.ChainID),
			PipelineRunID: strings.TrimSpace(row.PipelineRunID),
			RunURL:        strings.TrimSpace(row.RunURL),
			ActorName:     strings.TrimSpace(row.ActorName),
		})
	}
	return out
}

func formatTimestampFromUnixMs(ts int64) string {
	if ts <= 0 {
		return ""
	}
	return time.UnixMilli(ts).Local().Format("2006-01-02 15:04")
}

func enrichReleaseChangeLogs(rows []domain.DeploymentRecord) {
	byEnv := map[string][]int{}
	for i, row := range rows {
		env := strings.ToLower(strings.TrimSpace(row.Environment))
		byEnv[env] = append(byEnv[env], i)
	}
	for _, indexes := range byEnv {
		for i, idx := range indexes {
			current := strings.TrimSpace(rows[idx].Ref)
			if i+1 >= len(indexes) {
				rows[idx].PreviousRef = ""
				if current == "" {
					rows[idx].ChangeLog = "Initial deployment"
				} else {
					rows[idx].ChangeLog = "Initial release in this environment"
				}
				continue
			}
			prev := strings.TrimSpace(rows[indexes[i+1]].Ref)
			rows[idx].PreviousRef = prev
			switch {
			case current == "" && prev == "":
				rows[idx].ChangeLog = "Release reference unavailable"
			case current == prev:
				rows[idx].ChangeLog = "No code reference change"
			case prev == "":
				rows[idx].ChangeLog = "Reference started tracking"
			default:
				rows[idx].ChangeLog = "Updated from previous release"
			}
		}
	}
}

func enrichEnvironmentDeploymentStats(environments []domain.ServiceEnvironment, history []domain.DeploymentRecord) {
	now := time.Now()
	type counts struct{ c7, c30 int }
	byEnv := map[string]counts{}
	for _, row := range history {
		env := strings.ToLower(strings.TrimSpace(row.Environment))
		if env == "" {
			continue
		}
		parsed, err := time.ParseInLocation("2006-01-02 15:04", strings.TrimSpace(row.DeployedAt), time.Local)
		if err != nil {
			continue
		}
		delta := now.Sub(parsed)
		item := byEnv[env]
		if delta <= 7*24*time.Hour {
			item.c7++
		}
		if delta <= 30*24*time.Hour {
			item.c30++
		}
		byEnv[env] = item
	}
	for i := range environments {
		env := strings.ToLower(strings.TrimSpace(environments[i].Name))
		item := byEnv[env]
		environments[i].DeployCount7d = item.c7
		environments[i].DeployCount30d = item.c30
		environments[i].DailyRate30d = formatDailyRate(item.c30)
	}
}

func formatDailyRate(count30 int) string {
	if count30 <= 0 {
		return "0/day"
	}
	rate := float64(count30) / 30.0
	return strings.TrimRight(strings.TrimRight(fmtFloat(rate), "0"), ".") + "/day"
}

func fmtFloat(value float64) string {
	return strconv.FormatFloat(value, 'f', 2, 64)
}

type metadataFilterData struct {
	Options          []domain.MetadataFilterOption
	MissingByService map[string]int
	TagsByService    map[string]string
}

func (s *ServiceReadService) loadMetadataFilterData(ctx context.Context, organizationID int64) (metadataFilterData, error) {
	requiredRows, err := s.metadataStore.ListRequiredFields(ctx, organizationID)
	if err != nil {
		return metadataFilterData{}, err
	}

	required := map[string]bool{}
	filterable := map[string]string{}
	for _, row := range requiredRows {
		label := strings.ToLower(strings.TrimSpace(row.Label))
		if label == "" {
			continue
		}
		required[label] = true
		if row.Filterable {
			filterable[label] = strings.TrimSpace(row.Label)
		}
	}

	metadataRows, err := s.metadataStore.ListServiceMetadataValuesByOrganization(ctx, organizationID)
	if err != nil {
		return metadataFilterData{}, err
	}

	filledByService := map[string]map[string]bool{}
	tagsByService := map[string]map[string]bool{}
	tagLabels := map[string]string{}
	for _, row := range metadataRows {
		serviceName := strings.ToLower(strings.TrimSpace(row.ServiceName))
		label := strings.ToLower(strings.TrimSpace(row.Label))
		value := strings.TrimSpace(row.Value)
		if serviceName == "" || label == "" || value == "" {
			continue
		}
		if !required[label] {
			continue
		}
		if _, ok := filledByService[serviceName]; !ok {
			filledByService[serviceName] = map[string]bool{}
		}
		filledByService[serviceName][label] = true

		displayLabel, ok := filterable[label]
		if !ok {
			continue
		}
		tag := metadataTagValue(label, value)
		if tag == "" {
			continue
		}
		if _, ok := tagsByService[serviceName]; !ok {
			tagsByService[serviceName] = map[string]bool{}
		}
		tagsByService[serviceName][tag] = true
		tagLabels[tag] = displayLabel + ": " + value
	}

	missingByService := map[string]int{}
	requiredCount := len(required)
	if requiredCount > 0 {
		for serviceName, filled := range filledByService {
			missing := requiredCount - len(filled)
			if missing < 0 {
				missing = 0
			}
			missingByService[serviceName] = missing
		}
	}

	options := []domain.MetadataFilterOption{{Value: "all", Label: "All metadata tags"}}
	tagKeys := make([]string, 0, len(tagLabels))
	for tag := range tagLabels {
		tagKeys = append(tagKeys, tag)
	}
	sort.Strings(tagKeys)
	for _, tag := range tagKeys {
		options = append(options, domain.MetadataFilterOption{Value: tag, Label: tagLabels[tag]})
	}

	normalizedTags := map[string]string{}
	for serviceName, tags := range tagsByService {
		keys := make([]string, 0, len(tags))
		for tag := range tags {
			keys = append(keys, tag)
		}
		sort.Strings(keys)
		normalizedTags[serviceName] = "|" + strings.Join(keys, "|") + "|"
	}

	return metadataFilterData{Options: options, MissingByService: missingByService, TagsByService: normalizedTags}, nil
}

func applyMetadataToServices(services []domain.Service, metadataData metadataFilterData) []domain.Service {
	for i := range services {
		serviceName := strings.ToLower(strings.TrimSpace(services[i].Title))
		services[i].MissingMetadata = metadataData.MissingByService[serviceName]
		services[i].MetadataTags = metadataData.TagsByService[serviceName]
	}
	return services
}

func applyMetadataToDeployments(rows []domain.DeploymentRow, metadataData metadataFilterData) []domain.DeploymentRow {
	for i := range rows {
		serviceName := strings.ToLower(strings.TrimSpace(rows[i].Service))
		rows[i].MetadataTags = metadataData.TagsByService[serviceName]
	}
	return rows
}

func metadataTagValue(label, value string) string {
	label = strings.TrimSpace(strings.ToLower(label))
	value = strings.TrimSpace(strings.ToLower(value))
	if label == "" || value == "" {
		return ""
	}
	return label + ":" + value
}

func mapRequiredFields(rows []ports.RequiredField) []domain.MetadataField {
	fields := make([]domain.MetadataField, 0, len(rows))
	for _, row := range rows {
		fields = append(fields, domain.MetadataField{Label: row.Label, Value: "Missing", Filterable: row.Filterable})
	}
	return fields
}

func buildServiceMetadataFields(required []ports.RequiredField, existing []ports.MetadataValue) []domain.MetadataField {
	values := map[string]string{}
	for _, row := range existing {
		label := strings.TrimSpace(row.Label)
		if label != "" {
			values[strings.ToLower(label)] = strings.TrimSpace(row.Value)
		}
	}
	fields := make([]domain.MetadataField, 0, len(required))
	for _, req := range required {
		label := strings.TrimSpace(req.Label)
		if label == "" {
			continue
		}
		fields = append(fields, domain.MetadataField{Label: label, Value: strings.TrimSpace(values[strings.ToLower(label)])})
	}
	return fields
}

func normalizeEnvironmentOrderInput(values []string) []string {
	seen := map[string]bool{}
	result := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		key := strings.ToLower(value)
		if seen[key] {
			continue
		}
		seen[key] = true
		result = append(result, value)
	}
	return result
}

func applyEnvironmentPriorityOrder(environments []domain.ServiceEnvironment, orderedNames []string) {
	priority := map[string]int{}
	for i, name := range normalizeEnvironmentOrderInput(orderedNames) {
		priority[strings.ToLower(name)] = i
	}
	sort.SliceStable(environments, func(i, j int) bool {
		left := strings.ToLower(strings.TrimSpace(environments[i].Name))
		right := strings.ToLower(strings.TrimSpace(environments[j].Name))
		leftRank, leftKnown := priority[left]
		rightRank, rightKnown := priority[right]
		switch {
		case leftKnown && rightKnown:
			if leftRank != rightRank {
				return leftRank < rightRank
			}
			return left < right
		case leftKnown:
			return true
		case rightKnown:
			return false
		default:
			return left < right
		}
	})
}
