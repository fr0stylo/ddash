# Plan: Service Details Observability

## Goal

Provide high-signal service details that answer:

- What is deployed now in each environment?
- Is delivery healthy and improving?
- What changed and who changed it?
- Where are promotion/risk bottlenecks?

## KPI Definitions (M1 Task 1)

Default time windows:

- Operational window: last 7 days
- Trend window: last 30 days
- Baseline comparison: previous 30 days

All KPIs are organization-scoped and service-scoped unless stated otherwise.

### 1) Last Deploy Status

- Source: latest delivery event per environment/service.
- Values: `success`, `warning`, `error`, `unknown`.
- Formula: map latest event type in env to status.
- Display: top summary card + env matrix cell status.

### 2) Deployment Frequency

- Definition: successful deployments in window.
- Formula: count(delivery events where status=success, ts in window).
- Display: total in 7d, average/day in 30d, sparkline by day.

### 3) Change Failure Rate (CFR)

- Definition: percentage of deployments resulting in failure signal.
- Numerator: deployment chains that include rollback/removal/failure outcome.
- Denominator: total deployment chains started in window.
- Formula: `CFR = numerator / denominator * 100`.
- Display: 30d percentage with up/down trend vs previous 30d.

### 4) MTTR

- Definition: median recovery time from failure to next success for same service+env.
- Formula: median(success_ts - failure_ts) for paired failure->success incidents in window.
- Display: 30d median (hours/minutes).

### 5) Promotion Lead Time

- Definition: time from first successful deploy in lower env to successful prod deploy for same artifact/chain.
- Formula: `prod_success_ts - first_non_prod_success_ts` for matched artifact/chain.
- Display: p50/p95 in 30d.

### 6) Environment Drift

- Definition: environments running different artifact versions for same service.
- Formula: set of distinct artifact IDs across envs; drift if cardinality > 1.
- Display: drift badge + list of mismatching env artifacts.

### 7) Staleness

- Definition: time since last successful deploy per env/service.
- Formula: `now - last_success_ts`.
- Thresholds (default):
  - healthy: <= 7d
  - warning: > 7d and <= 30d
  - stale: > 30d

### 8) Risk Signal Count

- Definition: count of risky delivery patterns in 30d.
- Included signals:
  - repeated failures (>= 3 consecutive failures)
  - rollback after latest deploy
  - high deploy churn (> N/day threshold, org-configurable)
- Display: count + expandable event list.

## Service Detail Panels

### Summary cards

- Last deploy status/time
- Deployment frequency (7d/30d)
- CFR 30d
- MTTR 30d

### Timeline

- Ordered event stream with deploy, upgrade, rollback, failure markers.
- Must include actor/repo/commit/pipeline run when present.

### Environment matrix

- Rows: environments
- Columns: current artifact, status, last update, staleness age.

### Risk and audit context

- Risk event list with chain links.
- Linked source context: commit, PR, release, pipeline URL.

## Acceptance Criteria (M1 Task 1)

- KPI numbers are deterministic for same input set and timezone (`UTC`).
- KPI cards are computed from projection tables, not ad-hoc full scans.
- Service page load p95 target: <= 300ms for org with 1M+ events (warm cache).
- Timeline supports pagination and does not regress p95 over 500ms for first page.
- Empty-state behavior is explicit (shows `No delivery data yet` with onboarding link).
- All values are org-isolated (no cross-tenant leakage).

## Next M1 Tasks

- Task 2: event inventory + schema gap report.
- Task 3: normalized event taxonomy and field contract.
- Task 4: projection schema and index design.
- Task 5: migration/backfill/rollback plan.

## M1 Task 2 (Started): Current Event Coverage and Gaps

Current accepted delivery event families in ingestion layer:

- `dev.cdevents.service.deployed.*`
- `dev.cdevents.service.upgraded.*`
- `dev.cdevents.service.rolledback.*`
- `dev.cdevents.service.removed.*`
- `dev.cdevents.service.published.*`
- `dev.cdevents.environment.created.*`
- `dev.cdevents.environment.modified.*`
- `dev.cdevents.environment.deleted.*`

Current strengths:

- Good deployment lifecycle visibility.
- Supports environment-level context extraction.
- Has chain support via `chain_id` for cross-event linkage.

Gaps blocking richer service details:

- No pipeline run/stage outcomes (build/test/deploy stage failures/successes).
- No source-change context (commit/PR/merge/tag linkage).
- No actor/action identity beyond what may exist in raw payload.
- No incident/alert events for correlation with change risk.
- Limited first-class extraction of fields needed for DORA-like metrics (currently mainly derived from delivery events).

Planned next output for Task 2:

- Event inventory table by family with fields currently extracted vs needed fields.
- Priority mapping of missing fields into projection model.
