# DDash Context Map

This document defines DDash bounded contexts and integration boundaries.

## Bounded contexts

- Identity and Access
  - Users, auth sessions, organization membership, roles, join requests.
- Organization Configuration
  - Organization settings, feature flags, required metadata fields, environment ordering.
- Deployment Ingestion
  - CDEvent intake, validation, normalization, append-only storage.
- Service Catalog and Insights
  - Service details, delivery history, risk events, service dependencies.
- GitHub Integration
  - GitHub App install flow, setup intents, installation mappings.
- Presentation and Delivery
  - HTTP handlers, rendered templates, HTMX/SSE fragments.

## High-level relationships

- Presentation and Delivery -> Application services (all contexts)
- GitHub Integration -> Deployment Ingestion (via resolved organization context)
- Service Catalog and Insights -> Deployment Ingestion read models
- Organization Configuration -> Presentation feature toggles
- Identity and Access -> all organization-scoped contexts

## Rules

- Handlers do orchestration only; business invariants live in domain/application layers.
- Cross-context access goes through explicit application APIs, not sqlite/store internals.
- Infrastructure packages implement repositories; they do not own business decisions.
