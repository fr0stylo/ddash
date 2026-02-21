# Service Details Migrations and Backfill Plan

## Scope

Introduce projection tables that support service-detail queries without scanning full event history.

## Projection tables

- `service_current_state`
  - one row per `(organization_id, service_name)`
- `service_env_state`
  - one row per `(organization_id, service_name, environment)`
- `service_delivery_stats_daily`
  - one row per `(organization_id, service_name, day_utc)`
- `service_change_links`
  - one row per `(organization_id, service_name, event_seq)`

## Rollout sequence

1. Deploy schema migration (tables + indexes).
2. Deploy writer logic (on-ingest UPSERT updates).
3. Run historical backfill by organization in batches.
4. Enable read path behind feature flag.
5. Verify parity and latency targets.
6. Flip default read path to projections.

## Backfill strategy

- Process by `organization_id`, ordered by `seq`.
- Use small commits (e.g., 1000-5000 events/tx).
- Keep resume cursor (`organization_id`, `last_seq`) for restart safety.
- Idempotent UPSERT semantics so reruns are safe.

## Verification checks

- Row count sanity:
  - projection service count equals distinct service count from event store.
- Spot checks:
  - latest deploy timestamp by service matches raw-event query.
  - environment artifact in projection equals latest raw event.
- Performance checks:
  - service details endpoint p95 <= target with production-like volume.

## Rollback plan

- Keep raw event ingestion untouched.
- If projection read path regresses, disable feature flag and revert reads to raw queries.
- Projection tables can remain for debugging; drop in later cleanup migration if needed.
