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

const GitLabProjectIDHeader = "X-GitLab-Project-ID"

type GitLabProjectResolver interface {
	GetOrganizationByGitLabProjectID(ctx context.Context, projectID int64) (ports.Organization, error)
}

type GitLabAppHandler struct {
	ingest      *appingest.Service
	resolver    GitLabProjectResolver
	ingestorKey string
	metrics     providerIngestionMetrics
}

func NewGitLabAppHandler(storeFactory ports.IngestionStoreFactory, batchConfig appingest.BatchConfig, resolver GitLabProjectResolver, ingestorKey string) *GitLabAppHandler {
	return &GitLabAppHandler{
		ingest:      appingest.NewService(storeFactory, batchConfig),
		resolver:    resolver,
		ingestorKey: strings.TrimSpace(ingestorKey),
		metrics:     newProviderIngestionMetrics(),
	}
}

func (h *GitLabAppHandler) Handle(w http.ResponseWriter, r *http.Request) error {
	h.metrics.recordRequest(r.Context(), "gitlab")
	if strings.TrimSpace(h.ingestorKey) == "" {
		h.metrics.recordRejected(r.Context(), "gitlab", "not_configured")
		http.Error(w, "gitlab ingestion is not configured", http.StatusServiceUnavailable)
		return nil
	}
	if !validBearerToken(r.Header.Get(IngestorAuthHeader), h.ingestorKey) {
		h.metrics.recordRejected(r.Context(), "gitlab", "unauthorized")
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return nil
	}
	if h.resolver == nil {
		h.metrics.recordRejected(r.Context(), "gitlab", "resolver_unavailable")
		http.Error(w, "gitlab project resolver unavailable", http.StatusServiceUnavailable)
		return nil
	}

	projectID, err := strconv.ParseInt(strings.TrimSpace(r.Header.Get(GitLabProjectIDHeader)), 10, 64)
	if err != nil || projectID <= 0 {
		h.metrics.recordRejected(r.Context(), "gitlab", "invalid_project_id")
		http.Error(w, "invalid gitlab project id", http.StatusBadRequest)
		return nil
	}

	body, readErr := io.ReadAll(io.LimitReader(r.Body, maxPayloadBytes))
	if readErr != nil {
		h.metrics.recordRejected(r.Context(), "gitlab", "invalid_payload")
		http.Error(w, "invalid payload", http.StatusBadRequest)
		return readErr
	}

	org, resolveErr := h.resolver.GetOrganizationByGitLabProjectID(r.Context(), projectID)
	if resolveErr != nil {
		if resolveErr == sql.ErrNoRows {
			h.metrics.recordMapping(r.Context(), "gitlab", "miss")
			h.metrics.recordRejected(r.Context(), "gitlab", "mapping_miss")
			http.Error(w, "unknown gitlab project", http.StatusNotFound)
			return nil
		}
		h.metrics.recordMapping(r.Context(), "gitlab", "error")
		h.metrics.recordRejected(r.Context(), "gitlab", "mapping_error")
		return resolveErr
	}
	h.metrics.recordMapping(r.Context(), "gitlab", "hit")

	ingestErr := h.ingest.IngestForOrganization(r.Context(), org.ID, r.Header, body)
	if handled := writeIngestHTTPError(w, ingestErr); handled {
		h.metrics.recordRejected(r.Context(), "gitlab", string(appingest.ClassifyError(ingestErr)))
		return nil
	}
	if ingestErr != nil {
		h.metrics.recordRejected(r.Context(), "gitlab", "internal_error")
		return ingestErr
	}

	h.metrics.recordAccepted(r.Context(), "gitlab")
	w.WriteHeader(http.StatusAccepted)
	return nil
}
