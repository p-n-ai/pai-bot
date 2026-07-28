// Copyright 2026 the P&AI authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/p-n-ai/pai-bot/internal/platform/featureflags"
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
		"LEARN_TELEGRAM_BOT_TOKEN",
		"LEARN_SLACK_ENABLED",
		"LEARN_SLACK_BOT_TOKEN",
		"LEARN_SLACK_SIGNING_SECRET",
		"LEARN_DISCORD_ENABLED",
		"LEARN_DISCORD_BOT_TOKEN",
		"LEARN_DISCORD_PUBLIC_KEY",
		"LEARN_DISCORD_APPLICATION_ID",
		"LEARN_TEAMS_ENABLED",
		"LEARN_TEAMS_APP_ID",
		"LEARN_TEAMS_APP_PASSWORD",
		"LEARN_TEAMS_APP_TENANT_ID",
		"LEARN_FOCUSED_PAGE_BASE_URL",
		"LEARN_FOCUSED_PAGE_TELEGRAM_CTA_URL",
		"LEARN_EMBED_BASE_URL",
		"LEARN_EMAIL_SMTP_ADDR",
		"LEARN_EMAIL_SMTP_USERNAME",
		"LEARN_EMAIL_SMTP_PASSWORD",
		"LEARN_EMAIL_FROM_ADDRESS",
		"LEARN_EMAIL_FROM_NAME",
		"LEARN_EMAIL_BASE_URL",
		"LEARN_AI_OPENAI_API_KEY",
		"LEARN_AI_OPENAI_MODEL",
		"LEARN_AI_CODEX_ACCESS_TOKEN",
		"LEARN_AI_CODEX_REFRESH_TOKEN",
		"LEARN_AI_CODEX_ACCOUNT_ID",
		"LEARN_AI_CODEX_MODEL",
		"LEARN_AI_ANTHROPIC_API_KEY",
		"LEARN_AI_ANTHROPIC_MODEL",
		"LEARN_AI_DEEPSEEK_API_KEY",
		"LEARN_AI_DEEPSEEK_MODEL",
		"LEARN_AI_GOOGLE_API_KEY",
		"LEARN_AI_GOOGLE_MODEL",
		"LEARN_AI_OPENROUTER_API_KEY",
		"LEARN_AI_OPENROUTER_MODEL",
		"LEARN_AI_CODEX_ENABLED",
		"LEARN_AI_CODEX_HOME",
		"LEARN_AI_CODEX_MODEL",
		"LEARN_AI_DEFAULT_PROVIDER",
		"LEARN_AI_OLLAMA_ENABLED",
		"LEARN_AI_OLLAMA_URL",
		"LEARN_AI_OLLAMA_MODEL",
		"PAI_AUTH_SECRET",
		"PAI_CONFIG_ENCRYPTION_KEY",
		"PAI_CONFIG_PREVIOUS_ENCRYPTION_KEYS",
		"PAI_AUTH_GOOGLE_CLIENT_ID",
		"PAI_AUTH_GOOGLE_CLIENT_SECRET",
		"PAI_AUTH_GOOGLE_ALLOWED_DOMAIN",
		"PAI_AUTH_GOOGLE_DISCOVERY_URL",
		"PAI_AUTH_GOOGLE_EMULATOR_SIGNING_SECRET",
		"PAI_AUTH_GOOGLE_ADMIN_BASE_URL",
		"PAI_AUTH_BOOTSTRAP_ADMIN_EMAIL",
		"PAI_AUTH_BOOTSTRAP_ADMIN_PASSWORD",
		"LEARN_TENANT_MODE",
		"LEARN_WHATSAPP_ENABLED",
		"LEARN_LOG_LEVEL",
		"LEARN_LOG_FORMAT",
		"LEARN_CURRICULUM_PATH",
		"LEARN_DEV_MODE",
		"PAI_FEATURES",
		"PAI_AI_HEALTH_TOKEN",
		"LEARN_AI_PERSONALIZED_NUDGES_ENABLED",
		"LEARN_AI_MOCK_RESPONSE",
	}
	for _, v := range envVars {
		_ = os.Unsetenv(v)
	}
	t.Setenv("LEARN_AI_CODEX_HOME", filepath.Join(t.TempDir(), "codex"))
}

func setPrivateProductionAuth(t *testing.T) {
	t.Helper()
	t.Setenv("PAI_AUTH_SECRET", "private-test-secret")
	t.Setenv("PAI_AUTH_BOOTSTRAP_ADMIN_EMAIL", "owner@example.com")
	t.Setenv("PAI_AUTH_BOOTSTRAP_ADMIN_PASSWORD", "private-bootstrap-password")
}

