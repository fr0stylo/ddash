package githubbridge

import (
	"database/sql"
	"path/filepath"
	"testing"
)

func TestInstallStoreListMappingsByOrganization(t *testing.T) {
	t.Parallel()

	store := openTestInstallStore(t)
	defer func() { _ = store.Close() }()

	if err := store.UpsertInstallationMapping(InstallationMapping{InstallationID: 1, OrganizationID: 10, DDashEndpoint: "https://a", DDashAuthToken: "a", DDashWebhookSecret: "a", Enabled: true}); err != nil {
		t.Fatalf("insert mapping 1: %v", err)
	}
	if err := store.UpsertInstallationMapping(InstallationMapping{InstallationID: 2, OrganizationID: 20, DDashEndpoint: "https://b", DDashAuthToken: "b", DDashWebhookSecret: "b", Enabled: true}); err != nil {
		t.Fatalf("insert mapping 2: %v", err)
	}

	mappings, err := store.ListInstallationMappings(10)
	if err != nil {
		t.Fatalf("ListInstallationMappings: %v", err)
	}
	if len(mappings) != 1 {
		t.Fatalf("expected 1 mapping, got %d", len(mappings))
	}
	if mappings[0].InstallationID != 1 {
		t.Fatalf("unexpected installation ID: %d", mappings[0].InstallationID)
	}
}

func TestInstallStoreDeleteInstallationMappingHonorsOrganizationID(t *testing.T) {
	t.Parallel()

	store := openTestInstallStore(t)
	defer func() { _ = store.Close() }()

	if err := store.UpsertInstallationMapping(InstallationMapping{InstallationID: 9, OrganizationID: 55, DDashEndpoint: "https://a", DDashAuthToken: "a", DDashWebhookSecret: "a", Enabled: true}); err != nil {
		t.Fatalf("insert mapping: %v", err)
	}

	err := store.DeleteInstallationMapping(9, 77)
	if err == nil {
		t.Fatalf("expected error for mismatched org")
	}
	if err != sql.ErrNoRows {
		t.Fatalf("expected sql.ErrNoRows, got %v", err)
	}

	if err := store.DeleteInstallationMapping(9, 55); err != nil {
		t.Fatalf("delete mapping: %v", err)
	}
}

func openTestInstallStore(t *testing.T) *InstallStore {
	t.Helper()
	path := filepath.Join(t.TempDir(), "test-store")
	store, err := OpenInstallStore(path)
	if err != nil {
		t.Fatalf("OpenInstallStore: %v", err)
	}
	return store
}
