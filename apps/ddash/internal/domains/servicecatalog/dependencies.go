package servicecatalog

import "strings"

func NormalizeDependencyInput(serviceName, dependsOn string) (string, string, bool) {
	serviceName = strings.TrimSpace(serviceName)
	dependsOn = strings.TrimSpace(dependsOn)
	if serviceName == "" || dependsOn == "" {
		return "", "", false
	}
	if strings.EqualFold(serviceName, dependsOn) {
		return "", "", false
	}
	return serviceName, dependsOn, true
}

func ParseDependencyInputs(serviceName, rawDependsOn string) []string {
	serviceName = strings.TrimSpace(serviceName)
	if serviceName == "" {
		return nil
	}

	normalized := strings.NewReplacer("\n", ",", "\r", ",", ";", ",").Replace(rawDependsOn)
	parts := strings.Split(normalized, ",")
	unique := make(map[string]bool, len(parts))
	values := make([]string, 0, len(parts))
	for _, part := range parts {
		candidate := strings.TrimSpace(part)
		if candidate == "" {
			continue
		}
		if strings.EqualFold(candidate, serviceName) {
			continue
		}
		key := strings.ToLower(candidate)
		if unique[key] {
			continue
		}
		unique[key] = true
		values = append(values, candidate)
	}
	return values
}
