package sqlite

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/fr0stylo/ddash/internal/app/ports"
	"github.com/fr0stylo/ddash/internal/db"
	"github.com/fr0stylo/ddash/internal/db/queries"
)

func TestIngestionStoreRoundTrip(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	dbPath := filepath.Join(t.TempDir(), "ingestion-test")
	database, err := db.New(dbPath)
	if err != nil {
		t.Fatalf("open test db: %v", err)
	}

	org, err := database.CreateOrganization(ctx, queries.CreateOrganizationParams{
		Name:          "default",
		AuthToken:     "token-ingest",
		WebhookSecret: "secret-ingest",
		Enabled:       1,
	})
	if err != nil {
		t.Fatalf("create organization: %v", err)
	}

	if err := database.Close(); err != nil {
		t.Fatalf("close seed db: %v", err)
	}

	factory := NewIngestionStoreFactory(dbPath)
	store, err := factory.Open()
	if err != nil {
		t.Fatalf("open ingestion store: %v", err)
	}

	loadedOrg, err := store.GetOrganizationByAuthToken(ctx, "token-ingest")
	if err != nil {
		t.Fatalf("get org by token: %v", err)
	}
	if loadedOrg.ID != org.ID || !loadedOrg.Enabled {
		t.Fatalf("unexpected organization mapping: %+v", loadedOrg)
	}

	raw := `{"context":{"id":"evt-1","source":"tests/source","type":"dev.cdevents.service.deployed.0.3.0","timestamp":"2026-02-20T12:00:00Z","specversion":"0.5.0"},"subject":{"id":"service/orders","source":"tests/source","content":{"environment":{"id":"staging"},"artifactId":"pkg:generic/orders@v1"}}}`
	subjectSource := "tests/source"
	chainID := "chain-1"

	err = store.AppendEvent(ctx, ports.EventRecord{
		OrganizationID: org.ID,
		EventID:        "evt-1",
		EventType:      "dev.cdevents.service.deployed.0.3.0",
		EventSource:    "tests/source",
		EventTimestamp: "2026-02-20T12:00:00Z",
		SubjectID:      "service/orders",
		SubjectSource:  &subjectSource,
		SubjectType:    "service",
		ChainID:        &chainID,
		RawEventJSON:   raw,
	})
	if err != nil {
		t.Fatalf("append event: %v", err)
	}

	if err := store.Close(); err != nil {
		t.Fatalf("close ingestion store: %v", err)
	}

	reloaded, err := db.New(dbPath)
	if err != nil {
		t.Fatalf("reopen db: %v", err)
	}
	t.Cleanup(func() { _ = reloaded.Close() })

	count, err := reloaded.CountEventStore(ctx)
	if err != nil {
		t.Fatalf("count event store: %v", err)
	}
	if count != 1 {
		t.Fatalf("expected 1 stored event, got %d", count)
	}

	rows, err := reloaded.ListDeploymentsFromEvents(ctx, queries.ListDeploymentsFromEventsParams{OrganizationID: org.ID, Env: "", Service: ""})
	if err != nil {
		t.Fatalf("list deployments from events: %v", err)
	}
	if len(rows) != 1 {
		t.Fatalf("expected 1 deployment row, got %d", len(rows))
	}
}

func TestIngestionStoreEventIdempotencyIsOrganizationScoped(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	dbPath := filepath.Join(t.TempDir(), "ingestion-org-scoped")
	database, err := db.New(dbPath)
	if err != nil {
		t.Fatalf("open test db: %v", err)
	}
	t.Cleanup(func() { _ = database.Close() })

	orgA, err := database.CreateOrganization(ctx, queries.CreateOrganizationParams{
		Name:          "org-a",
		AuthToken:     "token-a",
		WebhookSecret: "secret-a",
		Enabled:       1,
	})
	if err != nil {
		t.Fatalf("create org a: %v", err)
	}
	orgB, err := database.CreateOrganization(ctx, queries.CreateOrganizationParams{
		Name:          "org-b",
		AuthToken:     "token-b",
		WebhookSecret: "secret-b",
		Enabled:       1,
	})
	if err != nil {
		t.Fatalf("create org b: %v", err)
	}

	factory := NewSharedIngestionStoreFactory(database)
	storeA, err := factory.Open()
	if err != nil {
		t.Fatalf("open store a: %v", err)
	}
	storeB, err := factory.Open()
	if err != nil {
		t.Fatalf("open store b: %v", err)
	}

	record := ports.EventRecord{
		EventID:        "shared-event-id",
		EventType:      "dev.cdevents.service.deployed.0.3.0",
		EventSource:    "tests/source",
		EventTimestamp: "2026-02-20T12:00:00Z",
		SubjectID:      "service/orders",
		SubjectType:    "service",
		RawEventJSON:   `{"subject":{"content":{"environment":{"id":"staging"}}}}`,
	}

	record.OrganizationID = orgA.ID
	if err := storeA.AppendEvent(ctx, record); err != nil {
		t.Fatalf("append org a event: %v", err)
	}
	record.OrganizationID = orgB.ID
	if err := storeB.AppendEvent(ctx, record); err != nil {
		t.Fatalf("append org b event: %v", err)
	}

	rowsA, err := database.ListDeploymentsFromEvents(ctx, queries.ListDeploymentsFromEventsParams{OrganizationID: orgA.ID, Env: "", Service: ""})
	if err != nil {
		t.Fatalf("list org a deployments: %v", err)
	}
	if len(rowsA) != 1 {
		t.Fatalf("expected 1 event for org a, got %d", len(rowsA))
	}
	rowsB, err := database.ListDeploymentsFromEvents(ctx, queries.ListDeploymentsFromEventsParams{OrganizationID: orgB.ID, Env: "", Service: ""})
	if err != nil {
		t.Fatalf("list org b deployments: %v", err)
	}
	if len(rowsB) != 1 {
		t.Fatalf("expected 1 event for org b, got %d", len(rowsB))
	}
}
