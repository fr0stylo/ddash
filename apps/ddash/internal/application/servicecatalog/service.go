package servicecatalog

import (
	"context"

	"github.com/fr0stylo/ddash/apps/ddash/internal/app/domain"
	"github.com/fr0stylo/ddash/apps/ddash/internal/app/ports"
	appservices "github.com/fr0stylo/ddash/apps/ddash/internal/app/services"
	domaincatalog "github.com/fr0stylo/ddash/apps/ddash/internal/domains/servicecatalog"
)

type Service struct {
	read *appservices.ServiceReadService
}

func NewService(store ports.ServiceReadStore) *Service {
	return &Service{read: appservices.NewServiceReadServiceFromStore(store)}
}

func (s *Service) GetHomeData(ctx context.Context, organizationID int64) ([]domain.Service, []domain.MetadataFilterOption, error) {
	return s.read.GetHomeData(ctx, organizationID)
}

func (s *Service) GetServicesByEnv(ctx context.Context, organizationID int64, env string) ([]domain.Service, error) {
	return s.read.GetServicesByEnv(ctx, organizationID, env)
}

func (s *Service) GetDeployments(ctx context.Context, organizationID int64, env, service string) ([]domain.DeploymentRow, []domain.MetadataFilterOption, error) {
	return s.read.GetDeployments(ctx, organizationID, env, service)
}

func (s *Service) GetOrganizationRenderVersion(ctx context.Context, organizationID int64) (int64, error) {
	return s.read.GetOrganizationRenderVersion(ctx, organizationID)
}

func (s *Service) GetServiceDetail(ctx context.Context, organizationID int64, name string) (domain.ServiceDetail, error) {
	return s.read.GetServiceDetail(ctx, organizationID, name)
}

func (s *Service) UpsertServiceDependency(ctx context.Context, organizationID int64, serviceName, dependsOn string) error {
	serviceName, dependsOn, ok := domaincatalog.NormalizeDependencyInput(serviceName, dependsOn)
	if !ok {
		return nil
	}
	return s.read.UpsertServiceDependency(ctx, organizationID, serviceName, dependsOn)
}

func (s *Service) DeleteServiceDependency(ctx context.Context, organizationID int64, serviceName, dependsOn string) error {
	serviceName, dependsOn, ok := domaincatalog.NormalizeDependencyInput(serviceName, dependsOn)
	if !ok {
		return nil
	}
	return s.read.DeleteServiceDependency(ctx, organizationID, serviceName, dependsOn)
}
