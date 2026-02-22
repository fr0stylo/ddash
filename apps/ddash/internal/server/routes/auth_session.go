package routes

import (
	"fmt"
	"strings"

	"github.com/gorilla/sessions"
	"github.com/labstack/echo/v4"
	"github.com/markbates/goth/gothic"
)

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
	return authUserFromSessionValue(value)
}

func setSessionAuthUser(session *sessions.Session, user AuthUser) {
	session.Values["user"] = user
}

func authUserFromSessionValue(value any) (AuthUser, bool) {
	if user, ok := value.(AuthUser); ok {
		return user, true
	}
	if user, ok := value.(legacyAuthUser); ok {
		return AuthUser(user), true
	}
	fields, ok := value.(map[string]any)
	if !ok {
		return AuthUser{}, false
	}

	id, ok := toInt64(fields["id"])
	if !ok || id <= 0 {
		return AuthUser{}, false
	}

	return AuthUser{
		ID:        id,
		Name:      toString(fields["name"]),
		NickName:  toString(fields["nickname"]),
		Email:     toString(fields["email"]),
		AvatarURL: toString(fields["avatar_url"]),
	}, true
}

func toInt64(value any) (int64, bool) {
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

func toString(value any) string {
	if str, ok := value.(string); ok {
		return strings.TrimSpace(str)
	}
	return ""
}
