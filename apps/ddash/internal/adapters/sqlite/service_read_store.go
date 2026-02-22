package sqlite

import (
	"context"
	"database/sql"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/fr0stylo/ddash/apps/ddash/internal/app/domain"
	"github.com/fr0stylo/ddash/apps/ddash/internal/app/ports"
	"github.com/fr0stylo/ddash/internal/db/queries"
)

var _ ports.ServiceReadStore = (*Store)(nil)

// ListServiceInstances lists service projections, optionally filtered by environment.
func (s *Store) ListServiceInstances(ctx context.Context, organizationID int64, env string) ([]domain.Service, error) {
	if env == "" || env == "all" {
		rows, err := s.database.ListServiceInstancesFromEvents(ctx, organizationID)
		if err != nil {
			return nil, err
		}
		return mapServiceInstancesRows(rows), nil
	}

	rows, err := s.database.ListServiceInstancesByEnvFromEvents(ctx, queries.ListServiceInstancesByEnvFromEventsParams{OrganizationID: organizationID, Env: env})
	if err != nil {
		return nil, err
	}
	return mapServiceInstancesByEnvRows(rows), nil
}

// ListDeployments lists deployment projections for filters.
func (s *Store) ListDeployments(ctx context.Context, organizationID int64, env, service string) ([]domain.DeploymentRow, error) {
	rows, err := s.database.ListDeploymentsFromEvents(ctx, queries.ListDeploymentsFromEventsParams{
		OrganizationID: organizationID,
		Env:            env,
		Service:        service,
	})
	if err != nil {
		return nil, err
	}
	return mapDeploymentsRows(rows), nil
}

// GetOrganizationRenderVersion returns a coarse version for rendered fragments.
func (s *Store) GetOrganizationRenderVersion(ctx context.Context, organizationID int64) (int64, error) {
	version, err := s.database.GetOrganizationRenderVersion(ctx, organizationID)
	if err != nil {
		return 0, err
	}
	return toInt64(version), nil
}

// GetServiceLatest returns latest known state for a service.
func (s *Store) GetServiceLatest(ctx context.Context, organizationID int64, name string) (ports.ServiceLatest, error) {
	row, err := s.database.GetServiceLatestFromEvents(ctx, queries.GetServiceLatestFromEventsParams{OrganizationID: organizationID, Service: name})
	if err != nil {
		return ports.ServiceLatest{}, err
	}
	return ports.ServiceLatest{
		Name:            toString(row.ServiceName),
		IntegrationType: row.IntegrationType,
	}, nil
}

// ListServiceEnvironments returns latest deployment per environment for a service.
func (s *Store) ListServiceEnvironments(ctx context.Context, organizationID int64, service string) ([]domain.ServiceEnvironment, error) {
	rows, err := s.database.ListServiceEnvironmentsFromEvents(ctx, queries.ListServiceEnvironmentsFromEventsParams{OrganizationID: organizationID, Service: service})
	if err != nil {
		return nil, err
	}
	out := make([]domain.ServiceEnvironment, 0, len(rows))
	for _, row := range rows {
		formatted := formatTimestamp(row.ReleasedAt)
		out = append(out, domain.ServiceEnvironment{
			Name:            toString(row.Name),
			LastDeploy:      formatted,
			LastDeployedAgo: relativeFromFormattedTimestamp(formatted),
			Ref:             toString(row.Ref),
		})
	}
	return out, nil
}

// ListDeploymentHistory returns deployment history for a service.
func (s *Store) ListDeploymentHistory(ctx context.Context, organizationID int64, service string, limit int64) ([]domain.DeploymentRecord, error) {
	rows, err := s.database.ListDeploymentHistoryByServiceFromEvents(ctx, queries.ListDeploymentHistoryByServiceFromEventsParams{
		OrganizationID: organizationID,
		Service:        service,
		Limit:          limit,
	})
	if err != nil {
		return nil, err
	}
	out := make([]domain.DeploymentRecord, 0, len(rows))
	for _, row := range rows {
		formatted := formatTimestamp(row.DeployedAt)
		out = append(out, domain.DeploymentRecord{
			Ref:         toString(row.ReleaseRef),
			Commits:     0,
			DeployedAt:  formatted,
			DeployedAgo: relativeFromFormattedTimestamp(formatted),
			Environment: toString(row.Environment),
		})
	}
	return out, nil
}

// ListServiceDependencies returns names of services this service depends on.
func (s *Store) ListServiceDependencies(ctx context.Context, organizationID int64, service string) ([]string, error) {
	rows, err := s.database.ListServiceDependencies(ctx, queries.ListServiceDependenciesParams{
		OrganizationID: organizationID,
		ServiceName:    strings.TrimSpace(service),
	})
	if err != nil {
		return nil, err
	}
	out := make([]string, 0, len(rows))
	for _, row := range rows {
		value := strings.TrimSpace(row)
		if value != "" {
			out = append(out, value)
		}
	}
	return out, nil
}

