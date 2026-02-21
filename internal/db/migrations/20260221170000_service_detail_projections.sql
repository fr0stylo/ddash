-- +goose Up
CREATE TABLE IF NOT EXISTS service_current_state
(
    organization_id      INTEGER NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    service_name         TEXT NOT NULL,
    latest_event_seq     INTEGER NOT NULL,
    latest_event_type    TEXT NOT NULL,
    latest_event_ts_ms   INTEGER NOT NULL,
    latest_status        TEXT NOT NULL,
    latest_artifact_id   TEXT NOT NULL DEFAULT '',
    latest_environment   TEXT NOT NULL DEFAULT 'unknown',
    drift_count          INTEGER NOT NULL DEFAULT 0,
    failed_streak        INTEGER NOT NULL DEFAULT 0,
    updated_at           DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (organization_id, service_name)
);

CREATE TABLE IF NOT EXISTS service_env_state
(
    organization_id      INTEGER NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    service_name         TEXT NOT NULL,
    environment          TEXT NOT NULL,
    latest_event_seq     INTEGER NOT NULL,
    latest_event_type    TEXT NOT NULL,
    latest_event_ts_ms   INTEGER NOT NULL,
    latest_status        TEXT NOT NULL,
    latest_artifact_id   TEXT NOT NULL DEFAULT '',
    updated_at           DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (organization_id, service_name, environment)
);

CREATE TABLE IF NOT EXISTS service_delivery_stats_daily
(
    organization_id      INTEGER NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    service_name         TEXT NOT NULL,
    day_utc              TEXT NOT NULL,
    deploy_success_count INTEGER NOT NULL DEFAULT 0,
    deploy_failure_count INTEGER NOT NULL DEFAULT 0,
    rollback_count       INTEGER NOT NULL DEFAULT 0,
    updated_at           DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (organization_id, service_name, day_utc)
);

CREATE TABLE IF NOT EXISTS service_change_links
(
    organization_id      INTEGER NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    service_name         TEXT NOT NULL,
    event_seq            INTEGER NOT NULL,
    event_ts_ms          INTEGER NOT NULL,
    chain_id             TEXT,
    environment          TEXT NOT NULL DEFAULT 'unknown',
    artifact_id          TEXT NOT NULL DEFAULT '',
    repo                 TEXT NOT NULL DEFAULT '',
    commit_sha           TEXT NOT NULL DEFAULT '',
    pr_number            TEXT NOT NULL DEFAULT '',
    pipeline_run_id      TEXT NOT NULL DEFAULT '',
    run_url              TEXT NOT NULL DEFAULT '',
    actor_name           TEXT NOT NULL DEFAULT '',
    created_at           DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (organization_id, service_name, event_seq)
);

CREATE INDEX IF NOT EXISTS idx_service_current_state_org_status
ON service_current_state(organization_id, latest_status, latest_event_ts_ms DESC);

CREATE INDEX IF NOT EXISTS idx_service_env_state_org_env
ON service_env_state(organization_id, environment, latest_event_ts_ms DESC);

CREATE INDEX IF NOT EXISTS idx_service_delivery_stats_daily_org_day
ON service_delivery_stats_daily(organization_id, day_utc DESC);

CREATE INDEX IF NOT EXISTS idx_service_change_links_org_service_time
ON service_change_links(organization_id, service_name, event_ts_ms DESC);

-- +goose Down
DROP INDEX IF EXISTS idx_service_change_links_org_service_time;
DROP INDEX IF EXISTS idx_service_delivery_stats_daily_org_day;
DROP INDEX IF EXISTS idx_service_env_state_org_env;
DROP INDEX IF EXISTS idx_service_current_state_org_status;

DROP TABLE IF EXISTS service_change_links;
DROP TABLE IF EXISTS service_delivery_stats_daily;
DROP TABLE IF EXISTS service_env_state;
DROP TABLE IF EXISTS service_current_state;
