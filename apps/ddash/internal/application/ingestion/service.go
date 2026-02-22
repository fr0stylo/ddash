package ingestion

import (
	"context"
	"net/http"

	"github.com/fr0stylo/ddash/apps/ddash/internal/app/ports"
	appservices "github.com/fr0stylo/ddash/apps/ddash/internal/app/services"
)

type BatchConfig = appservices.IngestBatchConfig
type Command = appservices.IngestCommand
type ErrorKind = appservices.IngestErrorKind

const (
	ErrorUnknown          ErrorKind = appservices.IngestErrorUnknown
	ErrorMissingAuth      ErrorKind = appservices.IngestErrorMissingAuth
	ErrorInvalidAuth      ErrorKind = appservices.IngestErrorInvalidAuth
	ErrorInvalidSignature ErrorKind = appservices.IngestErrorInvalidSignature
	ErrorInvalidPayload   ErrorKind = appservices.IngestErrorInvalidPayload
	ErrorInvalidSchema    ErrorKind = appservices.IngestErrorInvalidSchema
	ErrorUnsupportedType  ErrorKind = appservices.IngestErrorUnsupportedType
	ErrorBusy             ErrorKind = appservices.IngestErrorBusy
)

var (
	ErrMissingAuthToken = appservices.ErrMissingAuthToken
	ErrInvalidAuthToken = appservices.ErrInvalidAuthToken
)

type Service struct {
	delegate *appservices.EventIngestService
}

func NewService(storeFactory ports.IngestionStoreFactory, batchConfig BatchConfig) *Service {
	return &Service{delegate: appservices.NewEventIngestServiceWithConfig(storeFactory, batchConfig)}
}

func (s *Service) Ingest(ctx context.Context, cmd Command) error {
	return s.delegate.Ingest(ctx, cmd)
}

func (s *Service) IngestForOrganization(ctx context.Context, organizationID int64, headers http.Header, body []byte) error {
	return s.delegate.IngestForOrganization(ctx, organizationID, headers, body)
}

func ClassifyError(err error) ErrorKind {
	return appservices.ClassifyIngestError(err)
}