// ListServiceDependants returns names of services that depend on this service.
func (s *Store) ListServiceDependants(ctx context.Context, organizationID int64, service string) ([]string, error) {
	rows, err := s.database.ListServiceDependants(ctx, queries.ListServiceDependantsParams{
		OrganizationID: organizationID,
		ServiceName:    strings.TrimSpace(service),
	})
	if err != nil {
		return nil, err
	}
	out := make([]string, 0, len(rows))
	for _, row := range rows {
		value := strings.TrimSpace(row)
		if value != "" {
			out = append(out, value)
		}
	}
	return out, nil
}

// UpsertServiceDependency creates a dependency edge.
func (s *Store) UpsertServiceDependency(ctx context.Context, organizationID int64, serviceName, dependsOnServiceName string) error {
	serviceName = strings.TrimSpace(serviceName)
	dependsOnServiceName = strings.TrimSpace(dependsOnServiceName)
	if serviceName == "" || dependsOnServiceName == "" || strings.EqualFold(serviceName, dependsOnServiceName) {
		return nil
	}
	return s.database.UpsertServiceDependency(ctx, queries.UpsertServiceDependencyParams{
		OrganizationID:       organizationID,
		ServiceName:          serviceName,
		DependsOnServiceName: dependsOnServiceName,
	})
}

// DeleteServiceDependency removes a dependency edge.
func (s *Store) DeleteServiceDependency(ctx context.Context, organizationID int64, serviceName, dependsOnServiceName string) error {
	serviceName = strings.TrimSpace(serviceName)
	dependsOnServiceName = strings.TrimSpace(dependsOnServiceName)
	if serviceName == "" || dependsOnServiceName == "" {
		return nil
	}
	return s.database.DeleteServiceDependency(ctx, queries.DeleteServiceDependencyParams{
		OrganizationID:       organizationID,
		ServiceName:          serviceName,
		DependsOnServiceName: dependsOnServiceName,
	})
}

// ListRequiredFields returns required metadata fields for an organization.
func (s *Store) ListRequiredFields(ctx context.Context, organizationID int64) ([]ports.RequiredField, error) {
	rows, err := s.database.ListOrganizationRequiredFields(ctx, organizationID)
	if err != nil {
		return nil, err
	}
	out := make([]ports.RequiredField, 0, len(rows))
	for _, row := range rows {
		out = append(out, ports.RequiredField{
			Label:      row.Label,
			Type:       row.FieldType,
			Filterable: row.IsFilterable != 0,
		})
	}
	return out, nil
}

// ListServiceMetadata returns metadata values for a service.
func (s *Store) ListServiceMetadata(ctx context.Context, organizationID int64, service string) ([]ports.MetadataValue, error) {
	rows, err := s.database.ListServiceMetadataByService(ctx, queries.ListServiceMetadataByServiceParams{
		OrganizationID: organizationID,
		ServiceName:    service,
	})
	if err != nil {
		return nil, err
	}
	out := make([]ports.MetadataValue, 0, len(rows))
	for _, row := range rows {
		out = append(out, ports.MetadataValue{Label: row.Label, Value: row.Value})
	}
	return out, nil
}

// ListServiceMetadataValuesByOrganization returns metadata values across all services.
func (s *Store) ListServiceMetadataValuesByOrganization(ctx context.Context, organizationID int64) ([]ports.ServiceMetadataValue, error) {
	rows, err := s.database.ListServiceMetadataByOrganization(ctx, organizationID)
	if err != nil {
		return nil, err
	}
	out := make([]ports.ServiceMetadataValue, 0, len(rows))
	for _, row := range rows {
		out = append(out, ports.ServiceMetadataValue{
			ServiceName: row.ServiceName,
			Label:       row.Label,
			Value:       row.Value,
		})
	}
	return out, nil
}

// ListEnvironmentPriorities returns configured environment ordering.
func (s *Store) ListEnvironmentPriorities(ctx context.Context, organizationID int64) ([]string, error) {
	rows, err := s.database.ListOrganizationEnvironmentPriorities(ctx, organizationID)
	if err != nil {
		if isMissingEnvPriorityTableErr(err) {
			return nil, nil
		}
		return nil, err
	}
	out := make([]string, 0, len(rows))
	for _, row := range rows {
		value := strings.TrimSpace(row.Environment)
		if value != "" {
			out = append(out, value)
		}
	}
	return out, nil
}

