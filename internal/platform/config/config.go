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
	Server         ServerConfig
	Database       DatabaseConfig
	Cache          CacheConfig
	NATS           NATSConfig
	AI             AIConfig
	Telegram       TelegramConfig
	WhatsApp       WhatsAppConfig
	Auth           AuthConfig
	Tenant         TenantConfig
	Log            LogConfig
	CurriculumPath string
}

// ServerConfig holds HTTP server settings.
type ServerConfig struct {
	Port int
	Host string
}

// DatabaseConfig holds PostgreSQL connection settings.
type DatabaseConfig struct {
	URL      string
	MaxConns int
	MinConns int
}

// CacheConfig holds Dragonfly/Redis connection settings.
type CacheConfig struct {
	URL string
}

// NATSConfig holds NATS connection settings.
type NATSConfig struct {
	URL string
}

// AIConfig holds configuration for all AI providers.
type AIConfig struct {
	OpenAI     OpenAIConfig
	Anthropic  AnthropicConfig
	DeepSeek   DeepSeekConfig
	Google     GoogleConfig
	Ollama     OllamaConfig
	OpenRouter OpenRouterConfig
}

// OpenAIConfig holds OpenAI provider settings.
type OpenAIConfig struct {
	APIKey string
}

// AnthropicConfig holds Anthropic provider settings.
type AnthropicConfig struct {
	APIKey string
}

// DeepSeekConfig holds DeepSeek provider settings (OpenAI-compatible).
type DeepSeekConfig struct {
	APIKey string
}

// GoogleConfig holds Google Gemini provider settings.
type GoogleConfig struct {
	APIKey string
}

// OllamaConfig holds self-hosted Ollama settings.
type OllamaConfig struct {
	Enabled bool
	URL     string
}

// OpenRouterConfig holds OpenRouter provider settings.
type OpenRouterConfig struct {
	APIKey string
}

// TelegramConfig holds Telegram Bot API settings.
type TelegramConfig struct {
	BotToken string
}

// WhatsAppConfig holds WhatsApp Cloud API settings.
type WhatsAppConfig struct {
	Enabled     bool
	AccessToken string
	PhoneID     string
	VerifyToken string
}

// AuthConfig holds authentication settings.
type AuthConfig struct {
	JWTSecret       string
	AccessTokenTTL  int // minutes
	RefreshTokenTTL int // days
}

// TenantConfig holds multi-tenancy settings.
type TenantConfig struct {
	Mode string // "single" or "multi"
}

// LogConfig holds logging settings.
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
			DeepSeek: DeepSeekConfig{
				APIKey: envStr("LEARN_AI_DEEPSEEK_API_KEY", ""),
			},
			Google: GoogleConfig{
				APIKey: envStr("LEARN_AI_GOOGLE_API_KEY", ""),
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
		CurriculumPath: envStr("LEARN_CURRICULUM_PATH", "./oss"),
	}

	return cfg, nil
}

// Validate checks that required configuration is present.
func (c *Config) Validate() error {
	if c.Telegram.BotToken == "" {
		return fmt.Errorf("LEARN_TELEGRAM_BOT_TOKEN is required")
	}

	if !c.HasAIProvider() {
		return fmt.Errorf("at least one AI provider must be configured")
	}

	if c.Tenant.Mode != "single" && c.Tenant.Mode != "multi" {
		return fmt.Errorf("LEARN_TENANT_MODE must be 'single' or 'multi', got %q", c.Tenant.Mode)
	}

	return nil
}

// HasAIProvider returns true if at least one AI provider is configured.
func (c *Config) HasAIProvider() bool {
	return c.AI.OpenAI.APIKey != "" ||
		c.AI.Anthropic.APIKey != "" ||
		c.AI.DeepSeek.APIKey != "" ||
		c.AI.Google.APIKey != "" ||
		c.AI.OpenRouter.APIKey != "" ||
		c.AI.Ollama.Enabled
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
