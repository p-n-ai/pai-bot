package config

import (
	"os"
	"testing"
)

// clearEnv unsets all LEARN_ environment variables for a clean test.
func clearEnv(t *testing.T) {
	t.Helper()
	envVars := []string{
		"LEARN_SERVER_PORT",
		"LEARN_SERVER_HOST",
		"LEARN_DATABASE_URL",
		"LEARN_DATABASE_MAX_CONNS",
		"LEARN_DATABASE_MIN_CONNS",
		"LEARN_CACHE_URL",
		"LEARN_NATS_URL",
		"LEARN_TELEGRAM_BOT_TOKEN",
		"LEARN_AI_OPENAI_API_KEY",
		"LEARN_AI_ANTHROPIC_API_KEY",
		"LEARN_AI_DEEPSEEK_API_KEY",
		"LEARN_AI_GOOGLE_API_KEY",
		"LEARN_AI_OPENROUTER_API_KEY",
		"LEARN_AI_OLLAMA_ENABLED",
		"LEARN_AI_OLLAMA_URL",
		"LEARN_AUTH_JWT_SECRET",
		"LEARN_AUTH_ACCESS_TOKEN_TTL",
		"LEARN_AUTH_REFRESH_TOKEN_TTL",
		"LEARN_TENANT_MODE",
		"LEARN_WHATSAPP_ENABLED",
		"LEARN_LOG_LEVEL",
		"LEARN_LOG_FORMAT",
	}
	for _, v := range envVars {
		_ = os.Unsetenv(v)
	}
}

func TestLoad_Defaults(t *testing.T) {
	clearEnv(t)

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if cfg.Server.Port != 8080 {
		t.Errorf("Server.Port = %d, want 8080", cfg.Server.Port)
	}
	if cfg.Database.MaxConns != 25 {
		t.Errorf("Database.MaxConns = %d, want 25", cfg.Database.MaxConns)
	}
	if cfg.Database.MinConns != 5 {
		t.Errorf("Database.MinConns = %d, want 5", cfg.Database.MinConns)
	}
	if cfg.Database.URL != "postgres://pai:pai@localhost:5432/pai?sslmode=disable" {
		t.Errorf("Database.URL = %q, want default postgres URL", cfg.Database.URL)
	}
	if cfg.Cache.URL != "redis://localhost:6379" {
		t.Errorf("Cache.URL = %q, want redis://localhost:6379", cfg.Cache.URL)
	}
	if cfg.NATS.URL != "nats://localhost:4222" {
		t.Errorf("NATS.URL = %q, want nats://localhost:4222", cfg.NATS.URL)
	}
	if cfg.Tenant.Mode != "single" {
		t.Errorf("Tenant.Mode = %q, want single", cfg.Tenant.Mode)
	}
	if cfg.Auth.AccessTokenTTL != 15 {
		t.Errorf("Auth.AccessTokenTTL = %d, want 15", cfg.Auth.AccessTokenTTL)
	}
	if cfg.Auth.RefreshTokenTTL != 7 {
		t.Errorf("Auth.RefreshTokenTTL = %d, want 7", cfg.Auth.RefreshTokenTTL)
	}
}

