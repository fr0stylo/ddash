package sqlite

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/fr0stylo/ddash/internal/app/domain"
	"github.com/fr0stylo/ddash/internal/app/ports"
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
		out = append(out, domain.ServiceEnvironment{
			Name:       toString(row.Name),
			LastDeploy: formatTimestamp(row.ReleasedAt),
			Ref:        toString(row.Ref),
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
		out = append(out, domain.DeploymentRecord{
			Ref:         toString(row.ReleaseRef),
			Commits:     0,
			DeployedAt:  formatTimestamp(row.DeployedAt),
			Environment: toString(row.Environment),
		})
	}
	return out, nil
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

func isMissingEnvPriorityTableErr(err error) bool {
	if err == nil {
		return false
	}
	message := strings.ToLower(err.Error())
	return strings.Contains(message, "no such table") && strings.Contains(message, "organization_environment_priorities")
}
