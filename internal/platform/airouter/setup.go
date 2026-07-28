// Copyright 2026 the P&AI authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package airouter

import (
	"errors"
	"fmt"
	"log/slog"
	"strings"

	"github.com/p-n-ai/pai-bot/internal/ai"
	"github.com/p-n-ai/pai-bot/internal/platform/config"
)

var defaultProviderOrder = []string{"openai", "codex", "anthropic", "deepseek", "google", "ollama", "openrouter"}

// Plan is a fully constructed provider set that can be atomically installed
// without further validation or provider construction.
type Plan struct {
	registrations []ai.ProviderRegistration
}

// Apply atomically installs the prepared provider set.
func (p Plan) Apply(router *ai.Router) {
	router.ReplaceProviders(p.registrations)
}

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
	plan, err := PrepareWithCodexAuth(cfg, codexAuth)
	if err != nil {
		slog.Warn("preferred AI provider could not be prepared", "error", err)
	}
	plan.Apply(router)
}

// PrepareWithCodexAuth constructs the exact provider set to install and
// rejects a preferred provider that cannot be constructed.
func PrepareWithCodexAuth(cfg config.AIConfig, codexAuth ai.CodexAppServerClient) (Plan, error) {
	var regs []ai.ProviderRegistration
	preferred := strings.ToLower(strings.TrimSpace(cfg.DefaultProvider))
	var preferredErr error
	for _, name := range providerOrder(cfg.DefaultProvider) {
		reg, ok, err := buildProviderChecked(name, cfg, codexAuth)
		if err != nil {
			if name == preferred {
				preferredErr = fmt.Errorf("provider %q is not registrable", name)
			}
			slog.Warn("AI provider not registered", "provider", name)
			continue
		}
		if !ok {
			continue
		}
		regs = append(regs, reg)
		slog.Info("AI provider registered", "provider", name, "model", strings.TrimSpace(reg.DefaultModel))
	}
	return Plan{registrations: regs}, preferredErr
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

// CanRegister reports whether the fully wired runtime can register provider
// name. Managed Codex requires both enabled configuration and an available
// app-server session; other providers are determined entirely by cfg.
func CanRegister(name string, cfg config.AIConfig, managedCodexAvailable bool) bool {
	if name == "codex" && cfg.Codex.Enabled {
		return managedCodexAvailable
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
	reg, ok, err := buildProviderChecked(name, cfg, codexAuth)
	if err != nil {
		slog.Warn("AI provider not registered", "provider", name)
		return ai.ProviderRegistration{}, false
	}
	return reg, ok
}

func buildProviderChecked(
	name string,
	cfg config.AIConfig,
	codexAuth ai.CodexAppServerClient,
) (ai.ProviderRegistration, bool, error) {
	switch name {
	case "mock":
		if cfg.Mock.Response == "" {
			return ai.ProviderRegistration{}, false, nil
		}
		return ai.ProviderRegistration{Name: name, Provider: ai.NewMockProvider(cfg.Mock.Response)}, true, nil
	case "openai":
		if cfg.OpenAI.APIKey == "" {
			return ai.ProviderRegistration{}, false, nil
		}
		return ai.ProviderRegistration{Name: name, Provider: ai.NewOpenAIProvider(cfg.OpenAI.APIKey), DefaultModel: cfg.OpenAI.Model}, true, nil
	case "anthropic":
		if cfg.Anthropic.APIKey == "" {
			return ai.ProviderRegistration{}, false, nil
		}
		provider, err := ai.NewAnthropicProvider(cfg.Anthropic.APIKey)
		if err != nil {
			return ai.ProviderRegistration{}, true, err
		}
		return ai.ProviderRegistration{Name: name, Provider: provider, DefaultModel: cfg.Anthropic.Model}, true, nil
	case "deepseek":
		if cfg.DeepSeek.APIKey == "" {
			return ai.ProviderRegistration{}, false, nil
		}
		return ai.ProviderRegistration{Name: name, Provider: ai.NewDeepSeekProvider(cfg.DeepSeek.APIKey), DefaultModel: cfg.DeepSeek.Model}, true, nil
	case "google":
		if cfg.Google.APIKey == "" {
			return ai.ProviderRegistration{}, false, nil
		}
		return ai.ProviderRegistration{Name: name, Provider: ai.NewGoogleProvider(cfg.Google.APIKey), DefaultModel: cfg.Google.Model}, true, nil
	case "ollama":
		if !cfg.Ollama.Enabled {
			return ai.ProviderRegistration{}, false, nil
		}
		return ai.ProviderRegistration{Name: name, Provider: ai.NewOllamaProvider(cfg.Ollama.URL), DefaultModel: cfg.Ollama.Model}, true, nil
	case "openrouter":
		if cfg.OpenRouter.APIKey == "" {
			return ai.ProviderRegistration{}, false, nil
		}
		return ai.ProviderRegistration{Name: name, Provider: ai.NewOpenRouterLLMAdapter(cfg.OpenRouter.APIKey), DefaultModel: cfg.OpenRouter.Model}, true, nil
	case "codex":
		if cfg.Codex.Enabled {
			if codexAuth == nil || !codexAuth.Available() {
				return ai.ProviderRegistration{}, true, errors.New("managed Codex app-server is unavailable")
			}
			return ai.ProviderRegistration{
				Name:         name,
				Provider:     ai.NewCodexAppServerProvider(codexAuth, cfg.Codex.Model),
				DefaultModel: cfg.Codex.Model,
			}, true, nil
		}
		if strings.TrimSpace(cfg.Codex.AccessToken) == "" {
			return ai.ProviderRegistration{}, false, nil
		}
		provider, err := ai.NewCodexProvider(
			cfg.Codex.AccessToken,
			ai.WithCodexRefreshToken(cfg.Codex.RefreshToken),
			ai.WithCodexAccountID(cfg.Codex.AccountID),
		)
		if err != nil {
			return ai.ProviderRegistration{}, true, err
		}
		return ai.ProviderRegistration{Name: name, Provider: provider, DefaultModel: cfg.Codex.Model}, true, nil
	}
	return ai.ProviderRegistration{}, false, nil
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
