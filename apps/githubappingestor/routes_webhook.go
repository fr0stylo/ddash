package main

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/fr0stylo/ddash/apps/githubappingestor/internal/githubbridge"
	"github.com/fr0stylo/ddash/pkg/eventpublisher"
	gh "github.com/google/go-github/v81/github"
)

func (a ingestorRuntime) registerWebhookRoute(mux *http.ServeMux) {
	mux.HandleFunc(a.webhookPath, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		payload, err := io.ReadAll(io.LimitReader(r.Body, 1<<20))
		if err != nil {
			http.Error(w, "invalid payload", http.StatusBadRequest)
			return
		}
		if _, err := gh.ValidatePayloadFromBody(r.Header.Get("Content-Type"), bytes.NewReader(payload), a.githubSecret, []byte(r.Header.Get("X-Hub-Signature-256"))); err != nil {
			http.Error(w, "invalid signature", http.StatusUnauthorized)
			return
		}

		eventName := strings.TrimSpace(r.Header.Get("X-GitHub-Event"))
		deliveryID := strings.TrimSpace(r.Header.Get("X-GitHub-Delivery"))
		installationID, hasInstallation := extractInstallationIDFromGitHubEvent(eventName, payload)

		convertCfg := a.defaultConvertCfg

		if strings.TrimSpace(a.ddashEndpoint) == "" || strings.TrimSpace(a.ingestorToken) == "" {
			slog.Warn("ignoring webhook because DDash endpoint or setup token are missing", "installation_id", installationID, "event", eventName)
			w.WriteHeader(http.StatusAccepted)
			_, _ = w.Write([]byte("ignored"))
			return
		}
		if !hasInstallation {
			slog.Warn("ignoring webhook because installation id is missing", "event", eventName)
			w.WriteHeader(http.StatusAccepted)
			_, _ = w.Write([]byte("ignored"))
			return
		}

		events, err := githubbridge.Convert(eventName, deliveryID, payload, convertCfg)
		if err != nil {
			http.Error(w, "invalid github payload", http.StatusBadRequest)
			return
		}
		if len(events) == 0 {
			w.WriteHeader(http.StatusAccepted)
			_, _ = w.Write([]byte("ignored"))
			return
		}

		published := 0
		for _, event := range events {
			if err := publishGitHubAppEvent(r.Context(), a.ddashEndpoint, a.ingestorToken, installationID, eventName, deliveryID, event); err != nil {
				slog.Error("failed to publish converted event", "github_event", eventName, "installation_id", installationID, "error", err)
				http.Error(w, "publish failed", http.StatusBadGateway)
				return
			}
			published++
		}
		w.WriteHeader(http.StatusAccepted)
		_, _ = w.Write([]byte(fmt.Sprintf("published=%d", published)))
	})
}

func extractInstallationIDFromGitHubEvent(eventName string, payload []byte) (int64, bool) {
	event, err := gh.ParseWebHook(strings.TrimSpace(eventName), payload)
	if err != nil {
		return 0, false
	}
	switch typed := event.(type) {
	case *gh.ReleaseEvent:
		return typed.GetInstallation().GetID(), typed.GetInstallation().GetID() > 0
	case *gh.DeploymentStatusEvent:
		return typed.GetInstallation().GetID(), typed.GetInstallation().GetID() > 0
	case *gh.WorkflowRunEvent:
		return typed.GetInstallation().GetID(), typed.GetInstallation().GetID() > 0
	case *gh.PushEvent:
		return typed.GetInstallation().GetID(), typed.GetInstallation().GetID() > 0
	case *gh.PullRequestEvent:
		return typed.GetInstallation().GetID(), typed.GetInstallation().GetID() > 0
	default:
		return 0, false
	}
}

func publishGitHubAppEvent(ctx context.Context, endpoint, setupToken string, installationID int64, eventName, deliveryID string, event eventpublisher.Event) error {
	body, _, err := eventpublisher.BuildEventBody(event)
	if err != nil {
		return err
	}

	base := strings.TrimRight(strings.TrimSpace(endpoint), "/")
	if parsed, parseErr := url.Parse(base); parseErr == nil {
		path := strings.TrimSpace(parsed.Path)
		if strings.HasSuffix(path, "/webhooks/custom") {
			parsed.Path = strings.TrimSuffix(path, "/webhooks/custom")
			base = strings.TrimRight(parsed.String(), "/")
		}
		if strings.HasSuffix(path, "/webhooks/cdevents") {
			parsed.Path = strings.TrimSuffix(path, "/webhooks/cdevents")
			base = strings.TrimRight(parsed.String(), "/")
		}
	}

	requestURL := base + "/webhooks/github-app"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, requestURL, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+strings.TrimSpace(setupToken))
	req.Header.Set("X-GitHub-Installation-ID", strconv.FormatInt(installationID, 10))
	req.Header.Set("X-GitHub-Event", strings.TrimSpace(eventName))
	req.Header.Set("X-GitHub-Delivery", strings.TrimSpace(deliveryID))

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= http.StatusMultipleChoices {
		payload, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("webhook rejected: status=%s body=%s", resp.Status, strings.TrimSpace(string(payload)))
	}
	return nil
}
