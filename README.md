# DDash

DDash is a deployment dashboard for tracking service deployments across environments.

It ingests CDEvents webhooks, stores immutable events in SQLite, and renders projected views for services and deployments.

## Documentation

- Architecture: `docs/architecture.md`
- Features and possibilities: `docs/features-and-possibilities.md`
- Feature catalog (build intake): `docs/feature-catalog.md`
- Next feature plan: `docs/plans/sync-status-visibility.md`
- Playwright checklist: `docs/testing/playwright-checklist.md`
- Deployment guide: `docs/operations/deployment.md`
- MVP final checklist: `docs/operations/mvp-final-checklist.md`

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
- `task events:shape DB=data/default ORG=0 WINDOW_DAYS=30` - print event-store workload shape snapshot
- `task events:projections:rebuild DB=data/default ORG=0` - rebuild service detail projection tables from event store
- `task load:server`, `task load:seed`, `task load:test:ingest|read|mixed`, `task load:stop` - run local load-test setup (k6)
- `task load:all` - run complete local load-test flow end-to-end
- `task githubapp:run` - run GitHub webhook ingestor that converts useful GitHub events to CDEvents and forwards to DDash
- `task mocks` - regenerate test mocks using mockery
- `task mvp:check:quick` - run automated MVP preflight checks
- `task mvp:check:full` - run automated MVP checks including Playwright E2E
- `task mvp:final:tasks` - print final manual MVP checklist path

GitHub ingestor setup details: `docs/operations/github-app-ingestor.md`
GitHub ingestor integrated setup UI (default): `http://localhost:8081/setup`

To enable unified DDash-side integration UI (`/settings/integrations/github`), set:
- `GITHUB_APP_INGESTOR_URL` (e.g. `http://localhost:8081`)
- `GITHUB_APP_INGESTOR_SETUP_TOKEN`
- `DDASH_PUBLIC_URL` (public base URL of this DDash instance)

## Deployment (Helm)

Helm chart path: `deploy/helm/ddash`.

Render defaults:

```bash
helm template ddash ./deploy/helm/ddash
```

Use StatefulSets (with `volumeClaimTemplates`) instead of Deployments:

```bash
helm upgrade --install ddash ./deploy/helm/ddash \
  --set ddash.workload.kind=StatefulSet \
  --set githubIngestor.workload.kind=StatefulSet
```

Enable Gateway API HTTPRoutes:

```bash
helm upgrade --install ddash ./deploy/helm/ddash \
  --set gatewayAPI.enabled=true \
  --set gatewayAPI.ddash.enabled=true \
  --set gatewayAPI.githubIngestor.enabled=true
```

Ingress and Gateway API can be configured independently via `values.yaml`.

Published OCI chart (GHCR):

```bash
helm pull oci://ghcr.io/<owner>/charts/ddash --version <version>
helm upgrade --install ddash oci://ghcr.io/<owner>/charts/ddash --version <version>
```

Published container images (GHCR):

- `ghcr.io/<owner>/ddash-server:<version>`
- `ghcr.io/<owner>/ddash-githubappingestor:<version>`

Images are published as multi-arch manifests for:

- `linux/amd64`
- `linux/arm64`
- `linux/arm/v7`

## OpenTelemetry setup

DDash ships with built-in OpenTelemetry tracing for Echo HTTP requests and database query spans.

Enable OTEL with either:

- `DDASH_OTEL_ENABLED=true`
- or setting `OTEL_EXPORTER_OTLP_ENDPOINT` (auto-enables when present)

Example:

```bash
export DDASH_OTEL_ENABLED=true
export OTEL_EXPORTER_OTLP_ENDPOINT="http://localhost:4318"
export OTEL_EXPORTER_OTLP_PROTOCOL="http/protobuf"
export OTEL_SERVICE_NAME="ddash"
export OTEL_SERVICE_VERSION="dev"
export DDASH_OTEL_SAMPLING_RATIO="0.2"
go run ./cmd/server
```

Notes:

- Propagation uses W3C Trace Context + Baggage.
- HTTP server tracing is enabled via Echo middleware with request span enrichment (`request.id`, `http.route`, `enduser.id`, `ddash.org_id`).
- DB spans are emitted per sqlc query using query name labels and include user/org attributes when available.
- Logs include `trace_id` and `span_id` when called with request context.
- Sampling is explicit and parent-based via `DDASH_OTEL_SAMPLING_RATIO` (`0.0` to `1.0`, default `1.0`).
- Noisy endpoints/assets are skipped from tracing (health/static files/GitHub callback).
- Default outbound HTTP client is OTEL-instrumented for end-to-end traces.

Docker Compose includes an observability stack out of the box:

- Jaeger: `http://localhost:16686`
- Prometheus: `http://localhost:9090`
- Grafana: `http://localhost:3000`

In compose mode, DDash OTEL metrics are pushed to Prometheus via OTEL Collector remote-write.

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
- `environment.created`
- `environment.modified`
- `environment.deleted`
- any fully-qualified accepted custom type prefix:
  - `dev.cdevents.pipeline.*`
  - `dev.cdevents.change.*`
  - `dev.cdevents.artifact.*`
  - `dev.cdevents.incident.*`

Additional optional flags for advanced/custom events:
- `-subject-id`, `-subject-type`
- `-chain-id`
- `-actor`
- `-pipeline-run`, `-pipeline-url`

Task helpers with ready-made examples:
- `task events:publish:examples`
- `task events:publish:example:service ENDPOINT=... TOKEN=... SECRET=...`
- `task events:publish:example:environment ENDPOINT=... TOKEN=... SECRET=...`
- `task events:publish:example:pipeline ENDPOINT=... TOKEN=... SECRET=...`
- `task events:publish:example:change ENDPOINT=... TOKEN=... SECRET=...`
- `task events:publish:example:artifact ENDPOINT=... TOKEN=... SECRET=...`
- `task events:publish:example:incident ENDPOINT=... TOKEN=... SECRET=...`

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
- GHCR publish workflow: `.github/workflows/publish-ghcr.yml`
  - triggers on tags matching `v*`
  - builds and pushes multi-arch images to GHCR:
    - `ghcr.io/<owner>/ddash-server`
    - `ghcr.io/<owner>/ddash-githubappingestor`
  - packages and pushes Helm chart as OCI artifact:
    - `oci://ghcr.io/<owner>/charts/ddash`
- E2E workflow: `.github/workflows/e2e-playwright.yml`
  - starts local DDash in dev mode
  - seeds deterministic org/user data and publishes sample events
  - runs Playwright smoke tests (`tests/e2e/smoke.spec.js`)

## Notes

- Event-store idempotency is organization-scoped using `(organization_id, event_source, event_id)`.
- UI state-changing endpoints are CSRF-protected; forms include `_csrf` and JSON POSTs send `X-CSRF-Token`.
