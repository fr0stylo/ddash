# Monorepo layout

This repository now follows a monorepo-oriented structure with clear app and package boundaries.

## Structure

- `apps/`
  - application runtime modules (`apps/ddash`, `apps/githubappingestor`)
  - each app owns its internal packages under `apps/<app>/internal/*`
- `internal/`
  - shared platform modules used by multiple apps (`config`, `observability`, `db`)
- `pkg/`
  - public Go packages in the main module
- `packages/`
  - independently consumable packages as nested modules

## Exported shared module

`packages/eventpublisher` is a nested module that exposes event publishing contracts for external consumers:

```go
import "github.com/fr0stylo/ddash/packages/eventpublisher"
```

It currently re-exports the stable API from `pkg/eventpublisher`:

- `type Client`
- `type Event`
- `BuildEventBody`

## Workspace

`go.work` ties the root module and nested exported module together for local development.
