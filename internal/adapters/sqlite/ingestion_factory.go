package sqlite

import (
	"github.com/fr0stylo/ddash/internal/app/ports"
	"github.com/fr0stylo/ddash/internal/db"
)

// IngestionStoreFactory opens sqlite-backed stores for webhook ingestion.
type IngestionStoreFactory struct {
	dbPath string
	shared *db.Database
}

// NewIngestionStoreFactory creates a sqlite ingestion store factory backed by DB path.
// Opened stores own and close their DB handle.
func NewIngestionStoreFactory(dbPath string) *IngestionStoreFactory {
	return &IngestionStoreFactory{dbPath: dbPath}
}

// NewSharedIngestionStoreFactory creates a factory backed by an existing shared DB handle.
// Opened stores do not close the shared handle.
func NewSharedIngestionStoreFactory(shared *db.Database) *IngestionStoreFactory {
	return &IngestionStoreFactory{shared: shared}
}

// Open creates a request-scoped sqlite ingestion store.
func (f *IngestionStoreFactory) Open() (ports.IngestionStore, error) {
	if f.shared != nil {
		return newIngestionStore(f.shared, nil), nil
	}
	database, err := db.New(f.dbPath)
	if err != nil {
		return nil, err
	}
	return newIngestionStore(database, database.Close), nil
}

var _ ports.IngestionStoreFactory = (*IngestionStoreFactory)(nil)
