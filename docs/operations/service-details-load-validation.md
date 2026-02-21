# Service Details Load Validation

## Key query EXPLAIN checks

Validated on local database (`2026-02-21`):

1. Current state lookup:

```sql
EXPLAIN QUERY PLAN
SELECT latest_status, latest_event_ts_ms, drift_count, failed_streak
FROM service_current_state
WHERE organization_id = 1 AND service_name = 'orders'
LIMIT 1;
```

Result:

- `SEARCH service_current_state USING INDEX sqlite_autoindex_service_current_state_1`

2. Recent change links:

```sql
EXPLAIN QUERY PLAN
SELECT event_ts_ms, chain_id, environment, artifact_id, pipeline_run_id, run_url, actor_name
FROM service_change_links
WHERE organization_id = 1 AND service_name = 'orders'
ORDER BY event_ts_ms DESC
LIMIT 20;
```

Result:

- `SEARCH service_change_links USING INDEX idx_service_change_links_org_service_time`

3. 30-day stats aggregation:

```sql
EXPLAIN QUERY PLAN
SELECT COALESCE(SUM(deploy_success_count),0), COALESCE(SUM(deploy_failure_count),0), COALESCE(SUM(rollback_count),0)
FROM service_delivery_stats_daily
WHERE organization_id = 1 AND service_name = 'orders'
  AND day_utc >= date('now', '-30 day');
```

Result:

- `SEARCH service_delivery_stats_daily USING INDEX sqlite_autoindex_service_delivery_stats_daily_1`

## Rebuild and parity workflow

1. Rebuild projections:

```bash
task events:projections:rebuild DB=data/default ORG=0
```

2. Compare key services between legacy event-based views and projected detail cards.

3. Monitor DB timing logs:

```bash
DDASH_DB_TIMING=true go run ./cmd/server
```
