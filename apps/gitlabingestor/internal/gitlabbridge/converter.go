package gitlabbridge

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
	case "Push Hook":
		return convertPush(payload, deliveryID, cfg)
	case "Tag Push Hook":
		return convertTagPush(payload, cfg)
	case "Pipeline Hook":
		return convertPipeline(payload, cfg)
	case "Deployment Hook":
		return convertDeployment(payload, cfg)
	case "Merge Request Hook":
		return convertMergeRequest(payload, deliveryID, cfg)
	default:
		return nil, nil
	}
}

func ExtractProjectID(payload []byte) (int64, bool) {
	var body struct {
		ProjectID int64 `json:"project_id"`
		Project   struct {
			ID int64 `json:"id"`
		} `json:"project"`
	}
	if err := json.Unmarshal(payload, &body); err != nil {
		return 0, false
	}
	if body.ProjectID > 0 {
		return body.ProjectID, true
	}
	if body.Project.ID > 0 {
		return body.Project.ID, true
	}
	return 0, false
}

func convertPush(payload []byte, deliveryID string, cfg ConvertConfig) ([]eventpublisher.Event, error) {
	var body struct {
		After    string `json:"after"`
		UserName string `json:"user_name"`
		Project  struct {
			Name string `json:"name"`
		} `json:"project"`
		Repository struct {
			Name string `json:"name"`
		} `json:"repository"`
	}
	if err := json.Unmarshal(payload, &body); err != nil {
		return nil, err
	}
	service := firstNonEmpty(body.Project.Name, body.Repository.Name)
	service = strings.TrimSpace(service)
	if service == "" {
		return nil, nil
	}
	sha := shortSHA(body.After)
	return []eventpublisher.Event{{
		Type:        "dev.cdevents.change.pushed.0.3.0",
		Source:      defaultSource(cfg),
		Service:     service,
		Environment: defaultEnvironment(cfg),
		Artifact:    fmt.Sprintf("pkg:generic/%s@%s", service, sha),
		SubjectType: "change",
		SubjectID:   "change/" + sha,
		ChainID:     strings.TrimSpace(deliveryID),
		ActorName:   strings.TrimSpace(body.UserName),
	}}, nil
}

func convertTagPush(payload []byte, cfg ConvertConfig) ([]eventpublisher.Event, error) {
	var body struct {
		Ref      string `json:"ref"`
		UserName string `json:"user_name"`
		Project  struct {
			Name string `json:"name"`
		} `json:"project"`
		Repository struct {
			HomePage string `json:"homepage"`
		} `json:"repository"`
	}
	if err := json.Unmarshal(payload, &body); err != nil {
		return nil, err
	}
	service := strings.TrimSpace(body.Project.Name)
	if service == "" {
		return nil, nil
	}
	tag := strings.TrimPrefix(strings.TrimSpace(body.Ref), "refs/tags/")
	if tag == "" {
		tag = "latest"
	}
	return []eventpublisher.Event{{
		Type:        "service.published",
		Source:      defaultSource(cfg),
		Service:     service,
		Environment: defaultEnvironment(cfg),
		Artifact:    fmt.Sprintf("pkg:generic/%s@%s", service, tag),
		ActorName:   strings.TrimSpace(body.UserName),
		PipelineURL: strings.TrimSpace(body.Repository.HomePage),
	}}, nil
}

