package eventpublisher

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	cdeventsapi "github.com/cdevents/sdk-go/pkg/api"
	cdeventsv05 "github.com/cdevents/sdk-go/pkg/api/v05"
)

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
				"environment": map[string]any{"id": strings.TrimSpace(event.Environment)},
				"artifactId":  strings.TrimSpace(event.Artifact),
				"pipeline": map[string]any{
					"runId": strings.TrimSpace(event.PipelineRun),
					"url":   strings.TrimSpace(event.PipelineURL),
				},
				"actor": map[string]any{"name": strings.TrimSpace(event.ActorName)},
			},
		},
	}
	if chainID := strings.TrimSpace(event.ChainID); chainID != "" {
		payload["context"].(map[string]any)["chainId"] = chainID
	}
	return json.Marshal(payload)
}
