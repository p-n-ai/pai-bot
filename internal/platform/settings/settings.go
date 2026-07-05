// Copyright 2026 the P&AI authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

// Package settings owns DB-backed runtime settings layered over env config.
package settings

import (
	"github.com/p-n-ai/pai-bot/internal/platform/config"
	"github.com/p-n-ai/pai-bot/internal/platform/featureflags"
)

// AISettings holds admin-editable AI router overrides.
// OpenRouterAPIKey deliberately never serializes; it lives encrypted in the
// secrets column, not in the ai jsonb.
type AISettings struct {
	DefaultProvider  string `json:"default_provider"`
	OpenRouterModel  string `json:"openrouter_model"`
	OpenRouterAPIKey string `json:"-"`
}

// Settings is the full runtime settings document.
type Settings struct {
	AI    AISettings
	Flags map[string]bool
}

// MergeAI returns env with non-empty Settings fields overriding it; the DB
// wins over env, env is the seed.
func MergeAI(env config.AIConfig, st Settings) config.AIConfig {
	merged := env
	if st.AI.DefaultProvider != "" {
		merged.DefaultProvider = st.AI.DefaultProvider
	}
	if st.AI.OpenRouterModel != "" {
		merged.OpenRouter.Model = st.AI.OpenRouterModel
	}
	if st.AI.OpenRouterAPIKey != "" {
		merged.OpenRouter.APIKey = st.AI.OpenRouterAPIKey
	}
	return merged
}

// MergeFlags applies DB flag overrides on top of the env-derived feature set.
// Unknown flag names are rejected.
func MergeFlags(base featureflags.Features, overrides map[string]bool) (featureflags.Features, error) {
	return base.WithOverrides(overrides)
}

// KeyLast4 returns the last four characters of key for display.
func KeyLast4(key string) string {
	if len(key) <= 4 {
		return key
	}
	return key[len(key)-4:]
}
