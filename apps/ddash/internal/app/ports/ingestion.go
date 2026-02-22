package ports

import (
	"context"
)

// IngestionStore is the minimal storage contract needed by webhook ingestion.
type IngestionStore interface {
	GetOrganizationByAuthToken(ctx context.Context, token string) (Organization, error)
	AppendEvent(ctx context.Context, event EventRecord) error
	AppendEvents(ctx context.Context, events []EventRecord) error
	Close() error
}

// EventRecord is one normalized event-store append request.
type EventRecord struct {
	OrganizationID int64
	EventID        string
	EventType      string
	EventSource    string
	EventTimestamp string
	EventTSMs      int64
	SubjectID      string
	SubjectSource  *string
	SubjectType    string
	ChainID        *string
	RawEventJSON   string
}

// IngestionStoreFactory creates request-scoped ingestion stores.
type IngestionStoreFactory interface {
	Open() (IngestionStore, error)
}
