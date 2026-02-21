package services

import (
	"context"
	"net/url"
	"testing"

	"github.com/fr0stylo/ddash/internal/app/domain"
	"github.com/fr0stylo/ddash/internal/app/ports"
)

type fakeServiceReadStore struct {
	services            []domain.Service
	deployments         []domain.DeploymentRow
	renderVersion       int64
	latest              ports.ServiceLatest
	serviceEnvs         []domain.ServiceEnvironment
	history             []domain.DeploymentRecord
	required            []ports.RequiredField
	serviceMetadata     []ports.MetadataValue
	orgMetadata         []ports.ServiceMetadataValue
	envPriorities       []string
	discoveredEnvs      []string
	serviceListEnvInput string
	serviceOrgIDInput   int64
	deploymentInputEnv  string
	deploymentInputSvc  string
	deploymentOrgID     int64
}

func (f *fakeServiceReadStore) ListServiceInstances(_ context.Context, organizationID int64, env string) ([]domain.Service, error) {
	f.serviceOrgIDInput = organizationID
	f.serviceListEnvInput = env
	return f.services, nil
}

func (f *fakeServiceReadStore) ListDeployments(_ context.Context, organizationID int64, env, service string) ([]domain.DeploymentRow, error) {
	f.deploymentOrgID = organizationID
	f.deploymentInputEnv = env
	f.deploymentInputSvc = service
	return f.deployments, nil
}

func (f *fakeServiceReadStore) GetOrganizationRenderVersion(context.Context, int64) (int64, error) {
	if f.renderVersion == 0 {
		return 1, nil
	}
	return f.renderVersion, nil
}

func (f *fakeServiceReadStore) GetServiceLatest(context.Context, int64, string) (ports.ServiceLatest, error) {
	return f.latest, nil
}

func (f *fakeServiceReadStore) ListServiceEnvironments(context.Context, int64, string) ([]domain.ServiceEnvironment, error) {
	return f.serviceEnvs, nil
}

func (f *fakeServiceReadStore) ListDeploymentHistory(context.Context, int64, string, int64) ([]domain.DeploymentRecord, error) {
	return f.history, nil
}

func (f *fakeServiceReadStore) ListRequiredFields(context.Context, int64) ([]ports.RequiredField, error) {
	return f.required, nil
}

func (f *fakeServiceReadStore) ListServiceMetadata(context.Context, int64, string) ([]ports.MetadataValue, error) {
	return f.serviceMetadata, nil
}

func (f *fakeServiceReadStore) ListServiceMetadataValuesByOrganization(context.Context, int64) ([]ports.ServiceMetadataValue, error) {
	return f.orgMetadata, nil
}

func (f *fakeServiceReadStore) ListEnvironmentPriorities(context.Context, int64) ([]string, error) {
	return f.envPriorities, nil
}

func (f *fakeServiceReadStore) ListDiscoveredEnvironments(context.Context, int64) ([]string, error) {
	return f.discoveredEnvs, nil
}

func (f *fakeServiceReadStore) GetServiceCurrentState(context.Context, int64, string) (ports.ServiceCurrentState, error) {
	return ports.ServiceCurrentState{}, nil
}

func (f *fakeServiceReadStore) GetServiceDeliveryStats30d(context.Context, int64, string) (ports.ServiceDeliveryStats, error) {
	return ports.ServiceDeliveryStats{}, nil
}

func (f *fakeServiceReadStore) ListServiceChangeLinksRecent(context.Context, int64, string, int64) ([]ports.ServiceChangeLink, error) {
	return nil, nil
}

func TestGetServicesByEnv_AppliesMetadata(t *testing.T) {
	store := &fakeServiceReadStore{
		services: []domain.Service{
			{Title: "svc-a"},
			{Title: "svc-b"},
		},
		required: []ports.RequiredField{
			{Label: "team", Filterable: true},
			{Label: "tier", Filterable: false},
		},
		orgMetadata: []ports.ServiceMetadataValue{
			{ServiceName: "svc-a", Label: "team", Value: "Platform"},
			{ServiceName: "svc-a", Label: "tier", Value: "Backend"},
			{ServiceName: "svc-b", Label: "team", Value: "Ops"},
		},
	}

	svc := NewServiceReadServiceFromStore(store)
	rows, err := svc.GetServicesByEnv(context.Background(), 101, "prod")
	if err != nil {
		t.Fatalf("GetServicesByEnv returned error: %v", err)
	}

	if store.serviceListEnvInput != "prod" {
		t.Fatalf("expected env to be passed through, got %q", store.serviceListEnvInput)
	}
	if store.serviceOrgIDInput != 101 {
		t.Fatalf("expected organization id 101, got %d", store.serviceOrgIDInput)
	}
	if len(rows) != 2 {
		t.Fatalf("expected 2 services, got %d", len(rows))
	}

	if rows[0].MissingMetadata != 0 {
		t.Fatalf("expected svc-a missing metadata to be 0, got %d", rows[0].MissingMetadata)
	}
	if rows[0].MetadataTags != "|team:platform|" {
		t.Fatalf("expected svc-a tags to include only filterable label, got %q", rows[0].MetadataTags)
	}

	if rows[1].MissingMetadata != 1 {
		t.Fatalf("expected svc-b missing metadata to be 1, got %d", rows[1].MissingMetadata)
	}
	if rows[1].MetadataTags != "|team:ops|" {
		t.Fatalf("expected svc-b tags to be |team:ops|, got %q", rows[1].MetadataTags)
	}
}

