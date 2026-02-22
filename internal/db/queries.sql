-- name: GetDefaultOrganization :one
SELECT *
FROM organizations
ORDER BY id
LIMIT 1;

-- name: GetOrganizationByAuthToken :one
SELECT *
FROM organizations
WHERE auth_token = ?
LIMIT 1;

-- name: GetOrganizationByID :one
SELECT *
FROM organizations
WHERE id = ?
LIMIT 1;

-- name: GetOrganizationByJoinCode :one
SELECT *
FROM organizations
WHERE join_code = ?
LIMIT 1;

-- name: ListOrganizations :many
SELECT *
FROM organizations
ORDER BY name, id;

-- name: UpdateOrganizationName :exec
UPDATE organizations
SET name = ?, updated_at = CURRENT_TIMESTAMP
WHERE id = ?;

-- name: UpdateOrganizationEnabled :exec
UPDATE organizations
SET enabled = ?, updated_at = CURRENT_TIMESTAMP
WHERE id = ?;

-- name: DeleteOrganization :exec
DELETE FROM organizations
WHERE id = ?;

-- name: UpsertUser :one
INSERT INTO users (github_id, email, nickname, name, avatar_url)
VALUES (?, ?, ?, ?, ?)
ON CONFLICT(email) DO UPDATE SET
  github_id = excluded.github_id,
  nickname = excluded.nickname,
  name = excluded.name,
  avatar_url = excluded.avatar_url,
  updated_at = CURRENT_TIMESTAMP
RETURNING *;

-- name: GetUserByID :one
SELECT *
FROM users
WHERE id = ?
LIMIT 1;

-- name: GetUserByEmailOrNickname :one
SELECT *
FROM users
WHERE email = ? OR nickname = ?
LIMIT 1;

-- name: ListOrganizationsByUser :many
SELECT o.*
FROM organizations o
JOIN organization_members m ON m.organization_id = o.id
WHERE m.user_id = ?
ORDER BY o.name, o.id;

-- name: GetOrganizationMemberRole :one
SELECT role
FROM organization_members
WHERE organization_id = ? AND user_id = ?
LIMIT 1;

-- name: UpsertOrganizationMember :exec
INSERT INTO organization_members (organization_id, user_id, role)
VALUES (?, ?, ?)
ON CONFLICT(organization_id, user_id) DO UPDATE SET
  role = excluded.role,
  updated_at = CURRENT_TIMESTAMP;

-- name: DeleteOrganizationMember :exec
DELETE FROM organization_members
WHERE organization_id = ? AND user_id = ?;

-- name: CountOrganizationOwners :one
SELECT COUNT(*)
FROM organization_members
WHERE organization_id = ? AND role = 'owner';

-- name: ListOrganizationMembers :many
SELECT
  u.id AS user_id,
  u.email,
  u.nickname,
  u.name,
  u.avatar_url,
  m.role
FROM organization_members m
JOIN users u ON u.id = m.user_id
WHERE m.organization_id = ?
ORDER BY
  CASE m.role WHEN 'owner' THEN 0 WHEN 'admin' THEN 1 ELSE 2 END,
  COALESCE(NULLIF(u.name, ''), u.nickname, u.email);

-- name: UpsertGitHubInstallationMapping :exec
INSERT INTO github_installation_mappings (
  installation_id,
  organization_id,
  organization_label,
  default_environment,
  enabled
)
VALUES (
  sqlc.arg('installation_id'),
  sqlc.arg('organization_id'),
  sqlc.arg('organization_label'),
  sqlc.arg('default_environment'),
  sqlc.arg('enabled')
)
ON CONFLICT(installation_id) DO UPDATE SET
  organization_id = excluded.organization_id,
  organization_label = excluded.organization_label,
  default_environment = excluded.default_environment,
  enabled = excluded.enabled,
  updated_at = CURRENT_TIMESTAMP;

-- name: ListGitHubInstallationMappings :many
SELECT
  installation_id,
  organization_id,
  organization_label,
  default_environment,
  enabled
