-- +goose Up
PRAGMA foreign_keys = OFF;

ALTER TABLE event_store RENAME TO event_store_old;

CREATE TABLE event_store
(
    seq             INTEGER PRIMARY KEY AUTOINCREMENT,
    organization_id INTEGER NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    event_id        TEXT NOT NULL,
    event_type      TEXT NOT NULL,
    event_source    TEXT NOT NULL,
    event_timestamp TEXT NOT NULL,
    subject_id      TEXT NOT NULL,
    subject_source  TEXT,
    subject_type    TEXT NOT NULL,
    chain_id        TEXT,
    raw_event_json  TEXT NOT NULL,
    ingested_at     DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE (organization_id, event_source, event_id)
);

INSERT INTO event_store (
    seq,
    organization_id,
    event_id,
    event_type,
    event_source,
    event_timestamp,
    subject_id,
    subject_source,
    subject_type,
    chain_id,
    raw_event_json,
    ingested_at
)
SELECT
    seq,
    organization_id,
    event_id,
    event_type,
    event_source,
    event_timestamp,
    subject_id,
    subject_source,
    subject_type,
    chain_id,
    raw_event_json,
    ingested_at
FROM event_store_old;

DROP TABLE event_store_old;

CREATE INDEX idx_event_store_org_seq ON event_store(organization_id, seq DESC);
CREATE INDEX idx_event_store_org_type_time ON event_store(organization_id, event_type, event_timestamp DESC);
CREATE INDEX idx_event_store_org_subject_time ON event_store(organization_id, subject_id, event_timestamp DESC);

PRAGMA foreign_keys = ON;

-- +goose Down
PRAGMA foreign_keys = OFF;

ALTER TABLE event_store RENAME TO event_store_new;

CREATE TABLE event_store
(
    seq             INTEGER PRIMARY KEY AUTOINCREMENT,
    organization_id INTEGER NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    event_id        TEXT NOT NULL,
    event_type      TEXT NOT NULL,
    event_source    TEXT NOT NULL,
    event_timestamp TEXT NOT NULL,
    subject_id      TEXT NOT NULL,
    subject_source  TEXT,
    subject_type    TEXT NOT NULL,
    chain_id        TEXT,
    raw_event_json  TEXT NOT NULL,
    ingested_at     DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE (event_source, event_id)
);

INSERT INTO event_store (
    seq,
    organization_id,
    event_id,
    event_type,
    event_source,
    event_timestamp,
    subject_id,
    subject_source,
    subject_type,
    chain_id,
    raw_event_json,
    ingested_at
)
SELECT
    seq,
    organization_id,
    event_id,
    event_type,
    event_source,
    event_timestamp,
    subject_id,
    subject_source,
    subject_type,
    chain_id,
    raw_event_json,
    ingested_at
FROM event_store_new;

DROP TABLE event_store_new;

CREATE INDEX idx_event_store_org_seq ON event_store(organization_id, seq DESC);
CREATE INDEX idx_event_store_org_type_time ON event_store(organization_id, event_type, event_timestamp DESC);
CREATE INDEX idx_event_store_org_subject_time ON event_store(organization_id, subject_id, event_timestamp DESC);

PRAGMA foreign_keys = ON;
