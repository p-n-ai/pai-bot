# Implementation Guide — P&AI Bot

> **Companion to:** [development-timeline.md](development-timeline.md)
> **Architecture reference:** [technical-plan.md](technical-plan.md)
> **Duration:** Day 0 → Day 30 (6 weeks)
> **Scope:** Go backend, AI gateway, Telegram chat adapter, agent engine, progress tracking, motivation features, Next.js admin panel

This guide provides step-by-step executable instructions for every day of the pai-bot development timeline. Each day includes entry criteria, exact file paths, code templates, test specifications, validation commands, and exit checklists.

## How to Use This Guide

1. Work through days sequentially — each day builds on the previous
2. Check **entry criteria** before starting a day
3. **Write tests first** — every feature follows TDD (test → implement → verify)
4. Complete all tasks, run **validation commands**
5. Verify all **exit criteria** checkboxes before moving to the next day
6. Track cumulative progress in the dashboard at the bottom of each day

### Task Owner Legend

| Icon | Owner | Meaning |
|------|-------|---------|
| 🤖 | Claude Code | Can be executed autonomously |
| 🧑 | Human | Requires human action (deploy, test with real users, etc.) |
| 🧑🤖 | Collaborative | AI implements, human validates |

### TDD Workflow (Mandatory)

Every feature follows this strict cycle:

```
1. Write tests first     → define expected behavior
2. Run tests (RED)       → confirm tests fail
3. Implement             → write minimum code to pass
4. Run package tests     → go test ./internal/<package>/...
5. Run FULL test suite   → make test-all
6. Never skip step 5     → every feature must pass the full suite
```

---

## Prerequisites

### Required Tools

```bash
# Go 1.22+ (backend)
go version   # Expected: go1.22.x or higher

# Node.js 20 LTS (admin panel — needed from Week 4)
node --version   # Expected: v20.x.x

# golangci-lint (Go linter)
golangci-lint --version   # Expected: ≥1.55

# Docker + Docker Compose (deployment)
docker --version && docker compose version

# Air (hot reload — optional but recommended)
go install github.com/air-verse/air@latest
```

### Verify Setup

```bash
# All should succeed without errors
go version && docker --version && docker compose version
```

---

## DAY 0 — SETUP (4 hours)

**Entry criteria:** Repository exists with documentation only (README.md, CLAUDE.md, AGENTS.md, docs/). No Go code exists yet.

### Tasks

| # | Task ID | Task | Owner | Files Created |
|---|---------|------|-------|---------------|
| 0.1 | `P-D0-1` | Initialize Go 1.22 project + directory structure | 🤖 | `go.mod`, directories |
| 0.2 | `P-D0-2` | Config loader with `LEARN_` prefix | 🤖 | `internal/platform/config/config.go` |
| 0.3 | `P-D0-3` | Database + cache clients | 🤖 | `internal/platform/database/`, `internal/platform/cache/` |
| 0.4 | `P-D0-4` | Docker Compose + multi-stage Dockerfile | 🤖 | `docker-compose.yml`, `deploy/docker/Dockerfile` |
| 0.5 | `P-D0-5` | Initial database migration | 🤖 | `migrations/001_initial.up.sql` |
| 0.6 | `P-D0-6` | AI Gateway: Provider interface + implementations | 🤖 | `internal/ai/` |
| 0.7 | `P-D0-7` | GitHub Actions CI | 🤖 | `.github/workflows/ci.yml` |
| 0.8 | `P-D0-8` | Create Telegram bot via @BotFather | 🧑 | Bot token saved |

### 0.1 — Initialize Go Module + Directory Structure

```bash
# Initialize Go module
go mod init github.com/p-n-ai/pai-bot

# Create directory structure
mkdir -p cmd/server
mkdir -p internal/{ai,agent,chat,curriculum,progress,auth,tenant}
mkdir -p internal/platform/{config,database,cache,messaging,storage,telemetry,health}
mkdir -p migrations
mkdir -p deploy/docker
mkdir -p deploy/helm/pai/templates
mkdir -p scripts
mkdir -p .github/workflows
```

**File:** `cmd/server/main.go`

```go
package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
	slog.SetDefault(logger)

	slog.Info("starting pai-bot server", "version", "0.1.0")

	// Health check endpoint
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, `{"status":"ok"}`)
	})

	server := &http.Server{
		Addr:         ":8080",
		Handler:      mux,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Graceful shutdown
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	go func() {
		slog.Info("listening", "addr", server.Addr)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("server error", "error", err)
			os.Exit(1)
		}
	}()

	<-ctx.Done()
	slog.Info("shutting down gracefully")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		slog.Error("shutdown error", "error", err)
	}
}
```

### 0.2 — Config Loader (TDD)

**Step 1: Write tests**

**File:** `internal/platform/config/config_test.go`

```go
package config_test

import (
	"os"
	"testing"

	"github.com/p-n-ai/pai-bot/internal/platform/config"
)

func TestLoad_Defaults(t *testing.T) {
	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if cfg.Server.Port != 8080 {
		t.Errorf("Server.Port = %d, want 8080", cfg.Server.Port)
	}
	if cfg.Database.MaxConns != 25 {
		t.Errorf("Database.MaxConns = %d, want 25", cfg.Database.MaxConns)
	}
}

func TestLoad_FromEnv(t *testing.T) {
	t.Setenv("LEARN_SERVER_PORT", "9090")
	t.Setenv("LEARN_DATABASE_URL", "postgres://test:test@localhost/testdb")
	t.Setenv("LEARN_TELEGRAM_BOT_TOKEN", "test-token-123")

	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if cfg.Server.Port != 9090 {
		t.Errorf("Server.Port = %d, want 9090", cfg.Server.Port)
	}
	if cfg.Database.URL != "postgres://test:test@localhost/testdb" {
		t.Errorf("Database.URL = %q, want postgres URL", cfg.Database.URL)
	}
	if cfg.Telegram.BotToken != "test-token-123" {
		t.Errorf("Telegram.BotToken = %q, want test-token-123", cfg.Telegram.BotToken)
	}
}

func TestLoad_TenantMode(t *testing.T) {
	tests := []struct {
		name     string
		envVal   string
		expected string
	}{
		{"default", "", "single"},
		{"single", "single", "single"},
		{"multi", "multi", "multi"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.envVal != "" {
				t.Setenv("LEARN_TENANT_MODE", tt.envVal)
			}
			cfg, err := config.Load()
			if err != nil {
				t.Fatalf("Load() error = %v", err)
			}
			if cfg.Tenant.Mode != tt.expected {
				t.Errorf("Tenant.Mode = %q, want %q", cfg.Tenant.Mode, tt.expected)
			}
		})
	}
}

func TestLoad_AIProviders(t *testing.T) {
	t.Setenv("LEARN_AI_OPENAI_API_KEY", "sk-test")
	t.Setenv("LEARN_AI_OLLAMA_ENABLED", "true")

	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if cfg.AI.OpenAI.APIKey != "sk-test" {
		t.Errorf("AI.OpenAI.APIKey = %q, want sk-test", cfg.AI.OpenAI.APIKey)
	}
	if !cfg.AI.Ollama.Enabled {
		t.Error("AI.Ollama.Enabled should be true")
	}
}
```

**Step 2: Implement**

**File:** `internal/platform/config/config.go`

```go
// Package config loads application configuration from environment variables.
// All variables use the LEARN_ prefix.
package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
)

// Config holds all application configuration.
type Config struct {
	Server   ServerConfig
	Database DatabaseConfig
	Cache    CacheConfig
	NATS     NATSConfig
	AI       AIConfig
	Telegram TelegramConfig
	WhatsApp WhatsAppConfig
	Auth     AuthConfig
	Tenant   TenantConfig
	Log      LogConfig
}

type ServerConfig struct {
	Port int
	Host string
}

type DatabaseConfig struct {
	URL      string
	MaxConns int
	MinConns int
}

type CacheConfig struct {
	URL string
}

type NATSConfig struct {
	URL string
}

type AIConfig struct {
	OpenAI     OpenAIConfig
	Anthropic  AnthropicConfig
	Ollama     OllamaConfig
	OpenRouter OpenRouterConfig
}

type OpenAIConfig struct {
	APIKey string
}

type AnthropicConfig struct {
	APIKey string
}

type OllamaConfig struct {
	Enabled bool
	URL     string
}

type OpenRouterConfig struct {
	APIKey string
}

type TelegramConfig struct {
	BotToken string
}

type WhatsAppConfig struct {
	Enabled     bool
	AccessToken string
	PhoneID     string
	VerifyToken string
}

type AuthConfig struct {
	JWTSecret         string
	AccessTokenTTL    int // minutes
	RefreshTokenTTL   int // days
}

type TenantConfig struct {
	Mode string // "single" or "multi"
}

type LogConfig struct {
	Level  string
	Format string
}

// Load reads configuration from environment variables with LEARN_ prefix.
func Load() (*Config, error) {
	cfg := &Config{
		Server: ServerConfig{
			Port: envInt("LEARN_SERVER_PORT", 8080),
			Host: envStr("LEARN_SERVER_HOST", "0.0.0.0"),
		},
		Database: DatabaseConfig{
			URL:      envStr("LEARN_DATABASE_URL", "postgres://pai:pai@localhost:5432/pai?sslmode=disable"),
			MaxConns: envInt("LEARN_DATABASE_MAX_CONNS", 25),
			MinConns: envInt("LEARN_DATABASE_MIN_CONNS", 5),
		},
		Cache: CacheConfig{
			URL: envStr("LEARN_CACHE_URL", "redis://localhost:6379"),
		},
		NATS: NATSConfig{
			URL: envStr("LEARN_NATS_URL", "nats://localhost:4222"),
		},
		AI: AIConfig{
			OpenAI: OpenAIConfig{
				APIKey: envStr("LEARN_AI_OPENAI_API_KEY", ""),
			},
			Anthropic: AnthropicConfig{
				APIKey: envStr("LEARN_AI_ANTHROPIC_API_KEY", ""),
			},
			Ollama: OllamaConfig{
				Enabled: envBool("LEARN_AI_OLLAMA_ENABLED", false),
				URL:     envStr("LEARN_AI_OLLAMA_URL", "http://localhost:11434"),
			},
			OpenRouter: OpenRouterConfig{
				APIKey: envStr("LEARN_AI_OPENROUTER_API_KEY", ""),
			},
		},
		Telegram: TelegramConfig{
			BotToken: envStr("LEARN_TELEGRAM_BOT_TOKEN", ""),
		},
		WhatsApp: WhatsAppConfig{
			Enabled:     envBool("LEARN_WHATSAPP_ENABLED", false),
			AccessToken: envStr("LEARN_WHATSAPP_ACCESS_TOKEN", ""),
			PhoneID:     envStr("LEARN_WHATSAPP_PHONE_ID", ""),
			VerifyToken: envStr("LEARN_WHATSAPP_VERIFY_TOKEN", ""),
		},
		Auth: AuthConfig{
			JWTSecret:       envStr("LEARN_AUTH_JWT_SECRET", "change-me-in-production"),
			AccessTokenTTL:  envInt("LEARN_AUTH_ACCESS_TOKEN_TTL", 15),
			RefreshTokenTTL: envInt("LEARN_AUTH_REFRESH_TOKEN_TTL", 7),
		},
		Tenant: TenantConfig{
			Mode: envStr("LEARN_TENANT_MODE", "single"),
		},
		Log: LogConfig{
			Level:  envStr("LEARN_LOG_LEVEL", "info"),
			Format: envStr("LEARN_LOG_FORMAT", "json"),
		},
	}

	return cfg, nil
}

// Validate checks that required configuration is present.
func (c *Config) Validate() error {
	if c.Telegram.BotToken == "" {
		return fmt.Errorf("LEARN_TELEGRAM_BOT_TOKEN is required")
	}

	hasAI := c.AI.OpenAI.APIKey != "" ||
		c.AI.Anthropic.APIKey != "" ||
		c.AI.Ollama.Enabled ||
		c.AI.OpenRouter.APIKey != ""
	if !hasAI {
		return fmt.Errorf("at least one AI provider must be configured")
	}

	if c.Tenant.Mode != "single" && c.Tenant.Mode != "multi" {
		return fmt.Errorf("LEARN_TENANT_MODE must be 'single' or 'multi', got %q", c.Tenant.Mode)
	}

	return nil
}

func envStr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func envInt(key string, fallback int) int {
	if v := os.Getenv(key); v != "" {
		if i, err := strconv.Atoi(v); err == nil {
			return i
		}
	}
	return fallback
}

func envBool(key string, fallback bool) bool {
	if v := os.Getenv(key); v != "" {
		return strings.EqualFold(v, "true") || v == "1"
	}
	return fallback
}
```

### 0.3 — Database + Cache Clients (TDD)

**Step 1: Write tests**

**File:** `internal/platform/database/database_test.go`

```go
package database_test

import (
	"testing"

	"github.com/p-n-ai/pai-bot/internal/platform/database"
)

func TestParseURL(t *testing.T) {
	tests := []struct {
		name    string
		url     string
		wantErr bool
	}{
		{"valid", "postgres://user:pass@localhost:5432/db", false},
		{"empty", "", true},
		{"invalid", "not-a-url", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := database.ParseURL(tt.url)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseURL() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
```

**Step 2: Implement**

**File:** `internal/platform/database/database.go`

```go
// Package database provides PostgreSQL connection management via pgx.
package database

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// DB wraps a pgx connection pool.
type DB struct {
	Pool *pgxpool.Pool
}

// ParseURL validates a PostgreSQL connection URL.
func ParseURL(url string) (*pgxpool.Config, error) {
	if url == "" {
		return nil, fmt.Errorf("database URL is empty")
	}
	cfg, err := pgxpool.ParseConfig(url)
	if err != nil {
		return nil, fmt.Errorf("invalid database URL: %w", err)
	}
	return cfg, nil
}

// New creates a new database connection pool.
func New(ctx context.Context, url string, maxConns, minConns int) (*DB, error) {
	cfg, err := ParseURL(url)
	if err != nil {
		return nil, err
	}

	cfg.MaxConns = int32(maxConns)
	cfg.MinConns = int32(minConns)
	cfg.MaxConnLifetime = 30 * time.Minute
	cfg.MaxConnIdleTime = 5 * time.Minute

	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		return nil, fmt.Errorf("creating connection pool: %w", err)
	}

	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("pinging database: %w", err)
	}

	return &DB{Pool: pool}, nil
}

// Close shuts down the connection pool.
func (db *DB) Close() {
	db.Pool.Close()
}

// HealthCheck verifies the database connection is alive.
func (db *DB) HealthCheck(ctx context.Context) error {
	return db.Pool.Ping(ctx)
}
```

**File:** `internal/platform/cache/cache_test.go`

```go
package cache_test

import (
	"testing"

	"github.com/p-n-ai/pai-bot/internal/platform/cache"
)

func TestParseURL(t *testing.T) {
	tests := []struct {
		name    string
		url     string
		wantErr bool
	}{
		{"valid-redis", "redis://localhost:6379", false},
		{"valid-with-db", "redis://localhost:6379/0", false},
		{"empty", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := cache.ParseURL(tt.url)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseURL() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
```

**File:** `internal/platform/cache/cache.go`

```go
// Package cache provides a Dragonfly/Redis client wrapper.
package cache

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

// Cache wraps a Redis/Dragonfly client.
type Cache struct {
	Client *redis.Client
}

// ParseURL validates a Redis connection URL.
func ParseURL(url string) (*redis.Options, error) {
	if url == "" {
		return nil, fmt.Errorf("cache URL is empty")
	}
	opts, err := redis.ParseURL(url)
	if err != nil {
		return nil, fmt.Errorf("invalid cache URL: %w", err)
	}
	return opts, nil
}

// New creates a new cache client.
func New(ctx context.Context, url string) (*Cache, error) {
	opts, err := ParseURL(url)
	if err != nil {
		return nil, err
	}

	opts.DialTimeout = 5 * time.Second
	opts.ReadTimeout = 3 * time.Second
	opts.WriteTimeout = 3 * time.Second

	client := redis.NewClient(opts)

	if err := client.Ping(ctx).Err(); err != nil {
		client.Close()
		return nil, fmt.Errorf("pinging cache: %w", err)
	}

	return &Cache{Client: client}, nil
}

// Close shuts down the cache client.
func (c *Cache) Close() error {
	return c.Client.Close()
}

// HealthCheck verifies the cache connection is alive.
func (c *Cache) HealthCheck(ctx context.Context) error {
	return c.Client.Ping(ctx).Err()
}
```

### 0.4 — Docker Compose + Dockerfile

**File:** `docker-compose.yml`

