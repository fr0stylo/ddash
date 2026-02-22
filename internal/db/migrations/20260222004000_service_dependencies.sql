-- +goose Up
CREATE TABLE IF NOT EXISTS service_dependencies
(
    organization_id         INTEGER NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    service_name            TEXT NOT NULL,
    depends_on_service_name TEXT NOT NULL,
    created_at              DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (organization_id, service_name, depends_on_service_name),
    CHECK (length(trim(service_name)) > 0),
    CHECK (length(trim(depends_on_service_name)) > 0),
    CHECK (service_name <> depends_on_service_name)
);

CREATE INDEX IF NOT EXISTS idx_service_dependencies_org_depends_on
ON service_dependencies(organization_id, depends_on_service_name, service_name);

-- +goose Down
DROP INDEX IF EXISTS idx_service_dependencies_org_depends_on;
DROP TABLE IF EXISTS service_dependencies;
