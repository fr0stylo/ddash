-- +goose Up
CREATE TABLE IF NOT EXISTS gitlab_project_mappings
(
    project_id          INTEGER PRIMARY KEY,
    organization_id     INTEGER NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    project_path        TEXT NOT NULL DEFAULT '',
    default_environment TEXT NOT NULL DEFAULT '',
    enabled             INTEGER NOT NULL DEFAULT 1,
    created_at          DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at          DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_gitlab_project_mappings_org
ON gitlab_project_mappings(organization_id, project_id);

-- +goose Down
DROP INDEX IF EXISTS idx_gitlab_project_mappings_org;
DROP TABLE IF EXISTS gitlab_project_mappings;
