package services

import (
	"context"
	"errors"
	"strings"

	"github.com/fr0stylo/ddash/apps/ddash/internal/app/ports"
)

// ErrRequiredMetadataMissing is returned when strict metadata enforcement is enabled and required fields are missing.
var ErrRequiredMetadataMissing = errors.New("required metadata missing")

// MetadataService handles service metadata write operations.
type MetadataService struct {
	store ports.AppStore
}

// NewMetadataService constructs metadata write service.
func NewMetadataService(store ports.AppStore) *MetadataService {
	return &MetadataService{store: store}
}

// MetadataFieldUpdate represents one metadata field update value.
type MetadataFieldUpdate struct {
	Label string
	Value string
}

// UpdateServiceMetadata updates service metadata values for one service.
func (s *MetadataService) UpdateServiceMetadata(ctx context.Context, organizationID int64, serviceName string, fields []MetadataFieldUpdate, strict bool) error {
	serviceName = strings.TrimSpace(serviceName)
	if serviceName == "" || organizationID <= 0 {
		return nil
	}
	required, err := s.store.ListOrganizationRequiredFields(ctx, organizationID)
	if err != nil {
		return err
	}

	allowed := map[string]string{}
	for _, field := range required {
		label := strings.TrimSpace(field.Label)
		if label == "" {
			continue
		}
		allowed[strings.ToLower(label)] = label
	}

	clean := make([]MetadataFieldUpdate, 0, len(fields))
	seen := map[string]bool{}
	for _, field := range fields {
		label := strings.TrimSpace(field.Label)
		key := strings.ToLower(label)
		if label == "" || seen[key] {
			continue
		}
		canonicalLabel, ok := allowed[key]
		if !ok {
			continue
		}
		seen[key] = true
		clean = append(clean, MetadataFieldUpdate{
			Label: canonicalLabel,
			Value: strings.TrimSpace(field.Value),
		})
	}

	values := make([]ports.MetadataValue, 0, len(clean))
	present := map[string]bool{}
	for _, field := range clean {
		if field.Value == "" {
			continue
		}
		present[strings.ToLower(strings.TrimSpace(field.Label))] = true
		values = append(values, ports.MetadataValue{Label: field.Label, Value: field.Value})
	}

	if strict {
		for _, field := range required {
			label := strings.ToLower(strings.TrimSpace(field.Label))
			if label == "" {
				continue
			}
			if !present[label] {
				return ErrRequiredMetadataMissing
			}
		}
	}

	return s.store.ReplaceServiceMetadata(ctx, organizationID, serviceName, values)
}