```yaml
services:
  postgres:
    image: postgres:17-alpine
    ports:
      - "5432:5432"
    environment:
      POSTGRES_USER: pai
      POSTGRES_PASSWORD: pai
      POSTGRES_DB: pai
    volumes:
      - postgres-data:/var/lib/postgresql/data
    healthcheck:
      test: ["CMD-SHELL", "pg_isready -U pai"]
      interval: 5s
      timeout: 3s
      retries: 5

  dragonfly:
    image: docker.dragonflydb.io/dragonflydb/dragonfly
    ports:
      - "6379:6379"
    volumes:
      - dragonfly-data:/data
    healthcheck:
      test: ["CMD", "redis-cli", "ping"]
      interval: 5s
      timeout: 3s
      retries: 5

  nats:
    image: nats:2.10-alpine
    ports:
      - "4222:4222"
      - "8222:8222"
    command: ["--jetstream", "--store_dir", "/data", "--http_port", "8222"]
    volumes:
      - nats-data:/data

  app:
    build:
      context: .
      dockerfile: deploy/docker/Dockerfile
    ports:
      - "8080:8080"
    env_file:
      - .env
    depends_on:
      postgres:
        condition: service_healthy
      dragonfly:
        condition: service_healthy
      nats:
        condition: service_started
    restart: unless-stopped

  ollama:
    image: ollama/ollama:latest
    ports:
      - "11434:11434"
    volumes:
      - ollama-data:/root/.ollama
    profiles:
      - ollama

volumes:
  postgres-data:
  dragonfly-data:
  nats-data:
  ollama-data:
```

**File:** `deploy/docker/Dockerfile`

```dockerfile
# Stage 1: Build Go binary
FROM golang:1.22-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o /pai-server ./cmd/server

# Stage 2: Final image (~25MB)
FROM alpine:3.20
RUN apk add --no-cache ca-certificates tzdata
COPY --from=builder /pai-server /pai-server
EXPOSE 8080
ENTRYPOINT ["/pai-server"]
```

### 0.5 — Initial Database Migration

**File:** `migrations/001_initial.up.sql`

```sql
-- P&AI Bot — Initial Schema
-- All tables include tenant_id for multi-tenancy.

-- Multi-tenancy
CREATE TABLE tenants (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name        TEXT NOT NULL,
    slug        TEXT UNIQUE NOT NULL,
    config      JSONB DEFAULT '{}',
    created_at  TIMESTAMPTZ DEFAULT NOW()
);

-- Users (students, teachers, parents, admins)
CREATE TABLE users (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id   UUID NOT NULL REFERENCES tenants(id),
    role        TEXT NOT NULL CHECK (role IN ('student', 'teacher', 'parent', 'admin', 'platform_admin')),
    name        TEXT NOT NULL,
    external_id TEXT,
    channel     TEXT NOT NULL DEFAULT 'telegram',
    form        TEXT,
    config      JSONB DEFAULT '{}',
    created_at  TIMESTAMPTZ DEFAULT NOW(),
    updated_at  TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_users_external_id ON users(external_id);
CREATE INDEX idx_users_tenant_id ON users(tenant_id);

-- Conversations
CREATE TABLE conversations (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id     UUID NOT NULL REFERENCES users(id),
    tenant_id   UUID NOT NULL REFERENCES tenants(id),
    topic_id    TEXT,
    state       TEXT NOT NULL DEFAULT 'idle',
    metadata    JSONB DEFAULT '{}',
    started_at  TIMESTAMPTZ DEFAULT NOW(),
    ended_at    TIMESTAMPTZ
);

CREATE INDEX idx_conversations_user_id ON conversations(user_id);

-- Messages
CREATE TABLE messages (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    conversation_id UUID NOT NULL REFERENCES conversations(id),
    tenant_id       UUID NOT NULL REFERENCES tenants(id),
    role            TEXT NOT NULL CHECK (role IN ('user', 'assistant', 'system')),
    content         TEXT NOT NULL,
    model           TEXT,
    input_tokens    INTEGER,
    output_tokens   INTEGER,
    created_at      TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_messages_conversation_id ON messages(conversation_id);

-- Learning progress per topic (SM-2 data)
CREATE TABLE learning_progress (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id         UUID NOT NULL REFERENCES users(id),
    tenant_id       UUID NOT NULL REFERENCES tenants(id),
    syllabus_id     TEXT NOT NULL,
    topic_id        TEXT NOT NULL,
    mastery_score   REAL DEFAULT 0.0,
    ease_factor     REAL DEFAULT 2.5,
    interval_days   INTEGER DEFAULT 1,
    repetitions     INTEGER DEFAULT 0,
    next_review_at  TIMESTAMPTZ,
    last_studied_at TIMESTAMPTZ,
    created_at      TIMESTAMPTZ DEFAULT NOW(),
    updated_at      TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE(user_id, syllabus_id, topic_id)
);

CREATE INDEX idx_learning_progress_user_id ON learning_progress(user_id);
CREATE INDEX idx_learning_progress_next_review ON learning_progress(next_review_at);

-- Events (analytics / audit log)
CREATE TABLE events (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id   UUID NOT NULL REFERENCES tenants(id),
    user_id     UUID REFERENCES users(id),
    event_type  TEXT NOT NULL,
    data        JSONB DEFAULT '{}',
    created_at  TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_events_user_id ON events(user_id);
CREATE INDEX idx_events_type ON events(event_type);
CREATE INDEX idx_events_created_at ON events(created_at);

-- Insert default tenant for single-tenant mode
INSERT INTO tenants (name, slug) VALUES ('Default', 'default');
```

**File:** `migrations/001_initial.down.sql`

```sql
DROP TABLE IF EXISTS events;
DROP TABLE IF EXISTS learning_progress;
DROP TABLE IF EXISTS messages;
DROP TABLE IF EXISTS conversations;
DROP TABLE IF EXISTS users;
DROP TABLE IF EXISTS tenants;
```

### 0.6 — AI Gateway: Provider Interface + Implementations (TDD)

**Step 1: Write tests**

**File:** `internal/ai/gateway_test.go`

```go
package ai_test

import (
	"context"
	"testing"

	"github.com/p-n-ai/pai-bot/internal/ai"
)

func TestMockProvider_Complete(t *testing.T) {
	mock := ai.NewMockProvider("test response")

	resp, err := mock.Complete(context.Background(), ai.CompletionRequest{
		Messages: []ai.Message{
			{Role: "user", Content: "Hello"},
		},
	})
	if err != nil {
		t.Fatalf("Complete() error = %v", err)
	}
	if resp.Content != "test response" {
		t.Errorf("Content = %q, want %q", resp.Content, "test response")
	}
	if resp.Model != "mock" {
		t.Errorf("Model = %q, want %q", resp.Model, "mock")
	}
}

func TestMockProvider_HealthCheck(t *testing.T) {
	mock := ai.NewMockProvider("response")
	if err := mock.HealthCheck(context.Background()); err != nil {
		t.Errorf("HealthCheck() error = %v", err)
	}
}

func TestMockProvider_Models(t *testing.T) {
	mock := ai.NewMockProvider("response")
	models := mock.Models()
	if len(models) == 0 {
		t.Error("Models() returned empty")
	}
}

func TestTaskType_String(t *testing.T) {
	tests := []struct {
		task     ai.TaskType
		expected string
	}{
		{ai.TaskTeaching, "teaching"},
		{ai.TaskGrading, "grading"},
		{ai.TaskNudge, "nudge"},
		{ai.TaskAnalysis, "analysis"},
	}
	for _, tt := range tests {
		if tt.task.String() != tt.expected {
			t.Errorf("TaskType.String() = %q, want %q", tt.task.String(), tt.expected)
		}
	}
}
```

**Step 2: Implement**

**File:** `internal/ai/gateway.go`

```go
// Package ai provides a provider-agnostic AI gateway with task-based routing.
package ai

import "context"

// TaskType defines the kind of AI task for routing purposes.
type TaskType int

const (
	TaskTeaching TaskType = iota
	TaskGrading
	TaskNudge
	TaskAnalysis
)

func (t TaskType) String() string {
	switch t {
	case TaskTeaching:
		return "teaching"
	case TaskGrading:
		return "grading"
	case TaskNudge:
		return "nudge"
	case TaskAnalysis:
		return "analysis"
	default:
		return "unknown"
	}
}

// Message represents a chat message.
type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// CompletionRequest is the input to an AI completion.
type CompletionRequest struct {
	Messages    []Message `json:"messages"`
	Model       string    `json:"model,omitempty"`
	MaxTokens   int       `json:"max_tokens,omitempty"`
	Temperature float64   `json:"temperature,omitempty"`
	Task        TaskType  `json:"task,omitempty"`
}

// CompletionResponse is the output from an AI completion.
type CompletionResponse struct {
	Content      string `json:"content"`
	Model        string `json:"model"`
	InputTokens  int    `json:"input_tokens"`
	OutputTokens int    `json:"output_tokens"`
}

// StreamChunk represents a streaming response chunk.
type StreamChunk struct {
	Content string
	Done    bool
	Error   error
}

// ModelInfo describes an available model.
type ModelInfo struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	MaxTokens   int    `json:"max_tokens"`
	Description string `json:"description"`
}

// Provider is the interface all AI providers must implement.
type Provider interface {
	Complete(ctx context.Context, req CompletionRequest) (CompletionResponse, error)
	StreamComplete(ctx context.Context, req CompletionRequest) (<-chan StreamChunk, error)
	Models() []ModelInfo
	HealthCheck(ctx context.Context) error
}
```

**File:** `internal/ai/mock.go`

```go
package ai

import "context"

// MockProvider is a test double for AI providers.
type MockProvider struct {
	Response string
	Err      error
}

func NewMockProvider(response string) *MockProvider {
	return &MockProvider{Response: response}
}

func (m *MockProvider) Complete(_ context.Context, _ CompletionRequest) (CompletionResponse, error) {
	if m.Err != nil {
		return CompletionResponse{}, m.Err
	}
	return CompletionResponse{
		Content:      m.Response,
		Model:        "mock",
		InputTokens:  10,
		OutputTokens: len(m.Response),
	}, nil
}

func (m *MockProvider) StreamComplete(_ context.Context, _ CompletionRequest) (<-chan StreamChunk, error) {
	ch := make(chan StreamChunk, 1)
	go func() {
		defer close(ch)
		ch <- StreamChunk{Content: m.Response, Done: true}
	}()
	return ch, nil
}

func (m *MockProvider) Models() []ModelInfo {
	return []ModelInfo{
		{ID: "mock", Name: "Mock Model", MaxTokens: 4096, Description: "Test mock"},
	}
}

func (m *MockProvider) HealthCheck(_ context.Context) error {
	return m.Err
}
```

**File:** `internal/ai/router.go`

```go
package ai

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
)

// Router selects the best provider based on task type and availability.
type Router struct {
	providers map[string]Provider
	fallback  []string // ordered fallback chain
	mu        sync.RWMutex
}

// NewRouter creates a new AI router with the given providers.
func NewRouter() *Router {
	return &Router{
		providers: make(map[string]Provider),
	}
}

// Register adds a provider to the router.
func (r *Router) Register(name string, provider Provider) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.providers[name] = provider
	r.fallback = append(r.fallback, name)
}

// Complete routes a request to the best available provider.
func (r *Router) Complete(ctx context.Context, req CompletionRequest) (CompletionResponse, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	// Try each provider in fallback order
	for _, name := range r.fallback {
		provider := r.providers[name]

		resp, err := provider.Complete(ctx, req)
		if err != nil {
			slog.Warn("AI provider failed, trying next",
				"provider", name,
				"error", err,
			)
			continue
		}

		slog.Debug("AI request completed",
			"provider", name,
			"model", resp.Model,
			"input_tokens", resp.InputTokens,
			"output_tokens", resp.OutputTokens,
		)
		return resp, nil
	}

	return CompletionResponse{}, fmt.Errorf("all AI providers failed")
}

// HasProvider returns true if at least one provider is registered.
func (r *Router) HasProvider() bool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return len(r.providers) > 0
}
```

**File:** `internal/ai/provider_openai.go`

```go
package ai

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

// OpenAIProvider implements the Provider interface for OpenAI.
type OpenAIProvider struct {
	apiKey  string
	baseURL string
	client  *http.Client
}

func NewOpenAIProvider(apiKey string) (*OpenAIProvider, error) {
	if apiKey == "" {
		return nil, fmt.Errorf("OpenAI API key is required")
	}
	return &OpenAIProvider{
		apiKey:  apiKey,
		baseURL: "https://api.openai.com/v1",
		client:  &http.Client{},
	}, nil
}

func (p *OpenAIProvider) Complete(ctx context.Context, req CompletionRequest) (CompletionResponse, error) {
	model := req.Model
	if model == "" {
		model = "gpt-4o"
	}
	maxTokens := req.MaxTokens
	if maxTokens == 0 {
		maxTokens = 4096
	}

	body := map[string]interface{}{
		"model":      model,
		"messages":   req.Messages,
		"max_tokens": maxTokens,
	}
	if req.Temperature > 0 {
		body["temperature"] = req.Temperature
	}

	jsonBody, err := json.Marshal(body)
	if err != nil {
		return CompletionResponse{}, fmt.Errorf("marshaling request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", p.baseURL+"/chat/completions", bytes.NewReader(jsonBody))
	if err != nil {
		return CompletionResponse{}, err
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+p.apiKey)

	resp, err := p.client.Do(httpReq)
	if err != nil {
		return CompletionResponse{}, fmt.Errorf("OpenAI API call: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return CompletionResponse{}, fmt.Errorf("reading response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return CompletionResponse{}, fmt.Errorf("OpenAI API error %d: %s", resp.StatusCode, string(respBody))
	}

	var result struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
		Usage struct {
			PromptTokens     int `json:"prompt_tokens"`
			CompletionTokens int `json:"completion_tokens"`
		} `json:"usage"`
		Model string `json:"model"`
	}
	if err := json.Unmarshal(respBody, &result); err != nil {
		return CompletionResponse{}, fmt.Errorf("parsing response: %w", err)
	}

	if len(result.Choices) == 0 {
		return CompletionResponse{}, fmt.Errorf("OpenAI returned no choices")
	}

	return CompletionResponse{
		Content:      result.Choices[0].Message.Content,
		Model:        result.Model,
		InputTokens:  result.Usage.PromptTokens,
		OutputTokens: result.Usage.CompletionTokens,
	}, nil
}

func (p *OpenAIProvider) StreamComplete(ctx context.Context, req CompletionRequest) (<-chan StreamChunk, error) {
	ch := make(chan StreamChunk, 1)
	go func() {
		defer close(ch)
		resp, err := p.Complete(ctx, req)
		if err != nil {
			ch <- StreamChunk{Error: err}
			return
		}
		ch <- StreamChunk{Content: resp.Content, Done: true}
	}()
	return ch, nil
}

func (p *OpenAIProvider) Models() []ModelInfo {
	return []ModelInfo{
		{ID: "gpt-4o", Name: "GPT-4o", MaxTokens: 128000, Description: "Most capable"},
		{ID: "gpt-4o-mini", Name: "GPT-4o Mini", MaxTokens: 128000, Description: "Fast and affordable"},
	}
}

func (p *OpenAIProvider) HealthCheck(ctx context.Context) error {
	httpReq, err := http.NewRequestWithContext(ctx, "GET", p.baseURL+"/models", nil)
	if err != nil {
		return err
	}
	httpReq.Header.Set("Authorization", "Bearer "+p.apiKey)
	resp, err := p.client.Do(httpReq)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("OpenAI health check failed: %d", resp.StatusCode)
	}
	return nil
}
```

**File:** `internal/ai/provider_anthropic.go`

```go
package ai

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

// AnthropicProvider implements the Provider interface for Anthropic Claude.
type AnthropicProvider struct {
	apiKey  string
	baseURL string
	client  *http.Client
}

func NewAnthropicProvider(apiKey string) (*AnthropicProvider, error) {
	if apiKey == "" {
		return nil, fmt.Errorf("Anthropic API key is required")
	}
	return &AnthropicProvider{
		apiKey:  apiKey,
		baseURL: "https://api.anthropic.com/v1",
		client:  &http.Client{},
	}, nil
}

func (p *AnthropicProvider) Complete(ctx context.Context, req CompletionRequest) (CompletionResponse, error) {
	model := req.Model
	if model == "" {
		model = "claude-sonnet-4-6"
	}
	maxTokens := req.MaxTokens
	if maxTokens == 0 {
		maxTokens = 4096
	}

	// Separate system message from user/assistant messages
	var systemPrompt string
	var messages []map[string]string
	for _, m := range req.Messages {
		if m.Role == "system" {
			systemPrompt = m.Content
			continue
		}
		messages = append(messages, map[string]string{
			"role":    m.Role,
			"content": m.Content,
		})
	}

	body := map[string]interface{}{
		"model":      model,
		"max_tokens": maxTokens,
		"messages":   messages,
	}
	if systemPrompt != "" {
		body["system"] = systemPrompt
	}
	if req.Temperature > 0 {
		body["temperature"] = req.Temperature
	}

	jsonBody, err := json.Marshal(body)
	if err != nil {
		return CompletionResponse{}, fmt.Errorf("marshaling request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", p.baseURL+"/messages", bytes.NewReader(jsonBody))
	if err != nil {
		return CompletionResponse{}, err
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("x-api-key", p.apiKey)
	httpReq.Header.Set("anthropic-version", "2023-06-01")

	resp, err := p.client.Do(httpReq)
	if err != nil {
		return CompletionResponse{}, fmt.Errorf("Anthropic API call: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return CompletionResponse{}, fmt.Errorf("reading response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return CompletionResponse{}, fmt.Errorf("Anthropic API error %d: %s", resp.StatusCode, string(respBody))
	}

	var result struct {
		Content []struct {
			Text string `json:"text"`
		} `json:"content"`
		Model string `json:"model"`
		Usage struct {
			InputTokens  int `json:"input_tokens"`
			OutputTokens int `json:"output_tokens"`
		} `json:"usage"`
	}
	if err := json.Unmarshal(respBody, &result); err != nil {
		return CompletionResponse{}, fmt.Errorf("parsing response: %w", err)
	}

	if len(result.Content) == 0 {
		return CompletionResponse{}, fmt.Errorf("Anthropic returned no content")
	}

	return CompletionResponse{
		Content:      result.Content[0].Text,
		Model:        result.Model,
		InputTokens:  result.Usage.InputTokens,
		OutputTokens: result.Usage.OutputTokens,
	}, nil
}

func (p *AnthropicProvider) StreamComplete(ctx context.Context, req CompletionRequest) (<-chan StreamChunk, error) {
	ch := make(chan StreamChunk, 1)
	go func() {
		defer close(ch)
		resp, err := p.Complete(ctx, req)
		if err != nil {
			ch <- StreamChunk{Error: err}
			return
		}
		ch <- StreamChunk{Content: resp.Content, Done: true}
	}()
	return ch, nil
}

func (p *AnthropicProvider) Models() []ModelInfo {
	return []ModelInfo{
		{ID: "claude-sonnet-4-6", Name: "Claude Sonnet 4.6", MaxTokens: 200000, Description: "Best for teaching"},
		{ID: "claude-haiku-4-5-20251001", Name: "Claude Haiku 4.5", MaxTokens: 200000, Description: "Fast grading"},
	}
}

func (p *AnthropicProvider) HealthCheck(ctx context.Context) error {
	_, err := p.Complete(ctx, CompletionRequest{
		Messages:  []Message{{Role: "user", Content: "ping"}},
		MaxTokens: 1,
	})
	return err
}
```