func TestLoad_FromEnv(t *testing.T) {
	clearEnv(t)

	t.Setenv("LEARN_SERVER_PORT", "9090")
	t.Setenv("LEARN_DATABASE_URL", "postgres://test:test@localhost/testdb")
	t.Setenv("LEARN_TELEGRAM_BOT_TOKEN", "test-token-123")
	t.Setenv("LEARN_AI_OPENAI_API_KEY", "sk-test-key")
	t.Setenv("LEARN_AI_OLLAMA_URL", "http://localhost:11434")
	t.Setenv("LEARN_AUTH_JWT_SECRET", "super-secret")
	t.Setenv("LEARN_TENANT_MODE", "multi")

	cfg, err := Load()
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
	if cfg.AI.OpenAI.APIKey != "sk-test-key" {
		t.Errorf("AI.OpenAI.APIKey = %q, want sk-test-key", cfg.AI.OpenAI.APIKey)
	}
	if cfg.AI.Ollama.URL != "http://localhost:11434" {
		t.Errorf("AI.Ollama.URL = %q, want http://localhost:11434", cfg.AI.Ollama.URL)
	}
	if cfg.Auth.JWTSecret != "super-secret" {
		t.Errorf("Auth.JWTSecret = %q, want super-secret", cfg.Auth.JWTSecret)
	}
	if cfg.Tenant.Mode != "multi" {
		t.Errorf("Tenant.Mode = %q, want multi", cfg.Tenant.Mode)
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
			clearEnv(t)
			if tt.envVal != "" {
				t.Setenv("LEARN_TENANT_MODE", tt.envVal)
			}
			cfg, err := Load()
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
	clearEnv(t)

	t.Setenv("LEARN_AI_OPENAI_API_KEY", "sk-test")
	t.Setenv("LEARN_AI_OLLAMA_ENABLED", "true")

	cfg, err := Load()
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

func TestValidate_MissingBotToken(t *testing.T) {
	clearEnv(t)
	t.Setenv("LEARN_AI_OLLAMA_ENABLED", "true")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if err := cfg.Validate(); err == nil {
		t.Fatal("Validate() should return error when bot token is missing")
	}
}

func TestValidate_MissingAIProvider(t *testing.T) {
	clearEnv(t)
	t.Setenv("LEARN_TELEGRAM_BOT_TOKEN", "test-token")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if err := cfg.Validate(); err == nil {
		t.Fatal("Validate() should return error when no AI provider is configured")
	}
}

func TestValidate_InvalidTenantMode(t *testing.T) {
	clearEnv(t)
	t.Setenv("LEARN_TELEGRAM_BOT_TOKEN", "test-token")
	t.Setenv("LEARN_AI_OLLAMA_ENABLED", "true")
	t.Setenv("LEARN_TENANT_MODE", "invalid")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if err := cfg.Validate(); err == nil {
		t.Fatal("Validate() should return error for invalid tenant mode")
	}
}

func TestValidate_Success(t *testing.T) {
	clearEnv(t)
	t.Setenv("LEARN_TELEGRAM_BOT_TOKEN", "test-token")
	t.Setenv("LEARN_AI_OLLAMA_ENABLED", "true")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if err := cfg.Validate(); err != nil {
		t.Fatalf("Validate() error = %v; should pass", err)
	}
}

func TestHasAIProvider(t *testing.T) {
	tests := []struct {
		name   string
		envKey string
		envVal string
		want   bool
	}{
		{"none", "", "", false},
		{"OpenAI", "LEARN_AI_OPENAI_API_KEY", "sk-test", true},
		{"Anthropic", "LEARN_AI_ANTHROPIC_API_KEY", "sk-ant-test", true},
		{"DeepSeek", "LEARN_AI_DEEPSEEK_API_KEY", "sk-ds-test", true},
		{"Google", "LEARN_AI_GOOGLE_API_KEY", "AIza-test", true},
		{"OpenRouter", "LEARN_AI_OPENROUTER_API_KEY", "sk-or-test", true},
		{"Ollama", "LEARN_AI_OLLAMA_ENABLED", "true", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			clearEnv(t)
			if tt.envKey != "" {
				t.Setenv(tt.envKey, tt.envVal)
			}

			cfg, err := Load()
			if err != nil {
				t.Fatalf("Load() error = %v", err)
			}
			if cfg.HasAIProvider() != tt.want {
				t.Errorf("HasAIProvider() = %v, want %v", cfg.HasAIProvider(), tt.want)
			}
		})
	}
}

func TestOllamaEnabledParsing(t *testing.T) {
	tests := []struct {
		name string
		val  string
		want bool
	}{
		{"true", "true", true},
		{"TRUE", "TRUE", true},
		{"false", "false", false},
		{"1", "1", true},
		{"0", "0", false},
		{"empty", "", false},
		{"invalid", "notabool", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			clearEnv(t)
			if tt.val != "" {
				t.Setenv("LEARN_AI_OLLAMA_ENABLED", tt.val)
			}

			cfg, err := Load()
			if err != nil {
				t.Fatalf("Load() error = %v", err)
			}
			if cfg.AI.Ollama.Enabled != tt.want {
				t.Errorf("AI.Ollama.Enabled = %v, want %v", cfg.AI.Ollama.Enabled, tt.want)
			}
		})
	}
}
