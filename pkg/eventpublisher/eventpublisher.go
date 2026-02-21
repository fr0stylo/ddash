package eventpublisher

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	cdeventsapi "github.com/cdevents/sdk-go/pkg/api"
	cdeventsv05 "github.com/cdevents/sdk-go/pkg/api/v05"
)

type Client struct {
	Endpoint   string
	Token      string
	Secret     string
	Timeout    time.Duration
	HTTPClient *http.Client
}

type Event struct {
	Type        string
	Source      string
	Service     string
	Environment string
	Artifact    string
	SubjectID   string
	SubjectType string
	ChainID     string
	ActorName   string
	PipelineRun string
	PipelineURL string
}

func (c Client) Publish(ctx context.Context, event Event) (string, error) {
	body, resolvedType, err := BuildEventBody(event)
	if err != nil {
		return "", err
	}
	if err := c.publishBody(ctx, body); err != nil {
		return "", err
	}
	return resolvedType, nil
}

func BuildEventBody(event Event) ([]byte, string, error) {
	service := strings.TrimSpace(event.Service)
	environment := strings.TrimSpace(event.Environment)
	source := strings.TrimSpace(event.Source)
	if source == "" {
		source = "ci/pipeline"
	}

	resolved := normalizeType(event.Type)
	artifact := strings.TrimSpace(event.Artifact)
	if artifact == "" && strings.HasPrefix(resolved, "service.") {
		artifact = fmt.Sprintf("pkg:generic/%s@%d", service, time.Now().Unix())
	}

	subjectType := strings.TrimSpace(event.SubjectType)
	subjectID := strings.TrimSpace(event.SubjectID)
	if subjectType == "" {
		subjectType = inferSubjectType(resolved)
	}
	if subjectID == "" {
		subjectID = inferSubjectID(subjectType, service, environment)
	}
	if subjectID == "" {
		return nil, "", fmt.Errorf("subject is required (set subject-id, service, or environment based on event type)")
	}
	if !strings.Contains(subjectID, "/") && subjectType != "" {
		subjectID = subjectType + "/" + subjectID
	}

	switch resolved {
	case "service.deployed":
		if service == "" || environment == "" {
			return nil, "", fmt.Errorf("service and environment are required for service events")
		}
		e, err := cdeventsv05.NewServiceDeployedEvent()
		if err != nil {
			return nil, "", err
		}
		e.SetSource(source)
		e.SetSubjectId(subjectID)
		e.SetSubjectEnvironment(&cdeventsapi.Reference{Id: environment})
		e.SetSubjectArtifactId(artifact)
		body, err := cdeventsapi.AsJsonBytes(e)
		return body, cdeventsv05.ServiceDeployedEventType.String(), err
	case "service.upgraded":
		if service == "" || environment == "" {
			return nil, "", fmt.Errorf("service and environment are required for service events")
		}
		e, err := cdeventsv05.NewServiceUpgradedEvent()
		if err != nil {
			return nil, "", err
		}
		e.SetSource(source)
		e.SetSubjectId(subjectID)
		e.SetSubjectEnvironment(&cdeventsapi.Reference{Id: environment})
		e.SetSubjectArtifactId(artifact)
		body, err := cdeventsapi.AsJsonBytes(e)
		return body, cdeventsv05.ServiceUpgradedEventType.String(), err
	case "service.rolledback":
		if service == "" || environment == "" {
			return nil, "", fmt.Errorf("service and environment are required for service events")
		}
		e, err := cdeventsv05.NewServiceRolledbackEvent()
		if err != nil {
			return nil, "", err
		}
		e.SetSource(source)
		e.SetSubjectId(subjectID)
		e.SetSubjectEnvironment(&cdeventsapi.Reference{Id: environment})
		e.SetSubjectArtifactId(artifact)
		body, err := cdeventsapi.AsJsonBytes(e)
		return body, cdeventsv05.ServiceRolledbackEventType.String(), err
	case "service.removed":
		if service == "" || environment == "" {
			return nil, "", fmt.Errorf("service and environment are required for service events")
		}
		e, err := cdeventsv05.NewServiceRemovedEvent()
		if err != nil {
			return nil, "", err
		}
		e.SetSource(source)
		e.SetSubjectId(subjectID)
		e.SetSubjectEnvironment(&cdeventsapi.Reference{Id: environment})
		body, err := cdeventsapi.AsJsonBytes(e)
		return body, cdeventsv05.ServiceRemovedEventType.String(), err
	case "service.published":
		if service == "" || environment == "" {
			return nil, "", fmt.Errorf("service and environment are required for service events")
		}
		e, err := cdeventsv05.NewServicePublishedEvent()
		if err != nil {
			return nil, "", err
		}
		e.SetSource(source)
		e.SetSubjectId(subjectID)
		e.SetSubjectEnvironment(&cdeventsapi.Reference{Id: environment})
		body, err := cdeventsapi.AsJsonBytes(e)
		return body, cdeventsv05.ServicePublishedEventType.String(), err
	case "environment.created":
		if environment == "" {
			return nil, "", fmt.Errorf("environment is required for environment events")
		}
		e, err := cdeventsv05.NewEnvironmentCreatedEvent()
		if err != nil {
			return nil, "", err
		}
		e.SetSource(source)
		e.SetSubjectId(subjectID)
		body, err := cdeventsapi.AsJsonBytes(e)
		return body, cdeventsv05.EnvironmentCreatedEventType.String(), err
	case "environment.modified":
		if environment == "" {
			return nil, "", fmt.Errorf("environment is required for environment events")
		}
		e, err := cdeventsv05.NewEnvironmentModifiedEvent()
		if err != nil {
			return nil, "", err
		}
		e.SetSource(source)
		e.SetSubjectId(subjectID)
		body, err := cdeventsapi.AsJsonBytes(e)
		return body, cdeventsv05.EnvironmentModifiedEventType.String(), err
	case "environment.deleted":
		if environment == "" {
			return nil, "", fmt.Errorf("environment is required for environment events")
		}
		e, err := cdeventsv05.NewEnvironmentDeletedEvent()
		if err != nil {
			return nil, "", err
		}
		e.SetSource(source)
		e.SetSubjectId(subjectID)
		body, err := cdeventsapi.AsJsonBytes(e)
		return body, cdeventsv05.EnvironmentDeletedEventType.String(), err
	default:
		if isAcceptedCustomType(resolved) {
			body, err := buildGenericEventBody(Event{
				Type:        resolved,
				Source:      source,
				Environment: environment,
				Artifact:    artifact,
				SubjectID:   subjectID,
				SubjectType: subjectType,
				ChainID:     strings.TrimSpace(event.ChainID),
				ActorName:   strings.TrimSpace(event.ActorName),
				PipelineRun: strings.TrimSpace(event.PipelineRun),
				PipelineURL: strings.TrimSpace(event.PipelineURL),
			})
			return body, resolved, err
		}
		return nil, "", fmt.Errorf("unsupported event type %q", event.Type)
	}
}

