// Copyright 2026 the P&AI authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package airouter

import (
	"testing"

	"github.com/p-n-ai/pai-bot/internal/platform/config"
)

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
