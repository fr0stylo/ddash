package routes

import (
	"context"
	"net/http"
	"net/url"
	"strings"
	"testing"

	"github.com/labstack/echo/v4"

	"github.com/fr0stylo/ddash/apps/ddash/internal/app/domain"
	"github.com/fr0stylo/ddash/apps/ddash/internal/app/ports"
	"github.com/fr0stylo/ddash/apps/ddash/internal/app/ports/mocks"
	mock "github.com/stretchr/testify/mock"
)

type mockServiceReadStore struct {
	*mockServiceQueryStoreWrapper
	*mockServiceMetadataStoreWrapper
	*mockServiceAnalyticsStoreWrapper
}

type mockServiceQueryStoreWrapper struct {
	*mocks.MockServiceQueryStore
}

type mockServiceMetadataStoreWrapper struct {
	*mocks.MockServiceMetadataStore
}

type mockServiceAnalyticsStoreWrapper struct {
	*mocks.MockServiceAnalyticsStore
}

func newMockServiceReadStore(t interface {
	mock.TestingT
	Cleanup(func())
}) *mockServiceReadStore {
	return &mockServiceReadStore{
		mockServiceQueryStoreWrapper:     &mockServiceQueryStoreWrapper{mocks.NewMockServiceQueryStore(t)},
		mockServiceMetadataStoreWrapper:  &mockServiceMetadataStoreWrapper{mocks.NewMockServiceMetadataStore(t)},
		mockServiceAnalyticsStoreWrapper: &mockServiceAnalyticsStoreWrapper{mocks.NewMockServiceAnalyticsStore(t)},
	}
}

func (m *mockServiceReadStore) ListServiceInstances(ctx context.Context, organizationID int64, env string) ([]domain.Service, error) {
	return m.MockServiceQueryStore.ListServiceInstances(ctx, organizationID, env)
}

func (m *mockServiceReadStore) ListDeployments(ctx context.Context, organizationID int64, env, service string) ([]domain.DeploymentRow, error) {
	return m.MockServiceQueryStore.ListDeployments(ctx, organizationID, env, service)
}

func (m *mockServiceReadStore) GetServiceLatest(ctx context.Context, organizationID int64, name string) (ports.ServiceLatest, error) {
	return m.MockServiceQueryStore.GetServiceLatest(ctx, organizationID, name)
}

func (m *mockServiceReadStore) ListServiceEnvironments(ctx context.Context, organizationID int64, service string) ([]domain.ServiceEnvironment, error) {
	return m.MockServiceQueryStore.ListServiceEnvironments(ctx, organizationID, service)
}

func (m *mockServiceReadStore) ListDeploymentHistory(ctx context.Context, organizationID int64, service string, limit int64) ([]domain.DeploymentRecord, error) {
	return m.MockServiceQueryStore.ListDeploymentHistory(ctx, organizationID, service, limit)
}

func (m *mockServiceReadStore) ListServiceDependencies(ctx context.Context, organizationID int64, service string) ([]string, error) {
	return m.MockServiceQueryStore.ListServiceDependencies(ctx, organizationID, service)
}

func (m *mockServiceReadStore) ListServiceDependants(ctx context.Context, organizationID int64, service string) ([]string, error) {
	return m.MockServiceQueryStore.ListServiceDependants(ctx, organizationID, service)
}

func (m *mockServiceReadStore) UpsertServiceDependency(ctx context.Context, organizationID int64, serviceName, dependsOnServiceName string) error {
	return m.MockServiceQueryStore.UpsertServiceDependency(ctx, organizationID, serviceName, dependsOnServiceName)
}

func (m *mockServiceReadStore) DeleteServiceDependency(ctx context.Context, organizationID int64, serviceName, dependsOnServiceName string) error {
	return m.MockServiceQueryStore.DeleteServiceDependency(ctx, organizationID, serviceName, dependsOnServiceName)
}

func (m *mockServiceReadStore) GetOrganizationRenderVersion(ctx context.Context, organizationID int64) (int64, error) {
	return m.MockServiceQueryStore.GetOrganizationRenderVersion(ctx, organizationID)
}

func (m *mockServiceReadStore) ListRequiredFields(ctx context.Context, organizationID int64) ([]ports.RequiredField, error) {
	return m.MockServiceMetadataStore.ListRequiredFields(ctx, organizationID)
}

func (m *mockServiceReadStore) ListServiceMetadata(ctx context.Context, organizationID int64, service string) ([]ports.MetadataValue, error) {
	return m.MockServiceMetadataStore.ListServiceMetadata(ctx, organizationID, service)
}