FROM github_installation_mappings
WHERE organization_id = sqlc.arg('organization_id')
ORDER BY installation_id ASC;

-- name: DeleteGitHubInstallationMapping :execrows
DELETE FROM github_installation_mappings
WHERE installation_id = sqlc.arg('installation_id')
  AND organization_id = sqlc.arg('organization_id');

-- name: GetOrganizationByGitHubInstallationID :one
SELECT o.*
FROM organizations o
JOIN github_installation_mappings m ON m.organization_id = o.id
WHERE m.installation_id = sqlc.arg('installation_id')
  AND m.enabled = 1
  AND o.enabled = 1
LIMIT 1;

-- name: UpsertGitLabProjectMapping :exec
INSERT INTO gitlab_project_mappings (
  project_id,
  organization_id,
  project_path,
  default_environment,
  enabled
)
VALUES (
  sqlc.arg('project_id'),
  sqlc.arg('organization_id'),
  sqlc.arg('project_path'),
  sqlc.arg('default_environment'),
  sqlc.arg('enabled')
)
ON CONFLICT(project_id) DO UPDATE SET
  organization_id = excluded.organization_id,
  project_path = excluded.project_path,
  default_environment = excluded.default_environment,
  enabled = excluded.enabled,
  updated_at = CURRENT_TIMESTAMP;

-- name: GetOrganizationByGitLabProjectID :one
SELECT o.*
FROM organizations o
JOIN gitlab_project_mappings m ON m.organization_id = o.id
WHERE m.project_id = sqlc.arg('project_id')
  AND m.enabled = 1
  AND o.enabled = 1
LIMIT 1;

-- name: CreateGitHubSetupIntent :exec
INSERT INTO github_setup_intents (
  state,
  organization_id,
  organization_label,
  default_environment,
  expires_at
)
VALUES (
  sqlc.arg('state'),
  sqlc.arg('organization_id'),
  sqlc.arg('organization_label'),
  sqlc.arg('default_environment'),
  sqlc.arg('expires_at')
);

-- name: GetGitHubSetupIntentByState :one
SELECT
  state,
  organization_id,
  organization_label,
  default_environment,
  expires_at
FROM github_setup_intents
WHERE state = sqlc.arg('state')
LIMIT 1;

-- name: DeleteGitHubSetupIntent :exec
DELETE FROM github_setup_intents
WHERE state = sqlc.arg('state');

-- name: CreateOrganization :one
INSERT INTO organizations (name, auth_token, join_code, webhook_secret, enabled)
VALUES (sqlc.arg('name'), sqlc.arg('auth_token'), sqlc.arg('join_code'), sqlc.arg('webhook_secret'), sqlc.arg('enabled'))
RETURNING *;

-- name: UpsertOrganizationJoinRequest :exec
INSERT INTO organization_join_requests (organization_id, user_id, request_code, status)
VALUES (?, ?, ?, 'pending')
ON CONFLICT(organization_id, user_id) DO UPDATE SET
  request_code = excluded.request_code,
  status = 'pending',
  reviewed_by = NULL,
  reviewed_at = NULL,
  updated_at = CURRENT_TIMESTAMP;

-- name: ListPendingOrganizationJoinRequests :many
SELECT
  r.organization_id,
  r.user_id,
  r.request_code,
  r.status,
  u.email,
  u.nickname,
  u.name
FROM organization_join_requests r
JOIN users u ON u.id = r.user_id
WHERE r.organization_id = ?
  AND r.status = 'pending'
ORDER BY r.created_at, r.id;

-- name: SetOrganizationJoinRequestStatus :exec
UPDATE organization_join_requests
SET status = ?, reviewed_by = ?, reviewed_at = CURRENT_TIMESTAMP, updated_at = CURRENT_TIMESTAMP
WHERE organization_id = ? AND user_id = ?;

