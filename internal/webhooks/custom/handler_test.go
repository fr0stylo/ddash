package custom

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
	"time"

	cdeventsapi "github.com/cdevents/sdk-go/pkg/api"
	cdeventsv05 "github.com/cdevents/sdk-go/pkg/api/v05"

	"github.com/fr0stylo/ddash/internal/adapters/sqlite"
	"github.com/fr0stylo/ddash/internal/db"
	"github.com/fr0stylo/ddash/internal/db/queries"
)

func TestHandleAcceptsValidDeliveryCDEvent(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	dbPath := filepath.Join(t.TempDir(), "testdb")
	database, err := db.New(dbPath)
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	t.Cleanup(func() { _ = database.Close() })

	const token = "test-token"
	const secret = "test-secret"
	org, err := database.CreateOrganization(ctx, queries.CreateOrganizationParams{
		Name:          "default",
		AuthToken:     token,
		WebhookSecret: secret,
		Enabled:       1,
	})
	if err != nil {
		t.Fatalf("create org: %v", err)
	}

	event, err := cdeventsv05.NewServiceDeployedEvent()
	if err != nil {
		t.Fatalf("new event: %v", err)
	}
	event.SetId("evt-1")
	event.SetSource("tests/source")
	event.SetTimestamp(time.Date(2026, 2, 19, 10, 0, 0, 0, time.UTC))
	event.SetSubjectId("service/orders")
	event.SetSubjectEnvironment(&cdeventsapi.Reference{Id: "staging"})
	event.SetSubjectArtifactId("pkg:generic/orders@abc123")

	body, err := cdeventsapi.AsJsonBytes(event)
	if err != nil {
		t.Fatalf("encode event: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/webhooks/cdevents", bytes.NewReader(body))
	req.Header.Set(AuthorizationHeader, "Bearer "+token)
	req.Header.Set(SignatureHeader, signTest(body, secret))
	req.Header.Set("Content-Type", "application/json")

	rec := httptest.NewRecorder()
	h := NewHandler(sqlite.NewIngestionStoreFactory(dbPath))
	if err := h.Handle(rec, req); err != nil {
		t.Fatalf("handle request: %v", err)
	}

	if rec.Code != http.StatusAccepted {
		t.Fatalf("unexpected status: got=%d want=%d", rec.Code, http.StatusAccepted)
	}

	if err := database.Close(); err != nil {
		t.Fatalf("close initial db: %v", err)
	}

	reloaded, err := db.New(dbPath)
	if err != nil {
		t.Fatalf("reopen db: %v", err)
	}
	t.Cleanup(func() { _ = reloaded.Close() })

	rows, err := reloaded.ListDeploymentsFromEvents(ctx, queries.ListDeploymentsFromEventsParams{OrganizationID: org.ID, Env: "", Service: ""})
	if err != nil {
		t.Fatalf("query projections: %v", err)
	}
	count, err := reloaded.CountEventStore(ctx)
	if err != nil {
		t.Fatalf("count events: %v", err)
	}
	if count != 1 {
		t.Fatalf("unexpected event-store count: got=%d want=1", count)
	}
	serviceCount, err := reloaded.CountEventStoreBySubjectType(ctx, "service")
	if err != nil {
		t.Fatalf("count service events: %v", err)
	}
	if len(rows) != 1 {
		t.Fatalf("unexpected row count: got=%d want=1 (total=%d service=%d)", len(rows), count, serviceCount)
	}
}

func TestHandleRejectsInvalidSignature(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	dbPath := filepath.Join(t.TempDir(), "testdb")
	database, err := db.New(dbPath)
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	t.Cleanup(func() { _ = database.Close() })

	org, err := database.CreateOrganization(ctx, queries.CreateOrganizationParams{
		Name:          "default",
		AuthToken:     "test-token",
		WebhookSecret: "test-secret",
		Enabled:       1,
	})
	if err != nil {
		t.Fatalf("create org: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/webhooks/cdevents", bytes.NewReader([]byte(`{"invalid":true}`)))
	req.Header.Set(AuthorizationHeader, "Bearer test-token")
	req.Header.Set(SignatureHeader, "deadbeef")

	rec := httptest.NewRecorder()
	h := NewHandler(sqlite.NewIngestionStoreFactory(dbPath))
	if err := h.Handle(rec, req); err != nil {
		t.Fatalf("handle request: %v", err)
	}

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("unexpected status: got=%d want=%d", rec.Code, http.StatusUnauthorized)
	}

	if err := database.Close(); err != nil {
		t.Fatalf("close initial db: %v", err)
	}

	reloaded, err := db.New(dbPath)
	if err != nil {
		t.Fatalf("reopen db: %v", err)
	}
	t.Cleanup(func() { _ = reloaded.Close() })

	rows, err := reloaded.ListDeploymentsFromEvents(ctx, queries.ListDeploymentsFromEventsParams{OrganizationID: org.ID, Env: "", Service: ""})
	if err != nil {
		t.Fatalf("query projections: %v", err)
	}
	count, err := reloaded.CountEventStore(ctx)
	if err != nil {
		t.Fatalf("count events: %v", err)
	}
	if count != 0 {
		t.Fatalf("unexpected event-store count: got=%d want=0", count)
	}
	if len(rows) != 0 {
		t.Fatalf("unexpected row count: got=%d want=0", len(rows))
	}
}

func signTest(body []byte, secret string) string {
	mac := hmac.New(sha256.New, []byte(secret))
	_, _ = mac.Write(body)
	return hex.EncodeToString(mac.Sum(nil))
}
