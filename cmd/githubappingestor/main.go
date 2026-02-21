package main

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"html/template"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/joho/godotenv"
	"github.com/spf13/viper"

	"github.com/fr0stylo/ddash/internal/githubbridge"
	"github.com/fr0stylo/ddash/pkg/eventpublisher"
)

func main() {
	_ = godotenv.Load()
	v := viper.New()
	v.AutomaticEnv()

	listenAddr := valueOrDefault(v.GetString("GITHUB_APP_INGESTOR_ADDR"), ":8081")
	webhookPath := valueOrDefault(v.GetString("GITHUB_APP_INGESTOR_PATH"), "/webhooks/github")
	setupUIPath := valueOrDefault(v.GetString("GITHUB_APP_INGESTOR_SETUP_UI_PATH"), "/setup")
	setupStartPath := valueOrDefault(v.GetString("GITHUB_APP_INGESTOR_SETUP_START_PATH"), "/setup/start")
	setupCallbackPath := valueOrDefault(v.GetString("GITHUB_APP_INGESTOR_SETUP_CALLBACK_PATH"), "/setup/callback")
	setupDeletePath := valueOrDefault(v.GetString("GITHUB_APP_INGESTOR_SETUP_DELETE_PATH"), "/setup/mappings/delete")
	installURL := strings.TrimSpace(v.GetString("GITHUB_APP_INSTALL_URL"))
	setupToken := strings.TrimSpace(v.GetString("GITHUB_APP_INGESTOR_SETUP_TOKEN"))

	githubSecret := strings.TrimSpace(v.GetString("GITHUB_WEBHOOK_SECRET"))
	if githubSecret == "" {
		slog.Error("GITHUB_WEBHOOK_SECRET is required")
		os.Exit(1)
	}

	storePath := valueOrDefault(v.GetString("GITHUB_APP_INGESTOR_DB_PATH"), "data/githubapp-ingestor")
	installStore, err := githubbridge.OpenInstallStore(storePath)
	if err != nil {
		slog.Error("failed to open installation store", "error", err)
		os.Exit(1)
	}
	defer func() {
		_ = installStore.Close()
	}()

	defaultClient := eventpublisher.Client{
		Endpoint: strings.TrimSpace(v.GetString("DDASH_ENDPOINT")),
		Token:    strings.TrimSpace(v.GetString("DDASH_AUTH_TOKEN")),
		Secret:   strings.TrimSpace(v.GetString("DDASH_WEBHOOK_SECRET")),
		Timeout:  10 * time.Second,
	}
	defaultConvertCfg := githubbridge.ConvertConfig{
		DefaultEnvironment: strings.TrimSpace(v.GetString("GITHUB_APP_INGESTOR_DEFAULT_ENV")),
		Source:             valueOrDefault(v.GetString("GITHUB_APP_INGESTOR_SOURCE"), "github/app"),
	}

	mux := http.NewServeMux()

	mux.HandleFunc("/api/setup/start", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		if setupToken != "" && !authorizedSetupRequest(r, setupToken) {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		payload := setupStartPayload{}
		if err := json.NewDecoder(io.LimitReader(r.Body, 1<<20)).Decode(&payload); err != nil {
			http.Error(w, "invalid json payload", http.StatusBadRequest)
			return
		}
		intent, redirectURL, err := createSetupIntent(installStore, installURL, defaultConvertCfg.DefaultEnvironment, payload)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		_ = writeJSON(w, map[string]string{
			"state":        intent.State,
			"redirect_url": redirectURL,
		})
	})

	mux.HandleFunc("/api/mappings", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		if setupToken != "" && !authorizedSetupRequest(r, setupToken) {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		orgID, _ := strconv.ParseInt(strings.TrimSpace(r.URL.Query().Get("org_id")), 10, 64)
		mappings, err := installStore.ListInstallationMappings(orgID)
		if err != nil {
			http.Error(w, "failed to list mappings", http.StatusInternalServerError)
			return
		}
		_ = writeJSON(w, map[string]any{"mappings": mappings})
	})

	mux.HandleFunc("/api/mappings/delete", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		if setupToken != "" && !authorizedSetupRequest(r, setupToken) {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		var payload struct {
			InstallationID int64 `json:"installation_id"`
			OrganizationID int64 `json:"organization_id"`
		}
		if err := json.NewDecoder(io.LimitReader(r.Body, 1<<20)).Decode(&payload); err != nil {
			http.Error(w, "invalid json payload", http.StatusBadRequest)
			return
		}
		if payload.InstallationID <= 0 {
			http.Error(w, "invalid installation_id", http.StatusBadRequest)
			return
		}
		if err := installStore.DeleteInstallationMapping(payload.InstallationID, payload.OrganizationID); err != nil {
			if err == sql.ErrNoRows {
				http.Error(w, "mapping not found", http.StatusNotFound)
				return
			}
			http.Error(w, "failed to delete mapping", http.StatusInternalServerError)
			return
		}
		_ = writeJSON(w, map[string]any{"deleted": true})
	})

	mux.HandleFunc(setupUIPath, func(w http.ResponseWriter, r *http.Request) {
		if setupToken != "" && !authorizedSetupRequest(r, setupToken) {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		mappings, err := installStore.ListInstallationMappings(0)
		if err != nil {
			http.Error(w, "failed to list mappings", http.StatusInternalServerError)
			return
		}
		uiData := setupPageData{
			InstallURL:           installURL,
			SetupToken:           setupToken,
			SetupStartPath:       setupStartPath,
			SetupDeletePath:      setupDeletePath,
			DefaultDDashEndpoint: defaultClient.Endpoint,
			DefaultDDashToken:    defaultClient.Token,
			DefaultDDashSecret:   defaultClient.Secret,
			DefaultEnvironment:   defaultConvertCfg.DefaultEnvironment,
			Mappings:             mappings,
		}
		if err := renderSetupPage(w, uiData); err != nil {
			http.Error(w, "failed to render setup page", http.StatusInternalServerError)
			return
		}
	})

	mux.HandleFunc(setupStartPath, func(w http.ResponseWriter, r *http.Request) {
		if setupToken != "" && !authorizedSetupRequest(r, setupToken) {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}

		organizationID, _ := strconv.ParseInt(strings.TrimSpace(r.URL.Query().Get("organization_id")), 10, 64)
		payload := setupStartPayload{
			OrganizationID:     organizationID,
			OrganizationLabel:  strings.TrimSpace(r.URL.Query().Get("organization")),
			DDashEndpoint:      strings.TrimSpace(firstNonEmpty(r.URL.Query().Get("ddash_endpoint"), defaultClient.Endpoint)),
			DDashAuthToken:     strings.TrimSpace(firstNonEmpty(r.URL.Query().Get("ddash_auth_token"), defaultClient.Token)),
			DDashWebhookSecret: strings.TrimSpace(firstNonEmpty(r.URL.Query().Get("ddash_webhook_secret"), defaultClient.Secret)),
			DefaultEnvironment: strings.TrimSpace(firstNonEmpty(r.URL.Query().Get("default_environment"), defaultConvertCfg.DefaultEnvironment)),
		}
		_, redirectURL, err := createSetupIntent(installStore, installURL, defaultConvertCfg.DefaultEnvironment, payload)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		http.Redirect(w, r, redirectURL, http.StatusFound)
	})

	mux.HandleFunc(setupCallbackPath, func(w http.ResponseWriter, r *http.Request) {
		state := strings.TrimSpace(r.URL.Query().Get("state"))
		installationIDRaw := strings.TrimSpace(firstNonEmpty(r.URL.Query().Get("installation_id"), r.URL.Query().Get("installationId")))
		if state == "" || installationIDRaw == "" {
			http.Error(w, "missing state or installation_id", http.StatusBadRequest)
			return
		}
		installationID, err := strconv.ParseInt(installationIDRaw, 10, 64)
		if err != nil || installationID <= 0 {
			http.Error(w, "invalid installation_id", http.StatusBadRequest)
			return
		}

		intent, err := installStore.GetSetupIntent(state)
		if err != nil {
			if err == sql.ErrNoRows {
				http.Error(w, "unknown setup state", http.StatusNotFound)
				return
			}
			http.Error(w, "failed to resolve setup state", http.StatusInternalServerError)
			return
		}
		if time.Now().UTC().After(intent.ExpiresAt.UTC()) {
			_ = installStore.DeleteSetupIntent(state)
			http.Error(w, "setup state expired", http.StatusGone)
			return
		}

		mapping := githubbridge.InstallationMapping{
			InstallationID:     installationID,
			OrganizationID:     intent.OrganizationID,
			OrganizationLabel:  intent.OrganizationLabel,
			DDashEndpoint:      intent.DDashEndpoint,
			DDashAuthToken:     intent.DDashAuthToken,
			DDashWebhookSecret: intent.DDashWebhookSecret,
			DefaultEnvironment: intent.DefaultEnvironment,
			Enabled:            true,
		}
		if err := installStore.UpsertInstallationMapping(mapping); err != nil {
			http.Error(w, "failed to save installation mapping", http.StatusInternalServerError)
			return
		}
		_ = installStore.DeleteSetupIntent(state)

		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = w.Write([]byte("<html><body><h2>GitHub installation mapped successfully.</h2><p>You can close this window.</p><p><a href='" + setupUIPath + "?setup_token=" + url.QueryEscape(setupToken) + "'>Open mapping dashboard</a></p></body></html>"))
	})

	mux.HandleFunc(setupDeletePath, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		if setupToken != "" && !authorizedSetupRequest(r, setupToken) {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		if err := r.ParseForm(); err != nil {
			http.Error(w, "invalid form", http.StatusBadRequest)
			return
		}
		installationID, err := strconv.ParseInt(strings.TrimSpace(r.FormValue("installation_id")), 10, 64)
		if err != nil || installationID <= 0 {
			http.Error(w, "invalid installation id", http.StatusBadRequest)
			return
		}
		if err := installStore.DeleteInstallationMapping(installationID, 0); err != nil {
			http.Error(w, "failed to delete mapping", http.StatusInternalServerError)
			return
		}
		http.Redirect(w, r, setupUIPath+"?setup_token="+url.QueryEscape(setupToken), http.StatusFound)
	})

	mux.HandleFunc(webhookPath, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		payload, err := io.ReadAll(io.LimitReader(r.Body, 1<<20))
		if err != nil {
			http.Error(w, "invalid payload", http.StatusBadRequest)
			return
		}
		if !validGitHubSignature(payload, githubSecret, r.Header.Get("X-Hub-Signature-256")) {
			http.Error(w, "invalid signature", http.StatusUnauthorized)
			return
		}

		eventName := strings.TrimSpace(r.Header.Get("X-GitHub-Event"))
		deliveryID := strings.TrimSpace(r.Header.Get("X-GitHub-Delivery"))
		installationID, hasInstallation := extractInstallationID(payload)

		publishClient := defaultClient
		convertCfg := defaultConvertCfg
		if hasInstallation {
			mapping, mapErr := installStore.GetInstallationMapping(installationID)
			if mapErr == nil && mapping.Enabled {
				publishClient = eventpublisher.Client{
					Endpoint: mapping.DDashEndpoint,
					Token:    mapping.DDashAuthToken,
					Secret:   mapping.DDashWebhookSecret,
					Timeout:  10 * time.Second,
				}
				if strings.TrimSpace(mapping.DefaultEnvironment) != "" {
					convertCfg.DefaultEnvironment = strings.TrimSpace(mapping.DefaultEnvironment)
				}
			} else if mapErr != nil && mapErr != sql.ErrNoRows {
				slog.Error("failed to load installation mapping", "installation_id", installationID, "error", mapErr)
				http.Error(w, "mapping resolution failed", http.StatusInternalServerError)
				return
			}
		}

		if strings.TrimSpace(publishClient.Endpoint) == "" || strings.TrimSpace(publishClient.Token) == "" || strings.TrimSpace(publishClient.Secret) == "" {
			slog.Warn("ignoring webhook because no DDash mapping/default credentials are available", "installation_id", installationID, "event", eventName)
			w.WriteHeader(http.StatusAccepted)
			_, _ = w.Write([]byte("ignored"))
			return
		}

		events, err := githubbridge.Convert(eventName, deliveryID, payload, convertCfg)
		if err != nil {
			http.Error(w, "invalid github payload", http.StatusBadRequest)
			return
		}
		if len(events) == 0 {
			w.WriteHeader(http.StatusAccepted)
			_, _ = w.Write([]byte("ignored"))
			return
		}

		published := 0
		for _, event := range events {
			if _, err := publishClient.Publish(r.Context(), event); err != nil {
				slog.Error("failed to publish converted event", "github_event", eventName, "installation_id", installationID, "error", err)
				http.Error(w, "publish failed", http.StatusBadGateway)
				return
			}
			published++
		}
		w.WriteHeader(http.StatusAccepted)
		_, _ = w.Write([]byte(fmt.Sprintf("published=%d", published)))
	})

	slog.Info("GitHub App ingestor listening", "addr", listenAddr, "webhook_path", webhookPath, "setup_ui_path", setupUIPath, "setup_start_path", setupStartPath, "setup_callback_path", setupCallbackPath)
	if err := http.ListenAndServe(listenAddr, mux); err != nil {
		slog.Error("github app ingestor stopped", "error", err)
		os.Exit(1)
	}
}

