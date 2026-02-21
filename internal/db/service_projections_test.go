package db

import (
	"context"
	"database/sql"
	"fmt"
	"testing"

	"github.com/fr0stylo/ddash/internal/db/queries"
)

func TestAppendEventStore_UpdatesServiceProjectionTables(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	database := newTestDatabase(t)
	org := createTestOrganization(t, ctx, database)

	appendEvent(t, ctx, database, org.ID, "p1", "dev.cdevents.service.deployed.0.3.0", "2026-02-21T10:00:00Z", "service/payments", "staging", "pkg:generic/payments@v1")
	appendEvent(t, ctx, database, org.ID, "p2", "dev.cdevents.service.rolledback.0.3.0", "2026-02-21T11:00:00Z", "service/payments", "staging", "pkg:generic/payments@v0")

	state, err := database.GetServiceCurrentState(ctx, queries.GetServiceCurrentStateParams{
		OrganizationID: org.ID,
		ServiceName:    "payments",
	})
	if err != nil {
		t.Fatalf("get service current state: %v", err)
	}
	if state.LatestStatus != "warning" {
		t.Fatalf("unexpected latest status: got=%q want=%q", state.LatestStatus, "warning")
	}

	stats, err := database.GetServiceDeliveryStats30d(ctx, queries.GetServiceDeliveryStats30dParams{
		OrganizationID: org.ID,
		ServiceName:    "payments",
	})
	if err != nil {
		t.Fatalf("get service delivery stats: %v", err)
	}
	if toInt64(stats.DeploySuccessCount) != 1 {
		t.Fatalf("unexpected success count: got=%d want=1", toInt64(stats.DeploySuccessCount))
	}
	if toInt64(stats.RollbackCount) != 1 {
		t.Fatalf("unexpected rollback count: got=%d want=1", toInt64(stats.RollbackCount))
	}

	links, err := database.ListServiceChangeLinksRecent(ctx, queries.ListServiceChangeLinksRecentParams{
		OrganizationID: org.ID,
		ServiceName:    "payments",
		Limit:          5,
	})
	if err != nil {
		t.Fatalf("list service change links: %v", err)
	}
	if len(links) != 2 {
		t.Fatalf("unexpected links len: got=%d want=2", len(links))
	}
}

func toInt64(value interface{}) int64 {
	switch typed := value.(type) {
	case nil:
		return 0
	case int64:
		return typed
	case int:
		return int64(typed)
	case float64:
		return int64(typed)
	case []byte:
		var parsed int64
		_, _ = fmt.Sscan(string(typed), &parsed)
		return parsed
	case string:
		var parsed int64
		_, _ = fmt.Sscan(typed, &parsed)
		return parsed
	default:
		return 0
	}
}

func TestAppendEventStore_DuplicateDoesNotDoubleCountProjection(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	database := newTestDatabase(t)
	org := createTestOrganization(t, ctx, database)

	raw := `{"context":{"id":"dup-1","source":"tests/source","type":"dev.cdevents.service.deployed.0.3.0","timestamp":"2026-02-21T10:00:00Z","specversion":"0.5.0"},"subject":{"id":"service/catalog","source":"tests/source","content":{"environment":{"id":"prod"},"artifactId":"pkg:generic/catalog@v1"}}}`
	params := queries.AppendEventStoreParams{
		OrganizationID: org.ID,
		EventID:        "dup-1",
		EventType:      "dev.cdevents.service.deployed.0.3.0",
		EventSource:    "tests/source",
		EventTimestamp: "2026-02-21T10:00:00Z",
		EventTsMs:      mustUnixMillis(t, "2026-02-21T10:00:00Z"),
		SubjectID:      "service/catalog",
		SubjectSource:  sql.NullString{String: "tests/source", Valid: true},
		SubjectType:    "service",
		ChainID:        sql.NullString{},
		RawEventJson:   raw,
	}
	if err := database.AppendEventStore(ctx, params); err != nil {
		t.Fatalf("append first: %v", err)
	}
	if err := database.AppendEventStore(ctx, params); err != nil {
		t.Fatalf("append duplicate: %v", err)
	}

	stats, err := database.GetServiceDeliveryStats30d(ctx, queries.GetServiceDeliveryStats30dParams{
		OrganizationID: org.ID,
		ServiceName:    "catalog",
	})
	if err != nil {
		t.Fatalf("get stats: %v", err)
	}
	if toInt64(stats.DeploySuccessCount) != 1 {
		t.Fatalf("duplicate should not double count: got=%d want=1", toInt64(stats.DeploySuccessCount))
	}
}
