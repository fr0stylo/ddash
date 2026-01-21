package custom

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"path/filepath"
	"strings"
	"time"

	"github.com/fr0stylo/ddash/internal/db"
	"github.com/fr0stylo/ddash/internal/db/queries"
)

const (
	// SignatureHeader is the HMAC signature header.
	SignatureHeader = "X-Webhook-Signature"
	// AuthorizationHeader contains the bearer token.
	AuthorizationHeader = "Authorization"
	// BearerPrefix prefixes the auth token.
	BearerPrefix    = "Bearer "
	maxPayloadBytes = 1 << 20
)

var errUnauthorized = errors.New("unauthorized")

// Handler processes custom webhook payloads.
type Handler struct {
	baseDir string
}

// Payload represents the incoming webhook payload.
type Payload struct {
	Name        string `json:"name"`
	Environment string `json:"environment"`
	Reference   string `json:"reference"`
}

// NewHandler constructs a custom webhook handler.
func NewHandler(baseDir string) *Handler {
	if strings.TrimSpace(baseDir) == "" {
		baseDir = "data"
	}
	return &Handler{baseDir: baseDir}
}

// Handle validates and processes a webhook request.
func (h *Handler) Handle(w http.ResponseWriter, r *http.Request) error {
	token, err := bearerToken(r.Header.Get(AuthorizationHeader))
	if err != nil {
		http.Error(w, "missing auth token", http.StatusUnauthorized)
		return nil
	}

	database, org, err := h.lookupOrganization(r.Context(), token)
	if err != nil {
		if errors.Is(err, errUnauthorized) {
			http.Error(w, "invalid auth token", http.StatusUnauthorized)
			return nil
		}
		return err
	}
	defer func() {
		_ = database.Close()
	}()

	body, err := io.ReadAll(io.LimitReader(r.Body, maxPayloadBytes))
	if err != nil {
		http.Error(w, "invalid payload", http.StatusBadRequest)
		return nil
	}
	if !validSignature(body, org.WebhookSecret, r.Header.Get(SignatureHeader)) {
		http.Error(w, "invalid signature", http.StatusUnauthorized)
		return nil
	}

	var payload Payload
	decoder := json.NewDecoder(bytes.NewReader(body))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&payload); err != nil {
		http.Error(w, "invalid payload", http.StatusBadRequest)
		return nil
	}

	payload.Name = strings.TrimSpace(payload.Name)
	payload.Environment = strings.TrimSpace(payload.Environment)
	payload.Reference = strings.TrimSpace(payload.Reference)
	if payload.Name == "" || payload.Environment == "" || payload.Reference == "" {
		http.Error(w, "missing required fields", http.StatusBadRequest)
		return nil
	}

	if err := h.applyDeployment(r.Context(), database, payload); err != nil {
		return err
	}

	w.WriteHeader(http.StatusAccepted)
	return nil
}

func bearerToken(value string) (string, error) {
	trimmed := strings.TrimSpace(value)
	if !strings.HasPrefix(trimmed, BearerPrefix) {
		return "", errUnauthorized
	}
	return strings.TrimSpace(strings.TrimPrefix(trimmed, BearerPrefix)), nil
}

func (h *Handler) lookupOrganization(ctx context.Context, token string) (*db.Database, queries.Organization, error) {
	token = strings.TrimSpace(token)
	if token == "" {
		return nil, queries.Organization{}, errUnauthorized
	}
	if strings.Contains(token, "..") || strings.ContainsAny(token, `/\\`) {
		return nil, queries.Organization{}, errUnauthorized
	}

	database, err := db.New(filepath.Join(h.baseDir, token))
	if err != nil {
		return nil, queries.Organization{}, err
	}

	org, err := database.GetOrganizationByAuthToken(ctx, token)
	if err != nil {
		_ = database.Close()
		if errors.Is(err, sql.ErrNoRows) {
			return nil, queries.Organization{}, errUnauthorized
		}
		return nil, queries.Organization{}, err
	}
	if org.Enabled == 0 {
		_ = database.Close()
		return nil, queries.Organization{}, errUnauthorized
	}

	return database, org, nil
}

func validSignature(body []byte, secret, signature string) bool {
	signature = strings.ToLower(strings.TrimSpace(signature))
	if signature == "" {
		return false
	}
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(body)
	expected := hex.EncodeToString(mac.Sum(nil))
	return hmac.Equal([]byte(expected), []byte(signature))
}

func (h *Handler) applyDeployment(ctx context.Context, database *db.Database, payload Payload) error {
	service, err := h.getOrCreateService(ctx, database, payload.Name)
	if err != nil {
		return err
	}
	if err := database.MarkServiceIntegrationType(ctx, "custom", service.ID); err != nil {
		return err
	}

	environment, err := h.getOrCreateEnvironment(ctx, database, payload.Environment)
	if err != nil {
		return err
	}

	timestamp := time.Now().Format(time.RFC3339)
	_, err = database.UpsertServiceInstance(ctx, queries.UpsertServiceInstanceParams{
		ServiceID:      service.ID,
		EnvironmentID:  environment.ID,
		Status:         "synced",
		LastDeployAt:   sql.NullString{String: timestamp, Valid: true},
		Revision:       sql.NullString{String: payload.Reference, Valid: true},
		CommitSha:      sql.NullString{String: payload.Reference, Valid: true},
		CommitUrl:      sql.NullString{},
		ActionLabel:    sql.NullString{},
		ActionKind:     sql.NullString{},
		ActionDisabled: 0,
	})
	if err != nil {
		return err
	}

	_, err = database.CreateDeploymentSimple(ctx, queries.CreateDeploymentSimpleParams{
		ServiceID:     service.ID,
		EnvironmentID: environment.ID,
		DeployedAt:    timestamp,
		Status:        "success",
		ReleaseRef:    sql.NullString{String: payload.Reference, Valid: true},
	})
	if err != nil {
		return err
	}

	return nil
}

func (h *Handler) getOrCreateService(ctx context.Context, database *db.Database, name string) (queries.Service, error) {
	service, err := database.GetServiceByName(ctx, name)
	if err == nil {
		return service, nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return queries.Service{}, err
	}
	service, err = database.CreateService(ctx, name)
	if err != nil {
		return database.GetServiceByName(ctx, name)
	}
	return service, nil
}

func (h *Handler) getOrCreateEnvironment(ctx context.Context, database *db.Database, name string) (queries.Environment, error) {
	environment, err := database.GetEnvironmentByName(ctx, name)
	if err == nil {
		return environment, nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return queries.Environment{}, err
	}
	environment, err = database.CreateEnvironment(ctx, name)
	if err != nil {
		return database.GetEnvironmentByName(ctx, name)
	}
	return environment, nil
}
