# Stock Service Makefile
# Uses the trading-platform containers (postgres, redpanda, redis)

# Default database connection for local development
DB_LOCAL ?= postgres://trader:REDACTED_PASSWORD@localhost:5432/trading_platform?sslmode=disable

.PHONY: build run test clean migrate-up migrate-down migrate-status migrate-create docker-build docker-run

# ─────────────────────────────────────────────────────────────
# Development
# ─────────────────────────────────────────────────────────────

build:
	@echo "🔨 Building stock-service..."
	go build -o bin/stock-service ./cmd/server

run: build
	@echo "🚀 Running stock-service..."
	./bin/stock-service

run-dev:
	@echo "🚀 Running with hot reload (requires air)..."
	air

test:
	@echo "🧪 Running tests..."
	go test -v ./...

test-short:
	@echo "🧪 Running unit tests (no integration)..."
	go test -v -short ./...

test-integration:
	@echo "🧪 Running integration tests..."
	go test -v -run Integration ./...

clean:
	@echo "🧹 Cleaning..."
	rm -rf bin/
	go clean

# ─────────────────────────────────────────────────────────────
# Database Migrations
# ─────────────────────────────────────────────────────────────

migrate-up:
	@echo "⬆️  Running migrations..."
	migrate -path db/migrations -database "$(DB_LOCAL)" up

migrate-down:
	@echo "⬇️  Rolling back last migration..."
	migrate -path db/migrations -database "$(DB_LOCAL)" down 1

migrate-down-all:
	@echo "⬇️  Rolling back ALL migrations..."
	migrate -path db/migrations -database "$(DB_LOCAL)" down -all

migrate-status:
	@echo "📊 Migration status:"
	migrate -path db/migrations -database "$(DB_LOCAL)" version

migrate-create:
	@read -p "Migration name: " name; \
	migrate create -ext sql -dir db/migrations -seq $$name

migrate-force:
	@read -p "Force version to: " version; \
	migrate -path db/migrations -database "$(DB_LOCAL)" force $$version

# ─────────────────────────────────────────────────────────────
# Docker
# ─────────────────────────────────────────────────────────────

docker-build:
	@echo "🐳 Building Docker image..."
	docker build -t stock-service:latest .

docker-run:
	@echo "🐳 Running in Docker (connects to trading-network)..."
	docker-compose up -d stock-service

docker-migrate:
	@echo "🐳 Running migrations via Docker..."
	docker-compose --profile migrate run --rm migrate

docker-logs:
	@echo "📜 Showing logs..."
	docker-compose logs -f stock-service

docker-stop:
	@echo "⏹️  Stopping..."
	docker-compose down

# ─────────────────────────────────────────────────────────────
# Database Access
# ─────────────────────────────────────────────────────────────

db-shell:
	@echo "🐘 Connecting to PostgreSQL..."
	docker exec -it trading-db psql -U trader -d trading_platform

db-check:
	@echo "🔍 Checking database connection..."
	@docker exec trading-db pg_isready -U trader -d trading_platform && echo "✅ Database is ready"

# ─────────────────────────────────────────────────────────────
# Redis Access
# ─────────────────────────────────────────────────────────────

redis-shell:
	@echo "🔴 Connecting to Redis..."
	docker exec -it trading-realtime-data-shared redis-cli

redis-check:
	@echo "🔍 Checking Redis connection..."
	@docker exec trading-realtime-data-shared redis-cli ping && echo "✅ Redis is ready"

# ─────────────────────────────────────────────────────────────
# Kafka/Redpanda Access
# ─────────────────────────────────────────────────────────────

kafka-topics:
	@echo "📋 Listing Kafka topics..."
	docker exec trading-redpanda rpk topic list

kafka-create-topic:
	@read -p "Topic name: " topic; \
	docker exec trading-redpanda rpk topic create $$topic

kafka-consume:
	@read -p "Topic name: " topic; \
	docker exec trading-redpanda rpk topic consume $$topic

# ─────────────────────────────────────────────────────────────
# Health Check
# ─────────────────────────────────────────────────────────────

health:
	@echo "🏥 Checking service health..."
	@curl -s http://localhost:8081/health | jq . || echo "Service not running on port 8081"

check-all:
	@echo "🔍 Checking all services..."
	@make db-check
	@make redis-check
	@echo "📡 Redpanda Console: http://localhost:8080"
	@echo ""
	@make health

# ─────────────────────────────────────────────────────────────
# Help
# ─────────────────────────────────────────────────────────────

help:
	@echo "Stock Service - Available Commands"
	@echo ""
	@echo "Development:"
	@echo "  make build          - Build the binary"
	@echo "  make run            - Build and run locally"
	@echo "  make test           - Run all tests"
	@echo "  make test-short     - Run unit tests only"
	@echo "  make clean          - Clean build artifacts"
	@echo ""
	@echo "Migrations:"
	@echo "  make migrate-up     - Run all pending migrations"
	@echo "  make migrate-down   - Rollback last migration"
	@echo "  make migrate-status - Show current migration version"
	@echo "  make migrate-create - Create a new migration"
	@echo ""
	@echo "Docker:"
	@echo "  make docker-build   - Build Docker image"
	@echo "  make docker-run     - Run in Docker (joins trading-network)"
	@echo "  make docker-logs    - Show container logs"
	@echo "  make docker-stop    - Stop container"
	@echo ""
	@echo "Database/Redis/Kafka:"
	@echo "  make db-shell       - Open psql shell"
	@echo "  make redis-shell    - Open redis-cli shell"
	@echo "  make kafka-topics   - List Kafka topics"
	@echo ""
	@echo "Health:"
	@echo "  make health         - Check service health endpoint"
	@echo "  make check-all      - Check all service connections"
