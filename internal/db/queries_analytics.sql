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

-- name: ListServiceLeadTimeSamplesFromEvents :many
WITH deploy_events AS (
  SELECT
    es.event_ts_ms AS deploy_ts_ms,
    date(datetime(es.event_ts_ms / 1000, 'unixepoch')) AS day_utc,
    CASE
      WHEN instr(es.subject_id, '/') > 0 THEN substr(es.subject_id, instr(es.subject_id, '/') + 1)
      ELSE es.subject_id
    END AS service_name
  FROM event_store es
  WHERE es.organization_id = sqlc.arg('organization_id')
    AND es.subject_type = 'service'
    AND es.event_type LIKE 'dev.cdevents.service.deployed.%'
    AND es.event_ts_ms >= sqlc.arg('since_ms')
), change_events AS (
  SELECT
    es.event_ts_ms AS change_ts_ms,
    CASE
      WHEN instr(json_extract(es.raw_event_json, '$.subject.content.artifactId'), 'pkg:generic/') = 1
       AND instr(substr(json_extract(es.raw_event_json, '$.subject.content.artifactId'), 13), '@') > 0
      THEN substr(
        json_extract(es.raw_event_json, '$.subject.content.artifactId'),
        13,
        instr(substr(json_extract(es.raw_event_json, '$.subject.content.artifactId'), 13), '@') - 1
      )
      ELSE ''
    END AS service_name
  FROM event_store es
  WHERE es.organization_id = sqlc.arg('organization_id')
    AND es.subject_type = 'change'
    AND es.event_ts_ms >= sqlc.arg('since_ms')
)
SELECT day_utc, service_name, lead_seconds
FROM (
  SELECT
    d.day_utc,
    d.service_name,
    (d.deploy_ts_ms - (
      SELECT MAX(c.change_ts_ms)
      FROM change_events c
      WHERE c.service_name = d.service_name
        AND c.change_ts_ms <= d.deploy_ts_ms
    )) / 1000 AS lead_seconds
  FROM deploy_events d
  WHERE d.service_name != ''
)
WHERE lead_seconds IS NOT NULL
  AND lead_seconds >= 0
ORDER BY day_utc DESC, service_name ASC, lead_seconds ASC;

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