type setupPageData struct {
	InstallURL           string
	SetupToken           string
	SetupStartPath       string
	SetupDeletePath      string
	DefaultDDashEndpoint string
	DefaultDDashToken    string
	DefaultDDashSecret   string
	DefaultEnvironment   string
	Mappings             []githubbridge.InstallationMapping
}

type setupStartPayload struct {
	OrganizationID     int64  `json:"organization_id"`
	OrganizationLabel  string `json:"organization_label"`
	DDashEndpoint      string `json:"ddash_endpoint"`
	DDashAuthToken     string `json:"ddash_auth_token"`
	DDashWebhookSecret string `json:"ddash_webhook_secret"`
	DefaultEnvironment string `json:"default_environment"`
}

func renderSetupPage(w http.ResponseWriter, data setupPageData) error {
	tmpl := template.Must(template.New("setup").Funcs(template.FuncMap{
		"mask": maskValue,
	}).Parse(setupPageHTML))
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	return tmpl.Execute(w, data)
}

func maskValue(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	if len(value) <= 8 {
		return "****"
	}
	return value[:4] + "..." + value[len(value)-4:]
}

const setupPageHTML = `<!doctype html>
<html>
<head>
  <meta charset="utf-8" />
  <meta name="viewport" content="width=device-width, initial-scale=1" />
  <title>GitHub App Setup</title>
  <style>
    body { font-family: ui-sans-serif, system-ui, -apple-system, Segoe UI, Roboto, Arial; margin: 2rem; color: #111827; }
    .card { border: 1px solid #e5e7eb; border-radius: 10px; padding: 1rem; margin-bottom: 1rem; background: #fff; }
    .grid { display: grid; grid-template-columns: repeat(2, minmax(0, 1fr)); gap: .75rem; }
    .full { grid-column: 1 / -1; }
    label { font-size: 12px; color: #6b7280; display:block; margin-bottom: .25rem; }
    input { width: 100%; height: 36px; border: 1px solid #d1d5db; border-radius: 8px; padding: 0 .5rem; }
    button, .btn { height: 36px; border: 1px solid #111827; background: #111827; color: white; border-radius: 8px; padding: 0 .75rem; cursor: pointer; text-decoration: none; display: inline-flex; align-items: center; }
    table { width: 100%; border-collapse: collapse; }
    th, td { border-bottom: 1px solid #e5e7eb; text-align: left; padding: .5rem; font-size: 14px; }
    .muted { color: #6b7280; font-size: 12px; }
    .danger { background: #fff; color: #991b1b; border-color: #ef4444; }
  </style>
</head>
<body>
  <h1>GitHub App Installation Mapping</h1>
  <p class="muted">Create setup intents and map GitHub installation IDs to DDash organization credentials.</p>

  <div class="card">
    <h3>Create installation intent</h3>
    <form method="get" action="{{.SetupStartPath}}">
      <div class="grid">
        <div><label>Organization label</label><input name="organization" placeholder="acme-prod" /></div>
        <div><label>Default environment</label><input name="default_environment" value="{{.DefaultEnvironment}}" /></div>
        <div class="full"><label>DDash endpoint</label><input name="ddash_endpoint" value="{{.DefaultDDashEndpoint}}" /></div>
        <div><label>DDash auth token</label><input name="ddash_auth_token" value="{{.DefaultDDashToken}}" /></div>
        <div><label>DDash webhook secret</label><input name="ddash_webhook_secret" value="{{.DefaultDDashSecret}}" /></div>
        <div class="full"><label>Setup token</label><input name="setup_token" value="{{.SetupToken}}" /></div>
      </div>
      <div style="margin-top:.75rem; display:flex; gap:.5rem; align-items:center;">
        <button type="submit">Create setup and continue install</button>
        {{if .InstallURL}}<span class="muted">Install URL configured</span>{{else}}<span class="muted">Install URL not configured (returns JSON state only)</span>{{end}}
      </div>
    </form>
  </div>

  <div class="card">
    <h3>Mapped installations</h3>
    {{if .Mappings}}
      <table>
        <thead><tr><th>Installation</th><th>Organization</th><th>Endpoint</th><th>Token</th><th>Secret</th><th>Default env</th><th>Status</th><th>Action</th></tr></thead>
        <tbody>
          {{range .Mappings}}
            <tr>
              <td>{{.InstallationID}}</td>
              <td>{{.OrganizationLabel}}</td>
              <td>{{.DDashEndpoint}}</td>
              <td>{{mask .DDashAuthToken}}</td>
              <td>{{mask .DDashWebhookSecret}}</td>
              <td>{{.DefaultEnvironment}}</td>
              <td>{{if .Enabled}}enabled{{else}}disabled{{end}}</td>
              <td>
                <form method="post" action="{{$.SetupDeletePath}}" style="display:inline;">
                  <input type="hidden" name="installation_id" value="{{.InstallationID}}" />
                  <input type="hidden" name="setup_token" value="{{$.SetupToken}}" />
                  <button class="danger" type="submit">Revoke</button>
                </form>
              </td>
            </tr>
          {{end}}
        </tbody>
      </table>
    {{else}}
      <p class="muted">No installation mappings yet.</p>
    {{end}}
  </div>
</body>
</html>`

