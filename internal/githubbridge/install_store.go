package githubbridge

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	_ "modernc.org/sqlite"
)

type InstallationMapping struct {
	InstallationID     int64
	OrganizationID     int64
	OrganizationLabel  string
	DDashEndpoint      string
	DDashAuthToken     string
	DDashWebhookSecret string
	DefaultEnvironment string
	Enabled            bool
}

type SetupIntent struct {
	State              string
	OrganizationID     int64
	OrganizationLabel  string
	DDashEndpoint      string
	DDashAuthToken     string
	DDashWebhookSecret string
	DefaultEnvironment string
	ExpiresAt          time.Time
}

type InstallStore struct {
	db *sql.DB
}

func OpenInstallStore(path string) (*InstallStore, error) {
	path = strings.TrimSpace(path)
	if path == "" {
		path = "data/githubapp-ingestor"
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, fmt.Errorf("create store dir: %w", err)
	}
	db, err := sql.Open("sqlite", fmt.Sprintf("file:%s.sqlite?_fk=1", path))
	if err != nil {
		return nil, err
	}
	store := &InstallStore{db: db}
	if err := store.migrate(); err != nil {
		_ = db.Close()
		return nil, err
	}
	return store, nil
}

func (s *InstallStore) Close() error {
	if s == nil || s.db == nil {
		return nil
	}
	return s.db.Close()
}

func (s *InstallStore) migrate() error {
	statements := []string{
		`CREATE TABLE IF NOT EXISTS github_installation_mappings (
			installation_id INTEGER PRIMARY KEY,
			organization_id INTEGER NOT NULL DEFAULT 0,
			organization_label TEXT NOT NULL DEFAULT '',
			ddash_endpoint TEXT NOT NULL,
			ddash_auth_token TEXT NOT NULL,
			ddash_webhook_secret TEXT NOT NULL,
			default_environment TEXT NOT NULL DEFAULT '',
			enabled INTEGER NOT NULL DEFAULT 1,
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS github_setup_intents (
			state TEXT PRIMARY KEY,
			organization_id INTEGER NOT NULL DEFAULT 0,
			organization_label TEXT NOT NULL DEFAULT '',
			ddash_endpoint TEXT NOT NULL,
			ddash_auth_token TEXT NOT NULL,
			ddash_webhook_secret TEXT NOT NULL,
			default_environment TEXT NOT NULL DEFAULT '',
			expires_at DATETIME NOT NULL,
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE INDEX IF NOT EXISTS idx_github_setup_intents_expires_at ON github_setup_intents(expires_at)`,
	}
	for _, statement := range statements {
		if _, err := s.db.Exec(statement); err != nil {
			return err
		}
	}
	_, _ = s.db.Exec(`DELETE FROM github_setup_intents WHERE expires_at <= CURRENT_TIMESTAMP`)
	_, _ = s.db.Exec(`ALTER TABLE github_installation_mappings ADD COLUMN organization_id INTEGER NOT NULL DEFAULT 0`)
	_, _ = s.db.Exec(`ALTER TABLE github_setup_intents ADD COLUMN organization_id INTEGER NOT NULL DEFAULT 0`)
	return nil
}

func (s *InstallStore) CreateSetupIntent(intent SetupIntent) error {
	_, err := s.db.Exec(`
		INSERT INTO github_setup_intents (
			state, organization_id, organization_label, ddash_endpoint, ddash_auth_token,
			ddash_webhook_secret, default_environment, expires_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`, intent.State, intent.OrganizationID, intent.OrganizationLabel, intent.DDashEndpoint, intent.DDashAuthToken, intent.DDashWebhookSecret, intent.DefaultEnvironment, intent.ExpiresAt.UTC().Format(time.RFC3339))
	return err
}

func (s *InstallStore) GetSetupIntent(state string) (SetupIntent, error) {
	row := s.db.QueryRow(`
		SELECT state, organization_id, organization_label, ddash_endpoint, ddash_auth_token, ddash_webhook_secret, default_environment, expires_at
		FROM github_setup_intents
		WHERE state = ?
	`, strings.TrimSpace(state))
	var intent SetupIntent
	var expires string
	err := row.Scan(&intent.State, &intent.OrganizationID, &intent.OrganizationLabel, &intent.DDashEndpoint, &intent.DDashAuthToken, &intent.DDashWebhookSecret, &intent.DefaultEnvironment, &expires)
	if err != nil {
		return SetupIntent{}, err
	}
	intent.ExpiresAt, err = time.Parse(time.RFC3339, expires)
	if err != nil {
		return SetupIntent{}, err
	}
	return intent, nil
}

