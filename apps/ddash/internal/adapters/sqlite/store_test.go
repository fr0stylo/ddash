package sqlite

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/fr0stylo/ddash/apps/ddash/internal/app/ports"
	"github.com/fr0stylo/ddash/internal/db"
	"github.com/fr0stylo/ddash/internal/db/queries"
)

func newTestStore(t *testing.T) (*Store, *db.Database) {
	t.Helper()

	database, err := db.New(filepath.Join(t.TempDir(), "adapter-test"))
	if err != nil {
		t.Fatalf("open test db: %v", err)
	}
	t.Cleanup(func() { _ = database.Close() })

	return NewStore(database), database
}

func TestUpdateOrganizationSettingsPersistsAllFields(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	store, database := newTestStore(t)

	org, err := store.CreateOrganization(ctx, ports.CreateOrganizationInput{
		Name:          "org-a",
		AuthToken:     "token-a",
		WebhookSecret: "secret-a",
		Enabled:       true,
	})
	if err != nil {
		t.Fatalf("create org: %v", err)
	}

	err = store.UpdateOrganizationSettings(ctx, org.ID, ports.OrganizationSettingsUpdate{
		AuthToken:     "new-token",
		WebhookSecret: "new-secret",
		Enabled:       false,
		RequiredFields: []ports.RequiredField{
			{Label: "team", Type: "text", Filterable: true},
			{Label: "tier", Type: "select", Filterable: false},
		},
		EnvironmentOrder: []string{"production", "staging"},
	})
	if err != nil {
		t.Fatalf("update settings: %v", err)
	}

	updated, err := store.GetDefaultOrganization(ctx)
	if err != nil {
		t.Fatalf("get default org: %v", err)
	}
	if updated.AuthToken != "new-token" || updated.WebhookSecret != "new-secret" || updated.Enabled {
		t.Fatalf("unexpected updated org: %+v", updated)
	}

	required, err := database.ListOrganizationRequiredFields(ctx, org.ID)
	if err != nil {
		t.Fatalf("list required fields: %v", err)
	}
	if len(required) != 2 {
		t.Fatalf("expected 2 required fields, got %d", len(required))
	}
	if required[0].Label != "team" || required[0].FieldType != "text" || required[0].IsFilterable != 1 {
		t.Fatalf("unexpected first required field: %+v", required[0])
	}

	envOrder, err := database.ListOrganizationEnvironmentPriorities(ctx, org.ID)
	if err != nil {
		t.Fatalf("list environment priorities: %v", err)
	}
	if len(envOrder) != 2 {
		t.Fatalf("expected 2 environment priorities, got %d", len(envOrder))
	}
	if envOrder[0].Environment != "production" || envOrder[1].Environment != "staging" {
		t.Fatalf("unexpected environment order: %+v", envOrder)
	}
}

func TestReplaceServiceMetadataReplacesPreviousRows(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	store, database := newTestStore(t)

	org, err := store.CreateOrganization(ctx, ports.CreateOrganizationInput{
		Name:          "org-b",
		AuthToken:     "token-b",
		WebhookSecret: "secret-b",
		Enabled:       true,
	})
	if err != nil {
		t.Fatalf("create org: %v", err)
	}

	if err := store.ReplaceServiceMetadata(ctx, org.ID, "orders", []ports.MetadataValue{{Label: "team", Value: "platform"}, {Label: "tier", Value: "backend"}}); err != nil {
		t.Fatalf("initial replace metadata: %v", err)
	}

	if err := store.ReplaceServiceMetadata(ctx, org.ID, "orders", []ports.MetadataValue{{Label: "team", Value: "core"}}); err != nil {
		t.Fatalf("second replace metadata: %v", err)
	}

	rows, err := database.ListServiceMetadataByService(ctx, queries.ListServiceMetadataByServiceParams{
		OrganizationID: org.ID,
		ServiceName:    "orders",
	})
	if err != nil {
		t.Fatalf("list service metadata: %v", err)
	}
	if len(rows) != 1 {
		t.Fatalf("expected 1 metadata row after replacement, got %d", len(rows))
	}
	if rows[0].Label != "team" || rows[0].Value != "core" {
		t.Fatalf("unexpected metadata row: %+v", rows[0])
	}
}

func TestUpdateOrganizationNameAndEnabled(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	store, _ := newTestStore(t)

	org, err := store.CreateOrganization(ctx, ports.CreateOrganizationInput{
		Name:          "org-c",
		AuthToken:     "token-c",
		WebhookSecret: "secret-c",
		Enabled:       true,
	})
	if err != nil {
		t.Fatalf("create org: %v", err)
	}

	if err := store.UpdateOrganizationName(ctx, org.ID, "org-c-renamed"); err != nil {
		t.Fatalf("update org name: %v", err)
	}
	if err := store.UpdateOrganizationEnabled(ctx, org.ID, false); err != nil {
		t.Fatalf("update org enabled: %v", err)
	}

	updated, err := store.GetOrganizationByID(ctx, org.ID)
	if err != nil {
		t.Fatalf("get org by id: %v", err)
	}
	if updated.Name != "org-c-renamed" {
		t.Fatalf("expected renamed org, got %q", updated.Name)
	}
	if updated.Enabled {
		t.Fatalf("expected org to be disabled")
	}
}

func TestDeleteOrganizationRemovesRow(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	store, _ := newTestStore(t)

	orgA, err := store.CreateOrganization(ctx, ports.CreateOrganizationInput{
		Name:          "org-d",
		AuthToken:     "token-d",
		WebhookSecret: "secret-d",
		Enabled:       true,
	})
	if err != nil {
		t.Fatalf("create org a: %v", err)
	}
	_, err = store.CreateOrganization(ctx, ports.CreateOrganizationInput{
		Name:          "org-e",
		AuthToken:     "token-e",
		WebhookSecret: "secret-e",
		Enabled:       true,
	})
	if err != nil {
		t.Fatalf("create org b: %v", err)
	}

	if err := store.DeleteOrganization(ctx, orgA.ID); err != nil {
		t.Fatalf("delete org: %v", err)
	}

	rows, err := store.ListOrganizations(ctx)
	if err != nil {
		t.Fatalf("list orgs: %v", err)
	}
	if len(rows) != 1 {
		t.Fatalf("expected 1 organization left, got %d", len(rows))
	}
	if rows[0].Name != "org-e" {
		t.Fatalf("expected org-e to remain, got %q", rows[0].Name)
	}
}