func writeJSON(w http.ResponseWriter, payload any) error {
	w.Header().Set("Content-Type", "application/json")
	return json.NewEncoder(w).Encode(payload)
}

func createSetupIntent(store *githubbridge.InstallStore, installURL, defaultEnv string, payload setupStartPayload) (githubbridge.SetupIntent, string, error) {
	ddashEndpoint := strings.TrimSpace(payload.DDashEndpoint)
	ddashToken := strings.TrimSpace(payload.DDashAuthToken)
	ddashSecret := strings.TrimSpace(payload.DDashWebhookSecret)
	if ddashEndpoint == "" || ddashToken == "" || ddashSecret == "" {
		return githubbridge.SetupIntent{}, "", fmt.Errorf("ddash_endpoint, ddash_auth_token, ddash_webhook_secret are required")
	}
	env := strings.TrimSpace(payload.DefaultEnvironment)
	if env == "" {
		env = strings.TrimSpace(defaultEnv)
	}

	state, err := randomHex(16)
	if err != nil {
		return githubbridge.SetupIntent{}, "", fmt.Errorf("failed to create setup state")
	}
	intent := githubbridge.SetupIntent{
		State:              state,
		OrganizationID:     payload.OrganizationID,
		OrganizationLabel:  strings.TrimSpace(payload.OrganizationLabel),
		DDashEndpoint:      ddashEndpoint,
		DDashAuthToken:     ddashToken,
		DDashWebhookSecret: ddashSecret,
		DefaultEnvironment: env,
		ExpiresAt:          time.Now().UTC().Add(15 * time.Minute),
	}
	if err := store.CreateSetupIntent(intent); err != nil {
		return githubbridge.SetupIntent{}, "", fmt.Errorf("failed to save setup intent")
	}

	if strings.TrimSpace(installURL) == "" {
		return intent, "/setup/callback?state=" + url.QueryEscape(intent.State), nil
	}
	redirectURL, err := appendStateToURL(installURL, intent.State)
	if err != nil {
		return githubbridge.SetupIntent{}, "", fmt.Errorf("invalid install url")
	}
	return intent, redirectURL, nil
}

