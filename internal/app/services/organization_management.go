package services

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"github.com/fr0stylo/ddash/internal/app/ports"
)

// ErrCannotDeleteLastOrganization is returned when delete would remove the only remaining organization.
var ErrCannotDeleteLastOrganization = errors.New("cannot delete last organization")

// ErrOrganizationAccessDenied is returned when user is not allowed to access the organization.
var ErrOrganizationAccessDenied = errors.New("organization access denied")

// ErrOrganizationAdminRequired is returned when admin-level organization role is required.
var ErrOrganizationAdminRequired = errors.New("organization admin role required")

// ErrOrganizationMembershipRequired is returned when user has no organization membership yet.
var ErrOrganizationMembershipRequired = errors.New("organization membership required")

// ErrCannotRemoveLastOwner is returned when membership change would remove the last owner.
var ErrCannotRemoveLastOwner = errors.New("cannot remove last owner")

const (
	memberRoleOwner  = "owner"
	memberRoleAdmin  = "admin"
	memberRoleMember = "member"
)

// OrganizationManagementService handles listing, selecting, and creating organizations.
type OrganizationManagementService struct {
	store ports.AppStore
}

// NewOrganizationManagementService constructs organization management service.
func NewOrganizationManagementService(store ports.AppStore) *OrganizationManagementService {
	return &OrganizationManagementService{store: store}
}

// EnsureDefaultOrganization returns the default organization, creating one if needed.
func (s *OrganizationManagementService) EnsureDefaultOrganization(ctx context.Context) (ports.Organization, error) {
	return getOrCreateDefaultOrganization(ctx, s.store)
}

// EnsureDefaultOrganizationForUser returns first enabled user organization or creates default and assigns owner membership.
func (s *OrganizationManagementService) EnsureDefaultOrganizationForUser(ctx context.Context, userID int64) (ports.Organization, error) {
	if userID <= 0 {
		return ports.Organization{}, ErrOrganizationAccessDenied
	}
	orgs, err := s.store.ListOrganizationsByUser(ctx, userID)
	if err != nil {
		return ports.Organization{}, err
	}
	for _, org := range orgs {
		if org.Enabled {
			return org, nil
		}
	}
	return ports.Organization{}, ErrOrganizationMembershipRequired
}

func (s *OrganizationManagementService) createInitialOrganizationForUser(ctx context.Context, userID int64, user ports.User) (ports.Organization, error) {
	baseName := userScopedDefaultOrgName(user, userID)
	for i := 0; i < 20; i++ {
		name := baseName
		if i > 0 {
			name = fmt.Sprintf("%s-%d", baseName, i+1)
		}

		authToken, err := randomHexToken(16)
		if err != nil {
			return ports.Organization{}, err
		}
		secret, err := randomHexToken(24)
		if err != nil {
			return ports.Organization{}, err
		}
		joinCode, err := randomHexToken(6)
		if err != nil {
			return ports.Organization{}, err
		}

		org, err := s.store.CreateOrganization(ctx, ports.CreateOrganizationInput{
			Name:          name,
			AuthToken:     authToken,
			JoinCode:      joinCode,
			WebhookSecret: secret,
			Enabled:       true,
		})
		if err != nil {
			if isOrganizationNameConflict(err) {
				continue
			}
			return ports.Organization{}, err
		}
		if err := s.store.UpsertOrganizationMember(ctx, org.ID, userID, memberRoleOwner); err != nil {
			return ports.Organization{}, err
		}
		return org, nil
	}
	return ports.Organization{}, errors.New("failed to create unique organization for user")
}

// CreateInitialOrganizationForUser creates a first organization for a user with no memberships.
func (s *OrganizationManagementService) CreateInitialOrganizationForUser(ctx context.Context, userID int64) (ports.Organization, error) {
	if userID <= 0 {
		return ports.Organization{}, ErrOrganizationAccessDenied
	}
	user, err := s.store.GetUserByID(ctx, userID)
	if err != nil {
		return ports.Organization{}, err
	}
	return s.createInitialOrganizationForUser(ctx, userID, user)
}

