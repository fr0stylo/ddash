# Apps

Monorepo application boundaries and runtime packages:

- `apps/ddash` (main dashboard server runtime)
- `apps/githubappingestor` (GitHub App webhook bridge runtime)

Executable entrypoints live directly under `apps/*`:

- `go run ./apps/ddash`
- `go run ./apps/githubappingestor`

Shared/domain code remains under `internal/` and reusable public libraries under `pkg/` and `packages/`.

App-local internals:

- `apps/ddash/internal/*` for DDash-only server/app layers
- `apps/githubappingestor/internal/*` for ingestor-only bridge layers
