# Architecture

This document describes the current DDash runtime architecture and main boundaries.

## High-level flow

1. External systems send deployment events to `POST /webhooks/cdevents`.
2. The ingestion service validates auth token, signature, payload schema, and event type.
3. Valid events are appended to `event_store` (append-only model).
4. Read queries project service/deployment views from event history.
5. Echo routes render templ pages and HTMX/SSE fragments.

## Layers and boundaries

### HTTP / transport layer

- `internal/server/routes`
  - Auth routes (GitHub OAuth + session auth)
  - View routes split by domain files (home, services, deployments, settings, organizations)
  - Webhook routes
- `internal/webhooks/custom`
  - Thin transport adapter around ingestion service

### Application layer

- `internal/app/services`
  - Business use cases:
    - service read models
    - metadata update flow
    - organization settings flow
    - webhook event ingestion
- `internal/app/domain`
  - Domain-oriented view models
- `internal/app/ports`
  - Backend-agnostic interfaces and DTOs

Important rule: the app layer does not import `internal/db/queries` (enforced by `internal/app/services/architecture_guard_test.go`).

### Adapter layer

- `internal/adapters/sqlite`
  - Implements app ports using SQLite/sqlc
  - Maps app DTOs <-> sqlc query params/rows
  - Owns SQL transaction details for write operations

### Persistence layer

- `internal/db`
  - DB initialization, migrations, query wrappers
- `internal/db/queries`
  - sqlc-generated query code
- `internal/db/migrations`
  - schema evolution

## Data model (current behavior)

- `event_store`
  - Source of truth for deployment lifecycle events
  - Idempotency via unique `(organization_id, event_source, event_id)`
- `organizations`
  - Auth token, webhook secret, enabled flag
- `organization_required_fields`
  - Required metadata definitions and filterable flags
- `service_metadata`
  - Per-service metadata values
- `organization_environment_priorities`
  - Preferred environment ordering

## Security model

- Webhook auth uses bearer token -> organization lookup
- Signature verification uses HMAC SHA-256 of request body with org secret (`X-Webhook-Signature`)
- Session auth for UI uses GitHub OAuth + secure cookie session
- CSRF protection is enabled for state-changing UI routes (form `_csrf` token or `X-CSRF-Token` header)
- `DDASH_SESSION_SECRET` is required outside local/dev/test environments

## Runtime composition

Entrypoint: `cmd/server/main.go`

- Initializes logger and env
- Opens main DB (`DDASH_DB_PATH`)
- Wires routes:
  - auth routes
  - view routes with sqlite app/read store adapter
  - webhook routes with shared ingestion store factory (no per-request DB open/migration)

## Real-time UI behavior

- Service/deployment updates are streamed with SSE endpoints:
  - `/services/stream`
  - `/deployments/stream`
- HTML fragments are rendered server-side via templ + renderer helpers.