// ListDiscoveredEnvironments returns discovered environment names from event stream.
func (s *Store) ListDiscoveredEnvironments(ctx context.Context, organizationID int64) ([]string, error) {
	return s.database.ListDistinctServiceEnvironmentsFromEvents(ctx, organizationID)
}

// GetServiceCurrentState returns latest projected state values.
func (s *Store) GetServiceCurrentState(ctx context.Context, organizationID int64, service string) (ports.ServiceCurrentState, error) {
	row, err := s.database.GetServiceCurrentState(ctx, queries.GetServiceCurrentStateParams{
		OrganizationID: organizationID,
		ServiceName:    service,
	})
	if err != nil {
		if err == sql.ErrNoRows {
			return ports.ServiceCurrentState{}, nil
		}
		return ports.ServiceCurrentState{}, err
	}
	return ports.ServiceCurrentState{
		LastStatus:    strings.TrimSpace(row.LatestStatus),
		LastEventTSMs: row.LatestEventTsMs,
		DriftCount:    int(row.DriftCount),
		FailedStreak:  int(row.FailedStreak),
	}, nil
}

// GetServiceDeliveryStats30d returns 30-day delivery counters.
func (s *Store) GetServiceDeliveryStats30d(ctx context.Context, organizationID int64, service string) (ports.ServiceDeliveryStats, error) {
	row, err := s.database.GetServiceDeliveryStats30d(ctx, queries.GetServiceDeliveryStats30dParams{
		OrganizationID: organizationID,
		ServiceName:    service,
	})
	if err != nil {
		if err == sql.ErrNoRows {
			return ports.ServiceDeliveryStats{}, nil
		}
		return ports.ServiceDeliveryStats{}, err
	}
	return ports.ServiceDeliveryStats{
		Success30d:   int(toInt64(row.DeploySuccessCount)),
		Failures30d:  int(toInt64(row.DeployFailureCount)),
		Rollbacks30d: int(toInt64(row.RollbackCount)),
	}, nil
}

// ListServiceChangeLinksRecent returns recent risk/audit links.
func (s *Store) ListServiceChangeLinksRecent(ctx context.Context, organizationID int64, service string, limit int64) ([]ports.ServiceChangeLink, error) {
	rows, err := s.database.ListServiceChangeLinksRecent(ctx, queries.ListServiceChangeLinksRecentParams{
		OrganizationID: organizationID,
		ServiceName:    service,
		Limit:          limit,
	})
	if err != nil {
		return nil, err
	}
	out := make([]ports.ServiceChangeLink, 0, len(rows))
	for _, row := range rows {
		item := ports.ServiceChangeLink{
			EventTSMs:     row.EventTsMs,
			Environment:   strings.TrimSpace(row.Environment),
			ArtifactID:    strings.TrimSpace(row.ArtifactID),
			PipelineRunID: strings.TrimSpace(row.PipelineRunID),
			RunURL:        strings.TrimSpace(row.RunUrl),
			ActorName:     strings.TrimSpace(row.ActorName),
		}
		if row.ChainID.Valid {
			item.ChainID = strings.TrimSpace(row.ChainID.String)
		}
		out = append(out, item)
	}
	return out, nil
}

// ListServiceLeadTimeSamples returns raw lead-time samples (seconds) for change->deploy ordering.
func (s *Store) ListServiceLeadTimeSamples(ctx context.Context, organizationID int64, sinceMs int64) ([]ports.ServiceLeadTimeSample, error) {
	rows, err := s.database.ListServiceLeadTimeSamplesFromEvents(ctx, queries.ListServiceLeadTimeSamplesFromEventsParams{
		OrganizationID: organizationID,
		SinceMs:        sinceMs,
	})
	if err != nil {
		return nil, err
	}
	out := make([]ports.ServiceLeadTimeSample, 0, len(rows))
	for _, row := range rows {
		out = append(out, ports.ServiceLeadTimeSample{
			DayUTC:      toString(row.DayUtc),
			ServiceName: toString(row.ServiceName),
			LeadSeconds: row.LeadSeconds,
		})
	}
	return out, nil
}

func mapOrganization(org queries.Organization) ports.Organization {
	joinCode := ""
	if org.JoinCode.Valid {
		joinCode = org.JoinCode.String
	}
	return ports.Organization{
		ID:            org.ID,
		Name:          org.Name,
		AuthToken:     org.AuthToken,
		JoinCode:      joinCode,
		WebhookSecret: org.WebhookSecret,
		Enabled:       org.Enabled != 0,
	}
}