**File:** `internal/ai/provider_ollama.go`

```go
package ai

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

// OllamaProvider implements the Provider interface for self-hosted Ollama.
type OllamaProvider struct {
	baseURL string
	client  *http.Client
}

func NewOllamaProvider(baseURL string) (*OllamaProvider, error) {
	if baseURL == "" {
		baseURL = "http://localhost:11434"
	}
	return &OllamaProvider{
		baseURL: baseURL,
		client:  &http.Client{},
	}, nil
}

func (p *OllamaProvider) Complete(ctx context.Context, req CompletionRequest) (CompletionResponse, error) {
	model := req.Model
	if model == "" {
		model = "llama3"
	}

	body := map[string]interface{}{
		"model":    model,
		"messages": req.Messages,
		"stream":   false,
	}
	if req.Temperature > 0 {
		body["options"] = map[string]interface{}{
			"temperature": req.Temperature,
		}
	}

	jsonBody, err := json.Marshal(body)
	if err != nil {
		return CompletionResponse{}, fmt.Errorf("marshaling request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", p.baseURL+"/api/chat", bytes.NewReader(jsonBody))
	if err != nil {
		return CompletionResponse{}, err
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := p.client.Do(httpReq)
	if err != nil {
		return CompletionResponse{}, fmt.Errorf("Ollama API call: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return CompletionResponse{}, fmt.Errorf("reading response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return CompletionResponse{}, fmt.Errorf("Ollama API error %d: %s", resp.StatusCode, string(respBody))
	}

	var result struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
		Model string `json:"model"`
	}
	if err := json.Unmarshal(respBody, &result); err != nil {
		return CompletionResponse{}, fmt.Errorf("parsing response: %w", err)
	}

	return CompletionResponse{
		Content:      result.Message.Content,
		Model:        result.Model,
		InputTokens:  0,
		OutputTokens: 0,
	}, nil
}

func (p *OllamaProvider) StreamComplete(ctx context.Context, req CompletionRequest) (<-chan StreamChunk, error) {
	ch := make(chan StreamChunk, 1)
	go func() {
		defer close(ch)
		resp, err := p.Complete(ctx, req)
		if err != nil {
			ch <- StreamChunk{Error: err}
			return
		}
		ch <- StreamChunk{Content: resp.Content, Done: true}
	}()
	return ch, nil
}

func (p *OllamaProvider) Models() []ModelInfo {
	return []ModelInfo{
		{ID: "llama3", Name: "Llama 3", MaxTokens: 8192, Description: "Free self-hosted"},
		{ID: "mistral", Name: "Mistral", MaxTokens: 32768, Description: "Free self-hosted"},
	}
}

func (p *OllamaProvider) HealthCheck(ctx context.Context) error {
	httpReq, err := http.NewRequestWithContext(ctx, "GET", p.baseURL+"/api/tags", nil)
	if err != nil {
		return err
	}
	resp, err := p.client.Do(httpReq)
	if err != nil {
		return fmt.Errorf("Ollama not reachable: %w", err)
	}
	defer resp.Body.Close()
	return nil
}
```

### 0.7 — GitHub Actions CI

**File:** `.github/workflows/ci.yml`

```yaml
name: CI
on: [push, pull_request]

jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4

      - name: Set up Go
        uses: actions/setup-go@v5
        with:
          go-version: "1.22"

      - name: Download dependencies
        run: go mod download

      - name: Run tests
        run: go test ./...

      - name: Run linter
        uses: golangci/golangci-lint-action@v4
        with:
          version: latest

  build:
    runs-on: ubuntu-latest
    needs: test
    steps:
      - uses: actions/checkout@v4

      - name: Set up Go
        uses: actions/setup-go@v5
        with:
          go-version: "1.22"

      - name: Build server
        run: CGO_ENABLED=0 go build -o bin/pai-server ./cmd/server

      - name: Build Docker image
        run: docker build -f deploy/docker/Dockerfile -t pai-bot .
```

### 0.8 — Create Makefile + .env.example

**File:** `Makefile`

```makefile
.PHONY: setup dev test test-integration lint test-all migrate build docker start stop logs analytics

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
```

**File:** `.env.example`

```bash
# P&AI Bot Configuration
# Copy to .env and fill in your values
# All variables use the LEARN_ prefix

# --- Server ---
LEARN_SERVER_PORT=8080

# --- Database ---
LEARN_DATABASE_URL=postgres://pai:pai@localhost:5432/pai?sslmode=disable
LEARN_DATABASE_MAX_CONNS=25

# --- Cache (Dragonfly/Redis) ---
LEARN_CACHE_URL=redis://localhost:6379

# --- NATS ---
LEARN_NATS_URL=nats://localhost:4222

# --- Telegram (Required) ---
LEARN_TELEGRAM_BOT_TOKEN=

# --- AI Providers (at least one required) ---
LEARN_AI_OPENAI_API_KEY=
LEARN_AI_ANTHROPIC_API_KEY=
LEARN_AI_OLLAMA_ENABLED=false
LEARN_AI_OLLAMA_URL=http://localhost:11434
LEARN_AI_OPENROUTER_API_KEY=

# --- Auth ---
LEARN_AUTH_JWT_SECRET=change-me-in-production

# --- Tenancy ---
LEARN_TENANT_MODE=single

# --- WhatsApp (Optional) ---
LEARN_WHATSAPP_ENABLED=false
LEARN_WHATSAPP_ACCESS_TOKEN=
LEARN_WHATSAPP_PHONE_ID=
LEARN_WHATSAPP_VERIFY_TOKEN=

# --- Logging ---
LEARN_LOG_LEVEL=info
LEARN_LOG_FORMAT=json
```

### Day 0 Validation

```bash
# Install dependencies
go mod tidy

# Run tests (must pass)
go test ./...

# Build binary
go build ./cmd/server

# Start infrastructure
docker compose up -d postgres dragonfly nats

# Run migration
docker exec -i $(docker compose ps -q postgres) psql -U pai pai < migrations/001_initial.up.sql

# Test health endpoint
go run ./cmd/server &
curl http://localhost:8080/healthz
kill %1

# Stop infrastructure
docker compose down
```

### Day 0 Exit Criteria

- [ ] `go.mod` exists with Go 1.22+
- [ ] Directory structure created: `cmd/`, `internal/`, `migrations/`, `deploy/`, `scripts/`
- [ ] `cmd/server/main.go` builds and returns health check 200
- [ ] `internal/platform/config/` — config loads from `LEARN_` env vars with tests
- [ ] `internal/platform/database/` — pgx pool wrapper with tests
- [ ] `internal/platform/cache/` — go-redis wrapper with tests
- [ ] `internal/ai/` — Provider interface, MockProvider, OpenAI, Anthropic, Ollama, Router with tests
- [ ] `docker-compose.yml` — Postgres 17, Dragonfly, NATS, app, optional Ollama
- [ ] `migrations/001_initial.up.sql` — tenants, users, conversations, messages, learning_progress, events
- [ ] `Makefile`, `.env.example`, `.github/workflows/ci.yml`
- [ ] `go test ./...` passes with zero failures

**Progress:** Foundation | 3 packages (config, database, cache, ai) | Docker Compose | CI

---

## WEEK 1 — THE TALKING SKELETON

### Day 1 (Mon) — Wire Telegram → AI → Student

**Entry criteria:** Day 0 complete. `go test ./...` passes. `docker compose up` starts infrastructure. Migration applied.

#### Tasks

| # | Task ID | Task | Owner | Files Created |
|---|---------|------|-------|---------------|
| 1.1 | `P-W1D1-1` | Chat Gateway: InboundMessage, OutboundMessage, Channel interface, Gateway router | 🤖 | `internal/chat/gateway.go` |
| 1.2 | `P-W1D1-2` | Telegram adapter: Bot API with long polling, /start handler, markdown splitting | 🤖 | `internal/chat/telegram.go` |
| 1.3 | `P-W1D1-3` | Agent Engine: ProcessMessage pipeline | 🤖 | `internal/agent/engine.go` |
| 1.4 | `P-W1D1-4` | Curriculum loader: load YAML topics + teaching notes from filesystem | 🤖 | `internal/curriculum/loader.go` |
| 1.5 | `P-W1D1-5` | Wire main.go: config → db → cache → AI → curriculum → agent → chat → start | 🤖 | Update `cmd/server/main.go` |

#### 1.1 — Chat Gateway (TDD)

**Step 1: Write tests**

**File:** `internal/chat/gateway_test.go`

```go
package chat_test

import (
	"context"
	"testing"

	"github.com/p-n-ai/pai-bot/internal/chat"
)

func TestNewGateway(t *testing.T) {
	gw := chat.NewGateway()
	if gw == nil {
		t.Fatal("NewGateway() returned nil")
	}
}

func TestGateway_RegisterChannel(t *testing.T) {
	gw := chat.NewGateway()
	mock := &chat.MockChannel{}

	gw.Register("telegram", mock)

	if !gw.HasChannel("telegram") {
		t.Error("HasChannel(telegram) should be true after Register")
	}
}

func TestGateway_SendMessage(t *testing.T) {
	gw := chat.NewGateway()
	mock := &chat.MockChannel{}
	gw.Register("telegram", mock)

	err := gw.Send(context.Background(), chat.OutboundMessage{
		Channel: "telegram",
		UserID:  "123",
		Text:    "Hello!",
	})
	if err != nil {
		t.Fatalf("Send() error = %v", err)
	}
	if len(mock.SentMessages) != 1 {
		t.Errorf("SentMessages = %d, want 1", len(mock.SentMessages))
	}
}

func TestInboundMessage_Fields(t *testing.T) {
	msg := chat.InboundMessage{
		Channel:    "telegram",
		UserID:     "123456",
		ExternalID: "tg_123456",
		Text:       "Hello bot",
		Username:   "testuser",
	}
	if msg.Channel != "telegram" {
		t.Errorf("Channel = %q, want telegram", msg.Channel)
	}
	if msg.UserID != "123456" {
		t.Errorf("UserID = %q, want 123456", msg.UserID)
	}
}
```

**Step 2: Implement**

**File:** `internal/chat/gateway.go`

```go
// Package chat provides a unified interface for messaging channels (Telegram, WhatsApp, WebSocket).
package chat

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
)

// InboundMessage is a message received from any channel.
type InboundMessage struct {
	Channel    string
	UserID     string
	ExternalID string
	Text       string
	Username   string
	FirstName  string
	LastName   string
	Language   string
}

// OutboundMessage is a message to send via any channel.
type OutboundMessage struct {
	Channel   string
	UserID    string
	Text      string
	ParseMode string // "Markdown", "HTML", or ""
}

// Channel is the interface each messaging platform must implement.
type Channel interface {
	SendMessage(ctx context.Context, userID string, msg OutboundMessage) error
	Start(ctx context.Context, handler func(InboundMessage)) error
	Stop() error
}

// Gateway routes messages to/from registered channels.
type Gateway struct {
	channels map[string]Channel
	mu       sync.RWMutex
}

// NewGateway creates a new chat gateway.
func NewGateway() *Gateway {
	return &Gateway{
		channels: make(map[string]Channel),
	}
}

// Register adds a channel to the gateway.
func (g *Gateway) Register(name string, ch Channel) {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.channels[name] = ch
	slog.Info("chat channel registered", "channel", name)
}

// HasChannel returns true if the named channel is registered.
func (g *Gateway) HasChannel(name string) bool {
	g.mu.RLock()
	defer g.mu.RUnlock()
	_, ok := g.channels[name]
	return ok
}

// Send dispatches a message to the appropriate channel.
func (g *Gateway) Send(ctx context.Context, msg OutboundMessage) error {
	g.mu.RLock()
	ch, ok := g.channels[msg.Channel]
	g.mu.RUnlock()

	if !ok {
		return fmt.Errorf("unknown channel: %s", msg.Channel)
	}

	return ch.SendMessage(ctx, msg.UserID, msg)
}

// StartAll starts all registered channels with the given message handler.
func (g *Gateway) StartAll(ctx context.Context, handler func(InboundMessage)) error {
	g.mu.RLock()
	defer g.mu.RUnlock()

	for name, ch := range g.channels {
		slog.Info("starting channel", "channel", name)
		if err := ch.Start(ctx, handler); err != nil {
			return fmt.Errorf("starting channel %s: %w", name, err)
		}
	}
	return nil
}

// MockChannel is a test double for Channel.
type MockChannel struct {
	SentMessages []OutboundMessage
}

func (m *MockChannel) SendMessage(_ context.Context, _ string, msg OutboundMessage) error {
	m.SentMessages = append(m.SentMessages, msg)
	return nil
}

func (m *MockChannel) Start(_ context.Context, _ func(InboundMessage)) error {
	return nil
}

func (m *MockChannel) Stop() error {
	return nil
}
```

#### 1.2 — Telegram Adapter (TDD)

**Step 1: Write tests**

**File:** `internal/chat/telegram_test.go`

```go
package chat_test

import (
	"testing"

	"github.com/p-n-ai/pai-bot/internal/chat"
)

func TestSplitMessage(t *testing.T) {
	tests := []struct {
		name      string
		text      string
		maxLen    int
		wantParts int
	}{
		{"short", "Hello", 4096, 1},
		{"exact", "Hello", 5, 1},
		{"split-needed", "Hello World, this is a test", 10, 3},
		{"empty", "", 4096, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parts := chat.SplitMessage(tt.text, tt.maxLen)
			if len(parts) != tt.wantParts {
				t.Errorf("SplitMessage() = %d parts, want %d", len(parts), tt.wantParts)
			}
		})
	}
}

func TestNewTelegramChannel_NoToken(t *testing.T) {
	_, err := chat.NewTelegramChannel("")
	if err == nil {
		t.Error("NewTelegramChannel() should error with empty token")
	}
}
```

**Step 2: Implement**

**File:** `internal/chat/telegram.go`

