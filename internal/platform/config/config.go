// Copyright 2026 the P&AI authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

// Package config loads application configuration from environment variables.
// Core app variables use the LEARN_ prefix; auth variables use PAI_AUTH_.
package config

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"unicode"

	"github.com/p-n-ai/pai-bot/internal/ai"
	"github.com/p-n-ai/pai-bot/internal/auth"
	"github.com/p-n-ai/pai-bot/internal/platform/featureflags"
)

// DefaultAuthSecret is the dev fallback for PAI_AUTH_SECRET; secrets must
// never be encrypted under it.
const DefaultAuthSecret = "change-me-in-production"

const (
	defaultBootstrapAdminEmail    = "platform-admin@example.com"
	defaultBootstrapAdminPassword = "demo-password"
)

// Config holds all application configuration.
type Config struct {
	Server         ServerConfig
	Database       DatabaseConfig
	Cache          CacheConfig
	Security       SecurityConfig
	AI             AIConfig
	Email          EmailConfig
	Telegram       TelegramConfig
	WhatsApp       WhatsAppConfig
	Slack          SlackConfig
	Discord        DiscordConfig
	Teams          TeamsConfig
	Auth           AuthConfig
	Tenant         TenantConfig
	Log            LogConfig
	Runtime        RuntimeConfig
	FeatureFlags   featureflags.Features
	FocusedPage    FocusedPageConfig
	Embed          EmbedConfig
	Retrieval      RetrievalConfig
	CurriculumPath string
	SkillsPath     string
}

// SecurityConfig holds process-level cryptographic roots with distinct purposes.
type SecurityConfig struct {
	RuntimeSettingsEncryptionKey   string
	PreviousSettingsEncryptionKeys []string
}

// RuntimeConfig holds runtime knobs. New product experiments use FeatureFlags.
type RuntimeConfig struct {
	DisableMultiLanguage        bool
	AIPersonalizedNudgesEnabled bool
	DevMode                     bool
}

// ServerConfig holds HTTP server settings.
type ServerConfig struct {
	Port int
	Host string
}

// FocusedPageConfig owns the server-selected public origin and fixed return action.
type FocusedPageConfig struct {
	BaseURL        string
	TelegramCTAURL string
}

type EmbedConfig struct {
	BaseURL string
}

type RetrievalConfig struct {
	EmbeddingBaseURL    string
	EmbeddingAPIKey     string
	EmbeddingModel      string
	EmbeddingDimensions int
	GraphDepth          int
	GraphFrontier       int
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
	DefaultProvider  string
	Mock             MockAIConfig
	OpenAI           OpenAIConfig
	Codex            CodexConfig
	Anthropic        AnthropicConfig
	Google           GoogleConfig
	Ollama           OllamaConfig
	OpenRouter       OpenRouterConfig
	CatalogProviders map[string]CatalogProviderConfig
}

