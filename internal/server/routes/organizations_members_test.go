package routes

import (
	"context"
	"database/sql"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/gorilla/sessions"
	"github.com/labstack/echo/v4"
	"github.com/markbates/goth/gothic"

	"github.com/fr0stylo/ddash/internal/app/ports"
	"github.com/fr0stylo/ddash/internal/renderer"
)

type orgRouteStoreFake struct {
	org            ports.Organization
	roleByUserID   map[int64]string
	members        []ports.OrganizationMember
	lookupUser     ports.User
	upsertedUserID int64
	deletedUserID  int64
}

func (f *orgRouteStoreFake) GetDefaultOrganization(context.Context) (ports.Organization, error) {
	return f.org, nil
}

func (f *orgRouteStoreFake) GetOrganizationByID(context.Context, int64) (ports.Organization, error) {
	return f.org, nil
}

func (f *orgRouteStoreFake) ListOrganizations(context.Context) ([]ports.Organization, error) {
	return []ports.Organization{f.org}, nil
}

func (f *orgRouteStoreFake) CreateOrganization(context.Context, ports.CreateOrganizationInput) (ports.Organization, error) {
	return f.org, nil
}
func (f *orgRouteStoreFake) UpdateOrganizationName(context.Context, int64, string) error { return nil }
func (f *orgRouteStoreFake) UpdateOrganizationEnabled(context.Context, int64, bool) error {
	return nil
}
func (f *orgRouteStoreFake) DeleteOrganization(context.Context, int64) error { return nil }
func (f *orgRouteStoreFake) UpsertUser(context.Context, ports.UpsertUserInput) (ports.User, error) {
	return ports.User{}, nil
}

func (f *orgRouteStoreFake) GetUserByID(context.Context, int64) (ports.User, error) {
	return ports.User{}, nil
}

func (f *orgRouteStoreFake) GetUserByEmailOrNickname(context.Context, string, string) (ports.User, error) {
	return f.lookupUser, nil
}

func (f *orgRouteStoreFake) ListOrganizationsByUser(context.Context, int64) ([]ports.Organization, error) {
	return []ports.Organization{f.org}, nil
}

func (f *orgRouteStoreFake) GetOrganizationMemberRole(_ context.Context, _, userID int64) (string, error) {
	role, ok := f.roleByUserID[userID]
	if !ok {
		return "", sql.ErrNoRows
	}
	return role, nil
}

func (f *orgRouteStoreFake) UpsertOrganizationMember(_ context.Context, _, userID int64, _ string) error {
	f.upsertedUserID = userID
	return nil
}

func (f *orgRouteStoreFake) DeleteOrganizationMember(_ context.Context, _, userID int64) error {
	f.deletedUserID = userID
	return nil
}

func (f *orgRouteStoreFake) CountOrganizationOwners(context.Context, int64) (int64, error) {
	return 2, nil
}

func (f *orgRouteStoreFake) ListOrganizationMembers(context.Context, int64) ([]ports.OrganizationMember, error) {
	return f.members, nil
}

func (f *orgRouteStoreFake) ListOrganizationRequiredFields(context.Context, int64) ([]ports.RequiredField, error) {
	return nil, nil
}

func (f *orgRouteStoreFake) ListOrganizationEnvironmentPriorities(context.Context, int64) ([]string, error) {
	return nil, nil
}

func (f *orgRouteStoreFake) ListOrganizationFeatures(context.Context, int64) ([]ports.OrganizationFeature, error) {
	return nil, nil
}

func (f *orgRouteStoreFake) ListOrganizationPreferences(context.Context, int64) ([]ports.OrganizationPreference, error) {
	return nil, nil
}

func (f *orgRouteStoreFake) ListDistinctServiceEnvironmentsFromEvents(context.Context, int64) ([]string, error) {
	return nil, nil
}

func (f *orgRouteStoreFake) UpdateOrganizationSettings(context.Context, int64, ports.OrganizationSettingsUpdate) error {
	return nil
}

func (f *orgRouteStoreFake) ReplaceServiceMetadata(context.Context, int64, string, []ports.MetadataValue) error {
	return nil
}

func initAuthStoreForTests() {
	store := sessions.NewCookieStore([]byte("test-session-secret-32-bytes-long"))
	store.Options = &sessions.Options{Path: "/", MaxAge: 3600, HttpOnly: true, SameSite: http.SameSiteLaxMode}
	gothic.Store = store
}

func newAuthedContext(t *testing.T, e *echo.Echo, method, target string, form url.Values) (echo.Context, *httptest.ResponseRecorder) {
	t.Helper()

	var body strings.Reader
	if form != nil {
		body = *strings.NewReader(form.Encode())
	} else {
		body = *strings.NewReader("")
	}
	req := httptest.NewRequest(method, target, &body)
	if form != nil {
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationForm)
	}
	seedRec := httptest.NewRecorder()
	seedSession, err := gothic.Store.Get(req, authSessionName)
	if err != nil {
		t.Fatalf("session get: %v", err)
	}
	seedSession.Values["user"] = AuthUser{ID: 10, Email: "u@example.com", NickName: "tester"}
	seedSession.Values[authSessionUserIDKey] = int64(10)
	seedSession.Values[authSessionActiveOrgIDKey] = int64(1)
	if err := seedSession.Save(req, seedRec); err != nil {
		t.Fatalf("session save: %v", err)
	}

	req2 := httptest.NewRequest(method, target, strings.NewReader(form.Encode()))
	if form != nil {
		req2.Header.Set(echo.HeaderContentType, echo.MIMEApplicationForm)
	}
	for _, cookie := range seedRec.Result().Cookies() {
		req2.AddCookie(cookie)
	}
	rec := httptest.NewRecorder()
	return e.NewContext(req2, rec), rec
}