func buildGenericEventBody(event Event) ([]byte, error) {
	now := time.Now().UTC().Format(time.RFC3339Nano)
	identifier := fmt.Sprintf("%d", time.Now().UTC().UnixNano())
	payload := map[string]any{
		"context": map[string]any{
			"id":          identifier,
			"source":      event.Source,
			"type":        event.Type,
			"timestamp":   now,
			"specversion": "0.5.0",
		},
		"subject": map[string]any{
			"id":     event.SubjectID,
			"source": event.Source,
			"type":   event.SubjectType,
			"content": map[string]any{
				"environment": map[string]any{
					"id": strings.TrimSpace(event.Environment),
				},
				"artifactId": strings.TrimSpace(event.Artifact),
				"pipeline": map[string]any{
					"runId": strings.TrimSpace(event.PipelineRun),
					"url":   strings.TrimSpace(event.PipelineURL),
				},
				"actor": map[string]any{
					"name": strings.TrimSpace(event.ActorName),
				},
			},
		},
	}
	if chainID := strings.TrimSpace(event.ChainID); chainID != "" {
		payload["context"].(map[string]any)["chainId"] = chainID
	}
	return json.Marshal(payload)
}

func (c Client) publishBody(ctx context.Context, body []byte) error {
	endpoint := strings.TrimSpace(c.Endpoint)
	token := strings.TrimSpace(c.Token)
	secret := strings.TrimSpace(c.Secret)
	if endpoint == "" || token == "" || secret == "" {
		return fmt.Errorf("endpoint/token/secret are required")
	}

	httpClient := c.HTTPClient
	if httpClient == nil {
		timeout := c.Timeout
		if timeout <= 0 {
			timeout = 10 * time.Second
		}
		httpClient = &http.Client{Timeout: timeout}
	}

	requestURL := strings.TrimRight(endpoint, "/") + "/webhooks/cdevents"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, requestURL, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("X-Webhook-Signature", sign(body, secret))
	req.Header.Set("Content-Type", "application/json")

	resp, err := httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= http.StatusMultipleChoices {
		payload, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("webhook rejected: status=%s body=%s", resp.Status, strings.TrimSpace(string(payload)))
	}
	return nil
}

