# Plan: Sync Status Visibility Control

## Goal

Allow each organization to turn sync-status indicators on/off in dashboards.

When disabled, UI should not show status labels like synced/out-of-sync/unknown. This is for teams with fully automated deployment reconciliation where those indicators create noise.

## Product behavior

- Setting name: `Show sync status`
- Scope: organization-wide
- Default: `true` (preserve current behavior)
- Applies to:
  - home service cards/table
  - deployments rows
  - service detail status displays (where applicable)
- Does not change ingestion or projection semantics; only presentation.

## Technical design

### Schema

- Add `show_sync_status` column to `organizations`:
  - type: integer/boolean-like
  - not null
  - default `1`

### App ports and services

- Extend `ports.Organization` with `ShowSyncStatus bool`.
- Extend settings DTO/service to read and persist `ShowSyncStatus`.
- Ensure `ServiceReadService` receives org setting and applies status masking for UI-facing DTOs.

### Adapter layer

- Update sqlite mappers in `internal/adapters/sqlite` to map new field.
- Update `UpdateOrganizationSettings` flow to persist `show_sync_status`.

### Routes/templates

- Settings page: add toggle control for `Show sync status`.
- Home/deployments/service views: hide status chips/text when setting is false.
- SSE fragments should respect current setting through normal server rendering.

## Acceptance criteria

- Org with `show_sync_status = false`:
  - no sync status indicators shown on service/deployment dashboards.
- Org with `show_sync_status = true`:
  - existing status behavior unchanged.
- Setting persists after restart.
- Tests validate enabled and disabled render/service output paths.

## Rollout notes

- Safe migration: add column with default `1`; no data backfill needed.
- Backward compatible for existing installs.
