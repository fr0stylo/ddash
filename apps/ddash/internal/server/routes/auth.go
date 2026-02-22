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

	"github.com/fr0stylo/ddash/apps/ddash/internal/app/ports"
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

type legacyAuthUser struct {
	ID        int64
	Name      string
	NickName  string
	Email     string
	AvatarURL string
}

func init() {
	gob.Register(AuthUser{})
	gob.Register(map[string]any{})
	gob.RegisterName("github.com/fr0stylo/ddash/internal/server/routes.AuthUser", legacyAuthUser{})
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
