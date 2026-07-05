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

// Apply (re)registers providers and provider order from cfg onto a live
// router. It serves both boot and runtime settings changes; it never
// unregisters, so providers dropped from cfg keep their last registration.
func Apply(router *ai.Router, cfg config.AIConfig) {
	order := providerOrder(cfg.DefaultProvider)
	for _, name := range order {
		switch name {
		case "mock":
			if cfg.Mock.Response == "" {
				continue
			}
			router.Register("mock", ai.NewMockProvider(cfg.Mock.Response))
			slog.Info("AI provider registered", "provider", "mock")
		case "openai":
			if cfg.OpenAI.APIKey == "" {
				continue
			}
			router.Register("openai", ai.NewOpenAIProvider(cfg.OpenAI.APIKey))
			router.SetDefaultModel("openai", cfg.OpenAI.Model)
			slog.Info("AI provider registered", "provider", "openai", "model", strings.TrimSpace(cfg.OpenAI.Model))
		case "anthropic":
			if cfg.Anthropic.APIKey == "" {
				continue
			}
			provider, err := ai.NewAnthropicProvider(cfg.Anthropic.APIKey)
			if err != nil {
				slog.Warn("failed to create Anthropic provider", "error", err)
				continue
			}
			router.Register("anthropic", provider)
			router.SetDefaultModel("anthropic", cfg.Anthropic.Model)
			slog.Info("AI provider registered", "provider", "anthropic", "model", strings.TrimSpace(cfg.Anthropic.Model))
		case "deepseek":
			if cfg.DeepSeek.APIKey == "" {
				continue
			}
			router.Register("deepseek", ai.NewDeepSeekProvider(cfg.DeepSeek.APIKey))
			router.SetDefaultModel("deepseek", cfg.DeepSeek.Model)
			slog.Info("AI provider registered", "provider", "deepseek", "model", strings.TrimSpace(cfg.DeepSeek.Model))
		case "google":
			if cfg.Google.APIKey == "" {
				continue
			}
			router.Register("google", ai.NewGoogleProvider(cfg.Google.APIKey))
			router.SetDefaultModel("google", cfg.Google.Model)
			slog.Info("AI provider registered", "provider", "google", "model", strings.TrimSpace(cfg.Google.Model))
		case "ollama":
			if !cfg.Ollama.Enabled {
				continue
			}
			router.Register("ollama", ai.NewOllamaProvider(cfg.Ollama.URL))
			router.SetDefaultModel("ollama", cfg.Ollama.Model)
			slog.Info("AI provider registered", "provider", "ollama", "model", strings.TrimSpace(cfg.Ollama.Model))
		case "openrouter":
			if cfg.OpenRouter.APIKey == "" {
				continue
			}
			router.Register("openrouter", ai.NewOpenRouterProvider(cfg.OpenRouter.APIKey))
			router.SetDefaultModel("openrouter", cfg.OpenRouter.Model)
			slog.Info("AI provider registered", "provider", "openrouter", "model", strings.TrimSpace(cfg.OpenRouter.Model))
		}
	}
	router.SetProviderOrder(order)
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
