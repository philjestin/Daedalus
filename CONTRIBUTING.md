# Contributing to Daedalus

Thanks for your interest in contributing! Here's how to get started.

## Development Setup

```bash
# Clone the repo
git clone https://github.com/philjestin/daedalus.git
cd daedalus

# Install dependencies
make deps

# Run in development mode (backend + frontend)
make dev
```

The backend runs on `:8080` and the frontend on `:5173`.

## Making Changes

1. Fork the repo and create a branch from `main`
2. Make your changes
3. Run tests: `make test`
4. Run the frontend type check: `cd web && npx tsc --noEmit`
5. Open a pull request

## Code Style

- **Go:** Standard `gofmt` formatting, explicit error returns, structured logging with `slog`
- **TypeScript/React:** Strict mode, functional components, TanStack Query for server state
- **CSS:** Tailwind CSS 4 utility classes

## Architecture

Changes should follow the layered architecture:

```
API handlers → Services → Repositories → SQLite
```

- Handlers handle HTTP concerns (parsing, validation, responses)
- Services contain business logic
- Repositories handle data access

## Tests

Every change should include tests. Run `make test` before submitting.

- Go: table-driven tests with in-memory SQLite
- Test files live alongside source: `*_test.go`

## Reporting Issues

Use [GitHub Issues](https://github.com/philjestin/daedalus/issues). Include:

- What you expected vs what happened
- Steps to reproduce
- OS and browser/app version
