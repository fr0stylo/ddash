# Features and Possibilities

This document lists current product capabilities and practical extension options.

## Current feature set

### Authentication and access

- GitHub OAuth login flow for dashboard access
- Session-based route protection (`RequireAuth`)

### Event ingestion

- CDEvents ingestion endpoint: `POST /webhooks/cdevents`
- Legacy endpoint `POST /webhooks/custom` returns `410 Gone`
- Validation pipeline:
  - bearer token authorization
  - HMAC signature check
  - CDEvents schema validation (SDK v0.5)
  - allowlist by event type

#### Sending events (quick snippets)

```bash
curl -X POST https://ddash.example.com/webhooks/cdevents \
  -H "Authorization: Bearer <auth-token>" \
  -H "X-Webhook-Signature: <hex-hmac-sha256-of-body>" \
  -H "Content-Type: application/json" \
  --data @event.json
```

```javascript
const body = await fs.readFile('./event.json', 'utf8')
const signature = crypto.createHmac('sha256', process.env.WEBHOOK_SECRET).update(body).digest('hex')
await fetch('https://ddash.example.com/webhooks/cdevents', {
  method: 'POST',
  headers: {
    Authorization: 'Bearer ' + process.env.AUTH_TOKEN,
    'X-Webhook-Signature': signature,
    'Content-Type': 'application/json',
  },
  body,
})
```

```python
body = open('event.json', 'rb').read()
signature = hmac.new(os.environ['WEBHOOK_SECRET'].encode(), body, hashlib.sha256).hexdigest()
requests.post(
    'https://ddash.example.com/webhooks/cdevents',
    headers={
        'Authorization': 'Bearer ' + os.environ['AUTH_TOKEN'],
        'X-Webhook-Signature': signature,
        'Content-Type': 'application/json',
    },
    data=body,
)
```

### Deployment visibility

- Home view with service cards/table projections from event store
- Deployments view with projected deployment timeline rows
- Service details view with:
  - latest integration info
  - per-environment latest deploy rows
  - deployment history

### Metadata management

- Organization-level required metadata fields
- `filterable` flag per required field for UI filtering
- Per-service metadata editing from service details page
- Missing metadata count and severity badge behavior

### Filtering and ordering

- Environment and service filtering for deployment/service views
- Metadata tag filtering in home/deployments
- Configurable environment priority ordering with lexical fallback

### Multi-organization management

- Organization management page (`/organizations`) with create/switch/rename/enable-disable/delete
- Active organization context in authenticated session
- Org-scoped reads/writes for services, deployments, settings, and metadata
- Safety guards for org lifecycle actions (e.g., last-organization delete protection)

### Operational tooling

- Backfill command for legacy data into event store: `task apps:eventbackfill:run`
- Sample event seeding helper: `task events:seed`
- Webhook generator helper: `task apps:webhookgenerator:run`
- Unified quality gate: `task check` (fmt + vet + lint + test)

## Architecture capabilities already enabled

- App layer decoupled from sqlc types (`internal/app` uses only app DTOs)
- Pluggable storage adapters via app ports
- SQLite adapter isolated under `apps/ddash/internal/adapters/sqlite`
- Ingestion port uses app-level event record model (transport/database independent)

## Possibilities (next iterations)

### 0) Sync status visibility policy

- Add per-organization setting to hide sync status indicators in service/deployment dashboards
- Keep status computation in backend but suppress status presentation when disabled
- Target users with fully automated deployment reconciliation

### 1) Additional storage backends

- Implement app ports for:
  - gRPC proxy backend
  - ClickHouse read model backend
  - split write/read stores for scale

### 2) Split ingestion service

- Move webhook ingestion into standalone service while keeping same app ingestion contract
- Publish accepted events to queue/stream for downstream projectors

### 3) Multi-organization UX

- Organization management UI (create/select/switch)
- Strong tenant scoping in all projections and settings pages

### 4) Advanced observability

- Metrics for ingestion acceptance/rejection rates
- Projection latency and staleness tracking
- Structured audit trail for settings and metadata changes

### 5) Product-level enhancements

- Deployment diff and rollout health indicators
- Team/owner drilldowns using metadata
- Alerting hooks on failed or stuck deployment flows

## Suggested prioritization

1. Sync status visibility policy
2. Standalone ingestion service
3. Alternative read backend (ClickHouse/gRPC)
4. Observability and alerting