func extractInstallationID(payload []byte) (int64, bool) {
	var body map[string]any
	if err := json.Unmarshal(payload, &body); err != nil {
		return 0, false
	}
	installationRaw, ok := body["installation"]
	if !ok {
		return 0, false
	}
	installation, ok := installationRaw.(map[string]any)
	if !ok {
		return 0, false
	}
	idRaw, ok := installation["id"]
	if !ok {
		return 0, false
	}
	switch typed := idRaw.(type) {
	case float64:
		if typed <= 0 {
			return 0, false
		}
		return int64(typed), true
	case int64:
		return typed, typed > 0
	case string:
		parsed, err := strconv.ParseInt(strings.TrimSpace(typed), 10, 64)
		if err != nil || parsed <= 0 {
			return 0, false
		}
		return parsed, true
	default:
		return 0, false
	}
}

func appendStateToURL(rawURL, state string) (string, error) {
	parsed, err := url.Parse(strings.TrimSpace(rawURL))
	if err != nil {
		return "", err
	}
	query := parsed.Query()
	query.Set("state", strings.TrimSpace(state))
	parsed.RawQuery = query.Encode()
	return parsed.String(), nil
}

func randomHex(size int) (string, error) {
	buf := make([]byte, size)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return hex.EncodeToString(buf), nil
}