func TestHandleOrganizationMembersRequiresAdmin(t *testing.T) {
	initAuthStoreForTests()
	e := echo.New()
	e.Renderer = &renderer.Renderer{}

	store := &orgRouteStoreFake{org: ports.Organization{ID: 1, Name: "org-a", Enabled: true}, roleByUserID: map[int64]string{10: "member"}}
	v := NewViewRoutes(store, nil)

	c, rec := newAuthedContext(t, e, http.MethodGet, "/organizations/members", nil)
	if err := v.handleOrganizationMembers(c); err != nil {
		t.Fatalf("handler error: %v", err)
	}
	if rec.Code != http.StatusFound {
		t.Fatalf("expected redirect, got %d", rec.Code)
	}
}

func TestHandleOrganizationMembersRendersForAdmin(t *testing.T) {
	initAuthStoreForTests()
	e := echo.New()
	e.Renderer = &renderer.Renderer{}

	store := &orgRouteStoreFake{
		org:          ports.Organization{ID: 1, Name: "org-a", Enabled: true},
		roleByUserID: map[int64]string{10: "admin"},
		members:      []ports.OrganizationMember{{UserID: 10, Email: "u@example.com", Nickname: "tester", Name: "Test User", Role: "admin"}},
	}
	v := NewViewRoutes(store, nil)

	c, rec := newAuthedContext(t, e, http.MethodGet, "/organizations/members", nil)
	if err := v.handleOrganizationMembers(c); err != nil {
		t.Fatalf("handler error: %v", err)
	}
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "Test User") {
		t.Fatalf("expected members page body to contain member name, got %q", rec.Body.String())
	}
}

func TestHandleOrganizationMemberAddAddsMembership(t *testing.T) {
	initAuthStoreForTests()
	e := echo.New()
	e.Renderer = &renderer.Renderer{}

	store := &orgRouteStoreFake{org: ports.Organization{ID: 1, Name: "org-a", Enabled: true}, roleByUserID: map[int64]string{10: "owner"}, lookupUser: ports.User{ID: 22}}
	v := NewViewRoutes(store, nil)

	form := url.Values{}
	form.Set("identity", "target@example.com")
	form.Set("role", "member")
	c, rec := newAuthedContext(t, e, http.MethodPost, "/organizations/members/add", form)
	if err := v.handleOrganizationMemberAdd(c); err != nil {
		t.Fatalf("handler error: %v", err)
	}
	if rec.Code != http.StatusFound {
		t.Fatalf("expected redirect, got %d", rec.Code)
	}
	if store.upsertedUserID != 22 {
		t.Fatalf("expected upserted user 22, got %d", store.upsertedUserID)
	}
}

func TestHandleOrganizationMemberRoleUpdatesMembership(t *testing.T) {
	initAuthStoreForTests()
	e := echo.New()
	e.Renderer = &renderer.Renderer{}

	store := &orgRouteStoreFake{
		org:          ports.Organization{ID: 1, Name: "org-a", Enabled: true},
		roleByUserID: map[int64]string{10: "admin", 22: "member"},
	}
	v := NewViewRoutes(store, nil)

	form := url.Values{}
	form.Set("userID", "22")
	form.Set("role", "admin")
	c, rec := newAuthedContext(t, e, http.MethodPost, "/organizations/members/role", form)
	if err := v.handleOrganizationMemberRole(c); err != nil {
		t.Fatalf("handler error: %v", err)
	}
	if rec.Code != http.StatusFound {
		t.Fatalf("expected redirect, got %d", rec.Code)
	}
	if store.upsertedUserID != 22 {
		t.Fatalf("expected updated user 22, got %d", store.upsertedUserID)
	}
}

func TestHandleOrganizationMemberRemoveDeletesMembership(t *testing.T) {
	initAuthStoreForTests()
	e := echo.New()
	e.Renderer = &renderer.Renderer{}

	store := &orgRouteStoreFake{
		org:          ports.Organization{ID: 1, Name: "org-a", Enabled: true},
		roleByUserID: map[int64]string{10: "owner", 22: "member"},
	}
	v := NewViewRoutes(store, nil)

	form := url.Values{}
	form.Set("userID", "22")
	c, rec := newAuthedContext(t, e, http.MethodPost, "/organizations/members/remove", form)
	if err := v.handleOrganizationMemberRemove(c); err != nil {
		t.Fatalf("handler error: %v", err)
	}
	if rec.Code != http.StatusFound {
		t.Fatalf("expected redirect, got %d", rec.Code)
	}
	if store.deletedUserID != 22 {
		t.Fatalf("expected deleted user 22, got %d", store.deletedUserID)
	}
}
