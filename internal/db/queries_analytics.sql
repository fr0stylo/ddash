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

-- Pipeline Execution Metrics
-- name: UpsertPipelineStatsFromEvent :exec
INSERT INTO service_pipeline_stats_daily (
  organization_id,
  service_name,
  day_utc,
  pipeline_started_count,
  pipeline_succeeded_count,
  pipeline_failed_count,
  total_duration_seconds,
  avg_duration_seconds
)
SELECT
  es.organization_id,
  COALESCE(
    NULLIF(json_extract(es.raw_event_json, '$.subject.content.service'), ''),
    CASE
      WHEN instr(es.subject_id, '/') > 0 THEN substr(es.subject_id, instr(es.subject_id, '/') + 1)
      ELSE es.subject_id
    END
  ) AS service_name,
  date(datetime(es.event_ts_ms / 1000, 'unixepoch')) AS day_utc,
  CASE WHEN es.event_type LIKE 'dev.cdevents.pipeline.run.started.%' THEN 1 ELSE 0 END,
  CASE WHEN es.event_type LIKE 'dev.cdevents.pipeline.run.succeeded.%' THEN 1 ELSE 0 END,
  CASE WHEN es.event_type LIKE 'dev.cdevents.pipeline.run.failed.%' THEN 1 ELSE 0 END,
  0,
  0
FROM event_store es
WHERE es.organization_id = sqlc.arg('organization_id')
  AND es.seq = sqlc.arg('seq')
  AND es.subject_type = 'pipeline'
ON CONFLICT(organization_id, service_name, day_utc) DO UPDATE SET
  pipeline_started_count = service_pipeline_stats_daily.pipeline_started_count + excluded.pipeline_started_count,
  pipeline_succeeded_count = service_pipeline_stats_daily.pipeline_succeeded_count + excluded.pipeline_succeeded_count,
  pipeline_failed_count = service_pipeline_stats_daily.pipeline_failed_count + excluded.pipeline_failed_count,
  updated_at = CURRENT_TIMESTAMP;

-- name: GetPipelineStats30d :one
SELECT
  COALESCE(SUM(pipeline_started_count), 0) AS pipeline_started_count,
  COALESCE(SUM(pipeline_succeeded_count), 0) AS pipeline_succeeded_count,
  COALESCE(SUM(pipeline_failed_count), 0) AS pipeline_failed_count,
  COALESCE(SUM(total_duration_seconds), 0) AS total_duration_seconds,
  CASE
    WHEN SUM(pipeline_started_count) > 0 THEN CAST(SUM(total_duration_seconds) AS REAL) / SUM(pipeline_started_count)
    ELSE 0
  END AS avg_duration_seconds
FROM service_pipeline_stats_daily
WHERE organization_id = sqlc.arg('organization_id')
  AND service_name = sqlc.arg('service_name')
  AND day_utc >= date('now', '-30 day');

-- Deployment Duration Metrics
-- name: InsertDeploymentDuration :exec
INSERT INTO service_deployment_durations (
  organization_id,
  service_name,
  environment,
  event_seq,
  event_ts_ms,
  duration_seconds,
  artifact_id
)
SELECT
  es.organization_id,
  CASE
    WHEN instr(es.subject_id, '/') > 0 THEN substr(es.subject_id, instr(es.subject_id, '/') + 1)
    ELSE es.subject_id
  END AS service_name,
  COALESCE(NULLIF(json_extract(es.raw_event_json, '$.subject.content.environment.id'), ''), 'unknown'),
  es.seq,
  es.event_ts_ms,
  COALESCE(json_extract(es.raw_event_json, '$.subject.content.durationSeconds'), 0),
  COALESCE(json_extract(es.raw_event_json, '$.subject.content.artifactId'), '')
FROM event_store es
WHERE es.organization_id = sqlc.arg('organization_id')
  AND es.seq = sqlc.arg('seq')
  AND es.subject_type = 'service'
  AND es.event_type LIKE 'dev.cdevents.service.deployed.%'
ON CONFLICT(organization_id, service_name, environment, event_seq) DO NOTHING;

