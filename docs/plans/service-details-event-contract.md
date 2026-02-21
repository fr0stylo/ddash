# Service Details Event Contract (Normalized)

## Goal

Define one normalized envelope for all ingested events so projections stay simple and queryable.

## Normalized Envelope

Required fields for all ingested events:

- `organization_id` (int64)
- `event_id` (string)
- `event_type` (string)
- `event_source` (string)
- `event_ts_ms` (int64, UTC epoch millis)
- `subject_type` (string)
- `subject_id` (string)

Recommended common fields:

- `chain_id` (string)
- `correlation_id` (string)
- `actor_id` (string)
- `actor_name` (string)
- `actor_type` (`user`/`service`/`system`)

Service delivery common fields:

- `service_name` (string)
- `environment` (string)
- `artifact_id` (string)
- `delivery_status` (`success`/`failure`/`warning`/`unknown`)

Pipeline common fields:

- `pipeline_id` (string)
- `pipeline_run_id` (string)
- `pipeline_stage` (string)
- `pipeline_stage_status` (`started`/`success`/`failed`/`cancelled`)
- `pipeline_url` (string)

Source/change common fields:

- `repo` (string)
- `commit_sha` (string)
- `pr_number` (string)
- `tag` (string)
- `release_id` (string)

Incident common fields:

- `incident_id` (string)
- `severity` (`sev1`/`sev2`/`sev3`/`sev4`)
- `incident_status` (`open`/`acknowledged`/`resolved`)

## Dedupe and linkage rules

- Global dedupe key: `(organization_id, event_source, event_id)`
- Logical operation grouping: `chain_id` (delivery chain)
- Cross-system correlation: `correlation_id`

## Event family mapping priorities

Phase 1:

- `service.deployed`, `service.upgraded`, `service.rolledback`, `service.removed`, `service.published`
- pipeline run + stage pass/fail events
- git/source merge/tag/release events

Phase 2:

- incident/alert events
- infra/environment detail events

## Rejection policy

- Missing required envelope fields: reject with `invalid_payload`
- Unsupported family: reject with `unsupported_type`
- Signature/auth mismatch: reject with `invalid_signature` / `invalid_auth`