// GetOrganizationByID returns one organization.
func (s *OrganizationManagementService) GetOrganizationByID(ctx context.Context, id int64) (ports.Organization, error) {
	return s.store.GetOrganizationByID(ctx, id)
}

// GetActiveOrDefaultOrganization resolves active organization if valid and enabled, otherwise falls back.
func (s *OrganizationManagementService) GetActiveOrDefaultOrganization(ctx context.Context, activeID int64) (ports.Organization, error) {
	if activeID > 0 {
		org, err := s.store.GetOrganizationByID(ctx, activeID)
		if err == nil && org.Enabled {
			return org, nil
		}
	}

	rows, err := s.store.ListOrganizations(ctx)
	if err != nil {
		return ports.Organization{}, err
	}
	for _, org := range rows {
		if org.Enabled {
			return org, nil
		}
	}

	org, err := getOrCreateDefaultOrganization(ctx, s.store)
	if err != nil {
		return ports.Organization{}, err
	}
	if !org.Enabled {
		if err := s.store.UpdateOrganizationEnabled(ctx, org.ID, true); err != nil {
			return ports.Organization{}, err
		}
		return s.store.GetOrganizationByID(ctx, org.ID)
	}
	return org, nil
}

// GetActiveOrDefaultOrganizationForUser resolves active org for a user with membership checks.
func (s *OrganizationManagementService) GetActiveOrDefaultOrganizationForUser(ctx context.Context, userID, activeID int64) (ports.Organization, error) {
	if userID <= 0 {
		return ports.Organization{}, ErrOrganizationAccessDenied
	}
	if activeID > 0 {
		org, err := s.store.GetOrganizationByID(ctx, activeID)
		if err == nil && org.Enabled {
			if _, roleErr := s.store.GetOrganizationMemberRole(ctx, activeID, userID); roleErr == nil {
				return org, nil
			}
		}
	}
	orgs, err := s.store.ListOrganizationsByUser(ctx, userID)
	if err != nil {
		return ports.Organization{}, err
	}
	for _, org := range orgs {
		if org.Enabled {
			return org, nil
		}
	}
	return ports.Organization{}, ErrOrganizationMembershipRequired
}

// ListOrganizations returns all organizations.
func (s *OrganizationManagementService) ListOrganizations(ctx context.Context) ([]ports.Organization, error) {
	return s.store.ListOrganizations(ctx)
}

// ListOrganizationsForUser returns organizations where user is a member.
func (s *OrganizationManagementService) ListOrganizationsForUser(ctx context.Context, userID int64) ([]ports.Organization, error) {
	if userID <= 0 {
		return nil, ErrOrganizationAccessDenied
	}
	return s.store.ListOrganizationsByUser(ctx, userID)
}

// CreateOrganization creates a new enabled organization with generated auth credentials.
func (s *OrganizationManagementService) CreateOrganization(ctx context.Context, userID int64, name string) (ports.Organization, error) {
	name = strings.TrimSpace(name)
	if userID <= 0 || name == "" {
		return ports.Organization{}, ErrOrganizationAccessDenied
	}
	authToken, err := randomHexToken(16)
	if err != nil {
		return ports.Organization{}, err
	}
	secret, err := randomHexToken(24)
	if err != nil {
		return ports.Organization{}, err
	}
	joinCode, err := randomHexToken(6)
	if err != nil {
		return ports.Organization{}, err
	}
	org, err := s.store.CreateOrganization(ctx, ports.CreateOrganizationInput{
		Name:          name,
		AuthToken:     authToken,
		JoinCode:      joinCode,
		WebhookSecret: secret,
		Enabled:       true,
	})
	if err != nil {
		return ports.Organization{}, err
	}
	if err := s.store.UpsertOrganizationMember(ctx, org.ID, userID, memberRoleOwner); err != nil {
		return ports.Organization{}, err
	}
	return org, nil
}

