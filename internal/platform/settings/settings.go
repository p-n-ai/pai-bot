// Copyright 2026 the P&AI authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

// Package settings owns DB-backed runtime settings layered over env config.
package settings

import (
	"maps"

	"github.com/p-n-ai/pai-bot/internal/platform/config"
	"github.com/p-n-ai/pai-bot/internal/platform/featureflags"
)

// ProviderKind is the closed family of runtime-managed provider configurations.
type ProviderKind string

const (
	ProviderKindAPIKey       ProviderKind = "api_key"
	ProviderKindOllama       ProviderKind = "ollama"
	ProviderKindManagedCodex ProviderKind = "managed_codex"
)

// APIKeyProvider is a provider whose runtime credential is one API key.
type APIKeyProvider string

const (
	APIKeyProviderOpenAI     APIKeyProvider = "openai"
	APIKeyProviderAnthropic  APIKeyProvider = "anthropic"
	APIKeyProviderDeepSeek   APIKeyProvider = "deepseek"
	APIKeyProviderGoogle     APIKeyProvider = "google"
	APIKeyProviderOpenRouter APIKeyProvider = "openrouter"
	APIKeyProviderGroq       APIKeyProvider = "groq"
	APIKeyProviderXAI        APIKeyProvider = "xai"
	APIKeyProviderMistral    APIKeyProvider = "mistral"
	APIKeyProviderCerebras   APIKeyProvider = "cerebras"
)

var apiKeyProviders = []APIKeyProvider{
	APIKeyProviderOpenAI,
	APIKeyProviderAnthropic,
	APIKeyProviderDeepSeek,
	APIKeyProviderGoogle,
	APIKeyProviderOpenRouter,
	APIKeyProviderGroq,
	APIKeyProviderXAI,
	APIKeyProviderMistral,
	APIKeyProviderCerebras,
}

// APIKeyProviders returns the closed API-key provider set.
func APIKeyProviders() []APIKeyProvider {
	return append([]APIKeyProvider(nil), apiKeyProviders...)
}

// ParseAPIKeyProvider refines a persisted or HTTP provider name.
func ParseAPIKeyProvider(name string) (APIKeyProvider, bool) {
	provider := APIKeyProvider(name)
	for _, candidate := range apiKeyProviders {
		if provider == candidate {
			return provider, true
		}
	}
	return "", false
}

// APIKeyProviderOverride is the sparse non-secret override for an API-key provider.
type APIKeyProviderOverride struct {
	Model *string `json:"model,omitempty"`
}

// EnabledModelOverride is the sparse override shared by Ollama and managed Codex.
type EnabledModelOverride struct {
	Enabled *bool   `json:"enabled,omitempty"`
	Model   *string `json:"model,omitempty"`
}

// ModelOverride is the only runtime-writable managed Codex field. Enabling
// managed Codex also constructs authentication machinery and remains an
// environment-owned composition-root decision.
type ModelOverride struct {
	Model *string `json:"model,omitempty"`
}

// ProviderOverrides is the canonical closed provider settings document.
type ProviderOverrides struct {
	APIKey       map[APIKeyProvider]APIKeyProviderOverride `json:"api_key,omitempty"`
	Ollama       *EnabledModelOverride                     `json:"ollama,omitempty"`
	ManagedCodex *ModelOverride                            `json:"managed_codex,omitempty"`
}

// CredentialOverride holds one write-only API-key override and its safe metadata.
type CredentialOverride struct {
	Value     string
	Operation SecretUpdateOperation
	Envelope  CredentialEnvelopeStatus
}