func convertPipeline(payload []byte, cfg ConvertConfig) ([]eventpublisher.Event, error) {
	var body struct {
		ObjectAttributes struct {
			ID     int64  `json:"id"`
			Status string `json:"status"`
			URL    string `json:"url"`
			SHA    string `json:"sha"`
			Ref    string `json:"ref"`
		} `json:"object_attributes"`
		User struct {
			Name string `json:"name"`
		} `json:"user"`
		Project struct {
			Name string `json:"name"`
		} `json:"project"`
	}
	if err := json.Unmarshal(payload, &body); err != nil {
		return nil, err
	}
	service := strings.TrimSpace(body.Project.Name)
	if service == "" {
		return nil, nil
	}
	status := strings.ToLower(strings.TrimSpace(body.ObjectAttributes.Status))
	eventType := "dev.cdevents.pipeline.run.started.0.3.0"
	if status == "success" {
		eventType = "dev.cdevents.pipeline.run.succeeded.0.3.0"
	}
	if status == "failed" || status == "canceled" {
		eventType = "dev.cdevents.pipeline.run.failed.0.3.0"
	}
	env := strings.TrimSpace(body.ObjectAttributes.Ref)
	if env == "" {
		env = defaultEnvironment(cfg)
	}
	return []eventpublisher.Event{{
		Type:        eventType,
		Source:      defaultSource(cfg),
		Service:     service,
		Environment: env,
		Artifact:    fmt.Sprintf("pkg:generic/%s@%s", service, shortSHA(body.ObjectAttributes.SHA)),
		SubjectType: "pipeline",
		SubjectID:   fmt.Sprintf("pipeline/%s/%d", service, body.ObjectAttributes.ID),
		PipelineRun: fmt.Sprintf("%d", body.ObjectAttributes.ID),
		PipelineURL: strings.TrimSpace(body.ObjectAttributes.URL),
		ActorName:   strings.TrimSpace(body.User.Name),
	}}, nil
}

func convertDeployment(payload []byte, cfg ConvertConfig) ([]eventpublisher.Event, error) {
	var body struct {
		Status      string `json:"status"`
		Environment string `json:"environment"`
		SHA         string `json:"sha"`
		ShortSHA    string `json:"short_sha"`
		User        struct {
			Name string `json:"name"`
		} `json:"user"`
		Project struct {
			Name string `json:"name"`
		} `json:"project"`
	}
	if err := json.Unmarshal(payload, &body); err != nil {
		return nil, err
	}
	service := strings.TrimSpace(body.Project.Name)
	if service == "" {
		return nil, nil
	}
	state := strings.ToLower(strings.TrimSpace(body.Status))
	typeName := "service.upgraded"
	if state == "success" {
		typeName = "service.deployed"
	}
	if state == "failed" || state == "canceled" {
		typeName = "service.removed"
	}
	env := strings.TrimSpace(body.Environment)
	if env == "" {
		env = defaultEnvironment(cfg)
	}
	sha := firstNonEmpty(body.ShortSHA, body.SHA)
	return []eventpublisher.Event{{
		Type:        typeName,
		Source:      defaultSource(cfg),
		Service:     service,
		Environment: env,
		Artifact:    fmt.Sprintf("pkg:generic/%s@%s", service, shortSHA(sha)),
		ActorName:   strings.TrimSpace(body.User.Name),
	}}, nil
}

func convertMergeRequest(payload []byte, deliveryID string, cfg ConvertConfig) ([]eventpublisher.Event, error) {
	var body struct {
		ObjectAttributes struct {
			IID         int64  `json:"iid"`
			Action      string `json:"action"`
			State       string `json:"state"`
			MergeStatus string `json:"merge_status"`
			LastCommit  struct {
				ID string `json:"id"`
			} `json:"last_commit"`
		} `json:"object_attributes"`
		User struct {
			Name string `json:"name"`
		} `json:"user"`
		Project struct {
			Name string `json:"name"`
		} `json:"project"`
	}
	if err := json.Unmarshal(payload, &body); err != nil {
		return nil, err
	}
	service := strings.TrimSpace(body.Project.Name)
	if service == "" {
		return nil, nil
	}
	action := strings.ToLower(strings.TrimSpace(body.ObjectAttributes.Action))
	if action == "" {
		action = strings.ToLower(strings.TrimSpace(body.ObjectAttributes.State))
	}
	suffix := action
	if suffix == "merge" || suffix == "merged" {
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
		Artifact:    fmt.Sprintf("pkg:generic/%s@%s", service, shortSHA(body.ObjectAttributes.LastCommit.ID)),
		SubjectType: "change",
		SubjectID:   fmt.Sprintf("change/mr-%d", body.ObjectAttributes.IID),
		ChainID:     strings.TrimSpace(deliveryID),
		ActorName:   strings.TrimSpace(body.User.Name),
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
		return "gitlab/webhook"
	}
	return value
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value != "" {
			return value
		}
	}
	return ""
}
