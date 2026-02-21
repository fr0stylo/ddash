package db

import (
	"database/sql"
	"embed"
	"fmt"
	"net/url"
	"strings"

	"github.com/pressly/goose/v3"
	// SQLite driver.
	_ "modernc.org/sqlite"

	"github.com/fr0stylo/ddash/internal/db/queries"
)

//go:embed migrations/*.sql
var migrationsFS embed.FS

var driver = "sqlite"

// Database wraps sqlc queries with the shared connection.
type Database struct {
	*queries.Queries
	db      *sql.DB
	tracker *queryLatencyTracker
}

// New opens the SQLite database at the provided path.
func New(path string, openParams ...string) (*Database, error) {
	if path == "" {
		path = "data/default"
	}
	dsn := sqliteDSN(path, openParams...)
	db, err := sql.Open(driver, dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	goose.SetBaseFS(migrationsFS)

	if err := goose.SetDialect(driver); err != nil {
		return nil, fmt.Errorf("failed to set goose dialect: %w", err)
	}

	if err := goose.Up(db, "migrations"); err != nil {
		return nil, fmt.Errorf("failed to migrate database: %w", err)
	}

	tracker := newQueryLatencyTracker()
	wrapped := newInstrumentedDBTX(db, tracker)

	return &Database{db: db, Queries: queries.New(wrapped), tracker: tracker}, nil
}

func sqliteDSN(path string, openParams ...string) string {
	values := url.Values{}
	values.Set("_fk", "1")

	values.Add("_pragma", "foreign_keys(ON)")
	values.Add("_pragma", "journal_mode(WAL)")
	values.Add("_pragma", "synchronous(NORMAL)")
	values.Add("_pragma", "busy_timeout(5000)")
	values.Add("_pragma", "temp_store(MEMORY)")
	values.Add("_pragma", "cache_size(-200000)")
	values.Add("_pragma", "wal_autocheckpoint(1000)")
	values.Add("_pragma", "optimize")

	for _, param := range openParams {
		part := strings.TrimSpace(strings.TrimPrefix(param, "&"))
		if part == "" {
			continue
		}
		key, value, ok := strings.Cut(part, "=")
		if !ok {
			continue
		}
		values.Add(strings.TrimSpace(key), strings.TrimSpace(value))
	}

	return fmt.Sprintf("file:%s.sqlite?%s", path, values.Encode())
}

// Close closes the underlying database connection.
func (c *Database) Close() error {
	return c.db.Close()
}
