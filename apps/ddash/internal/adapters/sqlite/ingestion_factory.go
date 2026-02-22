package sqlite

import (
	ingestionsqlite "github.com/fr0stylo/ddash/apps/ddash/internal/infrastructure/sqlite/ingestion"
	"github.com/fr0stylo/ddash/internal/db"
)

// IngestionStoreFactory is kept as a compatibility alias.
type IngestionStoreFactory = ingestionsqlite.StoreFactory

// NewIngestionStoreFactory creates a sqlite ingestion store factory backed by DB path.
func NewIngestionStoreFactory(dbPath string) *IngestionStoreFactory {
	return ingestionsqlite.NewStoreFactory(dbPath)
}

// NewSharedIngestionStoreFactory creates a factory backed by an existing shared DB handle.
func NewSharedIngestionStoreFactory(shared *db.Database) *IngestionStoreFactory {
	return ingestionsqlite.NewSharedStoreFactory(shared)
}
