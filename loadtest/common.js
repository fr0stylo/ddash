import crypto from 'k6/crypto'
import { check } from 'k6'
import http from 'k6/http'

export const BASE_URL = __ENV.BASE_URL || 'http://localhost:19090'
export const AUTH_TOKEN = __ENV.AUTH_TOKEN || 'loadtest-token-01'
export const WEBHOOK_SECRET = __ENV.WEBHOOK_SECRET || 'loadtest-secret-01'

export function cdeventType(shortType) {
  const map = {
    'service.deployed': 'dev.cdevents.service.deployed.0.3.0',
    'service.upgraded': 'dev.cdevents.service.upgraded.0.3.0',
    'service.rolledback': 'dev.cdevents.service.rolledback.0.3.0',
    'service.removed': 'dev.cdevents.service.removed.0.3.0',
    'service.published': 'dev.cdevents.service.published.0.3.0',
    'environment.created': 'dev.cdevents.environment.created.0.3.0',
    'environment.modified': 'dev.cdevents.environment.modified.0.3.0',
    'environment.deleted': 'dev.cdevents.environment.deleted.0.3.0',
  }
  return map[shortType] || shortType
}

export function buildServiceEvent(params) {
  const now = new Date().toISOString()
  const service = params.service || 'orders'
  const env = params.environment || 'staging'
  const eventType = cdeventType(params.type || 'service.deployed')
  const artifact = params.artifact || `pkg:generic/${service}@${params.sequence || Date.now()}`

  return {
    context: {
      id: `lt-${service}-${params.sequence || Date.now()}`,
      source: params.source || 'loadtest/k6',
      type: eventType,
      timestamp: now,
      specversion: '0.5.0',
      ...(params.chainId ? { chainId: params.chainId } : {}),
    },
    subject: {
      id: `service/${service}`,
      source: params.source || 'loadtest/k6',
      content: {
        environment: { id: env },
        artifactId: artifact,
        pipeline: {
          runId: params.pipelineRun || `run-${params.sequence || Date.now()}`,
          url: params.pipelineUrl || '',
        },
        actor: {
          name: params.actor || 'loadtest-bot',
        },
      },
    },
  }
}

export function webhookHeaders(body) {
  const signature = crypto.hmac('sha256', WEBHOOK_SECRET, body, 'hex')
  return {
    Authorization: `Bearer ${AUTH_TOKEN}`,
    'X-Webhook-Signature': signature,
    'Content-Type': 'application/json',
  }
}

export function postWebhook(payload) {
  const body = JSON.stringify(payload)
  const response = http.post(`${BASE_URL}/webhooks/cdevents`, body, {
    headers: webhookHeaders(body),
    tags: { endpoint: 'webhook_ingest' },
  })
  check(response, {
    'webhook accepted': (r) => r.status < 300,
  })
  return response
}

export function devLogin(email, nextPath) {
  const form = {
    email: email || 'loadtest-user@example.local',
    nickname: 'loadtest-user',
    name: 'Load Test User',
    next: nextPath || '/',
  }
  const response = http.post(`${BASE_URL}/auth/dev/login`, form, {
    redirects: 0,
    tags: { endpoint: 'dev_login' },
  })
  check(response, {
    'dev login redirect': (r) => r.status === 302 || r.status === 303,
  })
  return response
}
