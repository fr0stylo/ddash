package services

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
	"strings"
	"time"

	cdeventsapi "github.com/cdevents/sdk-go/pkg/api"
	cdeventsv05 "github.com/cdevents/sdk-go/pkg/api/v05"
	cebinding "github.com/cloudevents/sdk-go/v2/binding"
	ceevent "github.com/cloudevents/sdk-go/v2/event"
	cehttp "github.com/cloudevents/sdk-go/v2/protocol/http"

	"github.com/fr0stylo/ddash/internal/app/ports"
)

var (
	// ErrMissingAuthToken indicates missing bearer authorization token.
	ErrMissingAuthToken = errors.New("missing auth token")
	// ErrInvalidAuthToken indicates unknown or disabled organization token.
	ErrInvalidAuthToken = errors.New("invalid auth token")
	// ErrInvalidSignature indicates request signature validation failure.
	ErrInvalidSignature = errors.New("invalid signature")
	// ErrInvalidPayload indicates malformed CDEvent/CloudEvent payload.
	ErrInvalidPayload = errors.New("invalid payload")
	// ErrInvalidSchema indicates payload fails CDEvents schema validation.
	ErrInvalidSchema = errors.New("invalid schema")
	// ErrUnsupportedType indicates event type is not currently accepted.
	ErrUnsupportedType = errors.New("unsupported event type")
)

const bearerPrefix = "Bearer "

var allowedDeliveryTypes = map[string]struct{}{
	cdeventsv05.EnvironmentCreatedEventType.String():  {},
	cdeventsv05.EnvironmentModifiedEventType.String(): {},
	cdeventsv05.EnvironmentDeletedEventType.String():  {},
	cdeventsv05.ServiceDeployedEventType.String():     {},
	cdeventsv05.ServiceUpgradedEventType.String():     {},
	cdeventsv05.ServiceRolledbackEventType.String():   {},
	cdeventsv05.ServiceRemovedEventType.String():      {},
	cdeventsv05.ServicePublishedEventType.String():    {},
}

// EventIngestService validates and appends CDEvents to the event store.
type EventIngestService struct {
	storeFactory ports.IngestionStoreFactory
}

// IngestErrorKind classifies ingestion failures for transport-specific mapping.
type IngestErrorKind string

const (
	// IngestErrorUnknown is used when error is nil or not classified.
	IngestErrorUnknown IngestErrorKind = "unknown"
	// IngestErrorMissingAuth indicates missing bearer authorization header.
	IngestErrorMissingAuth IngestErrorKind = "missing_auth"
	// IngestErrorInvalidAuth indicates unknown or disabled organization token.
	IngestErrorInvalidAuth IngestErrorKind = "invalid_auth"
	// IngestErrorInvalidSignature indicates signature mismatch.
	IngestErrorInvalidSignature IngestErrorKind = "invalid_signature"
	// IngestErrorInvalidPayload indicates malformed event payload.
	IngestErrorInvalidPayload IngestErrorKind = "invalid_payload"
	// IngestErrorInvalidSchema indicates CDEvents schema validation failure.
	IngestErrorInvalidSchema IngestErrorKind = "invalid_schema"
	// IngestErrorUnsupportedType indicates event type is not accepted.
	IngestErrorUnsupportedType IngestErrorKind = "unsupported_type"
)

// IngestCommand is transport-agnostic webhook ingestion input.
type IngestCommand struct {
	AuthorizationHeader string
	SignatureHeader     string
	Headers             http.Header
	Body                []byte
}

// NewEventIngestService constructs an ingestion service.
func NewEventIngestService(storeFactory ports.IngestionStoreFactory) *EventIngestService {
	return &EventIngestService{storeFactory: storeFactory}
}

// ClassifyIngestError classifies a returned ingestion error.
func ClassifyIngestError(err error) IngestErrorKind {
	switch {
	case err == nil:
		return IngestErrorUnknown
	case errors.Is(err, ErrMissingAuthToken):
		return IngestErrorMissingAuth
	case errors.Is(err, ErrInvalidAuthToken):
		return IngestErrorInvalidAuth
	case errors.Is(err, ErrInvalidSignature):
		return IngestErrorInvalidSignature
	case errors.Is(err, ErrInvalidPayload):
		return IngestErrorInvalidPayload
	case errors.Is(err, ErrInvalidSchema):
		return IngestErrorInvalidSchema
	case errors.Is(err, ErrUnsupportedType):
		return IngestErrorUnsupportedType
	default:
		return IngestErrorUnknown
	}
}

