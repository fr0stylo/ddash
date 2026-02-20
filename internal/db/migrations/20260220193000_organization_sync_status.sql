-- +goose Up
CREATE TABLE organization_features
(
    id              INTEGER PRIMARY KEY AUTOINCREMENT,
    organization_id INTEGER NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    feature_key     TEXT NOT NULL,
    is_enabled      INTEGER NOT NULL DEFAULT 0,
    created_at      DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at      DATETIME DEFAULT CURRENT_TIMESTAMP,
    UNIQUE (organization_id, feature_key)
);

CREATE INDEX idx_org_features_org
    ON organization_features (organization_id);

-- +goose Down
DROP TABLE organization_features;
