package renderer

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"strings"

	"github.com/a-h/templ"
	"github.com/labstack/echo/v4"

	"github.com/fr0stylo/ddash/views/components"
)

// Renderer implements Echo's render interface for templ components.
type Renderer struct{}

// Render writes a templ component to the response writer.
func (t *Renderer) Render(w io.Writer, _ string, data interface{}, c echo.Context) error {
	tc, ok := data.(templ.Component)
	if !ok {
		return fmt.Errorf("invalid type %T", data)
	}

	return tc.Render(c.Request().Context(), w)
}

// DeploymentRow renders a deployment row to a string.
func DeploymentRow(ctx context.Context, row components.DeploymentRow) (string, error) {
	var buf bytes.Buffer
	if err := components.DeploymentRowItem(row).Render(ctx, &buf); err != nil {
		return "", err
	}
	return strings.ReplaceAll(buf.String(), "\n", ""), nil
}

// ServiceCard renders a service card to a string.
func ServiceCard(ctx context.Context, services []components.Service, service components.Service) (string, error) {
	var buf bytes.Buffer
	prodIndex, hasProd := components.ProductionCommitIndexForTitle(services, service.Title)
	if err := components.ServiceCard(service, prodIndex, hasProd).Render(ctx, &buf); err != nil {
		return "", err
	}
	return strings.ReplaceAll(buf.String(), "\n", ""), nil
}

// ServiceRow renders a service row to a string.
func ServiceRow(ctx context.Context, services []components.Service, service components.Service) (string, error) {
	var buf bytes.Buffer
	prodIndex, hasProd := components.ProductionCommitIndexForTitle(services, service.Title)
	if err := components.ServiceTableRow(service, prodIndex, hasProd).Render(ctx, &buf); err != nil {
		return "", err
	}
	return strings.ReplaceAll(buf.String(), "\n", ""), nil
}
