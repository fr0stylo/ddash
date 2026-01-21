package routes

import (
	"database/sql"
	"strings"
	"time"

	"github.com/fr0stylo/ddash/internal/db/queries"
	"github.com/fr0stylo/ddash/views/components"
)

func mapServiceInstances(rows []queries.ListServiceInstancesRow) []components.Service {
	services := make([]components.Service, 0, len(rows))
	for _, row := range rows {
		services = append(services, mapServiceInstanceRow(
			row.ServiceName,
			row.Environment,
			row.Status,
			row.Context,
			row.Team,
			row.Description,
			row.RepoUrl,
			row.LogsUrl,
			row.EndpointUrl,
			row.LastDeployAt,
			row.DeployDurationSeconds,
			row.Revision,
			row.CommitSha,
			row.CommitUrl,
			row.CommitIndex,
			row.ActionLabel,
			row.ActionKind,
			row.ActionDisabled,
		))
	}
	return services
}

func mapServiceInstancesByEnv(rows []queries.ListServiceInstancesByEnvRow) []components.Service {
	services := make([]components.Service, 0, len(rows))
	for _, row := range rows {
		services = append(services, mapServiceInstanceRow(
			row.ServiceName,
			row.Environment,
			row.Status,
			row.Context,
			row.Team,
			row.Description,
			row.RepoUrl,
			row.LogsUrl,
			row.EndpointUrl,
			row.LastDeployAt,
			row.DeployDurationSeconds,
			row.Revision,
			row.CommitSha,
			row.CommitUrl,
			row.CommitIndex,
			row.ActionLabel,
			row.ActionKind,
			row.ActionDisabled,
		))
	}
	return services
}

func mapServiceInstanceRow(
	name string,
	environment string,
	status string,
	context sql.NullString,
	team sql.NullString,
	description sql.NullString,
	repoURL sql.NullString,
	logsURL sql.NullString,
	endpointURL sql.NullString,
	lastDeploy sql.NullString,
	deployDuration sql.NullInt64,
	revision sql.NullString,
	commitSha sql.NullString,
	commitURL sql.NullString,
	commitIndex sql.NullInt64,
	actionLabel sql.NullString,
	actionKind sql.NullString,
	actionDisabled int64,
) components.Service {
	_ = commitIndex
	commit := valueOr(commitSha, revision)
	if strings.TrimSpace(commit) == "" {
		commit = "-"
	}

	return components.Service{
		Title:          name,
		Description:    nullString(description),
		Context:        nullString(context),
		Environment:    environment,
		Team:           nullString(team),
		Status:         statusFromString(status),
		LastDeploy:     nullStringOr(lastDeploy, "-"),
		DeployDuration: formatDuration(deployDuration),
		Revision:       nullStringOr(revision, "-"),
		CommitSHA:      commit,
		CommitURL:      nullString(commitURL),
		RepoURL:        nullString(repoURL),
		LogsURL:        nullString(logsURL),
		Endpoint:       nullString(endpointURL),
		ActionLabel:    nullString(actionLabel),
		ActionKind:     nullString(actionKind),
		ActionDisabled: actionDisabled != 0,
	}
}

func mapDeploymentRows(rows []queries.ListDeploymentsRow) []components.DeploymentRow {
	deployments := make([]components.DeploymentRow, 0, len(rows))
	for _, row := range rows {
		deployments = append(deployments, components.DeploymentRow{
			Service:     row.Service,
			Environment: row.Environment,
			DeployedAt:  row.DeployedAt,
			Status:      deploymentStatusFromString(row.Status),
			JobURL:      nullString(row.JobUrl),
		})
	}
	return deployments
}

func mapServiceEnvironments(rows []queries.ListServiceEnvironmentsRow) []components.ServiceEnvironment {
	environments := make([]components.ServiceEnvironment, 0, len(rows))
	for _, row := range rows {
		environments = append(environments, components.ServiceEnvironment{
			Name:        row.Name,
			LastDeploy:  row.ReleasedAt,
			DeployedRef: row.Ref,
			CommitURL:   nullString(row.ReleaseUrl),
		})
	}
	return environments
}

