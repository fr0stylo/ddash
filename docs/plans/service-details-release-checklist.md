# Service Details Release Checklist

## Pre-release

- [ ] Run migrations in staging and confirm projection tables exist.
- [ ] Run projection rebuild command and verify row counts are non-zero for active orgs:
  - `task events:projections:rebuild DB=<db-path> ORG=0`
- [ ] Validate service details pages for at least 3 representative services.
- [ ] Verify no cross-org data leaks using two test organizations.

## Performance

- [ ] Confirm EXPLAIN for key projection queries uses indexes.
- [ ] Validate endpoint p95 for service details under expected load.
- [ ] Confirm ingest p95 does not regress beyond agreed budget.

## Observability

- [ ] Confirm traces include `request.id`, `http.route`, `enduser.id`, `ddash.org_id` where expected.
- [ ] Confirm DB spans exist for service detail reads and ingestion writes.
- [ ] Confirm logs include `trace_id` and `span_id` for request-scoped logs.

## Functional checks

- [ ] Delivery insight cards show expected values for known event sequences.
- [ ] Risk/change links panel shows recent data and opens run URLs.
- [ ] Metadata editing and existing service details behavior remain intact.

## Rollout

- [ ] Start with canary org(s), monitor 24-48h.
- [ ] Expand rollout to all orgs.
- [ ] Record post-release summary and follow-up actions.
