package routes

import (
	"net/http"
	"strings"
	"time"

	"github.com/labstack/echo/v4"
)

func isInvalidSecureCookieError(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	if !strings.Contains(msg, "securecookie") {
		return false
	}
	return strings.Contains(msg, "not valid") || strings.Contains(msg, "name not registered for interface")
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
