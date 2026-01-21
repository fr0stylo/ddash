package routes

import (
	"encoding/gob"
	"net/http"
	"time"

	"github.com/gorilla/sessions"
	"github.com/labstack/echo/v4"
	"github.com/markbates/goth"
	"github.com/markbates/goth/gothic"
	"github.com/markbates/goth/providers/github"

	"github.com/fr0stylo/ddash/views/pages"
)

const authSessionName = "ddash-auth"

const githubProvider = "github"

type AuthConfig struct {
	SessionKey         string
	GitHubClientID     string
	GitHubClientSecret string
	GitHubCallbackURL  string
	SecureCookies      bool
}

type AuthUser struct {
	Name      string
	NickName  string
	Email     string
	AvatarURL string
}

func init() {
	gob.Register(AuthUser{})
}

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

type AuthRoutes struct{}

func NewAuthRoutes() *AuthRoutes {
	return &AuthRoutes{}
}

func (a *AuthRoutes) RegisterRoutes(s *echo.Echo) {
	s.GET("/login", a.handleLogin)
	s.GET("/logout", a.handleLogout)
	s.GET("/auth/:provider", a.handleAuthBegin)
	s.GET("/auth/:provider/callback", a.handleAuthCallback)
}

func RequireAuth(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		user, ok := authUserFromSession(c)
		if !ok {
			return c.Redirect(http.StatusFound, "/login")
		}
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
		return err
	}
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
		return err
	}

	session, err := gothic.Store.Get(request, authSessionName)
	if err != nil {
		return err
	}
	session.Values["user"] = AuthUser{
		Name:      user.Name,
		NickName:  user.NickName,
		Email:     user.Email,
		AvatarURL: user.AvatarURL,
	}
	if err := session.Save(request, c.Response()); err != nil {
		return err
	}
	return c.Redirect(http.StatusFound, "/")
}

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

func addProviderParam(request *http.Request, provider string) *http.Request {
	query := request.URL.Query()
	query.Set("provider", provider)
	request.URL.RawQuery = query.Encode()
	return request
}

func authUserFromSession(c echo.Context) (AuthUser, bool) {
	session, err := gothic.Store.Get(c.Request(), authSessionName)
	if err != nil {
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
