package githubbridge

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/fr0stylo/ddash/pkg/eventpublisher"
)

type ConvertConfig struct {
	DefaultEnvironment string
	Source             string
}

func Convert(eventName, deliveryID string, payload []byte, cfg ConvertConfig) ([]eventpublisher.Event, error) {
	switch strings.TrimSpace(eventName) {
	case "release":
		return convertRelease(payload, cfg)
	case "deployment_status":
		return convertDeploymentStatus(payload, cfg)
	case "workflow_run":
		return convertWorkflowRun(payload, cfg)
	case "push":
		return convertPush(payload, deliveryID, cfg)
	case "pull_request":
		return convertPullRequest(payload, deliveryID, cfg)
	default:
		return nil, nil
	}
}

func convertRelease(payload []byte, cfg ConvertConfig) ([]eventpublisher.Event, error) {
	var body struct {
		Action     string `json:"action"`
		Repository struct {
			Name string `json:"name"`
		} `json:"repository"`
		Release struct {
			TagName string `json:"tag_name"`
			HTMLURL string `json:"html_url"`
		} `json:"release"`
		Sender struct {
			Login string `json:"login"`
		} `json:"sender"`
	}
	if err := json.Unmarshal(payload, &body); err != nil {
		return nil, err
	}
	if strings.TrimSpace(body.Action) != "published" {
		return nil, nil
	}
	service := strings.TrimSpace(body.Repository.Name)
	if service == "" {
		return nil, nil
	}
	tag := strings.TrimSpace(body.Release.TagName)
	if tag == "" {
		tag = "latest"
	}
	env := defaultEnvironment(cfg)
	return []eventpublisher.Event{{
		Type:        "service.published",
		Source:      defaultSource(cfg),
		Service:     service,
		Environment: env,
		Artifact:    fmt.Sprintf("pkg:generic/%s@%s", service, tag),
		ActorName:   strings.TrimSpace(body.Sender.Login),
		PipelineURL: strings.TrimSpace(body.Release.HTMLURL),
	}}, nil
}

func convertDeploymentStatus(payload []byte, cfg ConvertConfig) ([]eventpublisher.Event, error) {
	var body struct {
		Repository struct {
			Name string `json:"name"`
		} `json:"repository"`
		Deployment struct {
			Environment string `json:"environment"`
			SHA         string `json:"sha"`
		} `json:"deployment"`
		DeploymentStatus struct {
			State       string `json:"state"`
			Environment string `json:"environment"`
			TargetURL   string `json:"target_url"`
		} `json:"deployment_status"`
		Sender struct {
			Login string `json:"login"`
		} `json:"sender"`
	}
	if err := json.Unmarshal(payload, &body); err != nil {
		return nil, err
	}
	service := strings.TrimSpace(body.Repository.Name)
	if service == "" {
		return nil, nil
	}
	state := strings.ToLower(strings.TrimSpace(body.DeploymentStatus.State))
	typeName := "service.upgraded"
	switch state {
	case "success":
		typeName = "service.deployed"
	case "failure", "error", "inactive":
		typeName = "service.removed"
	case "in_progress", "queued", "pending":
		typeName = "service.upgraded"
	}
	env := strings.TrimSpace(body.DeploymentStatus.Environment)
	if env == "" {
		env = strings.TrimSpace(body.Deployment.Environment)
	}
	if env == "" {
		env = defaultEnvironment(cfg)
	}
	sha := strings.TrimSpace(body.Deployment.SHA)
	if sha == "" {
		sha = "unknown"
	}
	return []eventpublisher.Event{{
		Type:        typeName,
		Source:      defaultSource(cfg),
		Service:     service,
		Environment: env,
		Artifact:    fmt.Sprintf("pkg:generic/%s@%s", service, shortSHA(sha)),
		ActorName:   strings.TrimSpace(body.Sender.Login),
		PipelineURL: strings.TrimSpace(body.DeploymentStatus.TargetURL),
	}}, nil
}

