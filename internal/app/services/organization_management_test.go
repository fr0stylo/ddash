package services

import (
	"context"
	"testing"

	"github.com/fr0stylo/ddash/internal/app/ports"
)

type fakeOrgStore struct {
	orgs      []ports.Organization
	updatedID int64
	updatedOn bool
	deletedID int64
}

func (f *fakeOrgStore) GetDefaultOrganization(context.Context) (ports.Organization, error) {
	return ports.Organization{}, nil
}

func (f *fakeOrgStore) CreateOrganization(context.Context, ports.CreateOrganizationInput) (ports.Organization, error) {
	return ports.Organization{}, nil
}

func (f *fakeOrgStore) GetOrganizationByID(_ context.Context, id int64) (ports.Organization, error) {
	for _, org := range f.orgs {
		if org.ID == id {
			return org, nil
		}
	}
	return ports.Organization{}, nil
}

func (f *fakeOrgStore) ListOrganizations(context.Context) ([]ports.Organization, error) {
	return f.orgs, nil
}

func (f *fakeOrgStore) UpdateOrganizationName(context.Context, int64, string) error { return nil }

func (f *fakeOrgStore) UpdateOrganizationEnabled(_ context.Context, organizationID int64, enabled bool) error {
	f.updatedID = organizationID
	f.updatedOn = enabled
	return nil
}

func (f *fakeOrgStore) DeleteOrganization(_ context.Context, organizationID int64) error {
	f.deletedID = organizationID
	return nil
}

func (f *fakeOrgStore) UpsertUser(context.Context, ports.UpsertUserInput) (ports.User, error) {
	return ports.User{}, nil
}

func (f *fakeOrgStore) GetUserByID(context.Context, int64) (ports.User, error) {
	return ports.User{}, nil
}

func (f *fakeOrgStore) GetUserByEmailOrNickname(context.Context, string, string) (ports.User, error) {
	return ports.User{}, nil
}

func (f *fakeOrgStore) ListOrganizationsByUser(context.Context, int64) ([]ports.Organization, error) {
	return f.orgs, nil
}

func (f *fakeOrgStore) GetOrganizationMemberRole(context.Context, int64, int64) (string, error) {
	return "owner", nil
}

func (f *fakeOrgStore) UpsertOrganizationMember(context.Context, int64, int64, string) error {
	return nil
}

func (f *fakeOrgStore) DeleteOrganizationMember(context.Context, int64, int64) error {
	return nil
}

func (f *fakeOrgStore) CountOrganizationOwners(context.Context, int64) (int64, error) {
	return 1, nil
}

func (f *fakeOrgStore) ListOrganizationMembers(context.Context, int64) ([]ports.OrganizationMember, error) {
	return nil, nil
}

func (f *fakeOrgStore) ListOrganizationRequiredFields(context.Context, int64) ([]ports.RequiredField, error) {
	return nil, nil
}

func (f *fakeOrgStore) ListOrganizationEnvironmentPriorities(context.Context, int64) ([]string, error) {
	return nil, nil
}

func (f *fakeOrgStore) ListOrganizationFeatures(context.Context, int64) ([]ports.OrganizationFeature, error) {
	return nil, nil
}

func (f *fakeOrgStore) ListOrganizationPreferences(context.Context, int64) ([]ports.OrganizationPreference, error) {
	return nil, nil
}

func (f *fakeOrgStore) ListDistinctServiceEnvironmentsFromEvents(context.Context, int64) ([]string, error) {
	return nil, nil
}

func (f *fakeOrgStore) UpdateOrganizationSettings(context.Context, int64, ports.OrganizationSettingsUpdate) error {
	return nil
}

func (f *fakeOrgStore) ReplaceServiceMetadata(context.Context, int64, string, []ports.MetadataValue) error {
	return nil
}

func TestGetActiveOrDefaultOrganizationFallsBackToEnabled(t *testing.T) {
	svc := NewOrganizationManagementService(&fakeOrgStore{orgs: []ports.Organization{{ID: 1, Name: "a", Enabled: false}, {ID: 2, Name: "b", Enabled: true}}})
	org, err := svc.GetActiveOrDefaultOrganization(context.Background(), 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if org.ID != 2 {
		t.Fatalf("expected fallback org id 2, got %d", org.ID)
	}
}

func TestSetOrganizationEnabledLeavesLastEnabled(t *testing.T) {
	store := &fakeOrgStore{orgs: []ports.Organization{{ID: 1, Name: "a", Enabled: true}}}
	svc := NewOrganizationManagementService(store)
	if err := svc.SetOrganizationEnabled(context.Background(), 1, false); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if store.updatedID != 0 {
		t.Fatalf("expected no update for last enabled org, got id %d", store.updatedID)
	}
}

func TestDeleteOrganizationSkipsLastOrganization(t *testing.T) {
	store := &fakeOrgStore{orgs: []ports.Organization{{ID: 1, Name: "a", Enabled: true}}}
	svc := NewOrganizationManagementService(store)
	err := svc.DeleteOrganization(context.Background(), 1)
	if err == nil {
		t.Fatalf("expected error when deleting last organization")
	}
	if store.deletedID != 0 {
		t.Fatalf("expected no delete call, got %d", store.deletedID)
	}
}
