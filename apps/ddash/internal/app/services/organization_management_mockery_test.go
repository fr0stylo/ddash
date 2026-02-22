package services

import (
	"context"
	"testing"

	"github.com/stretchr/testify/mock"

	"github.com/fr0stylo/ddash/apps/ddash/internal/app/ports"
	portmocks "github.com/fr0stylo/ddash/apps/ddash/internal/app/ports/mocks"
)

func TestRequestJoinByCode_UsesStoreFlow(t *testing.T) {
	store := portmocks.NewMockAppStore(t)
	svc := NewOrganizationManagementService(store)

	store.EXPECT().GetOrganizationByJoinCode(mock.Anything, "join-123").Return(ports.Organization{ID: 42, Name: "team-org", Enabled: true}, nil)
	store.EXPECT().UpsertOrganizationJoinRequest(mock.Anything, int64(42), int64(7), mock.Anything).Return(nil)

	err := svc.RequestJoinByCode(context.Background(), 7, "join-123")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
}

func TestApproveJoinRequest_UpdatesMembershipAndStatus(t *testing.T) {
	store := portmocks.NewMockAppStore(t)
	svc := NewOrganizationManagementService(store)

	store.EXPECT().UpsertOrganizationMember(mock.Anything, int64(9), int64(21), memberRoleMember).Return(nil)
	store.EXPECT().SetOrganizationJoinRequestStatus(mock.Anything, int64(9), int64(21), "approved", int64(3)).Return(nil)

	err := svc.ApproveJoinRequest(context.Background(), 9, 21, 3)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
}

func TestGetActiveOrDefaultOrganizationForUser_NoMembership(t *testing.T) {
	store := portmocks.NewMockAppStore(t)
	svc := NewOrganizationManagementService(store)

	store.EXPECT().ListOrganizationsByUser(mock.Anything, int64(88)).Return([]ports.Organization{}, nil)

	_, err := svc.GetActiveOrDefaultOrganizationForUser(context.Background(), 88, 0)
	if err != ErrOrganizationMembershipRequired {
		t.Fatalf("expected ErrOrganizationMembershipRequired, got %v", err)
	}
}
