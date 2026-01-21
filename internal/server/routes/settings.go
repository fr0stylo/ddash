package routes

import (
	"net/http"
	"strings"

	"github.com/labstack/echo/v4"

	"github.com/fr0stylo/ddash/internal/db/queries"
	"github.com/fr0stylo/ddash/views/components"
	"github.com/fr0stylo/ddash/views/pages"
)

type settingsPayload struct {
	AuthToken      string               `json:"authToken"`
	WebhookSecret  string               `json:"webhookSecret"`
	Enabled        bool                 `json:"enabled"`
	RequiredFields []settingsFieldInput `json:"requiredFields"`
}

type settingsFieldInput struct {
	Label string `json:"label"`
	Type  string `json:"type"`
}

func (v *ViewRoutes) handleSettings(c echo.Context) error {
	ctx := c.Request().Context()
	org, err := getOrCreateDefaultOrganization(ctx, v.db)
	if err != nil {
		return err
	}
	fields, err := v.db.ListOrganizationRequiredFields(ctx, org.ID)
	if err != nil {
		return err
	}

	return c.Render(http.StatusOK, "", pages.SettingsPage(
		normalizeSettingsFields(fields),
		org.AuthToken,
		org.WebhookSecret,
		org.Enabled != 0,
	))
}

func (v *ViewRoutes) handleSettingsUpdate(c echo.Context) error {
	ctx := c.Request().Context()
	org, err := getOrCreateDefaultOrganization(ctx, v.db)
	if err != nil {
		return err
	}

	payload := settingsPayload{}
	if err := c.Bind(&payload); err != nil {
		return err
	}

	payload.AuthToken = strings.TrimSpace(payload.AuthToken)
	payload.WebhookSecret = strings.TrimSpace(payload.WebhookSecret)

	if err := v.db.UpdateOrganizationSecrets(ctx, payload.AuthToken, payload.WebhookSecret, payload.Enabled, org.ID); err != nil {
		return err
	}

	if err := v.db.DeleteOrganizationRequiredFields(ctx, org.ID); err != nil {
		return err
	}

	for index, field := range payload.RequiredFields {
		label := strings.TrimSpace(field.Label)
		fieldType := strings.TrimSpace(field.Type)
		if label == "" || fieldType == "" {
			continue
		}
		_, err := v.db.CreateOrganizationRequiredField(ctx, queries.CreateOrganizationRequiredFieldParams{
			OrganizationID: org.ID,
			Label:          label,
			FieldType:      fieldType,
			SortOrder:      int64(index),
		})
		if err != nil {
			return err
		}
	}

	return c.NoContent(http.StatusNoContent)
}

func normalizeSettingsFields(rows []queries.ListOrganizationRequiredFieldsRow) []components.ServiceField {
	fields := make([]components.ServiceField, 0, len(rows))
	for _, row := range rows {
		fields = append(fields, components.ServiceField{
			Label: row.Label,
			Value: row.FieldType,
		})
	}
	return fields
}
