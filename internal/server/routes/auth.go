package routes

import (
	"encoding/gob"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gorilla/sessions"
	"github.com/labstack/echo/v4"
	"github.com/markbates/goth"
	"github.com/markbates/goth/gothic"
	"github.com/markbates/goth/providers/github"

	"github.com/fr0stylo/ddash/internal/app/ports"
	appservices "github.com/fr0stylo/ddash/internal/app/services"
	"github.com/fr0stylo/ddash/internal/observability"
	"github.com/fr0stylo/ddash/views/pages"
)

const (
	authSessionName           = "ddash-auth"
	authSessionActiveOrgIDKey = "activeOrgID"
	authSessionUserIDKey      = "userID"
	githubProvider            = "github"
	gothSessionName           = "_gothic_session"
)

// AuthConfig configures session and GitHub OAuth authentication.
type AuthConfig struct {
	SessionKey         string
	GitHubClientID     string
	GitHubClientSecret string
	GitHubCallbackURL  string
	SecureCookies      bool
}

// AuthUser is the authenticated user stored in session and context.
type AuthUser struct {
	ID        int64
	Name      string
	NickName  string
	Email     string
	AvatarURL string
}

func init() {
	gob.Register(AuthUser{})
}

// ConfigureAuth initializes session store and GitHub OAuth provider.
func ConfigureAuth(config AuthConfig) {
	store := sessions.NewCookieStore([]byte(config.SessionKey))
	store.Options = &sessions.Options{
		Path:     "/",
		MaxAge:   int((7 * 24 * time.Hour).Seconds()),
		HttpOnly: true,
		Secure:   config.SecureCookies,
		SameSite: http.SameSiteLaxMode,
	}
	gothic.Store = store

	goth.UseProviders(
		github.New(
			config.GitHubClientID,
			config.GitHubClientSecret,
			config.GitHubCallbackURL,
			"read:user",
			"user:email",
		),
	)
}

// AuthRoutes registers authentication endpoints.
type AuthRoutes struct {
	store          ports.AppStore
	enableDevLogin bool
}

// NewAuthRoutes constructs auth routes.
func NewAuthRoutes(store ports.AppStore, enableDevLogin bool) *AuthRoutes {
	return &AuthRoutes{store: store, enableDevLogin: enableDevLogin}
}

// RegisterRoutes registers authentication routes on the server.
func (a *AuthRoutes) RegisterRoutes(s *echo.Echo) {
	s.GET("/login", a.handleLogin)
	s.GET("/logout", a.handleLogout)
	s.GET("/auth/:provider", a.handleAuthBegin)
	s.GET("/auth/:provider/callback", a.handleAuthCallback)
	if a.enableDevLogin {
		s.GET("/auth/dev/login", a.handleDevLogin)
		s.POST("/auth/dev/login", a.handleDevLogin)
	}
}

// RequireAuth ensures a request has an authenticated user session.
func RequireAuth(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		user, ok := authUserFromSession(c)
		if !ok {
			return c.Redirect(http.StatusFound, "/login")
		}
		orgID, _ := GetActiveOrganizationID(c)
		ctx := observability.WithRequestIdentity(c.Request().Context(), user.ID, orgID)
		c.SetRequest(c.Request().WithContext(ctx))
		c.Set("authUser", user)
		return next(c)
	}
}

func (a *AuthRoutes) handleLogin(c echo.Context) error {
	if _, ok := authUserFromSession(c); ok {
		return c.Redirect(http.StatusFound, "/")
	}
	return c.Render(http.StatusOK, "", pages.LoginPage())
}

func (a *AuthRoutes) handleLogout(c echo.Context) error {
	session, err := gothic.Store.Get(c.Request(), authSessionName)
	if err != nil {
		if isInvalidSecureCookieError(err) {
			clearSessionCookie(c, authSessionName)
			clearSessionCookie(c, gothSessionName)
			return c.Redirect(http.StatusFound, "/login")
		}
		return err
	}
	delete(session.Values, authSessionActiveOrgIDKey)
	delete(session.Values, authSessionUserIDKey)
	session.Options.MaxAge = -1
	if err := session.Save(c.Request(), c.Response()); err != nil {
		return err
	}
	return c.Redirect(http.StatusFound, "/login")
}

func (a *AuthRoutes) handleAuthBegin(c echo.Context) error {
	provider := c.Param("provider")
	if provider != githubProvider {
		return c.NoContent(http.StatusNotFound)
	}
	request := addProviderParam(c.Request(), provider)
	gothic.BeginAuthHandler(c.Response(), request)
	return nil
}

