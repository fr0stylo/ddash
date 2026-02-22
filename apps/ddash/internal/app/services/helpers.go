package services

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"errors"

	"github.com/fr0stylo/ddash/apps/ddash/internal/app/ports"
)

func getOrCreateDefaultOrganization(ctx context.Context, store ports.AppStore) (ports.Organization, error) {
	org, err := store.GetDefaultOrganization(ctx)
	if err == nil {
		return org, nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return ports.Organization{}, err
	}

	authToken, err := randomHexToken(16)
	if err != nil {
		return ports.Organization{}, err
	}
	secret, err := randomHexToken(24)
	if err != nil {
		return ports.Organization{}, err
	}

	created, err := store.CreateOrganization(ctx, ports.CreateOrganizationInput{
		Name:          "default",
		AuthToken:     authToken,
		JoinCode:      "default-org",
		WebhookSecret: secret,
		Enabled:       true,
	})
	if err == nil {
		return created, nil
	}

	return store.GetDefaultOrganization(ctx)
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

func normalizeEnvironmentOrder(values []string) []string {
	return normalizeEnvironmentOrderInput(values)
}