-- name: ListOrganizationRequiredFields :many
SELECT id, organization_id, label, field_type, sort_order, is_filterable
FROM organization_required_fields
WHERE organization_id = ?
ORDER BY sort_order, id;

-- name: DeleteOrganizationRequiredFields :exec
DELETE FROM organization_required_fields
WHERE organization_id = ?;

-- name: CreateOrganizationRequiredField :one
INSERT INTO organization_required_fields (organization_id, label, field_type, sort_order, is_filterable)
VALUES (sqlc.arg('organization_id'), sqlc.arg('label'), sqlc.arg('field_type'), sqlc.arg('sort_order'), sqlc.arg('is_filterable'))
RETURNING *;

-- name: ListOrganizationEnvironmentPriorities :many
SELECT id, organization_id, environment, sort_order
FROM organization_environment_priorities
WHERE organization_id = ?
ORDER BY sort_order, id;

-- name: DeleteOrganizationEnvironmentPriorities :exec
DELETE FROM organization_environment_priorities
WHERE organization_id = ?;

-- name: CreateOrganizationEnvironmentPriority :one
INSERT INTO organization_environment_priorities (organization_id, environment, sort_order)
VALUES (sqlc.arg('organization_id'), sqlc.arg('environment'), sqlc.arg('sort_order'))
RETURNING *;

-- name: ListServiceMetadataByService :many
SELECT label, value
FROM service_metadata
WHERE organization_id = sqlc.arg('organization_id')
  AND service_name = sqlc.arg('service_name')
ORDER BY label;

-- name: ListServiceMetadataByOrganization :many
SELECT service_name, label, value
FROM service_metadata
WHERE organization_id = sqlc.arg('organization_id')
ORDER BY service_name, label;

-- name: DeleteServiceMetadataByService :exec
DELETE FROM service_metadata
WHERE organization_id = sqlc.arg('organization_id')
  AND service_name = sqlc.arg('service_name');

-- name: UpsertServiceMetadata :exec
INSERT INTO service_metadata (organization_id, service_name, label, value)
VALUES (sqlc.arg('organization_id'), sqlc.arg('service_name'), sqlc.arg('label'), sqlc.arg('value'))
ON CONFLICT(organization_id, service_name, label) DO UPDATE SET
  value = excluded.value,
  updated_at = CURRENT_TIMESTAMP;

-- name: ListDistinctServiceEnvironmentsFromEvents :many
SELECT DISTINCT COALESCE(NULLIF(json_extract(es.raw_event_json, '$.subject.content.environment.id'), ''), 'unknown') AS environment
FROM event_store es
WHERE es.organization_id = sqlc.arg('organization_id')
  AND es.subject_type = 'service'
ORDER BY environment;

-- name: UpdateOrganizationSecrets :exec
UPDATE organizations
SET auth_token = ?, webhook_secret = ?, enabled = ?, updated_at = CURRENT_TIMESTAMP
WHERE id = ?;

-- name: ListOrganizationFeatures :many
SELECT feature_key, is_enabled
FROM organization_features
WHERE organization_id = ?
ORDER BY feature_key;

-- name: UpsertOrganizationFeature :exec
INSERT INTO organization_features (organization_id, feature_key, is_enabled)
VALUES (?, ?, ?)
ON CONFLICT(organization_id, feature_key) DO UPDATE SET
  is_enabled = excluded.is_enabled,
  updated_at = CURRENT_TIMESTAMP;

-- name: ListOrganizationPreferences :many
SELECT preference_key, preference_value
FROM organization_preferences
WHERE organization_id = ?
ORDER BY preference_key;

-- name: UpsertOrganizationPreference :exec
INSERT INTO organization_preferences (organization_id, preference_key, preference_value)
VALUES (?, ?, ?)
ON CONFLICT(organization_id, preference_key) DO UPDATE SET
  preference_value = excluded.preference_value,
  updated_at = CURRENT_TIMESTAMP;

