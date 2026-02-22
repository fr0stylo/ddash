package routes

import (
	"database/sql"
	"errors"
	"net/http"
	"net/url"
	"strings"

	"github.com/labstack/echo/v4"

	appidentity "github.com/fr0stylo/ddash/apps/ddash/internal/application/identity"
	"github.com/fr0stylo/ddash/views/pages"
)

func welcomeRedirectURL(message, level string) string {
	values := url.Values{}
	if strings.TrimSpace(message) != "" {
		values.Set("msg", strings.TrimSpace(message))
	}
	if strings.TrimSpace(level) != "" {
		values.Set("level", strings.TrimSpace(level))
	}
	if len(values) == 0 {
		return "/welcome"
	}
	return "/welcome?" + values.Encode()
}

func (v *ViewRoutes) handleWelcome(c echo.Context) error {
	ctx := c.Request().Context()
	userID, ok := GetAuthUserID(c)
	if !ok || userID <= 0 {
		return c.Redirect(http.StatusFound, "/login")
	}
	activeID, _ := GetActiveOrganizationID(c)
	if _, err := v.orgs.GetActiveOrDefaultOrganizationForUser(ctx, userID, activeID); err == nil {
		return c.Redirect(http.StatusFound, "/")
	} else if !errors.Is(err, appidentity.ErrOrganizationMembershipRequired) {
		return err
	}

	user, _ := GetAuthUser(c)
	label := strings.TrimSpace(user.NickName)
	if label == "" {
		label = strings.TrimSpace(user.Name)
	}
	flashMessage := strings.TrimSpace(c.QueryParam("msg"))
	flashLevel := strings.TrimSpace(c.QueryParam("level"))
	if flashLevel != "error" {
		flashLevel = "success"
	}
	return c.Render(http.StatusOK, "", pages.WelcomePage(label, flashMessage, flashLevel, csrfToken(c)))
}

func (v *ViewRoutes) handleWelcomeCreateOrganization(c echo.Context) error {
	ctx := c.Request().Context()
	userID, ok := GetAuthUserID(c)
	if !ok || userID <= 0 {
		return c.Redirect(http.StatusFound, "/login")
	}
	userID, err := v.ensureAuthUserRecord(c, userID)
	if err != nil {
		return c.Redirect(http.StatusFound, welcomeRedirectURL("Unable to validate user profile", "error"))
	}
	name := strings.TrimSpace(c.FormValue("name"))
	if name == "" {
		user, _ := GetAuthUser(c)
		name = strings.TrimSpace(user.NickName)
		if name == "" {
			name = "my"
		}
		name = strings.ToLower(name) + "-org"
	}
	org, err := v.orgs.CreateOrganization(ctx, userID, name)
	if err != nil {
		c.Logger().Errorf("failed to create organization for user %d: %v", userID, err)
		return c.Redirect(http.StatusFound, welcomeRedirectURL("Unable to create organization", "error"))
	}
	if err := SetActiveOrganizationID(c, org.ID); err != nil {
		return err
	}
	return c.Redirect(http.StatusFound, "/")
}

func (v *ViewRoutes) handleWelcomeJoinOrganization(c echo.Context) error {
	ctx := c.Request().Context()
	userID, ok := GetAuthUserID(c)
	if !ok || userID <= 0 {
		return c.Redirect(http.StatusFound, "/login")
	}
	userID, err := v.ensureAuthUserRecord(c, userID)
	if err != nil {
		return c.Redirect(http.StatusFound, welcomeRedirectURL("Unable to validate user profile", "error"))
	}
	joinCode := strings.TrimSpace(c.FormValue("joinCode"))
	if joinCode == "" {
		return c.Redirect(http.StatusFound, welcomeRedirectURL("Join code is required", "error"))
	}
	if err := v.orgs.RequestJoinByCode(ctx, userID, joinCode); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return c.Redirect(http.StatusFound, welcomeRedirectURL("Invalid join code", "error"))
		}
		return c.Redirect(http.StatusFound, welcomeRedirectURL("Unable to submit request", "error"))
	}
	return c.Redirect(http.StatusFound, welcomeRedirectURL("Join request submitted. Wait for admin approval.", "success"))
}