func (s *InstallStore) DeleteSetupIntent(state string) error {
	_, err := s.db.Exec(`DELETE FROM github_setup_intents WHERE state = ?`, strings.TrimSpace(state))
	return err
}

func (s *InstallStore) UpsertInstallationMapping(mapping InstallationMapping) error {
	enabled := 0
	if mapping.Enabled {
		enabled = 1
	}
	_, err := s.db.Exec(`
		INSERT INTO github_installation_mappings (
			installation_id, organization_id, organization_label, ddash_endpoint, ddash_auth_token,
			ddash_webhook_secret, default_environment, enabled
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(installation_id) DO UPDATE SET
			organization_id = excluded.organization_id,
			organization_label = excluded.organization_label,
			ddash_endpoint = excluded.ddash_endpoint,
			ddash_auth_token = excluded.ddash_auth_token,
			ddash_webhook_secret = excluded.ddash_webhook_secret,
			default_environment = excluded.default_environment,
			enabled = excluded.enabled,
			updated_at = CURRENT_TIMESTAMP
	`, mapping.InstallationID, mapping.OrganizationID, mapping.OrganizationLabel, mapping.DDashEndpoint, mapping.DDashAuthToken, mapping.DDashWebhookSecret, mapping.DefaultEnvironment, enabled)
	return err
}

func (s *InstallStore) GetInstallationMapping(installationID int64) (InstallationMapping, error) {
	row := s.db.QueryRow(`
		SELECT installation_id, organization_id, organization_label, ddash_endpoint, ddash_auth_token, ddash_webhook_secret, default_environment, enabled
		FROM github_installation_mappings
		WHERE installation_id = ?
	`, installationID)
	var mapping InstallationMapping
	var enabled int
	err := row.Scan(&mapping.InstallationID, &mapping.OrganizationID, &mapping.OrganizationLabel, &mapping.DDashEndpoint, &mapping.DDashAuthToken, &mapping.DDashWebhookSecret, &mapping.DefaultEnvironment, &enabled)
	if err != nil {
		return InstallationMapping{}, err
	}
	mapping.Enabled = enabled == 1
	return mapping, nil
}

func (s *InstallStore) ListInstallationMappings(organizationID int64) ([]InstallationMapping, error) {
	query := `
		SELECT installation_id, organization_id, organization_label, ddash_endpoint, ddash_auth_token, ddash_webhook_secret, default_environment, enabled
		FROM github_installation_mappings
	`
	args := make([]any, 0, 1)
	if organizationID > 0 {
		query += " WHERE organization_id = ?"
		args = append(args, organizationID)
	}
	query += " ORDER BY installation_id ASC"

	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]InstallationMapping, 0)
	for rows.Next() {
		var item InstallationMapping
		var enabled int
		if err := rows.Scan(&item.InstallationID, &item.OrganizationID, &item.OrganizationLabel, &item.DDashEndpoint, &item.DDashAuthToken, &item.DDashWebhookSecret, &item.DefaultEnvironment, &enabled); err != nil {
			return nil, err
		}
		item.Enabled = enabled == 1
		out = append(out, item)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return out, nil
}

func (s *InstallStore) DeleteInstallationMapping(installationID, organizationID int64) error {
	if installationID <= 0 {
		return sql.ErrNoRows
	}
	if organizationID > 0 {
		result, err := s.db.Exec(`DELETE FROM github_installation_mappings WHERE installation_id = ? AND organization_id = ?`, installationID, organizationID)
		if err != nil {
			return err
		}
		affected, err := result.RowsAffected()
		if err != nil {
			return err
		}
		if affected == 0 {
			return sql.ErrNoRows
		}
		return nil
	}
	_, err := s.db.Exec(`DELETE FROM github_installation_mappings WHERE installation_id = ?`, installationID)
	return err
}