-- name: AppendEventStore :one
INSERT INTO event_store (
  organization_id,
  event_id,
  event_type,
  event_source,
  event_timestamp,
  event_ts_ms,
  subject_id,
  subject_source,
  subject_type,
  chain_id,
  raw_event_json
)
VALUES (
  sqlc.arg('organization_id'),
  sqlc.arg('event_id'),
  sqlc.arg('event_type'),
  sqlc.arg('event_source'),
  sqlc.arg('event_timestamp'),
  sqlc.arg('event_ts_ms'),
  sqlc.arg('subject_id'),
  sqlc.narg('subject_source'),
  sqlc.arg('subject_type'),
  sqlc.narg('chain_id'),
  sqlc.arg('raw_event_json')
)
ON CONFLICT(organization_id, event_source, event_id) DO NOTHING
RETURNING seq;

-- name: UpsertServiceEnvStateFromEventSeq :exec
INSERT INTO service_env_state (
  organization_id,
  service_name,
  environment,
  latest_event_seq,
  latest_event_type,
  latest_event_ts_ms,
  latest_status,
  latest_artifact_id
)
SELECT
  es.organization_id,
  CASE
    WHEN instr(es.subject_id, '/') > 0 THEN substr(es.subject_id, instr(es.subject_id, '/') + 1)
    ELSE es.subject_id
  END AS service_name,
  COALESCE(NULLIF(json_extract(es.raw_event_json, '$.subject.content.environment.id'), ''), 'unknown') AS environment,
  es.seq,
  es.event_type,
  es.event_ts_ms,
  CASE
    WHEN es.event_type LIKE 'dev.cdevents.service.deployed.%' THEN 'synced'
    WHEN es.event_type LIKE 'dev.cdevents.service.upgraded.%' THEN 'synced'
    WHEN es.event_type LIKE 'dev.cdevents.service.published.%' THEN 'synced'
    WHEN es.event_type LIKE 'dev.cdevents.service.rolledback.%' THEN 'warning'
    WHEN es.event_type LIKE 'dev.cdevents.service.removed.%' THEN 'out-of-sync'
    ELSE 'unknown'
  END AS latest_status,
  COALESCE(json_extract(es.raw_event_json, '$.subject.content.artifactId'), '') AS latest_artifact_id
FROM event_store es
WHERE es.organization_id = sqlc.arg('organization_id')
  AND es.seq = sqlc.arg('seq')
  AND es.subject_type = 'service'
ON CONFLICT(organization_id, service_name, environment) DO UPDATE SET
  latest_event_seq = excluded.latest_event_seq,
  latest_event_type = excluded.latest_event_type,
  latest_event_ts_ms = excluded.latest_event_ts_ms,
  latest_status = excluded.latest_status,
  latest_artifact_id = excluded.latest_artifact_id,
  updated_at = CURRENT_TIMESTAMP
WHERE
  excluded.latest_event_ts_ms > service_env_state.latest_event_ts_ms
  OR (excluded.latest_event_ts_ms = service_env_state.latest_event_ts_ms AND excluded.latest_event_seq > service_env_state.latest_event_seq);

-- name: UpsertServiceDeliveryStatsDailyFromEventSeq :exec
INSERT INTO service_delivery_stats_daily (
  organization_id,
  service_name,
  day_utc,
  deploy_success_count,
  deploy_failure_count,
  rollback_count
)
SELECT
  es.organization_id,
  CASE
    WHEN instr(es.subject_id, '/') > 0 THEN substr(es.subject_id, instr(es.subject_id, '/') + 1)
    ELSE es.subject_id
  END AS service_name,
  date(datetime(es.event_ts_ms / 1000, 'unixepoch')) AS day_utc,
  CASE
    WHEN es.event_type LIKE 'dev.cdevents.service.deployed.%'
      OR es.event_type LIKE 'dev.cdevents.service.upgraded.%'
      OR es.event_type LIKE 'dev.cdevents.service.published.%' THEN 1
    ELSE 0
  END AS deploy_success_count,
  CASE
    WHEN es.event_type LIKE 'dev.cdevents.service.removed.%' THEN 1
    ELSE 0
  END AS deploy_failure_count,
  CASE
    WHEN es.event_type LIKE 'dev.cdevents.service.rolledback.%' THEN 1
    ELSE 0
  END AS rollback_count
