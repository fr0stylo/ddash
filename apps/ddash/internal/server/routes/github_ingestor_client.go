package routes

import (
	"fmt"
	"net/url"
	"strings"
)

type GitHubIngestorClient struct {
	installURL string
	apiToken   string
}

func NewGitHubIngestorClient(installURL, apiToken, _ string) *GitHubIngestorClient {
	return &GitHubIngestorClient{
		installURL: strings.TrimSpace(installURL),
		apiToken:   strings.TrimSpace(apiToken),
	}
}

func (c *GitHubIngestorClient) Enabled() bool {
	return c != nil && strings.TrimSpace(c.installURL) != "" && c.apiToken != ""
}

func (c *GitHubIngestorClient) StartInstall(state string) (string, error) {
	if !c.Enabled() {
		return "", fmt.Errorf("github app integration not configured")
	}
	parsed, err := url.Parse(strings.TrimSpace(c.installURL))
	if err != nil {
		return "", fmt.Errorf("invalid GITHUB_APP_INSTALL_URL")
	}
	query := parsed.Query()
	query.Set("state", strings.TrimSpace(state))
	parsed.RawQuery = query.Encode()
	if parsed.Scheme == "" || parsed.Host == "" {
		return "", fmt.Errorf("invalid GITHUB_APP_INSTALL_URL")
	}
	return parsed.String(), nil
}
