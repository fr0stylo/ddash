package orgconfig

import (
	"context"

	"github.com/fr0stylo/ddash/apps/ddash/internal/app/ports"
	appservices "github.com/fr0stylo/ddash/apps/ddash/internal/app/services"
)

type OrganizationSettings = appservices.OrganizationSettings
type RequiredFieldInput = appservices.RequiredFieldInput
type OrganizationSettingsUpdate = appservices.OrganizationSettingsUpdate

type Service struct {
	delegate *appservices.OrganizationConfigService
}

func NewService(store ports.AppStore) *Service {
	return &Service{delegate: appservices.NewOrganizationConfigService(store)}
}

func (s *Service) GetSettings(ctx context.Context, organizationID int64) (OrganizationSettings, error) {
	return s.delegate.GetSettings(ctx, organizationID)
}

func (s *Service) UpdateSettings(ctx context.Context, organizationID int64, update OrganizationSettingsUpdate) error {
	return s.delegate.UpdateSettings(ctx, organizationID, update)
}
