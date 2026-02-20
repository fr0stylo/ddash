package sqlite

import (
	"context"
	"database/sql"

	"github.com/fr0stylo/ddash/internal/app/ports"
	"github.com/fr0stylo/ddash/internal/db/queries"
)

type ingestionDatabase interface {
	GetOrganizationByAuthToken(ctx context.Context, authToken string) (queries.Organization, error)
	AppendEventStore(ctx context.Context, params queries.AppendEventStoreParams) error
}

type ingestionStore struct {
	db      ingestionDatabase
	closeFn func() error
}

func newIngestionStore(database ingestionDatabase, closeFn func() error) *ingestionStore {
	return &ingestionStore{db: database, closeFn: closeFn}
}

func (s *ingestionStore) GetOrganizationByAuthToken(ctx context.Context, token string) (ports.Organization, error) {
	org, err := s.db.GetOrganizationByAuthToken(ctx, token)
	if err != nil {
		return ports.Organization{}, err
	}
	return ports.Organization{
		ID:            org.ID,
		Name:          org.Name,
		AuthToken:     org.AuthToken,
		WebhookSecret: org.WebhookSecret,
		Enabled:       org.Enabled != 0,
	}, nil
}

func (s *ingestionStore) AppendEvent(ctx context.Context, event ports.EventRecord) error {
	subjectSource := sql.NullString{}
	if event.SubjectSource != nil {
		subjectSource = sql.NullString{String: *event.SubjectSource, Valid: true}
	}

	chainID := sql.NullString{}
	if event.ChainID != nil {
		chainID = sql.NullString{String: *event.ChainID, Valid: true}
	}

	return s.db.AppendEventStore(ctx, queries.AppendEventStoreParams{
		OrganizationID: event.OrganizationID,
		EventID:        event.EventID,
		EventType:      event.EventType,
		EventSource:    event.EventSource,
		EventTimestamp: event.EventTimestamp,
		SubjectID:      event.SubjectID,
		SubjectSource:  subjectSource,
		SubjectType:    event.SubjectType,
		ChainID:        chainID,
		RawEventJson:   event.RawEventJSON,
	})
}

func (s *ingestionStore) Close() error {
	if s.closeFn == nil {
		return nil
	}
	return s.closeFn()
}

var _ ports.IngestionStore = (*ingestionStore)(nil)