```go
package chat

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

const telegramMaxMessageLen = 4096

// TelegramChannel implements the Channel interface for Telegram Bot API.
type TelegramChannel struct {
	token   string
	baseURL string
	client  *http.Client
	offset  int
	stop    chan struct{}
}

// NewTelegramChannel creates a Telegram channel adapter.
func NewTelegramChannel(token string) (*TelegramChannel, error) {
	if token == "" {
		return nil, fmt.Errorf("Telegram bot token is required (LEARN_TELEGRAM_BOT_TOKEN)")
	}
	return &TelegramChannel{
		token:   token,
		baseURL: "https://api.telegram.org/bot" + token,
		client: &http.Client{
			Timeout: 60 * time.Second,
		},
		stop: make(chan struct{}),
	}, nil
}

func (t *TelegramChannel) SendMessage(ctx context.Context, userID string, msg OutboundMessage) error {
	parts := SplitMessage(msg.Text, telegramMaxMessageLen)

	for _, part := range parts {
		params := url.Values{
			"chat_id": {userID},
			"text":    {part},
		}
		if msg.ParseMode != "" {
			params.Set("parse_mode", msg.ParseMode)
		}

		resp, err := t.client.PostForm(t.baseURL+"/sendMessage", params)
		if err != nil {
			return fmt.Errorf("sending Telegram message: %w", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			body, _ := io.ReadAll(resp.Body)
			// If Markdown parsing fails, retry without parse mode
			if msg.ParseMode != "" && resp.StatusCode == http.StatusBadRequest {
				slog.Warn("Telegram markdown parse failed, retrying plain", "error", string(body))
				params.Del("parse_mode")
				retryResp, retryErr := t.client.PostForm(t.baseURL+"/sendMessage", params)
				if retryErr != nil {
					return fmt.Errorf("sending Telegram message (retry): %w", retryErr)
				}
				defer retryResp.Body.Close()
				if retryResp.StatusCode != http.StatusOK {
					retryBody, _ := io.ReadAll(retryResp.Body)
					return fmt.Errorf("Telegram API error %d: %s", retryResp.StatusCode, string(retryBody))
				}
				continue
			}
			return fmt.Errorf("Telegram API error %d: %s", resp.StatusCode, string(body))
		}
	}

	return nil
}

func (t *TelegramChannel) Start(ctx context.Context, handler func(InboundMessage)) error {
	go t.pollLoop(ctx, handler)
	return nil
}

func (t *TelegramChannel) Stop() error {
	close(t.stop)
	return nil
}

func (t *TelegramChannel) pollLoop(ctx context.Context, handler func(InboundMessage)) {
	slog.Info("Telegram long-polling started")
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.stop:
			return
		default:
			updates, err := t.getUpdates(ctx)
			if err != nil {
				slog.Error("Telegram getUpdates error", "error", err)
				time.Sleep(5 * time.Second)
				continue
			}

			for _, u := range updates {
				t.offset = u.UpdateID + 1
				if u.Message == nil || u.Message.Text == "" {
					continue
				}

				msg := InboundMessage{
					Channel:    "telegram",
					UserID:     strconv.FormatInt(u.Message.Chat.ID, 10),
					ExternalID: strconv.FormatInt(u.Message.From.ID, 10),
					Text:       u.Message.Text,
					Username:   u.Message.From.Username,
					FirstName:  u.Message.From.FirstName,
					LastName:   u.Message.From.LastName,
					Language:   u.Message.From.LanguageCode,
				}

				go handler(msg)
			}
		}
	}
}

func (t *TelegramChannel) getUpdates(ctx context.Context) ([]tgUpdate, error) {
	params := url.Values{
		"offset":  {strconv.Itoa(t.offset)},
		"timeout": {"30"},
	}

	req, err := http.NewRequestWithContext(ctx, "GET", t.baseURL+"/getUpdates?"+params.Encode(), nil)
	if err != nil {
		return nil, err
	}

	resp, err := t.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result struct {
		OK     bool       `json:"ok"`
		Result []tgUpdate `json:"result"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	if !result.OK {
		return nil, fmt.Errorf("Telegram API returned ok=false")
	}

	return result.Result, nil
}

// Telegram API types (minimal)
type tgUpdate struct {
	UpdateID int        `json:"update_id"`
	Message  *tgMessage `json:"message"`
}

type tgMessage struct {
	Text string `json:"text"`
	Chat tgChat `json:"chat"`
	From tgUser `json:"from"`
}

type tgChat struct {
	ID int64 `json:"id"`
}

type tgUser struct {
	ID           int64  `json:"id"`
	Username     string `json:"username"`
	FirstName    string `json:"first_name"`
	LastName     string `json:"last_name"`
	LanguageCode string `json:"language_code"`
}

// SplitMessage splits text into chunks that fit Telegram's max message length.
func SplitMessage(text string, maxLen int) []string {
	if text == "" {
		return nil
	}
	if len(text) <= maxLen {
		return []string{text}
	}

	var parts []string
	for len(text) > 0 {
		if len(text) <= maxLen {
			parts = append(parts, text)
			break
		}
		// Find last newline or space within limit
		cutAt := maxLen
		if idx := strings.LastIndex(text[:maxLen], "\n"); idx > 0 {
			cutAt = idx + 1
		} else if idx := strings.LastIndex(text[:maxLen], " "); idx > 0 {
			cutAt = idx + 1
		}
		parts = append(parts, text[:cutAt])
		text = text[cutAt:]
	}
	return parts
}
```

#### 1.3 — Agent Engine (TDD)

**Step 1: Write tests**

**File:** `internal/agent/engine_test.go`

```go
package agent_test

import (
	"context"
	"testing"

	"github.com/p-n-ai/pai-bot/internal/agent"
	"github.com/p-n-ai/pai-bot/internal/ai"
	"github.com/p-n-ai/pai-bot/internal/chat"
)

func TestEngine_ProcessMessage(t *testing.T) {
	mockAI := ai.NewMockProvider("This is the AI response about algebra.")

	engine := agent.NewEngine(agent.EngineConfig{
		AIRouter: mockRouter(mockAI),
	})

	resp, err := engine.ProcessMessage(context.Background(), chat.InboundMessage{
		Channel: "telegram",
		UserID:  "123",
		Text:    "What is algebra?",
	})
	if err != nil {
		t.Fatalf("ProcessMessage() error = %v", err)
	}
	if resp == "" {
		t.Error("ProcessMessage() returned empty response")
	}
}

func TestEngine_ProcessMessage_StartCommand(t *testing.T) {
	mockAI := ai.NewMockProvider("Welcome!")

	engine := agent.NewEngine(agent.EngineConfig{
		AIRouter: mockRouter(mockAI),
	})

	resp, err := engine.ProcessMessage(context.Background(), chat.InboundMessage{
		Channel: "telegram",
		UserID:  "123",
		Text:    "/start",
	})
	if err != nil {
		t.Fatalf("ProcessMessage() error = %v", err)
	}
	if resp == "" {
		t.Error("ProcessMessage() returned empty response for /start")
	}
}

// mockRouter creates an AI router with a single mock provider.
func mockRouter(provider ai.Provider) *ai.Router {
	r := ai.NewRouter()
	r.Register("mock", provider)
	return r
}
```

**Step 2: Implement**

**File:** `internal/agent/engine.go`

```go
// Package agent implements the conversation state machine and pedagogical engine.
package agent

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	"github.com/p-n-ai/pai-bot/internal/ai"
	"github.com/p-n-ai/pai-bot/internal/chat"
)

// EngineConfig holds dependencies for the agent engine.
type EngineConfig struct {
	AIRouter *ai.Router
}

// Engine is the core conversation processor.
type Engine struct {
	aiRouter *ai.Router
}

// NewEngine creates a new agent engine.
func NewEngine(cfg EngineConfig) *Engine {
	return &Engine{
		aiRouter: cfg.AIRouter,
	}
}

// ProcessMessage handles an incoming message and returns a response.
func (e *Engine) ProcessMessage(ctx context.Context, msg chat.InboundMessage) (string, error) {
	slog.Info("processing message",
		"channel", msg.Channel,
		"user_id", msg.UserID,
		"text_len", len(msg.Text),
	)

	// Handle commands
	if strings.HasPrefix(msg.Text, "/") {
		return e.handleCommand(ctx, msg)
	}

	// Build system prompt
	systemPrompt := e.buildSystemPrompt(msg)

	// Call AI
	resp, err := e.aiRouter.Complete(ctx, ai.CompletionRequest{
		Messages: []ai.Message{
			{Role: "system", Content: systemPrompt},
			{Role: "user", Content: msg.Text},
		},
		Task:      ai.TaskTeaching,
		MaxTokens: 1024,
	})
	if err != nil {
		slog.Error("AI completion failed", "error", err)
		return "Maaf, saya sedang mengalami masalah teknikal. Cuba lagi sebentar. 🔧", nil
	}

	return resp.Content, nil
}

func (e *Engine) handleCommand(ctx context.Context, msg chat.InboundMessage) (string, error) {
	cmd := strings.Split(msg.Text, " ")[0]

	switch cmd {
	case "/start":
		return e.handleStart(ctx, msg)
	default:
		return fmt.Sprintf("Arahan tidak diketahui: %s\nGuna /start untuk bermula.", cmd), nil
	}
}

func (e *Engine) handleStart(_ context.Context, msg chat.InboundMessage) (string, error) {
	name := msg.FirstName
	if name == "" {
		name = msg.Username
	}
	if name == "" {
		name = "pelajar"
	}

	return fmt.Sprintf(`Hai %s! 👋

Saya P&AI Bot — tutor matematik peribadi anda!

Saya boleh membantu anda dengan KSSM Matematik:
• Tingkatan 1
• Tingkatan 2
• Tingkatan 3

Apa yang anda ingin belajar hari ini?`, name), nil
}

func (e *Engine) buildSystemPrompt(msg chat.InboundMessage) string {
	return `You are P&AI Bot, a friendly and encouraging mathematics tutor for Malaysian secondary school students.

CURRICULUM: KSSM Matematik (Form 1, 2, 3) — focus on Algebra topics.

LANGUAGE: Respond in the same language the student uses. Most students use Bahasa Melayu or English. Mix both if the student does.

TEACHING STYLE:
- Start with what the student knows, build from there
- Use simple, relatable examples (Malaysian context: ringgit, kopitiam, school scenarios)
- Break complex problems into small steps
- Celebrate small wins ("Bagus!", "Betul!")
- If the student is stuck, give a hint before the answer
- Use mathematical notation where needed
- Keep responses concise — this is a chat, not a textbook

RULES:
- Never give answers without explanation
- Always check if the student understood before moving on
- If unsure of the student's level, ask a diagnostic question
- Be patient and never condescending`
}
```

#### 1.4 — Curriculum Loader (TDD)

**Step 1: Write tests**

**File:** `internal/curriculum/loader_test.go`

```go
package curriculum_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/p-n-ai/pai-bot/internal/curriculum"
)

func TestLoader_LoadTopics(t *testing.T) {
	dir := setupTestCurriculum(t)

	loader, err := curriculum.NewLoader(dir)
	if err != nil {
		t.Fatalf("NewLoader() error = %v", err)
	}

	topics := loader.AllTopics()
	if len(topics) == 0 {
		t.Error("AllTopics() returned empty")
	}
}

func TestLoader_GetTopic(t *testing.T) {
	dir := setupTestCurriculum(t)

	loader, err := curriculum.NewLoader(dir)
	if err != nil {
		t.Fatalf("NewLoader() error = %v", err)
	}

	topic, found := loader.GetTopic("F1-01")
	if !found {
		t.Error("GetTopic(F1-01) not found")
	}
	if topic.Name == "" {
		t.Error("Topic.Name is empty")
	}
}

func TestLoader_GetTopic_NotFound(t *testing.T) {
	dir := setupTestCurriculum(t)

	loader, err := curriculum.NewLoader(dir)
	if err != nil {
		t.Fatalf("NewLoader() error = %v", err)
	}

	_, found := loader.GetTopic("NONEXISTENT")
	if found {
		t.Error("GetTopic(NONEXISTENT) should not be found")
	}
}

func TestLoader_GetTeachingNotes(t *testing.T) {
	dir := setupTestCurriculum(t)

	loader, err := curriculum.NewLoader(dir)
	if err != nil {
		t.Fatalf("NewLoader() error = %v", err)
	}

	notes, found := loader.GetTeachingNotes("F1-01")
	if !found {
		t.Error("GetTeachingNotes(F1-01) not found")
	}
	if notes == "" {
		t.Error("Teaching notes is empty")
	}
}

func setupTestCurriculum(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()

	topicsDir := filepath.Join(dir, "curricula", "malaysia", "kssm", "topics", "algebra")
	os.MkdirAll(topicsDir, 0o755)

	// Topic YAML
	os.WriteFile(filepath.Join(topicsDir, "01-variables.yaml"), []byte(`
id: F1-01
name: "Variables & Algebraic Expressions"
subject_id: algebra
syllabus_id: malaysia-kssm-matematik-tingkatan1
difficulty: beginner
learning_objectives:
  - id: LO1
    text: "Use letters to represent unknown quantities"
    bloom: remember
  - id: LO2
    text: "Write algebraic expressions from word problems"
    bloom: apply
prerequisites:
  required: []
quality_level: 1
provenance: human
`), 0o644)

	// Teaching notes markdown
	os.WriteFile(filepath.Join(topicsDir, "01-variables.teaching.md"), []byte(`# Variables & Algebraic Expressions — Teaching Notes

## Overview
This topic introduces the concept of using letters to represent unknown values.

## Teaching Sequence
1. Start with a guessing game (15 min)
2. Introduce variables as "mystery numbers" (10 min)
3. Practice writing expressions (20 min)

## Common Misconceptions
| Misconception | Remediation |
|---|---|
| 3x means "3 and x" not "3 times x" | Use multiplication sign explicitly first |
`), 0o644)

	return dir
}
```

**Step 2: Implement**

**File:** `internal/curriculum/loader.go`

```go
// Package curriculum loads and caches curriculum content from the OSS repository.
package curriculum

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"gopkg.in/yaml.v3"
)

// Topic represents a curriculum topic loaded from YAML.
type Topic struct {
	ID                 string              `yaml:"id"`
	Name               string              `yaml:"name"`
	SubjectID          string              `yaml:"subject_id"`
	SyllabusID         string              `yaml:"syllabus_id"`
	Difficulty         string              `yaml:"difficulty"`
	LearningObjectives []LearningObjective `yaml:"learning_objectives"`
	Prerequisites      Prerequisites       `yaml:"prerequisites"`
	QualityLevel       int                 `yaml:"quality_level"`
	Provenance         string              `yaml:"provenance"`
}

// LearningObjective represents a learning objective within a topic.
type LearningObjective struct {
	ID    string `yaml:"id"`
	Text  string `yaml:"text"`
	Bloom string `yaml:"bloom"`
}

// Prerequisites holds required and recommended prerequisites.
type Prerequisites struct {
	Required    []string `yaml:"required"`
	Recommended []string `yaml:"recommended"`
}

// Loader loads and caches curriculum content from the filesystem.
type Loader struct {
	rootDir       string
	topics        map[string]Topic
	teachingNotes map[string]string
	mu            sync.RWMutex
}

// NewLoader creates a new curriculum loader and loads all content.
func NewLoader(rootDir string) (*Loader, error) {
	l := &Loader{
		rootDir:       rootDir,
		topics:        make(map[string]Topic),
		teachingNotes: make(map[string]string),
	}

	if err := l.loadAll(); err != nil {
		return nil, fmt.Errorf("loading curriculum: %w", err)
	}

	slog.Info("curriculum loaded", "topics", len(l.topics))
	return l, nil
}

// GetTopic returns a topic by ID.
func (l *Loader) GetTopic(id string) (Topic, bool) {
	l.mu.RLock()
	defer l.mu.RUnlock()
	t, ok := l.topics[id]
	return t, ok
}

// GetTeachingNotes returns teaching notes for a topic ID.
func (l *Loader) GetTeachingNotes(id string) (string, bool) {
	l.mu.RLock()
	defer l.mu.RUnlock()
	n, ok := l.teachingNotes[id]
	return n, ok
}

// AllTopics returns all loaded topics.
func (l *Loader) AllTopics() []Topic {
	l.mu.RLock()
	defer l.mu.RUnlock()
	topics := make([]Topic, 0, len(l.topics))
	for _, t := range l.topics {
		topics = append(topics, t)
	}
	return topics
}

func (l *Loader) loadAll() error {
	return filepath.Walk(l.rootDir, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return nil
		}

		switch {
		case strings.HasSuffix(path, ".teaching.md"):
			return l.loadTeachingNotes(path)
		case strings.HasSuffix(path, ".yaml") || strings.HasSuffix(path, ".yml"):
			if strings.HasSuffix(path, ".assessments.yaml") || strings.HasSuffix(path, ".examples.yaml") {
				return nil // Skip non-topic YAML
			}
			return l.loadTopic(path)
		}
		return nil
	})
}

func (l *Loader) loadTopic(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	var topic Topic
	if err := yaml.Unmarshal(data, &topic); err != nil {
		slog.Warn("skipping invalid topic YAML", "path", path, "error", err)
		return nil
	}

	if topic.ID == "" {
		return nil // Not a topic file
	}

	l.mu.Lock()
	l.topics[topic.ID] = topic
	l.mu.Unlock()

	return nil
}

func (l *Loader) loadTeachingNotes(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	// Derive topic ID from matching YAML file
	yamlPath := strings.TrimSuffix(path, ".teaching.md") + ".yaml"
	yamlData, err := os.ReadFile(yamlPath)
	if err != nil {
		return nil // No matching YAML, skip
	}

	var partial struct {
		ID string `yaml:"id"`
	}
	if err := yaml.Unmarshal(yamlData, &partial); err != nil || partial.ID == "" {
		return nil
	}

	l.mu.Lock()
	l.teachingNotes[partial.ID] = string(data)
	l.mu.Unlock()

	return nil
}
```

**File:** `internal/curriculum/types.go`

```go
package curriculum

// Syllabus represents a top-level syllabus (e.g., KSSM Matematik Tingkatan 1).
type Syllabus struct {
	ID       string    `yaml:"id"`
	Name     string    `yaml:"name"`
	Country  string    `yaml:"country"`
	Board    string    `yaml:"board"`
	Level    string    `yaml:"level"`
	Subjects []Subject `yaml:"subjects"`
}

