.PHONY: help dev build run test clean frontend backend stop start restart show-version bump-patch bump-minor bump-major release site site-build lint lint-go lint-web

# Version
VERSION := $(shell cat VERSION | tr -d 'v\n')
COMMIT  := $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
DATE    := $(shell date -u +%Y-%m-%dT%H:%M:%SZ)
LDFLAGS := -X github.com/philjestin/daedalus/internal/version.Version=$(VERSION) \
           -X github.com/philjestin/daedalus/internal/version.Commit=$(COMMIT) \
           -X github.com/philjestin/daedalus/internal/version.Date=$(DATE) \
           -X github.com/philjestin/daedalus/internal/version.BuiltBy=make

# Etsy OAuth (optional - set these to enable Etsy integration)
ETSY_CLIENT_ID ?=
ETSY_REDIRECT_URI ?= http://localhost:8080/api/integrations/etsy/callback

# Default target
help:
	@echo "Daedalus - Print Farm Management"
	@echo ""
	@echo "Usage:"
	@echo "  make dev        - Run backend and frontend in development mode"
	@echo "  make backend    - Run Go backend only"
	@echo "  make frontend   - Run React frontend only"
	@echo "  make build      - Build production binaries"
	@echo "  make test       - Run tests"
	@echo "  make lint       - Run all linters"
	@echo "  make lint-go    - Run Go linter (golangci-lint)"
	@echo "  make lint-web   - Run frontend linter (ESLint)"
	@echo "  make clean      - Clean build artifacts"
	@echo ""
	@echo "Site:"
	@echo "  make site         - Start docs/marketing site dev server"
	@echo "  make site-build   - Build docs/marketing site"
	@echo ""
	@echo "Versioning:"
	@echo "  make show-version - Show current version"
	@echo "  make bump-patch   - Bump patch version (1.0.0 -> 1.0.1)"
	@echo "  make bump-minor   - Bump minor version (1.0.0 -> 1.1.0)"
	@echo "  make bump-major   - Bump major version (1.0.0 -> 2.0.0)"
	@echo "  make release      - Run interactive release wizard"

# Development
dev:
	@echo "Starting development servers..."
	@make -j2 backend frontend

backend:
	@echo "Starting Go backend..."
	ETSY_CLIENT_ID="$(ETSY_CLIENT_ID)" ETSY_REDIRECT_URI="$(ETSY_REDIRECT_URI)" go run -buildvcs=false ./cmd/server

frontend:
	@echo "Starting React frontend..."
	cd web && npm run dev

# Build
build: build-backend build-frontend

build-backend:
	@echo "Building Go backend (v$(VERSION))..."
	go build -buildvcs=false -ldflags '$(LDFLAGS)' -o bin/server ./cmd/server

build-frontend:
	@echo "Building React frontend..."
	cd web && npm run build

# Testing
test:
	go test -v ./...

test-coverage:
	go test -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out

# Linting
lint: lint-go lint-web

lint-go:
	golangci-lint run ./...

lint-web:
	cd web && npm run lint

# Cleanup
clean:
	rm -rf bin/
	rm -rf web/dist/
	rm -rf uploads/

# Install dependencies
deps:
	go mod download
	go mod tidy
	cd web && npm install

# Versioning
show-version:
	@echo "$(VERSION)"

bump-patch:
	@./scripts/bump-version.sh patch

bump-minor:
	@./scripts/bump-version.sh minor

bump-major:
	@./scripts/bump-version.sh major

release:
	@./scripts/release.sh

# Site (docs/marketing)
site:
	@echo "Starting docs/marketing site dev server..."
	cd site && npm run dev

site-build:
	@echo "Building docs/marketing site..."
	cd site && npm run build
