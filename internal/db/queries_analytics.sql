-- name: CountEventStore :one
SELECT COUNT(*)
FROM event_store;

-- name: CountEventStoreBySubjectType :one
SELECT COUNT(*)
FROM event_store
WHERE subject_type = sqlc.arg('subject_type');

-- name: CountEventStoreByOrganization :one
SELECT COUNT(*)
FROM event_store
WHERE organization_id = sqlc.arg('organization_id');

-- name: CountEventStoreByOrganizationSinceMs :one
SELECT COUNT(*)
FROM event_store
WHERE organization_id = sqlc.arg('organization_id')
  AND event_ts_ms >= sqlc.arg('since_ms');

-- name: ListEventStoreDailyVolume :many
SELECT
  date(datetime(event_ts_ms / 1000, 'unixepoch')) AS day,
  COUNT(*) AS total
FROM event_store
WHERE organization_id = sqlc.arg('organization_id')
  AND event_ts_ms >= sqlc.arg('since_ms')
GROUP BY day
ORDER BY day DESC
LIMIT sqlc.arg('limit');

-- name: GetServiceCurrentState :one
SELECT
  latest_status,
  latest_event_ts_ms,
  drift_count,
  failed_streak
FROM service_current_state
WHERE organization_id = sqlc.arg('organization_id')
  AND service_name = sqlc.arg('service_name')
LIMIT 1;

-- name: GetServiceDeliveryStats30d :one
SELECT
  COALESCE(SUM(deploy_success_count), 0) AS deploy_success_count,
  COALESCE(SUM(deploy_failure_count), 0) AS deploy_failure_count,
  COALESCE(SUM(rollback_count), 0) AS rollback_count
FROM service_delivery_stats_daily
WHERE organization_id = sqlc.arg('organization_id')
  AND service_name = sqlc.arg('service_name')
  AND day_utc >= date('now', '-30 day');

-- name: ListServiceChangeLinksRecent :many
SELECT
  event_ts_ms,
  chain_id,
  environment,
  artifact_id,
  pipeline_run_id,
  run_url,
  actor_name
FROM service_change_links
WHERE organization_id = sqlc.arg('organization_id')
  AND service_name = sqlc.arg('service_name')
ORDER BY event_ts_ms DESC
LIMIT sqlc.arg('limit');
