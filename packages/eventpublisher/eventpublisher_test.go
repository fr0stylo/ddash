package eventpublisher

import "testing"

func TestBuildEventBody(t *testing.T) {
	_, resolved, err := BuildEventBody(Event{
		Type:        "service.deployed",
		Source:      "ci/test",
		Service:     "billing",
		Environment: "staging",
	})
	if err != nil {
		t.Fatalf("BuildEventBody returned error: %v", err)
	}
	if resolved == "" {
		t.Fatal("expected resolved type")
	}
}
