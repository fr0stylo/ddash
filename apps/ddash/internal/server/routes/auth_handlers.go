package routes

import (
	"errors"
	"net/http"
	"strings"

	"github.com/labstack/echo/v4"
	"github.com/markbates/goth/gothic"

	"github.com/fr0stylo/ddash/apps/ddash/internal/app/ports"
	appidentity "github.com/fr0stylo/ddash/apps/ddash/internal/application/identity"
	"github.com/fr0stylo/ddash/views/pages"
)

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
	delete(session.Values, "user")
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
	setSessionAuthUser(session, AuthUser{
		ID:        localUser.ID,
		Name:      firstNonEmpty(user.Name, nickname),
		NickName:  nickname,
		Email:     email,
		AvatarURL: user.AvatarURL,
	})
	session.Values[authSessionUserIDKey] = localUser.ID
	orgService := appidentity.NewService(a.store)
	org, orgErr := orgService.GetActiveOrDefaultOrganizationForUser(request.Context(), localUser.ID, 0)
	if orgErr == nil {
		session.Values[authSessionActiveOrgIDKey] = org.ID
	} else {
		delete(session.Values, authSessionActiveOrgIDKey)
	}
	if err := session.Save(request, c.Response()); err != nil {
		return err
	}
	if errors.Is(orgErr, appidentity.ErrOrganizationMembershipRequired) {
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

	orgService := appidentity.NewService(a.store)
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
	setSessionAuthUser(session, AuthUser{
		ID:        localUser.ID,
		Name:      firstNonEmpty(localUser.Name, localUser.Nickname),
		NickName:  localUser.Nickname,
		Email:     localUser.Email,
		AvatarURL: localUser.AvatarURL,
	})
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
	if errors.Is(orgErr, appidentity.ErrOrganizationMembershipRequired) {
		return c.Redirect(http.StatusFound, "/welcome")
	}
	if orgErr != nil {
		return orgErr
	}
	return c.Redirect(http.StatusFound, next)
}

func addProviderParam(request *http.Request, provider string) *http.Request {
	query := request.URL.Query()
	query.Set("provider", provider)
	request.URL.RawQuery = query.Encode()
	return request
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
