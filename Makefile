.PHONY: help dev build run test clean migrate frontend backend stop start restart db-start db-stop db-restart

# Configuration
DB_PORT ?= 5433
DB_NAME ?= printfarm
DB_USER ?= postgres
DB_PASS ?= postgres
DATABASE_URL ?= postgres://$(DB_USER):$(DB_PASS)@localhost:$(DB_PORT)/$(DB_NAME)?sslmode=disable

# Etsy OAuth (optional - set these to enable Etsy integration)
ETSY_CLIENT_ID ?=
ETSY_REDIRECT_URI ?= http://localhost:8080/api/integrations/etsy/callback

# Default target
help:
	@echo "PrintFarm - Maker Project Management"
	@echo ""
	@echo "Usage:"
	@echo "  make start      - Start database and backend server"
	@echo "  make stop       - Stop backend server and database"
	@echo "  make restart    - Stop everything, rebuild, and restart"
	@echo "  make dev        - Run backend and frontend in development mode"
	@echo "  make backend    - Run Go backend only"
	@echo "  make frontend   - Run React frontend only"
	@echo "  make build      - Build production binaries"
	@echo "  make test       - Run tests"
	@echo "  make clean      - Clean build artifacts"
	@echo ""
	@echo "Database:"
	@echo "  make db-start   - Start PostgreSQL in Docker"
	@echo "  make db-stop    - Stop and remove PostgreSQL container"
	@echo "  make db-restart - Restart PostgreSQL (fresh database)"
	@echo "  make db-logs    - Show database logs"
	@echo "  make migrate    - Run database migrations"
	@echo ""
	@echo "Server:"
	@echo "  make server-stop  - Stop the Go backend server"
	@echo "  make server-start - Start the Go backend server"

# Quick start/stop commands
start: db-start migrate-docker server-start
	@echo "✅ PrintFarm is running!"
	@echo "   Backend:  http://localhost:8080"
	@echo "   Frontend: cd web && npm run dev"

stop: server-stop db-stop
	@echo "✅ PrintFarm stopped"

restart: stop
	@sleep 2
	@make start

# Server commands
server-start:
	@echo "Starting Go backend..."
	@DATABASE_URL="$(DATABASE_URL)" ETSY_CLIENT_ID="$(ETSY_CLIENT_ID)" ETSY_REDIRECT_URI="$(ETSY_REDIRECT_URI)" go run -buildvcs=false ./cmd/server &
	@sleep 2
	@echo "Backend started on http://localhost:8080"

server-stop:
	@echo "Stopping Go backend..."
	@pkill -f "go run.*cmd/server" 2>/dev/null || true
	@pkill -f "printfarm" 2>/dev/null || true
	@lsof -ti :8080 | xargs kill -9 2>/dev/null || true
	@echo "Backend stopped"

# Development
dev: db-start migrate-docker
	@echo "Starting development servers..."
	@DATABASE_URL="$(DATABASE_URL)" make -j2 backend frontend

backend:
	@echo "Starting Go backend..."
	DATABASE_URL="$(DATABASE_URL)" ETSY_CLIENT_ID="$(ETSY_CLIENT_ID)" ETSY_REDIRECT_URI="$(ETSY_REDIRECT_URI)" go run -buildvcs=false ./cmd/server

frontend:
	@echo "Starting React frontend..."
	cd web && npm run dev

# Build
build: build-backend build-frontend

build-backend:
	@echo "Building Go backend..."
	go build -buildvcs=false -o bin/server ./cmd/server

build-frontend:
	@echo "Building React frontend..."
	cd web && npm run build

# Database commands
db-start:
	@echo "Starting PostgreSQL in Docker on port $(DB_PORT)..."
	@docker start printfarm-db 2>/dev/null || \
		docker run -d \
			--name printfarm-db \
			-e POSTGRES_USER=$(DB_USER) \
			-e POSTGRES_PASSWORD=$(DB_PASS) \
			-e POSTGRES_DB=$(DB_NAME) \
			-p $(DB_PORT):5432 \
			postgres:16-alpine
	@echo "Waiting for database to be ready..."
	@sleep 3
	@echo "Database running on localhost:$(DB_PORT)"

db-stop:
	@echo "Stopping PostgreSQL..."
	@docker stop printfarm-db 2>/dev/null || true
	@docker rm printfarm-db 2>/dev/null || true
	@echo "Database stopped"

db-restart: db-stop
	@sleep 1
	@make db-start
	@make migrate-docker

db-logs:
	docker logs -f printfarm-db

db-shell:
	docker exec -it printfarm-db psql -U $(DB_USER) -d $(DB_NAME)

# Migrations
migrate:
	@echo "Running migrations..."
	@for f in migrations/*.sql; do \
		echo "Applying $$f..."; \
		psql $(DATABASE_URL) -f $$f; \
	done

migrate-docker:
	@echo "Running migrations on Docker PostgreSQL..."
	@for f in migrations/*.sql; do \
		echo "Applying $$f..."; \
		docker exec -i printfarm-db psql -U $(DB_USER) -d $(DB_NAME) < $$f 2>/dev/null || true; \
	done

# Testing
test:
	go test -v ./...

test-coverage:
	go test -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out

# Cleanup
clean: stop
	rm -rf bin/
	rm -rf web/dist/
	rm -rf uploads/

clean-all: clean
	@docker rm -f printfarm-db 2>/dev/null || true
	@echo "Cleaned everything including database"

# Install dependencies
deps:
	go mod download
	go mod tidy
	cd web && npm install

# Logs
logs:
	@echo "=== Recent server logs ==="
	@tail -50 /tmp/printfarm.log 2>/dev/null || echo "No log file found"

