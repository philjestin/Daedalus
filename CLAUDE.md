# Daedalus (spacechat/printfarm)

A print farm management platform for makers — product catalogs, multi-channel order tracking, printer fleet control, and profitability analytics.

## Tech Stack

- **Backend:** Go 1.24, Chi router, SQLite (WAL mode), slog for logging
- **Frontend:** React 19, TypeScript 5.9, Vite 7, Tailwind CSS 4, TanStack Query v5
- **Desktop:** Wails v2 (macOS/Windows)
- **Integrations:** Etsy (OAuth+PKCE), Squarespace, Bambu Lab (MQTT), OctoPrint, Moonraker
- **Real-time:** WebSocket hub (nhooyr.io/websocket)

## Project Structure

```
cmd/server/          # Standalone HTTP server entry point
internal/
  api/               # HTTP handlers & chi router
  service/           # Business logic layer
  repository/        # Data access (SQLite)
  model/             # Domain types & interfaces
  database/          # DB init, migrations, DBTX interface
  printer/           # Printer integrations
  realtime/          # WebSocket hub
  storage/           # File storage abstraction
  etsy/              # Etsy API client
  squarespace/       # Squarespace API client
  bambu/             # Bambu Lab cloud integration
  receipt/           # Receipt OCR
  threemf/           # 3MF file parsing
  crypto/            # Crypto utilities
  validation/        # Input validation
web/src/
  api/               # API client functions
  components/        # React components
  hooks/             # Custom hooks (useWebSocket, useMaterials, etc.)
  pages/             # Route components
  contexts/          # React contexts
  types/             # TypeScript type definitions
  lib/               # Utilities
migrations/          # SQL migration files (sequential numbering)
```

## Architecture

- **Layered:** API handlers → Services → Repositories → SQLite
- **Repository pattern** with `DBTX` interface for transaction support
- **Dependency injection** — services injected into API handlers
- **`WithTransaction` helper** for ACID operations across repositories
- **WebSocket hub** broadcasts printer status and job state changes in real-time

## Common Commands

```bash
make dev              # Start backend (8080) + frontend (5173) together
make backend          # Go server only
make frontend         # Vite dev server only
make test             # go test -v ./...
make test-coverage    # Tests with coverage report
make build            # Production build (backend binary + frontend assets)
make migrate          # Apply SQL migrations
```

## Testing — MANDATORY

**Every change must include tests and pass the existing suite.**

1. **Always write tests** for new or modified code — no exceptions. If adding a new repository method, service function, or API endpoint, add corresponding test cases.
2. **Run `make test` before and after making changes** to confirm nothing is broken. If any test fails, fix it before moving on.
3. **Run tests incrementally** — after each meaningful change, run at least the relevant package tests (`go test -v ./internal/repository/...`) to catch regressions early rather than waiting until the end.
4. **Do not consider a task complete until all tests pass**, including both new and existing tests.

### Test conventions
- Go standard `testing` package with table-driven tests
- In-memory SQLite (`:memory:`) via `openTestDB()` helper for fast, isolated tests
- Tests live alongside source: `*_test.go`
- Always use `t.Helper()` in test helpers and `t.Cleanup()` for teardown
- Test function names: `TestFunctionName_scenario` (e.g., `TestCreateOrder_duplicateSKU`)

## Database & Migrations

- SQLite with WAL mode, stored at `~/.daedalus/daedalus.db`
- Migrations in `migrations/` with sequential numbering (currently 014)
- Migrations applied automatically on startup
- When adding migrations: create the next numbered file (e.g., `015_description.sql`)

## Code Conventions

### Go
- Standard Go formatting (`gofmt`)
- Explicit error returns — no `panic` except during startup
- Structured logging with `log/slog` (JSON to stdout)
- CamelCase exported, camelCase unexported
- Doc comments on exported functions
- Interfaces defined near consumers, not implementors

### TypeScript/React
- Strict mode enabled (`noUnusedLocals`, `noUnusedParameters`)
- Functional components with hooks only — no class components
- Custom hooks prefixed with `use`
- PascalCase components, camelCase functions/variables
- Tailwind utility classes with `cn()` helper for conditional styling
- TanStack Query for all server state — no manual fetch/useEffect patterns

### CSS/Design
- Tailwind CSS 4 with custom theme
- Fonts: JetBrains Mono (body), Space Grotesk (display)
- Color palette: custom `surface` (grayscale) and `accent` (orange) scales
- Industrial/maker aesthetic

### Git
- Imperative mood, concise messages: "Fix tests", "Add order import", "Update material costs"
- No conventional commit prefixes

## Key Patterns to Follow

- New API endpoints: add handler in `internal/api/`, service in `internal/service/`, repository in `internal/repository/`
- New frontend pages: add page component in `web/src/pages/`, route in router, API client in `web/src/api/`
- Types shared between front/back: Go structs in `internal/model/`, mirrored TypeScript types in `web/src/types/index.ts`
- Prefer editing existing files over creating new ones
- Keep the layered architecture clean — handlers should not contain business logic, repositories should not contain business logic