func normalizeType(value string) string {
	v := strings.ToLower(strings.TrimSpace(value))
	switch v {
	case "", "dev.cdevents.service.deployed.0.3.0":
		return "service.deployed"
	case "dev.cdevents.service.upgraded.0.3.0":
		return "service.upgraded"
	case "dev.cdevents.service.rolledback.0.3.0":
		return "service.rolledback"
	case "dev.cdevents.service.removed.0.3.0":
		return "service.removed"
	case "dev.cdevents.service.published.0.3.0":
		return "service.published"
	case "environment.created", "dev.cdevents.environment.created.0.3.0":
		return "environment.created"
	case "environment.modified", "dev.cdevents.environment.modified.0.3.0":
		return "environment.modified"
	case "environment.deleted", "dev.cdevents.environment.deleted.0.3.0":
		return "environment.deleted"
	default:
		return v
	}
}

func inferSubjectType(resolvedType string) string {
	switch {
	case strings.HasPrefix(resolvedType, "service."):
		return "service"
	case strings.HasPrefix(resolvedType, "environment."):
		return "environment"
	case strings.Contains(resolvedType, ".pipeline."):
		return "pipeline"
	case strings.Contains(resolvedType, ".change."):
		return "change"
	case strings.Contains(resolvedType, ".artifact."):
		return "artifact"
	case strings.Contains(resolvedType, ".incident."):
		return "incident"
	default:
		return "service"
	}
}

func inferSubjectID(subjectType, service, environment string) string {
	service = strings.TrimSpace(service)
	environment = strings.TrimSpace(environment)
	subjectType = strings.TrimSpace(subjectType)
	if service != "" {
		if strings.Contains(service, "/") {
			return service
		}
		return subjectType + "/" + service
	}
	if subjectType == "environment" && environment != "" {
		return "environment/" + environment
	}
	return ""
}

func isAcceptedCustomType(eventType string) bool {
	eventType = strings.TrimSpace(strings.ToLower(eventType))
	return strings.HasPrefix(eventType, "dev.cdevents.pipeline.") ||
		strings.HasPrefix(eventType, "dev.cdevents.change.") ||
		strings.HasPrefix(eventType, "dev.cdevents.artifact.") ||
		strings.HasPrefix(eventType, "dev.cdevents.incident.")
}

func sign(body []byte, secret string) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(body)
	return hex.EncodeToString(mac.Sum(nil))
}
