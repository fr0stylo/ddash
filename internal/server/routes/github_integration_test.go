package routes

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync"
	"testing"

	"github.com/labstack/echo/v4"

	"github.com/fr0stylo/ddash/internal/app/ports"
	"github.com/fr0stylo/ddash/internal/renderer"
)

func TestHandleGitHubIntegrationRequestsOrgScopedMappings(t *testing.T) {
	initAuthStoreForTests()
	e := echo.New()
	e.Renderer = &renderer.Renderer{}

	var (
		gotOrgID string
		gotAuth  string
		mu       sync.Mutex
	)
	ingestor := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/mappings" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		mu.Lock()
		gotOrgID = strings.TrimSpace(r.URL.Query().Get("org_id"))
		gotAuth = strings.TrimSpace(r.Header.Get("Authorization"))
		mu.Unlock()
		_ = json.NewEncoder(w).Encode(map[string]any{
			"mappings": []map[string]any{{
				"installation_id":     123,
				"organization_id":     1,
				"organization_label":  "org-a",
				"ddash_endpoint":      "https://ddash.example.com",
				"default_environment": "production",
				"enabled":             true,
			}},
		})
	}))
	defer ingestor.Close()

	store := &orgRouteStoreFake{
		org: ports.Organization{ID: 1, Name: "org-a", AuthToken: "ddash-auth", WebhookSecret: "ddash-secret", Enabled: true},
	}
	v := NewViewRoutes(store, nil, ViewExternalConfig{
		PublicURL:           "https://ddash.example.com",
		GitHubIngestorURL:   ingestor.URL,
		GitHubIngestorToken: "setup-token",
	})

	c, rec := newAuthedContext(t, e, http.MethodGet, "/settings/integrations/github", url.Values{})
	c.Set("csrf", "csrf-token")
	if err := v.handleGitHubIntegration(c); err != nil {
		t.Fatalf("handler error: %v", err)
	}
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "123") {
		t.Fatalf("expected rendered mapping installation ID, body=%q", rec.Body.String())
	}

	mu.Lock()
	defer mu.Unlock()
	if gotOrgID != "1" {
		t.Fatalf("expected org_id=1, got %q", gotOrgID)
	}
	if gotAuth != "Bearer setup-token" {
		t.Fatalf("expected bearer auth header, got %q", gotAuth)
	}
}

func TestHandleGitHubIntegrationLinkSendsOrganizationAndCredentialPayload(t *testing.T) {
	initAuthStoreForTests()
	e := echo.New()
	e.Renderer = &renderer.Renderer{}

	type setupStartRequest struct {
		OrganizationID     int64  `json:"organization_id"`
		OrganizationLabel  string `json:"organization_label"`
		DDashEndpoint      string `json:"ddash_endpoint"`
		DDashAuthToken     string `json:"ddash_auth_token"`
		DDashWebhookSecret string `json:"ddash_webhook_secret"`
		DefaultEnvironment string `json:"default_environment"`
	}
	var (
		captured setupStartRequest
		mu       sync.Mutex
	)
	ingestor := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/setup/start" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		if err := json.NewDecoder(r.Body).Decode(&captured); err != nil {
			t.Fatalf("decode payload: %v", err)
		}
		mu.Lock()
		_ = json.NewEncoder(w).Encode(map[string]any{"redirect_url": "/setup/callback?state=abc"})
		mu.Unlock()
	}))
	defer ingestor.Close()

	store := &orgRouteStoreFake{
		org: ports.Organization{ID: 1, Name: "org-a", AuthToken: "ddash-auth", WebhookSecret: "ddash-secret", Enabled: true},
	}
	v := NewViewRoutes(store, nil, ViewExternalConfig{
		PublicURL:           "https://ddash.example.com",
		GitHubIngestorURL:   ingestor.URL,
		GitHubIngestorToken: "setup-token",
	})

	form := url.Values{}
	form.Set("default_environment", "staging")
	c, rec := newAuthedContext(t, e, http.MethodPost, "/settings/integrations/github/link", form)
	if err := v.handleGitHubIntegrationLink(c); err != nil {
		t.Fatalf("handler error: %v", err)
	}
	if rec.Code != http.StatusFound {
		t.Fatalf("expected redirect, got %d", rec.Code)
	}
	if rec.Header().Get("Location") != ingestor.URL+"/setup/callback?state=abc" {
		t.Fatalf("unexpected redirect location: %q", rec.Header().Get("Location"))
	}

	mu.Lock()
	defer mu.Unlock()
	if captured.OrganizationID != 1 {
		t.Fatalf("expected organization_id=1, got %d", captured.OrganizationID)
	}
	if captured.OrganizationLabel != "org-a" {
		t.Fatalf("unexpected organization_label: %q", captured.OrganizationLabel)
	}
	if captured.DDashEndpoint != "https://ddash.example.com" {
		t.Fatalf("unexpected ddash_endpoint: %q", captured.DDashEndpoint)
	}
	if captured.DDashAuthToken != "ddash-auth" {
		t.Fatalf("unexpected ddash_auth_token: %q", captured.DDashAuthToken)
	}
	if captured.DDashWebhookSecret != "ddash-secret" {
		t.Fatalf("unexpected ddash_webhook_secret: %q", captured.DDashWebhookSecret)
	}
	if captured.DefaultEnvironment != "staging" {
		t.Fatalf("unexpected default_environment: %q", captured.DefaultEnvironment)
	}
}

func TestHandleGitHubIntegrationDeleteSendsOrganizationScopedDelete(t *testing.T) {
	initAuthStoreForTests()
	e := echo.New()
	e.Renderer = &renderer.Renderer{}

	var (
		installationID int64
		organizationID int64
		mu             sync.Mutex
	)
	ingestor := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/mappings/delete" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		var payload struct {
			InstallationID int64 `json:"installation_id"`
			OrganizationID int64 `json:"organization_id"`
		}
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Fatalf("decode payload: %v", err)
		}
		mu.Lock()
		installationID = payload.InstallationID
		organizationID = payload.OrganizationID
		mu.Unlock()
		w.WriteHeader(http.StatusOK)
	}))
	defer ingestor.Close()

	store := &orgRouteStoreFake{
		org: ports.Organization{ID: 1, Name: "org-a", AuthToken: "ddash-auth", WebhookSecret: "ddash-secret", Enabled: true},
	}
	v := NewViewRoutes(store, nil, ViewExternalConfig{
		PublicURL:           "https://ddash.example.com",
		GitHubIngestorURL:   ingestor.URL,
		GitHubIngestorToken: "setup-token",
	})

	form := url.Values{}
	form.Set("installation_id", "123")
	c, rec := newAuthedContext(t, e, http.MethodPost, "/settings/integrations/github/delete", form)
	if err := v.handleGitHubIntegrationDelete(c); err != nil {
		t.Fatalf("handler error: %v", err)
	}
	if rec.Code != http.StatusFound {
		t.Fatalf("expected redirect, got %d", rec.Code)
	}

	mu.Lock()
	defer mu.Unlock()
	if installationID != 123 {
		t.Fatalf("expected installation_id=123, got %d", installationID)
	}
	if organizationID != 1 {
		t.Fatalf("expected organization_id=1, got %d", organizationID)
	}
}