// CatalogProviderConfig holds credentials and a default model for a catalog provider.
type CatalogProviderConfig struct {
	APIKey string
	Model  string
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

// CodexConfig holds local Codex CLI credential and model settings.
type CodexConfig struct {
	Enabled      bool
	Home         string
	AccessToken  string
	RefreshToken string
	AccountID    string
	Model        string
}

// TelegramConfig holds Telegram Bot API settings.
type TelegramConfig struct {
	BotToken string
}

// SlackConfig holds Slack Events API and Web API settings.
type SlackConfig struct {
	Enabled       bool
	BotToken      string
	SigningSecret string
}

// DiscordConfig holds Discord interaction and REST API settings.
type DiscordConfig struct {
	Enabled       bool
	BotToken      string
	PublicKey     string
	ApplicationID string
}

// TeamsConfig holds Microsoft Teams Bot Framework settings.
type TeamsConfig struct {
	Enabled     bool
	AppID       string
	AppPassword string
	AppTenantID string
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
	AppSecret   string // authenticates webhook POST bodies
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
	AdminBaseURL          string
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
	previousSettingsEncryptionKeys, err := secretListEnv("PAI_CONFIG_PREVIOUS_ENCRYPTION_KEYS")
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
		Security: SecurityConfig{
			RuntimeSettingsEncryptionKey:   envStr("PAI_CONFIG_ENCRYPTION_KEY", ""),
			PreviousSettingsEncryptionKeys: previousSettingsEncryptionKeys,
		},
		FocusedPage: FocusedPageConfig{
			BaseURL:        envStr("LEARN_FOCUSED_PAGE_BASE_URL", ""),
			TelegramCTAURL: envStr("LEARN_FOCUSED_PAGE_TELEGRAM_CTA_URL", ""),
		},
		Embed: EmbedConfig{
			BaseURL: envStr("LEARN_EMBED_BASE_URL", ""),
		},
		Retrieval: RetrievalConfig{
			EmbeddingBaseURL:    envStr("LEARN_RETRIEVAL_EMBEDDING_BASE_URL", ""),
			EmbeddingAPIKey:     envStr("LEARN_RETRIEVAL_EMBEDDING_API_KEY", ""),
			EmbeddingModel:      envStr("LEARN_RETRIEVAL_EMBEDDING_MODEL", "text-embedding-3-small"),
			EmbeddingDimensions: envInt("LEARN_RETRIEVAL_EMBEDDING_DIMENSIONS", 1536),
			GraphDepth:          envInt("LEARN_RETRIEVAL_GRAPH_DEPTH", 1),
			GraphFrontier:       envInt("LEARN_RETRIEVAL_GRAPH_FRONTIER", 40),
		},
		AI: AIConfig{
			DefaultProvider:  envStr("LEARN_AI_DEFAULT_PROVIDER", ""),
			CatalogProviders: loadCatalogProviderConfigs(),
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
			Codex: CodexConfig{
				Enabled:      envBool("LEARN_AI_CODEX_ENABLED", false),
				Home:         envStr("LEARN_AI_CODEX_HOME", defaultCodexHome()),
				AccessToken:  envStr("LEARN_AI_CODEX_ACCESS_TOKEN", ""),
				RefreshToken: envStr("LEARN_AI_CODEX_REFRESH_TOKEN", ""),
				AccountID:    envStr("LEARN_AI_CODEX_ACCOUNT_ID", ""),
				Model:        envStr("LEARN_AI_CODEX_MODEL", "gpt-5.4"),
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
			AppSecret:   envStr("LEARN_WHATSAPP_APP_SECRET", ""),
		},
		Slack: SlackConfig{
			Enabled:       envBool("LEARN_SLACK_ENABLED", false),
			BotToken:      envStr("LEARN_SLACK_BOT_TOKEN", ""),
			SigningSecret: envStr("LEARN_SLACK_SIGNING_SECRET", ""),
		},
		Discord: DiscordConfig{
			Enabled:       envBool("LEARN_DISCORD_ENABLED", false),
			BotToken:      envStr("LEARN_DISCORD_BOT_TOKEN", ""),
			PublicKey:     envStr("LEARN_DISCORD_PUBLIC_KEY", ""),
			ApplicationID: envStr("LEARN_DISCORD_APPLICATION_ID", ""),
		},
		Teams: TeamsConfig{
			Enabled:     envBool("LEARN_TEAMS_ENABLED", false),
			AppID:       envStr("LEARN_TEAMS_APP_ID", ""),
			AppPassword: envStr("LEARN_TEAMS_APP_PASSWORD", ""),
			AppTenantID: envStr("LEARN_TEAMS_APP_TENANT_ID", ""),
		},
		Auth: AuthConfig{
			JWTSecret: envStr("PAI_AUTH_SECRET", DefaultAuthSecret),
			Google: GoogleOAuthConfig{
				ClientID:              envStr("PAI_AUTH_GOOGLE_CLIENT_ID", ""),
				ClientSecret:          envStr("PAI_AUTH_GOOGLE_CLIENT_SECRET", ""),
				AllowedDomain:         envStr("PAI_AUTH_GOOGLE_ALLOWED_DOMAIN", ""),
				DiscoveryURL:          envStr("PAI_AUTH_GOOGLE_DISCOVERY_URL", "https://accounts.google.com/.well-known/openid-configuration"),
				EmulatorSigningSecret: envStr("PAI_AUTH_GOOGLE_EMULATOR_SIGNING_SECRET", ""),
				AdminBaseURL:          envStr("PAI_AUTH_GOOGLE_ADMIN_BASE_URL", ""),
			},
			BootstrapAdmin: BootstrapAdminConfig{
				Email:    envStr("PAI_AUTH_BOOTSTRAP_ADMIN_EMAIL", defaultBootstrapAdminEmail),
				Password: envStr("PAI_AUTH_BOOTSTRAP_ADMIN_PASSWORD", defaultBootstrapAdminPassword),
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
			AIPersonalizedNudgesEnabled: envBool("LEARN_AI_PERSONALIZED_NUDGES_ENABLED", true),
		},
		FeatureFlags:   parsedFeatureFlags,
		CurriculumPath: envStr("LEARN_CURRICULUM_PATH", "./oss"),
		SkillsPath:     envStr("LEARN_SKILLS_PATH", "./skills"),
	}

	return cfg, nil
}

// Validate checks that required configuration is present.
func (c *Config) Validate() error {
	if !c.Runtime.DevMode && !c.hasExternalChatAdapter() {
		return fmt.Errorf("at least one external chat adapter must be configured")
	}

	if c.AI.Codex.Enabled && isPersonalCodexHome(c.AI.Codex.Home) {
		return fmt.Errorf("LEARN_AI_CODEX_HOME must not use the backend user's personal .codex directory")
	}
	if !c.HasAIProvider() && !c.CodexDeviceAuthAvailable() && !c.Runtime.DevMode {
		return fmt.Errorf("at least one AI provider must be configured")
	}
	if !c.Runtime.DevMode {
		if strings.TrimSpace(c.Auth.JWTSecret) == "" || c.Auth.JWTSecret == DefaultAuthSecret {
			return fmt.Errorf("PAI_AUTH_SECRET must be set to a private secret in production")
		}
		if strings.TrimSpace(c.Auth.BootstrapAdmin.Email) == "" ||
			strings.TrimSpace(c.Auth.BootstrapAdmin.Password) == "" ||
			strings.EqualFold(strings.TrimSpace(c.Auth.BootstrapAdmin.Email), defaultBootstrapAdminEmail) ||
			c.Auth.BootstrapAdmin.Password == defaultBootstrapAdminPassword {
			return fmt.Errorf("PAI_AUTH_BOOTSTRAP_ADMIN_EMAIL and PAI_AUTH_BOOTSTRAP_ADMIN_PASSWORD must be set to private credentials in production")
		}
	}
	if c.AI.DefaultProvider != "" && !isKnownAIProvider(c.AI.DefaultProvider) {
		return fmt.Errorf("unsupported LEARN_AI_DEFAULT_PROVIDER %q", c.AI.DefaultProvider)
	}
	if err := ValidateRuntimeSettingsKeys(
		c.Security.RuntimeSettingsEncryptionKey,
		c.Auth.JWTSecret,
		c.Security.PreviousSettingsEncryptionKeys,
	); err != nil {
		return err
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
	if (strings.TrimSpace(c.FocusedPage.BaseURL) == "") != (strings.TrimSpace(c.FocusedPage.TelegramCTAURL) == "") {
		return fmt.Errorf("LEARN_FOCUSED_PAGE_BASE_URL and LEARN_FOCUSED_PAGE_TELEGRAM_CTA_URL must be configured together")
	}
	if strings.TrimSpace(c.FocusedPage.BaseURL) != "" && c.Auth.JWTSecret == DefaultAuthSecret {
		return fmt.Errorf("PAI_AUTH_SECRET must be set to a private secret when focused pages are enabled")
	}
	if raw := strings.TrimSpace(c.Embed.BaseURL); raw != "" {
		parsed, err := url.Parse(raw)
		if err != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") || parsed.Host == "" ||
			parsed.User != nil || (parsed.Path != "" && parsed.Path != "/") || parsed.RawQuery != "" || parsed.Fragment != "" {
			return fmt.Errorf("LEARN_EMBED_BASE_URL must be an HTTP or HTTPS origin")
		}
	}
	if c.Retrieval.EmbeddingDimensions != 0 && c.Retrieval.EmbeddingDimensions != 1536 {
		return fmt.Errorf("LEARN_RETRIEVAL_EMBEDDING_DIMENSIONS must be 1536")
	}
	if c.Retrieval.GraphDepth != 0 && (c.Retrieval.GraphDepth < 1 || c.Retrieval.GraphDepth > 3) {
		return fmt.Errorf("LEARN_RETRIEVAL_GRAPH_DEPTH must be between 1 and 3")
	}
	if c.Retrieval.GraphFrontier != 0 && (c.Retrieval.GraphFrontier < 1 || c.Retrieval.GraphFrontier > 200) {
		return fmt.Errorf("LEARN_RETRIEVAL_GRAPH_FRONTIER must be between 1 and 200")
	}
	if err := c.validateChatAdapterCredentials(); err != nil {
		return err
	}

	return nil
}

func (c *Config) hasExternalChatAdapter() bool {
	return strings.TrimSpace(c.Telegram.BotToken) != "" ||
		c.Slack.Enabled ||
		c.Discord.Enabled ||
		c.Teams.Enabled ||
		c.WhatsApp.Enabled
}

func (c *Config) validateChatAdapterCredentials() error {
	slackConfigured := c.Slack.Enabled ||
		strings.TrimSpace(c.Slack.BotToken) != "" ||
		strings.TrimSpace(c.Slack.SigningSecret) != ""
	if slackConfigured &&
		(strings.TrimSpace(c.Slack.BotToken) == "" || strings.TrimSpace(c.Slack.SigningSecret) == "") {
		return fmt.Errorf("LEARN_SLACK_BOT_TOKEN and LEARN_SLACK_SIGNING_SECRET must be configured together")
	}

	discordConfigured := c.Discord.Enabled ||
		strings.TrimSpace(c.Discord.BotToken) != "" ||
		strings.TrimSpace(c.Discord.PublicKey) != "" ||
		strings.TrimSpace(c.Discord.ApplicationID) != ""
	if discordConfigured &&
		(strings.TrimSpace(c.Discord.BotToken) == "" ||
			strings.TrimSpace(c.Discord.PublicKey) == "" ||
			strings.TrimSpace(c.Discord.ApplicationID) == "") {
		return fmt.Errorf("LEARN_DISCORD_BOT_TOKEN, LEARN_DISCORD_PUBLIC_KEY, and LEARN_DISCORD_APPLICATION_ID must be configured together")
	}

	teamsConfigured := c.Teams.Enabled ||
		strings.TrimSpace(c.Teams.AppID) != "" ||
		strings.TrimSpace(c.Teams.AppPassword) != "" ||
		strings.TrimSpace(c.Teams.AppTenantID) != ""
	if teamsConfigured &&
		(strings.TrimSpace(c.Teams.AppID) == "" || strings.TrimSpace(c.Teams.AppPassword) == "") {
		return fmt.Errorf("LEARN_TEAMS_APP_ID and LEARN_TEAMS_APP_PASSWORD must be configured together")
	}

	if c.WhatsApp.Enabled {
		if strings.TrimSpace(c.WhatsApp.AccessToken) == "" ||
			strings.TrimSpace(c.WhatsApp.PhoneID) == "" ||
			strings.TrimSpace(c.WhatsApp.VerifyToken) == "" ||
			strings.TrimSpace(c.WhatsApp.AppSecret) == "" {
			return fmt.Errorf("LEARN_WHATSAPP_ACCESS_TOKEN, LEARN_WHATSAPP_PHONE_ID, LEARN_WHATSAPP_VERIFY_TOKEN, and LEARN_WHATSAPP_APP_SECRET are required when WhatsApp is enabled")
		}
	}

	return nil
}

// HasAIProvider returns true if at least one AI provider is configured.
func (c *Config) HasAIProvider() bool {
	if c.mockAIProviderEnabled() || c.AI.OpenAI.APIKey != "" ||
		strings.TrimSpace(c.AI.Codex.AccessToken) != "" || c.AI.Anthropic.APIKey != "" ||
		c.AI.Google.APIKey != "" || c.AI.OpenRouter.APIKey != "" || c.AI.Ollama.Enabled ||
		c.CodexDeviceAuthAvailable() {
		return true
	}
	for _, provider := range ai.ProviderCatalog() {
		settings := c.AI.CatalogProviders[provider.ID]
		if settings.APIKey != "" {
			return true
		}
	}
	return false
}

// CodexDeviceAuthAvailable reports whether the server has isolated storage
// where an authenticated admin can establish the Codex provider.
func (c *Config) CodexDeviceAuthAvailable() bool {
	return c.AI.Codex.Enabled &&
		strings.TrimSpace(c.AI.Codex.Home) != ""
}

func (c *Config) mockAIProviderEnabled() bool {
	return strings.EqualFold(strings.TrimSpace(c.AI.DefaultProvider), "mock") &&
		strings.TrimSpace(c.AI.Mock.Response) != ""
}

func isKnownAIProvider(name string) bool {
	name = strings.ToLower(strings.TrimSpace(name))
	for _, core := range []string{"mock", "openai", "codex", "anthropic", "google", "ollama", "openrouter"} {
		if name == core {
			return true
		}
	}
	_, ok := ai.LookupProviderDefinition(name)
	return ok
}

func loadCatalogProviderConfigs() map[string]CatalogProviderConfig {
	providers := make(map[string]CatalogProviderConfig)
	for _, provider := range ai.ProviderCatalog() {
		prefix := "LEARN_AI_" + strings.ToUpper(provider.ID)
		providers[provider.ID] = CatalogProviderConfig{
			APIKey: envStr(prefix+"_API_KEY", ""),
			Model:  envStr(prefix+"_MODEL", ""),
		}
	}
	return providers
}

func defaultCodexHome() string {
	configDir, err := os.UserConfigDir()
	if err != nil {
		return ""
	}
	return filepath.Join(configDir, "pai-bot", "codex")
}

func isPersonalCodexHome(configured string) bool {
	home, err := os.UserHomeDir()
	if err != nil || strings.TrimSpace(configured) == "" {
		return false
	}
	personal := filepath.Join(home, ".codex")
	configuredAbs, configuredErr := filepath.Abs(configured)
	personalAbs, personalErr := filepath.Abs(personal)
	if configuredErr == nil && personalErr == nil && filepath.Clean(configuredAbs) == filepath.Clean(personalAbs) {
		return true
	}
	configuredInfo, configuredErr := os.Stat(configured)
	personalInfo, personalErr := os.Stat(personal)
	return configuredErr == nil && personalErr == nil && os.SameFile(configuredInfo, personalInfo)
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

func secretListEnv(key string) ([]string, error) {
	raw := os.Getenv(key)
	if raw == "" {
		return nil, nil
	}
	var values []string
	if err := json.Unmarshal([]byte(raw), &values); err != nil {
		return nil, fmt.Errorf("%s must be a JSON array of strings", key)
	}
	return values, nil
}

// ValidateRuntimeSettingsKeys checks the active and retired encryption roots
// without requiring an active root. Runtime secret writes enforce that
// requirement at their own boundary.
func ValidateRuntimeSettingsKeys(active, auth string, previous []string) error {
	if active != "" && nonWhitespaceLen(active) < 32 {
		return fmt.Errorf("PAI_CONFIG_ENCRYPTION_KEY must contain at least 32 non-whitespace characters")
	}
	if active != "" && weakSecretRoot(active) {
		return fmt.Errorf("PAI_CONFIG_ENCRYPTION_KEY must be a high-entropy secret")
	}
	if active != "" && active == auth {
		return fmt.Errorf("PAI_CONFIG_ENCRYPTION_KEY must differ from PAI_AUTH_SECRET")
	}
	if len(previous) > 8 {
		return fmt.Errorf("PAI_CONFIG_PREVIOUS_ENCRYPTION_KEYS must contain at most 8 keys")
	}
	total := 0
	seen := make(map[string]struct{}, len(previous))
	seenIDs := make(map[string]struct{}, len(previous)+1)
	if active != "" {
		seenIDs[RuntimeSettingsKeyID(active)] = struct{}{}
	}
	for _, key := range previous {
		total += len(key)
		if nonWhitespaceLen(key) < 32 {
			return fmt.Errorf("each PAI_CONFIG_PREVIOUS_ENCRYPTION_KEYS value must contain at least 32 non-whitespace characters")
		}
		if weakSecretRoot(key) {
			return fmt.Errorf("each PAI_CONFIG_PREVIOUS_ENCRYPTION_KEYS value must be a high-entropy secret")
		}
		if key == active || key == auth {
			return fmt.Errorf("PAI_CONFIG_PREVIOUS_ENCRYPTION_KEYS must differ from active encryption and auth keys")
		}
		if _, duplicate := seen[key]; duplicate {
			return fmt.Errorf("PAI_CONFIG_PREVIOUS_ENCRYPTION_KEYS must not contain duplicates")
		}
		seen[key] = struct{}{}
		keyID := RuntimeSettingsKeyID(key)
		if _, duplicate := seenIDs[keyID]; duplicate {
			return fmt.Errorf("PAI_CONFIG_PREVIOUS_ENCRYPTION_KEYS must have unique derived key IDs")
		}
		seenIDs[keyID] = struct{}{}
	}
	if total > 8*1024 {
		return fmt.Errorf("PAI_CONFIG_PREVIOUS_ENCRYPTION_KEYS exceeds the 8 KiB limit")
	}
	return nil
}

// ValidateProductionSecrets rejects deployment credentials that are missing,
// public defaults, or invalid runtime-settings encryption roots.
func ValidateProductionSecrets(authSecret, active string, previous []string, bootstrapAdminPassword string) error {
	if authSecret == "" || authSecret == DefaultAuthSecret {
		return fmt.Errorf("PAI_AUTH_SECRET must be a private value")
	}
	if active == "" {
		return fmt.Errorf("PAI_CONFIG_ENCRYPTION_KEY must be set to an independent high-entropy secret")
	}
	if bootstrapAdminPassword == "" || bootstrapAdminPassword == "demo-password" {
		return fmt.Errorf("PAI_AUTH_BOOTSTRAP_ADMIN_PASSWORD must be a private value")
	}
	if err := auth.ValidatePassword(bootstrapAdminPassword); err != nil {
		return fmt.Errorf("PAI_AUTH_BOOTSTRAP_ADMIN_PASSWORD does not meet the password policy: %w", err)
	}
	return ValidateRuntimeSettingsKeys(active, authSecret, previous)
}

// ValidateProductionSecretEnvironment parses and validates only the secrets
// required by production deployment preflights.
func ValidateProductionSecretEnvironment() error {
	previous, err := secretListEnv("PAI_CONFIG_PREVIOUS_ENCRYPTION_KEYS")
	if err != nil {
		return err
	}
	return ValidateProductionSecrets(
		os.Getenv("PAI_AUTH_SECRET"),
		os.Getenv("PAI_CONFIG_ENCRYPTION_KEY"),
		previous,
		os.Getenv("PAI_AUTH_BOOTSTRAP_ADMIN_PASSWORD"),
	)
}

func weakSecretRoot(value string) bool {
	distinct := make(map[rune]struct{})
	for _, r := range value {
		if !unicode.IsSpace(r) {
			distinct[r] = struct{}{}
		}
	}
	return len(distinct) < 12
}

// RuntimeSettingsKeyID derives the stable non-secret identifier used for
// exact encryption-root lookup in versioned credential envelopes.
func RuntimeSettingsKeyID(secret string) string {
	mac := hmac.New(sha256.New, []byte(secret))
	_, _ = mac.Write([]byte("pai-bot/runtime-settings/key-id/v1"))
	return base64.RawURLEncoding.EncodeToString(mac.Sum(nil)[:16])
}

func nonWhitespaceLen(value string) int {
	count := 0
	for _, r := range value {
		if !unicode.IsSpace(r) {
			count++
		}
	}
	return count
}
