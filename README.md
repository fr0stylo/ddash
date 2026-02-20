# DDash

DDash is a deployment dashboard for tracking service deployments across environments.

It ingests CDEvents webhooks, stores immutable events in SQLite, and renders projected views for services and deployments.

## Documentation

- Architecture: `docs/architecture.md`
- Features and possibilities: `docs/features-and-possibilities.md`
- Feature catalog (build intake): `docs/feature-catalog.md`
- Next feature plan: `docs/plans/sync-status-visibility.md`

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
- `task events:backfill` - backfill legacy deployments into event store

## Notes

- Event-store idempotency is organization-scoped using `(organization_id, event_source, event_id)`.
- UI state-changing endpoints are CSRF-protected; forms include `_csrf` and JSON POSTs send `X-CSRF-Token`.
