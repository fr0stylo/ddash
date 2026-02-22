package routes

import (
	"net/http"
	"net/url"
	"strings"
	"testing"

	"github.com/labstack/echo/v4"

	"github.com/fr0stylo/ddash/apps/ddash/internal/app/ports"
	"github.com/fr0stylo/ddash/apps/ddash/internal/renderer"
)

func TestHandleGitHubIntegrationRequestsOrgScopedMappings(t *testing.T) {
	initAuthStoreForTests()
	e := echo.New()
	e.Renderer = &renderer.Renderer{}

	store := &orgRouteStoreFake{
		org: ports.Organization{ID: 1, Name: "org-a", AuthToken: "ddash-auth", WebhookSecret: "ddash-secret", Enabled: true},
		githubMappings: []ports.GitHubInstallationMapping{{
			InstallationID:     123,
			OrganizationID:     1,
			OrganizationLabel:  "org-a",
			DefaultEnvironment: "production",
			Enabled:            true,
		}},
	}
	v := NewViewRoutes(store, nil, store, ViewExternalConfig{
		PublicURL:           "https://ddash.example.com",
		GitHubAppInstallURL: "https://github.com/apps/ddash/installations/new",
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
}

func TestHandleGitHubIntegrationLinkCreatesSetupStateAndRedirects(t *testing.T) {
	initAuthStoreForTests()
	e := echo.New()
	e.Renderer = &renderer.Renderer{}

	store := &orgRouteStoreFake{
		org: ports.Organization{ID: 1, Name: "org-a", AuthToken: "ddash-auth", WebhookSecret: "ddash-secret", Enabled: true},
	}
	v := NewViewRoutes(store, nil, store, ViewExternalConfig{
		PublicURL:           "https://ddash.example.com",
		GitHubAppInstallURL: "https://github.com/apps/ddash/installations/new",
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
	if !strings.Contains(rec.Header().Get("Location"), "github.com/apps/ddash/installations/new") {
		t.Fatalf("unexpected redirect location: %q", rec.Header().Get("Location"))
	}
	if len(store.setupIntents) != 1 {
		t.Fatalf("expected one setup intent, got %d", len(store.setupIntents))
	}
	for _, intent := range store.setupIntents {
		if intent.OrganizationID != 1 || intent.OrganizationLabel != "org-a" || intent.DefaultEnvironment != "staging" {
			t.Fatalf("unexpected setup intent: %+v", intent)
		}
	}
}

func TestHandleGitHubIntegrationDeleteRemovesLocalMapping(t *testing.T) {
	initAuthStoreForTests()
	e := echo.New()
	e.Renderer = &renderer.Renderer{}

	store := &orgRouteStoreFake{
		org: ports.Organization{ID: 1, Name: "org-a", AuthToken: "ddash-auth", WebhookSecret: "ddash-secret", Enabled: true},
	}
	v := NewViewRoutes(store, nil, store, ViewExternalConfig{
		PublicURL:           "https://ddash.example.com",
		GitHubAppInstallURL: "https://github.com/apps/ddash/installations/new",
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

	if store.deletedInstall != 123 {
		t.Fatalf("expected installation_id=123 deleted, got %d", store.deletedInstall)
	}
}
