package observability

import (
	"path"
	"strings"

	"github.com/labstack/echo/v4"
	"go.opentelemetry.io/contrib/instrumentation/github.com/labstack/echo/otelecho"
)

// EchoMiddleware returns the unified HTTP tracing middleware.
func EchoMiddleware() echo.MiddlewareFunc {
	return otelecho.Middleware("ddash", otelecho.WithSkipper(traceSkipper))
}

// EchoSpanEnrichmentMiddleware adds request attributes to the active root span.
func EchoSpanEnrichmentMiddleware() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			ctx := c.Request().Context()
			ctx = WithRequestMetadata(ctx, c.Response().Header().Get(echo.HeaderXRequestID), resolvedRoute(c))
			c.SetRequest(c.Request().WithContext(ctx))

			err := next(c)

			ctx = WithRequestMetadata(c.Request().Context(), c.Response().Header().Get(echo.HeaderXRequestID), resolvedRoute(c))
			c.SetRequest(c.Request().WithContext(ctx))
			return err
		}
	}
}

func traceSkipper(c echo.Context) bool {
	requestPath := strings.TrimSpace(c.Request().URL.Path)
	if requestPath == "" {
		return false
	}

	switch requestPath {
	case "/health", "/healthz", "/live", "/ready", "/favicon.ico", "/auth/github/callback":
		return true
	}

	if strings.HasPrefix(requestPath, "/public/") {
		return true
	}

	ext := strings.ToLower(path.Ext(requestPath))
	switch ext {
	case ".css", ".js", ".map", ".png", ".jpg", ".jpeg", ".gif", ".svg", ".ico", ".woff", ".woff2", ".ttf":
		return true
	default:
		return false
	}
}

func resolvedRoute(c echo.Context) string {
	route := strings.TrimSpace(c.Path())
	if route != "" {
		return route
	}
	return strings.TrimSpace(c.Request().URL.Path)
}
