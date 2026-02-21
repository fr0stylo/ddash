package routes

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestGitHubIngestorClientStartInstallResolvesRelativeRedirect(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/setup/start" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer token" {
			t.Fatalf("unexpected auth header: %s", got)
		}
		_ = json.NewEncoder(w).Encode(map[string]string{"redirect_url": "/setup/callback?state=abc"})
	}))
	defer srv.Close()

	client := NewGitHubIngestorClient(srv.URL, "token", "https://ddash.example.com")
	redirectURL, err := client.StartInstall(context.Background(), 11, "org", "auth", "secret", "production")
	if err != nil {
		t.Fatalf("StartInstall error = %v", err)
	}
	if redirectURL != srv.URL+"/setup/callback?state=abc" {
		t.Fatalf("unexpected redirect URL: %s", redirectURL)
	}
}

func TestGitHubIngestorClientListMappingsIncludesOrgFilter(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.URL.Query().Get("org_id"); got != "42" {
			t.Fatalf("unexpected org_id: %s", got)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"mappings": []map[string]any{}})
	}))
	defer srv.Close()

	client := NewGitHubIngestorClient(srv.URL, "token", "https://ddash.example.com")
	_, err := client.ListMappings(context.Background(), 42)
	if err != nil {
		t.Fatalf("ListMappings error = %v", err)
	}
}

func TestGitHubIngestorClientDeleteMappingIncludesOrganizationID(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/mappings/delete" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		var payload map[string]any
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Fatalf("decode payload: %v", err)
		}
		if payload["organization_id"] != float64(7) {
			t.Fatalf("unexpected organization_id payload: %#v", payload["organization_id"])
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	client := NewGitHubIngestorClient(srv.URL, "token", "https://ddash.example.com")
	if err := client.DeleteMapping(context.Background(), 99, 7); err != nil {
		t.Fatalf("DeleteMapping error = %v", err)
	}
}
