package custom

import (
	"io"
	"net/http"

	"github.com/fr0stylo/ddash/apps/ddash/internal/app/ports"
	appingest "github.com/fr0stylo/ddash/apps/ddash/internal/application/ingestion"
)

const (
	// SignatureHeader is the HMAC signature header.
	SignatureHeader = "X-Webhook-Signature"
	// AuthorizationHeader contains the bearer token.
	AuthorizationHeader = "Authorization"
	maxPayloadBytes     = 1 << 20
)

// Handler processes CDEvents delivery payloads.
type Handler struct {
	ingest *appingest.Service
}

// NewHandler constructs a CDEvents webhook handler.
func NewHandler(storeFactory ports.IngestionStoreFactory, batchConfig appingest.BatchConfig) *Handler {
	return &Handler{ingest: appingest.NewService(storeFactory, batchConfig)}
}

// Handle validates and processes a webhook request.
func (h *Handler) Handle(w http.ResponseWriter, r *http.Request) error {
	body, readErr := io.ReadAll(io.LimitReader(r.Body, maxPayloadBytes))
	if readErr != nil {
		http.Error(w, "invalid payload", http.StatusBadRequest)
		return readErr
	}
	ingestErr := h.ingest.Ingest(r.Context(), appingest.Command{
		AuthorizationHeader: r.Header.Get(AuthorizationHeader),
		SignatureHeader:     r.Header.Get(SignatureHeader),
		Headers:             r.Header,
		Body:                body,
	})
	if handled := writeIngestHTTPError(w, ingestErr); handled {
		return nil
	}
	if ingestErr != nil {
		return ingestErr
	}

	w.WriteHeader(http.StatusAccepted)
	return nil
}

func writeIngestHTTPError(w http.ResponseWriter, err error) bool {
	switch appingest.ClassifyError(err) {
	case appingest.ErrorUnknown:
		return false
	case appingest.ErrorMissingAuth:
		http.Error(w, appingest.ErrMissingAuthToken.Error(), http.StatusUnauthorized)
		return true
	case appingest.ErrorInvalidAuth:
		http.Error(w, appingest.ErrInvalidAuthToken.Error(), http.StatusUnauthorized)
		return true
	case appingest.ErrorInvalidSignature:
		http.Error(w, "invalid signature", http.StatusUnauthorized)
		return true
	case appingest.ErrorInvalidPayload:
		http.Error(w, "invalid cdevent payload", http.StatusBadRequest)
		return true
	case appingest.ErrorInvalidSchema:
		http.Error(w, "invalid cdevent schema", http.StatusBadRequest)
		return true
	case appingest.ErrorUnsupportedType:
		http.Error(w, "unsupported cdevent type", http.StatusUnprocessableEntity)
		return true
	case appingest.ErrorBusy:
		http.Error(w, "ingestion busy", http.StatusServiceUnavailable)
		return true
	}

	return false
}