FROM event_store es
WHERE es.organization_id = sqlc.arg('organization_id')
  AND es.seq = sqlc.arg('seq')
  AND es.subject_type = 'service'
ON CONFLICT(organization_id, service_name, day_utc) DO UPDATE SET
  deploy_success_count = service_delivery_stats_daily.deploy_success_count + excluded.deploy_success_count,
  deploy_failure_count = service_delivery_stats_daily.deploy_failure_count + excluded.deploy_failure_count,
  rollback_count = service_delivery_stats_daily.rollback_count + excluded.rollback_count,
  updated_at = CURRENT_TIMESTAMP;

-- name: UpsertServiceChangeLinkFromEventSeq :exec
INSERT INTO service_change_links (
  organization_id,
  service_name,
  event_seq,
  event_ts_ms,
  chain_id,
  environment,
  artifact_id,
  pipeline_run_id,
  run_url,
  actor_name
)
SELECT
  es.organization_id,
  CASE
    WHEN instr(es.subject_id, '/') > 0 THEN substr(es.subject_id, instr(es.subject_id, '/') + 1)
    ELSE es.subject_id
  END AS service_name,
  es.seq,
  es.event_ts_ms,
  es.chain_id,
  COALESCE(NULLIF(json_extract(es.raw_event_json, '$.subject.content.environment.id'), ''), 'unknown') AS environment,
  COALESCE(json_extract(es.raw_event_json, '$.subject.content.artifactId'), '') AS artifact_id,
  COALESCE(json_extract(es.raw_event_json, '$.subject.content.pipeline.runId'), '') AS pipeline_run_id,
  COALESCE(json_extract(es.raw_event_json, '$.subject.content.pipeline.url'), '') AS run_url,
  COALESCE(json_extract(es.raw_event_json, '$.subject.content.actor.name'), '') AS actor_name
FROM event_store es
WHERE es.organization_id = sqlc.arg('organization_id')
  AND es.seq = sqlc.arg('seq')
  AND es.subject_type = 'service'
ON CONFLICT(organization_id, service_name, event_seq) DO NOTHING;

-- name: UpsertServiceCurrentStateByService :exec
INSERT INTO service_current_state (
  organization_id,
  service_name,
  latest_event_seq,
  latest_event_type,
  latest_event_ts_ms,
  latest_status,
  latest_artifact_id,
  latest_environment,
  drift_count,
  failed_streak
)
SELECT
  sqlc.arg('organization_id'),
  sqlc.arg('service_name'),
  ses.latest_event_seq,
  ses.latest_event_type,
  ses.latest_event_ts_ms,
  ses.latest_status,
  ses.latest_artifact_id,
  ses.environment,
  COALESCE((
    SELECT COUNT(DISTINCT NULLIF(se2.latest_artifact_id, ''))
    FROM service_env_state se2
    WHERE se2.organization_id = sqlc.arg('organization_id')
      AND se2.service_name = sqlc.arg('service_name')
  ), 0),
  COALESCE((
    SELECT SUM(CASE WHEN se3.latest_status IN ('warning', 'out-of-sync') THEN 1 ELSE 0 END)
    FROM service_env_state se3
    WHERE se3.organization_id = sqlc.arg('organization_id')
      AND se3.service_name = sqlc.arg('service_name')
  ), 0)
FROM service_env_state ses
WHERE ses.organization_id = sqlc.arg('organization_id')
  AND ses.service_name = sqlc.arg('service_name')
