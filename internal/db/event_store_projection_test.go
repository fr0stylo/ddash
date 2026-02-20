package db

import (
	"context"
	"database/sql"
	"fmt"
	"path/filepath"
	"strings"
	"testing"

	"github.com/fr0stylo/ddash/internal/db/queries"
)

func TestServiceProjectionUsesLatestEventPerEnvironment(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	database := newTestDatabase(t)
	org := createTestOrganization(t, ctx, database)

	appendEvent(t, ctx, database, org.ID, "e1", "dev.cdevents.service.deployed.0.3.0", "2026-02-19T10:00:00Z", "service/orders", "staging", "pkg:generic/orders@a1")
	appendEvent(t, ctx, database, org.ID, "e2", "dev.cdevents.service.removed.0.3.0", "2026-02-19T10:05:00Z", "service/orders", "staging", "pkg:generic/orders@a2")

	rows, err := database.ListServiceInstancesFromEvents(ctx, org.ID)
	if err != nil {
		t.Fatalf("list service projections: %v", err)
	}
	if len(rows) != 1 {
		t.Fatalf("unexpected row count: got=%d want=1", len(rows))
	}

	if rows[0].Status != "out-of-sync" {
		t.Fatalf("unexpected status: got=%q want=%q", rows[0].Status, "out-of-sync")
	}
	if rows[0].LastDeployAt != "2026-02-19T10:05:00Z" {
		t.Fatalf("unexpected timestamp: got=%q", rows[0].LastDeployAt)
	}
}

func TestDeploymentHistoryProjectionHonorsLimitAndOrder(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	database := newTestDatabase(t)
	org := createTestOrganization(t, ctx, database)

	appendEvent(t, ctx, database, org.ID, "h1", "dev.cdevents.service.deployed.0.3.0", "2026-02-19T09:00:00Z", "service/billing", "dev", "pkg:generic/billing@v1")
	appendEvent(t, ctx, database, org.ID, "h2", "dev.cdevents.service.upgraded.0.3.0", "2026-02-19T10:00:00Z", "service/billing", "staging", "pkg:generic/billing@v2")
	appendEvent(t, ctx, database, org.ID, "h3", "dev.cdevents.service.deployed.0.3.0", "2026-02-19T11:00:00Z", "service/billing", "production", "pkg:generic/billing@v3")

	history, err := database.ListDeploymentHistoryByServiceFromEvents(ctx, queries.ListDeploymentHistoryByServiceFromEventsParams{
		OrganizationID: org.ID,
		Service:        "billing",
		Limit:          2,
	})
	if err != nil {
		t.Fatalf("list history: %v", err)
	}
	if len(history) != 2 {
		t.Fatalf("unexpected history count: got=%d want=2", len(history))
	}

	if history[0].DeployedAt != "2026-02-19T11:00:00Z" {
		t.Fatalf("unexpected first history timestamp: got=%q", history[0].DeployedAt)
	}
	if history[1].DeployedAt != "2026-02-19T10:00:00Z" {
		t.Fatalf("unexpected second history timestamp: got=%q", history[1].DeployedAt)
	}
}

func newTestDatabase(t *testing.T) *Database {
	t.Helper()

	database, err := New(filepath.Join(t.TempDir(), "events"))
	if err != nil {
		t.Fatalf("open test db: %v", err)
	}
	t.Cleanup(func() { _ = database.Close() })
	return database
}

func createTestOrganization(t *testing.T, ctx context.Context, database *Database) queries.Organization {
	t.Helper()
	name := strings.ReplaceAll(t.Name(), "/", "-")

	org, err := database.CreateOrganization(ctx, queries.CreateOrganizationParams{
		Name:          fmt.Sprintf("org-%s", name),
		AuthToken:     fmt.Sprintf("token-%s", name),
		WebhookSecret: "secret",
		Enabled:       1,
	})
	if err != nil {
		t.Fatalf("create organization: %v", err)
	}
	return org
}

func appendEvent(
	t *testing.T,
	ctx context.Context,
	database *Database,
	organizationID int64,
	eventID string,
	eventType string,
	timestamp string,
	subjectID string,
	environmentID string,
	artifactID string,
) {
	t.Helper()

	raw := fmt.Sprintf(`{"context":{"id":%q,"source":"tests/source","type":%q,"timestamp":%q,"specversion":"0.5.0"},"subject":{"id":%q,"source":"tests/source","content":{"environment":{"id":%q},"artifactId":%q}}}`,
		eventID,
		eventType,
		timestamp,
		subjectID,
		environmentID,
		artifactID,
	)

	err := database.AppendEventStore(ctx, queries.AppendEventStoreParams{
		OrganizationID: organizationID,
		EventID:        eventID,
		EventType:      eventType,
		EventSource:    "tests/source",
		EventTimestamp: timestamp,
		SubjectID:      subjectID,
		SubjectSource:  sql.NullString{String: "tests/source", Valid: true},
		SubjectType:    "service",
		ChainID:        sql.NullString{},
		RawEventJson:   raw,
	})
	if err != nil {
		t.Fatalf("append event %s: %v", eventID, err)
	}
}
