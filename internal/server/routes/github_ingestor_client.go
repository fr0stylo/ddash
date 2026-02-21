package routes

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

type GitHubIngestorClient struct {
	baseURL    string
	apiToken   string
	publicURL  string
	httpClient *http.Client
}

type gitHubInstallationMapping struct {
	InstallationID     int64  `json:"installation_id"`
	OrganizationID     int64  `json:"organization_id"`
	OrganizationLabel  string `json:"organization_label"`
	DDashEndpoint      string `json:"ddash_endpoint"`
	DefaultEnvironment string `json:"default_environment"`
	Enabled            bool   `json:"enabled"`
}

func NewGitHubIngestorClient(baseURL, apiToken, publicURL string) *GitHubIngestorClient {
	return &GitHubIngestorClient{
		baseURL:    strings.TrimRight(strings.TrimSpace(baseURL), "/"),
		apiToken:   strings.TrimSpace(apiToken),
		publicURL:  strings.TrimRight(strings.TrimSpace(publicURL), "/"),
		httpClient: &http.Client{Timeout: 10 * time.Second},
	}
}

func (c *GitHubIngestorClient) Enabled() bool {
	return c != nil && c.baseURL != "" && c.apiToken != ""
}

func (c *GitHubIngestorClient) StartInstall(ctx context.Context, organizationID int64, organizationLabel, authToken, webhookSecret, defaultEnvironment string) (string, error) {
	if !c.Enabled() {
		return "", fmt.Errorf("github ingestor not configured")
	}
	body := map[string]any{
		"organization_id":      organizationID,
		"organization_label":   strings.TrimSpace(organizationLabel),
		"ddash_endpoint":       c.publicURL,
		"ddash_auth_token":     strings.TrimSpace(authToken),
		"ddash_webhook_secret": strings.TrimSpace(webhookSecret),
		"default_environment":  strings.TrimSpace(defaultEnvironment),
	}
	raw, _ := json.Marshal(body)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/api/setup/start", bytes.NewReader(raw))
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", "Bearer "+c.apiToken)
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		payload, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("setup start failed: %s", strings.TrimSpace(string(payload)))
	}
	var parsed struct {
		RedirectURL string `json:"redirect_url"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&parsed); err != nil {
		return "", err
	}
	if strings.TrimSpace(parsed.RedirectURL) == "" {
		return "", fmt.Errorf("missing redirect_url from ingestor")
	}
	if strings.HasPrefix(parsed.RedirectURL, "/") {
		return c.baseURL + parsed.RedirectURL, nil
	}
	if parsedURL, err := url.Parse(parsed.RedirectURL); err == nil && parsedURL.Scheme != "" && parsedURL.Host != "" {
		return parsedURL.String(), nil
	}
	return "", fmt.Errorf("invalid redirect_url from ingestor")
}

func (c *GitHubIngestorClient) ListMappings(ctx context.Context, organizationID int64) ([]gitHubInstallationMapping, error) {
	if !c.Enabled() {
		return nil, nil
	}
	endpoint := c.baseURL + "/api/mappings"
	if organizationID > 0 {
		endpoint += "?org_id=" + strconv.FormatInt(organizationID, 10)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+c.apiToken)
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		payload, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("list mappings failed: %s", strings.TrimSpace(string(payload)))
	}
	var parsed struct {
		Mappings []gitHubInstallationMapping `json:"mappings"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&parsed); err != nil {
		return nil, err
	}
	return parsed.Mappings, nil
}

func (c *GitHubIngestorClient) DeleteMapping(ctx context.Context, installationID, organizationID int64) error {
	if !c.Enabled() {
		return fmt.Errorf("github ingestor not configured")
	}
	body := map[string]any{"installation_id": installationID}
	if organizationID > 0 {
		body["organization_id"] = organizationID
	}
	raw, _ := json.Marshal(body)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/api/mappings/delete", bytes.NewReader(raw))
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+c.apiToken)
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		payload, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("delete mapping failed: %s", strings.TrimSpace(string(payload)))
	}
	return nil
}
