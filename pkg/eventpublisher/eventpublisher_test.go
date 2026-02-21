package eventpublisher

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestBuildEventBodySupportsServiceTypes(t *testing.T) {
	types := []string{"service.deployed", "service.upgraded", "service.rolledback", "service.removed", "service.published"}
	for _, typ := range types {
		t.Run(typ, func(t *testing.T) {
			body, resolved, err := BuildEventBody(Event{
				Type:        typ,
				Source:      "ci/test",
				Service:     "billing-api",
				Environment: "staging",
			})
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if !strings.HasPrefix(resolved, "dev.cdevents.service.") {
				t.Fatalf("unexpected resolved type: %s", resolved)
			}
			var payload map[string]any
			if err := json.Unmarshal(body, &payload); err != nil {
				t.Fatalf("invalid json body: %v", err)
			}
		})
	}
}

func TestBuildEventBodySupportsEnvironmentTypes(t *testing.T) {
	types := []string{"environment.created", "environment.modified", "environment.deleted"}
	for _, typ := range types {
		t.Run(typ, func(t *testing.T) {
			body, resolved, err := BuildEventBody(Event{
				Type:        typ,
				Source:      "ci/test",
				Environment: "staging",
			})
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if !strings.HasPrefix(resolved, "dev.cdevents.environment.") {
				t.Fatalf("unexpected resolved type: %s", resolved)
			}
			var payload map[string]any
			if err := json.Unmarshal(body, &payload); err != nil {
				t.Fatalf("invalid json body: %v", err)
			}
		})
	}
}

func TestBuildEventBodySupportsAcceptedCustomPrefixes(t *testing.T) {
	body, resolved, err := BuildEventBody(Event{
		Type:        "dev.cdevents.pipeline.run.started.0.3.0",
		Source:      "ci/test",
		Service:     "payments",
		Environment: "staging",
		ActorName:   "build-bot",
		PipelineRun: "run-123",
		PipelineURL: "https://ci.example.local/runs/123",
		ChainID:     "chain-123",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resolved != "dev.cdevents.pipeline.run.started.0.3.0" {
		t.Fatalf("unexpected resolved type: %s", resolved)
	}
	var payload map[string]any
	if err := json.Unmarshal(body, &payload); err != nil {
		t.Fatalf("invalid json body: %v", err)
	}
	contextNode := payload["context"].(map[string]any)
	if contextNode["chainId"] != "chain-123" {
		t.Fatalf("expected chainId in payload")
	}
}

func TestClientPublishSendsSignedRequest(t *testing.T) {
	var gotAuth string
	var gotSignature string
	var gotContentType string
	var gotPath string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		gotSignature = r.Header.Get("X-Webhook-Signature")
		gotContentType = r.Header.Get("Content-Type")
		gotPath = r.URL.Path
		_, _ = io.Copy(io.Discard, r.Body)
		w.WriteHeader(http.StatusAccepted)
	}))
	defer server.Close()

	client := Client{
		Endpoint: server.URL,
		Token:    "token-123",
		Secret:   "secret-123",
	}

	resolvedType, err := client.Publish(context.Background(), Event{
		Type:        "service.deployed",
		Source:      "ci/test",
		Service:     "orders-api",
		Environment: "production",
		Artifact:    "pkg:generic/orders-api@abc123",
	})
	if err != nil {
		t.Fatalf("publish failed: %v", err)
	}
	if resolvedType != "dev.cdevents.service.deployed.0.3.0" {
		t.Fatalf("unexpected resolved type: %s", resolvedType)
	}
	if gotPath != "/webhooks/cdevents" {
		t.Fatalf("unexpected path: %s", gotPath)
	}
	if gotAuth != "Bearer token-123" {
		t.Fatalf("unexpected auth header: %s", gotAuth)
	}
	if gotSignature == "" {
		t.Fatalf("expected signature header to be set")
	}
	if gotContentType != "application/json" {
		t.Fatalf("unexpected content type: %s", gotContentType)
	}
}
