package server

import (
	"embed"
	"log/slog"
	"strings"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	slogecho "github.com/samber/slog-echo"

	"github.com/fr0stylo/ddash/internal/renderer"
)

// RouteRegister registers Echo routes.
type RouteRegister interface {
	RegisterRoutes(s *echo.Echo)
}

// Server holds the Echo instance.
type Server struct {
	e *echo.Echo
}

// New creates a new server instance.
func New(log *slog.Logger, publicFS embed.FS) *Server {
	e := echo.New()

	e.Renderer = &renderer.Renderer{}
	e.HideBanner = true
	e.HidePort = true

	e.Use(slogecho.New(log))
	e.Use(middleware.Recover())
	e.Use(middleware.RequestID())
	e.Use(middleware.CSRFWithConfig(middleware.CSRFConfig{
		TokenLookup: "header:X-CSRF-Token,form:_csrf",
		Skipper: func(c echo.Context) bool {
			return strings.HasPrefix(c.Path(), "/webhooks/")
		},
	}))

	e.StaticFS("/", publicFS)

	return &Server{
		e: e,
	}
}

// RegisterRouter attaches a route registrar.
func (s *Server) RegisterRouter(r RouteRegister) {
	r.RegisterRoutes(s.e)
}

// Start runs the HTTP server.
func (s *Server) Start(addr string) error {
	return s.e.Start(addr)
}
