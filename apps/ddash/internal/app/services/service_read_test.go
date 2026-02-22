package services

import (
	"context"
	"net/url"
	"testing"

	"github.com/fr0stylo/ddash/apps/ddash/internal/app/domain"
	"github.com/fr0stylo/ddash/apps/ddash/internal/app/ports"
	"github.com/fr0stylo/ddash/apps/ddash/internal/app/ports/mocks"
)

func TestGetServicesByEnv_AppliesMetadata(t *testing.T) {
	queryStore := mocks.NewMockServiceQueryStore(t)
	metadataStore := mocks.NewMockServiceMetadataStore(t)
	analyticsStore := mocks.NewMockServiceAnalyticsStore(t)

	queryStore.On("ListServiceInstances", context.Background(), int64(101), "prod").Return([]domain.Service{
		{Title: "svc-a"},
		{Title: "svc-b"},
	}, nil)

	metadataStore.On("ListRequiredFields", context.Background(), int64(101)).Return([]ports.RequiredField{
		{Label: "team", Filterable: true},
		{Label: "tier", Filterable: false},
	}, nil)

	metadataStore.On("ListServiceMetadataValuesByOrganization", context.Background(), int64(101)).Return([]ports.ServiceMetadataValue{
		{ServiceName: "svc-a", Label: "team", Value: "Platform"},
		{ServiceName: "svc-a", Label: "tier", Value: "Backend"},
		{ServiceName: "svc-b", Label: "team", Value: "Ops"},
	}, nil)

	svc := NewServiceReadService(queryStore, metadataStore, analyticsStore)
	rows, err := svc.GetServicesByEnv(context.Background(), 101, "prod")
	if err != nil {
		t.Fatalf("GetServicesByEnv returned error: %v", err)
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
	queryStore := mocks.NewMockServiceQueryStore(t)
	metadataStore := mocks.NewMockServiceMetadataStore(t)
	analyticsStore := mocks.NewMockServiceAnalyticsStore(t)

	queryStore.On("ListDeployments", context.Background(), int64(202), "dev", "svc-a").Return([]domain.DeploymentRow{
		{Service: "svc-a"},
		{Service: "svc-b"},
	}, nil)

	metadataStore.On("ListRequiredFields", context.Background(), int64(202)).Return([]ports.RequiredField{
		{Label: "team", Filterable: true},
	}, nil)

	metadataStore.On("ListServiceMetadataValuesByOrganization", context.Background(), int64(202)).Return([]ports.ServiceMetadataValue{
		{ServiceName: "svc-a", Label: "team", Value: "Platform"},
		{ServiceName: "svc-b", Label: "team", Value: "Ops"},
	}, nil)

	svc := NewServiceReadService(queryStore, metadataStore, analyticsStore)
	rows, options, err := svc.GetDeployments(context.Background(), 202, "dev", "svc-a")
	if err != nil {
		t.Fatalf("GetDeployments returned error: %v", err)
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
	queryStore := mocks.NewMockServiceQueryStore(t)
	metadataStore := mocks.NewMockServiceMetadataStore(t)
	analyticsStore := mocks.NewMockServiceAnalyticsStore(t)

	queryStore.On("GetServiceLatest", context.Background(), int64(303), "svc/a").Return(ports.ServiceLatest{Name: "svc/a", IntegrationType: "argocd"}, nil)
	queryStore.On("ListServiceEnvironments", context.Background(), int64(303), "svc/a").Return([]domain.ServiceEnvironment{
		{Name: "staging"},
		{Name: "dev"},
		{Name: "prod"},
	}, nil)
	queryStore.On("ListDeploymentHistory", context.Background(), int64(303), "svc/a", int64(200)).Return([]domain.DeploymentRecord{{Ref: "abc123"}}, nil)
	queryStore.On("ListServiceDependencies", context.Background(), int64(303), "svc/a").Return([]string{"db", "redis"}, nil)
	queryStore.On("ListServiceDependants", context.Background(), int64(303), "svc/a").Return([]string{"api"}, nil)

	metadataStore.On("ListRequiredFields", context.Background(), int64(303)).Return([]ports.RequiredField{
		{Label: "team", Filterable: true},
		{Label: "owner", Filterable: false},
	}, nil)

	metadataStore.On("ListServiceMetadata", context.Background(), int64(303), "svc/a").Return([]ports.MetadataValue{{Label: "team", Value: "platform"}}, nil)

	metadataStore.On("ListEnvironmentPriorities", context.Background(), int64(303)).Return([]string{"prod", "staging"}, nil)

	analyticsStore.On("GetServiceCurrentState", context.Background(), int64(303), "svc/a").Return(ports.ServiceCurrentState{}, nil)
	analyticsStore.On("GetServiceDeliveryStats30d", context.Background(), int64(303), "svc/a").Return(ports.ServiceDeliveryStats{}, nil)
	analyticsStore.On("ListServiceChangeLinksRecent", context.Background(), int64(303), "svc/a", int64(20)).Return(nil, nil)

	svc := NewServiceReadService(queryStore, metadataStore, analyticsStore)
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
	if len(detail.Dependencies) != 2 || detail.Dependencies[0] != "db" || detail.Dependencies[1] != "redis" {
		t.Fatalf("unexpected dependencies: %+v", detail.Dependencies)
	}
	if len(detail.Dependants) != 1 || detail.Dependants[0] != "api" {
		t.Fatalf("unexpected dependants: %+v", detail.Dependants)
	}
}