ORDER BY ses.latest_event_ts_ms DESC, ses.latest_event_seq DESC
LIMIT 1
ON CONFLICT(organization_id, service_name) DO UPDATE SET
  latest_event_seq = excluded.latest_event_seq,
  latest_event_type = excluded.latest_event_type,
  latest_event_ts_ms = excluded.latest_event_ts_ms,
  latest_status = excluded.latest_status,
  latest_artifact_id = excluded.latest_artifact_id,
  latest_environment = excluded.latest_environment,
  drift_count = excluded.drift_count,
  failed_streak = excluded.failed_streak,
  updated_at = CURRENT_TIMESTAMP;

-- name: ListServiceInstancesFromEvents :many
WITH service_events AS (
  SELECT
    es.seq,
    es.event_type,
    es.event_timestamp,
    es.event_ts_ms,
    CASE
      WHEN instr(es.subject_id, '/') > 0 THEN substr(es.subject_id, instr(es.subject_id, '/') + 1)
      ELSE es.subject_id
    END AS service_name,
    COALESCE(NULLIF(json_extract(es.raw_event_json, '$.subject.content.environment.id'), ''), 'unknown') AS environment,
    COALESCE(json_extract(es.raw_event_json, '$.subject.content.artifactId'), '') AS artifact_id,
    CASE
      WHEN es.event_type LIKE 'dev.cdevents.service.deployed.%' THEN 'synced'
      WHEN es.event_type LIKE 'dev.cdevents.service.upgraded.%' THEN 'synced'
      WHEN es.event_type LIKE 'dev.cdevents.service.published.%' THEN 'synced'
      WHEN es.event_type LIKE 'dev.cdevents.service.rolledback.%' THEN 'warning'
      WHEN es.event_type LIKE 'dev.cdevents.service.removed.%' THEN 'out-of-sync'
      ELSE 'unknown'
    END AS status
  FROM event_store es
  WHERE es.subject_type = 'service'
    AND es.organization_id = sqlc.arg('organization_id')
), ranked AS (
  SELECT
    service_name,
    environment,
    status,
    event_timestamp,
    artifact_id,
    row_number() OVER (
      PARTITION BY service_name
      ORDER BY event_ts_ms DESC, seq DESC
    ) AS rn
  FROM service_events
)
SELECT
  service_name,
  environment,
  status,
  event_timestamp AS last_deploy_at,
  artifact_id
FROM ranked
WHERE rn = 1
ORDER BY service_name, environment;

-- name: ListServiceInstancesByEnvFromEvents :many
WITH service_events AS (
  SELECT
    es.seq,
    es.event_type,
    es.event_timestamp,
    es.event_ts_ms,
    CASE
      WHEN instr(es.subject_id, '/') > 0 THEN substr(es.subject_id, instr(es.subject_id, '/') + 1)
      ELSE es.subject_id
    END AS service_name,
    COALESCE(NULLIF(json_extract(es.raw_event_json, '$.subject.content.environment.id'), ''), 'unknown') AS environment,
    COALESCE(json_extract(es.raw_event_json, '$.subject.content.artifactId'), '') AS artifact_id,
    CASE
      WHEN es.event_type LIKE 'dev.cdevents.service.deployed.%' THEN 'synced'
      WHEN es.event_type LIKE 'dev.cdevents.service.upgraded.%' THEN 'synced'
      WHEN es.event_type LIKE 'dev.cdevents.service.published.%' THEN 'synced'
      WHEN es.event_type LIKE 'dev.cdevents.service.rolledback.%' THEN 'warning'
      WHEN es.event_type LIKE 'dev.cdevents.service.removed.%' THEN 'out-of-sync'
      ELSE 'unknown'
    END AS status
  FROM event_store es
  WHERE es.subject_type = 'service'
    AND es.organization_id = sqlc.arg('organization_id')
    AND COALESCE(NULLIF(json_extract(es.raw_event_json, '$.subject.content.environment.id'), ''), 'unknown') = sqlc.arg('env')
), ranked AS (
  SELECT
    service_name,
    environment,
    status,
    event_timestamp,
    artifact_id,
    row_number() OVER (
      PARTITION BY service_name
      ORDER BY event_ts_ms DESC, seq DESC
    ) AS rn
  FROM service_events
)
SELECT
  service_name,
  environment,
  status,
  event_timestamp AS last_deploy_at,
  artifact_id
