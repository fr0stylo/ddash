package routes

import (
	"strings"
	"testing"
)

func TestGitHubIngestorClientStartInstallAppendsState(t *testing.T) {
	t.Parallel()

	client := NewGitHubIngestorClient("https://github.com/apps/my-app/installations/new", "token", "")
	redirectURL, err := client.StartInstall("abc")
	if err != nil {
		t.Fatalf("StartInstall error = %v", err)
	}
	if !strings.Contains(redirectURL, "state=abc") {
		t.Fatalf("unexpected redirect URL: %s", redirectURL)
	}
}
