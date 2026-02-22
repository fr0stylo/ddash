package routes

import (
	"net/http"

	"github.com/labstack/echo/v4"

	"github.com/fr0stylo/ddash/internal/observability"
)

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
