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

// Source tags where an effective settings value came from.
const (
	SourceDB   = "db"
	SourceEnv  = "env"
	SourceNone = "none"
)

// EffectiveSettings is the merged env+DB view the admin API reports.
type EffectiveSettings struct {
	DefaultProvider       string
	DefaultProviderSource string
	OpenRouterModel       string
	OpenRouterModelSource string
	OpenRouterAPIKey      string
	OpenRouterKeySource   string
	Flags                 map[string]bool
	FlagSources           map[string]string
}

// Effective merges env config and DB settings with DB > env > default precedence.
func Effective(envAI config.AIConfig, envFlags featureflags.Features, st Settings) EffectiveSettings {
	pick := func(db, env string) (string, string) {
		if db != "" {
			return db, SourceDB
		}
		if env != "" {
			return env, SourceEnv
		}
		return "", SourceNone
	}

	var eff EffectiveSettings
	eff.DefaultProvider, eff.DefaultProviderSource = pick(st.AI.DefaultProvider, envAI.DefaultProvider)
	eff.OpenRouterModel, eff.OpenRouterModelSource = pick(st.AI.OpenRouterModel, envAI.OpenRouter.Model)
	eff.OpenRouterAPIKey, eff.OpenRouterKeySource = pick(st.AI.OpenRouterAPIKey, envAI.OpenRouter.APIKey)

	defaults := featureflags.Defaults()
	eff.Flags = make(map[string]bool, len(defaults))
	eff.FlagSources = make(map[string]string, len(defaults))
	for name, defaultEnabled := range defaults {
		value, source := envFlags.Enabled(featureflags.Feature(name)), SourceNone
		if value != defaultEnabled {
			source = SourceEnv
		}
		if dbEnabled, ok := st.Flags[name]; ok {
			value, source = dbEnabled, SourceDB
		}
		eff.Flags[name] = value
		eff.FlagSources[name] = source
	}
	return eff
}

// KeyLast4 returns the last four characters of key for display; short keys yield "" so the hint never reveals most of the secret.
func KeyLast4(key string) string {
	if len(key) < 8 {
		return ""
	}
	return key[len(key)-4:]
}
