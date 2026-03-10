.PHONY: setup dev chat-terminal nudge-terminal test test-integration test-cover lint test-all migrate build docker start stop logs analytics analytics-xlsx analytics-example ollama-pull

# First-time setup
setup:
	cp -n .env.example .env 2>/dev/null || true
	go mod download
	@echo "Setup complete. Edit .env with your configuration."

# Development
dev:
	go run ./cmd/server

chat-terminal:
	docker compose run --rm --entrypoint /pai-terminal-chat app

nudge-terminal:
	docker compose run --rm --entrypoint /pai-terminal-nudge app --user-id $(USER_ID)

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
	@for f in $$(ls migrations/*.up.sql | sort); do \
		echo "Applying $$f"; \
		docker exec -i $$(docker compose ps -q postgres) psql -U pai -d pai < $$f || exit 1; \
	done

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
	./scripts/analytics.sh

analytics-xlsx:
	./scripts/analytics.sh --xlsx output/spreadsheet/pai-analytics.xlsx

analytics-example:
	./scripts/analytics.sh --example-xlsx output/spreadsheet/pai-analytics-example.xlsx
