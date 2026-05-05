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

// Setup builds an AI router from env-backed config, honoring a preferred
// default provider and per-provider default model selections.
func Setup(cfg *config.Config) *ai.Router {
	router := ai.NewRouter()

	for _, name := range providerOrder(cfg.AI.DefaultProvider) {
		switch name {
		case "mock":
			if cfg.AI.Mock.Response == "" {
				continue
			}
			router.Register("mock", ai.NewMockProvider(cfg.AI.Mock.Response))
			slog.Info("AI provider registered", "provider", "mock")
		case "openai":
			if cfg.AI.OpenAI.APIKey == "" {
				continue
			}
			router.Register("openai", ai.NewOpenAIProvider(cfg.AI.OpenAI.APIKey))
			router.SetDefaultModel("openai", cfg.AI.OpenAI.Model)
			slog.Info("AI provider registered", "provider", "openai", "model", strings.TrimSpace(cfg.AI.OpenAI.Model))
		case "anthropic":
			if cfg.AI.Anthropic.APIKey == "" {
				continue
			}
			provider, err := ai.NewAnthropicProvider(cfg.AI.Anthropic.APIKey)
			if err != nil {
				slog.Warn("failed to create Anthropic provider", "error", err)
				continue
			}
			router.Register("anthropic", provider)
			router.SetDefaultModel("anthropic", cfg.AI.Anthropic.Model)
			slog.Info("AI provider registered", "provider", "anthropic", "model", strings.TrimSpace(cfg.AI.Anthropic.Model))
		case "deepseek":
			if cfg.AI.DeepSeek.APIKey == "" {
				continue
			}
			router.Register("deepseek", ai.NewDeepSeekProvider(cfg.AI.DeepSeek.APIKey))
			router.SetDefaultModel("deepseek", cfg.AI.DeepSeek.Model)
			slog.Info("AI provider registered", "provider", "deepseek", "model", strings.TrimSpace(cfg.AI.DeepSeek.Model))
		case "google":
			if cfg.AI.Google.APIKey == "" {
				continue
			}
			router.Register("google", ai.NewGoogleProvider(cfg.AI.Google.APIKey))
			router.SetDefaultModel("google", cfg.AI.Google.Model)
			slog.Info("AI provider registered", "provider", "google", "model", strings.TrimSpace(cfg.AI.Google.Model))
		case "ollama":
			if !cfg.AI.Ollama.Enabled {
				continue
			}
			router.Register("ollama", ai.NewOllamaProvider(cfg.AI.Ollama.URL))
			router.SetDefaultModel("ollama", cfg.AI.Ollama.Model)
			slog.Info("AI provider registered", "provider", "ollama", "model", strings.TrimSpace(cfg.AI.Ollama.Model))
		case "openrouter":
			if cfg.AI.OpenRouter.APIKey == "" {
				continue
			}
			router.Register("openrouter", ai.NewOpenRouterProvider(cfg.AI.OpenRouter.APIKey))
			router.SetDefaultModel("openrouter", cfg.AI.OpenRouter.Model)
			slog.Info("AI provider registered", "provider", "openrouter", "model", strings.TrimSpace(cfg.AI.OpenRouter.Model))
		}
	}

	return router
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