func convertWorkflowRun(payload []byte, cfg ConvertConfig) ([]eventpublisher.Event, error) {
	var body struct {
		Action     string `json:"action"`
		Repository struct {
			Name string `json:"name"`
		} `json:"repository"`
		WorkflowRun struct {
			ID         int64  `json:"id"`
			Status     string `json:"status"`
			Conclusion string `json:"conclusion"`
			HTMLURL    string `json:"html_url"`
			HeadSHA    string `json:"head_sha"`
		} `json:"workflow_run"`
		Sender struct {
			Login string `json:"login"`
		} `json:"sender"`
	}
	if err := json.Unmarshal(payload, &body); err != nil {
		return nil, err
	}
	service := strings.TrimSpace(body.Repository.Name)
	if service == "" {
		return nil, nil
	}
	eventType := "dev.cdevents.pipeline.run.started.0.3.0"
	if strings.EqualFold(strings.TrimSpace(body.Action), "completed") {
		if strings.EqualFold(strings.TrimSpace(body.WorkflowRun.Conclusion), "success") {
			eventType = "dev.cdevents.pipeline.run.succeeded.0.3.0"
		} else {
			eventType = "dev.cdevents.pipeline.run.failed.0.3.0"
		}
	}
	env := defaultEnvironment(cfg)
	return []eventpublisher.Event{{
		Type:        eventType,
		Source:      defaultSource(cfg),
		Service:     service,
		Environment: env,
		Artifact:    fmt.Sprintf("pkg:generic/%s@%s", service, shortSHA(body.WorkflowRun.HeadSHA)),
		SubjectType: "pipeline",
		SubjectID:   fmt.Sprintf("pipeline/%s/%d", service, body.WorkflowRun.ID),
		PipelineRun: fmt.Sprintf("%d", body.WorkflowRun.ID),
		PipelineURL: strings.TrimSpace(body.WorkflowRun.HTMLURL),
		ActorName:   strings.TrimSpace(body.Sender.Login),
	}}, nil
}

func convertPush(payload []byte, deliveryID string, cfg ConvertConfig) ([]eventpublisher.Event, error) {
	var body struct {
		After      string `json:"after"`
		Repository struct {
			Name string `json:"name"`
		} `json:"repository"`
		Pusher struct {
			Name string `json:"name"`
		} `json:"pusher"`
	}
	if err := json.Unmarshal(payload, &body); err != nil {
		return nil, err
	}
	service := strings.TrimSpace(body.Repository.Name)
	if service == "" {
		return nil, nil
	}
	return []eventpublisher.Event{{
		Type:        "dev.cdevents.change.pushed.0.3.0",
		Source:      defaultSource(cfg),
		Service:     service,
		Environment: defaultEnvironment(cfg),
		Artifact:    fmt.Sprintf("pkg:generic/%s@%s", service, shortSHA(body.After)),
		SubjectType: "change",
		SubjectID:   "change/" + shortSHA(body.After),
		ChainID:     strings.TrimSpace(deliveryID),
		ActorName:   strings.TrimSpace(body.Pusher.Name),
	}}, nil
}

func convertPullRequest(payload []byte, deliveryID string, cfg ConvertConfig) ([]eventpublisher.Event, error) {
	var body struct {
		Action      string `json:"action"`
		PullRequest struct {
			Number int64 `json:"number"`
			Merged bool  `json:"merged"`
			Head   struct {
				SHA string `json:"sha"`
			} `json:"head"`
		} `json:"pull_request"`
		Repository struct {
			Name string `json:"name"`
		} `json:"repository"`
		Sender struct {
			Login string `json:"login"`
		} `json:"sender"`
	}
	if err := json.Unmarshal(payload, &body); err != nil {
		return nil, err
	}
	service := strings.TrimSpace(body.Repository.Name)
	if service == "" {
		return nil, nil
	}
	action := strings.ToLower(strings.TrimSpace(body.Action))
	suffix := action
	if action == "closed" && body.PullRequest.Merged {
		suffix = "merged"
	}
	if suffix == "" {
		suffix = "updated"
	}
	return []eventpublisher.Event{{
		Type:        "dev.cdevents.change." + suffix + ".0.3.0",
		Source:      defaultSource(cfg),
		Service:     service,
		Environment: defaultEnvironment(cfg),
		Artifact:    fmt.Sprintf("pkg:generic/%s@%s", service, shortSHA(body.PullRequest.Head.SHA)),
		SubjectType: "change",
		SubjectID:   fmt.Sprintf("change/pr-%d", body.PullRequest.Number),
		ChainID:     strings.TrimSpace(deliveryID),
		ActorName:   strings.TrimSpace(body.Sender.Login),
	}}, nil
}

func shortSHA(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return "unknown"
	}
	if len(value) > 12 {
		return value[:12]
	}
	return value
}

func defaultEnvironment(cfg ConvertConfig) string {
	value := strings.TrimSpace(cfg.DefaultEnvironment)
	if value == "" {
		return "production"
	}
	return value
}

func defaultSource(cfg ConvertConfig) string {
	value := strings.TrimSpace(cfg.Source)
	if value == "" {
		return "github/app"
	}
	return value
}
