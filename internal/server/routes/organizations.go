package routes

import (
	"database/sql"
	"errors"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/labstack/echo/v4"

	services "github.com/fr0stylo/ddash/internal/app/services"
	"github.com/fr0stylo/ddash/views/pages"
)

func organizationsRedirectURL(message, level string) string {
	message = strings.TrimSpace(message)
	level = strings.TrimSpace(level)
	if message == "" {
		return "/organizations"
	}
	values := url.Values{}
	values.Set("msg", message)
	if level != "" {
		values.Set("level", level)
	}
	return "/organizations?" + values.Encode()
}

func (v *ViewRoutes) currentOrganizationID(c echo.Context) (int64, error) {
	ctx := c.Request().Context()
	userID, ok := GetAuthUserID(c)
	if !ok || userID <= 0 {
		return 0, services.ErrOrganizationAccessDenied
	}
	activeID, _ := GetActiveOrganizationID(c)
	org, err := v.orgs.GetActiveOrDefaultOrganizationForUser(ctx, userID, activeID)
	if err != nil {
		return 0, err
	}
	if err := SetActiveOrganizationID(c, org.ID); err != nil {
		return 0, err
	}
	return org.ID, nil
}

func (v *ViewRoutes) handleOrganizations(c echo.Context) error {
	ctx := c.Request().Context()
	activeID, err := v.currentOrganizationID(c)
	if err != nil {
		return err
	}
	next := strings.TrimSpace(c.QueryParam("next"))
	if next == "" || !strings.HasPrefix(next, "/") {
		next = "/"
	}
	userID, ok := GetAuthUserID(c)
	if !ok || userID <= 0 {
		return c.Redirect(http.StatusFound, "/login")
	}
	rows, err := v.orgs.ListOrganizationsForUser(ctx, userID)
	if err != nil {
		return err
	}
	flashMessage := strings.TrimSpace(c.QueryParam("msg"))
	flashLevel := strings.TrimSpace(c.QueryParam("level"))
	if flashLevel != "error" {
		flashLevel = "success"
	}
	items := make([]pages.OrganizationRow, 0, len(rows))
	for _, row := range rows {
		items = append(items, pages.OrganizationRow{
			ID:      row.ID,
			Name:    row.Name,
			Enabled: row.Enabled,
			Active:  row.ID == activeID,
		})
	}
	return c.Render(http.StatusOK, "", pages.OrganizationsPage(items, next, flashMessage, flashLevel, csrfToken(c)))
}

func (v *ViewRoutes) handleOrganizationCurrent(c echo.Context) error {
	ctx := c.Request().Context()
	orgID, err := v.currentOrganizationID(c)
	if err != nil {
		return err
	}
	org, err := v.orgs.GetOrganizationByID(ctx, orgID)
	if err != nil {
		return err
	}
	return c.JSON(http.StatusOK, map[string]interface{}{
		"id":      org.ID,
		"name":    org.Name,
		"enabled": org.Enabled,
	})
}

func (v *ViewRoutes) handleOrganizationCreate(c echo.Context) error {
	ctx := c.Request().Context()
	name := strings.TrimSpace(c.FormValue("name"))
	if name == "" {
		return c.Redirect(http.StatusFound, organizationsRedirectURL("Organization name is required", "error"))
	}
	userID, ok := GetAuthUserID(c)
	if !ok || userID <= 0 {
		return c.Redirect(http.StatusFound, "/login")
	}
	org, err := v.orgs.CreateOrganization(ctx, userID, name)
	if err != nil {
		return err
	}
	if err := SetActiveOrganizationID(c, org.ID); err != nil {
		return err
	}
	return c.Redirect(http.StatusFound, organizationsRedirectURL("Organization created and selected", "success"))
}

