package services

import (
	"context"
	"errors"
	"testing"

	"github.com/fr0stylo/ddash/internal/app/ports"
)

type metadataStoreFake struct {
	required []ports.RequiredField
	values   []ports.MetadataValue
}

func (f *metadataStoreFake) GetDefaultOrganization(context.Context) (ports.Organization, error) {
	return ports.Organization{}, nil
}

func (f *metadataStoreFake) GetOrganizationByID(context.Context, int64) (ports.Organization, error) {
	return ports.Organization{}, nil
}

func (f *metadataStoreFake) GetOrganizationByJoinCode(context.Context, string) (ports.Organization, error) {
	return ports.Organization{}, nil
}

func (f *metadataStoreFake) ListOrganizations(context.Context) ([]ports.Organization, error) {
	return nil, nil
}

func (f *metadataStoreFake) CreateOrganization(context.Context, ports.CreateOrganizationInput) (ports.Organization, error) {
	return ports.Organization{}, nil
}

func (f *metadataStoreFake) UpdateOrganizationName(context.Context, int64, string) error {
	return nil
}

func (f *metadataStoreFake) UpdateOrganizationEnabled(context.Context, int64, bool) error {
	return nil
}

func (f *metadataStoreFake) DeleteOrganization(context.Context, int64) error {
	return nil
}

func (f *metadataStoreFake) UpsertUser(context.Context, ports.UpsertUserInput) (ports.User, error) {
	return ports.User{}, nil
}

func (f *metadataStoreFake) GetUserByID(context.Context, int64) (ports.User, error) {
	return ports.User{}, nil
}

func (f *metadataStoreFake) GetUserByEmailOrNickname(context.Context, string, string) (ports.User, error) {
	return ports.User{}, nil
}

func (f *metadataStoreFake) ListOrganizationsByUser(context.Context, int64) ([]ports.Organization, error) {
	return nil, nil
}

func (f *metadataStoreFake) GetOrganizationMemberRole(context.Context, int64, int64) (string, error) {
	return "owner", nil
}

func (f *metadataStoreFake) UpsertOrganizationMember(context.Context, int64, int64, string) error {
	return nil
}

func (f *metadataStoreFake) DeleteOrganizationMember(context.Context, int64, int64) error {
	return nil
}

func (f *metadataStoreFake) CountOrganizationOwners(context.Context, int64) (int64, error) {
	return 1, nil
}

func (f *metadataStoreFake) ListOrganizationMembers(context.Context, int64) ([]ports.OrganizationMember, error) {
	return nil, nil
}

func (f *metadataStoreFake) UpsertOrganizationJoinRequest(context.Context, int64, int64, string) error {
	return nil
}

func (f *metadataStoreFake) ListPendingOrganizationJoinRequests(context.Context, int64) ([]ports.OrganizationJoinRequest, error) {
	return nil, nil
}

func (f *metadataStoreFake) SetOrganizationJoinRequestStatus(context.Context, int64, int64, string, int64) error {
	return nil
}

func (f *metadataStoreFake) ListOrganizationRequiredFields(context.Context, int64) ([]ports.RequiredField, error) {
	return f.required, nil
}

func (f *metadataStoreFake) ListOrganizationEnvironmentPriorities(context.Context, int64) ([]string, error) {
	return nil, nil
}

func (f *metadataStoreFake) ListOrganizationFeatures(context.Context, int64) ([]ports.OrganizationFeature, error) {
	return nil, nil
}

func (f *metadataStoreFake) ListOrganizationPreferences(context.Context, int64) ([]ports.OrganizationPreference, error) {
	return nil, nil
}

func (f *metadataStoreFake) ListDistinctServiceEnvironmentsFromEvents(context.Context, int64) ([]string, error) {
	return nil, nil
}

func (f *metadataStoreFake) UpdateOrganizationSettings(context.Context, int64, ports.OrganizationSettingsUpdate) error {
	return nil
}

func (f *metadataStoreFake) ReplaceServiceMetadata(_ context.Context, _ int64, _ string, values []ports.MetadataValue) error {
	f.values = values
	return nil
}

func TestMetadataStrictRejectsMissingRequired(t *testing.T) {
	store := &metadataStoreFake{required: []ports.RequiredField{{Label: "team"}, {Label: "owner"}}}
	svc := NewMetadataService(store)

	err := svc.UpdateServiceMetadata(
		context.Background(),
		1,
		"svc-a",
		[]MetadataFieldUpdate{{Label: "team", Value: "platform"}},
		true,
	)
	if !errors.Is(err, ErrRequiredMetadataMissing) {
		t.Fatalf("expected ErrRequiredMetadataMissing, got %v", err)
	}
}

func TestMetadataNonStrictAllowsPartial(t *testing.T) {
	store := &metadataStoreFake{required: []ports.RequiredField{{Label: "team"}, {Label: "owner"}}}
	svc := NewMetadataService(store)

	err := svc.UpdateServiceMetadata(
		context.Background(),
		1,
		"svc-a",
		[]MetadataFieldUpdate{{Label: "team", Value: "platform"}},
		false,
	)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(store.values) != 1 || store.values[0].Label != "team" {
		t.Fatalf("unexpected persisted values: %+v", store.values)
	}
}
