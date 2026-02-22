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
)

type dependencyReadStoreFake struct {
	upsertService   string
	upsertDependsOn string
	deleteService   string
	deleteDependsOn string
}

func (f *dependencyReadStoreFake) ListServiceInstances(context.Context, int64, string) ([]domain.Service, error) {
	return nil, nil
}

func (f *dependencyReadStoreFake) ListDeployments(context.Context, int64, string, string) ([]domain.DeploymentRow, error) {
	return nil, nil
}

func (f *dependencyReadStoreFake) GetServiceLatest(context.Context, int64, string) (ports.ServiceLatest, error) {
	return ports.ServiceLatest{}, nil
}

func (f *dependencyReadStoreFake) ListServiceEnvironments(context.Context, int64, string) ([]domain.ServiceEnvironment, error) {
	return nil, nil
}

func (f *dependencyReadStoreFake) ListDeploymentHistory(context.Context, int64, string, int64) ([]domain.DeploymentRecord, error) {
	return nil, nil
}

func (f *dependencyReadStoreFake) ListServiceDependencies(context.Context, int64, string) ([]string, error) {
	return nil, nil
}

func (f *dependencyReadStoreFake) ListServiceDependants(context.Context, int64, string) ([]string, error) {
	return nil, nil
}

func (f *dependencyReadStoreFake) UpsertServiceDependency(_ context.Context, _ int64, serviceName, dependsOnServiceName string) error {
	f.upsertService = serviceName
	f.upsertDependsOn = dependsOnServiceName
	return nil
}

func (f *dependencyReadStoreFake) DeleteServiceDependency(_ context.Context, _ int64, serviceName, dependsOnServiceName string) error {
	f.deleteService = serviceName
	f.deleteDependsOn = dependsOnServiceName
	return nil
}

func (f *dependencyReadStoreFake) GetOrganizationRenderVersion(context.Context, int64) (int64, error) {
	return 1, nil
}

func (f *dependencyReadStoreFake) ListRequiredFields(context.Context, int64) ([]ports.RequiredField, error) {
	return nil, nil
}

func (f *dependencyReadStoreFake) ListServiceMetadata(context.Context, int64, string) ([]ports.MetadataValue, error) {
	return nil, nil
}

func (f *dependencyReadStoreFake) ListServiceMetadataValuesByOrganization(context.Context, int64) ([]ports.ServiceMetadataValue, error) {
	return nil, nil
}

func (f *dependencyReadStoreFake) ListEnvironmentPriorities(context.Context, int64) ([]string, error) {
	return nil, nil
}

func (f *dependencyReadStoreFake) ListDiscoveredEnvironments(context.Context, int64) ([]string, error) {
	return nil, nil
}

func (f *dependencyReadStoreFake) GetServiceCurrentState(context.Context, int64, string) (ports.ServiceCurrentState, error) {
	return ports.ServiceCurrentState{}, nil
}

func (f *dependencyReadStoreFake) GetServiceDeliveryStats30d(context.Context, int64, string) (ports.ServiceDeliveryStats, error) {
	return ports.ServiceDeliveryStats{}, nil
}

func (f *dependencyReadStoreFake) ListServiceChangeLinksRecent(context.Context, int64, string, int64) ([]ports.ServiceChangeLink, error) {
	return nil, nil
}

func TestHandleServiceDependencyUpsert(t *testing.T) {
	initAuthStoreForTests()
	e := echo.New()

	store := &orgRouteStoreFake{org: ports.Organization{ID: 1, Name: "org-a", Enabled: true}}
	readStore := &dependencyReadStoreFake{}
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
	if readStore.upsertService != "orders" || readStore.upsertDependsOn != "billing" {
		t.Fatalf("unexpected upsert payload service=%q depends_on=%q", readStore.upsertService, readStore.upsertDependsOn)
	}
	location := rec.Header().Get("Location")
	if !strings.Contains(location, "Dependency+added") {
		t.Fatalf("expected success message in redirect, got %q", location)
	}
}

func TestHandleServiceDependencyDelete(t *testing.T) {
	initAuthStoreForTests()
	e := echo.New()

	store := &orgRouteStoreFake{org: ports.Organization{ID: 1, Name: "org-a", Enabled: true}}
	readStore := &dependencyReadStoreFake{}
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
	if readStore.deleteService != "orders" || readStore.deleteDependsOn != "billing" {
		t.Fatalf("unexpected delete payload service=%q depends_on=%q", readStore.deleteService, readStore.deleteDependsOn)
	}
	location := rec.Header().Get("Location")
	if !strings.Contains(location, "Dependency+removed") {
		t.Fatalf("expected success message in redirect, got %q", location)
	}
}

func TestHandleServiceDependencyUpsertFeatureDisabled(t *testing.T) {
	initAuthStoreForTests()
	e := echo.New()

	store := &orgRouteStoreFake{
		org:      ports.Organization{ID: 1, Name: "org-a", Enabled: true},
		features: []ports.OrganizationFeature{{Key: "show_service_dependencies", Enabled: false}},
	}
	readStore := &dependencyReadStoreFake{}
	v := NewViewRoutes(store, readStore, store, ViewExternalConfig{})

	form := url.Values{}
	form.Set("depends_on", "billing")
	c, rec := newAuthedContext(t, e, http.MethodPost, "/s/orders/dependencies", form)
	c.SetParamNames("name")
	c.SetParamValues("orders")

	if err := v.handleServiceDependencyUpsert(c); err != nil {
		t.Fatalf("handler error: %v", err)
	}
	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected forbidden, got %d", rec.Code)
	}
	if readStore.upsertService != "" || readStore.upsertDependsOn != "" {
		t.Fatalf("expected no write when feature disabled")
	}
}
