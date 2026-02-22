-- +goose Up
CREATE TABLE IF NOT EXISTS github_installation_mappings
(
    installation_id     INTEGER PRIMARY KEY,
    organization_id     INTEGER NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    organization_label  TEXT NOT NULL DEFAULT '',
    default_environment TEXT NOT NULL DEFAULT '',
    enabled             INTEGER NOT NULL DEFAULT 1,
    created_at          DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at          DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_github_installation_mappings_org
ON github_installation_mappings(organization_id, installation_id);

-- +goose Down
DROP INDEX IF EXISTS idx_github_installation_mappings_org;
DROP TABLE IF EXISTS github_installation_mappings;
