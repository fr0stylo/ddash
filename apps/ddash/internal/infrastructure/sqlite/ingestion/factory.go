package ingestion

import (
	"github.com/fr0stylo/ddash/apps/ddash/internal/app/ports"
	"github.com/fr0stylo/ddash/internal/db"
)

// StoreFactory opens sqlite-backed stores for webhook ingestion.
type StoreFactory struct {
	dbPath string
	shared *db.Database
}

// NewStoreFactory creates a sqlite ingestion store factory backed by DB path.
func NewStoreFactory(dbPath string) *StoreFactory {
	return &StoreFactory{dbPath: dbPath}
}

// NewSharedStoreFactory creates a factory backed by an existing shared DB handle.
func NewSharedStoreFactory(shared *db.Database) *StoreFactory {
	return &StoreFactory{shared: shared}
}

// Open creates a request-scoped sqlite ingestion store.
func (f *StoreFactory) Open() (ports.IngestionStore, error) {
	if f.shared != nil {
		return newStore(f.shared, nil), nil
	}
	database, err := db.New(f.dbPath)
	if err != nil {
		return nil, err
	}
	return newStore(database, database.Close), nil
}

var _ ports.IngestionStoreFactory = (*StoreFactory)(nil)