func mapServiceInstancesRows(rows []queries.ListServiceInstancesFromEventsRow) []domain.Service {
	out := make([]domain.Service, 0, len(rows))
	for _, row := range rows {
		out = append(out, domain.Service{
			Title:          toString(row.ServiceName),
			Environment:    toString(row.Environment),
			Status:         mapServiceStatus(row.Status),
			LastDeploy:     formatTimestamp(row.LastDeployAt),
			Revision:       toString(row.ArtifactID),
			CommitSHA:      toString(row.ArtifactID),
			DeployDuration: "-",
		})
	}
	return out
}

func mapServiceInstancesByEnvRows(rows []queries.ListServiceInstancesByEnvFromEventsRow) []domain.Service {
	out := make([]domain.Service, 0, len(rows))
	for _, row := range rows {
		out = append(out, domain.Service{
			Title:          toString(row.ServiceName),
			Environment:    toString(row.Environment),
			Status:         mapServiceStatus(row.Status),
			LastDeploy:     formatTimestamp(row.LastDeployAt),
			Revision:       toString(row.ArtifactID),
			CommitSHA:      toString(row.ArtifactID),
			DeployDuration: "-",
		})
	}
	return out
}

func mapDeploymentsRows(rows []queries.ListDeploymentsFromEventsRow) []domain.DeploymentRow {
	out := make([]domain.DeploymentRow, 0, len(rows))
	for _, row := range rows {
		out = append(out, domain.DeploymentRow{
			Service:     toString(row.Service),
			Environment: toString(row.Environment),
			DeployedAt:  formatTimestamp(row.DeployedAt),
			Status:      mapDeploymentStatus(row.Status),
		})
	}
	return out
}

func mapServiceStatus(value string) domain.ServiceStatus {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "synced":
		return domain.ServiceStatusSynced
	case "progressing":
		return domain.ServiceStatusProgressing
	case "out-of-sync", "out_of_sync", "outofsync":
		return domain.ServiceStatusOutOfSync
	case "warning":
		return domain.ServiceStatusWarning
	case "all":
		return domain.ServiceStatusAll
	default:
		return domain.ServiceStatusUnknown
	}
}

func mapDeploymentStatus(value string) domain.DeploymentStatus {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "queued":
		return domain.DeploymentStatusQueued
	case "processing":
		return domain.DeploymentStatusProcessing
	case "success":
		return domain.DeploymentStatusSuccess
	case "error":
		return domain.DeploymentStatusError
	default:
		return domain.DeploymentStatusQueued
	}
}

func toString(value interface{}) string {
	switch v := value.(type) {
	case nil:
		return ""
	case string:
		return v
	case []byte:
		return string(v)
	default:
		return strings.TrimSpace(fmt.Sprint(v))
	}
}

func toInt64(value interface{}) int64 {
	switch v := value.(type) {
	case nil:
		return 0
	case int64:
		return v
	case int32:
		return int64(v)
	case int:
		return int64(v)
	case float64:
		return int64(v)
	case float32:
		return int64(v)
	case []byte:
		parsed, err := strconv.ParseInt(strings.TrimSpace(string(v)), 10, 64)
		if err != nil {
			return 0
		}
		return parsed
	case string:
		parsed, err := strconv.ParseInt(strings.TrimSpace(v), 10, 64)
		if err != nil {
			return 0
		}
		return parsed
	default:
		text := strings.TrimSpace(fmt.Sprint(v))
		parsed, err := strconv.ParseInt(text, 10, 64)
		if err != nil {
			return 0
		}
		return parsed
	}
}

func formatTimestamp(value string) string {
	text := strings.TrimSpace(value)
	if text == "" {
		return ""
	}
	for _, layout := range []string{time.RFC3339Nano, time.RFC3339, "2006-01-02 15:04:05", "2006-01-02 15:04"} {
		parsed, err := time.Parse(layout, text)
		if err == nil {
			return parsed.Local().Format("2006-01-02 15:04")
		}
	}
	return text
}

func relativeFromFormattedTimestamp(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	parsed, err := time.ParseInLocation("2006-01-02 15:04", value, time.Local)
	if err != nil {
		return ""
	}
	delta := time.Since(parsed)
	if delta < 0 {
		delta = -delta
	}
	hours := int(delta.Hours())
	switch {
	case hours < 1:
		minutes := int(delta.Minutes())
		if minutes <= 1 {
			return "just now"
		}
		return fmt.Sprintf("%dm ago", minutes)
	case hours < 24:
		return fmt.Sprintf("%dh ago", hours)
	case hours < 24*30:
		return fmt.Sprintf("%dd ago", hours/24)
	default:
		return fmt.Sprintf("%dmo ago", hours/(24*30))
	}
}

func isMissingEnvPriorityTableErr(err error) bool {
	if err == nil {
		return false
	}
	message := strings.ToLower(err.Error())
	return strings.Contains(message, "no such table") && strings.Contains(message, "organization_environment_priorities")
}