FROM ranked
WHERE rn = 1
ORDER BY service_name, environment;

-- name: ListDeploymentsFromEvents :many
SELECT
  es.event_timestamp AS deployed_at,
  CASE
    WHEN instr(es.subject_id, '/') > 0 THEN substr(es.subject_id, instr(es.subject_id, '/') + 1)
    ELSE es.subject_id
  END AS service,
  COALESCE(NULLIF(json_extract(es.raw_event_json, '$.subject.content.environment.id'), ''), 'unknown') AS environment,
  CASE
    WHEN es.event_type LIKE 'dev.cdevents.service.deployed.%' THEN 'success'
    WHEN es.event_type LIKE 'dev.cdevents.service.upgraded.%' THEN 'success'
    WHEN es.event_type LIKE 'dev.cdevents.service.published.%' THEN 'success'
    WHEN es.event_type LIKE 'dev.cdevents.service.rolledback.%' THEN 'error'
    WHEN es.event_type LIKE 'dev.cdevents.service.removed.%' THEN 'error'
    ELSE 'queued'
  END AS status
FROM event_store es
WHERE es.subject_type = 'service'
  AND es.organization_id = sqlc.arg('organization_id')
  AND (sqlc.arg('env') = '' OR sqlc.arg('env') = 'all' OR json_extract(es.raw_event_json, '$.subject.content.environment.id') = sqlc.arg('env'))
  AND (sqlc.arg('service') = '' OR sqlc.arg('service') = 'all' OR es.subject_id = sqlc.arg('service') OR substr(es.subject_id, instr(es.subject_id, '/') + 1) = sqlc.arg('service'))
ORDER BY es.event_ts_ms DESC, es.seq DESC;

-- name: GetServiceLatestFromEvents :one
SELECT
  CASE
    WHEN instr(es.subject_id, '/') > 0 THEN substr(es.subject_id, instr(es.subject_id, '/') + 1)
    ELSE es.subject_id
  END AS service_name,
  es.subject_id AS raw_subject_id,
  es.event_timestamp AS last_deploy_at,
  'cdevents' AS integration_type
FROM event_store es
WHERE es.subject_type = 'service'
  AND es.organization_id = sqlc.arg('organization_id')
  AND (es.subject_id = sqlc.arg('service') OR substr(es.subject_id, instr(es.subject_id, '/') + 1) = sqlc.arg('service'))
ORDER BY es.event_ts_ms DESC, es.seq DESC
LIMIT 1;

-- name: ListServiceEnvironmentsFromEvents :many
WITH service_events AS (
  SELECT
    es.seq,
    COALESCE(NULLIF(json_extract(es.raw_event_json, '$.subject.content.environment.id'), ''), 'unknown') AS environment,
    es.event_timestamp,
    es.event_ts_ms,
    COALESCE(json_extract(es.raw_event_json, '$.subject.content.artifactId'), '') AS artifact_id
  FROM event_store es
  WHERE es.subject_type = 'service'
    AND es.organization_id = sqlc.arg('organization_id')
    AND (es.subject_id = sqlc.arg('service') OR substr(es.subject_id, instr(es.subject_id, '/') + 1) = sqlc.arg('service'))
), ranked AS (
  SELECT
    environment,
    event_timestamp,
    artifact_id,
    row_number() OVER (
      PARTITION BY environment
      ORDER BY event_ts_ms DESC, seq DESC
    ) AS rn
  FROM service_events
)
SELECT
  environment AS name,
  event_timestamp AS released_at,
  artifact_id AS ref
FROM ranked
WHERE rn = 1
ORDER BY released_at DESC;

-- name: ListDeploymentHistoryByServiceFromEvents :many
SELECT
  es.event_timestamp AS deployed_at,
  COALESCE(json_extract(es.raw_event_json, '$.subject.content.artifactId'), '') AS release_ref,
  COALESCE(NULLIF(json_extract(es.raw_event_json, '$.subject.content.environment.id'), ''), 'unknown') AS environment