func (m *mockServiceReadStore) ListServiceMetadataValuesByOrganization(ctx context.Context, organizationID int64) ([]ports.ServiceMetadataValue, error) {
	return m.MockServiceMetadataStore.ListServiceMetadataValuesByOrganization(ctx, organizationID)
}

func (m *mockServiceReadStore) ListEnvironmentPriorities(ctx context.Context, organizationID int64) ([]string, error) {
	return m.MockServiceMetadataStore.ListEnvironmentPriorities(ctx, organizationID)
}

func (m *mockServiceReadStore) ListDiscoveredEnvironments(ctx context.Context, organizationID int64) ([]string, error) {
	return m.MockServiceMetadataStore.ListDiscoveredEnvironments(ctx, organizationID)
}

func (m *mockServiceReadStore) GetServiceCurrentState(ctx context.Context, organizationID int64, service string) (ports.ServiceCurrentState, error) {
	return m.MockServiceAnalyticsStore.GetServiceCurrentState(ctx, organizationID, service)
}

func (m *mockServiceReadStore) GetServiceDeliveryStats30d(ctx context.Context, organizationID int64, service string) (ports.ServiceDeliveryStats, error) {
	return m.MockServiceAnalyticsStore.GetServiceDeliveryStats30d(ctx, organizationID, service)
}

func (m *mockServiceReadStore) ListServiceChangeLinksRecent(ctx context.Context, organizationID int64, service string, limit int64) ([]ports.ServiceChangeLink, error) {
	return m.MockServiceAnalyticsStore.ListServiceChangeLinksRecent(ctx, organizationID, service, limit)
}

func (m *mockServiceReadStore) ListServiceLeadTimeSamples(ctx context.Context, organizationID int64, sinceMs int64) ([]ports.ServiceLeadTimeSample, error) {
	return m.MockServiceAnalyticsStore.ListServiceLeadTimeSamples(ctx, organizationID, sinceMs)
}

func (m *mockServiceReadStore) GetPipelineStats30d(ctx context.Context, organizationID int64, service string) (ports.PipelineStats, error) {
	return m.MockServiceAnalyticsStore.GetPipelineStats30d(ctx, organizationID, service)
}

func (m *mockServiceReadStore) GetDeploymentDurationStats(ctx context.Context, organizationID int64, service string, environment string, sinceMs int64) (ports.DeploymentDurationStats, error) {
	return m.MockServiceAnalyticsStore.GetDeploymentDurationStats(ctx, organizationID, service, environment, sinceMs)
}

func (m *mockServiceReadStore) GetEnvironmentDriftCount(ctx context.Context, organizationID int64, service string, sinceMs int64) (int64, error) {
	return m.MockServiceAnalyticsStore.GetEnvironmentDriftCount(ctx, organizationID, service, sinceMs)
}

func (m *mockServiceReadStore) ListEnvironmentDrifts(ctx context.Context, organizationID int64, service string, limit int64) ([]ports.EnvironmentDrift, error) {
	return m.MockServiceAnalyticsStore.ListEnvironmentDrifts(ctx, organizationID, service, limit)
}

func (m *mockServiceReadStore) GetRedeploymentRate30d(ctx context.Context, organizationID int64, service string) (ports.RedeploymentRate, error) {
	return m.MockServiceAnalyticsStore.GetRedeploymentRate30d(ctx, organizationID, service)
}

func (m *mockServiceReadStore) GetThroughputStats(ctx context.Context, organizationID int64, service string) (ports.WeeklyThroughput, error) {
	return m.MockServiceAnalyticsStore.GetThroughputStats(ctx, organizationID, service)
}

func (m *mockServiceReadStore) ListWeeklyThroughput(ctx context.Context, organizationID int64, service string, limit int64) ([]ports.WeeklyThroughput, error) {
	return m.MockServiceAnalyticsStore.ListWeeklyThroughput(ctx, organizationID, service, limit)
}

func (m *mockServiceReadStore) GetArtifactAgeByEnvironment(ctx context.Context, organizationID int64, service string) ([]ports.ArtifactAge, error) {
	return m.MockServiceAnalyticsStore.GetArtifactAgeByEnvironment(ctx, organizationID, service)
}

func (m *mockServiceReadStore) GetMTTR(ctx context.Context, organizationID int64, sinceMs int64) (ports.MTTRStats, error) {
	return m.MockServiceAnalyticsStore.GetMTTR(ctx, organizationID, sinceMs)
}