func TestLoadAIHealthToken(t *testing.T) {
	clearEnv(t)
	t.Setenv("PAI_AI_HEALTH_TOKEN", "monitor-secret")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if cfg.Runtime.AIHealthToken != "monitor-secret" {
		t.Fatal("AI health token was not loaded")
	}
}

func TestLoad_FeatureFlagsRejectUnknown(t *testing.T) {
	clearEnv(t)
	t.Setenv("PAI_FEATURES", "unknown_feature")

	if _, err := Load(); err == nil {
		t.Fatal("Load() should reject unknown PAI_FEATURES entry")
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
	if cfg.Tenant.Mode != "single" {
		t.Errorf("Tenant.Mode = %q, want single", cfg.Tenant.Mode)
	}
	if cfg.Auth.Google.DiscoveryURL != "https://accounts.google.com/.well-known/openid-configuration" {
		t.Errorf("Auth.Google.DiscoveryURL = %q, want Google discovery URL", cfg.Auth.Google.DiscoveryURL)
	}
	if cfg.Auth.BootstrapAdmin.Email != "platform-admin@example.com" {
		t.Errorf("Auth.BootstrapAdmin.Email = %q, want platform-admin@example.com", cfg.Auth.BootstrapAdmin.Email)
	}
	if cfg.Auth.BootstrapAdmin.Password != "demo-password" {
		t.Errorf("Auth.BootstrapAdmin.Password = %q, want demo-password", cfg.Auth.BootstrapAdmin.Password)
	}
	if cfg.CurriculumPath != "./oss" {
		t.Errorf("CurriculumPath = %q, want ./oss", cfg.CurriculumPath)
	}
	if !cfg.Runtime.AIPersonalizedNudgesEnabled {
		t.Error("Runtime.AIPersonalizedNudgesEnabled should default to true")
	}
	if cfg.FeatureFlags.Enabled("unknown_feature") {
		t.Fatal("unknown feature should not be enabled")
	}
	if cfg.FeatureFlags.Enabled(featureflags.TurnHooks) {
		t.Fatal("turn_hooks should default to disabled")
	}
}

func TestLoad_FromEnv(t *testing.T) {
	clearEnv(t)

	t.Setenv("LEARN_SERVER_PORT", "9090")
	t.Setenv("LEARN_DATABASE_URL", "postgres://test:test@localhost/testdb")
	t.Setenv("LEARN_TELEGRAM_BOT_TOKEN", "test-token-123")
	t.Setenv("LEARN_SLACK_ENABLED", "true")
	t.Setenv("LEARN_SLACK_BOT_TOKEN", "xoxb-test-token")
	t.Setenv("LEARN_SLACK_SIGNING_SECRET", "slack-signing-secret")
	t.Setenv("LEARN_DISCORD_ENABLED", "true")
	t.Setenv("LEARN_DISCORD_BOT_TOKEN", "discord-test-token")
	t.Setenv("LEARN_DISCORD_PUBLIC_KEY", "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef")
	t.Setenv("LEARN_DISCORD_APPLICATION_ID", "discord-application-id")
	t.Setenv("LEARN_TEAMS_ENABLED", "true")
	t.Setenv("LEARN_TEAMS_APP_ID", "teams-app-id")
	t.Setenv("LEARN_TEAMS_APP_PASSWORD", "teams-app-password")
	t.Setenv("LEARN_TEAMS_APP_TENANT_ID", "teams-tenant-id")
	t.Setenv("LEARN_EMAIL_SMTP_ADDR", "smtp.example.com:587")
	t.Setenv("LEARN_EMAIL_SMTP_USERNAME", "mailer")
	t.Setenv("LEARN_EMAIL_SMTP_PASSWORD", "mailer-secret")
	t.Setenv("LEARN_EMAIL_FROM_ADDRESS", "bot@example.com")
	t.Setenv("LEARN_EMAIL_FROM_NAME", "Pandai Mailer")
	t.Setenv("LEARN_EMAIL_BASE_URL", "https://admin.example.com")
	t.Setenv("LEARN_EMBED_BASE_URL", "https://chat.example.com")
	t.Setenv("LEARN_AI_OPENAI_API_KEY", "sk-test-key")
	t.Setenv("LEARN_AI_OPENAI_MODEL", "gpt-4.1-mini")
	t.Setenv("LEARN_AI_MOCK_RESPONSE", "mock tutor response")
	t.Setenv("LEARN_AI_OLLAMA_URL", "http://localhost:11434")
	t.Setenv("LEARN_AI_OLLAMA_MODEL", "qwen3:14b")
	t.Setenv("LEARN_AI_CODEX_ENABLED", "true")
	t.Setenv("LEARN_AI_CODEX_HOME", "/tmp/pai-bot-codex")
	t.Setenv("LEARN_AI_CODEX_MODEL", "gpt-test")
	t.Setenv("LEARN_AI_DEFAULT_PROVIDER", "openrouter")
	t.Setenv("PAI_AUTH_SECRET", "super-secret")
	t.Setenv("PAI_CONFIG_ENCRYPTION_KEY", "runtime-settings-encryption-key-123")
	t.Setenv("PAI_AUTH_GOOGLE_CLIENT_ID", "google-client")
	t.Setenv("PAI_AUTH_GOOGLE_CLIENT_SECRET", "google-secret")
	t.Setenv("PAI_AUTH_GOOGLE_ALLOWED_DOMAIN", "pandai.org")
	t.Setenv("PAI_AUTH_GOOGLE_DISCOVERY_URL", "http://127.0.0.1:4002/.well-known/openid-configuration")
	t.Setenv("PAI_AUTH_GOOGLE_EMULATOR_SIGNING_SECRET", "emu-secret")
	t.Setenv("PAI_AUTH_GOOGLE_ADMIN_BASE_URL", "http://127.0.0.1:4178")
	t.Setenv("PAI_AUTH_BOOTSTRAP_ADMIN_EMAIL", "root@example.com")
	t.Setenv("PAI_AUTH_BOOTSTRAP_ADMIN_PASSWORD", "secret-bootstrap")
	t.Setenv("LEARN_TENANT_MODE", "multi")
	t.Setenv("LEARN_CURRICULUM_PATH", "/tmp/oss")
	t.Setenv("LEARN_AI_PERSONALIZED_NUDGES_ENABLED", "false")
	t.Setenv("PAI_FEATURES", "turn_hooks")

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
	if !cfg.Slack.Enabled {
		t.Error("Slack.Enabled should be true")
	}
	if cfg.Slack.BotToken != "xoxb-test-token" || cfg.Slack.SigningSecret != "slack-signing-secret" {
		t.Error("Slack credentials were not loaded")
	}
	if !cfg.Discord.Enabled ||
		cfg.Discord.BotToken != "discord-test-token" ||
		cfg.Discord.PublicKey != "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef" ||
		cfg.Discord.ApplicationID != "discord-application-id" {
		t.Error("Discord configuration was not loaded")
	}
	if !cfg.Teams.Enabled ||
		cfg.Teams.AppID != "teams-app-id" ||
		cfg.Teams.AppPassword != "teams-app-password" ||
		cfg.Teams.AppTenantID != "teams-tenant-id" {
		t.Error("Teams configuration was not loaded")
	}
	if cfg.Email.SMTPAddr != "smtp.example.com:587" {
		t.Errorf("Email.SMTPAddr = %q, want smtp.example.com:587", cfg.Email.SMTPAddr)
	}
	if cfg.Embed.BaseURL != "https://chat.example.com" {
		t.Errorf("Embed.BaseURL = %q, want https://chat.example.com", cfg.Embed.BaseURL)
	}
	if cfg.Email.SMTPUsername != "mailer" {
		t.Errorf("Email.SMTPUsername = %q, want mailer", cfg.Email.SMTPUsername)
	}
	if cfg.Email.SMTPPassword != "mailer-secret" {
		t.Errorf("Email.SMTPPassword = %q, want mailer-secret", cfg.Email.SMTPPassword)
	}
	if cfg.Email.FromAddress != "bot@example.com" {
		t.Errorf("Email.FromAddress = %q, want bot@example.com", cfg.Email.FromAddress)
	}
	if cfg.Email.FromName != "Pandai Mailer" {
		t.Errorf("Email.FromName = %q, want Pandai Mailer", cfg.Email.FromName)
	}
	if cfg.Email.BaseURL != "https://admin.example.com" {
		t.Errorf("Email.BaseURL = %q, want https://admin.example.com", cfg.Email.BaseURL)
	}
	if cfg.AI.OpenAI.APIKey != "sk-test-key" {
		t.Errorf("AI.OpenAI.APIKey = %q, want sk-test-key", cfg.AI.OpenAI.APIKey)
	}
	if cfg.AI.OpenAI.Model != "gpt-4.1-mini" {
		t.Errorf("AI.OpenAI.Model = %q, want gpt-4.1-mini", cfg.AI.OpenAI.Model)
	}
	if cfg.AI.Mock.Response != "mock tutor response" {
		t.Errorf("AI.Mock.Response = %q, want mock tutor response", cfg.AI.Mock.Response)
	}
	if cfg.AI.Ollama.URL != "http://localhost:11434" {
		t.Errorf("AI.Ollama.URL = %q, want http://localhost:11434", cfg.AI.Ollama.URL)
	}
	if cfg.AI.Ollama.Model != "qwen3:14b" {
		t.Errorf("AI.Ollama.Model = %q, want qwen3:14b", cfg.AI.Ollama.Model)
	}
	if !cfg.AI.Codex.Enabled ||
		cfg.AI.Codex.Home != "/tmp/pai-bot-codex" ||
		cfg.AI.Codex.Model != "gpt-test" {
		t.Errorf("AI.Codex = %#v", cfg.AI.Codex)
	}
	if cfg.AI.DefaultProvider != "openrouter" {
		t.Errorf("AI.DefaultProvider = %q, want openrouter", cfg.AI.DefaultProvider)
	}
	if cfg.Auth.JWTSecret != "super-secret" {
		t.Errorf("Auth.JWTSecret = %q, want super-secret", cfg.Auth.JWTSecret)
	}
	if cfg.Security.RuntimeSettingsEncryptionKey != "runtime-settings-encryption-key-123" {
		t.Error("Security.RuntimeSettingsEncryptionKey was not loaded")
	}
	if cfg.Auth.Google.ClientID != "google-client" {
		t.Errorf("Auth.Google.ClientID = %q, want google-client", cfg.Auth.Google.ClientID)
	}
	if cfg.Auth.Google.ClientSecret != "google-secret" {
		t.Errorf("Auth.Google.ClientSecret = %q, want google-secret", cfg.Auth.Google.ClientSecret)
	}
	if cfg.Auth.Google.AllowedDomain != "pandai.org" {
		t.Errorf("Auth.Google.AllowedDomain = %q, want pandai.org", cfg.Auth.Google.AllowedDomain)
	}
	if cfg.Auth.Google.DiscoveryURL != "http://127.0.0.1:4002/.well-known/openid-configuration" {
		t.Errorf("Auth.Google.DiscoveryURL = %q, want emulator discovery URL", cfg.Auth.Google.DiscoveryURL)
	}
	if cfg.Auth.Google.EmulatorSigningSecret != "emu-secret" {
		t.Errorf("Auth.Google.EmulatorSigningSecret = %q, want emu-secret", cfg.Auth.Google.EmulatorSigningSecret)
	}
	if cfg.Auth.Google.AdminBaseURL != "http://127.0.0.1:4178" {
		t.Errorf("Auth.Google.AdminBaseURL = %q, want http://127.0.0.1:4178", cfg.Auth.Google.AdminBaseURL)
	}
	if cfg.Auth.BootstrapAdmin.Email != "root@example.com" {
		t.Errorf("Auth.BootstrapAdmin.Email = %q, want root@example.com", cfg.Auth.BootstrapAdmin.Email)
	}
	if cfg.Auth.BootstrapAdmin.Password != "secret-bootstrap" {
		t.Errorf("Auth.BootstrapAdmin.Password = %q, want secret-bootstrap", cfg.Auth.BootstrapAdmin.Password)
	}
	if cfg.Tenant.Mode != "multi" {
		t.Errorf("Tenant.Mode = %q, want multi", cfg.Tenant.Mode)
	}
	if cfg.CurriculumPath != "/tmp/oss" {
		t.Errorf("CurriculumPath = %q, want /tmp/oss", cfg.CurriculumPath)
	}
	if cfg.Runtime.AIPersonalizedNudgesEnabled {
		t.Error("Runtime.AIPersonalizedNudgesEnabled should be false when configured")
	}
	if !cfg.FeatureFlags.Enabled(featureflags.TurnHooks) {
		t.Fatal("turn_hooks should be enabled from PAI_FEATURES")
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

func TestLoad_CodexProvider(t *testing.T) {
	clearEnv(t)
	t.Setenv("LEARN_AI_CODEX_ACCESS_TOKEN", "codex-access-token")
	t.Setenv("LEARN_AI_CODEX_REFRESH_TOKEN", "codex-refresh-token")
	t.Setenv("LEARN_AI_CODEX_ACCOUNT_ID", "codex-account-id")
	t.Setenv("LEARN_AI_CODEX_MODEL", "gpt-codex-test")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if cfg.AI.Codex.AccessToken != "codex-access-token" {
		t.Error("AI.Codex.AccessToken was not loaded")
	}
	if cfg.AI.Codex.RefreshToken != "codex-refresh-token" {
		t.Error("AI.Codex.RefreshToken was not loaded")
	}
	if cfg.AI.Codex.AccountID != "codex-account-id" {
		t.Errorf("AI.Codex.AccountID = %q, want codex-account-id", cfg.AI.Codex.AccountID)
	}
	if cfg.AI.Codex.Model != "gpt-codex-test" {
		t.Errorf("AI.Codex.Model = %q, want gpt-codex-test", cfg.AI.Codex.Model)
	}
}

func TestValidate_DefaultProvider(t *testing.T) {
	clearEnv(t)
	t.Setenv("LEARN_DEV_MODE", "true")
	t.Setenv("LEARN_AI_DEFAULT_PROVIDER", "codex")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if err := cfg.Validate(); err != nil {
		t.Fatalf("Validate() error = %v", err)
	}
}

func TestValidate_DefaultProvider_Invalid(t *testing.T) {
	clearEnv(t)
	t.Setenv("LEARN_DEV_MODE", "true")
	t.Setenv("LEARN_AI_DEFAULT_PROVIDER", "not-a-provider")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if err := cfg.Validate(); err == nil {
		t.Fatal("Validate() should reject unsupported LEARN_AI_DEFAULT_PROVIDER")
	}
}

func TestValidate_ProductionRequiresPrivateAuthConfiguration(t *testing.T) {
	base := Config{
		Runtime: RuntimeConfig{DevMode: false},
		Tenant:  TenantConfig{Mode: "single"},
		Telegram: TelegramConfig{
			BotToken: "telegram-token",
		},
		AI: AIConfig{
			Ollama: OllamaConfig{Enabled: true},
		},
		Auth: AuthConfig{
			JWTSecret: DefaultAuthSecret,
			BootstrapAdmin: BootstrapAdminConfig{
				Email:    "platform-admin@example.com",
				Password: "demo-password",
			},
		},
	}

	if err := base.Validate(); err == nil || !strings.Contains(err.Error(), "PAI_AUTH_SECRET") {
		t.Fatalf("default auth secret error = %v", err)
	}

	base.Auth.JWTSecret = "private-test-secret"
	if err := base.Validate(); err == nil || !strings.Contains(err.Error(), "PAI_AUTH_BOOTSTRAP") {
		t.Fatalf("default bootstrap credentials error = %v", err)
	}

	base.Auth.BootstrapAdmin = BootstrapAdminConfig{
		Email:    "owner@example.com",
		Password: "private-bootstrap-password",
	}
	if err := base.Validate(); err != nil {
		t.Fatalf("private production auth config error = %v", err)
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
		t.Fatal("Validate() should return error when no external chat adapter is configured")
	}
}

func TestValidate_AllowsProductionWithoutTelegram(t *testing.T) {
	tests := []struct {
		name string
		env  map[string]string
	}{
		{
			name: "Slack",
			env: map[string]string{
				"LEARN_SLACK_ENABLED":        "true",
				"LEARN_SLACK_BOT_TOKEN":      "xoxb-test",
				"LEARN_SLACK_SIGNING_SECRET": "signing-secret",
			},
		},
		{
			name: "Discord",
			env: map[string]string{
				"LEARN_DISCORD_ENABLED":        "true",
				"LEARN_DISCORD_BOT_TOKEN":      "discord-token",
				"LEARN_DISCORD_PUBLIC_KEY":     strings.Repeat("01", 32),
				"LEARN_DISCORD_APPLICATION_ID": "application-id",
			},
		},
		{
			name: "Teams",
			env: map[string]string{
				"LEARN_TEAMS_ENABLED":      "true",
				"LEARN_TEAMS_APP_ID":       "app-id",
				"LEARN_TEAMS_APP_PASSWORD": "app-password",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			clearEnv(t)
			setPrivateProductionAuth(t)
			t.Setenv("LEARN_AI_OLLAMA_ENABLED", "true")
			for name, value := range tt.env {
				t.Setenv(name, value)
			}

			cfg, err := Load()
			if err != nil {
				t.Fatalf("Load() error = %v", err)
			}
			if err := cfg.Validate(); err != nil {
				t.Fatalf("Validate() error = %v", err)
			}
		})
	}
}

func TestValidate_MissingAIProvider(t *testing.T) {
	clearEnv(t)
	t.Setenv("LEARN_TELEGRAM_BOT_TOKEN", "test-token")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	cfg.AI.Codex = CodexConfig{}

	if err := cfg.Validate(); err == nil {
		t.Fatal("Validate() should return error when no AI provider is configured")
	}
}

func TestValidate_DevModeAllowsMissingTelegramAndAI(t *testing.T) {
	clearEnv(t)
	t.Setenv("LEARN_DEV_MODE", "true")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if err := cfg.Validate(); err != nil {
		t.Fatalf("Validate() error = %v; should pass in dev mode", err)
	}
}

func TestLoad_ChatAdaptersAreDisabledByDefault(t *testing.T) {
	clearEnv(t)

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if cfg.Slack.Enabled || cfg.Discord.Enabled || cfg.Teams.Enabled {
		t.Fatalf("chat adapters should default to disabled: Slack=%t Discord=%t Teams=%t",
			cfg.Slack.Enabled, cfg.Discord.Enabled, cfg.Teams.Enabled)
	}
}

func TestValidate_ChatAdapterCredentialsAreAtomic(t *testing.T) {
	tests := []struct {
		name       string
		configure  func(*Config)
		wantFields []string
		secrets    []string
	}{
		{
			name: "Slack",
			configure: func(cfg *Config) {
				cfg.Slack.BotToken = "xoxb-secret-value"
			},
			wantFields: []string{"LEARN_SLACK_BOT_TOKEN", "LEARN_SLACK_SIGNING_SECRET"},
			secrets:    []string{"xoxb-secret-value"},
		},
		{
			name: "Discord",
			configure: func(cfg *Config) {
				cfg.Discord.Enabled = true
				cfg.Discord.PublicKey = "discord-public-key-value"
			},
			wantFields: []string{
				"LEARN_DISCORD_BOT_TOKEN",
				"LEARN_DISCORD_PUBLIC_KEY",
				"LEARN_DISCORD_APPLICATION_ID",
			},
			secrets: []string{"discord-public-key-value"},
		},
		{
			name: "Teams",
			configure: func(cfg *Config) {
				cfg.Teams.AppPassword = "teams-secret-value"
			},
			wantFields: []string{"LEARN_TEAMS_APP_ID", "LEARN_TEAMS_APP_PASSWORD"},
			secrets:    []string{"teams-secret-value"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &Config{
				Runtime: RuntimeConfig{DevMode: true},
				Tenant:  TenantConfig{Mode: "single"},
			}
			tt.configure(cfg)

			err := cfg.Validate()
			if err == nil {
				t.Fatal("Validate() should reject a partial adapter credential set")
			}
			for _, field := range tt.wantFields {
				if !strings.Contains(err.Error(), field) {
					t.Errorf("Validate() error = %q, want field %s", err, field)
				}
			}
			for _, secret := range tt.secrets {
				if strings.Contains(err.Error(), secret) {
					t.Errorf("Validate() error exposed credential %q", secret)
				}
			}
		})
	}
}

func TestValidate_ChatAdapterCredentialSets(t *testing.T) {
	cfg := &Config{
		Runtime: RuntimeConfig{DevMode: true},
		Tenant:  TenantConfig{Mode: "single"},
		Slack: SlackConfig{
			Enabled:       true,
			BotToken:      "xoxb-test-token",
			SigningSecret: "slack-signing-secret",
		},
		Discord: DiscordConfig{
			Enabled:       true,
			BotToken:      "discord-test-token",
			PublicKey:     "discord-public-key",
			ApplicationID: "discord-application-id",
		},
		Teams: TeamsConfig{
			Enabled:     true,
			AppID:       "teams-app-id",
			AppPassword: "teams-app-password",
		},
	}

	if err := cfg.Validate(); err != nil {
		t.Fatalf("Validate() error = %v", err)
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

func TestValidateConfigEncryptionKeyIsLongAndIndependent(t *testing.T) {
	base := Config{Runtime: RuntimeConfig{DevMode: true}, Tenant: TenantConfig{Mode: "single"}}
	base.Auth.JWTSecret = "shared-secret-value-that-is-long-enough"

	base.Security.RuntimeSettingsEncryptionKey = "too-short"
	if err := base.Validate(); err == nil || !strings.Contains(err.Error(), "at least 32") {
		t.Fatalf("short encryption key error = %v", err)
	}

	base.Security.RuntimeSettingsEncryptionKey = base.Auth.JWTSecret
	if err := base.Validate(); err == nil || !strings.Contains(err.Error(), "must differ") {
		t.Fatalf("shared encryption key error = %v", err)
	}

	base.Security.RuntimeSettingsEncryptionKey = strings.Repeat("a", 32)
	if err := base.Validate(); err == nil || !strings.Contains(err.Error(), "high-entropy") {
		t.Fatalf("weak encryption key error = %v", err)
	}

	base.Security.RuntimeSettingsEncryptionKey = "independent-runtime-settings-key-123"
	if err := base.Validate(); err != nil {
		t.Fatalf("independent encryption key error = %v", err)
	}
}

func TestLoadPreviousConfigEncryptionKeys(t *testing.T) {
	clearEnv(t)
	t.Setenv("PAI_CONFIG_PREVIOUS_ENCRYPTION_KEYS", `["previous-settings-encryption-key-one","previous-settings-encryption-key-two"]`)

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if len(cfg.Security.PreviousSettingsEncryptionKeys) != 2 ||
		cfg.Security.PreviousSettingsEncryptionKeys[0] != "previous-settings-encryption-key-one" {
		t.Fatalf("PreviousSettingsEncryptionKeys = %#v", cfg.Security.PreviousSettingsEncryptionKeys)
	}
}

func TestLoadRejectsMalformedPreviousConfigEncryptionKeys(t *testing.T) {
	clearEnv(t)
	t.Setenv("PAI_CONFIG_PREVIOUS_ENCRYPTION_KEYS", `not-json`)

	if _, err := Load(); err == nil || !strings.Contains(err.Error(), "JSON array") {
		t.Fatalf("Load() error = %v, want safe JSON-array error", err)
	}
}

func TestValidatePreviousConfigEncryptionKeys(t *testing.T) {
	base := Config{Runtime: RuntimeConfig{DevMode: true}, Tenant: TenantConfig{Mode: "single"}}
	base.Auth.JWTSecret = "auth-secret-value-that-is-long-enough"
	base.Security.RuntimeSettingsEncryptionKey = "active-settings-encryption-key-1234"

	tests := []struct {
		name string
		keys []string
	}{
		{name: "short", keys: []string{"short"}},
		{name: "active reused", keys: []string{base.Security.RuntimeSettingsEncryptionKey}},
		{name: "auth reused", keys: []string{base.Auth.JWTSecret}},
		{name: "duplicate", keys: []string{"previous-settings-encryption-key-123", "previous-settings-encryption-key-123"}},
		{name: "weak", keys: []string{strings.Repeat("a", 32)}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := base
			cfg.Security.PreviousSettingsEncryptionKeys = tt.keys
			if err := cfg.Validate(); err == nil {
				t.Fatal("Validate() should reject invalid previous encryption keys")
			}
		})
	}

	base.Security.PreviousSettingsEncryptionKeys = []string{"previous-settings-encryption-key-123"}
	if err := base.Validate(); err != nil {
		t.Fatalf("Validate() valid previous key error = %v", err)
	}
}

func TestValidate_EmailDeliveryRequiresSMTPAndFromAddress(t *testing.T) {
	clearEnv(t)
	t.Setenv("LEARN_TELEGRAM_BOT_TOKEN", "test-token")
	t.Setenv("LEARN_AI_OLLAMA_ENABLED", "true")
	t.Setenv("LEARN_EMAIL_BASE_URL", "https://admin.example.com")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if err := cfg.Validate(); err == nil {
		t.Fatal("Validate() should return error when email delivery is partially configured")
	}
}

func TestValidate_GoogleAdminBaseURLDoesNotConfigureEmailDelivery(t *testing.T) {
	clearEnv(t)
	t.Setenv("LEARN_DEV_MODE", "true")
	t.Setenv("PAI_AUTH_GOOGLE_ADMIN_BASE_URL", "http://127.0.0.1:4178")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if err := cfg.Validate(); err != nil {
		t.Fatalf("Validate() error = %v; Google admin redirects must not require SMTP", err)
	}
}

func TestValidateFocusedPageConfigurationIsPairedAndUsesPrivateSecret(t *testing.T) {
	base := Config{Runtime: RuntimeConfig{DevMode: true}, Tenant: TenantConfig{Mode: "single"}}
	base.FocusedPage.BaseURL = "https://pages.example"
	if err := base.Validate(); err == nil || !strings.Contains(err.Error(), "configured together") {
		t.Fatalf("one-sided focused page config error = %v", err)
	}

	base.FocusedPage.TelegramCTAURL = "https://t.me/pandai_bot"
	base.Auth.JWTSecret = DefaultAuthSecret
	if err := base.Validate(); err == nil || !strings.Contains(err.Error(), "PAI_AUTH_SECRET") {
		t.Fatalf("default focused page secret error = %v", err)
	}

	base.Auth.JWTSecret = "private-test-secret-value"
	if err := base.Validate(); err != nil {
		t.Fatalf("valid focused page config error = %v", err)
	}
}

func TestValidateEmbedBaseURLRequiresOrigin(t *testing.T) {
	base := Config{Runtime: RuntimeConfig{DevMode: true}, Tenant: TenantConfig{Mode: "single"}}
	base.Embed.BaseURL = "https://chat.example/path"
	if err := base.Validate(); err == nil || !strings.Contains(err.Error(), "LEARN_EMBED_BASE_URL") {
		t.Fatalf("Validate() error = %v, want embed base URL error", err)
	}
	base.Embed.BaseURL = "https://chat.example"
	if err := base.Validate(); err != nil {
		t.Fatalf("Validate() valid embed origin error = %v", err)
	}
}

func TestValidate_Success(t *testing.T) {
	clearEnv(t)
	setPrivateProductionAuth(t)
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
		{"Codex", "LEARN_AI_CODEX_ACCESS_TOKEN", "codex-test", true},
		{"Codex whitespace", "LEARN_AI_CODEX_ACCESS_TOKEN", "   ", false},
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

func TestHasAIProviderRecognizesEnabledCodexDeviceSetup(t *testing.T) {
	clearEnv(t)
	home := t.TempDir()
	t.Setenv("LEARN_AI_CODEX_ENABLED", "true")
	t.Setenv("LEARN_AI_CODEX_HOME", home)

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if !cfg.HasAIProvider() {
		t.Fatal("HasAIProvider() = false, want true for enabled Codex device setup")
	}
}

func TestHasAIProviderRejectsCodexDeviceSetupWithoutHome(t *testing.T) {
	cfg := Config{AI: AIConfig{Codex: CodexConfig{Enabled: true}}}
	if cfg.HasAIProvider() {
		t.Fatal("HasAIProvider() = true without Codex device storage")
	}
}

func TestValidateAllowsAdminCodexSetupWithoutExistingProvider(t *testing.T) {
	clearEnv(t)
	setPrivateProductionAuth(t)
	t.Setenv("LEARN_TELEGRAM_BOT_TOKEN", "test-token")
	t.Setenv("LEARN_AI_CODEX_ENABLED", "true")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if !cfg.HasAIProvider() {
		t.Fatal("HasAIProvider() = false for enabled Codex app-server setup")
	}
	if err := cfg.Validate(); err != nil {
		t.Fatalf("Validate() error = %v, want Admin Codex setup allowed", err)
	}
}

func TestValidateRejectsAdminCodexSetupWhenDisabled(t *testing.T) {
	clearEnv(t)
	t.Setenv("LEARN_TELEGRAM_BOT_TOKEN", "test-token")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if cfg.CodexDeviceAuthAvailable() {
		t.Fatal("CodexDeviceAuthAvailable() = true while Codex is disabled")
	}
	if err := cfg.Validate(); err == nil {
		t.Fatal("Validate() should reject zero providers while Codex is disabled")
	}
}

func TestValidateRejectsPersonalCodexHome(t *testing.T) {
	clearEnv(t)
	t.Setenv("LEARN_TELEGRAM_BOT_TOKEN", "test-token")
	t.Setenv("LEARN_AI_CODEX_ENABLED", "true")
	home, err := os.UserHomeDir()
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv("LEARN_AI_CODEX_HOME", filepath.Join(home, ".codex"))

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	err = cfg.Validate()
	if err == nil || !strings.Contains(err.Error(), "personal .codex") {
		t.Fatalf("Validate() error = %v, want personal Codex home rejection", err)
	}
}

func TestHasAIProvider_MockRequiresExplicitDefaultProvider(t *testing.T) {
	clearEnv(t)
	t.Setenv("LEARN_AI_MOCK_RESPONSE", "mock response")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if cfg.HasAIProvider() {
		t.Fatal("mock response alone should not count as a runtime AI provider")
	}

	t.Setenv("LEARN_AI_DEFAULT_PROVIDER", "mock")
	cfg, err = Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if !cfg.HasAIProvider() {
		t.Fatal("mock response should count when LEARN_AI_DEFAULT_PROVIDER=mock")
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