// Subject represents a subject within a syllabus (e.g., Algebra).
type Subject struct {
	ID       string   `yaml:"id"`
	Name     string   `yaml:"name"`
	TopicIDs []string `yaml:"topic_ids"`
}
```

#### 1.5 — Wire Everything in main.go

Update `cmd/server/main.go` to wire all components together. The entrypoint should:

1. Load config
2. Connect to DB, cache (log warnings if unavailable — don't fail)
3. Initialize AI providers based on config
4. Load curriculum from filesystem
5. Create agent engine
6. Create Telegram channel + chat gateway
7. Start long-polling
8. Listen for HTTP (health check)
9. Graceful shutdown

#### Day 1 Validation

```bash
# Run all tests
make test-all

# Build and verify
go build ./cmd/server

# Verify with real bot (requires LEARN_TELEGRAM_BOT_TOKEN set)
# LEARN_TELEGRAM_BOT_TOKEN=<token> LEARN_AI_OLLAMA_ENABLED=true go run ./cmd/server
```

#### Day 1 Exit Criteria

- [ ] `internal/chat/gateway.go` + tests — unified message routing
- [ ] `internal/chat/telegram.go` + tests — Telegram long-polling adapter
- [ ] `internal/agent/engine.go` + tests — ProcessMessage pipeline with /start handler
- [ ] `internal/curriculum/loader.go` + tests — loads YAML topics + teaching notes
- [ ] `cmd/server/main.go` wires everything together
- [ ] Team members can chat with the bot on Telegram. AI responds using curriculum context.
- [ ] `make test-all` passes with zero failures

**Progress:** Foundation + chat + agent + curriculum | 7 packages | Bot responds on Telegram

---

### Day 2 (Tue) — Logging + Quality

**Entry criteria:** Day 1 complete. Bot responds on Telegram. `make test-all` passes.

#### Tasks

| # | Task ID | Task | Owner | Files Created |
|---|---------|------|-------|---------------|
| 2.1 | `P-W1D2-1` | Message persistence: save every exchange to `messages` table | 🤖 | `internal/agent/store.go` |
| 2.2 | `P-W1D2-2` | Event logging: log session_started, message_sent, ai_response (non-blocking) | 🤖 | `internal/agent/events.go` |
| 2.3 | `P-W1D2-3` | Topic detection: keyword scan → load matching teaching notes into system prompt | 🤖 | `internal/agent/topics.go` |
| 2.4 | `P-W1D2-4` | Update AI router: add task-based routing (teaching → best, grading → cheapest) | 🤖 | Update `internal/ai/router.go` |
| 2.5 | `P-W1D2-5` | 🧑 Test 30 conversation scenarios, log bad responses, rewrite system prompt v2 | 🧑 | Manual |

#### 2.1 — Message Persistence (TDD)

**Step 1: Write tests**

**File:** `internal/agent/store_test.go`

```go
package agent_test

import (
	"testing"

	"github.com/p-n-ai/pai-bot/internal/agent"
)

func TestConversationStore_Interface(t *testing.T) {
	store := agent.NewMemoryStore()

	// Save a conversation
	conv := agent.Conversation{
		UserID:   "123",
		TopicID:  "F1-01",
		State:    "teaching",
		Messages: []agent.StoredMessage{},
	}

	id, err := store.CreateConversation(conv)
	if err != nil {
		t.Fatalf("CreateConversation() error = %v", err)
	}
	if id == "" {
		t.Error("CreateConversation() returned empty ID")
	}

	// Add a message
	err = store.AddMessage(id, agent.StoredMessage{
		Role:    "user",
		Content: "What is algebra?",
	})
	if err != nil {
		t.Fatalf("AddMessage() error = %v", err)
	}

	// Get conversation
	got, err := store.GetConversation(id)
	if err != nil {
		t.Fatalf("GetConversation() error = %v", err)
	}
	if len(got.Messages) != 1 {
		t.Errorf("Messages count = %d, want 1", len(got.Messages))
	}
}

func TestConversationStore_GetActiveForUser(t *testing.T) {
	store := agent.NewMemoryStore()

	conv := agent.Conversation{
		UserID: "123",
		State:  "teaching",
	}
	_, err := store.CreateConversation(conv)
	if err != nil {
		t.Fatalf("CreateConversation() error = %v", err)
	}

	active, found := store.GetActiveConversation("123")
	if !found {
		t.Error("GetActiveConversation() should find active conversation")
	}
	if active.UserID != "123" {
		t.Errorf("UserID = %q, want 123", active.UserID)
	}
}
```

**Step 2: Implement**

**File:** `internal/agent/store.go`

```go
package agent

import (
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
)

// Conversation represents a chat conversation.
type Conversation struct {
	ID        string
	UserID    string
	TopicID   string
	State     string // idle, teaching, quizzing, reviewing
	Messages  []StoredMessage
	StartedAt time.Time
	EndedAt   *time.Time
}

// StoredMessage represents a persisted message.
type StoredMessage struct {
	Role         string
	Content      string
	Model        string
	InputTokens  int
	OutputTokens int
	CreatedAt    time.Time
}

// ConversationStore is the interface for conversation persistence.
type ConversationStore interface {
	CreateConversation(conv Conversation) (string, error)
	GetConversation(id string) (*Conversation, error)
	GetActiveConversation(userID string) (*Conversation, bool)
	AddMessage(convID string, msg StoredMessage) error
	EndConversation(id string) error
}

// MemoryStore is an in-memory ConversationStore for development/testing.
type MemoryStore struct {
	conversations map[string]*Conversation
	mu            sync.RWMutex
}

// NewMemoryStore creates an in-memory store.
func NewMemoryStore() *MemoryStore {
	return &MemoryStore{
		conversations: make(map[string]*Conversation),
	}
}

func (s *MemoryStore) CreateConversation(conv Conversation) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	conv.ID = uuid.New().String()
	conv.StartedAt = time.Now()
	s.conversations[conv.ID] = &conv
	return conv.ID, nil
}

func (s *MemoryStore) GetConversation(id string) (*Conversation, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	conv, ok := s.conversations[id]
	if !ok {
		return nil, fmt.Errorf("conversation %s not found", id)
	}
	return conv, nil
}

func (s *MemoryStore) GetActiveConversation(userID string) (*Conversation, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	for _, conv := range s.conversations {
		if conv.UserID == userID && conv.EndedAt == nil {
			return conv, true
		}
	}
	return nil, false
}

func (s *MemoryStore) AddMessage(convID string, msg StoredMessage) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	conv, ok := s.conversations[convID]
	if !ok {
		return fmt.Errorf("conversation %s not found", convID)
	}

	msg.CreatedAt = time.Now()
	conv.Messages = append(conv.Messages, msg)
	return nil
}

func (s *MemoryStore) EndConversation(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	conv, ok := s.conversations[id]
	if !ok {
		return fmt.Errorf("conversation %s not found", id)
	}

	now := time.Now()
	conv.EndedAt = &now
	return nil
}
```

#### 2.2 — Event Logger (TDD)

**File:** `internal/agent/events_test.go`

```go
package agent_test

import (
	"testing"

	"github.com/p-n-ai/pai-bot/internal/agent"
)

func TestEventLogger_LogEvent(t *testing.T) {
	logger := agent.NewMemoryEventLogger()

	logger.LogEvent(agent.Event{
		UserID:    "123",
		EventType: "message_sent",
		Data:      map[string]interface{}{"text_len": 42},
	})

	events := logger.Events()
	if len(events) != 1 {
		t.Errorf("Events() = %d, want 1", len(events))
	}
	if events[0].EventType != "message_sent" {
		t.Errorf("EventType = %q, want message_sent", events[0].EventType)
	}
}
```

**File:** `internal/agent/events.go`

```go
package agent

import (
	"log/slog"
	"sync"
	"time"
)

// Event represents an analytics/audit event.
type Event struct {
	UserID    string
	EventType string
	Data      map[string]interface{}
	CreatedAt time.Time
}

// EventLogger is the interface for event logging.
type EventLogger interface {
	LogEvent(event Event)
}

// MemoryEventLogger stores events in memory (for dev/testing).
type MemoryEventLogger struct {
	events []Event
	mu     sync.Mutex
}

func NewMemoryEventLogger() *MemoryEventLogger {
	return &MemoryEventLogger{}
}

func (l *MemoryEventLogger) LogEvent(event Event) {
	event.CreatedAt = time.Now()
	l.mu.Lock()
	l.events = append(l.events, event)
	l.mu.Unlock()

	slog.Info("event logged",
		"type", event.EventType,
		"user_id", event.UserID,
	)
}

func (l *MemoryEventLogger) Events() []Event {
	l.mu.Lock()
	defer l.mu.Unlock()
	return append([]Event{}, l.events...)
}
```

#### 2.3 — Topic Detection (TDD)

**File:** `internal/agent/topics_test.go`

```go
package agent_test

import (
	"testing"

	"github.com/p-n-ai/pai-bot/internal/agent"
	"github.com/p-n-ai/pai-bot/internal/curriculum"
)

func TestDetectTopic(t *testing.T) {
	topics := []curriculum.Topic{
		{ID: "F1-01", Name: "Variables & Algebraic Expressions"},
		{ID: "F1-02", Name: "Linear Equations"},
		{ID: "F2-01", Name: "Quadratic Expressions"},
	}

	tests := []struct {
		name    string
		text    string
		wantID  string
		wantOK  bool
	}{
		{"algebra-keyword", "I want to learn about variables", "F1-01", true},
		{"equation-keyword", "Help me solve linear equations", "F1-02", true},
		{"quadratic", "What is quadratic expression?", "F2-01", true},
		{"no-match", "Hello there", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			id, ok := agent.DetectTopic(tt.text, topics)
			if ok != tt.wantOK {
				t.Errorf("DetectTopic() ok = %v, want %v", ok, tt.wantOK)
			}
			if ok && id != tt.wantID {
				t.Errorf("DetectTopic() id = %q, want %q", id, tt.wantID)
			}
		})
	}
}
```

**File:** `internal/agent/topics.go`

```go
package agent

import (
	"strings"

	"github.com/p-n-ai/pai-bot/internal/curriculum"
)

// DetectTopic scans user text for keywords matching loaded topics.
// Returns the best matching topic ID and true, or empty string and false.
func DetectTopic(text string, topics []curriculum.Topic) (string, bool) {
	lower := strings.ToLower(text)

	bestID := ""
	bestScore := 0

	for _, topic := range topics {
		score := 0
		// Check topic name words
		nameWords := strings.Fields(strings.ToLower(topic.Name))
		for _, word := range nameWords {
			if len(word) >= 3 && strings.Contains(lower, word) {
				score++
			}
		}
		// Check learning objective text
		for _, lo := range topic.LearningObjectives {
			loWords := strings.Fields(strings.ToLower(lo.Text))
			for _, word := range loWords {
				if len(word) >= 4 && strings.Contains(lower, word) {
					score++
				}
			}
		}

		if score > bestScore {
			bestScore = score
			bestID = topic.ID
		}
	}

	if bestScore >= 1 {
		return bestID, true
	}
	return "", false
}
```

#### Day 2 Validation

```bash
make test-all
```

#### Day 2 Exit Criteria

- [ ] `internal/agent/store.go` + tests — ConversationStore with MemoryStore implementation
- [ ] `internal/agent/events.go` + tests — EventLogger with MemoryEventLogger
- [ ] `internal/agent/topics.go` + tests — keyword-based topic detection
- [ ] AI router updated with task-based routing preferences
- [ ] System prompt includes curriculum context when topic is detected
- [ ] 🧑 Human tested 30 conversation scenarios, system prompt v2 applied
- [ ] `make test-all` passes with zero failures

**Progress:** Foundation + chat + agent (engine, store, events, topics) + curriculum + ai | 7 packages

---

### Day 3 (Wed) — Deploy + First Students

**Entry criteria:** Day 2 complete. Bot responds with curriculum context. `make test-all` passes.

#### Tasks

| # | Task ID | Task | Owner | Files Created |
|---|---------|------|-------|---------------|
| 3.1 | `P-W1D3-1` | Deploy script: SSH → pull → build → restart → tail logs | 🤖 | `scripts/deploy.sh` |
| 3.2 | `P-W1D3-2` | `/start` onboarding: create user record, welcome message, ask what to study | 🤖 | Update `internal/agent/engine.go` |
| 3.3 | `P-W1D3-3` | User lookup by telegram_id in chat flow, auto-trigger /start if new | 🤖 | `internal/agent/users.go` |
| 3.4 | `P-W1D3-4` | Error recovery: retry with backoff, provider fallback, friendly error messages | 🤖 | Update `internal/ai/router.go` |
| 3.5 | `P-W1D3-5` | 🧑 Deploy to AWS (t3.medium), onboard first 3 pilot students | 🧑 | Manual |

#### 3.1 — Deploy Script

**File:** `scripts/deploy.sh`

```bash
#!/usr/bin/env bash
set -euo pipefail

SERVER="${DEPLOY_HOST:-your-server}"
USER="${DEPLOY_USER:-ubuntu}"
APP_DIR="${DEPLOY_DIR:-/opt/pai-bot}"

echo "=== Deploying P&AI Bot to $SERVER ==="

ssh "$USER@$SERVER" << 'REMOTE'
set -euo pipefail
cd /opt/pai-bot
git pull origin main
docker compose build app
docker compose up -d app
echo "=== Deploy complete ==="
docker compose logs --tail=20 app
REMOTE
```

#### 3.3 — User Management (TDD)

**File:** `internal/agent/users_test.go`

```go
package agent_test

import (
	"testing"

	"github.com/p-n-ai/pai-bot/internal/agent"
)

func TestUserStore_CreateAndGet(t *testing.T) {
	store := agent.NewMemoryUserStore()

	user := agent.User{
		ExternalID: "tg_123456",
		Channel:    "telegram",
		Name:       "Ali",
		Role:       "student",
	}

	id, err := store.CreateUser(user)
	if err != nil {
		t.Fatalf("CreateUser() error = %v", err)
	}
	if id == "" {
		t.Error("CreateUser() returned empty ID")
	}

	got, found := store.GetByExternalID("tg_123456")
	if !found {
		t.Error("GetByExternalID() should find user")
	}
	if got.Name != "Ali" {
		t.Errorf("Name = %q, want Ali", got.Name)
	}
}

func TestUserStore_GetByExternalID_NotFound(t *testing.T) {
	store := agent.NewMemoryUserStore()

	_, found := store.GetByExternalID("nonexistent")
	if found {
		t.Error("GetByExternalID() should not find non-existent user")
	}
}
```

**File:** `internal/agent/users.go`

```go
package agent

import (
	"sync"
	"time"

	"github.com/google/uuid"
)

// User represents a registered user.
type User struct {
	ID         string
	TenantID   string
	ExternalID string
	Channel    string
	Name       string
	Role       string
	Form       string
	CreatedAt  time.Time
}

// UserStore is the interface for user persistence.
type UserStore interface {
	CreateUser(user User) (string, error)
	GetByExternalID(externalID string) (*User, bool)
}

// MemoryUserStore is an in-memory UserStore for development/testing.
type MemoryUserStore struct {
	users map[string]*User
	byExt map[string]string // external_id -> user_id
	mu    sync.RWMutex
}

func NewMemoryUserStore() *MemoryUserStore {
	return &MemoryUserStore{
		users: make(map[string]*User),
		byExt: make(map[string]string),
	}
}

func (s *MemoryUserStore) CreateUser(user User) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	user.ID = uuid.New().String()
	user.CreatedAt = time.Now()
	s.users[user.ID] = &user
	s.byExt[user.ExternalID] = user.ID
	return user.ID, nil
}

func (s *MemoryUserStore) GetByExternalID(externalID string) (*User, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	id, ok := s.byExt[externalID]
	if !ok {
		return nil, false
	}
	user, ok := s.users[id]
	return user, ok
}
```

#### 3.4 — Error Recovery with Retry + Backoff

Update `internal/ai/router.go` to add retry logic with exponential backoff and circuit breaker pattern. Key behavior:

- Retry up to 3 times with exponential backoff (1s, 2s, 4s)
- On provider failure, try next provider in fallback chain
- Never let a student see a raw error — return a friendly message in BM/English
- Log all failures for debugging

#### Day 3 Exit Criteria

- [ ] `scripts/deploy.sh` — automated deployment script
- [ ] `/start` creates user record, sends welcome message with form selection
- [ ] Auto-lookup user by telegram_id on every message, auto-trigger /start for new users
- [ ] AI router retries with backoff, falls back through provider chain
- [ ] 🧑 Deployed to AWS, 3 pilot students onboarded and chatting
- [ ] `make test-all` passes

---

### Day 4 (Thu) — Iterate on Real Feedback

**Entry criteria:** Day 3 complete. Bot deployed. 3+ students chatting.

#### Tasks

| # | Task ID | Task | Owner | Files Created |
|---|---------|------|-------|---------------|
| 4.1 | `P-W1D4-1` | Analytics script: DAU, messages/session, AI latency, tokens by model | 🤖 | `scripts/analytics.sh` |
| 4.2 | `P-W1D4-2` | Session management: new conversation after 30min silence, summarize previous | 🤖 | Update `internal/agent/engine.go` |
| 4.3 | `P-W1D4-3` | In-chat rating: after every 5th response ask 1-5 rating, log as event | 🤖 | Update `internal/agent/engine.go` |
| 4.4 | `P-W1D4-4` | 🧑 Read ALL pilot conversations, categorize issues, rewrite system prompt v3 | 🧑 | Manual |
| 4.5 | `P-W1D4-5` | 🧑 Onboard remaining 7 pilot students (total 10) | 🧑 | Manual |

#### 4.2 — Session Management (TDD)

Add session timeout logic to the engine:

- If >30 minutes since last message, create a new conversation
- Summarize the previous session's last few messages for context continuity
- Store session summary in new conversation's metadata

**File:** `internal/agent/session_test.go`

```go
package agent_test

