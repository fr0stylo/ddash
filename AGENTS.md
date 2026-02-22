# Repository Guidelines

## Project Overview
This is a Go deployment dashboard application using Echo framework, templ for HTML templates, SQLite with sqlc for database operations, and Tailwind CSS for styling. The application tracks service deployments across environments.

## Project Structure & Module Organization
- `apps/ddash/app.go` is the main DDash entrypoint; it wires Echo routes, embeds `apps/ddash/public/`, and renders templ components.
- `apps/ddash/internal/server/` contains Echo server setup and route registration logic.
- `apps/ddash/internal/server/routes/` organizes DDash route handlers by type (view, api, webhooks).
- `views/` holds UI templates (`.templ`) and their generated Go output (`*_templ.go`). Edit the `.templ` files, not the generated ones.
- `internal/db/` contains database models, queries, and connection logic using sqlc.
- `apps/ddash/internal/webhooks/custom/` handles CDEvents webhook transport into ingestion services.
- `apps/ddash/internal/renderer/` provides template rendering utilities for HTMX/SSE responses.
- `assets/` contains source CSS (Tailwind entrypoint); `apps/ddash/public/` is the compiled, served static output (`apps/ddash/public/styles.css`).
- `internal/db/migrations/` contains SQL schema migration files.
- `go.mod`/`go.sum` define module dependencies, including `templ`, Echo, sqlc, and SQLite.

## Build, Test, and Development Commands
- `go run ./apps/ddash` starts the HTTP server on `:8080`.
- `go build ./...` compiles the entire module.
- `go build -o tmp/ddash ./apps/ddash` builds an executable to `tmp/ddash`.
- `task apps:ddash:run` runs the HTTP server (Taskfile).
- `task apps:ddash:build` builds the server binary.
- `task apps:webhookgenerator:run CONFIG=apps/webhookgenerator/sample.yaml` sends webhooks from a YAML config.
- `apps/webhookgenerator/sample.yaml` shows the webhook generator schema.
- `task check` should run after every change set.
- Dev: run `templ generate --watch --proxy="http://localhost:8080" --cmd="go run ./apps/ddash"` in parallel with `npx @tailwindcss/cli -i ./assets/styles.css -o ./apps/ddash/public/styles.css --watch`.
- `sqlc generate` generates Go code from SQL queries in `internal/db/queries.sql`.
- `go tool sqlc generate` regenerates sqlc without a local install.
- `go test ./...` runs all tests in the repository.
- `go test ./apps/ddash/internal/server/...` runs tests for DDash server packages.
- `go test -v ./apps/ddash/internal/server/routes -run TestSpecificFunction` runs a single route test with verbose output.
- `go test -run TestXxx ./path/to/package` runs tests matching the pattern in a specific package.
- `go test -bench ./...` runs all benchmarks.
- `go test -cover ./...` runs tests with coverage reporting.
- `gofmt -w .` formats all Go files in the repository.
- `go vet ./...` performs static analysis on Go code.
- `golangci-lint run` runs lint checks (requires `golangci-lint`).
- `go mod tidy` cleans up module dependencies.
- `DDASH_DB_PATH=data/<tenant>` configures the SQLite path (default `data/default`).
- `DDASH_WEBHOOK_DB_BASE=data` controls where tenant DB files live for custom webhooks.


## Database Operations
- `sqlc generate` regenerates query code from `internal/db/queries.sql`.
- Database schema changes go in `internal/db/migrations/` directory.
- Use sqlc for type-safe database queries; avoid raw SQL strings in Go code.
- All database models are in `internal/db/queries/models.go` (auto-generated).

## Coding Style & Naming Conventions
- Go formatting: run `gofmt` on changed Go files before committing.
- Go naming: exported identifiers use `PascalCase`; locals use `camelCase`.
- Package names: use lowercase, single words when possible (e.g., `routes`, `server`, `github`).
- Templ: keep component names `PascalCase` and files lowercase (`views/pages/home.templ`).
- Do not edit generated files (`views/**/_templ.go`, `internal/db/queries/*.go`); treat them as build artifacts.
- Error handling: use explicit error returns, avoid panic in production code.
- Logging: use structured logging with `log/slog` throughout the application.
- Context: pass `context.Context` as first parameter to functions that need it.

## Import Organization
- Group imports into three sections: standard library, third-party packages, internal packages.
- Use blank imports (`_`) only when necessary for side effects.
- Prefer aliasing for commonly used packages (e.g., `gh "github.com/google/go-github/v81/github"`).

## HTTP Handler Patterns
- All route handlers implement `echo.HandlerFunc` signature.
- Return errors from handlers; Echo middleware will handle them.
- Use `c.Render()` for template responses with proper HTTP status codes.
- For API endpoints, use `c.JSON()` with appropriate status codes.
- Implement proper HTTP method validation and parameter extraction.

## Template (templ) Guidelines
- Use `templ` components for reusable UI elements.
- Pass data as parameters to components rather than using global state.
- Use conditional rendering with `if/else` blocks in templates.
- For dynamic updates, use HTMX attributes and Server-Sent Events (SSE).
- Keep business logic in Go handlers, not in templates.

## Testing Guidelines
- Place test files alongside the code they test (`*_test.go`).
- Use Go's standard `TestXxx` naming for unit tests and `BenchmarkXxx` for benchmarks.
- Use table-driven tests for multiple test cases.
- Mock external dependencies (database, HTTP clients) in tests.
- Test both success and error paths for all functions.
- Integration tests should use a separate test database.

## Environment & Configuration
- Use environment variables for configuration (e.g., `GITHUB_WEBHOOK_SECRET`, `DDASH_WEBHOOK_DB_BASE`, `DDASH_DB_PATH`).
- Database runs on SQLite by default; connection string in `internal/db/con.go`.
- Server runs on port 8080 by default.
- Static assets are embedded from `apps/ddash/public/` via Go's `embed.FS`.

## Security Considerations
- Validate webhook signatures using provided secrets.
- Sanitize all user inputs and query parameters.
- Use proper HTTP status codes for different error conditions.
- Avoid exposing internal error details to clients.

## Performance Guidelines
- Use connection pooling for database operations.
- Implement proper caching for frequently accessed data.
- Use streaming responses (SSE) for real-time updates.
- Optimize database queries with proper indexes.

## Commit & Pull Request Guidelines
- Use short, imperative commit subjects (e.g., "Add service card footer").
- Include context in commit body when changes are non-trivial.
- PRs should describe behavior changes and reference relevant issues.
- Include screenshots for UI changes in PR descriptions.
- Ensure all tests pass and code is formatted before submitting PRs.
