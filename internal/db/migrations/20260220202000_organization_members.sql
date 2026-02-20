-- +goose Up
CREATE TABLE users
(
    id         INTEGER PRIMARY KEY AUTOINCREMENT,
    github_id  TEXT,
    email      TEXT NOT NULL,
    nickname   TEXT NOT NULL,
    name       TEXT,
    avatar_url TEXT,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(email),
    UNIQUE(nickname)
);

CREATE TABLE organization_members
(
    id              INTEGER PRIMARY KEY AUTOINCREMENT,
    organization_id INTEGER NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    user_id         INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    role            TEXT NOT NULL,
    created_at      DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at      DATETIME DEFAULT CURRENT_TIMESTAMP,
    UNIQUE (organization_id, user_id),
    CHECK (role IN ('owner', 'admin', 'member'))
);

CREATE INDEX idx_org_members_org
    ON organization_members (organization_id);

CREATE INDEX idx_org_members_user
    ON organization_members (user_id);

-- +goose Down
DROP TABLE organization_members;
DROP TABLE users;
