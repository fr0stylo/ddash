# GitHub App Ingestor

`apps/githubappingestor` receives GitHub webhooks, validates signatures, converts useful events to CDEvents, and forwards them to DDash.

The ingestor is stateless. DDash owns all installation setup state and mapping (`installation_id -> organization`).

## Run

```bash
task apps:githubappingestor:run
```

Setup happens in DDash UI:

- `GET /settings/integrations/github`
- callback handled by DDash: `GET /settings/integrations/github/callback`

## Required environment variables

- `GITHUB_WEBHOOK_SECRET`
- `DDASH_ENDPOINT`
- `GITHUB_APP_INGESTOR_SETUP_TOKEN`

## Optional environment variables

- `GITHUB_APP_INGESTOR_ADDR` (default `:8081`)
- `GITHUB_APP_INGESTOR_PATH` (default `/webhooks/github`)
- `GITHUB_APP_INGESTOR_DEFAULT_ENV` (default `production`)
- `GITHUB_APP_INGESTOR_SOURCE` (default `github/app`)

`DDASH_ENDPOINT` should be the DDash base URL (for example `https://ddash.example.com`).

## Installation mapping flow

1. Open DDash UI (`/settings/integrations/github`) and start installation.
2. Browser is redirected to GitHub App install URL with a short-lived state.
3. GitHub redirects to DDash callback (`/settings/integrations/github/callback`) with `installation_id` and `state`.
4. DDash persists mapping: `installation_id -> organization_id`.
5. Ingestor forwards webhook-derived CDEvents with `X-GitHub-Installation-ID`.
6. DDash resolves organization from mapping and ingests event.

## Event mappings

- `release` (published) -> `service.published`
- `deployment_status` -> `service.deployed` / `service.upgraded` / `service.removed`
- `workflow_run` -> `dev.cdevents.pipeline.run.*.0.3.0`
- `push` -> `dev.cdevents.change.pushed.0.3.0`
- `pull_request` -> `dev.cdevents.change.<action>.0.3.0`

Unsupported GitHub webhook events are acknowledged and ignored.
