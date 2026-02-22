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

	"github.com/fr0stylo/ddash/apps/gitlabingestor/internal/gitlabbridge"
	"github.com/fr0stylo/ddash/pkg/eventpublisher"
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
		if strings.TrimSpace(r.Header.Get("X-Gitlab-Token")) != strings.TrimSpace(a.webhookToken) {
			http.Error(w, "invalid token", http.StatusUnauthorized)
			return
		}

		eventName := strings.TrimSpace(r.Header.Get("X-Gitlab-Event"))
		deliveryID := strings.TrimSpace(r.Header.Get("X-Gitlab-Event-UUID"))
		projectID, hasProject := gitlabbridge.ExtractProjectID(payload)

		if strings.TrimSpace(a.ddashEndpoint) == "" || strings.TrimSpace(a.ingestorToken) == "" {
			slog.Warn("ignoring webhook because DDash endpoint or setup token are missing", "project_id", projectID, "event", eventName)
			w.WriteHeader(http.StatusAccepted)
			_, _ = w.Write([]byte("ignored"))
			return
		}
		if !hasProject {
			slog.Warn("ignoring webhook because project id is missing", "event", eventName)
			w.WriteHeader(http.StatusAccepted)
			_, _ = w.Write([]byte("ignored"))
			return
		}

		events, err := gitlabbridge.Convert(eventName, deliveryID, payload, a.defaultConvertCfg)
		if err != nil {
			http.Error(w, "invalid gitlab payload", http.StatusBadRequest)
			return
		}
		if len(events) == 0 {
			w.WriteHeader(http.StatusAccepted)
			_, _ = w.Write([]byte("ignored"))
			return
		}

		published := 0
		for _, event := range events {
			if err := publishGitLabEvent(r.Context(), a.ddashEndpoint, a.ingestorToken, projectID, eventName, deliveryID, event); err != nil {
				slog.Error("failed to publish converted event", "gitlab_event", eventName, "project_id", projectID, "error", err)
				http.Error(w, "publish failed", http.StatusBadGateway)
				return
			}
			published++
		}
		w.WriteHeader(http.StatusAccepted)
		_, _ = w.Write([]byte(fmt.Sprintf("published=%d", published)))
	})
}

func publishGitLabEvent(ctx context.Context, endpoint, setupToken string, projectID int64, eventName, deliveryID string, event eventpublisher.Event) error {
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

	requestURL := base + "/webhooks/gitlab-app"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, requestURL, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+strings.TrimSpace(setupToken))
	req.Header.Set("X-GitLab-Project-ID", strconv.FormatInt(projectID, 10))
	req.Header.Set("X-GitLab-Event", strings.TrimSpace(eventName))
	req.Header.Set("X-GitLab-Event-UUID", strings.TrimSpace(deliveryID))

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
