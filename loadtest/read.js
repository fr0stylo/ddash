import http from 'k6/http'
import { check, sleep } from 'k6'

import { BASE_URL, devLogin } from './common.js'

export const options = {
  scenarios: {
    read_mix: {
      executor: 'ramping-vus',
      startVUs: 5,
      stages: [
        { duration: __ENV.READ_STAGE_1 || '2m', target: Number(__ENV.READ_VUS_1 || 40) },
        { duration: __ENV.READ_STAGE_2 || '3m', target: Number(__ENV.READ_VUS_2 || 100) },
        { duration: __ENV.READ_STAGE_3 || '2m', target: 0 },
      ],
      exec: 'runRead',
    },
  },
  thresholds: {
    http_req_failed: ['rate<0.01'],
    'http_req_duration{endpoint:home}': ['p(95)<400'],
    'http_req_duration{endpoint:services_grid}': ['p(95)<400'],
    'http_req_duration{endpoint:deployments}': ['p(95)<450'],
    'http_req_duration{endpoint:service_detail}': ['p(95)<450'],
  },
}

let loggedIn = false

function ensureLogin() {
  if (loggedIn) return
  devLogin('loadtest-admin@example.local', '/')
  loggedIn = true
}

export function runRead() {
  ensureLogin()

  const roll = Math.random()
  let response

  if (roll < 0.4) {
    response = http.get(`${BASE_URL}/services/grid?env=all`, { tags: { endpoint: 'services_grid' } })
  } else if (roll < 0.65) {
    response = http.get(`${BASE_URL}/deployments`, { tags: { endpoint: 'deployments' } })
  } else if (roll < 0.85) {
    response = http.get(`${BASE_URL}/s/orders`, { tags: { endpoint: 'service_detail' } })
  } else {
    response = http.get(`${BASE_URL}/`, { tags: { endpoint: 'home' } })
  }

  check(response, {
    'read status 200': (r) => r.status === 200,
  })

  sleep(0.1)
}