func (a *AuthRoutes) handleAuthCallback(c echo.Context) error {
	provider := c.Param("provider")
	if provider != githubProvider {
		return c.NoContent(http.StatusNotFound)
	}
	request := addProviderParam(c.Request(), provider)
	user, err := gothic.CompleteUserAuth(c.Response(), request)
	if err != nil {
		if isInvalidSecureCookieError(err) {
			clearSessionCookie(c, authSessionName)
			clearSessionCookie(c, gothSessionName)
			return c.Redirect(http.StatusFound, "/login")
		}
		return err
	}

	email := strings.TrimSpace(user.Email)
	if email == "" {
		nick := strings.TrimSpace(user.NickName)
		if nick == "" {
			nick = "user"
		}
		email = nick + "@local.invalid"
	}
	nickname := strings.TrimSpace(user.NickName)
	if nickname == "" {
		nickname = strings.Split(email, "@")[0]
	}

	localUser, err := a.store.UpsertUser(request.Context(), ports.UpsertUserInput{
		GitHubID:  user.UserID,
		Email:     email,
		Nickname:  nickname,
		Name:      user.Name,
		AvatarURL: user.AvatarURL,
	})
	if err != nil {
		return err
	}

	session, err := gothic.Store.Get(request, authSessionName)
	if err != nil {
		if isInvalidSecureCookieError(err) {
			clearSessionCookie(c, authSessionName)
			clearSessionCookie(c, gothSessionName)
			return c.Redirect(http.StatusFound, "/login")
		}
		return err
	}
	session.Values["user"] = AuthUser{
		ID:        localUser.ID,
		Name:      firstNonEmpty(user.Name, nickname),
		NickName:  nickname,
		Email:     email,
		AvatarURL: user.AvatarURL,
	}
	session.Values[authSessionUserIDKey] = localUser.ID
	orgService := appservices.NewOrganizationManagementService(a.store)
	org, orgErr := orgService.GetActiveOrDefaultOrganizationForUser(request.Context(), localUser.ID, 0)
	if orgErr == nil {
		session.Values[authSessionActiveOrgIDKey] = org.ID
	} else {
		delete(session.Values, authSessionActiveOrgIDKey)
	}
	if err := session.Save(request, c.Response()); err != nil {
		return err
	}
	if errors.Is(orgErr, appservices.ErrOrganizationMembershipRequired) {
		return c.Redirect(http.StatusFound, "/welcome")
	}
	if orgErr != nil {
		return orgErr
	}
	return c.Redirect(http.StatusFound, "/")
}

func (a *AuthRoutes) handleDevLogin(c echo.Context) error {
	if !a.enableDevLogin {
		return c.NoContent(http.StatusNotFound)
	}

	email := strings.TrimSpace(c.FormValue("email"))
	if email == "" {
		email = "dev-user@example.local"
	}
	nickname := strings.TrimSpace(c.FormValue("nickname"))
	if nickname == "" {
		nickname = strings.Split(email, "@")[0]
	}
	name := strings.TrimSpace(c.FormValue("name"))
	if name == "" {
		name = nickname
	}
	avatarURL := strings.TrimSpace(c.FormValue("avatar_url"))
	githubID := strings.TrimSpace(c.FormValue("github_id"))
	if githubID == "" {
		githubID = "dev:" + nickname
	}

	localUser, err := a.store.UpsertUser(c.Request().Context(), ports.UpsertUserInput{
		GitHubID:  githubID,
		Email:     email,
		Nickname:  nickname,
		Name:      name,
		AvatarURL: avatarURL,
	})
	if err != nil {
		return err
	}

	orgService := appservices.NewOrganizationManagementService(a.store)
	org, orgErr := orgService.GetActiveOrDefaultOrganizationForUser(c.Request().Context(), localUser.ID, 0)

	session, err := gothic.Store.Get(c.Request(), authSessionName)
	if err != nil {
		if isInvalidSecureCookieError(err) {
			clearSessionCookie(c, authSessionName)
			clearSessionCookie(c, gothSessionName)
			return c.Redirect(http.StatusFound, "/login")
		}
		return err
	}
	session.Values["user"] = AuthUser{
		ID:        localUser.ID,
		Name:      firstNonEmpty(localUser.Name, localUser.Nickname),
		NickName:  localUser.Nickname,
		Email:     localUser.Email,
		AvatarURL: localUser.AvatarURL,
	}
	session.Values[authSessionUserIDKey] = localUser.ID
	if orgErr == nil {
		session.Values[authSessionActiveOrgIDKey] = org.ID
	} else {
		delete(session.Values, authSessionActiveOrgIDKey)
	}
	if err := session.Save(c.Request(), c.Response()); err != nil {
		return err
	}

	next := strings.TrimSpace(c.FormValue("next"))
	if next == "" || !strings.HasPrefix(next, "/") {
		next = "/"
	}
	if errors.Is(orgErr, appservices.ErrOrganizationMembershipRequired) {
		return c.Redirect(http.StatusFound, "/welcome")
	}
	if orgErr != nil {
		return orgErr
	}
	return c.Redirect(http.StatusFound, next)
}

