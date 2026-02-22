# GitLab ingestor

`apps/gitlabingestor` receives GitLab webhooks, converts selected events to CDEvents, and forwards them to DDash.

## Required env vars

- `GITLAB_WEBHOOK_SECRET`
- `DDASH_ENDPOINT`
- `GITHUB_APP_INGESTOR_SETUP_TOKEN`

## Optional env vars

- `GITLAB_INGESTOR_ADDR` (default `:8082`)
- `GITLAB_INGESTOR_PATH` (default `/webhooks/gitlab`)
- `GITLAB_INGESTOR_DEFAULT_ENV` (default `production`)
- `GITLAB_INGESTOR_SOURCE` (default `gitlab/webhook`)

## Event flow

1. GitLab sends webhook to ingestor (`X-Gitlab-Token` validated).
2. Ingestor extracts `project_id` and converts event payload to CDEvents.
3. Ingestor forwards CDEvents to DDash `/webhooks/gitlab-app` with headers:
   - `Authorization: Bearer <setup-token>`
   - `X-GitLab-Project-ID: <project_id>`
4. DDash resolves `project_id -> organization_id` and ingests the event.
