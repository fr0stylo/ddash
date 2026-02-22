# Feature Catalog (Build Intake)

This document is the intake point for planning and building new features in DDash.

Use it as:
- a source of truth for what already exists,
- a backlog of candidate features,
- a lightweight template for defining new work before implementation.

## Current capabilities (implemented)

### Ingestion and event pipeline

- CDEvents endpoint: `POST /webhooks/cdevents`
- Legacy endpoint disabled: `POST /webhooks/custom` returns `410 Gone`
- Validation pipeline:
  - bearer token organization lookup
  - HMAC SHA-256 signature check (`X-Webhook-Signature`)
  - CDEvents schema validation (SDK v0.5 parser/validator)
  - event type allowlist validation
- Accepted delivery event types:
  - `dev.cdevents.environment.created.0.3.0`
  - `dev.cdevents.environment.modified.0.3.0`
  - `dev.cdevents.environment.deleted.0.3.0`
  - `dev.cdevents.service.deployed.0.3.0`
  - `dev.cdevents.service.upgraded.0.3.0`
  - `dev.cdevents.service.rolledback.0.3.0`
  - `dev.cdevents.service.removed.0.3.0`
  - `dev.cdevents.service.published.0.3.0`
- Event persistence:
  - append-only `event_store`
  - idempotency on `(event_source, event_id)`

### Deployment visibility

- Home dashboard with service projections
- Deployments page with projected deployment rows
- Service details page with:
  - environment-level latest deploy state
  - deployment history
  - integration type display

### Metadata and governance

- Organization-required metadata fields
- Filterable metadata flags for query/filter UI
- Service metadata editing from service details
- Missing metadata count + badge severity behavior

### Settings and controls

- Webhook auth token/secret configuration
- Organization enable/disable state
- Environment priority ordering per organization
- Organization feature toggle: show/hide sync status on dashboards

### Organization-level feature controls (implemented)

- `show_sync_status`
- `show_metadata_badges`
- `show_environment_column`
- `enable_sse_live_updates`
- `show_deployment_history`
- `show_metadata_filters`
- `strict_metadata_enforcement`
- `mask_sensitive_metadata_values`
- `allow_service_metadata_editing`
- `show_onboarding_hints`
- `show_integration_type_badges`

### Organization-level preferences (implemented)

- `deployment_retention_days`
- `default_dashboard_view` (`grid` / `table`)
- `status_semantics_mode` (`technical` / `plain`)

### Multi-organization management

- Organization list/create/switch UI (`/organizations`)
- Active organization context stored in session
- Tenant-aware services/deployments/settings/metadata flows
- Organization rename and enable/disable controls
- Safe delete with guard (`cannot delete last organization`)

### Runtime and quality

- SSE update endpoints for services/deployments
- Unified quality gate: `task check` (`fmt`, `vet`, `lint`, `test`)
- App-layer architectural guard against sqlc coupling

## Intake template for new features

Create a new entry in the backlog section using this template:

```md
### <Feature name>
- **Problem**: <what user/system pain this solves>
- **Outcome**: <measurable result>
- **Primary users**: <who benefits>
- **Scope (MVP)**:
  - <item>
  - <item>
- **Out of scope**:
  - <item>
- **Architecture touchpoints**:
  - Routes: <paths>
  - App services: <service names>
  - Ports/adapters: <ports and adapters>
  - DB/migrations: <tables/migrations>
- **Acceptance criteria**:
  - [ ] <behavioral expectation>
  - [ ] <testability/verification expectation>
- **Rollout/ops notes**: <flags, migration order, backfill, compatibility>
```

## Prioritized backlog (feature build queue)

### 1) Service sync-status visibility control
- **Problem**: teams with fully automated deploy reconciliation do not want out-of-sync/sync badges to imply manual action.
- **Outcome**: each organization can choose whether sync status is shown on dashboards.
- **Primary users**: platform teams and service owners using automated rollout flows.
- **Scope (MVP)**:
  - org-level setting: `show_sync_status` (default `true` for backward compatibility)
  - hide/suppress sync-status UI badges/labels in service cards/tables and deployment rows when disabled
  - keep internal status computation unchanged (presentation toggle only)
  - settings page toggle with immediate effect on page refresh/SSE render
- **Out of scope**:
  - changing ingest semantics
  - removing status from stored projections/events
  - per-service override (org-level only in MVP)
- **Architecture touchpoints**:
  - Routes: `apps/ddash/internal/server/routes/view.go`, `apps/ddash/internal/server/routes/settings.go`
  - App services: `ServiceReadService`, `OrganizationConfigService`
  - Ports/adapters: org settings DTOs and sqlite store mapping
  - DB/migrations: add `organizations.show_sync_status` or dedicated settings table field
- **Acceptance criteria**:
  - [ ] when disabled, sync/out-of-sync status is not shown in dashboard/service/deployments UI
  - [ ] when enabled, existing status behavior remains unchanged
  - [ ] toggle persists per organization and survives restart
  - [ ] tests cover both enabled/disabled render paths
- **Rollout/ops notes**: migration should default to enabled; no data backfill required.

### 2) Standalone ingestion service
- **Problem**: webhook ingest lifecycle is coupled to dashboard runtime.
- **Outcome**: independent scaling/reliability for ingestion.
- **MVP scope**:
  - isolate ingestion command handling into separate process
  - preserve app-level ingestion contract
  - durable handoff to projection consumers
- **Architecture touchpoints**:
  - app ports: ingestion interfaces
  - adapters: grpc/http/event bus adapter
  - ops: deployment topology, health checks

### 3) Advanced deployment health views
- **Problem**: dashboard shows status snapshots but limited trend/health context.
- **Outcome**: faster diagnosis of rollout regressions.
- **MVP scope**:
  - per-service deployment success/error trend cards
  - environment risk indicators
  - recent failures panel with links to service detail

### 4) Metadata-driven ownership drilldowns
- **Problem**: metadata exists but ownership/impact navigation is limited.
- **Outcome**: teams can filter and manage by owner/team metadata quickly.
- **MVP scope**:
  - owner/team landing filters
  - aggregate counts by metadata dimensions
  - quick links from dashboards to filtered lists

### 5) Alert hooks and notifications
- **Problem**: no native proactive notification on failed/stuck deployments.
- **Outcome**: teams are notified without watching dashboard continuously.
- **MVP scope**:
  - threshold/condition rules
  - webhook/slack notifier integration
  - dedupe/rate-limiting for noisy events

## Recently completed

### Multi-organization management UI
- **Problem**: current UX implicitly relies on default org fallback; explicit org management is limited.
- **Outcome**: users can create/select/switch orgs safely from UI.
- **MVP scope**:
  - org selector in authenticated layout and header context
  - org create/switch/rename/enable-disable/delete controls
  - tenant-aware settings and service/deployment projections
  - session-safe active-org fallback behavior

## Definition of ready (before implementation starts)

A feature is ready when:
- problem and outcome are explicitly written,
- architecture touchpoints are identified,
- migration and compatibility plan exists (if schema/runtime changes),
- acceptance criteria are testable,
- rollout plan is documented.

## Definition of done (before merge)

- behavior implemented according to acceptance criteria,
- tests added/updated (`go test ./...` passes),
- `task check` passes,
- docs updated (`docs/architecture.md`, this file, and endpoint notes if needed),
- backward compatibility and migration notes documented.