-- name: GetDeploymentDurationStats :one
SELECT
  COUNT(*) AS sample_count,
  COALESCE(AVG(duration_seconds), 0) AS avg_duration_seconds,
  COALESCE(MIN(duration_seconds), 0) AS min_duration_seconds,
  COALESCE(MAX(duration_seconds), 0) AS max_duration_seconds,
  COALESCE(
    (SELECT duration_seconds FROM service_deployment_durations d2
     WHERE d2.organization_id = service_deployment_durations.organization_id
       AND d2.service_name = service_deployment_durations.service_name
       AND d2.environment = sqlc.arg('environment')
     ORDER BY event_ts_ms DESC LIMIT 1),
    0
  ) AS last_duration_seconds
FROM service_deployment_durations
WHERE service_deployment_durations.organization_id = sqlc.arg('organization_id')
  AND service_deployment_durations.service_name = sqlc.arg('service_name')
  AND service_deployment_durations.event_ts_ms >= sqlc.arg('since_ms');

-- name: ListDeploymentDurationsByEnvironment :many
SELECT
  environment,
  COUNT(*) AS sample_count,
  COALESCE(AVG(duration_seconds), 0) AS avg_duration_seconds,
  COALESCE(
    (SELECT duration_seconds FROM service_deployment_durations d2
     WHERE d2.organization_id = service_deployment_durations.organization_id
       AND d2.service_name = service_deployment_durations.service_name
       AND d2.environment = service_deployment_durations.environment
     ORDER BY event_ts_ms DESC LIMIT 1),
    0
  ) AS last_duration_seconds
FROM service_deployment_durations
WHERE service_deployment_durations.organization_id = sqlc.arg('organization_id')
  AND service_deployment_durations.service_name = sqlc.arg('service_name')
  AND service_deployment_durations.event_ts_ms >= sqlc.arg('since_ms')
GROUP BY service_deployment_durations.environment;

-- Environment Drift Detection
-- name: InsertEnvironmentDrift :exec
INSERT INTO service_environment_drift (
  organization_id,
  service_name,
  environment_from,
  environment_to,
  artifact_id_from,
  artifact_id_to,
  drift_detected_at
)
SELECT
  ses1.organization_id,
  ses1.service_name,
  ses1.environment,
  ses2.environment,
  ses1.latest_artifact_id,
  ses2.latest_artifact_id,
  strftime('%s', 'now') * 1000
FROM service_env_state ses1
JOIN service_env_state ses2 ON
  ses1.organization_id = ses2.organization_id
  AND ses1.service_name = ses2.service_name
  AND ses1.environment != ses2.environment
  AND ses1.latest_artifact_id != ses2.latest_artifact_id
WHERE ses1.organization_id = sqlc.arg('organization_id')
  AND ses1.service_name = sqlc.arg('service_name')
  AND ses1.latest_artifact_id != ''
  AND ses2.latest_artifact_id != ''
ON CONFLICT(organization_id, service_name, environment_from, environment_to, drift_detected_at) DO NOTHING;

-- name: GetEnvironmentDriftCount :one
SELECT COUNT(*) AS drift_count
FROM service_environment_drift
WHERE organization_id = sqlc.arg('organization_id')
  AND service_name = sqlc.arg('service_name')
  AND drift_detected_at >= sqlc.arg('since_ms');

-- name: ListEnvironmentDrifts :many
SELECT
  environment_from,
  environment_to,
  artifact_id_from,
  artifact_id_to,
  drift_detected_at
FROM service_environment_drift
WHERE organization_id = sqlc.arg('organization_id')
  AND service_name = sqlc.arg('service_name')
ORDER BY drift_detected_at DESC
LIMIT sqlc.arg('limit');

-- Re-deployment Rate Metrics
-- name: UpsertRedeploymentStats :exec
INSERT INTO service_redeployment_stats (
  organization_id,
  service_name,
  day_utc,
  redeploy_count
)
SELECT
  sqlc.arg('organization_id') AS organization_id,
  sqlc.arg('service_name') AS service_name,
  sqlc.arg('day_utc') AS day_utc,
  CASE WHEN sqlc.arg('same_artifact') = '1' THEN 1 ELSE 0 END AS redeploy_count
ON CONFLICT(organization_id, service_name, day_utc) DO UPDATE SET
  redeploy_count = service_redeployment_stats.redeploy_count + excluded.redeploy_count,
  updated_at = CURRENT_TIMESTAMP;

