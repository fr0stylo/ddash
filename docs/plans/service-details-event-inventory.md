# Service Details Event Inventory

## Current Coverage (Implemented)

Accepted and persisted event families:

| Family | Type Pattern | Used Today | Notes |
|---|---|---|---|
| Service deployed | `dev.cdevents.service.deployed.*` | Yes | Primary success signal |
| Service upgraded | `dev.cdevents.service.upgraded.*` | Yes | Treated as success |
| Service rolled back | `dev.cdevents.service.rolledback.*` | Yes | Risk/failure signal |
| Service removed | `dev.cdevents.service.removed.*` | Yes | Out-of-sync/failure signal |
| Service published | `dev.cdevents.service.published.*` | Yes | Delivery success-like signal |
| Environment created | `dev.cdevents.environment.created.*` | Partial | Stored, limited UI usage |
| Environment modified | `dev.cdevents.environment.modified.*` | Partial | Stored, limited UI usage |
| Environment deleted | `dev.cdevents.environment.deleted.*` | Partial | Stored, limited UI usage |

Persisted extracted fields in `event_store` today:

- `organization_id`
- `event_id`, `event_type`, `event_source`
- `event_timestamp`, `event_ts_ms`
- `subject_id`, `subject_source`, `subject_type`
- `chain_id`
- `raw_event_json`

## Gap Analysis

## Priority A (Needed for core service details KPIs)

| Domain | Needed Event Family | Gap |
|---|---|---|
| Pipeline execution | pipeline run/stage start/success/failure | Not ingested |
| Build/test outcome | build/test check pass/fail events | Not ingested |
| Source linkage | commit/PR/merge/tag/release events | Not ingested |
| Actor attribution | initiator identity and actor type | Not normalized |

## Priority B (High-value correlations)

| Domain | Needed Event Family | Gap |
|---|---|---|
| Incidents/alerts | incident opened/resolved, alert fired/resolved | Not ingested |
| Infra change | environment or infra rollout/change signals | Only partial env events |

## Priority C (Future enhancement)

| Domain | Needed Event Family | Gap |
|---|---|---|
| SLO/burn-rate | SLI/SLO breach and recovery events | Not ingested |
| Deployment verification | canary/analysis pass-fail events | Not ingested |

## Required Normalized Fields (for new families)

- Identity: `organization_id`, `service_name`, `environment`
- Time/order: `event_ts_ms`, `chain_id`, `correlation_id`
- Change linkage: `artifact_id`, `commit_sha`, `pr_number`, `tag`, `release_id`
- Pipeline linkage: `pipeline_id`, `run_id`, `stage`, `stage_status`, `run_url`
- Actor: `actor_id`, `actor_name`, `actor_type`
- Incident: `incident_id`, `severity`, `incident_status`

## Recommendation

Implement in this order:

1. Pipeline + source/linkage families
2. Incident/alert families
3. Infra and SLO-style signals