import (
	"testing"
	"time"

	"github.com/p-n-ai/pai-bot/internal/agent"
)

func TestShouldStartNewSession(t *testing.T) {
	tests := []struct {
		name       string
		lastMsg    time.Time
		timeout    time.Duration
		wantNew    bool
	}{
		{"recent", time.Now().Add(-5 * time.Minute), 30 * time.Minute, false},
		{"expired", time.Now().Add(-45 * time.Minute), 30 * time.Minute, true},
		{"exact", time.Now().Add(-30 * time.Minute), 30 * time.Minute, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := agent.ShouldStartNewSession(tt.lastMsg, tt.timeout)
			if got != tt.wantNew {
				t.Errorf("ShouldStartNewSession() = %v, want %v", got, tt.wantNew)
			}
		})
	}
}

func TestSummarizeMessages(t *testing.T) {
	messages := []agent.StoredMessage{
		{Role: "user", Content: "What is algebra?"},
		{Role: "assistant", Content: "Algebra uses letters to represent numbers."},
		{Role: "user", Content: "Give me an example"},
		{Role: "assistant", Content: "For example, 2x + 3 = 7"},
	}

	summary := agent.SummarizeMessages(messages, 3)
	if summary == "" {
		t.Error("SummarizeMessages() returned empty")
	}
}
```

**File:** `internal/agent/session.go`

```go
package agent

import (
	"fmt"
	"strings"
	"time"
)

const SessionTimeout = 30 * time.Minute

// ShouldStartNewSession returns true if enough time has passed since the last message.
func ShouldStartNewSession(lastMessageAt time.Time, timeout time.Duration) bool {
	return time.Since(lastMessageAt) >= timeout
}

// SummarizeMessages creates a brief summary of recent messages for context continuity.
func SummarizeMessages(messages []StoredMessage, maxMessages int) string {
	if len(messages) == 0 {
		return ""
	}

	start := 0
	if len(messages) > maxMessages {
		start = len(messages) - maxMessages
	}

	var sb strings.Builder
	sb.WriteString("Previous session summary:\n")
	for _, msg := range messages[start:] {
		role := "Student"
		if msg.Role == "assistant" {
			role = "Tutor"
		}
		content := msg.Content
		if len(content) > 100 {
			content = content[:100] + "..."
		}
		sb.WriteString(fmt.Sprintf("- %s: %s\n", role, content))
	}
	return sb.String()
}
```

#### Day 4 Exit Criteria

- [ ] `scripts/analytics.sh` — queries events table for key metrics
- [ ] Session auto-expires after 30min, new session gets context summary
- [ ] In-chat rating prompt every 5th response, logged as event
- [ ] 🧑 System prompt v3 applied based on pilot conversation review
- [ ] 🧑 10 pilot students onboarded
- [ ] `make test-all` passes

---

### Day 5 (Fri) — Week 1 Retro

**Entry criteria:** Day 4 complete. 10 students onboarded. Analytics available.

#### Tasks

| # | Task ID | Task | Owner |
|---|---------|------|-------|
| 5.1 | `P-W1D5-1` | 🧑 Run analytics, compile Week 1 numbers | 🧑 |
| 5.2 | `P-W1D5-2` | 🧑 1hr retro: demo, review conversations, identify top 3 problems | 🧑 |
| 5.3 | `P-W1D5-3` | 🧑 Call top 3 and bottom 3 students — 10min each | 🧑 |

#### Week 1 Targets

- [ ] 10 students used bot
- [ ] ≥7 returned for a second session
- [ ] Average session ≥6 messages
- [ ] System prompt at v3+

**Week 1 Progress:** 7 packages (config, database, cache, ai, chat, agent, curriculum) | Bot live on Telegram | 10 students

---

## WEEK 2 — PROGRESS + ASSESSMENT + 50 STUDENTS

### Day 6 (Mon) — Mastery Tracking

**Entry criteria:** Week 1 complete. Bot live with 10 students. System prompt v3+. `make test-all` passes.

#### Tasks

| # | Task ID | Task | Owner | Files Created |
|---|---------|------|-------|---------------|
| 6.1 | `P-W2D6-1` | Progress tracking: AI call after exchange to assess mastery_delta | 🤖 | `internal/progress/tracker.go` |
| 6.2 | `P-W2D6-2` | SM-2 spaced repetition scheduler | 🤖 | `internal/progress/spaced_rep.go` |
| 6.3 | `P-W2D6-3` | `/progress` command: Unicode bars per topic, XP, streak, next review | 🤖 | `internal/progress/display.go` |
| 6.4 | `P-W2D6-4` | Progress context in system prompt | 🤖 | Update `internal/agent/engine.go` |
| 6.5 | `P-W2D6-5` | 🧑 Recruit 40 more students from Pandai | 🧑 | Manual |

#### 6.1 — Progress Tracker (TDD)

**Step 1: Write tests**

**File:** `internal/progress/tracker_test.go`

```go
package progress_test

import (
	"testing"

	"github.com/p-n-ai/pai-bot/internal/progress"
)

func TestTracker_UpdateMastery(t *testing.T) {
	tracker := progress.NewMemoryTracker()

	err := tracker.UpdateMastery("user1", "syllabus1", "F1-01", 0.8)
	if err != nil {
		t.Fatalf("UpdateMastery() error = %v", err)
	}

	score, err := tracker.GetMastery("user1", "syllabus1", "F1-01")
	if err != nil {
		t.Fatalf("GetMastery() error = %v", err)
	}
	if score < 0.7 || score > 0.9 {
		t.Errorf("Mastery score = %f, expected around 0.8", score)
	}
}

func TestTracker_GetAllProgress(t *testing.T) {
	tracker := progress.NewMemoryTracker()

	tracker.UpdateMastery("user1", "s1", "F1-01", 0.5)
	tracker.UpdateMastery("user1", "s1", "F1-02", 0.9)

	items, err := tracker.GetAllProgress("user1")
	if err != nil {
		t.Fatalf("GetAllProgress() error = %v", err)
	}
	if len(items) != 2 {
		t.Errorf("GetAllProgress() = %d items, want 2", len(items))
	}
}

func TestMasteryThreshold(t *testing.T) {
	if progress.IsMastered(0.74) {
		t.Error("0.74 should not be mastered (threshold 0.75)")
	}
	if !progress.IsMastered(0.75) {
		t.Error("0.75 should be mastered")
	}
}
```

**Step 2: Implement**

**File:** `internal/progress/tracker.go`

```go
// Package progress implements mastery scoring, spaced repetition, and streaks/XP.
package progress

import (
	"fmt"
	"sync"
	"time"
)

const MasteryThreshold = 0.75

// ProgressItem represents a student's progress on a single topic.
type ProgressItem struct {
	UserID       string
	SyllabusID   string
	TopicID      string
	MasteryScore float64
	EaseFactor   float64
	IntervalDays int
	Repetitions  int
	NextReviewAt *time.Time
	LastStudied  *time.Time
}

// Tracker is the interface for progress tracking.
type Tracker interface {
	UpdateMastery(userID, syllabusID, topicID string, delta float64) error
	GetMastery(userID, syllabusID, topicID string) (float64, error)
	GetAllProgress(userID string) ([]ProgressItem, error)
	GetDueReviews(userID string) ([]ProgressItem, error)
}

// IsMastered returns true if the score meets the mastery threshold.
func IsMastered(score float64) bool {
	return score >= MasteryThreshold
}

// MemoryTracker is an in-memory Tracker for development/testing.
type MemoryTracker struct {
	items map[string]*ProgressItem // key: userID|syllabusID|topicID
	mu    sync.RWMutex
}

func NewMemoryTracker() *MemoryTracker {
	return &MemoryTracker{
		items: make(map[string]*ProgressItem),
	}
}

func (t *MemoryTracker) key(userID, syllabusID, topicID string) string {
	return fmt.Sprintf("%s|%s|%s", userID, syllabusID, topicID)
}

func (t *MemoryTracker) UpdateMastery(userID, syllabusID, topicID string, delta float64) error {
	t.mu.Lock()
	defer t.mu.Unlock()

	k := t.key(userID, syllabusID, topicID)
	item, ok := t.items[k]
	if !ok {
		now := time.Now()
		item = &ProgressItem{
			UserID:       userID,
			SyllabusID:   syllabusID,
			TopicID:      topicID,
			MasteryScore: 0,
			EaseFactor:   2.5,
			IntervalDays: 1,
			Repetitions:  0,
			LastStudied:  &now,
		}
		t.items[k] = item
	}

	// Update mastery — weighted blend of existing + new signal
	item.MasteryScore = clamp(item.MasteryScore*0.7+delta*0.3, 0, 1)
	now := time.Now()
	item.LastStudied = &now

	return nil
}

func (t *MemoryTracker) GetMastery(userID, syllabusID, topicID string) (float64, error) {
	t.mu.RLock()
	defer t.mu.RUnlock()

	k := t.key(userID, syllabusID, topicID)
	item, ok := t.items[k]
	if !ok {
		return 0, nil
	}
	return item.MasteryScore, nil
}

func (t *MemoryTracker) GetAllProgress(userID string) ([]ProgressItem, error) {
	t.mu.RLock()
	defer t.mu.RUnlock()

	var items []ProgressItem
	for _, item := range t.items {
		if item.UserID == userID {
			items = append(items, *item)
		}
	}
	return items, nil
}

func (t *MemoryTracker) GetDueReviews(userID string) ([]ProgressItem, error) {
	t.mu.RLock()
	defer t.mu.RUnlock()

	now := time.Now()
	var due []ProgressItem
	for _, item := range t.items {
		if item.UserID == userID && item.NextReviewAt != nil && item.NextReviewAt.Before(now) {
			due = append(due, *item)
		}
	}
	return due, nil
}

func clamp(v, min, max float64) float64 {
	if v < min {
		return min
	}
	if v > max {
		return max
	}
	return v
}
```

#### 6.2 — SM-2 Spaced Repetition (TDD)

**File:** `internal/progress/spaced_rep_test.go`

```go
package progress_test

import (
	"testing"

	"github.com/p-n-ai/pai-bot/internal/progress"
)

func TestSM2_Calculate(t *testing.T) {
	tests := []struct {
		name           string
		quality        int // 0-5 response quality
		repetitions    int
		easeFactor     float64
		interval       int
		wantRepGrow    bool
		wantIntervalUp bool
	}{
		{"perfect-first", 5, 0, 2.5, 1, true, true},
		{"good-second", 4, 1, 2.5, 1, true, true},
		{"fail-reset", 1, 5, 2.5, 10, false, false},
		{"barely-pass", 3, 2, 2.5, 6, true, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := progress.SM2Calculate(tt.quality, tt.repetitions, tt.easeFactor, tt.interval)

			if tt.wantRepGrow && result.Repetitions <= tt.repetitions {
				t.Errorf("Repetitions should grow: got %d, was %d", result.Repetitions, tt.repetitions)
			}
			if !tt.wantRepGrow && result.Repetitions != 0 {
				t.Errorf("Repetitions should reset to 0: got %d", result.Repetitions)
			}
			if result.EaseFactor < 1.3 {
				t.Errorf("EaseFactor should not go below 1.3: got %f", result.EaseFactor)
			}
		})
	}
}
```

**File:** `internal/progress/spaced_rep.go`

```go
package progress

import "math"

// SM2Result holds the output of an SM-2 calculation.
type SM2Result struct {
	Repetitions  int
	EaseFactor   float64
	IntervalDays int
}

// SM2Calculate implements the SuperMemo 2 algorithm.
// quality: 0-5 (0=blackout, 5=perfect)
func SM2Calculate(quality, repetitions int, easeFactor float64, intervalDays int) SM2Result {
	if quality < 3 {
		// Failed — reset
		return SM2Result{
			Repetitions:  0,
			EaseFactor:   math.Max(1.3, easeFactor-0.2),
			IntervalDays: 1,
		}
	}

	// Update ease factor
	newEF := easeFactor + (0.1 - float64(5-quality)*(0.08+float64(5-quality)*0.02))
	if newEF < 1.3 {
		newEF = 1.3
	}

	// Calculate new interval
	var newInterval int
	newReps := repetitions + 1

	switch {
	case repetitions == 0:
		newInterval = 1
	case repetitions == 1:
		newInterval = 6
	default:
		newInterval = int(math.Round(float64(intervalDays) * newEF))
	}

	return SM2Result{
		Repetitions:  newReps,
		EaseFactor:   newEF,
		IntervalDays: newInterval,
	}
}
```

#### 6.3 — Progress Display (TDD)

**File:** `internal/progress/display_test.go`

```go
package progress_test

import (
	"strings"
	"testing"

	"github.com/p-n-ai/pai-bot/internal/progress"
)

func TestFormatProgressBar(t *testing.T) {
	tests := []struct {
		name     string
		score    float64
		wantLen  int
	}{
		{"empty", 0.0, 10},
		{"half", 0.5, 10},
		{"full", 1.0, 10},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			bar := progress.FormatProgressBar(tt.score, tt.wantLen)
			if bar == "" {
				t.Error("FormatProgressBar() returned empty")
			}
		})
	}
}

func TestFormatProgressReport(t *testing.T) {
	items := []progress.ProgressItem{
		{TopicID: "F1-01", MasteryScore: 0.8},
		{TopicID: "F1-02", MasteryScore: 0.3},
	}

	report := progress.FormatProgressReport(items, 150, 3)
	if !strings.Contains(report, "F1-01") {
		t.Error("Report should contain topic IDs")
	}
}
```

**File:** `internal/progress/display.go`

```go
package progress

import (
	"fmt"
	"strings"
)

// FormatProgressBar creates a Unicode progress bar.
func FormatProgressBar(score float64, width int) string {
	filled := int(score * float64(width))
	if filled > width {
		filled = width
	}
	empty := width - filled
	return strings.Repeat("█", filled) + strings.Repeat("░", empty)
}

// FormatProgressReport creates a text report of all progress items.
func FormatProgressReport(items []ProgressItem, totalXP int, streak int) string {
	var sb strings.Builder

	sb.WriteString("📊 *Your Progress*\n\n")

	if streak > 0 {
		sb.WriteString(fmt.Sprintf("🔥 Streak: %d days\n", streak))
	}
	sb.WriteString(fmt.Sprintf("⭐ XP: %d\n\n", totalXP))

	for _, item := range items {
		bar := FormatProgressBar(item.MasteryScore, 10)
		pct := int(item.MasteryScore * 100)
		status := "📖"
		if IsMastered(item.MasteryScore) {
			status = "✅"
		}
		sb.WriteString(fmt.Sprintf("%s %s %s %d%%\n", status, item.TopicID, bar, pct))
	}

	if len(items) == 0 {
		sb.WriteString("Belum ada kemajuan lagi. Mari mula belajar! 🚀\n")
	}

	return sb.String()
}
```

#### Day 6 Exit Criteria

- [ ] `internal/progress/tracker.go` + tests — mastery tracking with weighted updates
- [ ] `internal/progress/spaced_rep.go` + tests — SM-2 algorithm implementation
- [ ] `internal/progress/display.go` + tests — Unicode progress bars, `/progress` report
- [ ] System prompt includes student progress context
- [ ] 🧑 40 more students recruited (50 total target)
- [ ] `make test-all` passes

---

### Day 7 (Tue) — Quiz Engine

**Entry criteria:** Day 6 complete. Progress tracking works. `make test-all` passes.

#### Tasks

| # | Task ID | Task | Owner | Files Created |
|---|---------|------|-------|---------------|
| 7.1 | `P-W2D7-1` | `/quiz` command: load questions, present sequentially, AI-grade, hints, summary | 🤖 | `internal/agent/quiz.go` |
| 7.2 | `P-W2D7-2` | Quiz state management: session_mode field (chat/quiz/challenge) | 🤖 | Update `internal/agent/engine.go` |
| 7.3 | `P-W2D7-3` | `CompleteJSON` fast-path in AI gateway for structured grading responses | 🤖 | Update `internal/ai/gateway.go` |
| 7.4 | `P-W2D7-4` | 🧑 Review all KSSM Algebra assessments for accuracy | 🧑 | Manual |

#### 7.1 — Quiz Engine (TDD)

**File:** `internal/agent/quiz_test.go`

```go
package agent_test

import (
	"testing"

	"github.com/p-n-ai/pai-bot/internal/agent"
)

