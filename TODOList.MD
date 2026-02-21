# Service Details Improvement Program

## Milestone Board

### M1: Foundation (Data + Contracts)
- [x] Define service-details KPI formulas and acceptance criteria.
- [x] Inventory current ingested CDEvent types and identify schema gaps.
- [x] Design normalized event taxonomy and mapping rules (required fields, dedupe keys, chain linkage).
- [x] Design projection schema (`service_current_state`, `service_env_state`, `service_delivery_stats_daily`, `service_change_links`).
- [x] Produce migration, backfill, and rollback strategy.

### M2: Backend Execution (Ingest + Projections + API)
- [x] Implement ingestion support for pipeline/source/environment/incident events.
- [x] Implement projection updaters on ingest (UPSERT) and reconciliation job.
- [x] Implement service-details read APIs/queries for cards, timeline, env matrix, and risky changes.
- [x] Add observability metrics (ingest lag, projection lag, rejection reasons, p95 queries).
- [~] Complete load/performance validation and index tuning.

### M3: UX + Rollout (UI + Validation + Launch)
- [x] Implement service details UI (cards, timeline, env matrix, audit context panel).
- [ ] Add feature-flag rollout controls per organization.
- [x] Backfill historical data for existing organizations and verify parity.
- [x] Expand test coverage (unit, integration, migration, E2E).
- [x] Publish event publishing contract + examples for CI providers.
- [~] Execute release checklist and post-release success review.

## Execution Log

- [x] Created milestone board and task decomposition.
- [x] Completed M1 task 1: KPI formulas and acceptance criteria (see `docs/plans/service-details-observability.md`).
- [x] Completed M1 task 2: event coverage and gap inventory (see `docs/plans/service-details-event-inventory.md`).
- [x] Completed M1 task 3: normalized event contract draft (see `docs/plans/service-details-event-contract.md`).
- [x] Completed M1 task 4: projection schema design and migration added (`internal/db/migrations/20260221170000_service_detail_projections.sql`).
- [x] Completed M1 task 5: migration/backfill/rollback plan (see `docs/plans/service-details-migration-backfill.md`).
- [x] Completed M2 task 1: ingestion support for broader event families (pipeline/change/artifact/incident prefixes).
- [x] Completed M2 task 2: projection updaters added on ingest (`internal/db/store.go`, `internal/db/queries.sql`) and rebuild command (`cmd/projectionsync`).
- [x] Completed M2 task 3: service-detail reads now use projection-backed metrics and links (`internal/adapters/sqlite/service_read_store.go`, `internal/app/services/service_read.go`).
- [x] Completed M3 UI implementation for delivery insights and risk/change links (`views/pages/service.templ`).
- [x] Added projection rebuild tooling (`cmd/projectionsync`, `task events:projections:rebuild`) and validation docs.
- [~] Added release checklist draft (`docs/plans/service-details-release-checklist.md`) and load-validation notes (`docs/operations/service-details-load-validation.md`).