func (m *mockServiceReadStore) ListIncidentLinks(ctx context.Context, organizationID int64, service string, limit int64) ([]ports.IncidentLink, error) {
	return m.MockServiceAnalyticsStore.ListIncidentLinks(ctx, organizationID, service, limit)
}

func (m *mockServiceReadStore) GetComprehensiveDeliveryMetrics(ctx context.Context, organizationID int64, sinceMs int64) (ports.ComprehensiveDeliveryMetrics, error) {
	return m.MockServiceAnalyticsStore.GetComprehensiveDeliveryMetrics(ctx, organizationID, sinceMs)
}

func TestHandleServiceDependencyUpsert(t *testing.T) {
	initAuthStoreForTests()
	e := echo.New()

	store := &orgRouteStoreFake{org: ports.Organization{ID: 1, Name: "org-a", Enabled: true}}
	readStore := newMockServiceReadStore(t)

	readStore.MockServiceQueryStore.On("UpsertServiceDependency", context.Background(), int64(1), "orders", "billing").Return(nil)

	v := NewViewRoutes(store, readStore, store, ViewExternalConfig{})

	form := url.Values{}
	form.Set("depends_on", "billing")
	c, rec := newAuthedContext(t, e, http.MethodPost, "/s/orders/dependencies", form)
	c.SetParamNames("name")
	c.SetParamValues("orders")

	if err := v.handleServiceDependencyUpsert(c); err != nil {
		t.Fatalf("handler error: %v", err)
	}
	if rec.Code != http.StatusFound {
		t.Fatalf("expected redirect, got %d", rec.Code)
	}
	location := rec.Header().Get("Location")
	if !strings.Contains(location, "Added+1+dependency") {
		t.Fatalf("expected success message in redirect, got %q", location)
	}

	readStore.MockServiceQueryStore.AssertCalled(t, "UpsertServiceDependency", context.Background(), int64(1), "orders", "billing")
}

func TestHandleServiceDependencyUpsertMultiple(t *testing.T) {
	initAuthStoreForTests()
	e := echo.New()

	store := &orgRouteStoreFake{org: ports.Organization{ID: 1, Name: "org-a", Enabled: true}}
	readStore := newMockServiceReadStore(t)

	readStore.MockServiceQueryStore.On("UpsertServiceDependency", context.Background(), int64(1), "orders", "billing").Return(nil).Once()
	readStore.MockServiceQueryStore.On("UpsertServiceDependency", context.Background(), int64(1), "orders", "auth").Return(nil).Once()

	v := NewViewRoutes(store, readStore, store, ViewExternalConfig{})

	form := url.Values{}
	form.Set("depends_on", "billing, auth, billing")
	c, rec := newAuthedContext(t, e, http.MethodPost, "/s/orders/dependencies", form)
	c.SetParamNames("name")
	c.SetParamValues("orders")

	if err := v.handleServiceDependencyUpsert(c); err != nil {
		t.Fatalf("handler error: %v", err)
	}
	if rec.Code != http.StatusFound {
		t.Fatalf("expected redirect, got %d", rec.Code)
	}
	location := rec.Header().Get("Location")
	if !strings.Contains(location, "Added+2+dependencies") {
		t.Fatalf("expected multi-add message in redirect, got %q", location)
	}
}

func TestHandleServiceDependencyDelete(t *testing.T) {
	initAuthStoreForTests()
	e := echo.New()

	store := &orgRouteStoreFake{org: ports.Organization{ID: 1, Name: "org-a", Enabled: true}}
	readStore := newMockServiceReadStore(t)

	readStore.MockServiceQueryStore.On("DeleteServiceDependency", context.Background(), int64(1), "orders", "billing").Return(nil)

	v := NewViewRoutes(store, readStore, store, ViewExternalConfig{})

	form := url.Values{}
	form.Set("depends_on", "billing")
	c, rec := newAuthedContext(t, e, http.MethodPost, "/s/orders/dependencies/delete", form)
	c.SetParamNames("name")
	c.SetParamValues("orders")

	if err := v.handleServiceDependencyDelete(c); err != nil {
		t.Fatalf("handler error: %v", err)
	}
	if rec.Code != http.StatusFound {
		t.Fatalf("expected redirect, got %d", rec.Code)
	}
	location := rec.Header().Get("Location")
	if !strings.Contains(location, "Dependency+removed") {
		t.Fatalf("expected success message in redirect, got %q", location)
	}

	readStore.MockServiceQueryStore.AssertCalled(t, "DeleteServiceDependency", context.Background(), int64(1), "orders", "billing")
}
