import http from 'k6/http'
import { check, sleep } from 'k6'

import { BASE_URL, buildServiceEvent, devLogin, postWebhook } from './common.js'

export const options = {
  scenarios: {
    mixed_ingest: {
      executor: 'constant-arrival-rate',
      rate: Number(__ENV.MIXED_INGEST_RPS || 40),
      timeUnit: '1s',
      duration: __ENV.MIXED_DURATION || '8m',
      preAllocatedVUs: Number(__ENV.MIXED_INGEST_PRE_VUS || 20),
      maxVUs: Number(__ENV.MIXED_INGEST_MAX_VUS || 200),
      exec: 'runIngest',
    },
    mixed_read: {
      executor: 'constant-vus',
      vus: Number(__ENV.MIXED_READ_VUS || 80),
      duration: __ENV.MIXED_DURATION || '8m',
      exec: 'runRead',
    },
  },
  thresholds: {
    http_req_failed: ['rate<0.02'],
    http_req_duration: ['p(95)<700'],
    'http_req_duration{endpoint:webhook_ingest}': ['p(95)<450'],
    'http_req_duration{endpoint:service_detail}': ['p(95)<600'],
  },
}

let ingestSequence = 0
let loggedIn = false

function ensureLogin() {
  if (loggedIn) return
  devLogin('loadtest-admin@example.local', '/')
  loggedIn = true
}

export function runIngest() {
  ingestSequence += 1
  const payload = buildServiceEvent({
    type: ingestSequence % 6 === 0 ? 'service.rolledback' : 'service.deployed',
    service: ingestSequence % 2 === 0 ? 'orders' : 'billing',
    environment: ingestSequence % 3 === 0 ? 'production' : 'staging',
    sequence: ingestSequence,
    chainId: `mixed-chain-${Math.floor(ingestSequence / 4)}`,
  })
  postWebhook(payload)
  sleep(0.02)
}

export function runRead() {
  ensureLogin()
  const response = http.get(`${BASE_URL}/s/orders`, { tags: { endpoint: 'service_detail' } })
  check(response, {
    'service detail 200': (r) => r.status === 200,
  })
  sleep(0.08)
}
