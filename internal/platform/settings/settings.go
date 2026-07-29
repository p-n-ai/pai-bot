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
	DefaultProvider           string                   `json:"default_provider,omitempty"`
	OpenRouterModel           string                   `json:"openrouter_model,omitempty"`
	OpenRouterAPIKey          string                   `json:"-"`
	OpenRouterAPIKeyOperation SecretUpdateOperation    `json:"-"`
	OpenRouterEnvelope        CredentialEnvelopeStatus `json:"-"`
}

// CredentialEnvelopeStatus is safe operational metadata for one stored
// credential. It never contains ciphertext or plaintext characteristics.
type CredentialEnvelopeStatus struct {
	Stored          bool
	Readable        bool
	Version         string
	Algorithm       string
	KeyID           string
	MigrationNeeded bool
}

// SecretUpdateOperation distinguishes an omitted secret from an explicit
// clear when an existing ciphertext cannot be decrypted.
type SecretUpdateOperation uint8

const (
	SecretPreserve SecretUpdateOperation = iota
	SecretReplace
	SecretClear
)

// Settings is the full runtime settings document.
type Settings struct {
	AI       AISettings
	Flags    map[string]bool
	Revision int64
}

// PreparedApply is a non-failing runtime mutation built and validated before
// the desired settings transaction commits.
type PreparedApply func()

// PrepareApply validates runtime construction without mutating live state and
// returns the exact atomic mutation to run after commit.
type PrepareApply func(Settings) (PreparedApply, error)

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
	OpenRouterKeySet      bool
	OpenRouterKeyLast4    string
	OpenRouterKeySource   string
	Flags                 map[string]bool
	FlagSources           map[string]string
	Baseline              AISettingsView
	Override              AISettingsOverrideView
	Effective             AISettingsView
	Revision              int64
	AppliedRevision       int64
	Drift                 bool
	OpenRouterEnvelope    CredentialEnvelopeStatus
}

// SecretView is the only projection of a provider credential allowed outside
// the settings boundary.
type SecretView struct {
	Set   bool
	Last4 string
}

// AISettingsView is a redacted configuration projection.
type AISettingsView struct {
	DefaultProvider string
	OpenRouterModel string
	OpenRouterKey   SecretView
	Flags           map[string]bool
}

// AISettingsOverrideView reports explicit database overrides. Nil scalar
// fields mean the environment baseline remains in control.
type AISettingsOverrideView struct {
	DefaultProvider *string
	OpenRouterModel *string
	OpenRouterKey   SecretView
	Flags           map[string]bool
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
	eff.Revision = st.Revision
	eff.OpenRouterEnvelope = st.AI.OpenRouterEnvelope
	eff.DefaultProvider, eff.DefaultProviderSource = pick(st.AI.DefaultProvider, envAI.DefaultProvider)
	eff.OpenRouterModel, eff.OpenRouterModelSource = pick(st.AI.OpenRouterModel, envAI.OpenRouter.Model)
	openRouterKey, openRouterKeySource := pick(st.AI.OpenRouterAPIKey, envAI.OpenRouter.APIKey)
	eff.OpenRouterKeySet = openRouterKey != ""
	eff.OpenRouterKeyLast4 = KeyLast4(openRouterKey)
	eff.OpenRouterKeySource = openRouterKeySource
	eff.Baseline = AISettingsView{
		DefaultProvider: envAI.DefaultProvider,
		OpenRouterModel: envAI.OpenRouter.Model,
		OpenRouterKey: SecretView{
			Set: envAI.OpenRouter.APIKey != "",
		},
	}
	if st.AI.DefaultProvider != "" {
		value := st.AI.DefaultProvider
		eff.Override.DefaultProvider = &value
	}
	if st.AI.OpenRouterModel != "" {
		value := st.AI.OpenRouterModel
		eff.Override.OpenRouterModel = &value
	}
	eff.Override.OpenRouterKey = SecretView{
		Set:   st.AI.OpenRouterAPIKey != "",
		Last4: KeyLast4(st.AI.OpenRouterAPIKey),
	}
	eff.Effective = AISettingsView{
		DefaultProvider: eff.DefaultProvider,
		OpenRouterModel: eff.OpenRouterModel,
		OpenRouterKey: SecretView{
			Set:   eff.OpenRouterKeySet,
			Last4: eff.OpenRouterKeyLast4,
		},
	}

	defaults := featureflags.Defaults()
	eff.Flags = make(map[string]bool, len(defaults))
	eff.FlagSources = make(map[string]string, len(defaults))
	eff.Baseline.Flags = make(map[string]bool, len(defaults))
	eff.Override.Flags = make(map[string]bool, len(st.Flags))
	eff.Effective.Flags = make(map[string]bool, len(defaults))
	for name := range defaults {
		feature := featureflags.Feature(name)
		value, source := envFlags.Enabled(feature), SourceNone
		if _, explicitlyConfigured := envFlags.Override(feature); explicitlyConfigured {
			source = SourceEnv
		}
		if dbEnabled, ok := st.Flags[name]; ok {
			value, source = dbEnabled, SourceDB
			eff.Override.Flags[name] = dbEnabled
		}
		eff.Flags[name] = value
		eff.FlagSources[name] = source
		eff.Baseline.Flags[name] = envFlags.Enabled(feature)
		eff.Effective.Flags[name] = value
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
