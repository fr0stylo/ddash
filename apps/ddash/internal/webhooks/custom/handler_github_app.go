package custom

import (
	"context"
	"database/sql"
	"io"
	"net/http"
	"strconv"
	"strings"

	"github.com/fr0stylo/ddash/apps/ddash/internal/app/ports"
	appingest "github.com/fr0stylo/ddash/apps/ddash/internal/application/ingestion"
)

const (
	GitHubInstallationIDHeader = "X-GitHub-Installation-ID"
	IngestorAuthHeader         = "Authorization"
)

type GitHubInstallationResolver interface {
	GetOrganizationByGitHubInstallationID(ctx context.Context, installationID int64) (ports.Organization, error)
}

type GitHubAppHandler struct {
	ingest      *appingest.Service
	resolver    GitHubInstallationResolver
	ingestorKey string
	metrics     providerIngestionMetrics
}

func NewGitHubAppHandler(storeFactory ports.IngestionStoreFactory, batchConfig appingest.BatchConfig, resolver GitHubInstallationResolver, ingestorKey string) *GitHubAppHandler {
	return &GitHubAppHandler{
		ingest:      appingest.NewService(storeFactory, batchConfig),
		resolver:    resolver,
		ingestorKey: strings.TrimSpace(ingestorKey),
		metrics:     newProviderIngestionMetrics(),
	}
}

func (h *GitHubAppHandler) Handle(w http.ResponseWriter, r *http.Request) error {
	h.metrics.recordRequest(r.Context(), "github")
	if strings.TrimSpace(h.ingestorKey) == "" {
		h.metrics.recordRejected(r.Context(), "github", "not_configured")
		http.Error(w, "github app ingestion is not configured", http.StatusServiceUnavailable)
		return nil
	}
	if !validBearerToken(r.Header.Get(IngestorAuthHeader), h.ingestorKey) {
		h.metrics.recordRejected(r.Context(), "github", "unauthorized")
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return nil
	}
	if h.resolver == nil {
		h.metrics.recordRejected(r.Context(), "github", "resolver_unavailable")
		http.Error(w, "github installation resolver unavailable", http.StatusServiceUnavailable)
		return nil
	}

	installationID, err := strconv.ParseInt(strings.TrimSpace(r.Header.Get(GitHubInstallationIDHeader)), 10, 64)
	if err != nil || installationID <= 0 {
		h.metrics.recordRejected(r.Context(), "github", "invalid_installation_id")
		http.Error(w, "invalid github installation id", http.StatusBadRequest)
		return nil
	}

	body, readErr := io.ReadAll(io.LimitReader(r.Body, maxPayloadBytes))
	if readErr != nil {
		h.metrics.recordRejected(r.Context(), "github", "invalid_payload")
		http.Error(w, "invalid payload", http.StatusBadRequest)
		return readErr
	}

	org, resolveErr := h.resolver.GetOrganizationByGitHubInstallationID(r.Context(), installationID)
	if resolveErr != nil {
		if resolveErr == sql.ErrNoRows {
			h.metrics.recordMapping(r.Context(), "github", "miss")
			h.metrics.recordRejected(r.Context(), "github", "mapping_miss")
			http.Error(w, "unknown github installation", http.StatusNotFound)
			return nil
		}
		h.metrics.recordMapping(r.Context(), "github", "error")
		h.metrics.recordRejected(r.Context(), "github", "mapping_error")
		return resolveErr
	}
	h.metrics.recordMapping(r.Context(), "github", "hit")

	ingestErr := h.ingest.IngestForOrganization(r.Context(), org.ID, r.Header, body)
	if handled := writeIngestHTTPError(w, ingestErr); handled {
		h.metrics.recordRejected(r.Context(), "github", string(appingest.ClassifyError(ingestErr)))
		return nil
	}
	if ingestErr != nil {
		h.metrics.recordRejected(r.Context(), "github", "internal_error")
		return ingestErr
	}

	h.metrics.recordAccepted(r.Context(), "github")
	w.WriteHeader(http.StatusAccepted)
	return nil
}

func validBearerToken(raw, expected string) bool {
	raw = strings.TrimSpace(raw)
	expected = strings.TrimSpace(expected)
	if raw == "" || expected == "" {
		return false
	}
	if !strings.HasPrefix(strings.ToLower(raw), "bearer ") {
		return false
	}
	token := strings.TrimSpace(raw[len("Bearer "):])
	if token == "" {
		return false
	}
	return token == expected
}
