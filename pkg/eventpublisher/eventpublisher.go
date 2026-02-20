package eventpublisher

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
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
	if service == "" || environment == "" {
		return nil, "", fmt.Errorf("service and environment are required")
	}
	source := strings.TrimSpace(event.Source)
	if source == "" {
		source = "ci/pipeline"
	}
	artifact := strings.TrimSpace(event.Artifact)
	if artifact == "" {
		artifact = fmt.Sprintf("pkg:generic/%s@%d", service, time.Now().Unix())
	}
	subjectID := service
	if !strings.Contains(subjectID, "/") {
		subjectID = "service/" + subjectID
	}

	switch normalizeType(event.Type) {
	case "service.deployed":
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
		e, err := cdeventsv05.NewServicePublishedEvent()
		if err != nil {
			return nil, "", err
		}
		e.SetSource(source)
		e.SetSubjectId(subjectID)
		e.SetSubjectEnvironment(&cdeventsapi.Reference{Id: environment})
		body, err := cdeventsapi.AsJsonBytes(e)
		return body, cdeventsv05.ServicePublishedEventType.String(), err
	default:
		return nil, "", fmt.Errorf("unsupported event type %q", event.Type)
	}
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
	default:
		return v
	}
}

func sign(body []byte, secret string) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(body)
	return hex.EncodeToString(mac.Sum(nil))
}