func (v *ViewRoutes) handleOrganizationSwitch(c echo.Context) error {
	ctx := c.Request().Context()
	value := strings.TrimSpace(c.FormValue("organizationID"))
	if value == "" {
		return c.Redirect(http.StatusFound, organizationsRedirectURL("Organization id is required", "error"))
	}
	id, err := strconv.ParseInt(value, 10, 64)
	if err != nil || id <= 0 {
		return c.Redirect(http.StatusFound, organizationsRedirectURL("Invalid organization id", "error"))
	}
	org, err := v.orgs.GetOrganizationByID(ctx, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return c.Redirect(http.StatusFound, organizationsRedirectURL("Organization not found", "error"))
		}
		return err
	}
	if !org.Enabled {
		return c.Redirect(http.StatusFound, organizationsRedirectURL("Cannot switch to disabled organization", "error"))
	}
	userID, ok := GetAuthUserID(c)
	if !ok || userID <= 0 {
		return c.Redirect(http.StatusFound, "/login")
	}
	if _, err := v.orgs.GetActiveOrDefaultOrganizationForUser(ctx, userID, id); err != nil {
		return c.Redirect(http.StatusFound, organizationsRedirectURL("Organization access denied", "error"))
	}
	if err := SetActiveOrganizationID(c, id); err != nil {
		return err
	}
	next := strings.TrimSpace(c.FormValue("next"))
	if next == "" {
		referer := strings.TrimSpace(c.Request().Referer())
		if referer != "" {
			if parsed, err := url.Parse(referer); err == nil {
				next = parsed.EscapedPath()
				if parsed.RawQuery != "" {
					next = next + "?" + parsed.RawQuery
				}
			}
		}
	}
	if next == "" {
		next = "/"
	}
	if !strings.HasPrefix(next, "/") {
		if u := c.Request().URL; u != nil {
			next = u.Path
		}
		if next == "" {
			next = "/"
		}
	}
	return c.Redirect(http.StatusFound, next)
}

func (v *ViewRoutes) handleOrganizationRename(c echo.Context) error {
	ctx := c.Request().Context()
	id, err := strconv.ParseInt(strings.TrimSpace(c.FormValue("organizationID")), 10, 64)
	if err != nil || id <= 0 {
		return c.Redirect(http.StatusFound, organizationsRedirectURL("Invalid organization id", "error"))
	}
	name := strings.TrimSpace(c.FormValue("name"))
	if name == "" {
		return c.Redirect(http.StatusFound, organizationsRedirectURL("Organization name is required", "error"))
	}
	if err := v.requireOrganizationAdmin(c, id); err != nil {
		return err
	}
	if err := v.orgs.RenameOrganization(ctx, id, name); err != nil {
		return err
	}
	return c.Redirect(http.StatusFound, organizationsRedirectURL("Organization renamed", "success"))
}

func (v *ViewRoutes) handleOrganizationToggle(c echo.Context) error {
	ctx := c.Request().Context()
	id, err := strconv.ParseInt(strings.TrimSpace(c.FormValue("organizationID")), 10, 64)
	if err != nil || id <= 0 {
		return c.Redirect(http.StatusFound, organizationsRedirectURL("Invalid organization id", "error"))
	}
	enabled := strings.EqualFold(strings.TrimSpace(c.FormValue("enabled")), "true")
	if err := v.requireOrganizationAdmin(c, id); err != nil {
		return err
	}
	if err := v.orgs.SetOrganizationEnabled(ctx, id, enabled); err != nil {
		return err
	}
	if !enabled {
		if activeID, ok := GetActiveOrganizationID(c); ok && activeID == id {
			if resolvedID, resolveErr := v.currentOrganizationID(c); resolveErr == nil {
				_ = SetActiveOrganizationID(c, resolvedID)
			}
		}
	}
	if enabled {
		return c.Redirect(http.StatusFound, organizationsRedirectURL("Organization enabled", "success"))
	}
	return c.Redirect(http.StatusFound, organizationsRedirectURL("Organization disabled", "success"))
}

func (v *ViewRoutes) handleOrganizationDelete(c echo.Context) error {
	ctx := c.Request().Context()
	id, err := strconv.ParseInt(strings.TrimSpace(c.FormValue("organizationID")), 10, 64)
	if err != nil || id <= 0 {
		return c.Redirect(http.StatusFound, organizationsRedirectURL("Invalid organization id", "error"))
	}
	if err := v.requireOrganizationAdmin(c, id); err != nil {
		return err
	}
	if err := v.orgs.DeleteOrganization(ctx, id); err != nil {
		if errors.Is(err, services.ErrCannotDeleteLastOrganization) {
			return c.Redirect(http.StatusFound, organizationsRedirectURL("Cannot delete the last organization", "error"))
		}
		return err
	}
	if activeID, ok := GetActiveOrganizationID(c); ok && activeID == id {
		if resolvedID, resolveErr := v.currentOrganizationID(c); resolveErr == nil {
			_ = SetActiveOrganizationID(c, resolvedID)
		}
	}
	return c.Redirect(http.StatusFound, organizationsRedirectURL("Organization deleted", "success"))
}

