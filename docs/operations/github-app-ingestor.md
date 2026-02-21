# GitHub App Ingestor

`cmd/githubappingestor` receives GitHub webhooks, validates signatures, converts useful events to CDEvents, and forwards them to DDash.

It supports per-installation mapping (Option B onboarding flow): each GitHub App installation can map to a different DDash organization token/secret.

## Run

```bash
task githubapp:run
```

Open integrated setup UI:

```text
http://localhost:8081/setup?setup_token=<GITHUB_APP_INGESTOR_SETUP_TOKEN>
```

When DDash unified UI is configured, DDash calls ingestor APIs directly:

- `POST /api/setup/start`
- `GET /api/mappings?org_id=<id>`
- `POST /api/mappings/delete` (with `installation_id`, optional `organization_id`)

## Required environment variables

- `GITHUB_WEBHOOK_SECRET`
- `DDASH_ENDPOINT`
- `DDASH_AUTH_TOKEN`
- `DDASH_WEBHOOK_SECRET`

## Optional environment variables

- `GITHUB_APP_INGESTOR_ADDR` (default `:8081`)
- `GITHUB_APP_INGESTOR_PATH` (default `/webhooks/github`)
- `GITHUB_APP_INGESTOR_DEFAULT_ENV` (default `production`)
- `GITHUB_APP_INGESTOR_SOURCE` (default `github/app`)
- `GITHUB_APP_INGESTOR_DB_PATH` (default `data/githubapp-ingestor`)
- `GITHUB_APP_INSTALL_URL` (GitHub app installation URL used by setup flow)
- `GITHUB_APP_INGESTOR_SETUP_START_PATH` (default `/setup/start`)
- `GITHUB_APP_INGESTOR_SETUP_CALLBACK_PATH` (default `/setup/callback`)
- `GITHUB_APP_INGESTOR_SETUP_UI_PATH` (default `/setup`)
- `GITHUB_APP_INGESTOR_SETUP_DELETE_PATH` (default `/setup/mappings/delete`)
- `GITHUB_APP_INGESTOR_SETUP_TOKEN` (optional protection token for setup start endpoint)

## Installation mapping flow (Option B)

1. Open DDash UI (`/settings/integrations/github`) and start installation (recommended), or use ingestor setup UI as fallback.

2. Browser is redirected to GitHub App install URL with a short-lived state.
3. After install, GitHub redirects to callback (`/setup/callback`) with `installation_id` and `state`.
4. Ingestor persists mapping: `installation_id -> DDash credentials`.
5. Future webhook events route automatically using that installation mapping.

You can revoke mappings from the same setup UI.

DDash uses organization-scoped list/delete calls so each DDash organization only manages its own mappings.

If mapping is missing, ingestor falls back to default `DDASH_*` credentials when present.

## Event mappings

- `release` (published) -> `service.published`
- `deployment_status` -> `service.deployed` / `service.upgraded` / `service.removed`
- `workflow_run` -> `dev.cdevents.pipeline.run.*.0.3.0`
- `push` -> `dev.cdevents.change.pushed.0.3.0`
- `pull_request` -> `dev.cdevents.change.<action>.0.3.0`

Unsupported GitHub webhook events are acknowledged and ignored.
