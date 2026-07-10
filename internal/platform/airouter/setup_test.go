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
