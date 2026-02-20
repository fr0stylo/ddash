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

-- name: CreateOrganization :one
INSERT INTO organizations (name, auth_token, webhook_secret, enabled)
VALUES (sqlc.arg('name'), sqlc.arg('auth_token'), sqlc.arg('webhook_secret'), sqlc.arg('enabled'))
RETURNING *;

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

-- name: AppendEventStore :exec
INSERT INTO event_store (
  organization_id,
  event_id,
  event_type,
  event_source,
  event_timestamp,
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
  sqlc.arg('subject_id'),
  sqlc.narg('subject_source'),
  sqlc.arg('subject_type'),
  sqlc.narg('chain_id'),
  sqlc.arg('raw_event_json')
)
ON CONFLICT(organization_id, event_source, event_id) DO NOTHING;

-- name: CountEventStore :one
SELECT COUNT(*)
FROM event_store;

-- name: CountEventStoreBySubjectType :one
SELECT COUNT(*)
FROM event_store
WHERE subject_type = sqlc.arg('subject_type');

-- name: ListServiceInstancesFromEvents :many
WITH service_events AS (
  SELECT
    es.seq,
    es.event_type,
    es.event_timestamp,
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
      ORDER BY event_timestamp DESC, seq DESC
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
      ORDER BY event_timestamp DESC, seq DESC
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
ORDER BY es.event_timestamp DESC, es.seq DESC;

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
ORDER BY es.event_timestamp DESC, es.seq DESC
LIMIT 1;

-- name: ListServiceEnvironmentsFromEvents :many
WITH service_events AS (
  SELECT
    es.seq,
    COALESCE(NULLIF(json_extract(es.raw_event_json, '$.subject.content.environment.id'), ''), 'unknown') AS environment,
    es.event_timestamp,
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
      ORDER BY event_timestamp DESC, seq DESC
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
ORDER BY es.event_timestamp DESC, es.seq DESC
LIMIT sqlc.arg('limit');

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
