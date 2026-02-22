-- +goose Up
CREATE TABLE IF NOT EXISTS github_setup_intents
(
    state               TEXT PRIMARY KEY,
    organization_id     INTEGER NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    organization_label  TEXT NOT NULL DEFAULT '',
    default_environment TEXT NOT NULL DEFAULT '',
    expires_at          DATETIME NOT NULL,
    created_at          DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_github_setup_intents_expires_at
ON github_setup_intents(expires_at);

-- +goose Down
DROP INDEX IF EXISTS idx_github_setup_intents_expires_at;
DROP TABLE IF EXISTS github_setup_intents;
