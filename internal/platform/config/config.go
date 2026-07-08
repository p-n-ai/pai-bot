// Copyright 2026 the P&AI authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

// Package config loads application configuration from environment variables.
// Core app variables use the LEARN_ prefix; auth variables use PAI_AUTH_.
package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/p-n-ai/pai-bot/internal/platform/featureflags"
)

// DefaultAuthSecret is the dev fallback for PAI_AUTH_SECRET; secrets must
// never be encrypted under it.
const DefaultAuthSecret = "change-me-in-production"

// Config holds all application configuration.
type Config struct {
	Server         ServerConfig
	Database       DatabaseConfig
	Cache          CacheConfig
	AI             AIConfig
	Email          EmailConfig
	Telegram       TelegramConfig
	WhatsApp       WhatsAppConfig
	Auth           AuthConfig
	Tenant         TenantConfig
	Log            LogConfig
	Runtime        RuntimeConfig
	FeatureFlags   featureflags.Features
	CurriculumPath string
}

// RuntimeConfig holds runtime knobs. New product experiments use FeatureFlags.
type RuntimeConfig struct {
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

// AIConfig holds configuration for all AI providers.
type AIConfig struct {
	DefaultProvider string
	Mock            MockAIConfig
	OpenAI          OpenAIConfig
	Anthropic       AnthropicConfig
	DeepSeek        DeepSeekConfig
	Google          GoogleConfig
	Ollama          OllamaConfig
	OpenRouter      OpenRouterConfig
}

// MockAIConfig holds local dev-only mock AI settings.
type MockAIConfig struct {
	Response string
}

// OpenAIConfig holds OpenAI provider settings.
type OpenAIConfig struct {
	APIKey string
	Model  string
}

// AnthropicConfig holds Anthropic provider settings.
type AnthropicConfig struct {
	APIKey string
	Model  string
}

// DeepSeekConfig holds DeepSeek provider settings (OpenAI-compatible).
type DeepSeekConfig struct {
	APIKey string
	Model  string
}

// GoogleConfig holds Google Gemini provider settings.
type GoogleConfig struct {
	APIKey string
	Model  string
}

// OllamaConfig holds self-hosted Ollama settings.
type OllamaConfig struct {
	Enabled bool
	URL     string
	Model   string
}

// OpenRouterConfig holds OpenRouter provider settings.
type OpenRouterConfig struct {
	APIKey string
	Model  string
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

// WhatsAppConfig holds WhatsApp settings.
// Backend selects the adapter: "cloudapi" (Meta Cloud API) or "meow" (whatsmeow, default).
type WhatsAppConfig struct {
	Enabled     bool
	Backend     string // "cloudapi" or "meow"
	AccessToken string // Cloud API only
	PhoneID     string // Cloud API only
	VerifyToken string // Cloud API only
	MeowDBPath  string // whatsmeow session DB path
	QRToken     string // token to access /whatsapp/qr endpoint
}

// AuthConfig holds authentication settings.
type AuthConfig struct {
	JWTSecret      string
	Google         GoogleOAuthConfig
	BootstrapAdmin BootstrapAdminConfig
}

// GoogleOAuthConfig holds Google OIDC settings for admin login/linking.
type GoogleOAuthConfig struct {
	ClientID              string
	ClientSecret          string
	AllowedDomain         string
	DiscoveryURL          string
	EmulatorSigningSecret string
}

// BootstrapAdminConfig holds startup bootstrap credentials for the first platform admin.
type BootstrapAdminConfig struct {
	Email    string
	Password string
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
	// Unlike the one-env-to-one-field values below, PAI_FEATURES is a compact
	// list of overrides that needs validation before it can be stored.
	parsedFeatureFlags, err := featureflags.Parse(envStr("PAI_FEATURES", ""))
	if err != nil {
		return nil, err
	}

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
		AI: AIConfig{
			DefaultProvider: envStr("LEARN_AI_DEFAULT_PROVIDER", ""),
			Mock: MockAIConfig{
				Response: envStr("LEARN_AI_MOCK_RESPONSE", ""),
			},
			OpenAI: OpenAIConfig{
				APIKey: envStr("LEARN_AI_OPENAI_API_KEY", ""),
				Model:  envStr("LEARN_AI_OPENAI_MODEL", ""),
			},
			Anthropic: AnthropicConfig{
				APIKey: envStr("LEARN_AI_ANTHROPIC_API_KEY", ""),
				Model:  envStr("LEARN_AI_ANTHROPIC_MODEL", ""),
			},
			DeepSeek: DeepSeekConfig{
				APIKey: envStr("LEARN_AI_DEEPSEEK_API_KEY", ""),
				Model:  envStr("LEARN_AI_DEEPSEEK_MODEL", ""),
			},
			Google: GoogleConfig{
				APIKey: envStr("LEARN_AI_GOOGLE_API_KEY", ""),
				Model:  envStr("LEARN_AI_GOOGLE_MODEL", ""),
			},
			Ollama: OllamaConfig{
				Enabled: envBool("LEARN_AI_OLLAMA_ENABLED", false),
				URL:     envStr("LEARN_AI_OLLAMA_URL", "http://localhost:11434"),
				Model:   envStr("LEARN_AI_OLLAMA_MODEL", ""),
			},
			OpenRouter: OpenRouterConfig{
				APIKey: envStr("LEARN_AI_OPENROUTER_API_KEY", ""),
				Model:  envStr("LEARN_AI_OPENROUTER_MODEL", ""),
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
			Backend:     envStr("LEARN_WHATSAPP_BACKEND", "meow"),
			AccessToken: envStr("LEARN_WHATSAPP_ACCESS_TOKEN", ""),
			PhoneID:     envStr("LEARN_WHATSAPP_PHONE_ID", ""),
			VerifyToken: envStr("LEARN_WHATSAPP_VERIFY_TOKEN", ""),
			MeowDBPath:  envStr("LEARN_WHATSAPP_MEOW_DB", "file:whatsmeow.db?_pragma=foreign_keys(1)&_pragma=journal_mode(WAL)&_pragma=busy_timeout(5000)"),
			QRToken:     envStr("LEARN_WHATSAPP_QR_TOKEN", ""),
		},
		Auth: AuthConfig{
			JWTSecret: envStr("PAI_AUTH_SECRET", DefaultAuthSecret),
			Google: GoogleOAuthConfig{
				ClientID:              envStr("PAI_AUTH_GOOGLE_CLIENT_ID", ""),
				ClientSecret:          envStr("PAI_AUTH_GOOGLE_CLIENT_SECRET", ""),
				AllowedDomain:         envStr("PAI_AUTH_GOOGLE_ALLOWED_DOMAIN", ""),
				DiscoveryURL:          envStr("PAI_AUTH_GOOGLE_DISCOVERY_URL", "https://accounts.google.com/.well-known/openid-configuration"),
				EmulatorSigningSecret: envStr("PAI_AUTH_GOOGLE_EMULATOR_SIGNING_SECRET", ""),
			},
			BootstrapAdmin: BootstrapAdminConfig{
				Email:    envStr("PAI_AUTH_BOOTSTRAP_ADMIN_EMAIL", "platform-admin@example.com"),
				Password: envStr("PAI_AUTH_BOOTSTRAP_ADMIN_PASSWORD", "demo-password"),
			},
		},
		Tenant: TenantConfig{
			Mode: envStr("LEARN_TENANT_MODE", "single"),
		},
		Log: LogConfig{
			Level:  envStr("LEARN_LOG_LEVEL", "info"),
			Format: envStr("LEARN_LOG_FORMAT", "json"),
		},
		Runtime: RuntimeConfig{
			DevMode:                     envBool("LEARN_DEV_MODE", false),
			DisableMultiLanguage:        envBool("LEARN_DISABLE_MULTI_LANGUAGE", false),
			RatingPromptEvery:           envInt("LEARN_RATING_PROMPT_EVERY_REPLIES", 5),
			AIPersonalizedNudgesEnabled: envBool("LEARN_AI_PERSONALIZED_NUDGES_ENABLED", true),
		},
		FeatureFlags:   parsedFeatureFlags,
		CurriculumPath: envStr("LEARN_CURRICULUM_PATH", "./oss"),
	}

	return cfg, nil
}

// Validate checks that required configuration is present.
func (c *Config) Validate() error {
	if c.Telegram.BotToken == "" && !c.Runtime.DevMode {
		return fmt.Errorf("LEARN_TELEGRAM_BOT_TOKEN is required")
	}

	if !c.HasAIProvider() && !c.Runtime.DevMode {
		return fmt.Errorf("at least one AI provider must be configured")
	}
	if c.AI.DefaultProvider != "" && !isKnownAIProvider(c.AI.DefaultProvider) {
		return fmt.Errorf("unsupported LEARN_AI_DEFAULT_PROVIDER %q", c.AI.DefaultProvider)
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
	return c.mockAIProviderEnabled() ||
		c.AI.OpenAI.APIKey != "" ||
		c.AI.Anthropic.APIKey != "" ||
		c.AI.DeepSeek.APIKey != "" ||
		c.AI.Google.APIKey != "" ||
		c.AI.OpenRouter.APIKey != "" ||
		c.AI.Ollama.Enabled
}

func (c *Config) mockAIProviderEnabled() bool {
	return strings.EqualFold(strings.TrimSpace(c.AI.DefaultProvider), "mock") &&
		strings.TrimSpace(c.AI.Mock.Response) != ""
}

func isKnownAIProvider(name string) bool {
	switch strings.ToLower(strings.TrimSpace(name)) {
	case "mock", "openai", "anthropic", "deepseek", "google", "ollama", "openrouter":
		return true
	default:
		return false
	}
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
