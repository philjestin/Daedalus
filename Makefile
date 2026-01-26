.PHONY: help dev build run test clean migrate frontend backend

# Default target
help:
	@echo "PrintFarm - Maker Project Management"
	@echo ""
	@echo "Usage:"
	@echo "  make dev       - Run backend and frontend in development mode"
	@echo "  make backend   - Run Go backend only"
	@echo "  make frontend  - Run React frontend only"
	@echo "  make build     - Build production binaries"
	@echo "  make migrate   - Run database migrations"
	@echo "  make test      - Run tests"
	@echo "  make clean     - Clean build artifacts"

# Development
dev:
	@echo "Starting development servers..."
	@make -j2 backend frontend

backend:
	@echo "Starting Go backend..."
	go run ./cmd/server

frontend:
	@echo "Starting React frontend..."
	cd web && npm run dev

# Build
build: build-backend build-frontend

build-backend:
	@echo "Building Go backend..."
	go build -o bin/server ./cmd/server

build-frontend:
	@echo "Building React frontend..."
	cd web && npm run build

# Database
migrate:
	@echo "Running migrations..."
	@for f in migrations/*.sql; do \
		echo "Applying $$f..."; \
		psql $$DATABASE_URL -f $$f; \
	done

migrate-docker:
	@echo "Running migrations on Docker PostgreSQL..."
	@for f in migrations/*.sql; do \
		echo "Applying $$f..."; \
		docker exec -i printfarm-db psql -U postgres -d printfarm < $$f; \
	done

# Docker
docker-db:
	@echo "Starting PostgreSQL in Docker..."
	docker run -d \
		--name printfarm-db \
		-e POSTGRES_USER=postgres \
		-e POSTGRES_PASSWORD=postgres \
		-e POSTGRES_DB=printfarm \
		-p 5432:5432 \
		postgres:16-alpine

docker-db-stop:
	docker stop printfarm-db || true
	docker rm printfarm-db || true

# Testing
test:
	go test -v ./...

test-coverage:
	go test -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out

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