func TestQuizSession_NextQuestion(t *testing.T) {
	questions := []agent.QuizQuestion{
		{ID: "Q1", Text: "What is 2x when x=3?", Answer: "6"},
		{ID: "Q2", Text: "Simplify 3a + 2a", Answer: "5a"},
	}

	session := agent.NewQuizSession("user1", "F1-01", questions)

	q, hasMore := session.NextQuestion()
	if !hasMore {
		t.Error("NextQuestion() should have more questions")
	}
	if q.ID != "Q1" {
		t.Errorf("First question ID = %q, want Q1", q.ID)
	}
}

func TestQuizSession_SubmitAnswer(t *testing.T) {
	questions := []agent.QuizQuestion{
		{ID: "Q1", Text: "What is 2x when x=3?", Answer: "6"},
	}

	session := agent.NewQuizSession("user1", "F1-01", questions)
	session.NextQuestion()

	result := session.SubmitAnswer("6")
	if !result.Correct {
		t.Error("Answer '6' should be correct")
	}
}

func TestQuizSession_Summary(t *testing.T) {
	questions := []agent.QuizQuestion{
		{ID: "Q1", Text: "Q1", Answer: "6"},
		{ID: "Q2", Text: "Q2", Answer: "5a"},
	}

	session := agent.NewQuizSession("user1", "F1-01", questions)

	// Answer both
	session.NextQuestion()
	session.SubmitAnswer("6")
	session.NextQuestion()
	session.SubmitAnswer("wrong")

	summary := session.Summary()
	if summary.Total != 2 {
		t.Errorf("Total = %d, want 2", summary.Total)
	}
	if summary.Correct != 1 {
		t.Errorf("Correct = %d, want 1", summary.Correct)
	}
}
```

**File:** `internal/agent/quiz.go`

```go
package agent

import (
	"fmt"
	"strings"
)

// QuizQuestion represents a single quiz question.
type QuizQuestion struct {
	ID                string
	Text              string
	Difficulty        string
	LearningObjective string
	Answer            string
	Hints             []string
	Distractors       []QuizDistractor
}

// QuizDistractor represents a common wrong answer with feedback.
type QuizDistractor struct {
	Value    string
	Feedback string
}

// AnswerResult holds the result of a submitted answer.
type AnswerResult struct {
	Correct    bool
	Feedback   string
	HintUsed   bool
}

// QuizSummary holds the results of a completed quiz.
type QuizSummary struct {
	TopicID  string
	Total    int
	Correct  int
	Score    float64
	Results  []QuestionResult
}

// QuestionResult holds the result of a single question.
type QuestionResult struct {
	QuestionID string
	Correct    bool
	UserAnswer string
	RightAnswer string
}

// QuizSession manages an active quiz for a user.
type QuizSession struct {
	UserID    string
	TopicID   string
	Questions []QuizQuestion
	Current   int
	Results   []QuestionResult
	HintIndex int
}

// NewQuizSession creates a new quiz session.
func NewQuizSession(userID, topicID string, questions []QuizQuestion) *QuizSession {
	return &QuizSession{
		UserID:    userID,
		TopicID:   topicID,
		Questions: questions,
		Current:   -1,
	}
}

// NextQuestion advances to the next question.
func (s *QuizSession) NextQuestion() (QuizQuestion, bool) {
	s.Current++
	s.HintIndex = 0
	if s.Current >= len(s.Questions) {
		return QuizQuestion{}, false
	}
	return s.Questions[s.Current], true
}

// SubmitAnswer checks the user's answer against the correct answer.
func (s *QuizSession) SubmitAnswer(answer string) AnswerResult {
	if s.Current < 0 || s.Current >= len(s.Questions) {
		return AnswerResult{Feedback: "No active question."}
	}

	q := s.Questions[s.Current]
	correct := strings.EqualFold(strings.TrimSpace(answer), strings.TrimSpace(q.Answer))

	result := QuestionResult{
		QuestionID:  q.ID,
		Correct:     correct,
		UserAnswer:  answer,
		RightAnswer: q.Answer,
	}
	s.Results = append(s.Results, result)

	feedback := ""
	if correct {
		feedback = "Betul! ✅"
	} else {
		// Check distractors for specific feedback
		for _, d := range q.Distractors {
			if strings.EqualFold(strings.TrimSpace(answer), strings.TrimSpace(d.Value)) {
				feedback = fmt.Sprintf("❌ %s\nJawapan betul: %s", d.Feedback, q.Answer)
				break
			}
		}
		if feedback == "" {
			feedback = fmt.Sprintf("❌ Jawapan betul: %s", q.Answer)
		}
	}

	return AnswerResult{
		Correct:  correct,
		Feedback: feedback,
	}
}

// GetHint returns the next hint for the current question.
func (s *QuizSession) GetHint() (string, bool) {
	if s.Current < 0 || s.Current >= len(s.Questions) {
		return "", false
	}
	q := s.Questions[s.Current]
	if s.HintIndex >= len(q.Hints) {
		return "", false
	}
	hint := q.Hints[s.HintIndex]
	s.HintIndex++
	return hint, true
}

// Summary returns the quiz results.
func (s *QuizSession) Summary() QuizSummary {
	correct := 0
	for _, r := range s.Results {
		if r.Correct {
			correct++
		}
	}

	score := 0.0
	if len(s.Results) > 0 {
		score = float64(correct) / float64(len(s.Results))
	}

	return QuizSummary{
		TopicID: s.TopicID,
		Total:   len(s.Results),
		Correct: correct,
		Score:   score,
		Results: s.Results,
	}
}

// IsComplete returns true if all questions have been answered.
func (s *QuizSession) IsComplete() bool {
	return len(s.Results) >= len(s.Questions)
}
```

#### Day 7 Exit Criteria

- [ ] `internal/agent/quiz.go` + tests — quiz engine with questions, answers, hints, distractors
- [ ] Session mode routing: chat vs quiz vs challenge
- [ ] `CompleteJSON` added to AI gateway for structured grading
- [ ] `/quiz` command loads questions and presents sequentially
- [ ] 🧑 KSSM Algebra assessments reviewed for accuracy
- [ ] `make test-all` passes

---

### Day 8 (Wed) — Proactive Nudges + Streaks

**Entry criteria:** Day 7 complete. Quiz engine works. `make test-all` passes.

#### Tasks

| # | Task ID | Task | Owner | Files Created |
|---|---------|------|-------|---------------|
| 8.1 | `P-W2D8-1` | Agent scheduler: check due reviews, respect quiet hours (21:00-07:00 MYT) | 🤖 | `internal/agent/scheduler.go` |
| 8.2 | `P-W2D8-2` | Streak tracking: consecutive days, milestones (3/7/14/30), celebrations | 🤖 | `internal/progress/streaks.go` |
| 8.3 | `P-W2D8-3` | XP system: session XP, quiz XP, mastery XP, streak XP | 🤖 | `internal/progress/xp.go` |
| 8.4 | `P-W2D8-4` | 🧑 Check metrics: how many of 50 students active? | 🧑 | Manual |

#### 8.1 — Scheduler (TDD)

**File:** `internal/agent/scheduler_test.go`

```go
package agent_test

import (
	"testing"
	"time"

	"github.com/p-n-ai/pai-bot/internal/agent"
)

func TestIsQuietHours(t *testing.T) {
	loc, _ := time.LoadLocation("Asia/Kuala_Lumpur")

	tests := []struct {
		name  string
		hour  int
		quiet bool
	}{
		{"midnight", 0, true},
		{"early-morning", 6, true},
		{"morning", 8, false},
		{"afternoon", 14, false},
		{"evening", 20, false},
		{"late-night", 22, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			now := time.Date(2026, 3, 1, tt.hour, 0, 0, 0, loc)
			got := agent.IsQuietHours(now)
			if got != tt.quiet {
				t.Errorf("IsQuietHours(%d:00) = %v, want %v", tt.hour, got, tt.quiet)
			}
		})
	}
}
```

**File:** `internal/agent/scheduler.go`

```go
package agent

import "time"

// QuietHoursStart is 21:00 MYT (no nudges after this).
const QuietHoursStart = 21

// QuietHoursEnd is 07:00 MYT (nudges resume after this).
const QuietHoursEnd = 7

// MaxNudgesPerDay limits proactive messages per student per day.
const MaxNudgesPerDay = 3

// IsQuietHours returns true if the given time is within quiet hours (MYT).
func IsQuietHours(t time.Time) bool {
	loc, err := time.LoadLocation("Asia/Kuala_Lumpur")
	if err != nil {
		loc = time.FixedZone("MYT", 8*60*60)
	}
	hour := t.In(loc).Hour()
	return hour >= QuietHoursStart || hour < QuietHoursEnd
}
```

#### 8.2 — Streaks (TDD)

**File:** `internal/progress/streaks_test.go`

```go
package progress_test

import (
	"testing"
	"time"

	"github.com/p-n-ai/pai-bot/internal/progress"
)

func TestStreakTracker_RecordActivity(t *testing.T) {
	tracker := progress.NewStreakTracker()

	tracker.RecordActivity("user1", time.Now())
	streak := tracker.GetStreak("user1")

	if streak.CurrentStreak != 1 {
		t.Errorf("CurrentStreak = %d, want 1", streak.CurrentStreak)
	}
}

func TestStreakTracker_ConsecutiveDays(t *testing.T) {
	tracker := progress.NewStreakTracker()
	now := time.Now()

	tracker.RecordActivity("user1", now.Add(-48*time.Hour))
	tracker.RecordActivity("user1", now.Add(-24*time.Hour))
	tracker.RecordActivity("user1", now)

	streak := tracker.GetStreak("user1")
	if streak.CurrentStreak != 3 {
		t.Errorf("CurrentStreak = %d, want 3", streak.CurrentStreak)
	}
}

func TestStreakMilestone(t *testing.T) {
	tests := []struct {
		days       int
		isMilestone bool
	}{
		{1, false},
		{3, true},
		{5, false},
		{7, true},
		{14, true},
		{30, true},
	}

	for _, tt := range tests {
		got := progress.IsStreakMilestone(tt.days)
		if got != tt.isMilestone {
			t.Errorf("IsStreakMilestone(%d) = %v, want %v", tt.days, got, tt.isMilestone)
		}
	}
}
```

**File:** `internal/progress/streaks.go`

```go
package progress

import (
	"sync"
	"time"
)

// Streak holds a user's streak data.
type Streak struct {
	UserID        string
	CurrentStreak int
	LongestStreak int
	LastActiveAt  time.Time
}

// StreakTracker manages streaks for all users.
type StreakTracker struct {
	streaks map[string]*Streak
	mu      sync.RWMutex
}

func NewStreakTracker() *StreakTracker {
	return &StreakTracker{
		streaks: make(map[string]*Streak),
	}
}

// RecordActivity records that a user was active at the given time.
func (t *StreakTracker) RecordActivity(userID string, at time.Time) {
	t.mu.Lock()
	defer t.mu.Unlock()

	s, ok := t.streaks[userID]
	if !ok {
		t.streaks[userID] = &Streak{
			UserID:        userID,
			CurrentStreak: 1,
			LongestStreak: 1,
			LastActiveAt:  at,
		}
		return
	}

	// Check if this is a new day
	lastDay := s.LastActiveAt.Truncate(24 * time.Hour)
	today := at.Truncate(24 * time.Hour)
	diff := today.Sub(lastDay)

	switch {
	case diff == 0:
		// Same day — no change
	case diff <= 24*time.Hour:
		// Next day — increment streak
		s.CurrentStreak++
		if s.CurrentStreak > s.LongestStreak {
			s.LongestStreak = s.CurrentStreak
		}
	default:
		// Missed a day — reset streak
		s.CurrentStreak = 1
	}

	s.LastActiveAt = at
}

// GetStreak returns a user's streak info.
func (t *StreakTracker) GetStreak(userID string) Streak {
	t.mu.RLock()
	defer t.mu.RUnlock()

	s, ok := t.streaks[userID]
	if !ok {
		return Streak{UserID: userID}
	}
	return *s
}

// IsStreakMilestone returns true if the streak count is a milestone.
func IsStreakMilestone(days int) bool {
	milestones := map[int]bool{3: true, 7: true, 14: true, 30: true, 60: true, 100: true}
	return milestones[days]
}
```

#### Day 8-10 Exit Criteria

- [ ] Scheduler checks due reviews, respects quiet hours, max 3 nudges/day
- [ ] Streak tracking with milestones and celebrations
- [ ] XP system: session, quiz, mastery, streak XP
- [ ] Topic unlocking when mastery ≥0.8
- [ ] `/learn [topic]` command sets current topic
- [ ] Daily summary computed at 22:00

**Week 2 Targets:**
- [ ] 50 students onboarded, 30+ active
- [ ] Progress tracking + quizzes live
- [ ] Nudge response ≥25%
- [ ] Day-7 retention ≥35%

**Week 2 Progress:** 8 packages | Progress tracking + quizzes + streaks + scheduler | 50 students

---

## WEEK 3 — MOTIVATION ENGINE

### Day 11 (Mon) — Goals + Challenges

**Entry criteria:** Week 2 complete. Progress tracking, quizzes, streaks live. `make test-all` passes.

#### Tasks

| # | Task ID | Task | Owner | Files Created |
|---|---------|------|-------|---------------|
| 11.1 | `P-W3D11-1` | Goal setting: `goals` table, `/goal` command, AI parses natural language | 🤖 | `internal/agent/goals.go` |
| 11.2 | `P-W3D11-2` | Goal progress tracking: auto-update after mastery changes | 🤖 | Update goals.go |
| 11.3 | `P-W3D11-3` | Peer challenges: `/challenge` command, 6-char code, simultaneous quiz | 🤖 | `internal/agent/challenge.go` |
| 11.4 | `P-W3D11-4` | 🧑 Design battle question sets for all KSSM Algebra topics | 🧑 | Manual |

#### 11.3 — Peer Challenge System (TDD)

**File:** `internal/agent/challenge_test.go`

```go
package agent_test

import (
	"testing"

	"github.com/p-n-ai/pai-bot/internal/agent"
)

func TestGenerateChallengeCode(t *testing.T) {
	code := agent.GenerateChallengeCode()
	if len(code) != 6 {
		t.Errorf("Challenge code length = %d, want 6", len(code))
	}
}

func TestChallenge_Create(t *testing.T) {
	store := agent.NewMemoryChallengeStore()

	ch, err := store.CreateChallenge("user1", "F1-01", 5)
	if err != nil {
		t.Fatalf("CreateChallenge() error = %v", err)
	}
	if ch.Code == "" {
		t.Error("Challenge.Code should not be empty")
	}
	if ch.CreatorID != "user1" {
		t.Errorf("CreatorID = %q, want user1", ch.CreatorID)
	}
}

func TestChallenge_Join(t *testing.T) {
	store := agent.NewMemoryChallengeStore()

	ch, _ := store.CreateChallenge("user1", "F1-01", 5)

	err := store.JoinChallenge(ch.Code, "user2")
	if err != nil {
		t.Fatalf("JoinChallenge() error = %v", err)
	}
}

func TestChallenge_Join_NotFound(t *testing.T) {
	store := agent.NewMemoryChallengeStore()

	err := store.JoinChallenge("XXXXXX", "user2")
	if err == nil {
		t.Error("JoinChallenge() should error for invalid code")
	}
}
```

**File:** `internal/agent/challenge.go`

```go
package agent

import (
	"crypto/rand"
	"fmt"
	"math/big"
	"sync"
	"time"
)

// Challenge represents a peer challenge (battle).
type Challenge struct {
	ID           string
	Code         string
	CreatorID    string
	OpponentID   string
	TopicID      string
	QuestionCount int
	State        string // waiting, active, completed
	CreatedAt    time.Time
}

// ChallengeStore is the interface for challenge persistence.
type ChallengeStore interface {
	CreateChallenge(creatorID, topicID string, questionCount int) (*Challenge, error)
	JoinChallenge(code, opponentID string) error
	GetChallenge(code string) (*Challenge, bool)
}

// MemoryChallengeStore is an in-memory ChallengeStore.
type MemoryChallengeStore struct {
	challenges map[string]*Challenge
	mu         sync.RWMutex
}

func NewMemoryChallengeStore() *MemoryChallengeStore {
	return &MemoryChallengeStore{
		challenges: make(map[string]*Challenge),
	}
}

func (s *MemoryChallengeStore) CreateChallenge(creatorID, topicID string, questionCount int) (*Challenge, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	code := GenerateChallengeCode()
	ch := &Challenge{
		ID:            code,
		Code:          code,
		CreatorID:     creatorID,
		TopicID:       topicID,
		QuestionCount: questionCount,
		State:         "waiting",
		CreatedAt:     time.Now(),
	}
	s.challenges[code] = ch
	return ch, nil
}

func (s *MemoryChallengeStore) JoinChallenge(code, opponentID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	ch, ok := s.challenges[code]
	if !ok {
		return fmt.Errorf("challenge %s not found", code)
	}
	if ch.State != "waiting" {
		return fmt.Errorf("challenge %s is not waiting for opponent", code)
	}

	ch.OpponentID = opponentID
	ch.State = "active"
	return nil
}

func (s *MemoryChallengeStore) GetChallenge(code string) (*Challenge, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	ch, ok := s.challenges[code]
	return ch, ok
}

