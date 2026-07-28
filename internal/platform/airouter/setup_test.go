// Copyright 2026 the P&AI authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package airouter

import (
	"context"
	"reflect"
	"strings"
	"testing"

	"github.com/p-n-ai/pai-bot/internal/ai"
	"github.com/p-n-ai/pai-bot/internal/platform/config"
)

func TestBuildProviderUsesOpenRouterLLMAdapter(t *testing.T) {
	cfg := config.AIConfig{}
	cfg.OpenRouter.APIKey = "test-openrouter-key"
	cfg.OpenRouter.Model = "test-openrouter-model"

	reg, ok := buildProvider("openrouter", cfg)
	if !ok {
		t.Fatal("buildProvider(openrouter) = not registered with key set")
	}
	if got, want := reflect.TypeOf(reg.Provider), reflect.TypeOf(ai.NewOpenRouterLLMAdapter("")); got != want {
		t.Fatalf("OpenRouter provider type = %v, want %v", got, want)
	}
	if reg.Name != "openrouter" {
		t.Fatalf("OpenRouter registration name = %q, want openrouter", reg.Name)
	}
	if reg.DefaultModel != cfg.OpenRouter.Model {
		t.Fatalf("OpenRouter default model = %q, want %q", reg.DefaultModel, cfg.OpenRouter.Model)
	}
}

func TestBuildProviderUsesCodexCredentialsAndModel(t *testing.T) {
	cfg := config.AIConfig{
		Codex: config.CodexConfig{
			AccessToken:  "test-codex-access-token",
			RefreshToken: "test-codex-refresh-token",
			AccountID:    "test-codex-account-id",
			Model:        "test-codex-model",
		},
	}

	reg, ok := buildProvider("codex", cfg)
	if !ok {
		t.Fatal("buildProvider(codex) = not registered with access token set")
	}
	if _, ok := reg.Provider.(*ai.CodexProvider); !ok {
		t.Fatalf("Codex provider type = %T, want *ai.CodexProvider", reg.Provider)
	}
	if reg.Name != "codex" {
		t.Fatalf("Codex registration name = %q, want codex", reg.Name)
	}
	if reg.DefaultModel != cfg.Codex.Model {
		t.Fatalf("Codex default model = %q, want %q", reg.DefaultModel, cfg.Codex.Model)
	}
}

func TestBuildProviderUsesCodexAppServerWhenEnabled(t *testing.T) {
	cfg := config.AIConfig{}
	cfg.Codex.Enabled = true
	cfg.Codex.Model = "gpt-test"

	reg, ok := buildProviderWithCodexAuth("codex", cfg, stubCodexAuth{})
	if !ok {
		t.Fatal("buildProvider(codex) = not registered with app-server available")
	}
	if _, native := reg.Provider.(ai.NativeProvider); native {
		t.Fatalf("Codex app-server provider unexpectedly implements NativeProvider: %T", reg.Provider)
	}
	if reg.Name != "codex" || reg.DefaultModel != "gpt-test" {
		t.Fatalf("Codex registration = %#v", reg)
	}
}

type stubCodexAuth struct{}

func (stubCodexAuth) Refresh(context.Context) error { return nil }
func (stubCodexAuth) Available() bool               { return true }
func (stubCodexAuth) Complete(
	context.Context,
	ai.CompletionRequest,
) (ai.CompletionResponse, error) {
	return ai.CompletionResponse{Content: "ok"}, nil
}

func TestSetupWithCodexAuthRegistersManagedLoginAdapter(t *testing.T) {
	cfg := config.AIConfig{DefaultProvider: "codex"}
	cfg.Codex.Enabled = true

	router := SetupWithCodexAuth(cfg, stubCodexAuth{})

	if !router.HasProvider() {
		t.Fatal("Codex app-server provider was not registered")
	}
	if router.HasNativeProvider() {
		t.Fatal("Codex app-server provider must not expose PaiBot native tool continuation")
	}
}

func TestManagedCodexRequiresRuntimeManagerToRegister(t *testing.T) {
	cfg := config.AIConfig{DefaultProvider: "codex"}
	cfg.Codex.Enabled = true

	if Setup(cfg).HasProvider() {
		t.Fatal("Setup() registered managed Codex without app-server")
	}
	if !HasProviderConfiguration("codex", cfg) {
		t.Fatal("HasProviderConfiguration() = false for connected managed Codex")
	}
	if CanRegister("codex", cfg, false) {
		t.Fatal("CanRegister() = true without available managed Codex runtime")
	}
	if !CanRegister("codex", cfg, true) {
		t.Fatal("CanRegister() = false with available managed Codex runtime")
	}
}

type unavailableCodexAuth struct{ stubCodexAuth }

func (unavailableCodexAuth) Available() bool { return false }

