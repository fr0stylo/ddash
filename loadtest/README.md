# Load Test Setup

This folder contains a `k6` setup for DDash traffic capacity checks.

## Prerequisites

- Install `k6` locally.
- Keep port `19090` free (or override `PORT`).

## Quick start

1. Start isolated load-test server:

```bash
task load:server
```

2. Seed base org/user and sample events:

```bash
task load:seed
```

3. Run scenarios:

```bash
task load:test:ingest
task load:test:read
task load:test:mixed
```

4. Stop server:

```bash
task load:stop
```

## Scenarios

- `loadtest/ingest.js`
  - ramping ingestion (webhook write path)
- `loadtest/read.js`
  - authenticated dashboard/read mix
- `loadtest/mixed.js`
  - concurrent ingestion + read pressure

## Key environment overrides

- `BASE_URL` (default `http://localhost:19090`)
- `AUTH_TOKEN` (default `loadtest-token-01`)
- `WEBHOOK_SECRET` (default `loadtest-secret-01`)

Scenario-specific knobs are available in each script via env variables
(`INGEST_*`, `READ_*`, `MIXED_*`).

Notes:

- By default, ingest scenario sends schema-valid delivery/environment style payloads only.
- To include custom prefixed event types in ingest mix, set:

```bash
INGEST_INCLUDE_CUSTOM_TYPES=true task load:test:ingest
```

Custom prefixed events may be rejected by current CDEvents schema validation, which is useful when you want to measure rejection behavior.