func mapServiceFields(rows []queries.ServiceField) []components.ServiceField {
	fields := make([]components.ServiceField, 0, len(rows))
	for _, row := range rows {
		fields = append(fields, components.ServiceField{
			Label: row.Label,
			Value: row.Value,
		})
	}
	return fields
}

func mapOrganizationRequiredFields(rows []queries.ListOrganizationRequiredFieldsRow) []components.ServiceField {
	fields := make([]components.ServiceField, 0, len(rows))
	for _, row := range rows {
		fields = append(fields, components.ServiceField{
			Label: row.Label,
			Value: row.FieldType,
		})
	}
	return fields
}

func mapPendingCommits(rows []queries.ListPendingCommitsNotInProdRow) []components.GitCommit {
	commits := make([]components.GitCommit, 0, len(rows))
	for _, row := range rows {
		commits = append(commits, components.GitCommit{
			SHA:     row.Sha,
			Message: row.Message,
			URL:     nullString(row.Url),
		})
	}
	return commits
}

func mapDeploymentHistory(rows []queries.ListDeploymentHistoryByServiceRow) []components.DeploymentRecord {
	records := make([]components.DeploymentRecord, 0, len(rows))
	for _, row := range rows {
		records = append(records, components.DeploymentRecord{
			Ref:         nullString(row.ReleaseRef),
			Commits:     int(nullInt64(row.CommitCount)),
			DeployedAt:  row.DeployedAt,
			ReleaseURL:  nullString(row.ReleaseUrl),
			Environment: row.Environment,
		})
	}
	return records
}

func nullString(value sql.NullString) string {
	if value.Valid {
		return value.String
	}
	return ""
}

func nullInt64(value sql.NullInt64) int64 {
	if value.Valid {
		return value.Int64
	}
	return 0
}

func nullStringOr(value sql.NullString, fallback string) string {
	if value.Valid {
		return value.String
	}
	return fallback
}

func valueOr(primary, fallback sql.NullString) string {
	if primary.Valid {
		return primary.String
	}
	if fallback.Valid {
		return fallback.String
	}
	return ""
}

func statusFromString(value string) components.Status {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "synced":
		return components.StatusSynced
	case "progressing":
		return components.StatusProgressing
	case "out-of-sync", "out_of_sync", "outofsync":
		return components.StatusOutOfSync
	case "warning":
		return components.StatusWarning
	case "unknown":
		return components.StatusUnknown
	case "all":
		return components.StatusAll
	default:
		return components.StatusUnknown
	}
}

func deploymentStatusFromString(value string) components.DeploymentStatus {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "queued":
		return components.DeploymentQueued
	case "processing":
		return components.DeploymentProcessing
	case "success":
		return components.DeploymentSuccess
	case "error":
		return components.DeploymentError
	default:
		return components.DeploymentQueued
	}
}

func formatDuration(value sql.NullInt64) string {
	if !value.Valid || value.Int64 <= 0 {
		return "-"
	}
	return time.Duration(value.Int64 * int64(time.Second)).String()
}

func nextServiceStatus(status components.Status) components.Status {
	switch status {
	case components.StatusSynced:
		return components.StatusOutOfSync
	case components.StatusOutOfSync:
		return components.StatusProgressing
	case components.StatusProgressing:
		return components.StatusSynced
	case components.StatusWarning, components.StatusUnknown, components.StatusAll:
		return components.StatusSynced
	default:
		return components.StatusSynced
	}
}

func deploymentMatchesFilter(row components.DeploymentRow, env, service string) bool {
	if env != "" && env != "all" && row.Environment != env {
		return false
	}
	if service != "" && service != "all" && row.Service != service {
		return false
	}
	return true
}
