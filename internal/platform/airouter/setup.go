// Copyright 2026 the P&AI authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package airouter

import (
	"log/slog"
	"strings"

	"github.com/p-n-ai/pai-bot/internal/ai"
	"github.com/p-n-ai/pai-bot/internal/platform/config"
)

var defaultProviderOrder = []string{"openai", "anthropic", "deepseek", "google", "ollama", "openrouter"}

// ProviderNames returns every provider name Apply can register.
func ProviderNames() []string {
	return append(append([]string(nil), defaultProviderOrder...), "mock")
}

// Setup builds an AI router from env-backed config, honoring a preferred
// default provider and per-provider default model selections.
func Setup(cfg config.AIConfig) *ai.Router {
	router := ai.NewRouter()
	Apply(router, cfg)
	return router
}

// Apply replaces the router's provider set from cfg; providers with no config (e.g. a cleared API key) unregister.
func Apply(router *ai.Router, cfg config.AIConfig) {
	var regs []ai.ProviderRegistration
	for _, name := range providerOrder(cfg.DefaultProvider) {
		reg, ok := buildProvider(name, cfg)
		if !ok {
			continue
		}
		regs = append(regs, reg)
		slog.Info("AI provider registered", "provider", name, "model", strings.TrimSpace(reg.DefaultModel))
	}
	router.ReplaceProviders(regs)
}

func buildProvider(name string, cfg config.AIConfig) (ai.ProviderRegistration, bool) {
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
		return ai.ProviderRegistration{Name: name, Provider: ai.NewOpenRouterProvider(cfg.OpenRouter.APIKey), DefaultModel: cfg.OpenRouter.Model}, true
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