// GenerateChallengeCode generates a 6-character alphanumeric code.
func GenerateChallengeCode() string {
	const charset = "ABCDEFGHJKLMNPQRSTUVWXYZ23456789" // No I/O/0/1 to avoid confusion
	code := make([]byte, 6)
	for i := range code {
		n, _ := rand.Int(rand.Reader, big.NewInt(int64(len(charset))))
		code[i] = charset[n.Int64()]
	}
	return string(code)
}
```

### Day 12-15 — Groups, Leaderboards, A/B Test, Analytics Dashboard

Follow the same TDD pattern for:

- **Day 12:** Class groups (`groups` + `group_members` tables), `/join [code]`, `/create_group [name]`, `/leaderboard` showing top 10 by weekly mastery gain
- **Day 13:** A/B test infrastructure (`user_flags` JSONB), post-challenge learning review, milestone celebrations
- **Day 14:** Analytics HTML page at `/admin/metrics`, smart nudge personalization (streak, goal, struggle area in nudge context)
- **Day 15:** Week 3 retro

**Week 3 Targets:**
- [ ] Goals, challenges, leaderboards live
- [ ] ≥1 school group active
- [ ] Challenge participation ≥20%
- [ ] 80+ students active

**Week 3 Progress:** 8 packages | Goals + challenges + leaderboards + A/B test | 80+ students

---

## WEEK 4 — ADMIN PANEL + FORM SELECTION

### Day 16 (Mon) — Scaffold Admin Panel

**Entry criteria:** Week 3 complete. Motivation features live. `make test-all` passes.

#### Tasks

| # | Task ID | Task | Owner | Files Created |
|---|---------|------|-------|---------------|
| 16.1 | `P-W4D16-1` | Scaffold Next.js 14 + TypeScript + Tailwind + shadcn/ui + Refine | 🤖 | `admin/` directory |
| 16.2 | `P-W4D16-2` | Teacher dashboard: mastery heatmap grid, "Nudge" button per student | 🤖 | `admin/src/app/dashboard/page.tsx` |
| 16.3 | `P-W4D16-3` | Student detail page: profile, mastery radar, activity grid, conversations | 🤖 | `admin/src/app/students/[id]/page.tsx` |
| 16.4 | `P-W4D16-4` | 🧑 Brief frontend engineer on 3 dashboard views | 🧑 | Manual |

#### 16.1 — Scaffold Admin Panel

```bash
cd admin
npx create-next-app@latest . --typescript --tailwind --eslint --app --src-dir --no-import-alias
npm install @refinedev/core @refinedev/nextjs-router @refinedev/react-hook-form
npm install @tanstack/react-query@5 recharts zod @hookform/resolvers date-fns lucide-react
npx shadcn@latest init
npx shadcn@latest add button card input label select textarea tabs badge table dialog
```

**File:** `admin/src/lib/api.ts`

```typescript
const API_BASE = process.env.NEXT_PUBLIC_API_URL || 'http://localhost:8080';

export interface Student {
  id: string;
  name: string;
  external_id: string;
  channel: string;
  form: string;
  created_at: string;
}

export interface ProgressItem {
  topic_id: string;
  mastery_score: number;
  ease_factor: number;
  interval_days: number;
  next_review_at: string | null;
  last_studied_at: string | null;
}

export interface ClassProgress {
  students: {
    id: string;
    name: string;
    topics: Record<string, number>; // topic_id -> mastery_score
  }[];
  topic_ids: string[];
}

export async function getClassProgress(classId: string): Promise<ClassProgress> {
  const res = await fetch(`${API_BASE}/api/admin/classes/${classId}/progress`, {
    headers: { Authorization: `Bearer ${getToken()}` },
  });
  if (!res.ok) throw new Error(`Failed: ${res.statusText}`);
  return res.json();
}

export async function getStudentDetail(studentId: string): Promise<{
  student: Student;
  progress: ProgressItem[];
  streak: { current: number; longest: number; total_xp: number };
}> {
  const res = await fetch(`${API_BASE}/api/admin/students/${studentId}`, {
    headers: { Authorization: `Bearer ${getToken()}` },
  });
  if (!res.ok) throw new Error(`Failed: ${res.statusText}`);
  return res.json();
}

function getToken(): string {
  if (typeof window !== 'undefined') {
    return localStorage.getItem('pai_token') || '';
  }
  return '';
}
```

### Day 17-20 — API Endpoints, Parent View, Form Selection, Reports, Budget Tracking

Follow the same pattern:

- **Day 17:** Admin API endpoints (GET classes/{id}/progress, GET students/{id}/detail, GET students/{id}/conversations, GET ai/usage). Parent view with child summary. Form/syllabus selection after /start.
- **Day 18:** Deploy admin panel via docker-compose with nginx reverse proxy. Class management page.
- **Day 19:** Weekly parent reports (Sunday 20:00 scheduler). Token budget tracking page.
- **Day 20:** Week 4 retro.

**Week 4 Targets:**
- [ ] Admin panel live
- [ ] All 3 Forms (F1, F2, F3) working
- [ ] 2+ teachers using dashboard
- [ ] 100+ students active
- [ ] Day-14 retention ≥30%

**Week 4 Progress:** 8 Go packages + admin panel | Teacher dashboard + parent view | 100+ students

---

## WEEK 5 — SELF-HOSTABLE + OPEN SOURCE PREP

### Day 21-22 — Cleanup + Documentation

**Entry criteria:** Week 4 complete. Admin panel live. `make test-all` passes.

#### Tasks

| # | Task ID | Task | Owner | Files Created |
|---|---------|------|-------|---------------|
| 21.1 | `P-W5D21-1` | Codebase cleanup: remove hardcoded values, Go doc comments, copyright headers, golangci-lint fixes, .env.example | 🤖 | Various |
| 21.2 | `P-W5D21-2` | Write docs: setup.md, architecture.md, ai-providers.md, curriculum.md, deployment.md | 🤖 | `docs/` |
| 21.3 | `P-W5D21-3` | Comprehensive README.md update: hero, quick start (5 steps), features, architecture diagram | 🤖 | Update README.md |
| 21.4 | `P-W5D21-4` | `scripts/setup.sh`: check prereqs → copy .env → prompt for tokens → docker compose up → migrate → seed | 🤖 | `scripts/setup.sh` |
| 21.5 | `P-W5D21-5` | 🧑 Write launch blog post (1500 words) | 🧑 | Manual |
| 21.6 | `P-W5D21-6` | 🧑 Record 3-min demo video | 🧑 | Manual |

#### 21.4 — Setup Script

**File:** `scripts/setup.sh`

```bash
#!/usr/bin/env bash
set -euo pipefail

echo "=== P&AI Bot Setup ==="
echo ""

# Check prerequisites
check_cmd() {
    if ! command -v "$1" &> /dev/null; then
        echo "❌ $1 is required but not installed."
        exit 1
    fi
    echo "✅ $1 found"
}

check_cmd go
check_cmd docker
check_cmd node

# Copy .env
if [ ! -f .env ]; then
    cp .env.example .env
    echo "📄 Created .env from .env.example"
    echo ""
    echo "Please configure your .env file:"
    echo "  1. LEARN_TELEGRAM_BOT_TOKEN (required — get from @BotFather)"
    echo "  2. At least one AI provider API key"
    echo ""
    read -p "Press Enter after editing .env..."
fi

# Start infrastructure
echo "🐳 Starting Docker services..."
docker compose up -d postgres dragonfly nats

# Wait for Postgres
echo "⏳ Waiting for PostgreSQL..."
sleep 3

# Run migrations
echo "📦 Running database migrations..."
docker exec -i $(docker compose ps -q postgres) psql -U pai pai < migrations/001_initial.up.sql

# Download Go dependencies
echo "📥 Downloading Go dependencies..."
go mod download

# Build
echo "🔨 Building server..."
go build -o bin/pai-server ./cmd/server

echo ""
echo "=== Setup Complete ==="
echo ""
echo "Start the bot:  ./bin/pai-server"
echo "Or with Docker:  docker compose up -d"
echo ""
```

### Day 23 — Self-Host Testing

- Multi-tenancy: `LEARN_TENANT_MODE` single/multi, auto-create default tenant
- Helm chart: Deployment, StatefulSet (PG, Dragonfly), ConfigMap, Secret, Service, Ingress
- 🧑 Fresh machine test: new AWS instance, follow README only, fix every issue

### Day 24-25 — Security + WhatsApp + Data Export

- WhatsApp Cloud API adapter (behind `LEARN_WHATSAPP_ENABLED` flag)
- Data export: GET /export/students (CSV), /export/conversations (JSON), /export/progress (CSV)
- Security audit: auth on all endpoints, tenant isolation, rate limiting, parameterized queries

**Week 5 Targets:**
- [ ] Fresh `docker compose up` works in <10min
- [ ] README + docs complete
- [ ] Helm chart exists
- [ ] Security audit done
- [ ] 150+ students active

---

## WEEK 6 — LAUNCH + SCALE

### Day 26 (Mon) — LAUNCH DAY

#### Tasks

| # | Task ID | Task | Owner | Files Created |
|---|---------|------|-------|---------------|
| 26.1 | `P-W6D26-1` | Landing page at `/`: static HTML, "Try on Telegram" + "Self-host" buttons | 🤖 | Static HTML |
| 26.2 | `P-W6D26-2` | K8s health probes: /healthz, /readyz, graceful shutdown on SIGTERM | 🤖 | Update `cmd/server/main.go` |
| 26.3 | `P-W6D26-3` | 🧑 Publish blog, HN submission, Twitter/LinkedIn/Reddit, 50 personal emails | 🧑 | Manual |
| 26.4 | `P-W6D26-4` | 🧑 Monitor server + conversations all day | 🧑 | Manual |

### Day 27-30 — Respond, Onboard, i18n, Analytics, Report

- **Day 27:** Fix top 5 bugs. School onboarding wizard in admin.
- **Day 28:** i18n support (detect Telegram language_code, add to system prompt).
- **Day 29:** Comprehensive analytics API: GET /analytics/report.
- **Day 30:** 6-week report, final retro.

**Week 6 Targets:**
- [ ] Public launch
- [ ] 500+ GitHub stars
- [ ] 10+ schools
- [ ] 500-1,000 students
- [ ] A/B test conclusive

---

## Appendix A — Complete Package Reference

| Package | Created On | Key Files | Purpose |
|---------|-----------|-----------|---------|
| `internal/platform/config` | Day 0 | `config.go` | Environment variable loading with `LEARN_` prefix |
| `internal/platform/database` | Day 0 | `database.go` | PostgreSQL connection pool (pgx) |
| `internal/platform/cache` | Day 0 | `cache.go` | Dragonfly/Redis client (go-redis) |
| `internal/ai` | Day 0 | `gateway.go`, `router.go`, `mock.go`, `provider_openai.go`, `provider_anthropic.go`, `provider_ollama.go` | AI Gateway with provider-agnostic routing |
| `internal/chat` | Day 1 | `gateway.go`, `telegram.go` | Unified chat interface, Telegram adapter |
| `internal/agent` | Day 1-11 | `engine.go`, `store.go`, `events.go`, `topics.go`, `session.go`, `users.go`, `quiz.go`, `scheduler.go`, `goals.go`, `challenge.go` | Conversation engine, quiz, challenges, scheduling |
| `internal/curriculum` | Day 1 | `loader.go`, `types.go` | YAML curriculum loader |
| `internal/progress` | Day 6 | `tracker.go`, `spaced_rep.go`, `display.go`, `streaks.go`, `xp.go` | Mastery tracking, SM-2, streaks, XP |
| `internal/auth` | Day 16 | `jwt.go`, `middleware.go` | JWT auth + RBAC middleware |
| `internal/tenant` | Day 23 | `tenant.go`, `middleware.go` | Multi-tenancy isolation |

---

## Appendix B — Database Migration Reference

| Migration | Day | Tables Created |
|-----------|-----|---------------|
| `001_initial` | Day 0 | tenants, users, conversations, messages, learning_progress, events |
| `002_assessments` | Day 7 | assessments (quiz results) |
| `003_streaks` | Day 8 | streaks (engagement data) |
| `004_token_budgets` | Day 8 | token_budgets (AI cost tracking) |
| `005_goals` | Day 11 | goals (student goals) |
| `006_challenges` | Day 11 | challenges (peer battles) |
| `007_groups` | Day 12 | groups, group_members (class groups) |
| `008_user_flags` | Day 13 | Add user_flags JSONB to users (A/B testing) |

---

## Appendix C — Environment Variables Quick Reference

| Variable | Day | Required | Default |
|----------|-----|----------|---------|
| `LEARN_SERVER_PORT` | 0 | No | `8080` |
| `LEARN_DATABASE_URL` | 0 | No | `postgres://pai:pai@localhost:5432/pai` |
| `LEARN_CACHE_URL` | 0 | No | `redis://localhost:6379` |
| `LEARN_NATS_URL` | 0 | No | `nats://localhost:4222` |
| `LEARN_TELEGRAM_BOT_TOKEN` | 0 | Yes | — |
| `LEARN_AI_OPENAI_API_KEY` | 0 | No* | — |
| `LEARN_AI_ANTHROPIC_API_KEY` | 0 | No* | — |
| `LEARN_AI_OLLAMA_ENABLED` | 0 | No* | `false` |
| `LEARN_AI_OLLAMA_URL` | 0 | No | `http://localhost:11434` |
| `LEARN_AI_OPENROUTER_API_KEY` | 0 | No* | — |
| `LEARN_AUTH_JWT_SECRET` | 16 | No | `change-me-in-production` |
| `LEARN_TENANT_MODE` | 23 | No | `single` |
| `LEARN_WHATSAPP_ENABLED` | 24 | No | `false` |
| `LEARN_LOG_LEVEL` | 0 | No | `info` |

*At least one AI provider must be configured.

---

## Appendix D — Performance Targets

| Operation | Target | Validation |
|-----------|--------|------------|
| AI response latency (teaching, P95) | <3s | Measure via event logging |
| AI response latency (grading, P95) | <1s | Measure via event logging |
| Message processing throughput | 10K msg/s per instance | Load test with k6 |
| Database queries (P95) | <50ms | pgx prepared statements + Dragonfly cache |
| Admin panel page load (LCP) | <1s | Lighthouse audit |
| Docker image size | <30MB | `docker images` |
| Cold start time | <100ms | `time ./bin/pai-server --help` |
| `make test-all` | <30s | CI pipeline timing |

---

## Appendix E — Progress Tracking Dashboard

| Day | Packages | Tests | Bot Commands | Students | Key Feature |
|-----|----------|-------|-------------|----------|-------------|
| 0 | 4 | ✅ | — | 0 | Foundation + Docker + CI |
| 1 | 7 | ✅ | /start | 0 | Bot responds on Telegram |
| 2 | 7 | ✅ | /start | 0 | Message persistence + topic detection |
| 3 | 7 | ✅ | /start | 3 | Deployed + first students |
| 4 | 7 | ✅ | /start | 10 | Session management + ratings |
| 5 | 7 | ✅ | /start | 10 | Week 1 retro |
| 6 | 8 | ✅ | /start, /progress | 50 | Mastery tracking + SM-2 |
| 7 | 8 | ✅ | /start, /progress, /quiz | 50 | Quiz engine |
| 8 | 8 | ✅ | /start, /progress, /quiz | 50 | Streaks + XP + nudges |
| 9 | 8 | ✅ | /start, /progress, /quiz, /learn | 50 | Topic navigation |
| 10 | 8 | ✅ | all Week 2 | 50 | Week 2 retro |
| 11 | 8 | ✅ | + /goal, /challenge | 80 | Goals + peer battles |
| 12 | 8 | ✅ | + /join, /leaderboard | 80 | Groups + leaderboards |
| 13 | 8 | ✅ | all Week 3 | 80 | A/B test + milestones |
| 14 | 8 | ✅ | all Week 3 | 80 | Analytics dashboard |
| 15 | 8 | ✅ | all Week 3 | 80 | Week 3 retro |
| 16 | 9 | ✅ | all | 100 | Admin panel scaffold |
| 17 | 9 | ✅ | all | 100 | API + parent view + form selection |
| 18 | 9 | ✅ | all | 100 | Admin deployed |
| 19 | 9 | ✅ | all | 100 | Reports + budget tracking |
| 20 | 9 | ✅ | all | 100 | Week 4 retro |
| 21-22 | 9 | ✅ | all | 150 | Cleanup + docs |
| 23 | 10 | ✅ | all | 150 | Multi-tenancy + Helm |
| 24-25 | 10 | ✅ | all | 150 | WhatsApp + security + export |
| 26 | 10 | ✅ | all | 500+ | LAUNCH DAY |
| 27-30 | 10 | ✅ | all | 500-1K | Polish + scale + report |

---

## Appendix F — Task Count Summary

| Week | 🤖 Claude Code | 🧑 Human | Total |
|------|----------------|----------|-------|
| 0 | 8 | 0 | 8 |
| 1 | 17 | 8 | 25 |
| 2 | 15 | 6 | 21 |
| 3 | 11 | 5 | 16 |
| 4 | 11 | 5 | 16 |
| 5 | 9 | 5 | 14 |
| 6 | 6 | 6 | 12 |
| **Total** | **77** | **35** | **112** |