// EnsureUser upserts local user profile.
func (s *OrganizationManagementService) EnsureUser(ctx context.Context, input ports.UpsertUserInput) (ports.User, error) {
	return s.store.UpsertUser(ctx, input)
}

// CanManageOrganization returns true for owner/admin roles.
func (s *OrganizationManagementService) CanManageOrganization(ctx context.Context, organizationID, userID int64) (bool, error) {
	role, err := s.store.GetOrganizationMemberRole(ctx, organizationID, userID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return false, nil
		}
		return false, err
	}
	role = strings.ToLower(strings.TrimSpace(role))
	return role == memberRoleOwner || role == memberRoleAdmin, nil
}

// ListMembers returns organization members.
func (s *OrganizationManagementService) ListMembers(ctx context.Context, organizationID int64) ([]ports.OrganizationMember, error) {
	return s.store.ListOrganizationMembers(ctx, organizationID)
}

// RequestJoinByCode submits a pending join request using organization join code.
func (s *OrganizationManagementService) RequestJoinByCode(ctx context.Context, userID int64, joinCode string) error {
	if userID <= 0 {
		return ErrOrganizationAccessDenied
	}
	joinCode = strings.TrimSpace(joinCode)
	if joinCode == "" {
		return ErrOrganizationAccessDenied
	}
	org, err := s.store.GetOrganizationByJoinCode(ctx, joinCode)
	if err != nil {
		return err
	}
	requestCode, err := randomHexToken(5)
	if err != nil {
		return err
	}
	return s.store.UpsertOrganizationJoinRequest(ctx, org.ID, userID, requestCode)
}

// ListPendingJoinRequests returns pending join requests for one organization.
func (s *OrganizationManagementService) ListPendingJoinRequests(ctx context.Context, organizationID int64) ([]ports.OrganizationJoinRequest, error) {
	if organizationID <= 0 {
		return nil, nil
	}
	return s.store.ListPendingOrganizationJoinRequests(ctx, organizationID)
}

// ApproveJoinRequest approves a join request and grants member role.
func (s *OrganizationManagementService) ApproveJoinRequest(ctx context.Context, organizationID, userID, reviewedBy int64) error {
	if organizationID <= 0 || userID <= 0 || reviewedBy <= 0 {
		return ErrOrganizationAccessDenied
	}
	if err := s.store.UpsertOrganizationMember(ctx, organizationID, userID, memberRoleMember); err != nil {
		return err
	}
	return s.store.SetOrganizationJoinRequestStatus(ctx, organizationID, userID, "approved", reviewedBy)
}

// RejectJoinRequest rejects a pending join request.
func (s *OrganizationManagementService) RejectJoinRequest(ctx context.Context, organizationID, userID, reviewedBy int64) error {
	if organizationID <= 0 || userID <= 0 || reviewedBy <= 0 {
		return ErrOrganizationAccessDenied
	}
	return s.store.SetOrganizationJoinRequestStatus(ctx, organizationID, userID, "rejected", reviewedBy)
}

// AddMemberByLookup adds membership by existing user identity.
func (s *OrganizationManagementService) AddMemberByLookup(ctx context.Context, organizationID int64, emailOrNick, role string) error {
	if organizationID <= 0 {
		return ErrOrganizationAccessDenied
	}
	role = normalizeRole(role)
	if role == "" {
		return ErrOrganizationAdminRequired
	}
	lookup := strings.TrimSpace(emailOrNick)
	if lookup == "" {
		return nil
	}
	user, err := s.store.GetUserByEmailOrNickname(ctx, lookup, lookup)
	if err != nil {
		return err
	}
	return s.store.UpsertOrganizationMember(ctx, organizationID, user.ID, role)
}