// GetAuthUserID returns authenticated local user id.
func GetAuthUserID(c echo.Context) (int64, bool) {
	if user, ok := GetAuthUser(c); ok && user.ID > 0 {
		return user.ID, true
	}
	session, err := gothic.Store.Get(c.Request(), authSessionName)
	if err != nil {
		if isInvalidSecureCookieError(err) {
			clearSessionCookie(c, authSessionName)
			clearSessionCookie(c, gothSessionName)
		}
		return 0, false
	}
	value, ok := session.Values[authSessionUserIDKey]
	if !ok || value == nil {
		return 0, false
	}
	switch v := value.(type) {
	case int64:
		return v, true
	case int:
		return int64(v), true
	case int32:
		return int64(v), true
	case float64:
		return int64(v), true
	default:
		return 0, false
	}
}

// GetAuthUser returns the authenticated user from request context.
func GetAuthUser(c echo.Context) (AuthUser, bool) {
	value := c.Get("authUser")
	if value == nil {
		return AuthUser{}, false
	}
	user, ok := value.(AuthUser)
	if !ok {
		return AuthUser{}, false
	}
	return user, true
}

// GetActiveOrganizationID returns selected organization id from session.
func GetActiveOrganizationID(c echo.Context) (int64, bool) {
	session, err := gothic.Store.Get(c.Request(), authSessionName)
	if err != nil {
		if isInvalidSecureCookieError(err) {
			clearSessionCookie(c, authSessionName)
			clearSessionCookie(c, gothSessionName)
		}
		return 0, false
	}
	value, ok := session.Values[authSessionActiveOrgIDKey]
	if !ok || value == nil {
		return 0, false
	}

	switch v := value.(type) {
	case int64:
		return v, true
	case int:
		return int64(v), true
	case int32:
		return int64(v), true
	case float64:
		return int64(v), true
	default:
		return 0, false
	}
}

// SetActiveOrganizationID stores selected organization id in session.
func SetActiveOrganizationID(c echo.Context, organizationID int64) error {
	session, err := gothic.Store.Get(c.Request(), authSessionName)
	if err != nil {
		if isInvalidSecureCookieError(err) {
			clearSessionCookie(c, authSessionName)
			clearSessionCookie(c, gothSessionName)
			return nil
		}
		return err
	}
	session.Values[authSessionActiveOrgIDKey] = organizationID
	if err := session.Save(c.Request(), c.Response()); err != nil {
		return fmt.Errorf("save auth session: %w", err)
	}
	return nil
}

func addProviderParam(request *http.Request, provider string) *http.Request {
	query := request.URL.Query()
	query.Set("provider", provider)
	request.URL.RawQuery = query.Encode()
	return request
}

func authUserFromSession(c echo.Context) (AuthUser, bool) {
	session, err := gothic.Store.Get(c.Request(), authSessionName)
	if err != nil {
		if isInvalidSecureCookieError(err) {
			clearSessionCookie(c, authSessionName)
			clearSessionCookie(c, gothSessionName)
		}
		return AuthUser{}, false
	}
	value, ok := session.Values["user"]
	if !ok {
		return AuthUser{}, false
	}
	user, ok := value.(AuthUser)
	if !ok {
		return AuthUser{}, false
	}
	return user, true
}

func isInvalidSecureCookieError(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "securecookie") && strings.Contains(msg, "not valid")
}

func clearSessionCookie(c echo.Context, name string) {
	http.SetCookie(c.Response(), &http.Cookie{
		Name:     name,
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		Expires:  time.Unix(0, 0),
		HttpOnly: true,
		Secure:   c.Request().TLS != nil,
		SameSite: http.SameSiteLaxMode,
	})
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed != "" {
			return trimmed
		}
	}
	return ""
}
