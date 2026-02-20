-- +goose Up
ALTER TABLE organizations ADD COLUMN join_code TEXT;

UPDATE organizations
SET join_code = lower(hex(randomblob(8)))
WHERE join_code IS NULL OR trim(join_code) = '';

CREATE UNIQUE INDEX idx_organizations_join_code ON organizations(join_code);

CREATE TABLE organization_join_requests
(
    id              INTEGER PRIMARY KEY AUTOINCREMENT,
    organization_id INTEGER NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    user_id         INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    request_code    TEXT NOT NULL,
    status          TEXT NOT NULL DEFAULT 'pending',
    reviewed_by     INTEGER REFERENCES users(id) ON DELETE SET NULL,
    reviewed_at     DATETIME,
    created_at      DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at      DATETIME DEFAULT CURRENT_TIMESTAMP,
    UNIQUE (organization_id, user_id),
    CHECK (status IN ('pending', 'approved', 'rejected'))
);

CREATE INDEX idx_org_join_requests_org_status
    ON organization_join_requests (organization_id, status);

CREATE INDEX idx_org_join_requests_user_status
    ON organization_join_requests (user_id, status);

-- +goose Down
DROP TABLE organization_join_requests;
DROP INDEX idx_organizations_join_code;
