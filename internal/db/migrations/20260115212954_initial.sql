-- +goose Up
CREATE TABLE organizations
(
    id             INTEGER PRIMARY KEY AUTOINCREMENT,
    name           TEXT NOT NULL UNIQUE,
    auth_token     TEXT NOT NULL UNIQUE,
    webhook_secret TEXT NOT NULL,
    enabled        INTEGER NOT NULL DEFAULT 1,
    created_at     DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at     DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE organization_required_fields
(
    id              INTEGER PRIMARY KEY AUTOINCREMENT,
    organization_id INTEGER NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    label           TEXT NOT NULL,
    field_type      TEXT NOT NULL,
    sort_order      INTEGER NOT NULL DEFAULT 0,
    created_at      DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at      DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE services
(
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    name             TEXT NOT NULL UNIQUE,
    integration_type TEXT NOT NULL DEFAULT 'github',
    description      TEXT,
    context          TEXT,
    team             TEXT,
    repo_url         TEXT,
    logs_url         TEXT,
    endpoint_url     TEXT,
    created_at       DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at       DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE environments
(
    id   INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT NOT NULL UNIQUE
);

CREATE TABLE service_instances
(
    id                     INTEGER PRIMARY KEY AUTOINCREMENT,
    service_id             INTEGER NOT NULL REFERENCES services(id) ON DELETE CASCADE,
    environment_id         INTEGER NOT NULL REFERENCES environments(id) ON DELETE CASCADE,
    status                 TEXT NOT NULL,
    last_deploy_at         TEXT,
    deploy_duration_seconds INTEGER,
    revision               TEXT,
    commit_sha             TEXT,
    commit_url             TEXT,
    commit_index           INTEGER,
    action_label           TEXT,
    action_kind            TEXT,
    action_disabled        INTEGER NOT NULL DEFAULT 0,
    created_at             DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at             DATETIME DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(service_id, environment_id)
);

CREATE TABLE deployments
(
    id             INTEGER PRIMARY KEY AUTOINCREMENT,
    service_id     INTEGER NOT NULL REFERENCES services(id) ON DELETE CASCADE,
    environment_id INTEGER NOT NULL REFERENCES environments(id) ON DELETE CASCADE,
    deployed_at    TEXT NOT NULL,
    status         TEXT NOT NULL,
    job_url        TEXT,
    release_ref    TEXT,
    release_url    TEXT,
    commit_count   INTEGER,
    created_at     DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE commits
(
    id           INTEGER PRIMARY KEY AUTOINCREMENT,
    service_id   INTEGER NOT NULL REFERENCES services(id) ON DELETE CASCADE,
    sha          TEXT NOT NULL,
    message      TEXT NOT NULL,
    url          TEXT,
    committed_at TEXT NOT NULL
);

CREATE TABLE releases
(
    id             INTEGER PRIMARY KEY AUTOINCREMENT,
    service_id     INTEGER NOT NULL REFERENCES services(id) ON DELETE CASCADE,
    environment_id INTEGER NOT NULL REFERENCES environments(id) ON DELETE CASCADE,
    ref            TEXT NOT NULL,
    released_at    TEXT NOT NULL,
    release_url    TEXT
);

CREATE TABLE release_commits
(
    release_id INTEGER NOT NULL REFERENCES releases(id) ON DELETE CASCADE,
    commit_id  INTEGER NOT NULL REFERENCES commits(id) ON DELETE CASCADE,
    PRIMARY KEY (release_id, commit_id)
);

CREATE TABLE service_fields
(
    id         INTEGER PRIMARY KEY AUTOINCREMENT,
    service_id INTEGER NOT NULL REFERENCES services(id) ON DELETE CASCADE,
    label      TEXT NOT NULL,
    value      TEXT NOT NULL,
    sort_order INTEGER NOT NULL DEFAULT 0
);

CREATE INDEX idx_service_instances_service ON service_instances(service_id);
CREATE INDEX idx_service_instances_env ON service_instances(environment_id);
CREATE INDEX idx_deployments_service ON deployments(service_id);
CREATE INDEX idx_deployments_env ON deployments(environment_id);
CREATE INDEX idx_commits_service ON commits(service_id);
CREATE INDEX idx_releases_service_env ON releases(service_id, environment_id);

-- +goose Down
DROP TABLE service_fields;
DROP TABLE organization_required_fields;
DROP TABLE release_commits;
DROP TABLE releases;
DROP TABLE commits;
DROP TABLE deployments;
DROP TABLE service_instances;
DROP TABLE environments;
DROP TABLE services;
DROP TABLE organizations;
