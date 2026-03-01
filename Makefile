.PHONY: setup dev test test-integration test-cover lint test-all migrate build docker start stop logs analytics ollama-pull

# First-time setup
setup:
	cp -n .env.example .env 2>/dev/null || true
	go mod download
	@echo "Setup complete. Edit .env with your configuration."

# Development
dev:
	go run ./cmd/server

# Testing
test:
	go test ./...

test-v:
	go test -v ./...

test-integration:
	go test -tags=integration ./...

lint:
	golangci-lint run ./...

test-all: lint test

test-cover:
	go test -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html

# Database
migrate:
	@echo "Run: docker exec -i $$(docker compose ps -q postgres) psql -U pai pai < migrations/001_initial.up.sql"

# Build
build:
	CGO_ENABLED=0 go build -o bin/pai-server ./cmd/server

# Docker
docker:
	docker build -f deploy/docker/Dockerfile -t pai-bot .

start:
	docker compose up -d

stop:
	docker compose down

logs:
	docker compose logs -f app

# Ollama
ollama-pull:
	docker compose --profile ollama up -d ollama
	docker exec -it $$(docker compose ps -q ollama) ollama pull llama3

# Analytics
analytics:
	@echo "Analytics script — implemented Day 4"