func TestGetDeployments_AppliesMetadataAndReturnsOptions(t *testing.T) {
	store := &fakeServiceReadStore{
		deployments: []domain.DeploymentRow{
			{Service: "svc-a"},
			{Service: "svc-b"},
		},
		required: []ports.RequiredField{
			{Label: "team", Filterable: true},
		},
		orgMetadata: []ports.ServiceMetadataValue{
			{ServiceName: "svc-a", Label: "team", Value: "Platform"},
			{ServiceName: "svc-b", Label: "team", Value: "Ops"},
		},
	}

	svc := NewServiceReadServiceFromStore(store)
	rows, options, err := svc.GetDeployments(context.Background(), 202, "dev", "svc-a")
	if err != nil {
		t.Fatalf("GetDeployments returned error: %v", err)
	}

	if store.deploymentInputEnv != "dev" || store.deploymentInputSvc != "svc-a" {
		t.Fatalf("expected env/service filters passed to store, got env=%q svc=%q", store.deploymentInputEnv, store.deploymentInputSvc)
	}
	if store.deploymentOrgID != 202 {
		t.Fatalf("expected organization id 202, got %d", store.deploymentOrgID)
	}
	if rows[0].MetadataTags != "|team:platform|" {
		t.Fatalf("expected first row tags |team:platform|, got %q", rows[0].MetadataTags)
	}
	if rows[1].MetadataTags != "|team:ops|" {
		t.Fatalf("expected second row tags |team:ops|, got %q", rows[1].MetadataTags)
	}
	if len(options) != 3 {
		t.Fatalf("expected 3 metadata options (all + 2 tags), got %d", len(options))
	}
	if options[0].Value != "all" {
		t.Fatalf("expected first option to be all, got %q", options[0].Value)
	}
}

func TestGetServiceDetail_ComposesMetadataAndEnvironmentOrder(t *testing.T) {
	store := &fakeServiceReadStore{
		latest: ports.ServiceLatest{Name: "svc/a", IntegrationType: "argocd"},
		required: []ports.RequiredField{
			{Label: "team", Filterable: true},
			{Label: "owner", Filterable: false},
		},
		serviceMetadata: []ports.MetadataValue{{Label: "team", Value: "platform"}},
		serviceEnvs: []domain.ServiceEnvironment{
			{Name: "staging"},
			{Name: "dev"},
			{Name: "prod"},
		},
		envPriorities: []string{"prod", "staging"},
		history:       []domain.DeploymentRecord{{Ref: "abc123"}},
	}

	svc := NewServiceReadServiceFromStore(store)
	detail, err := svc.GetServiceDetail(context.Background(), 303, "svc/a")
	if err != nil {
		t.Fatalf("GetServiceDetail returned error: %v", err)
	}

	expectedSaveURL := "/s/" + url.PathEscape("svc/a") + "/metadata"
	if detail.MetadataSaveURL != expectedSaveURL {
		t.Fatalf("expected metadata save url %q, got %q", expectedSaveURL, detail.MetadataSaveURL)
	}
	if detail.MissingMetadata != 1 {
		t.Fatalf("expected missing metadata count 1, got %d", detail.MissingMetadata)
	}
	if len(detail.MetadataFields) != 2 {
		t.Fatalf("expected 2 metadata fields, got %d", len(detail.MetadataFields))
	}
	if detail.MetadataFields[0].Label != "team" || detail.MetadataFields[0].Value != "platform" {
		t.Fatalf("expected first metadata field to be team=platform, got %+v", detail.MetadataFields[0])
	}
	if detail.MetadataFields[1].Label != "owner" || detail.MetadataFields[1].Value != "" {
		t.Fatalf("expected second metadata field to be owner with empty value, got %+v", detail.MetadataFields[1])
	}

	if len(detail.Environments) != 3 {
		t.Fatalf("expected 3 environments, got %d", len(detail.Environments))
	}
	if detail.Environments[0].Name != "prod" || detail.Environments[1].Name != "staging" || detail.Environments[2].Name != "dev" {
		t.Fatalf("unexpected environment order: %+v", detail.Environments)
	}
}
