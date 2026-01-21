-- name: GetOrganizationByName :one
SELECT *
FROM organizations
WHERE name = ?
LIMIT 1;

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

-- name: CreateOrganization :one
INSERT INTO organizations (name, auth_token, webhook_secret, enabled)
VALUES (sqlc.arg('name'), sqlc.arg('auth_token'), sqlc.arg('webhook_secret'), sqlc.arg('enabled'))
RETURNING *;

-- name: ListOrganizationRequiredFields :many
SELECT id, organization_id, label, field_type, sort_order
FROM organization_required_fields
WHERE organization_id = ?
ORDER BY sort_order, id;

-- name: DeleteOrganizationRequiredFields :exec
DELETE FROM organization_required_fields
WHERE organization_id = ?;

-- name: CreateOrganizationRequiredField :one
INSERT INTO organization_required_fields (organization_id, label, field_type, sort_order)
VALUES (sqlc.arg('organization_id'), sqlc.arg('label'), sqlc.arg('field_type'), sqlc.arg('sort_order'))
RETURNING *;

-- name: UpdateOrganizationSecrets :exec
UPDATE organizations
SET auth_token = ?, webhook_secret = ?, enabled = ?, updated_at = CURRENT_TIMESTAMP
WHERE id = ?;

-- name: ListServiceInstances :many
SELECT
  si.id,
  si.service_id,
  s.name AS service_name,
  s.description,
  s.context,
  s.team,
  s.repo_url,
  s.logs_url,
  s.endpoint_url,
  e.name AS environment,
  si.status,
  si.last_deploy_at,
  si.deploy_duration_seconds,
  si.revision,
  si.commit_sha,
  si.commit_url,
  si.commit_index,
  si.action_label,
  si.action_kind,
  si.action_disabled
FROM service_instances si
JOIN services s ON s.id = si.service_id
JOIN environments e ON e.id = si.environment_id
ORDER BY s.name, e.name;

-- name: ListServiceInstancesByEnv :many
SELECT
  si.id,
  si.service_id,
  s.name AS service_name,
  s.description,
  s.context,
  s.team,
  s.repo_url,
  s.logs_url,
  s.endpoint_url,
  e.name AS environment,
  si.status,
  si.last_deploy_at,
  si.deploy_duration_seconds,
  si.revision,
  si.commit_sha,
  si.commit_url,
  si.commit_index,
  si.action_label,
  si.action_kind,
  si.action_disabled
FROM service_instances si
JOIN services s ON s.id = si.service_id
JOIN environments e ON e.id = si.environment_id
WHERE e.name = ?
ORDER BY s.name, e.name;

-- name: GetServiceByName :one
SELECT *
FROM services
WHERE name = ?
LIMIT 1;

-- name: CreateService :one
INSERT INTO services (name, integration_type)
VALUES (?, 'github')
RETURNING *;

-- name: GetEnvironmentByName :one
SELECT *
FROM environments
WHERE name = ?
LIMIT 1;

-- name: CreateEnvironment :one
INSERT INTO environments (name)
VALUES (?)
RETURNING *;

-- name: UpsertServiceInstance :one
INSERT INTO service_instances (
  service_id,
  environment_id,
  status,
  last_deploy_at,
  revision,
  commit_sha,
  commit_url,
  action_label,
  action_kind,
  action_disabled,
  updated_at
)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, CURRENT_TIMESTAMP)
ON CONFLICT(service_id, environment_id) DO UPDATE SET
  status = excluded.status,
  last_deploy_at = excluded.last_deploy_at,
  revision = excluded.revision,
  commit_sha = excluded.commit_sha,
  commit_url = excluded.commit_url,
  action_label = excluded.action_label,
  action_kind = excluded.action_kind,
  action_disabled = excluded.action_disabled,
  updated_at = CURRENT_TIMESTAMP
RETURNING *;

-- name: MarkServiceIntegrationType :exec
UPDATE services
SET integration_type = ?, updated_at = CURRENT_TIMESTAMP
WHERE id = ?;

-- name: ListServiceFields :many
SELECT *
FROM service_fields
WHERE service_id = ?
ORDER BY sort_order ASC, label ASC;

-- name: ListServiceEnvironments :many
SELECT
  e.name,
  r.released_at,
  r.ref,
  r.release_url
FROM releases r
JOIN environments e ON e.id = r.environment_id
WHERE r.service_id = ?
ORDER BY r.released_at DESC;

-- name: ListPendingCommitsNotInProd :many
WITH prod_release AS (
  SELECT id
  FROM releases
  WHERE service_id = sqlc.arg('service_id')
    AND environment_id = (SELECT id FROM environments WHERE name = 'production')
  ORDER BY released_at DESC
  LIMIT 1
)
SELECT
  c.id,
  c.sha,
  c.message,
  c.url,
  c.committed_at
FROM commits c
WHERE c.service_id = sqlc.arg('service_id')
  AND c.id NOT IN (
    SELECT rc.commit_id
    FROM release_commits rc
    JOIN prod_release pr ON pr.id = rc.release_id
  )
ORDER BY c.committed_at DESC
LIMIT sqlc.arg('limit');

-- name: ListDeploymentHistoryByService :many
SELECT
  d.deployed_at,
  d.commit_count,
  d.release_ref,
  d.release_url,
  e.name AS environment
FROM deployments d
JOIN environments e ON e.id = d.environment_id
WHERE d.service_id = ?
ORDER BY d.deployed_at DESC
LIMIT ?;

-- name: ListDeployments :many
SELECT
  d.deployed_at,
  s.name AS service,
  e.name AS environment,
  d.status,
  d.job_url
FROM deployments d
JOIN services s ON s.id = d.service_id
JOIN environments e ON e.id = d.environment_id
WHERE (sqlc.arg('env') = '' OR sqlc.arg('env') = 'all' OR e.name = sqlc.arg('env'))
  AND (sqlc.arg('service') = '' OR sqlc.arg('service') = 'all' OR s.name = sqlc.arg('service'))
ORDER BY d.deployed_at DESC;

-- name: UpdateServiceInstanceStatus :exec
UPDATE service_instances
SET status = ?,
    last_deploy_at = ?,
    updated_at = CURRENT_TIMESTAMP
WHERE id = ?;

-- name: CreateDeployment :one
INSERT INTO deployments (service_id, environment_id, deployed_at, status, job_url, release_ref, release_url, commit_count)
VALUES (sqlc.arg('service_id'), sqlc.arg('environment_id'), sqlc.arg('deployed_at'), sqlc.arg('status'),
        sqlc.arg('job_url'), sqlc.arg('release_ref'), sqlc.arg('release_url'), sqlc.arg('commit_count'))
RETURNING *;

-- name: CreateDeploymentSimple :one
INSERT INTO deployments (service_id, environment_id, deployed_at, status, release_ref)
VALUES (sqlc.arg('service_id'), sqlc.arg('environment_id'), sqlc.arg('deployed_at'), sqlc.arg('status'),
        sqlc.arg('release_ref'))
RETURNING *;
