-- +goose Up
CREATE TABLE organization_preferences
(
    id              INTEGER PRIMARY KEY AUTOINCREMENT,
    organization_id INTEGER NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    preference_key  TEXT NOT NULL,
    preference_value TEXT NOT NULL,
    created_at      DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at      DATETIME DEFAULT CURRENT_TIMESTAMP,
    UNIQUE (organization_id, preference_key)
);

CREATE INDEX idx_org_preferences_org
    ON organization_preferences (organization_id);

-- +goose Down
DROP TABLE organization_preferences;
