-- +goose Up
CREATE TABLE service_metadata
(
    id              INTEGER PRIMARY KEY AUTOINCREMENT,
    organization_id INTEGER NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    service_name    TEXT NOT NULL,
    label           TEXT NOT NULL,
    value           TEXT NOT NULL,
    created_at      DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at      DATETIME DEFAULT CURRENT_TIMESTAMP,
    UNIQUE (organization_id, service_name, label)
);

CREATE INDEX idx_service_metadata_org_service
    ON service_metadata (organization_id, service_name);

-- +goose Down
DROP TABLE service_metadata;
