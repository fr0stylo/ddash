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
