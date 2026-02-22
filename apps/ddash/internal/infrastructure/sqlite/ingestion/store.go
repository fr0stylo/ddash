package ingestion

import (
	"context"
	"database/sql"

	"github.com/fr0stylo/ddash/apps/ddash/internal/app/ports"
	"github.com/fr0stylo/ddash/internal/db/queries"
)

type databaseContract interface {
	GetOrganizationByAuthToken(ctx context.Context, authToken string) (queries.Organization, error)
	AppendEventStore(ctx context.Context, params queries.AppendEventStoreParams) error
	AppendEventStoreBatch(ctx context.Context, params []queries.AppendEventStoreParams) error
}

type store struct {
	db      databaseContract
	closeFn func() error
}

func newStore(database databaseContract, closeFn func() error) *store {
	return &store{db: database, closeFn: closeFn}
}

func (s *store) GetOrganizationByAuthToken(ctx context.Context, token string) (ports.Organization, error) {
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

func (s *store) AppendEvent(ctx context.Context, event ports.EventRecord) error {
	params := toAppendEventParams(event)
	return s.db.AppendEventStore(ctx, params)
}

func (s *store) AppendEvents(ctx context.Context, events []ports.EventRecord) error {
	if len(events) == 0 {
		return nil
	}
	params := make([]queries.AppendEventStoreParams, 0, len(events))
	for _, event := range events {
		params = append(params, toAppendEventParams(event))
	}
	return s.db.AppendEventStoreBatch(ctx, params)
}

func toAppendEventParams(event ports.EventRecord) queries.AppendEventStoreParams {
	subjectSource := sql.NullString{}
	if event.SubjectSource != nil {
		subjectSource = sql.NullString{String: *event.SubjectSource, Valid: true}
	}

	chainID := sql.NullString{}
	if event.ChainID != nil {
		chainID = sql.NullString{String: *event.ChainID, Valid: true}
	}

	return queries.AppendEventStoreParams{
		OrganizationID: event.OrganizationID,
		EventID:        event.EventID,
		EventType:      event.EventType,
		EventSource:    event.EventSource,
		EventTimestamp: event.EventTimestamp,
		EventTsMs:      event.EventTSMs,
		SubjectID:      event.SubjectID,
		SubjectSource:  subjectSource,
		SubjectType:    event.SubjectType,
		ChainID:        chainID,
		RawEventJson:   event.RawEventJSON,
	}
}

func (s *store) Close() error {
	if s.closeFn == nil {
		return nil
	}
	return s.closeFn()
}

var _ ports.IngestionStore = (*store)(nil)
