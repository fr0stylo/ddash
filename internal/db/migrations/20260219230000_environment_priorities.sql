-- +goose Up
CREATE TABLE organization_environment_priorities
(
    id              INTEGER PRIMARY KEY AUTOINCREMENT,
    organization_id INTEGER NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    environment     TEXT NOT NULL,
    sort_order      INTEGER NOT NULL DEFAULT 0,
    created_at      DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at      DATETIME DEFAULT CURRENT_TIMESTAMP,
    UNIQUE (organization_id, environment)
);

CREATE INDEX idx_org_environment_priorities_org_order
    ON organization_environment_priorities (organization_id, sort_order, id);

-- +goose Down
DROP TABLE organization_environment_priorities;
