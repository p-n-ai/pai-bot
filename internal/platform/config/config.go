// Package config loads application configuration from environment variables.
// Core app variables use the LEARN_ prefix; auth variables use PAI_AUTH_.
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
	Email          EmailConfig
	Telegram       TelegramConfig
	WhatsApp       WhatsAppConfig
	Auth           AuthConfig
	Tenant         TenantConfig
	Log            LogConfig
	Features       FeatureConfig
	CurriculumPath string
}

// FeatureConfig holds toggle-able product features.
type FeatureConfig struct {
	DisableMultiLanguage        bool
	RatingPromptEvery           int
	AIPersonalizedNudgesEnabled bool
	DevMode                     bool
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

// EmailConfig holds invite email delivery settings.
type EmailConfig struct {
	SMTPAddr     string
	SMTPUsername string
	SMTPPassword string
	FromAddress  string
	FromName     string
	BaseURL      string
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
	JWTSecret string
	Google    GoogleOAuthConfig
}

// GoogleOAuthConfig holds Google OIDC settings for admin login/linking.
type GoogleOAuthConfig struct {
	ClientID              string
	ClientSecret          string
	AllowedDomain         string
	DiscoveryURL          string
	EmulatorSigningSecret string
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

// Load reads configuration from environment variables.
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
		Email: EmailConfig{
			SMTPAddr:     envStr("LEARN_EMAIL_SMTP_ADDR", ""),
			SMTPUsername: envStr("LEARN_EMAIL_SMTP_USERNAME", ""),
			SMTPPassword: envStr("LEARN_EMAIL_SMTP_PASSWORD", ""),
			FromAddress:  envStr("LEARN_EMAIL_FROM_ADDRESS", ""),
			FromName:     envStr("LEARN_EMAIL_FROM_NAME", "P&AI Bot"),
			BaseURL:      envStr("LEARN_EMAIL_BASE_URL", ""),
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
			JWTSecret: envStr("PAI_AUTH_SECRET", "change-me-in-production"),
			Google: GoogleOAuthConfig{
				ClientID:              envStr("PAI_AUTH_GOOGLE_CLIENT_ID", ""),
				ClientSecret:          envStr("PAI_AUTH_GOOGLE_CLIENT_SECRET", ""),
				AllowedDomain:         envStr("PAI_AUTH_GOOGLE_ALLOWED_DOMAIN", ""),
				DiscoveryURL:          envStr("PAI_AUTH_GOOGLE_DISCOVERY_URL", "https://accounts.google.com/.well-known/openid-configuration"),
				EmulatorSigningSecret: envStr("PAI_AUTH_GOOGLE_EMULATOR_SIGNING_SECRET", ""),
			},
		},
		Tenant: TenantConfig{
			Mode: envStr("LEARN_TENANT_MODE", "single"),
		},
		Log: LogConfig{
			Level:  envStr("LEARN_LOG_LEVEL", "info"),
			Format: envStr("LEARN_LOG_FORMAT", "json"),
		},
		Features: FeatureConfig{
			DevMode:                     envBool("LEARN_DEV_MODE", false),
			DisableMultiLanguage:        envBool("LEARN_DISABLE_MULTI_LANGUAGE", false),
			RatingPromptEvery:           envInt("LEARN_RATING_PROMPT_EVERY_REPLIES", 5),
			AIPersonalizedNudgesEnabled: envBoolWithFallback("LEARN_AI_PERSONALIZED_NUDGES_ENABLED", "LEARN_AI_NUDGES_ENABLED", true),
		},
		CurriculumPath: envStr("LEARN_CURRICULUM_PATH", "./oss"),
	}

	return cfg, nil
}

// Validate checks that required configuration is present.
func (c *Config) Validate() error {
	if c.Telegram.BotToken == "" && !c.Features.DevMode {
		return fmt.Errorf("LEARN_TELEGRAM_BOT_TOKEN is required")
	}

	if !c.HasAIProvider() && !c.Features.DevMode {
		return fmt.Errorf("at least one AI provider must be configured")
	}

	if c.Tenant.Mode != "single" && c.Tenant.Mode != "multi" {
		return fmt.Errorf("LEARN_TENANT_MODE must be 'single' or 'multi', got %q", c.Tenant.Mode)
	}
	if c.Email.SMTPAddr != "" || c.Email.FromAddress != "" || c.Email.SMTPUsername != "" || c.Email.SMTPPassword != "" || c.Email.BaseURL != "" {
		if strings.TrimSpace(c.Email.SMTPAddr) == "" {
			return fmt.Errorf("LEARN_EMAIL_SMTP_ADDR is required when email delivery is configured")
		}
		if strings.TrimSpace(c.Email.FromAddress) == "" {
			return fmt.Errorf("LEARN_EMAIL_FROM_ADDRESS is required when email delivery is configured")
		}
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

func envBoolWithFallback(primaryKey, fallbackKey string, fallback bool) bool {
	if v := os.Getenv(primaryKey); v != "" {
		return strings.EqualFold(v, "true") || v == "1"
	}
	if v := os.Getenv(fallbackKey); v != "" {
		return strings.EqualFold(v, "true") || v == "1"
	}
	return fallback
}
