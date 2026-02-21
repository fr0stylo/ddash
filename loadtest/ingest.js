import { sleep } from 'k6'
import { Trend } from 'k6/metrics'

import { buildServiceEvent, postWebhook } from './common.js'

const ingestLatency = new Trend('ingest_latency_ms')

export const options = {
  scenarios: {
    ingest_step: {
      executor: 'ramping-arrival-rate',
      startRate: Number(__ENV.INGEST_START_RPS || 10),
      timeUnit: '1s',
      preAllocatedVUs: Number(__ENV.PRE_VUS || 20),
      maxVUs: Number(__ENV.MAX_VUS || 200),
      stages: [
        { target: Number(__ENV.INGEST_RPS_1 || 50), duration: __ENV.INGEST_STAGE_1 || '2m' },
        { target: Number(__ENV.INGEST_RPS_2 || 100), duration: __ENV.INGEST_STAGE_2 || '3m' },
        { target: Number(__ENV.INGEST_RPS_3 || 200), duration: __ENV.INGEST_STAGE_3 || '3m' },
      ],
      exec: 'runIngest',
    },
  },
  thresholds: {
    http_req_failed: ['rate<0.01'],
    http_req_duration: ['p(95)<500'],
    ingest_latency_ms: ['p(95)<350'],
  },
}

let sequence = 0
const includeCustomTypes = String(__ENV.INGEST_INCLUDE_CUSTOM_TYPES || 'false').toLowerCase() === 'true'

export function runIngest() {
  sequence += 1
  const selector = sequence % 10

  let eventType = 'service.deployed'
  if (selector === 7) eventType = 'service.rolledback'
  if (selector === 8) eventType = 'service.published'
  if (selector === 9) {
    eventType = includeCustomTypes ? 'dev.cdevents.pipeline.run.started.0.3.0' : 'service.deployed'
  }

  const payload = buildServiceEvent({
    type: eventType,
    service: selector % 2 === 0 ? 'orders' : 'billing',
    environment: selector % 3 === 0 ? 'production' : 'staging',
    sequence,
    chainId: `lt-chain-${Math.floor(sequence / 3)}`,
    pipelineRun: `lt-run-${sequence}`,
  })

  const response = postWebhook(payload)
  ingestLatency.add(response.timings.duration)
  sleep(0.05)
}
