BEGIN TRANSACTION;

INSERT OR IGNORE INTO organizations (name, auth_token, webhook_secret, enabled)
VALUES ('default', 'seed-token-default', 'seed-webhook-secret-default', 1);

INSERT OR IGNORE INTO event_store (
  organization_id,
  event_id,
  event_type,
  event_source,
  event_timestamp,
  subject_id,
  subject_source,
  subject_type,
  chain_id,
  raw_event_json
)
VALUES
(
  (SELECT id FROM organizations WHERE name = 'default' LIMIT 1),
  'seed-orders-dev-1',
  'dev.cdevents.service.deployed.0.3.0',
  'ddash://seed/sql',
  '2026-02-18T09:00:00Z',
  'service/orders',
  'ddash://seed/sql',
  'service',
  NULL,
  '{"context":{"id":"seed-orders-dev-1","source":"ddash://seed/sql","type":"dev.cdevents.service.deployed.0.3.0","timestamp":"2026-02-18T09:00:00Z","specversion":"0.5.0"},"subject":{"id":"service/orders","source":"ddash://seed/sql","content":{"environment":{"id":"dev"},"artifactId":"pkg:generic/orders@1.0.0"}}}'
),
(
  (SELECT id FROM organizations WHERE name = 'default' LIMIT 1),
  'seed-orders-staging-1',
  'dev.cdevents.service.upgraded.0.3.0',
  'ddash://seed/sql',
  '2026-02-18T11:00:00Z',
  'service/orders',
  'ddash://seed/sql',
  'service',
  NULL,
  '{"context":{"id":"seed-orders-staging-1","source":"ddash://seed/sql","type":"dev.cdevents.service.upgraded.0.3.0","timestamp":"2026-02-18T11:00:00Z","specversion":"0.5.0"},"subject":{"id":"service/orders","source":"ddash://seed/sql","content":{"environment":{"id":"staging"},"artifactId":"pkg:generic/orders@1.1.0"}}}'
),
(
  (SELECT id FROM organizations WHERE name = 'default' LIMIT 1),
  'seed-orders-prod-1',
  'dev.cdevents.service.deployed.0.3.0',
  'ddash://seed/sql',
  '2026-02-18T13:00:00Z',
  'service/orders',
  'ddash://seed/sql',
  'service',
  NULL,
  '{"context":{"id":"seed-orders-prod-1","source":"ddash://seed/sql","type":"dev.cdevents.service.deployed.0.3.0","timestamp":"2026-02-18T13:00:00Z","specversion":"0.5.0"},"subject":{"id":"service/orders","source":"ddash://seed/sql","content":{"environment":{"id":"production"},"artifactId":"pkg:generic/orders@1.1.0"}}}'
),
(
  (SELECT id FROM organizations WHERE name = 'default' LIMIT 1),
  'seed-billing-dev-1',
  'dev.cdevents.service.deployed.0.3.0',
  'ddash://seed/sql',
  '2026-02-18T08:45:00Z',
  'service/billing',
  'ddash://seed/sql',
  'service',
  NULL,
  '{"context":{"id":"seed-billing-dev-1","source":"ddash://seed/sql","type":"dev.cdevents.service.deployed.0.3.0","timestamp":"2026-02-18T08:45:00Z","specversion":"0.5.0"},"subject":{"id":"service/billing","source":"ddash://seed/sql","content":{"environment":{"id":"dev"},"artifactId":"pkg:generic/billing@2.3.0"}}}'
),
(
  (SELECT id FROM organizations WHERE name = 'default' LIMIT 1),
  'seed-billing-prod-1',
  'dev.cdevents.service.rolledback.0.3.0',
  'ddash://seed/sql',
  '2026-02-18T14:15:00Z',
  'service/billing',
  'ddash://seed/sql',
  'service',
  NULL,
  '{"context":{"id":"seed-billing-prod-1","source":"ddash://seed/sql","type":"dev.cdevents.service.rolledback.0.3.0","timestamp":"2026-02-18T14:15:00Z","specversion":"0.5.0"},"subject":{"id":"service/billing","source":"ddash://seed/sql","content":{"environment":{"id":"production"},"artifactId":"pkg:generic/billing@2.2.9"}}}'
),
(
  (SELECT id FROM organizations WHERE name = 'default' LIMIT 1),
  'seed-catalog-dev-1',
  'dev.cdevents.service.published.0.3.0',
  'ddash://seed/sql',
  '2026-02-18T10:20:00Z',
  'service/catalog',
  'ddash://seed/sql',
  'service',
  NULL,
  '{"context":{"id":"seed-catalog-dev-1","source":"ddash://seed/sql","type":"dev.cdevents.service.published.0.3.0","timestamp":"2026-02-18T10:20:00Z","specversion":"0.5.0"},"subject":{"id":"service/catalog","source":"ddash://seed/sql","content":{"environment":{"id":"dev"},"artifactId":"pkg:generic/catalog@0.9.0"}}}'
),
(
  (SELECT id FROM organizations WHERE name = 'default' LIMIT 1),
  'seed-catalog-staging-1',
  'dev.cdevents.service.deployed.0.3.0',
  'ddash://seed/sql',
  '2026-02-18T12:40:00Z',
  'service/catalog',
  'ddash://seed/sql',
  'service',
  NULL,
  '{"context":{"id":"seed-catalog-staging-1","source":"ddash://seed/sql","type":"dev.cdevents.service.deployed.0.3.0","timestamp":"2026-02-18T12:40:00Z","specversion":"0.5.0"},"subject":{"id":"service/catalog","source":"ddash://seed/sql","content":{"environment":{"id":"staging"},"artifactId":"pkg:generic/catalog@1.0.0"}}}'
),
(
  (SELECT id FROM organizations WHERE name = 'default' LIMIT 1),
  'seed-catalog-prod-1',
  'dev.cdevents.service.removed.0.3.0',
  'ddash://seed/sql',
  '2026-02-18T16:30:00Z',
  'service/catalog',
  'ddash://seed/sql',
  'service',
  NULL,
  '{"context":{"id":"seed-catalog-prod-1","source":"ddash://seed/sql","type":"dev.cdevents.service.removed.0.3.0","timestamp":"2026-02-18T16:30:00Z","specversion":"0.5.0"},"subject":{"id":"service/catalog","source":"ddash://seed/sql","content":{"environment":{"id":"production"},"artifactId":"pkg:generic/catalog@1.0.0"}}}'
)
ON CONFLICT(event_source, event_id) DO NOTHING;

COMMIT;