-- name: GetRedeploymentRate30d :one
SELECT
  COALESCE(SUM(redeploy_count), 0) AS redeploy_count,
  (SELECT COUNT(*) FROM service_delivery_stats_daily d
   WHERE d.organization_id = sqlc.arg('organization_id')
     AND d.service_name = sqlc.arg('service_name')
     AND d.day_utc >= date('now', '-30 day')
     AND d.deploy_success_count > 0) AS deploy_days
FROM service_redeployment_stats
WHERE organization_id = sqlc.arg('organization_id')
  AND service_name = sqlc.arg('service_name')
  AND day_utc >= date('now', '-30 day');

-- Throughput Metrics (Changes per week)
-- name: UpsertThroughputStats :exec
INSERT INTO service_throughput_stats (
  organization_id,
  service_name,
  week_start,
  changes_count,
  deployments_count
)
SELECT
  es.organization_id,
  CASE
    WHEN instr(es.subject_id, '/') > 0 THEN substr(es.subject_id, instr(es.subject_id, '/') + 1)
    ELSE es.subject_id
  END AS service_name,
  date(datetime((es.event_ts_ms / 1000) - (strftime('%w', datetime(es.event_ts_ms / 1000, 'unixepoch')) * 86400), 'unixepoch')) AS week_start,
  CASE WHEN es.subject_type = 'change' THEN 1 ELSE 0 END,
  CASE WHEN es.event_type LIKE 'dev.cdevents.service.deployed.%' THEN 1 ELSE 0 END
FROM event_store es
WHERE es.organization_id = sqlc.arg('organization_id')
  AND es.seq = sqlc.arg('seq')
ON CONFLICT(organization_id, service_name, week_start) DO UPDATE SET
  changes_count = service_throughput_stats.changes_count + excluded.changes_count,
  deployments_count = service_throughput_stats.deployments_count + excluded.deployments_count,
  updated_at = CURRENT_TIMESTAMP;

-- name: GetThroughputStats :one
SELECT
  COALESCE(SUM(changes_count), 0) AS changes_count,
  COALESCE(SUM(deployments_count), 0) AS deployments_count,
  CASE
    WHEN COUNT(*) > 0 THEN CAST(SUM(deployments_count) AS REAL) / COUNT(*)
    ELSE 0
  END AS avg_deployments_per_week
FROM service_throughput_stats
WHERE organization_id = sqlc.arg('organization_id')
  AND service_name = sqlc.arg('service_name')
  AND week_start >= date('now', '-12 week');

-- name: ListWeeklyThroughput :many
SELECT
  week_start,
  changes_count,
  deployments_count
FROM service_throughput_stats
WHERE organization_id = sqlc.arg('organization_id')
  AND service_name = sqlc.arg('service_name')
ORDER BY week_start DESC
LIMIT sqlc.arg('limit');

-- Incident Links (for MTTR calculation)
-- name: InsertIncidentLink :exec
INSERT INTO service_incident_links (
  organization_id,
  service_name,
  incident_id,
  incident_type,
  linked_at,
  deployment_event_seq
)
VALUES (
  sqlc.arg('organization_id'),
  sqlc.arg('service_name'),
  sqlc.arg('incident_id'),
  sqlc.arg('incident_type'),
  sqlc.arg('linked_at'),
  sqlc.arg('deployment_event_seq')
)
ON CONFLICT(organization_id, service_name, incident_id) DO NOTHING;

-- name: GetMTTR :one
WITH incident_resolutions AS (
  SELECT
    il.service_name,
    il.incident_id,
    il.linked_at AS incident_time_ms,
    (SELECT es.event_ts_ms FROM event_store es
     WHERE es.organization_id = il.organization_id
       AND es.subject_id LIKE '%' || il.service_name || '%'
       AND es.event_type LIKE 'dev.cdevents.service.deployed.%'
       AND es.event_ts_ms > il.linked_at
     ORDER BY es.event_ts_ms ASC LIMIT 1) AS resolved_at_ms
  FROM service_incident_links il
  WHERE il.organization_id = sqlc.arg('organization_id')
    AND il.linked_at >= sqlc.arg('since_ms')
)
SELECT
  COUNT(*) AS incident_count,
  COALESCE(AVG(resolved_at_ms - incident_time_ms), 0) / 1000 AS mttr_seconds,
  COALESCE(MIN(resolved_at_ms - incident_time_ms), 0) / 1000 AS mttd_seconds,
  COALESCE(MAX(resolved_at_ms - incident_time_ms), 0) / 1000 AS mtte_seconds
