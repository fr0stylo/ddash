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
	"log/slog"
	"net/http"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	cdeventsapi "github.com/cdevents/sdk-go/pkg/api"
	cdeventsv05 "github.com/cdevents/sdk-go/pkg/api/v05"
	cebinding "github.com/cloudevents/sdk-go/v2/binding"
	ceevent "github.com/cloudevents/sdk-go/v2/event"
	cehttp "github.com/cloudevents/sdk-go/v2/protocol/http"

	"github.com/fr0stylo/ddash/apps/ddash/internal/app/ports"
	"github.com/fr0stylo/ddash/internal/observability"
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
	// ErrIngestBusy indicates ingestion queue saturation.
	ErrIngestBusy = errors.New("ingestion busy")
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

var allowedTypePrefixes = []string{
	"dev.cdevents.pipeline.",
	"dev.cdevents.change.",
	"dev.cdevents.artifact.",
	"dev.cdevents.incident.",
}

// EventIngestService validates and appends CDEvents to the event store.
type EventIngestService struct {
	storeFactory ports.IngestionStoreFactory
	batcher      *ingestBatcher
}

type IngestBatchConfig struct {
	Enabled       bool
	Size          int
	FlushInterval time.Duration
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
	// IngestErrorBusy indicates ingestion queue saturation.
	IngestErrorBusy IngestErrorKind = "busy"
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
	return NewEventIngestServiceWithConfig(storeFactory, IngestBatchConfig{
		Enabled:       true,
		Size:          100,
		FlushInterval: 50 * time.Millisecond,
	})
}

func NewEventIngestServiceWithConfig(storeFactory ports.IngestionStoreFactory, batchCfg IngestBatchConfig) *EventIngestService {
	service := &EventIngestService{storeFactory: storeFactory}
	if batchCfg.Enabled {
		size := batchCfg.Size
		if size <= 0 {
			size = 100
		}
		if size > 2000 {
			size = 2000
		}
		interval := batchCfg.FlushInterval
		if interval <= 0 {
			interval = 50 * time.Millisecond
		}
		service.batcher = newIngestBatcher(storeFactory, size, interval)
	}
	return service
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
	case errors.Is(err, ErrIngestBusy):
		return IngestErrorBusy
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

	org, err := s.lookupOrganization(ctx, token)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ErrInvalidAuthToken
		}
		return err
	}
	ctx = observability.WithRequestIdentity(ctx, 0, org.ID)

	if !validSignature(cmd.Body, org.WebhookSecret, cmd.SignatureHeader) {
		return ErrInvalidSignature
	}

	event, err := parseIncomingEvent(ctx, cmd.Headers, cmd.Body)
	if err != nil {
		return ErrInvalidPayload
	}
	eventType := event.GetType().String()
	if !isSupportedEventType(eventType) {
		return ErrUnsupportedType
	}
	if requiresStrictSchemaValidation(eventType) {
		if err := cdeventsapi.Validate(event); err != nil {
			return ErrInvalidSchema
		}
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

	record := ports.EventRecord{
		OrganizationID: org.ID,
		EventID:        event.GetId(),
		EventType:      eventType,
		EventSource:    event.GetSource(),
		EventTimestamp: event.GetTimestamp().UTC().Format(time.RFC3339Nano),
		EventTSMs:      event.GetTimestamp().UTC().UnixMilli(),
		SubjectID:      event.GetSubjectId(),
		SubjectSource:  subjectSource,
		SubjectType:    subjectType,
		ChainID:        chainID,
		RawEventJSON:   raw,
	}

	return s.appendRecord(ctx, record)
}

