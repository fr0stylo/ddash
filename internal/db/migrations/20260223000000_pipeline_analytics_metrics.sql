-- +goose Up
CREATE TABLE IF NOT EXISTS service_pipeline_stats_daily
(
    organization_id          INTEGER NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    service_name           TEXT NOT NULL,
    day_utc               TEXT NOT NULL,
    pipeline_started_count INTEGER NOT NULL DEFAULT 0,
    pipeline_succeeded_count INTEGER NOT NULL DEFAULT 0,
    pipeline_failed_count INTEGER NOT NULL DEFAULT 0,
    total_duration_seconds INTEGER NOT NULL DEFAULT 0,
    avg_duration_seconds REAL NOT NULL DEFAULT 0,
    updated_at            DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (organization_id, service_name, day_utc)
);

CREATE TABLE IF NOT EXISTS service_deployment_durations
(
    organization_id      INTEGER NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    service_name       TEXT NOT NULL,
    environment        TEXT NOT NULL,
    event_seq          INTEGER NOT NULL,
    event_ts_ms        INTEGER NOT NULL,
    duration_seconds   INTEGER NOT NULL,
    artifact_id        TEXT NOT NULL DEFAULT '',
    created_at         DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (organization_id, service_name, environment, event_seq)
);

CREATE TABLE IF NOT EXISTS service_environment_drift
(
    organization_id          INTEGER NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    service_name             TEXT NOT NULL,
    environment_from         TEXT NOT NULL,
    environment_to           TEXT NOT NULL,
    artifact_id_from         TEXT NOT NULL DEFAULT '',
    artifact_id_to           TEXT NOT NULL DEFAULT '',
    drift_detected_at        INTEGER NOT NULL,
    created_at               DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (organization_id, service_name, environment_from, environment_to, drift_detected_at)
);

CREATE TABLE IF NOT EXISTS service_redeployment_stats
(
    organization_id      INTEGER NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    service_name       TEXT NOT NULL,
    day_utc           TEXT NOT NULL,
    redeploy_count    INTEGER NOT NULL DEFAULT 0,
    updated_at        DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (organization_id, service_name, day_utc)
);

CREATE TABLE IF NOT EXISTS service_throughput_stats
(
    organization_id      INTEGER NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    service_name       TEXT NOT NULL,
    week_start        TEXT NOT NULL,
    changes_count     INTEGER NOT NULL DEFAULT 0,
    deployments_count INTEGER NOT NULL DEFAULT 0,
    updated_at        DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (organization_id, service_name, week_start)
);

CREATE TABLE IF NOT EXISTS service_incident_links
(
    organization_id      INTEGER NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    service_name       TEXT NOT NULL,
    incident_id       TEXT NOT NULL,
    incident_type     TEXT NOT NULL,
    linked_at         INTEGER NOT NULL,
    deployment_event_seq INTEGER,
    created_at        DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (organization_id, service_name, incident_id)
);

CREATE INDEX IF NOT EXISTS idx_pipeline_stats_org_day
ON service_pipeline_stats_daily(organization_id, day_utc DESC);

CREATE INDEX IF NOT EXISTS idx_deployment_durations_org_time
ON service_deployment_durations(organization_id, event_ts_ms DESC);

CREATE INDEX IF NOT EXISTS idx_environment_drift_org_service
ON service_environment_drift(organization_id, service_name, drift_detected_at DESC);

CREATE INDEX IF NOT EXISTS idx_redeployment_stats_org_day
ON service_redeployment_stats(organization_id, day_utc DESC);

CREATE INDEX IF NOT EXISTS idx_throughput_stats_org_week
ON service_throughput_stats(organization_id, week_start DESC);

-- +goose Down
DROP INDEX IF EXISTS idx_throughput_stats_org_week;
DROP INDEX IF EXISTS idx_redeployment_stats_org_day;
DROP INDEX IF EXISTS idx_environment_drift_org_service;
DROP INDEX IF EXISTS idx_deployment_durations_org_time;
DROP INDEX IF EXISTS idx_pipeline_stats_org_day;

DROP TABLE IF EXISTS service_incident_links;
DROP TABLE IF EXISTS service_throughput_stats;
DROP TABLE IF EXISTS service_redeployment_stats;
DROP TABLE IF EXISTS service_environment_drift;
DROP TABLE IF EXISTS service_deployment_durations;
DROP TABLE IF EXISTS service_pipeline_stats_daily;
