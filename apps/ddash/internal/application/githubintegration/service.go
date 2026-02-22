package githubintegration

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/fr0stylo/ddash/apps/ddash/internal/app/ports"
	"github.com/fr0stylo/ddash/apps/ddash/internal/domains/githubintegration"
)

var (
	ErrSetupIntentNotFound = errors.New("setup intent not found")
	ErrSetupIntentExpired  = errors.New("setup intent expired")
)

type Installer interface {
	Enabled() bool
	StartInstall(state string) (string, error)
}

type Service struct {
	store     ports.GitHubInstallationStore
	installer Installer
}

func NewService(store ports.GitHubInstallationStore, installer Installer) *Service {
	return &Service{store: store, installer: installer}
}

func (s *Service) Enabled() bool {
	return s != nil && s.installer != nil && s.installer.Enabled()
}

func (s *Service) ListMappings(ctx context.Context, organizationID int64) ([]githubintegration.InstallationMapping, error) {
	rows, err := s.store.ListGitHubInstallationMappings(ctx, organizationID)
	if err != nil {
		return nil, err
	}
	out := make([]githubintegration.InstallationMapping, 0, len(rows))
	for _, row := range rows {
		out = append(out, githubintegration.InstallationMapping{
			InstallationID:     row.InstallationID,
			OrganizationID:     row.OrganizationID,
			OrganizationLabel:  row.OrganizationLabel,
			DefaultEnvironment: row.DefaultEnvironment,
			Enabled:            row.Enabled,
		})
	}
	return out, nil
}

func (s *Service) StartInstall(ctx context.Context, organizationID int64, organizationLabel, defaultEnvironment string) (string, error) {
	state, err := randomStateHex(16)
	if err != nil {
		return "", err
	}
	if err := s.store.CreateGitHubSetupIntent(ctx, ports.GitHubSetupIntent{
		State:              state,
		OrganizationID:     organizationID,
		OrganizationLabel:  strings.TrimSpace(organizationLabel),
		DefaultEnvironment: githubintegration.NormalizeEnvironment(defaultEnvironment, "production"),
		ExpiresAt:          time.Now().UTC().Add(15 * time.Minute),
	}); err != nil {
		return "", err
	}
	return s.installer.StartInstall(state)
}

func (s *Service) CompleteInstall(ctx context.Context, state string, installationID int64) error {
	intent, err := s.store.GetGitHubSetupIntentByState(ctx, strings.TrimSpace(state))
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ErrSetupIntentNotFound
		}
		return err
	}
	if time.Now().UTC().After(intent.ExpiresAt.UTC()) {
		_ = s.store.DeleteGitHubSetupIntent(ctx, state)
		return ErrSetupIntentExpired
	}
	if err := s.store.UpsertGitHubInstallationMapping(ctx, ports.GitHubInstallationMapping{
		InstallationID:     installationID,
		OrganizationID:     intent.OrganizationID,
		OrganizationLabel:  intent.OrganizationLabel,
		DefaultEnvironment: intent.DefaultEnvironment,
		Enabled:            true,
	}); err != nil {
		return err
	}
	return s.store.DeleteGitHubSetupIntent(ctx, state)
}

func (s *Service) DeleteMapping(ctx context.Context, organizationID int64, installationID int64) error {
	return s.store.DeleteGitHubInstallationMapping(ctx, installationID, organizationID)
}

func randomStateHex(size int) (string, error) {
	buf := make([]byte, size)
	if _, err := rand.Read(buf); err != nil {
		return "", fmt.Errorf("generate state: %w", err)
	}
	return hex.EncodeToString(buf), nil
}