// AISettings holds admin-editable AI router overrides.
// Credentials deliberately never serialize; they live encrypted in the
// secrets column, not in the ai jsonb.
type AISettings struct {
	DefaultProvider *string                               `json:"default_provider,omitempty"`
	Providers       ProviderOverrides                     `json:"providers,omitempty"`
	Credentials     map[APIKeyProvider]CredentialOverride `json:"-"`
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

// Source tags where an effective settings value came from.
const (
	SourceDB   = "db"
	SourceEnv  = "env"
	SourceNone = "none"
)

// SecretView is the only projection of a provider credential allowed outside
// the settings boundary.
type SecretView struct {
	Set   bool
	Last4 string
}

// ProviderSettingsView is one redacted effective or baseline provider projection.
type ProviderSettingsView struct {
	Kind       ProviderKind
	Name       string
	Enabled    bool
	Model      string
	Credential SecretView
}

// ProviderOverrideView reports sparse database ownership for one provider.
type ProviderOverrideView struct {
	Kind       ProviderKind
	Name       string
	Enabled    *bool
	Model      *string
	Credential SecretView
}

// ProviderSources reports ownership for each provider field.
type ProviderSources struct {
	Enabled    string
	Model      string
	Credential string
}

// AISettingsView is a redacted configuration projection.
type AISettingsView struct {
	DefaultProvider string
	Providers       map[string]ProviderSettingsView
	Flags           map[string]bool
}

// AISettingsOverrideView reports explicit database overrides.
type AISettingsOverrideView struct {
	DefaultProvider *string
	Providers       map[string]ProviderOverrideView
	Flags           map[string]bool
}

// EffectiveSettings is the reconciled env+DB view the admin API reports.
type EffectiveSettings struct {
	DefaultProvider       string
	DefaultProviderSource string
	Providers             map[string]ProviderSettingsView
	ProviderSources       map[string]ProviderSources
	Flags                 map[string]bool
	FlagSources           map[string]string
	Baseline              AISettingsView
	Override              AISettingsOverrideView
	Effective             AISettingsView
	Revision              int64
	AppliedRevision       int64
	Drift                 bool
	CredentialEnvelopes   map[string]CredentialEnvelopeStatus
}

// AIReconciliation is the one canonical provider precedence result.
type AIReconciliation struct {
	Config                config.AIConfig
	DefaultProvider       string
	DefaultProviderSource string
	Baseline              AISettingsView
	Override              AISettingsOverrideView
	Effective             AISettingsView
	ProviderSources       map[string]ProviderSources
	CredentialEnvelopes   map[string]CredentialEnvelopeStatus
}

// ReconcileAI applies sparse database overrides to the immutable environment baseline.
func ReconcileAI(env config.AIConfig, st AISettings) AIReconciliation {
	result := AIReconciliation{
		Config:                env,
		DefaultProvider:       env.DefaultProvider,
		DefaultProviderSource: sourceForString(env.DefaultProvider),
		Baseline: AISettingsView{
			DefaultProvider: env.DefaultProvider,
			Providers:       make(map[string]ProviderSettingsView, len(apiKeyProviders)+2),
		},
		Override: AISettingsOverrideView{
			DefaultProvider: cloneStringPointer(st.DefaultProvider),
			Providers:       make(map[string]ProviderOverrideView),
		},
		Effective: AISettingsView{
			DefaultProvider: env.DefaultProvider,
			Providers:       make(map[string]ProviderSettingsView, len(apiKeyProviders)+2),
		},
		ProviderSources:     make(map[string]ProviderSources, len(apiKeyProviders)+2),
		CredentialEnvelopes: make(map[string]CredentialEnvelopeStatus),
	}
	if st.DefaultProvider != nil {
		result.Config.DefaultProvider = *st.DefaultProvider
		result.DefaultProvider = *st.DefaultProvider
		result.Effective.DefaultProvider = *st.DefaultProvider
		result.DefaultProviderSource = SourceDB
	}

	for _, provider := range apiKeyProviders {
		name := string(provider)
		envModel, envKey := apiKeyConfig(env, provider)
		model, key := envModel, envKey
		modelSource, keySource := sourceForString(envModel), sourceForString(envKey)
		override := ProviderOverrideView{Kind: ProviderKindAPIKey, Name: name}
		if providerOverride, ok := st.Providers.APIKey[provider]; ok && providerOverride.Model != nil {
			model = *providerOverride.Model
			modelSource = SourceDB
			override.Model = cloneStringPointer(providerOverride.Model)
		}
		if credential, ok := st.Credentials[provider]; ok {
			if credential.Envelope.Stored && credential.Operation != SecretClear {
				key = ""
				keySource = SourceDB
				override.Credential.Set = true
				result.CredentialEnvelopes[name] = credential.Envelope
			}
			if credential.Value != "" {
				key = credential.Value
				keySource = SourceDB
				override.Credential = SecretView{Set: true, Last4: KeyLast4(credential.Value)}
			}
		}
		setAPIKeyConfig(&result.Config, provider, model, key)
		baseline := ProviderSettingsView{
			Kind: ProviderKindAPIKey, Name: name, Enabled: envKey != "",
			Model: envModel, Credential: SecretView{Set: envKey != ""},
		}
		effective := ProviderSettingsView{
			Kind: ProviderKindAPIKey, Name: name, Enabled: key != "",
			Model: model, Credential: SecretView{Set: key != "", Last4: KeyLast4(key)},
		}
		result.Baseline.Providers[name] = baseline
		result.Effective.Providers[name] = effective
		result.ProviderSources[name] = ProviderSources{
			Enabled: keySource, Model: modelSource, Credential: keySource,
		}
		if override.Model != nil || override.Credential.Set {
			result.Override.Providers[name] = override
		}
	}

	reconcileEnabledModel(
		"ollama", ProviderKindOllama, env.Ollama.Enabled, env.Ollama.Model,
		st.Providers.Ollama,
		func(enabled bool, model string) {
			result.Config.Ollama.Enabled = enabled
			result.Config.Ollama.Model = model
		},
		&result,
	)
	reconcileManagedCodex(env, st.Providers.ManagedCodex, &result)
	return result
}

func reconcileManagedCodex(env config.AIConfig, override *ModelOverride, result *AIReconciliation) {
	model, modelSource := env.Codex.Model, sourceForString(env.Codex.Model)
	view := ProviderOverrideView{Kind: ProviderKindManagedCodex, Name: "codex"}
	if override != nil && override.Model != nil {
		model = *override.Model
		modelSource = SourceDB
		view.Model = cloneStringPointer(override.Model)
	}
	result.Config.Codex.Model = model
	result.Baseline.Providers["codex"] = ProviderSettingsView{
		Kind: ProviderKindManagedCodex, Name: "codex",
		Enabled: env.Codex.Enabled, Model: env.Codex.Model,
	}
	result.Effective.Providers["codex"] = ProviderSettingsView{
		Kind: ProviderKindManagedCodex, Name: "codex",
		Enabled: env.Codex.Enabled, Model: model,
	}
	result.ProviderSources["codex"] = ProviderSources{
		Enabled: sourceForBool(env.Codex.Enabled),
		Model:   modelSource,
	}
	if view.Model != nil {
		result.Override.Providers["codex"] = view
	}
}

func reconcileEnabledModel(
	name string,
	kind ProviderKind,
	envEnabled bool,
	envModel string,
	override *EnabledModelOverride,
	apply func(bool, string),
	result *AIReconciliation,
) {
	enabled, model := envEnabled, envModel
	enabledSource, modelSource := SourceNone, sourceForString(envModel)
	view := ProviderOverrideView{Kind: kind, Name: name}
	if override != nil {
		if override.Enabled != nil {
			enabled = *override.Enabled
			enabledSource = SourceDB
			view.Enabled = cloneBoolPointer(override.Enabled)
		}
		if override.Model != nil {
			model = *override.Model
			modelSource = SourceDB
			view.Model = cloneStringPointer(override.Model)
		}
	}
	if enabledSource == SourceNone && envEnabled {
		enabledSource = SourceEnv
	}
	apply(enabled, model)
	result.Baseline.Providers[name] = ProviderSettingsView{
		Kind: kind, Name: name, Enabled: envEnabled, Model: envModel,
	}
	result.Effective.Providers[name] = ProviderSettingsView{
		Kind: kind, Name: name, Enabled: enabled, Model: model,
	}
	result.ProviderSources[name] = ProviderSources{Enabled: enabledSource, Model: modelSource, Credential: SourceNone}
	if view.Enabled != nil || view.Model != nil {
		result.Override.Providers[name] = view
	}
}

// MergeAI returns the canonical reconciled router configuration.
func MergeAI(env config.AIConfig, st Settings) config.AIConfig {
	return ReconcileAI(env, st.AI).Config
}

// Effective reconciles env config and DB settings with DB > env > default precedence.
func Effective(envAI config.AIConfig, envFlags featureflags.Features, st Settings) EffectiveSettings {
	ai := ReconcileAI(envAI, st.AI)
	eff := EffectiveSettings{
		DefaultProvider:       ai.DefaultProvider,
		DefaultProviderSource: ai.DefaultProviderSource,
		Providers:             maps.Clone(ai.Effective.Providers),
		ProviderSources:       maps.Clone(ai.ProviderSources),
		Baseline:              ai.Baseline,
		Override:              ai.Override,
		Effective:             ai.Effective,
		Revision:              st.Revision,
		CredentialEnvelopes:   maps.Clone(ai.CredentialEnvelopes),
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

func apiKeyConfig(cfg config.AIConfig, provider APIKeyProvider) (model, key string) {
	switch provider {
	case APIKeyProviderOpenAI:
		return cfg.OpenAI.Model, cfg.OpenAI.APIKey
	case APIKeyProviderAnthropic:
		return cfg.Anthropic.Model, cfg.Anthropic.APIKey
	case APIKeyProviderDeepSeek:
		return cfg.DeepSeek.Model, cfg.DeepSeek.APIKey
	case APIKeyProviderGoogle:
		return cfg.Google.Model, cfg.Google.APIKey
	case APIKeyProviderOpenRouter:
		return cfg.OpenRouter.Model, cfg.OpenRouter.APIKey
	case APIKeyProviderGroq, APIKeyProviderXAI, APIKeyProviderMistral, APIKeyProviderCerebras:
		providerConfig := cfg.CatalogProviders[string(provider)]
		return providerConfig.Model, providerConfig.APIKey
	}
	panic("unreachable API-key provider")
}

func setAPIKeyConfig(cfg *config.AIConfig, provider APIKeyProvider, model, key string) {
	switch provider {
	case APIKeyProviderOpenAI:
		cfg.OpenAI.Model, cfg.OpenAI.APIKey = model, key
	case APIKeyProviderAnthropic:
		cfg.Anthropic.Model, cfg.Anthropic.APIKey = model, key
	case APIKeyProviderDeepSeek:
		cfg.DeepSeek.Model, cfg.DeepSeek.APIKey = model, key
	case APIKeyProviderGoogle:
		cfg.Google.Model, cfg.Google.APIKey = model, key
	case APIKeyProviderOpenRouter:
		cfg.OpenRouter.Model, cfg.OpenRouter.APIKey = model, key
	case APIKeyProviderGroq, APIKeyProviderXAI, APIKeyProviderMistral, APIKeyProviderCerebras:
		if cfg.CatalogProviders == nil {
			cfg.CatalogProviders = make(map[string]config.CatalogProviderConfig)
		}
		cfg.CatalogProviders[string(provider)] = config.CatalogProviderConfig{Model: model, APIKey: key}
	default:
		panic("unreachable API-key provider")
	}
}

func sourceForString(value string) string {
	if value == "" {
		return SourceNone
	}
	return SourceEnv
}

func sourceForBool(value bool) string {
	if value {
		return SourceEnv
	}
	return SourceNone
}

func cloneStringPointer(value *string) *string {
	if value == nil {
		return nil
	}
	copy := *value
	return &copy
}

func cloneBoolPointer(value *bool) *bool {
	if value == nil {
		return nil
	}
	copy := *value
	return &copy
}

// KeyLast4 returns the last four characters of key for display; short keys yield "" so the hint never reveals most of the secret.
func KeyLast4(key string) string {
	if len(key) < 8 {
		return ""
	}
	return key[len(key)-4:]
}
