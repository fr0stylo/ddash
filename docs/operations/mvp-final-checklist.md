# MVP final checklist

Use this checklist to sign off DDash MVP readiness.

## A. Build and quality gates

- [ ] `task generate` passes on a clean checkout.
- [ ] `go build ./...` passes.
- [ ] `go test ./...` passes.
- [ ] `helm lint ./deploy/helm/ddash` passes.
- [ ] `helm template ddash ./deploy/helm/ddash` renders without errors.

## B. Local product smoke

- [ ] `task e2e` passes (Playwright smoke flow).
- [ ] UI loads and supports organization switch without errors.
- [ ] Service and deployment HTMX fragments work and return `X-Fragment-Cache` hit/miss headers.

## C. GitHub App integration smoke

- [ ] Start ingestor: `task githubapp:run`.
- [ ] DDash GitHub integration page (`/settings/integrations/github`) opens and lists mappings.
- [ ] Start install flow from DDash UI and complete callback.
- [ ] Mapping appears only for active organization.
- [ ] Delete mapping from DDash UI removes only that organization's mapping.

## D. Observability

- [ ] Traces export when `OTEL_EXPORTER_OTLP_ENDPOINT` is set.
- [ ] Common OTEL headers propagate (`OTEL_EXPORTER_OTLP_HEADERS`).
- [ ] Signal-specific headers override/add (`OTEL_EXPORTER_OTLP_TRACES_HEADERS`, `OTEL_EXPORTER_OTLP_METRICS_HEADERS`).
- [ ] Console metrics output appears when `DDASH_OTEL_METRICS_CONSOLE=true`.
- [ ] Fragment renderer metrics visible (`ddash.fragment_renderer.*`).

## E. Packaging and publish

- [ ] `Dockerfile.ddash` builds locally.
- [ ] `Dockerfile.githubappingestor` builds locally.
- [ ] GHCR publish workflow succeeds on a test tag (`.github/workflows/publish-ghcr.yml`).
- [ ] Multi-arch images are present in GHCR (`linux/amd64`, `linux/arm64`, `linux/arm/v7`).
- [ ] Helm chart OCI artifact is published to `oci://ghcr.io/<owner>/charts/ddash`.

## F. Deployment readiness

- [ ] Helm install works on a fresh namespace.
- [ ] StatefulSet mode renders and deploys.
- [ ] Gateway API mode renders and deploys.
- [ ] Ingress mode renders and deploys.

## G. Security and operations

- [ ] Production envs set secure secrets (`DDASH_SESSION_SECRET`, webhook secrets, auth tokens).
- [ ] Cookie security is configured for production (`DDASH_SECURE_COOKIE=true` behind TLS).
- [ ] Token/secret rotation process is documented.
- [ ] SQLite backup/restore process is documented and tested.

## H. Release decision

- [ ] Release notes drafted (major behavior changes + migration notes).
- [ ] Tag strategy confirmed (`vX.Y.Z`).
- [ ] Rollback path documented (previous image/chart version pin).
