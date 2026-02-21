package routes

import (
	"fmt"
	"strings"

	"github.com/labstack/echo/v4"
	"github.com/markbates/goth/gothic"

	"github.com/fr0stylo/ddash/internal/app/ports"
)

func (v *ViewRoutes) ensureAuthUserRecord(c echo.Context, sessionUserID int64) (int64, error) {
	user, ok := GetAuthUser(c)
	if !ok {
		return sessionUserID, nil
	}
	email := strings.TrimSpace(user.Email)
	if email == "" {
		email = strings.TrimSpace(user.NickName)
		if email == "" {
			email = fmt.Sprintf("user-%d@local.invalid", sessionUserID)
		} else {
			email = strings.ToLower(email) + "@local.invalid"
		}
	}
	nickname := strings.TrimSpace(user.NickName)
	if nickname == "" {
		nickname = strings.Split(email, "@")[0]
	}
	name := strings.TrimSpace(user.Name)
	if name == "" {
		name = nickname
	}
	githubID := fmt.Sprintf("session:%s", strings.ToLower(email))

	upserted, err := v.orgs.EnsureUser(c.Request().Context(), ports.UpsertUserInput{
		GitHubID:  githubID,
		Email:     email,
		Nickname:  nickname,
		Name:      name,
		AvatarURL: strings.TrimSpace(user.AvatarURL),
	})
	if err != nil {
		return sessionUserID, err
	}
	if upserted.ID <= 0 {
		return sessionUserID, nil
	}
	if upserted.ID == sessionUserID {
		return sessionUserID, nil
	}

	session, err := gothic.Store.Get(c.Request(), authSessionName)
	if err != nil {
		return sessionUserID, err
	}
	updatedUser := user
	updatedUser.ID = upserted.ID
	updatedUser.Email = upserted.Email
	updatedUser.NickName = upserted.Nickname
	updatedUser.Name = upserted.Name
	updatedUser.AvatarURL = upserted.AvatarURL
	session.Values["user"] = updatedUser
	session.Values[authSessionUserIDKey] = upserted.ID
	if err := session.Save(c.Request(), c.Response()); err != nil {
		return sessionUserID, err
	}
	c.Set("authUser", updatedUser)
	return upserted.ID, nil
}
