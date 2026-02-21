package githubbridge

import "testing"

func TestConvertReleasePublished(t *testing.T) {
	payload := []byte(`{"action":"published","repository":{"name":"orders"},"release":{"tag_name":"v1.2.3","html_url":"https://github.com/acme/orders/releases/tag/v1.2.3"},"sender":{"login":"octocat"}}`)
	events, err := Convert("release", "d-1", payload, ConvertConfig{DefaultEnvironment: "production", Source: "github/app"})
	if err != nil {
		t.Fatalf("convert: %v", err)
	}
	if len(events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(events))
	}
	if events[0].Type != "service.published" {
		t.Fatalf("unexpected type: %s", events[0].Type)
	}
}

func TestConvertWorkflowRunCompletedFailure(t *testing.T) {
	payload := []byte(`{"action":"completed","repository":{"name":"billing"},"workflow_run":{"id":42,"conclusion":"failure","head_sha":"abcdef1234567890","html_url":"https://github.com/acme/billing/actions/runs/42"},"sender":{"login":"ci-bot"}}`)
	events, err := Convert("workflow_run", "d-2", payload, ConvertConfig{DefaultEnvironment: "staging", Source: "github/app"})
	if err != nil {
		t.Fatalf("convert: %v", err)
	}
	if len(events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(events))
	}
	if events[0].Type != "dev.cdevents.pipeline.run.failed.0.3.0" {
		t.Fatalf("unexpected type: %s", events[0].Type)
	}
	if events[0].SubjectType != "pipeline" {
		t.Fatalf("unexpected subject type: %s", events[0].SubjectType)
	}
}

func TestConvertUnsupportedEventIgnored(t *testing.T) {
	events, err := Convert("issues", "d-3", []byte(`{"action":"opened"}`), ConvertConfig{})
	if err != nil {
		t.Fatalf("convert: %v", err)
	}
	if len(events) != 0 {
		t.Fatalf("expected 0 events, got %d", len(events))
	}
}