// IngestForOrganization ingests an event for a pre-resolved organization.
func (s *EventIngestService) IngestForOrganization(ctx context.Context, organizationID int64, headers http.Header, body []byte) error {
	if organizationID <= 0 {
		return ErrInvalidAuthToken
	}
	ctx = observability.WithRequestIdentity(ctx, 0, organizationID)

	event, err := parseIncomingEvent(ctx, headers, body)
	if err != nil {
		return ErrInvalidPayload
	}
	eventType := event.GetType().String()
	if !isSupportedEventType(eventType) {
		return ErrUnsupportedType
	}
	if requiresStrictSchemaValidation(eventType) {
		if err := cdeventsapi.Validate(event); err != nil {
			return ErrInvalidSchema
		}
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

	record := ports.EventRecord{
		OrganizationID: organizationID,
		EventID:        event.GetId(),
		EventType:      eventType,
		EventSource:    event.GetSource(),
		EventTimestamp: event.GetTimestamp().UTC().Format(time.RFC3339Nano),
		EventTSMs:      event.GetTimestamp().UTC().UnixMilli(),
		SubjectID:      event.GetSubjectId(),
		SubjectSource:  subjectSource,
		SubjectType:    subjectType,
		ChainID:        chainID,
		RawEventJSON:   raw,
	}

	return s.appendRecord(ctx, record)
}

func (s *EventIngestService) appendRecord(ctx context.Context, record ports.EventRecord) error {

	if s.batcher != nil {
		return s.batcher.append(ctx, record)
	}

	store, err := s.storeFactory.Open()
	if err != nil {
		return err
	}
	defer func() {
		_ = store.Close()
	}()
	return store.AppendEvent(ctx, record)
}

func isSupportedEventType(eventType string) bool {
	eventType = strings.TrimSpace(eventType)
	if eventType == "" {
		return false
	}
	if _, ok := allowedDeliveryTypes[eventType]; ok {
		return true
	}
	for _, prefix := range allowedTypePrefixes {
		if strings.HasPrefix(eventType, prefix) {
			return true
		}
	}
	return false
}

func requiresStrictSchemaValidation(eventType string) bool {
	_, ok := allowedDeliveryTypes[strings.TrimSpace(eventType)]
	return ok
}

func (s *EventIngestService) lookupOrganization(ctx context.Context, token string) (ports.Organization, error) {
	store, err := s.storeFactory.Open()
	if err != nil {
		return ports.Organization{}, err
	}
	defer func() {
		_ = store.Close()
	}()

	org, err := store.GetOrganizationByAuthToken(ctx, token)
	if err != nil {
		return ports.Organization{}, err
	}
	if !org.Enabled {
		return ports.Organization{}, sql.ErrNoRows
	}

	return org, nil
}

type ingestBatcher struct {
	storeFactory  ports.IngestionStoreFactory
	batchSize     int
	flushInterval time.Duration
	queue         chan ingestBatchRequest
	startOnce     sync.Once
	accepted      atomic.Int64
	flushBatches  atomic.Int64
	flushEvents   atomic.Int64
	flushErrors   atomic.Int64
}

type ingestBatchRequest struct {
	ctx    context.Context
	event  ports.EventRecord
	result chan error
}

func newIngestBatcher(storeFactory ports.IngestionStoreFactory, batchSize int, flushInterval time.Duration) *ingestBatcher {
	b := &ingestBatcher{
		storeFactory:  storeFactory,
		batchSize:     batchSize,
		flushInterval: flushInterval,
		queue:         make(chan ingestBatchRequest, batchSize*8),
	}
	b.startOnce.Do(func() {
		go b.run()
	})
	return b
}

func (b *ingestBatcher) append(ctx context.Context, event ports.EventRecord) error {
	result := make(chan error, 1)
	request := ingestBatchRequest{ctx: ctx, event: event, result: result}

	select {
	case b.queue <- request:
		b.accepted.Add(1)
	default:
		return ErrIngestBusy
	case <-ctx.Done():
		return ctx.Err()
	}

	select {
	case err := <-result:
		return err
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (b *ingestBatcher) run() {
	ticker := time.NewTicker(b.flushInterval)
	defer ticker.Stop()
	statsTicker := time.NewTicker(30 * time.Second)
	defer statsTicker.Stop()

	batch := make([]ingestBatchRequest, 0, b.batchSize)
	for {
		select {
		case request := <-b.queue:
			batch = append(batch, request)
			if len(batch) >= b.batchSize {
				b.flush(batch)
				batch = batch[:0]
			}
		case <-ticker.C:
			if len(batch) > 0 {
				b.flush(batch)
				batch = batch[:0]
			}
		case <-statsTicker.C:
			slog.Info("ingest_batcher_stats",
				"queue_len", len(b.queue),
				"queue_cap", cap(b.queue),
				"accepted", b.accepted.Load(),
				"flush_batches", b.flushBatches.Load(),
				"flush_events", b.flushEvents.Load(),
				"flush_errors", b.flushErrors.Load(),
			)
		}
	}
}

func (b *ingestBatcher) flush(batch []ingestBatchRequest) {
	store, err := b.storeFactory.Open()
	if err != nil {
		b.flushErrors.Add(1)
		slog.Error("ingest_batch_open_failed", "error", err, "batch_size", len(batch))
		for _, item := range batch {
			item.result <- err
		}
		return
	}
	defer func() {
		_ = store.Close()
	}()

	events := make([]ports.EventRecord, 0, len(batch))
	for _, item := range batch {
		events = append(events, item.event)
	}

	err = store.AppendEvents(context.Background(), events)
	if err != nil {
		b.flushErrors.Add(1)
		slog.Error("ingest_batch_flush_failed", "error", err, "batch_size", len(events))
	} else {
		b.flushBatches.Add(1)
		b.flushEvents.Add(int64(len(events)))
	}
	for _, item := range batch {
		item.result <- err
	}
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