func TestSetupWithUnavailableCodexAuthLeavesProviderUnregistered(t *testing.T) {
	cfg := config.AIConfig{DefaultProvider: "codex"}
	cfg.OpenAI.APIKey = "fallback-key"
	cfg.Codex.Enabled = true

	router := SetupWithCodexAuth(cfg, unavailableCodexAuth{})

	order := router.ProviderOrder()
	if len(order) != 1 || order[0] != "openai" {
		t.Fatalf("ProviderOrder() = %v, want [openai] fallback without Codex", order)
	}
}

func TestBuildProviderRejectsUnusableCodexCredentials(t *testing.T) {
	for _, test := range []struct {
		name   string
		config config.CodexConfig
	}{
		{name: "blank access token", config: config.CodexConfig{AccessToken: "   ", AccountID: "account-id"}},
		{name: "opaque token without account ID", config: config.CodexConfig{AccessToken: "opaque-token"}},
	} {
		t.Run(test.name, func(t *testing.T) {
			if _, ok := buildProvider("codex", config.AIConfig{Codex: test.config}); ok {
				t.Fatal("buildProvider(codex) registered unusable credentials")
			}
		})
	}
}

func TestProviderNamesIncludesCodexInDefaultOrder(t *testing.T) {
	names := ProviderNames()
	if len(names) < 2 || names[0] != "openai" || names[1] != "codex" {
		t.Fatalf("ProviderNames() = %v, want openai then codex", names)
	}
}

func TestProviderOrderSkipsMockByDefault(t *testing.T) {
	for _, provider := range providerOrder("") {
		if provider == "mock" {
			t.Fatal("mock provider should require explicit LEARN_AI_DEFAULT_PROVIDER=mock")
		}
	}
}

func TestProviderOrderAllowsExplicitMock(t *testing.T) {
	order := providerOrder("mock")
	if len(order) == 0 || order[0] != "mock" {
		t.Fatalf("providerOrder(mock) = %#v, want mock first", order)
	}
}

func TestApplyReordersLiveRouter(t *testing.T) {
	cfg := config.AIConfig{}
	cfg.OpenAI.APIKey = "test-openai-key"
	cfg.OpenRouter.APIKey = "test-openrouter-key"

	router := Setup(cfg)
	order := router.ProviderOrder()
	if len(order) != 2 || order[0] != "openai" || order[1] != "openrouter" {
		t.Fatalf("Setup order = %v, want [openai openrouter]", order)
	}

	cfg.DefaultProvider = "openrouter"
	Apply(router, cfg)

	order = router.ProviderOrder()
	if len(order) != 2 || order[0] != "openrouter" || order[1] != "openai" {
		t.Fatalf("Apply order = %v, want [openrouter openai] without duplicates", order)
	}
}

func TestWouldRegister(t *testing.T) {
	cfg := config.AIConfig{}
	cfg.OpenRouter.APIKey = "sk-or-test-key"

	if !WouldRegister("openrouter", cfg) {
		t.Fatal("WouldRegister(openrouter) = false with key set, want true")
	}
	for _, name := range []string{"anthropic", "openai", "unknown"} {
		if WouldRegister(name, cfg) {
			t.Fatalf("WouldRegister(%s) = true without config, want false", name)
		}
	}
}

func TestApplyUnregistersProviderWhenKeyCleared(t *testing.T) {
	cfg := config.AIConfig{}
	cfg.OpenRouter.APIKey = "sk-or-old-key"

	router := Setup(cfg)
	if order := router.ProviderOrder(); len(order) != 1 || order[0] != "openrouter" {
		t.Fatalf("Setup order = %v, want [openrouter]", order)
	}

	cfg.OpenRouter.APIKey = ""
	Apply(router, cfg)

	if router.HasProvider() {
		t.Fatalf("stale openrouter provider still registered after key clear: %v", router.ProviderOrder())
	}
	_, err := router.Complete(context.Background(), ai.CompletionRequest{
		Messages: []ai.Message{{Role: "user", Content: "hi"}},
	})
	if err == nil || !strings.Contains(err.Error(), "no providers registered") {
		t.Fatalf("Complete() error = %v, want no-providers failure without touching the stale provider", err)
	}
}

func TestPrepareRejectsUnbuildablePreferredProviderWithoutMutatingRouter(t *testing.T) {
	router := Setup(config.AIConfig{
		DefaultProvider: "mock",
		Mock:            config.MockAIConfig{Response: "existing"},
	})
	before := router.ProviderOrder()
	cfg := config.AIConfig{
		DefaultProvider: "codex",
		Codex: config.CodexConfig{
			Enabled: true,
		},
		Mock: config.MockAIConfig{Response: "replacement"},
	}

	if _, err := PrepareWithCodexAuth(cfg, nil); err == nil {
		t.Fatal("PrepareWithCodexAuth() error = nil, want unavailable managed Codex")
	}
	if got := router.ProviderOrder(); !reflect.DeepEqual(got, before) {
		t.Fatalf("router order after failed prepare = %v, want unchanged %v", got, before)
	}
}
