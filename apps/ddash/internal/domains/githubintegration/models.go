package githubintegration

import (
	"strings"
	"time"
)

type InstallationMapping struct {
	InstallationID     int64
	OrganizationID     int64
	OrganizationLabel  string
	DefaultEnvironment string
	Enabled            bool
}

type SetupIntent struct {
	State              string
	OrganizationID     int64
	OrganizationLabel  string
	DefaultEnvironment string
	ExpiresAt          time.Time
}

func NormalizeEnvironment(value string, fallback string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		value = strings.TrimSpace(fallback)
	}
	if value == "" {
		return "production"
	}
	return value
}