func (v *ViewRoutes) handleOrganizationMembers(c echo.Context) error {
	ctx := c.Request().Context()
	orgID, err := v.currentOrganizationID(c)
	if err != nil {
		return err
	}
	if err := v.requireOrganizationAdmin(c, orgID); err != nil {
		return err
	}
	org, err := v.orgs.GetOrganizationByID(ctx, orgID)
	if err != nil {
		return err
	}
	rows, err := v.orgs.ListMembers(ctx, orgID)
	if err != nil {
		return err
	}
	selfID, _ := GetAuthUserID(c)
	items := make([]pages.OrganizationMemberRow, 0, len(rows))
	for _, row := range rows {
		display := strings.TrimSpace(row.Name)
		if display == "" {
			display = row.Nickname
		}
		items = append(items, pages.OrganizationMemberRow{
			UserID:   row.UserID,
			Display:  display,
			Email:    row.Email,
			Nickname: row.Nickname,
			Role:     row.Role,
			Self:     row.UserID == selfID,
		})
	}
	flashMessage := strings.TrimSpace(c.QueryParam("msg"))
	flashLevel := strings.TrimSpace(c.QueryParam("level"))
	if flashLevel != "error" {
		flashLevel = "success"
	}
	return c.Render(http.StatusOK, "", pages.OrganizationMembersPage(org.Name, items, flashMessage, flashLevel, csrfToken(c)))
}

func (v *ViewRoutes) handleOrganizationMemberAdd(c echo.Context) error {
	ctx := c.Request().Context()
	orgID, err := v.currentOrganizationID(c)
	if err != nil {
		return err
	}
	if err := v.requireOrganizationAdmin(c, orgID); err != nil {
		return err
	}
	identity := strings.TrimSpace(c.FormValue("identity"))
	role := strings.TrimSpace(c.FormValue("role"))
	if err := v.orgs.AddMemberByLookup(ctx, orgID, identity, role); err != nil {
		return c.Redirect(http.StatusFound, "/organizations/members?msg=Unable+to+add+member&level=error")
	}
	return c.Redirect(http.StatusFound, "/organizations/members?msg=Member+added&level=success")
}

func (v *ViewRoutes) handleOrganizationMemberRole(c echo.Context) error {
	ctx := c.Request().Context()
	orgID, err := v.currentOrganizationID(c)
	if err != nil {
		return err
	}
	if err := v.requireOrganizationAdmin(c, orgID); err != nil {
		return err
	}
	userID, err := strconv.ParseInt(strings.TrimSpace(c.FormValue("userID")), 10, 64)
	if err != nil || userID <= 0 {
		return c.Redirect(http.StatusFound, "/organizations/members?msg=Invalid+user&level=error")
	}
	role := strings.TrimSpace(c.FormValue("role"))
	if err := v.orgs.UpdateMemberRole(ctx, orgID, userID, role); err != nil {
		if errors.Is(err, services.ErrCannotRemoveLastOwner) {
			return c.Redirect(http.StatusFound, "/organizations/members?msg=Cannot+remove+last+owner&level=error")
		}
		return c.Redirect(http.StatusFound, "/organizations/members?msg=Unable+to+update+role&level=error")
	}
	return c.Redirect(http.StatusFound, "/organizations/members?msg=Role+updated&level=success")
}

func (v *ViewRoutes) handleOrganizationMemberRemove(c echo.Context) error {
	ctx := c.Request().Context()
	orgID, err := v.currentOrganizationID(c)
	if err != nil {
		return err
	}
	if err := v.requireOrganizationAdmin(c, orgID); err != nil {
		return err
	}
	userID, err := strconv.ParseInt(strings.TrimSpace(c.FormValue("userID")), 10, 64)
	if err != nil || userID <= 0 {
		return c.Redirect(http.StatusFound, "/organizations/members?msg=Invalid+user&level=error")
	}
	if err := v.orgs.RemoveMember(ctx, orgID, userID); err != nil {
		if errors.Is(err, services.ErrCannotRemoveLastOwner) {
			return c.Redirect(http.StatusFound, "/organizations/members?msg=Cannot+remove+last+owner&level=error")
		}
		return c.Redirect(http.StatusFound, "/organizations/members?msg=Unable+to+remove+member&level=error")
	}
	return c.Redirect(http.StatusFound, "/organizations/members?msg=Member+removed&level=success")
}

func (v *ViewRoutes) requireOrganizationAdmin(c echo.Context, organizationID int64) error {
	ctx := c.Request().Context()
	userID, ok := GetAuthUserID(c)
	if !ok || userID <= 0 {
		return c.Redirect(http.StatusFound, "/login")
	}
	canManage, err := v.orgs.CanManageOrganization(ctx, organizationID, userID)
	if err != nil {
		return err
	}
	if !canManage {
		return c.Redirect(http.StatusFound, organizationsRedirectURL("Organization admin access required", "error"))
	}
	return nil
}
