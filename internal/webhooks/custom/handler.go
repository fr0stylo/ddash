package custom

import (
	"io"
	"net/http"

	"github.com/fr0stylo/ddash/internal/app/ports"
	"github.com/fr0stylo/ddash/internal/app/services"
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
	ingest *services.EventIngestService
}

// NewHandler constructs a CDEvents webhook handler.
func NewHandler(storeFactory ports.IngestionStoreFactory, batchConfig services.IngestBatchConfig) *Handler {
	return &Handler{ingest: services.NewEventIngestServiceWithConfig(storeFactory, batchConfig)}
}

// Handle validates and processes a webhook request.
func (h *Handler) Handle(w http.ResponseWriter, r *http.Request) error {
	body, readErr := io.ReadAll(io.LimitReader(r.Body, maxPayloadBytes))
	if readErr != nil {
		http.Error(w, "invalid payload", http.StatusBadRequest)
		return readErr
	}
	ingestErr := h.ingest.Ingest(r.Context(), services.IngestCommand{
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
	switch services.ClassifyIngestError(err) {
	case services.IngestErrorUnknown:
		return false
	case services.IngestErrorMissingAuth:
		http.Error(w, services.ErrMissingAuthToken.Error(), http.StatusUnauthorized)
		return true
	case services.IngestErrorInvalidAuth:
		http.Error(w, services.ErrInvalidAuthToken.Error(), http.StatusUnauthorized)
		return true
	case services.IngestErrorInvalidSignature:
		http.Error(w, "invalid signature", http.StatusUnauthorized)
		return true
	case services.IngestErrorInvalidPayload:
		http.Error(w, "invalid cdevent payload", http.StatusBadRequest)
		return true
	case services.IngestErrorInvalidSchema:
		http.Error(w, "invalid cdevent schema", http.StatusBadRequest)
		return true
	case services.IngestErrorUnsupportedType:
		http.Error(w, "unsupported cdevent type", http.StatusUnprocessableEntity)
		return true
	case services.IngestErrorBusy:
		http.Error(w, "ingestion busy", http.StatusServiceUnavailable)
		return true
	}

	return false
}
