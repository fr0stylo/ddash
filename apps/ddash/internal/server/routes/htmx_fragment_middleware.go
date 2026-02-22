package routes

import (
	"bytes"
	"fmt"
	"net/http"
	"strings"

	"github.com/labstack/echo/v4"
	"go.opentelemetry.io/otel/attribute"
)

func isHTMXRequest(c echo.Context) bool {
	return strings.EqualFold(strings.TrimSpace(c.Request().Header.Get("HX-Request")), "true")
}

func fragmentCacheKey(parts ...any) string {
	builder := strings.Builder{}
	for i, part := range parts {
		if i > 0 {
			builder.WriteString("|")
		}
		builder.WriteString(fmt.Sprint(part))
	}
	return builder.String()
}

func (v *ViewRoutes) htmxFragmentCacheMiddleware(fragment string) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			if v.fragments == nil {
				return next(c)
			}
			orgID, _ := GetActiveOrganizationID(c)
			attrs := []attribute.KeyValue{
				attribute.String("fragment", fragment),
				attribute.Int64("organization.id", orgID),
			}
			if !isHTMXRequest(c) || c.Request().Method != http.MethodGet {
				v.fragments.RecordBypass(c.Request().Context(), attrs...)
				return next(c)
			}

			key := fragmentCacheKey(fragment, orgID, strings.TrimSpace(c.Request().URL.RawQuery))
			if body, ok := v.fragments.TryGetTTL(c.Request().Context(), key, attrs...); ok {
				c.Response().Header().Set("X-Fragment-Cache", "hit")
				return c.Blob(http.StatusOK, echo.MIMETextHTMLCharsetUTF8, body)
			}

			writer := &captureResponseWriter{ResponseWriter: c.Response().Writer}
			c.Response().Writer = writer
			err := next(c)
			c.Response().Writer = writer.ResponseWriter
			if err != nil {
				v.fragments.RecordError(c.Request().Context(), attrs...)
				return err
			}

			status := writer.status
			if status == 0 {
				status = c.Response().Status
			}
			if status == 0 {
				status = http.StatusOK
			}
			contentType := strings.ToLower(strings.TrimSpace(c.Response().Header().Get(echo.HeaderContentType)))
			if status == http.StatusOK && strings.HasPrefix(contentType, "text/html") && writer.body.Len() > 0 {
				v.fragments.StoreTTL(c.Request().Context(), key, writer.body.Bytes(), attrs...)
				c.Response().Header().Set("X-Fragment-Cache", "miss")
			} else {
				v.fragments.RecordBypass(c.Request().Context(), attrs...)
			}

			return nil
		}
	}
}

type captureResponseWriter struct {
	http.ResponseWriter
	body   bytes.Buffer
	status int
}

func (w *captureResponseWriter) WriteHeader(code int) {
	w.status = code
	w.ResponseWriter.WriteHeader(code)
}

func (w *captureResponseWriter) Write(data []byte) (int, error) {
	if w.status == 0 {
		w.status = http.StatusOK
	}
	_, _ = w.body.Write(data)
	return w.ResponseWriter.Write(data)
}

func (w *captureResponseWriter) Flush() {
	if f, ok := w.ResponseWriter.(http.Flusher); ok {
		f.Flush()
	}
}
