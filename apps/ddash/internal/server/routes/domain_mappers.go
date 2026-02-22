package routes

import (
	"github.com/fr0stylo/ddash/apps/ddash/internal/app/domain"
	"github.com/fr0stylo/ddash/views/components"
)

func mapDomainServices(rows []domain.Service) []components.Service {
	out := make([]components.Service, 0, len(rows))
	for _, row := range rows {
		out = append(out, components.Service{
			Title:           row.Title,
			Environment:     row.Environment,
			Status:          mapDomainServiceStatus(row.Status),
			LastDeploy:      row.LastDeploy,
			Revision:        row.Revision,
			CommitSHA:       row.CommitSHA,
			DeployDuration:  row.DeployDuration,
			MissingMetadata: row.MissingMetadata,
			MetadataTags:    row.MetadataTags,
		})
	}
	return out
}

func mapDomainDeployments(rows []domain.DeploymentRow) []components.DeploymentRow {
	out := make([]components.DeploymentRow, 0, len(rows))
	for _, row := range rows {
		out = append(out, components.DeploymentRow{
			Service:      row.Service,
			Environment:  row.Environment,
			DeployedAt:   row.DeployedAt,
			Status:       mapDomainDeploymentStatus(row.Status),
			MetadataTags: row.MetadataTags,
		})
	}
	return out
}

func mapDomainMetadataOptions(rows []domain.MetadataFilterOption) []components.MetadataFilterOption {
	out := make([]components.MetadataFilterOption, 0, len(rows))
	for _, row := range rows {
		out = append(out, components.MetadataFilterOption{Value: row.Value, Label: row.Label})
	}
	return out
}

func mapDomainServiceDetail(detail domain.ServiceDetail) components.ServiceDetail {
	metadataFields := mapDomainMetadataFields(detail.MetadataFields)
	requiredFields := mapDomainMetadataFields(detail.OrgRequiredFields)

	envs := make([]components.ServiceEnvironment, 0, len(detail.Environments))
	for _, env := range detail.Environments {
		envs = append(envs, components.ServiceEnvironment{
			Name:            env.Name,
			LastDeploy:      env.LastDeploy,
			LastDeployedAgo: env.LastDeployedAgo,
			DeployedRef:     env.Ref,
			DeployCount7d:   env.DeployCount7d,
			DeployCount30d:  env.DeployCount30d,
			DailyRate30d:    env.DailyRate30d,
		})
	}

	history := make([]components.DeploymentRecord, 0, len(detail.DeploymentHistory))
	for _, row := range detail.DeploymentHistory {
		history = append(history, components.DeploymentRecord{
			Ref:         row.Ref,
			PreviousRef: row.PreviousRef,
			ChangeLog:   row.ChangeLog,
			Commits:     row.Commits,
			DeployedAt:  row.DeployedAt,
			DeployedAgo: row.DeployedAgo,
			Environment: row.Environment,
		})
	}

	riskEvents := make([]components.ServiceRiskEvent, 0, len(detail.RiskEvents))
	for _, row := range detail.RiskEvents {
		riskEvents = append(riskEvents, components.ServiceRiskEvent{
			When:          row.When,
			Environment:   row.Environment,
			Artifact:      row.Artifact,
			ChainID:       row.ChainID,
			PipelineRunID: row.PipelineRunID,
			RunURL:        row.RunURL,
			ActorName:     row.ActorName,
		})
	}

	return components.ServiceDetail{
		Title:             detail.Title,
		Description:       detail.Description,
		IntegrationType:   detail.IntegrationType,
		MissingMetadata:   detail.MissingMetadata,
		MetadataSaveURL:   detail.MetadataSaveURL,
		MetadataFields:    metadataFields,
		OrgRequiredFields: requiredFields,
		Environments:      envs,
		DeploymentHistory: history,
		LastStatus:        detail.LastStatus,
		DriftCount:        detail.DriftCount,
		FailedStreak:      detail.FailedStreak,
		Success30d:        detail.Success30d,
		Failures30d:       detail.Failures30d,
		Rollbacks30d:      detail.Rollbacks30d,
		ChangeFailureRate: detail.ChangeFailureRate,
		RiskEvents:        riskEvents,
		Dependencies:      detail.Dependencies,
		Dependants:        detail.Dependants,
		AvailableServices: detail.AvailableServices,
	}
}

func mapDomainMetadataFields(fields []domain.MetadataField) []components.ServiceField {
	out := make([]components.ServiceField, 0, len(fields))
	for _, field := range fields {
		out = append(out, components.ServiceField{
			Label:      field.Label,
			Value:      field.Value,
			Filterable: field.Filterable,
		})
	}
	return out
}

func mapDomainServiceStatus(status domain.ServiceStatus) components.Status {
	switch status {
	case domain.ServiceStatusSynced:
		return components.StatusSynced
	case domain.ServiceStatusProgressing:
		return components.StatusProgressing
	case domain.ServiceStatusOutOfSync:
		return components.StatusOutOfSync
	case domain.ServiceStatusWarning:
		return components.StatusWarning
	case domain.ServiceStatusUnknown:
		return components.StatusUnknown
	case domain.ServiceStatusAll:
		return components.StatusAll
	default:
		return components.StatusUnknown
	}
}

func mapDomainDeploymentStatus(status domain.DeploymentStatus) components.DeploymentStatus {
	switch status {
	case domain.DeploymentStatusQueued:
		return components.DeploymentQueued
	case domain.DeploymentStatusProcessing:
		return components.DeploymentProcessing
	case domain.DeploymentStatusSuccess:
		return components.DeploymentSuccess
	case domain.DeploymentStatusError:
		return components.DeploymentError
	default:
		return components.DeploymentQueued
	}
}
