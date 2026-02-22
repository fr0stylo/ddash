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
}

func NewGitLabAppHandler(storeFactory ports.IngestionStoreFactory, batchConfig appingest.BatchConfig, resolver GitLabProjectResolver, ingestorKey string) *GitLabAppHandler {
	return &GitLabAppHandler{
		ingest:      appingest.NewService(storeFactory, batchConfig),
		resolver:    resolver,
		ingestorKey: strings.TrimSpace(ingestorKey),
	}
}

func (h *GitLabAppHandler) Handle(w http.ResponseWriter, r *http.Request) error {
	if strings.TrimSpace(h.ingestorKey) == "" {
		http.Error(w, "gitlab ingestion is not configured", http.StatusServiceUnavailable)
		return nil
	}
	if !validBearerToken(r.Header.Get(IngestorAuthHeader), h.ingestorKey) {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return nil
	}
	if h.resolver == nil {
		http.Error(w, "gitlab project resolver unavailable", http.StatusServiceUnavailable)
		return nil
	}

	projectID, err := strconv.ParseInt(strings.TrimSpace(r.Header.Get(GitLabProjectIDHeader)), 10, 64)
	if err != nil || projectID <= 0 {
		http.Error(w, "invalid gitlab project id", http.StatusBadRequest)
		return nil
	}

	body, readErr := io.ReadAll(io.LimitReader(r.Body, maxPayloadBytes))
	if readErr != nil {
		http.Error(w, "invalid payload", http.StatusBadRequest)
		return readErr
	}

	org, resolveErr := h.resolver.GetOrganizationByGitLabProjectID(r.Context(), projectID)
	if resolveErr != nil {
		if resolveErr == sql.ErrNoRows {
			http.Error(w, "unknown gitlab project", http.StatusNotFound)
			return nil
		}
		return resolveErr
	}

	ingestErr := h.ingest.IngestForOrganization(r.Context(), org.ID, r.Header, body)
	if handled := writeIngestHTTPError(w, ingestErr); handled {
		return nil
	}
	if ingestErr != nil {
		return ingestErr
	}

	w.WriteHeader(http.StatusAccepted)
	return nil
}