FROM incident_resolutions
WHERE resolved_at_ms IS NOT NULL;

-- name: ListIncidentLinks :many
SELECT
  incident_id,
  incident_type,
  linked_at,
  deployment_event_seq
FROM service_incident_links
WHERE organization_id = sqlc.arg('organization_id')
  AND service_name = sqlc.arg('service_name')
ORDER BY linked_at DESC
LIMIT sqlc.arg('limit');

-- Artifact Age (Time since last deployment per environment)
-- name: GetArtifactAgeByEnvironment :many
SELECT
  environment,
  latest_artifact_id,
  (strftime('%s', 'now') * 1000 - latest_event_ts_ms) / 1000 AS age_seconds,
  latest_event_ts_ms
FROM service_env_state
WHERE organization_id = sqlc.arg('organization_id')
  AND service_name = sqlc.arg('service_name')
  AND latest_artifact_id != '';

-- Comprehensive Delivery Metrics Summary
-- name: GetComprehensiveDeliveryMetrics :one
SELECT
  (SELECT COALESCE(AVG(lead_seconds), 0)
   FROM (
     SELECT (d.deploy_ts_ms - (
       SELECT MAX(c.change_ts_ms)
       FROM (
         SELECT es.event_ts_ms AS change_ts_ms,
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
       ) c
       WHERE c.service_name = d.service_name
         AND c.change_ts_ms <= d.deploy_ts_ms
     )) / 1000 AS lead_seconds
     FROM (
       SELECT
         es.event_ts_ms AS deploy_ts_ms,
         CASE
           WHEN instr(es.subject_id, '/') > 0 THEN substr(es.subject_id, instr(es.subject_id, '/') + 1)
           ELSE es.subject_id
         END AS service_name
       FROM event_store es
       WHERE es.organization_id = sqlc.arg('organization_id')
         AND es.subject_type = 'service'
         AND es.event_type LIKE 'dev.cdevents.service.deployed.%'
         AND es.event_ts_ms >= sqlc.arg('since_ms')
     ) d
     WHERE d.service_name != ''
   )) AS lead_time_seconds,

  (SELECT COALESCE(SUM(deploy_success_count), 0)
   FROM service_delivery_stats_daily
   WHERE organization_id = sqlc.arg('organization_id')
     AND day_utc >= date('now', '-30 day')) AS deployment_frequency_30d,

  (SELECT CASE
     WHEN (SELECT SUM(deploy_success_count + deploy_failure_count) FROM service_delivery_stats_daily
           WHERE organization_id = sqlc.arg('organization_id')
             AND day_utc >= date('now', '-30 day')) > 0
     THEN CAST((SELECT SUM(deploy_failure_count + rollback_count) FROM service_delivery_stats_daily
                WHERE organization_id = sqlc.arg('organization_id')
                  AND day_utc >= date('now', '-30 day')) AS REAL) /
          (SELECT SUM(deploy_success_count + deploy_failure_count) FROM service_delivery_stats_daily
           WHERE organization_id = sqlc.arg('organization_id')
             AND day_utc >= date('now', '-30 day'))
     ELSE 0
   END) AS change_failure_rate,

  (SELECT COALESCE(AVG(duration_seconds), 0)
   FROM service_deployment_durations
   WHERE organization_id = sqlc.arg('organization_id')
     AND event_ts_ms >= sqlc.arg('since_ms')) AS avg_deployment_duration_seconds,

  (SELECT COALESCE(SUM(pipeline_succeeded_count), 0)
   FROM service_pipeline_stats_daily
   WHERE organization_id = sqlc.arg('organization_id')
     AND day_utc >= date('now', '-30 day')) AS pipeline_success_count_30d,

  (SELECT COALESCE(SUM(pipeline_failed_count), 0)
   FROM service_pipeline_stats_daily
   WHERE organization_id = sqlc.arg('organization_id')
     AND day_utc >= date('now', '-30 day')) AS pipeline_failure_count_30d,

  (SELECT COUNT(DISTINCT day_utc)
   FROM service_delivery_stats_daily
   WHERE organization_id = sqlc.arg('organization_id')
     AND day_utc >= date('now', '-30 day')
     AND deploy_success_count > 0) AS active_deploy_days_30d;
