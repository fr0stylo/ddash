package services

import (
	"context"
	"testing"

	"github.com/stretchr/testify/mock"

	"github.com/fr0stylo/ddash/internal/app/domain"
	"github.com/fr0stylo/ddash/internal/app/ports"
	portmocks "github.com/fr0stylo/ddash/internal/app/ports/mocks"
)

func TestServiceReadService_GetServiceDetail_UsesSeparatedStores(t *testing.T) {
	queryStore := portmocks.NewMockServiceQueryStore(t)
	metadataStore := portmocks.NewMockServiceMetadataStore(t)
	analyticsStore := portmocks.NewMockServiceAnalyticsStore(t)

	svc := NewServiceReadService(queryStore, metadataStore, analyticsStore)

	queryStore.EXPECT().GetServiceLatest(mock.Anything, int64(11), "orders").Return(ports.ServiceLatest{Name: "orders", IntegrationType: "cdevents"}, nil)
	metadataStore.EXPECT().ListRequiredFields(mock.Anything, int64(11)).Return([]ports.RequiredField{{Label: "team", Filterable: true}}, nil)
	metadataStore.EXPECT().ListServiceMetadata(mock.Anything, int64(11), "orders").Return([]ports.MetadataValue{{Label: "team", Value: "platform"}}, nil)
	queryStore.EXPECT().ListServiceEnvironments(mock.Anything, int64(11), "orders").Return([]domain.ServiceEnvironment{{Name: "staging", LastDeploy: "2026-02-21 10:00"}}, nil)
	metadataStore.EXPECT().ListEnvironmentPriorities(mock.Anything, int64(11)).Return([]string{"staging"}, nil)
	queryStore.EXPECT().ListDeploymentHistory(mock.Anything, int64(11), "orders", int64(200)).Return([]domain.DeploymentRecord{{Ref: "pkg:generic/orders@v1", Environment: "staging", DeployedAt: "2026-02-21 10:00"}}, nil)
	analyticsStore.EXPECT().GetServiceCurrentState(mock.Anything, int64(11), "orders").Return(ports.ServiceCurrentState{LastStatus: "synced", DriftCount: 1, FailedStreak: 0}, nil)
	analyticsStore.EXPECT().GetServiceDeliveryStats30d(mock.Anything, int64(11), "orders").Return(ports.ServiceDeliveryStats{Success30d: 8, Failures30d: 1, Rollbacks30d: 1}, nil)
	analyticsStore.EXPECT().ListServiceChangeLinksRecent(mock.Anything, int64(11), "orders", int64(20)).Return([]ports.ServiceChangeLink{{Environment: "staging", ArtifactID: "pkg:generic/orders@v1"}}, nil)

	detail, err := svc.GetServiceDetail(context.Background(), 11, "orders")
	if err != nil {
		t.Fatalf("GetServiceDetail returned error: %v", err)
	}

	if detail.LastStatus != "synced" {
		t.Fatalf("expected synced status, got %q", detail.LastStatus)
	}
	if detail.ChangeFailureRate != "20%" {
		t.Fatalf("expected 20%% change failure rate, got %q", detail.ChangeFailureRate)
	}
	if len(detail.RiskEvents) != 1 {
		t.Fatalf("expected 1 risk event, got %d", len(detail.RiskEvents))
	}
}

func TestServiceReadService_GetServicesByEnv_UsesQueryAndMetadataStores(t *testing.T) {
	queryStore := portmocks.NewMockServiceQueryStore(t)
	metadataStore := portmocks.NewMockServiceMetadataStore(t)
	analyticsStore := portmocks.NewMockServiceAnalyticsStore(t)

	svc := NewServiceReadService(queryStore, metadataStore, analyticsStore)

	queryStore.EXPECT().ListServiceInstances(mock.Anything, int64(22), "prod").Return([]domain.Service{{Title: "billing"}}, nil)
	metadataStore.EXPECT().ListRequiredFields(mock.Anything, int64(22)).Return([]ports.RequiredField{{Label: "team", Filterable: true}}, nil)
	metadataStore.EXPECT().ListServiceMetadataValuesByOrganization(mock.Anything, int64(22)).Return([]ports.ServiceMetadataValue{{ServiceName: "billing", Label: "team", Value: "platform"}}, nil)

	rows, err := svc.GetServicesByEnv(context.Background(), 22, "prod")
	if err != nil {
		t.Fatalf("GetServicesByEnv returned error: %v", err)
	}
	if len(rows) != 1 {
		t.Fatalf("expected 1 row, got %d", len(rows))
	}
	if rows[0].MetadataTags != "|team:platform|" {
		t.Fatalf("unexpected metadata tags: %q", rows[0].MetadataTags)
	}
}
