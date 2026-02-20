# DDash

DDash is a deployment dashboard for tracking service deployments across environments.

It ingests CDEvents webhooks, stores immutable events in SQLite, and renders projected views for services and deployments.

## Documentation

- Architecture: `docs/architecture.md`
- Features and possibilities: `docs/features-and-possibilities.md`
- Feature catalog (build intake): `docs/feature-catalog.md`
- Next feature plan: `docs/plans/sync-status-visibility.md`
- Playwright checklist: `docs/testing/playwright-checklist.md`

## Quick start

1. Run server:

```bash
export DDASH_SESSION_SECRET="replace-with-long-random-secret"
go run ./cmd/server
```

2. Open `http://localhost:8080`

3. Run quality gate:

```bash
task check
```

## Useful commands

- `task server` - run server
- `task build` - build binary
- `task webhooks:send CONFIG=cmd/webhookgenerator/sample.yaml` - send sample webhook stream
- `task events:publish ENDPOINT=... TOKEN=... SECRET=... SERVICE=billing-api ENV=staging` - publish one CI-style CDEvents delivery event
- `task events:backfill` - backfill legacy deployments into event store
- `task mocks` - regenerate test mocks using mockery

## Client-facing event publisher (CI/CD)

Use the event publisher CLI for CI pipelines:

```bash
go run ./cmd/eventpublisher \
  -endpoint "https://ddash.example.com" \
  -token "$DDASH_AUTH_TOKEN" \
  -secret "$DDASH_WEBHOOK_SECRET" \
  -type service.deployed \
  -service billing-api \
  -environment production \
  -artifact "pkg:generic/billing-api@${GITHUB_SHA}"
```

Supported `-type` values:
- `service.deployed`
- `service.upgraded`
- `service.rolledback`
- `service.removed`
- `service.published`

Library usage is available via `pkg/eventpublisher` for custom tooling.

## GitHub Actions

- CI workflow: `.github/workflows/ci.yml` (build + test on push/PR)
- Tag publish workflow: `.github/workflows/publish-cdevent-on-tag.yml`
  - triggers on tags matching `v*`
  - sends `service.published` event using `cmd/eventpublisher`
  - requires repository secrets:
    - `DDASH_ENDPOINT`
    - `DDASH_AUTH_TOKEN`
    - `DDASH_WEBHOOK_SECRET`
- Release workflow: `.github/workflows/release.yml`
  - triggers on tags matching `v*`
  - builds `cmd/server` binaries for linux/darwin/windows
  - creates GitHub release and uploads binaries
- E2E workflow: `.github/workflows/e2e-playwright.yml`
  - starts local DDash in dev mode
  - seeds deterministic org/user data and publishes sample events
  - runs Playwright smoke tests (`tests/e2e/smoke.spec.js`)

## Notes

- Event-store idempotency is organization-scoped using `(organization_id, event_source, event_id)`.
- UI state-changing endpoints are CSRF-protected; forms include `_csrf` and JSON POSTs send `X-CSRF-Token`.