FROM event_store es
WHERE es.subject_type = 'service'
  AND es.organization_id = sqlc.arg('organization_id')
  AND (es.subject_id = sqlc.arg('service') OR substr(es.subject_id, instr(es.subject_id, '/') + 1) = sqlc.arg('service'))
ORDER BY es.event_ts_ms DESC, es.seq DESC
LIMIT sqlc.arg('limit');

-- name: ListServiceDependencies :many
SELECT depends_on_service_name
FROM service_dependencies
WHERE organization_id = sqlc.arg('organization_id')
  AND service_name = sqlc.arg('service_name')
ORDER BY depends_on_service_name;

-- name: ListServiceDependants :many
SELECT service_name
FROM service_dependencies
WHERE organization_id = sqlc.arg('organization_id')
  AND depends_on_service_name = sqlc.arg('service_name')
ORDER BY service_name;

-- name: UpsertServiceDependency :exec
INSERT INTO service_dependencies (organization_id, service_name, depends_on_service_name)
VALUES (sqlc.arg('organization_id'), sqlc.arg('service_name'), sqlc.arg('depends_on_service_name'))
ON CONFLICT(organization_id, service_name, depends_on_service_name) DO NOTHING;

-- name: DeleteServiceDependency :exec
DELETE FROM service_dependencies
WHERE organization_id = sqlc.arg('organization_id')
  AND service_name = sqlc.arg('service_name')
  AND depends_on_service_name = sqlc.arg('depends_on_service_name');

-- name: ListLegacyDeploymentsForBackfill :many
SELECT
  d.id,
  d.deployed_at,
  s.name AS service,
  e.name AS environment,
  d.status,
  d.release_ref
FROM deployments d
JOIN services s ON s.id = d.service_id
JOIN environments e ON e.id = d.environment_id
ORDER BY d.id;

-- name: GetOrganizationRenderVersion :one
SELECT COALESCE(MAX(version_value), 0) AS version
FROM (
  SELECT COALESCE(MAX(seq), 0) AS version_value
  FROM event_store
  WHERE event_store.organization_id = sqlc.arg('org_id')

  UNION ALL

  SELECT COALESCE(MAX(CAST(strftime('%s', updated_at) AS INTEGER)), 0) AS version_value
  FROM organizations
  WHERE id = sqlc.arg('org_id')

  UNION ALL

  SELECT COALESCE(MAX(CAST(strftime('%s', updated_at) AS INTEGER)), 0) AS version_value
  FROM service_metadata
  WHERE service_metadata.organization_id = sqlc.arg('org_id')

  UNION ALL

  SELECT COALESCE(MAX(CAST(strftime('%s', updated_at) AS INTEGER)), 0) AS version_value
  FROM organization_required_fields
  WHERE organization_required_fields.organization_id = sqlc.arg('org_id')

  UNION ALL

  SELECT COALESCE(MAX(CAST(strftime('%s', updated_at) AS INTEGER)), 0) AS version_value
  FROM organization_environment_priorities
  WHERE organization_environment_priorities.organization_id = sqlc.arg('org_id')

  UNION ALL

  SELECT COALESCE(MAX(CAST(strftime('%s', updated_at) AS INTEGER)), 0) AS version_value
  FROM organization_features
  WHERE organization_features.organization_id = sqlc.arg('org_id')

  UNION ALL

  SELECT COALESCE(MAX(CAST(strftime('%s', updated_at) AS INTEGER)), 0) AS version_value
  FROM organization_preferences
  WHERE organization_preferences.organization_id = sqlc.arg('org_id')

  UNION ALL

  SELECT COALESCE(MAX(CAST(strftime('%s', created_at) AS INTEGER)), 0) AS version_value
  FROM service_dependencies
  WHERE service_dependencies.organization_id = sqlc.arg('org_id')
);
