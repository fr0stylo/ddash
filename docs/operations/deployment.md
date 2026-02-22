# Deployment guide

This repo supports three common deployment paths:

1. single-binary/manual (Go run/build),
2. Docker Compose,
3. Kubernetes via Helm.

## 1) Single binary/manual

Run DDash:

```bash
export DDASH_SESSION_SECRET="replace-with-long-random-secret"
go run ./apps/ddash
```

Run GitHub App ingestor in another shell:

```bash
export GITHUB_WEBHOOK_SECRET="replace-me"
export DDASH_ENDPOINT="http://localhost:8080"
export GITHUB_APP_INGESTOR_SETUP_TOKEN="replace-me"
go run ./apps/githubappingestor
```

## 2) Docker Compose

Simple one-command deploy script:

```bash
cp .env.deploy.example .env.deploy
# edit .env.deploy values
./scripts/deploy-all-services.sh
```

Options:

- `--no-build` restart/recreate without rebuilding images
- `--env-file /path/to/.env` use custom deployment env file

Use the included stack:

```bash
docker compose up --build
```

Services:

- DDash: `http://localhost:8080`
- GitHub ingestor: `http://localhost:8081`
- Grafana: `http://localhost:3000`
- Prometheus: `http://localhost:9090`
- Jaeger UI: `http://localhost:16686`

Main env vars to set before running:

- `DDASH_SESSION_SECRET`
- `GITHUB_APP_INGESTOR_SETUP_TOKEN`
- `GITHUB_WEBHOOK_SECRET`
- `DDASH_PUBLIC_URL` (public DDash URL used by setup flow)
- `GITHUB_APP_INSTALL_URL` (GitHub App install URL)

Observability defaults in compose:

- DDash exports OTLP traces/metrics to `otel-collector`.
- Collector forwards traces to Jaeger and sends metrics to Prometheus (`remote write`) and also exposes `/metrics` for scrape mode.
- Grafana is pre-provisioned with Prometheus and Jaeger datasources.

To also print OTel metrics to DDash container logs:

```bash
DDASH_OTEL_METRICS_CONSOLE=true docker compose up --build
```

## 3) Helm (Kubernetes)

Chart path:

```text
deploy/helm/ddash
```

Render chart:

```bash
helm template ddash ./deploy/helm/ddash
```

Install/upgrade:

```bash
helm upgrade --install ddash ./deploy/helm/ddash -n ddash --create-namespace
```

### StatefulSet mode

Both services default to Deployments. To switch to StatefulSets:

```bash
helm upgrade --install ddash ./deploy/helm/ddash \
  --set ddash.workload.kind=StatefulSet \
  --set githubIngestor.workload.kind=StatefulSet
```

In StatefulSet mode, persistence uses `volumeClaimTemplates` and standalone PVC templates are skipped.

### Gateway API mode

The chart can create HTTPRoutes for DDash and ingestor:

```bash
helm upgrade --install ddash ./deploy/helm/ddash \
  --set gatewayAPI.enabled=true \
  --set gatewayAPI.ddash.enabled=true \
  --set gatewayAPI.githubIngestor.enabled=true
```

Configure hostnames/parentRefs/path prefixes in `values.yaml`.

### Ingress mode

Ingress resources are independent from Gateway API and can be enabled separately:

- `ingress.ddash.enabled=true`
- `ingress.githubIngestor.enabled=true`

## Unified GitHub setup flow

DDash unified UI is available at:

```text
/settings/integrations/github
```

Required DDash env vars:

- `GITHUB_APP_INSTALL_URL`
- `GITHUB_APP_INGESTOR_SETUP_TOKEN`
- `DDASH_PUBLIC_URL`

This flow creates per-organization install intents and stores installation mappings in DDash state.
