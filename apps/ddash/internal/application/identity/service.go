package identity

import (
	"context"

	"github.com/fr0stylo/ddash/apps/ddash/internal/app/ports"
	appservices "github.com/fr0stylo/ddash/apps/ddash/internal/app/services"
)

var (
	ErrCannotDeleteLastOrganization   = appservices.ErrCannotDeleteLastOrganization
	ErrOrganizationAccessDenied       = appservices.ErrOrganizationAccessDenied
	ErrOrganizationAdminRequired      = appservices.ErrOrganizationAdminRequired
	ErrOrganizationMembershipRequired = appservices.ErrOrganizationMembershipRequired
	ErrCannotRemoveLastOwner          = appservices.ErrCannotRemoveLastOwner
)

type Service struct {
	delegate *appservices.OrganizationManagementService
}

func NewService(store ports.AppStore) *Service {
	return &Service{delegate: appservices.NewOrganizationManagementService(store)}
}

func (s *Service) EnsureDefaultOrganization(ctx context.Context) (ports.Organization, error) {
	return s.delegate.EnsureDefaultOrganization(ctx)
}

func (s *Service) EnsureDefaultOrganizationForUser(ctx context.Context, userID int64) (ports.Organization, error) {
	return s.delegate.EnsureDefaultOrganizationForUser(ctx, userID)
}

func (s *Service) CreateInitialOrganizationForUser(ctx context.Context, userID int64) (ports.Organization, error) {
	return s.delegate.CreateInitialOrganizationForUser(ctx, userID)
}

func (s *Service) GetOrganizationByID(ctx context.Context, id int64) (ports.Organization, error) {
	return s.delegate.GetOrganizationByID(ctx, id)
}

func (s *Service) GetActiveOrDefaultOrganization(ctx context.Context, activeID int64) (ports.Organization, error) {
	return s.delegate.GetActiveOrDefaultOrganization(ctx, activeID)
}

func (s *Service) GetActiveOrDefaultOrganizationForUser(ctx context.Context, userID, activeID int64) (ports.Organization, error) {
	return s.delegate.GetActiveOrDefaultOrganizationForUser(ctx, userID, activeID)
}

func (s *Service) ListOrganizations(ctx context.Context) ([]ports.Organization, error) {
	return s.delegate.ListOrganizations(ctx)
}

func (s *Service) ListOrganizationsForUser(ctx context.Context, userID int64) ([]ports.Organization, error) {
	return s.delegate.ListOrganizationsForUser(ctx, userID)
}

func (s *Service) CreateOrganization(ctx context.Context, userID int64, name string) (ports.Organization, error) {
	return s.delegate.CreateOrganization(ctx, userID, name)
}

func (s *Service) EnsureUser(ctx context.Context, input ports.UpsertUserInput) (ports.User, error) {
	return s.delegate.EnsureUser(ctx, input)
}

func (s *Service) CanManageOrganization(ctx context.Context, organizationID, userID int64) (bool, error) {
	return s.delegate.CanManageOrganization(ctx, organizationID, userID)
}

func (s *Service) ListMembers(ctx context.Context, organizationID int64) ([]ports.OrganizationMember, error) {
	return s.delegate.ListMembers(ctx, organizationID)
}

func (s *Service) RequestJoinByCode(ctx context.Context, userID int64, joinCode string) error {
	return s.delegate.RequestJoinByCode(ctx, userID, joinCode)
}

func (s *Service) ListPendingJoinRequests(ctx context.Context, organizationID int64) ([]ports.OrganizationJoinRequest, error) {
	return s.delegate.ListPendingJoinRequests(ctx, organizationID)
}

func (s *Service) ApproveJoinRequest(ctx context.Context, organizationID, userID, reviewedBy int64) error {
	return s.delegate.ApproveJoinRequest(ctx, organizationID, userID, reviewedBy)
}

func (s *Service) RejectJoinRequest(ctx context.Context, organizationID, userID, reviewedBy int64) error {
	return s.delegate.RejectJoinRequest(ctx, organizationID, userID, reviewedBy)
}

func (s *Service) AddMemberByLookup(ctx context.Context, organizationID int64, emailOrNick, role string) error {
	return s.delegate.AddMemberByLookup(ctx, organizationID, emailOrNick, role)
}

func (s *Service) UpdateMemberRole(ctx context.Context, organizationID, userID int64, role string) error {
	return s.delegate.UpdateMemberRole(ctx, organizationID, userID, role)
}

func (s *Service) RemoveMember(ctx context.Context, organizationID, userID int64) error {
	return s.delegate.RemoveMember(ctx, organizationID, userID)
}

func (s *Service) RenameOrganization(ctx context.Context, organizationID int64, name string) error {
	return s.delegate.RenameOrganization(ctx, organizationID, name)
}

func (s *Service) SetOrganizationEnabled(ctx context.Context, organizationID int64, enabled bool) error {
	return s.delegate.SetOrganizationEnabled(ctx, organizationID, enabled)
}

func (s *Service) DeleteOrganization(ctx context.Context, organizationID int64) error {
	return s.delegate.DeleteOrganization(ctx, organizationID)
}