// Ingest validates command auth/signature/event and appends to event store.
func (s *EventIngestService) Ingest(ctx context.Context, cmd IngestCommand) error {
	token, err := bearerToken(cmd.AuthorizationHeader)
	if err != nil {
		return ErrMissingAuthToken
	}

	store, org, err := s.lookupOrganization(ctx, token)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ErrInvalidAuthToken
		}
		return err
	}
	defer func() {
		_ = store.Close()
	}()

	if !validSignature(cmd.Body, org.WebhookSecret, cmd.SignatureHeader) {
		return ErrInvalidSignature
	}

	event, err := parseIncomingEvent(ctx, cmd.Headers, cmd.Body)
	if err != nil {
		return ErrInvalidPayload
	}
	if err := cdeventsapi.Validate(event); err != nil {
		return ErrInvalidSchema
	}

	eventType := event.GetType().String()
	if _, ok := allowedDeliveryTypes[eventType]; !ok {
		return ErrUnsupportedType
	}

	subjectType := strings.TrimSpace(event.GetType().Subject)
	if subjectType == "" {
		parsedType, parseErr := cdeventsapi.ParseType(eventType)
		if parseErr == nil {
			subjectType = strings.TrimSpace(parsedType.Subject)
		}
	}
	if subjectType == "" {
		subjectType = "unknown"
	}

	raw, err := cdeventsapi.AsJsonString(event)
	if err != nil {
		return err
	}

	var chainID *string
	if eventV04, ok := event.(cdeventsapi.CDEventReaderV04); ok {
		value := strings.TrimSpace(eventV04.GetChainId())
		if value != "" {
			chainID = &value
		}
	}

	var subjectSource *string
	if value := strings.TrimSpace(event.GetSubjectSource()); value != "" {
		subjectSource = &value
	}

	return store.AppendEvent(ctx, ports.EventRecord{
		OrganizationID: org.ID,
		EventID:        event.GetId(),
		EventType:      eventType,
		EventSource:    event.GetSource(),
		EventTimestamp: event.GetTimestamp().UTC().Format(time.RFC3339Nano),
		SubjectID:      event.GetSubjectId(),
		SubjectSource:  subjectSource,
		SubjectType:    subjectType,
		ChainID:        chainID,
		RawEventJSON:   raw,
	})
}

func (s *EventIngestService) lookupOrganization(ctx context.Context, token string) (ports.IngestionStore, ports.Organization, error) {
	store, err := s.storeFactory.Open()
	if err != nil {
		return nil, ports.Organization{}, err
	}

	org, err := store.GetOrganizationByAuthToken(ctx, token)
	if err != nil {
		_ = store.Close()
		return nil, ports.Organization{}, err
	}
	if !org.Enabled {
		_ = store.Close()
		return nil, ports.Organization{}, sql.ErrNoRows
	}

	return store, org, nil
}

func parseIncomingEvent(ctx context.Context, headers http.Header, body []byte) (cdeventsapi.CDEventV04, error) {
	event, err := cdeventsv05.NewFromJsonBytes(body)
	if err == nil {
		return event, nil
	}

	req := &http.Request{
		Method: http.MethodPost,
		Header: headers.Clone(),
		Body:   io.NopCloser(bytes.NewReader(body)),
	}
	message := cehttp.NewMessageFromHttpRequest(req)
	defer func() {
		_ = message.Finish(nil)
	}()

	cloudEvent, err := cebinding.ToEvent(ctx, message)
	if err != nil {
		return nil, err
	}

	raw, err := cloudEventData(cloudEvent)
	if err != nil {
		return nil, err
	}

	return cdeventsv05.NewFromJsonBytes(raw)
}

func cloudEventData(event *ceevent.Event) ([]byte, error) {
	if event == nil {
		return nil, errors.New("cloud event is nil")
	}
	raw := json.RawMessage{}
	if err := event.DataAs(&raw); err != nil {
		return nil, err
	}
	if len(raw) == 0 {
		return nil, errors.New("cloud event data is empty")
	}
	return raw, nil
}

func bearerToken(value string) (string, error) {
	trimmed := strings.TrimSpace(value)
	if !strings.HasPrefix(trimmed, bearerPrefix) {
		return "", errors.New("missing bearer prefix")
	}
	token := strings.TrimSpace(strings.TrimPrefix(trimmed, bearerPrefix))
	if token == "" {
		return "", errors.New("empty token")
	}
	return token, nil
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
