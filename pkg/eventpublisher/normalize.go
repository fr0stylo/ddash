package eventpublisher

import "strings"

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
