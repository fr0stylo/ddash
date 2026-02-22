# SQLite Performance Baseline

Use this checklist to verify DDash SQLite behavior and collect an apples-to-apples baseline before tuning.

## Workload shape snapshot

Run:

```bash
task apps:dbshape:run DB=data/default ORG=0 WINDOW_DAYS=30 MAX_CONCURRENT_USERS=0
```

Current local sample (2026-02-21):

- `event_store rows`: `0`
- `events/day`: `n/a` (no organization data in `data/default.sqlite`)
- `dashboard window`: `30d`
- `max concurrent users`: unknown in local DB snapshot

## Top 5 query shapes to monitor

The UI relies most heavily on:

1. `ListDeploymentsFromEvents`
2. `ListServiceInstancesFromEvents`
3. `ListServiceInstancesByEnvFromEvents`
4. `GetServiceLatestFromEvents`
5. `ListDeploymentHistoryByServiceFromEvents`

## EXPLAIN verification

After adding `event_ts_ms` and service timeline index, key paths should be index-backed.

Local check examples:

```sql
EXPLAIN QUERY PLAN
SELECT ...
FROM event_store es
WHERE es.organization_id = 1
  AND es.subject_type = 'service'
ORDER BY es.event_ts_ms DESC, es.seq DESC;
```

Expected core signal:

- `SEARCH es USING INDEX idx_event_store_org_subjecttype_time`

The CTE/window-based service list still uses temp b-trees for ranking; that is expected for now and is the strongest candidate for a projection table optimization.

## DB timing telemetry

- Enable DB query latency logs (top queries by p95):

```bash
DDASH_DB_TIMING=true go run ./apps/ddash
```

- Server logs emit `db_query_latency` entries with:
  - `query`
  - `count`
  - `p50_ms`
  - `p95_ms`
  - `max_ms`

Use these before/after changes to validate improvements.