// UpdateMemberRole updates one member role.
func (s *OrganizationManagementService) UpdateMemberRole(ctx context.Context, organizationID, userID int64, role string) error {
	role = normalizeRole(role)
	if role == "" {
		return ErrOrganizationAdminRequired
	}
	if role != memberRoleOwner {
		currentRole, err := s.store.GetOrganizationMemberRole(ctx, organizationID, userID)
		if err == nil && strings.EqualFold(strings.TrimSpace(currentRole), memberRoleOwner) {
			count, countErr := s.store.CountOrganizationOwners(ctx, organizationID)
			if countErr != nil {
				return countErr
			}
			if count <= 1 {
				return ErrCannotRemoveLastOwner
			}
		}
	}
	return s.store.UpsertOrganizationMember(ctx, organizationID, userID, role)
}

// RemoveMember removes a member with last-owner protection.
func (s *OrganizationManagementService) RemoveMember(ctx context.Context, organizationID, userID int64) error {
	role, err := s.store.GetOrganizationMemberRole(ctx, organizationID, userID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil
		}
		return err
	}
	if strings.EqualFold(strings.TrimSpace(role), memberRoleOwner) {
		count, countErr := s.store.CountOrganizationOwners(ctx, organizationID)
		if countErr != nil {
			return countErr
		}
		if count <= 1 {
			return ErrCannotRemoveLastOwner
		}
	}
	return s.store.DeleteOrganizationMember(ctx, organizationID, userID)
}

// RenameOrganization updates one organization name.
func (s *OrganizationManagementService) RenameOrganization(ctx context.Context, organizationID int64, name string) error {
	name = strings.TrimSpace(name)
	if organizationID <= 0 || name == "" {
		return nil
	}
	return s.store.UpdateOrganizationName(ctx, organizationID, name)
}

// SetOrganizationEnabled updates one organization enabled state.
func (s *OrganizationManagementService) SetOrganizationEnabled(ctx context.Context, organizationID int64, enabled bool) error {
	if organizationID <= 0 {
		return nil
	}
	if !enabled {
		rows, err := s.store.ListOrganizations(ctx)
		if err != nil {
			return err
		}
		enabledCount := 0
		for _, row := range rows {
			if row.Enabled {
				enabledCount++
			}
		}
		org, err := s.store.GetOrganizationByID(ctx, organizationID)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return nil
			}
			return err
		}
		if org.Enabled && enabledCount <= 1 {
			return nil
		}
	}
	return s.store.UpdateOrganizationEnabled(ctx, organizationID, enabled)
}

// DeleteOrganization removes one organization when safe.
func (s *OrganizationManagementService) DeleteOrganization(ctx context.Context, organizationID int64) error {
	if organizationID <= 0 {
		return nil
	}
	rows, err := s.store.ListOrganizations(ctx)
	if err != nil {
		return err
	}
	if len(rows) <= 1 {
		return ErrCannotDeleteLastOrganization
	}
	return s.store.DeleteOrganization(ctx, organizationID)
}

func normalizeRole(role string) string {
	role = strings.ToLower(strings.TrimSpace(role))
	switch role {
	case memberRoleOwner, memberRoleAdmin, memberRoleMember:
		return role
	default:
		return ""
	}
}

func userScopedDefaultOrgName(user ports.User, userID int64) string {
	base := strings.TrimSpace(user.Nickname)
	if base == "" {
		emailParts := strings.Split(strings.TrimSpace(user.Email), "@")
		if len(emailParts) > 0 {
			base = emailParts[0]
		}
	}
	if base == "" {
		base = strings.TrimSpace(user.Name)
	}
	if base == "" {
		base = fmt.Sprintf("user-%d", userID)
	}

	builder := strings.Builder{}
	prevDash := false
	for _, r := range strings.ToLower(base) {
		isLetter := r >= 'a' && r <= 'z'
		isDigit := r >= '0' && r <= '9'
		if isLetter || isDigit {
			builder.WriteRune(r)
			prevDash = false
			continue
		}
		if !prevDash {
			builder.WriteRune('-')
			prevDash = true
		}
	}
	clean := strings.Trim(builder.String(), "-")
	if clean == "" {
		clean = fmt.Sprintf("user-%d", userID)
	}
	return clean + "-org"
}

func isOrganizationNameConflict(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "unique") && strings.Contains(msg, "organizations.name")
}
