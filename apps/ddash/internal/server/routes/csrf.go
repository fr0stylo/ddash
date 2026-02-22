package routes

import (
	"strings"

	"github.com/labstack/echo/v4"
)

func csrfToken(c echo.Context) string {
	value, ok := c.Get("csrf").(string)
	if !ok {
		return ""
	}
	return strings.TrimSpace(value)
}