func authorizedSetupRequest(r *http.Request, token string) bool {
	token = strings.TrimSpace(token)
	if token == "" {
		return true
	}
	auth := strings.TrimSpace(r.Header.Get("Authorization"))
	if strings.HasPrefix(strings.ToLower(auth), "bearer ") {
		if strings.TrimSpace(strings.TrimPrefix(auth, "Bearer ")) == token || strings.TrimSpace(strings.TrimPrefix(auth, "bearer ")) == token {
			return true
		}
	}
	if strings.TrimSpace(r.URL.Query().Get("setup_token")) == token {
		return true
	}
	if err := r.ParseForm(); err == nil {
		if strings.TrimSpace(r.FormValue("setup_token")) == token {
			return true
		}
	}
	return false
}

func valueOrDefault(value, fallback string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return fallback
	}
	return value
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed != "" {
			return trimmed
		}
	}
	return ""
}

func validGitHubSignature(body []byte, secret, signatureHeader string) bool {
	signatureHeader = strings.TrimSpace(strings.ToLower(signatureHeader))
	if !strings.HasPrefix(signatureHeader, "sha256=") {
		return false
	}
	received := strings.TrimPrefix(signatureHeader, "sha256=")
	mac := hmac.New(sha256.New, []byte(secret))
	_, _ = mac.Write(body)
	expected := hex.EncodeToString(mac.Sum(nil))
	return hmac.Equal([]byte(expected), []byte(received))
}
