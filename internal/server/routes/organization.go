package routes

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"errors"

	"github.com/fr0stylo/ddash/internal/db"
	"github.com/fr0stylo/ddash/internal/db/queries"
)

func getOrCreateDefaultOrganization(ctx context.Context, database *db.Database) (queries.Organization, error) {
	org, err := database.GetDefaultOrganization(ctx)
	if err == nil {
		return org, nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return queries.Organization{}, err
	}

	authToken, err := randomHexToken(16)
	if err != nil {
		return queries.Organization{}, err
	}
	secret, err := randomHexToken(24)
	if err != nil {
		return queries.Organization{}, err
	}

	org, err = database.CreateOrganization(ctx, queries.CreateOrganizationParams{
		Name:          "default",
		AuthToken:     authToken,
		WebhookSecret: secret,
		Enabled:       1,
	})
	if err != nil {
		return database.GetDefaultOrganization(ctx)
	}
	return org, nil
}

func randomHexToken(size int) (string, error) {
	if size <= 0 {
		return "", errors.New("invalid token size")
	}
	buf := make([]byte, size)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return hex.EncodeToString(buf), nil
}
