// Copyright 2026 the P&AI authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package airouter

import (
	"log/slog"
	"strings"

	"github.com/p-n-ai/pai-bot/internal/ai"
	"github.com/p-n-ai/pai-bot/internal/platform/config"
)

var defaultProviderOrder = []string{"openai", "codex", "anthropic", "deepseek", "google", "ollama", "openrouter"}

// ProviderNames returns every provider name Apply can register.
func ProviderNames() []string {
	return append(append([]string(nil), defaultProviderOrder...), "mock")
}

// Setup builds an AI router from env-backed config, honoring a preferred
// default provider and per-provider default model selections.
func Setup(cfg config.AIConfig) *ai.Router {
	return SetupWithCodexAuth(cfg, nil)
}

// SetupWithCodexAuth builds a router whose managed Codex provider runs through app-server.
func SetupWithCodexAuth(cfg config.AIConfig, codexAuth ai.CodexAppServerClient) *ai.Router {
	router := ai.NewRouter()
	ApplyWithCodexAuth(router, cfg, codexAuth)
	return router
}

// Apply replaces the router's provider set from cfg; providers with no config (e.g. a cleared API key) unregister.
func Apply(router *ai.Router, cfg config.AIConfig) {
	ApplyWithCodexAuth(router, cfg, nil)
}

// ApplyWithCodexAuth replaces providers and gives Codex its managed-session refresher.
func ApplyWithCodexAuth(router *ai.Router, cfg config.AIConfig, codexAuth ai.CodexAppServerClient) {
	var regs []ai.ProviderRegistration
	for _, name := range providerOrder(cfg.DefaultProvider) {
		reg, ok := buildProviderWithCodexAuth(name, cfg, codexAuth)
		if !ok {
			continue
		}
		regs = append(regs, reg)
		slog.Info("AI provider registered", "provider", name, "model", strings.TrimSpace(reg.DefaultModel))
	}
	router.ReplaceProviders(regs)
}

// WouldRegister reports whether Apply would register provider name under cfg.
func WouldRegister(name string, cfg config.AIConfig) bool {
	_, ok := buildProvider(name, cfg)
	return ok
}

// HasProviderConfiguration reports whether name has credentials or managed
// app-server configuration that a fully wired runtime can register.
func HasProviderConfiguration(name string, cfg config.AIConfig) bool {
	if name == "codex" &&
		cfg.Codex.Enabled {
		return true
	}
	return WouldRegister(name, cfg)
}

func buildProvider(name string, cfg config.AIConfig) (ai.ProviderRegistration, bool) {
	return buildProviderWithCodexAuth(name, cfg, nil)
}

func buildProviderWithCodexAuth(
	name string,
	cfg config.AIConfig,
	codexAuth ai.CodexAppServerClient,
) (ai.ProviderRegistration, bool) {
	switch name {
	case "mock":
		if cfg.Mock.Response == "" {
			return ai.ProviderRegistration{}, false
		}
		return ai.ProviderRegistration{Name: name, Provider: ai.NewMockProvider(cfg.Mock.Response)}, true
	case "openai":
		if cfg.OpenAI.APIKey == "" {
			return ai.ProviderRegistration{}, false
		}
		return ai.ProviderRegistration{Name: name, Provider: ai.NewOpenAIProvider(cfg.OpenAI.APIKey), DefaultModel: cfg.OpenAI.Model}, true
	case "anthropic":
		if cfg.Anthropic.APIKey == "" {
			return ai.ProviderRegistration{}, false
		}
		provider, err := ai.NewAnthropicProvider(cfg.Anthropic.APIKey)
		if err != nil {
			slog.Warn("failed to create Anthropic provider", "error", err)
			return ai.ProviderRegistration{}, false
		}
		return ai.ProviderRegistration{Name: name, Provider: provider, DefaultModel: cfg.Anthropic.Model}, true
	case "deepseek":
		if cfg.DeepSeek.APIKey == "" {
			return ai.ProviderRegistration{}, false
		}
		return ai.ProviderRegistration{Name: name, Provider: ai.NewDeepSeekProvider(cfg.DeepSeek.APIKey), DefaultModel: cfg.DeepSeek.Model}, true
	case "google":
		if cfg.Google.APIKey == "" {
			return ai.ProviderRegistration{}, false
		}
		return ai.ProviderRegistration{Name: name, Provider: ai.NewGoogleProvider(cfg.Google.APIKey), DefaultModel: cfg.Google.Model}, true
	case "ollama":
		if !cfg.Ollama.Enabled {
			return ai.ProviderRegistration{}, false
		}
		return ai.ProviderRegistration{Name: name, Provider: ai.NewOllamaProvider(cfg.Ollama.URL), DefaultModel: cfg.Ollama.Model}, true
	case "openrouter":
		if cfg.OpenRouter.APIKey == "" {
			return ai.ProviderRegistration{}, false
		}
		return ai.ProviderRegistration{Name: name, Provider: ai.NewOpenRouterLLMAdapter(cfg.OpenRouter.APIKey), DefaultModel: cfg.OpenRouter.Model}, true
	case "codex":
		if cfg.Codex.Enabled {
			availability, ok := codexAuth.(interface{ Available() bool })
			if codexAuth == nil || !ok || !availability.Available() {
				slog.Warn("Codex provider not registered; Codex app-server is unavailable")
				return ai.ProviderRegistration{}, false
			}
			return ai.ProviderRegistration{
				Name:         name,
				Provider:     ai.NewCodexAppServerProvider(codexAuth, cfg.Codex.Model),
				DefaultModel: cfg.Codex.Model,
			}, true
		}
		if strings.TrimSpace(cfg.Codex.AccessToken) == "" {
			return ai.ProviderRegistration{}, false
		}
		provider, err := ai.NewCodexProvider(
			cfg.Codex.AccessToken,
			ai.WithCodexRefreshToken(cfg.Codex.RefreshToken),
			ai.WithCodexAccountID(cfg.Codex.AccountID),
		)
		if err != nil {
			slog.Warn("failed to create Codex provider", "error", err)
			return ai.ProviderRegistration{}, false
		}
		return ai.ProviderRegistration{Name: name, Provider: provider, DefaultModel: cfg.Codex.Model}, true
	}
	return ai.ProviderRegistration{}, false
}

func providerOrder(preferred string) []string {
	preferred = strings.ToLower(strings.TrimSpace(preferred))
	if preferred == "" {
		return append([]string(nil), defaultProviderOrder...)
	}

	order := make([]string, 0, len(defaultProviderOrder))
	order = append(order, preferred)
	for _, candidate := range defaultProviderOrder {
		if candidate == preferred {
			continue
		}
		order = append(order, candidate)
	}
	return order
}
