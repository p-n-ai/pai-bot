// Copyright 2026 the P&AI authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package server

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"maps"
	"net/http"
	"slices"
	"strings"

	"github.com/p-n-ai/pai-bot/internal/platform/airouter"
	"github.com/p-n-ai/pai-bot/internal/platform/codexauth"
	"github.com/p-n-ai/pai-bot/internal/platform/config"
	"github.com/p-n-ai/pai-bot/internal/platform/featureflags"
	"github.com/p-n-ai/pai-bot/internal/platform/settings"
)

// runtimeSettingsStore is the seam the AI settings handlers need; the
// concrete *settings.Store satisfies it.
type runtimeSettingsStore interface {
	Current() settings.Settings
	Effective() settings.EffectiveSettings
	EffectiveFor(settings.Settings) settings.EffectiveSettings
	MergedAI(settings.Settings) config.AIConfig
	Update(ctx context.Context, mutate func(settings.Settings) (settings.Settings, error), prepare settings.PrepareApply) (settings.Settings, error)
}

type codexDeviceAuth interface {
	Status() codexauth.Status
	Start() (codexauth.Status, error)
	Available() bool
}

type aiSettingsKeyStatus struct {
	Set   bool   `json:"set"`
	Last4 string `json:"last4"`
}

type aiProviderSelector struct {
	Type settings.ProviderKind `json:"type"`
	Name string                `json:"name,omitempty"`
}

type aiSettingsResponse struct {
	DefaultProvider aiDefaultProviderProjection `json:"defaultProvider"`
	Providers       []any                       `json:"providers"`
	Flags           aiFlagsProjection           `json:"flags"`
	Revision        int64                       `json:"revision"`
	AppliedRevision int64                       `json:"appliedRevision"`
	Drift           bool                        `json:"drift"`
}

type aiDefaultProviderProjection struct {
	Baseline  *aiProviderSelector `json:"baseline"`
	Override  *aiProviderSelector `json:"override"`
	Effective *aiProviderSelector `json:"effective"`
	Source    string              `json:"source"`
}

type aiStringProjection struct {
	Baseline  *string `json:"baseline"`
	Override  *string `json:"override"`
	Effective *string `json:"effective"`
	Source    string  `json:"source"`
}

type aiBoolProjection struct {
	Baseline  bool   `json:"baseline"`
	Override  *bool  `json:"override"`
	Effective bool   `json:"effective"`
	Source    string `json:"source"`
}

type aiFlagsProjection struct {
	Baseline  map[string]bool   `json:"baseline"`
	Override  map[string]bool   `json:"override"`
	Effective map[string]bool   `json:"effective"`
	Sources   map[string]string `json:"sources"`
}

type aiProviderReadiness struct {
	Supported   bool   `json:"supported"`
	Configured  bool   `json:"configured"`
	Registrable bool   `json:"registrable"`
	Effective   bool   `json:"effective"`
	ManagedBy   string `json:"managedBy"`
}

type aiCredentialHealth struct {
	Stored          bool   `json:"stored"`
	Readable        bool   `json:"readable"`
	Version         string `json:"version"`
	Algorithm       string `json:"algorithm"`
	KeyID           string `json:"keyId"`
	MigrationNeeded bool   `json:"migrationNeeded"`
}

type aiCredentialProjection struct {
	Baseline  aiSettingsKeyStatus `json:"baseline"`
	Override  aiSettingsKeyStatus `json:"override"`
	Effective aiSettingsKeyStatus `json:"effective"`
	Source    string              `json:"source"`
	Health    aiCredentialHealth  `json:"health"`
}

type aiAPIKeyProviderProjection struct {
	Type        settings.ProviderKind  `json:"type"`
	Name        string                 `json:"name"`
	DisplayName string                 `json:"displayName"`
	Model       aiStringProjection     `json:"model"`
	Credential  aiCredentialProjection `json:"credential"`
	Readiness   aiProviderReadiness    `json:"readiness"`
}

type aiOllamaProviderProjection struct {
	Type      settings.ProviderKind `json:"type"`
	Enabled   aiBoolProjection      `json:"enabled"`
	Model     aiStringProjection    `json:"model"`
	Readiness aiProviderReadiness   `json:"readiness"`
}

type aiManagedCodexProviderProjection struct {
	Type      settings.ProviderKind `json:"type"`
	Enabled   aiBoolProjection      `json:"enabled"`
	Model     aiStringProjection    `json:"model"`
	Readiness aiProviderReadiness   `json:"readiness"`
}

type nullableString struct {
	Present bool
	Value   *string
}

func (v *nullableString) UnmarshalJSON(data []byte) error {
	v.Present = true
	if string(data) == "null" {
		v.Value = nil
		return nil
	}
	var value string
	if err := json.Unmarshal(data, &value); err != nil {
		return err
	}
	v.Value = &value
	return nil
}

type nullableBool struct {
	Present bool
	Value   *bool
}

func (v *nullableBool) UnmarshalJSON(data []byte) error {
	v.Present = true
	if string(data) == "null" {
		v.Value = nil
		return nil
	}
	var value bool
	if err := json.Unmarshal(data, &value); err != nil {
		return err
	}
	v.Value = &value
	return nil
}

type optionalJSON struct {
	Present bool
	Value   json.RawMessage
}

func (v *optionalJSON) UnmarshalJSON(data []byte) error {
	v.Present = true
	v.Value = append(v.Value[:0], data...)
	return nil
}

type optionalFlags struct {
	Present bool
	Value   map[string]*bool
}

func (v *optionalFlags) UnmarshalJSON(data []byte) error {
	v.Present = true
	if string(data) == "null" {
		return errors.New("flags must be an object")
	}
	return json.Unmarshal(data, &v.Value)
}

type aiSettingsUpdateWire struct {
	DefaultProvider  optionalJSON  `json:"defaultProvider"`
	Provider         optionalJSON  `json:"provider"`
	Flags            optionalFlags `json:"flags"`
	ExpectedRevision *int64        `json:"expectedRevision"`
}

type parsedAISettingsUpdate struct {
	DefaultProvider  optionalProviderSelector
	Provider         providerPatch
	Flags            optionalFlags
	ExpectedRevision int64
}

type optionalProviderSelector struct {
	Present bool
	Value   *aiProviderSelector
}

type providerPatch interface {
	providerPatch()
}

type apiKeyProviderPatch struct {
	Provider settings.APIKeyProvider
	Model    nullableString
	APIKey   nullableString
}

func (apiKeyProviderPatch) providerPatch() {}

type ollamaProviderPatch struct {
	Enabled nullableBool
	Model   nullableString
}

func (ollamaProviderPatch) providerPatch() {}

type managedCodexProviderPatch struct {
	Model nullableString
}

func (managedCodexProviderPatch) providerPatch() {}

func handleAdminGetAISettings(store runtimeSettingsStore, deviceAuth codexDeviceAuth) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		effective := store.Effective()
		writeJSON(w, http.StatusOK, buildAISettingsResponse(
			effective,
			store.MergedAI(store.Current()),
			codexAvailable(deviceAuth),
		))
	}
}

func handleAdminUpdateAISettings(store runtimeSettingsStore, prepareSettings settings.PrepareApply, deviceAuth codexDeviceAuth) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var wire aiSettingsUpdateWire
		if err := decodeStrictJSONBody(r, &wire); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		body, err := parseAISettingsUpdate(wire)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		var badReq error
		saved, err := store.Update(r.Context(), func(cur settings.Settings) (settings.Settings, error) {
			if body.ExpectedRevision != cur.Revision {
				return settings.Settings{}, settings.ErrRevisionConflict
			}
			next, err := applyAISettingsUpdate(cur, body)
			merged := store.MergedAI(next)
			if err == nil && merged.DefaultProvider != "" &&
				!airouter.CanRegister(merged.DefaultProvider, merged, codexAvailable(deviceAuth)) {
				err = fmt.Errorf("default provider %q has no usable configuration", merged.DefaultProvider)
			}
			if err == nil && providerPatchCanRemoveProvider(body.Provider) &&
				!anyProviderRegistrable(merged, codexAvailable(deviceAuth)) {
				err = errors.New("update would leave no AI providers configured")
			}
			badReq = err
			if err != nil {
				return settings.Settings{}, err
			}
			return next, nil
		}, prepareSettings)
		if badReq != nil {
			http.Error(w, badReq.Error(), http.StatusBadRequest)
			return
		}
		if err != nil {
			if errors.Is(err, settings.ErrRevisionConflict) {
				http.Error(w, err.Error(), http.StatusConflict)
				return
			}
			if errors.Is(err, settings.ErrConfigEncryptionKey) ||
				errors.Is(err, settings.ErrCredentialTooLarge) {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}
		writeJSON(w, http.StatusOK, buildAISettingsResponse(
			store.EffectiveFor(saved),
			store.MergedAI(saved),
			codexAvailable(deviceAuth),
		))
	}
}

func handleAdminGetCodexAuth(deviceAuth codexDeviceAuth) http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, http.StatusOK, deviceAuth.Status())
	}
}

func handleAdminStartCodexAuth(deviceAuth codexDeviceAuth) http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) {
		status, _ := deviceAuth.Start()
		writeJSON(w, http.StatusOK, status)
	}
}

func codexAvailable(deviceAuth codexDeviceAuth) bool {
	return deviceAuth != nil && deviceAuth.Available()
}

func anyProviderRegistrable(cfg config.AIConfig, managedCodexAvailable bool) bool {
	return slices.ContainsFunc(airouter.ProviderNames(), func(name string) bool {
		return airouter.CanRegister(name, cfg, managedCodexAvailable)
	})
}

// decodeStrictJSONBody mirrors decodeJSONBody but rejects unknown fields and
// trailing data, so a typoed field (e.g. GET's "openrouterKey") fails loudly
// instead of silently no-oping.
func decodeStrictJSONBody(r *http.Request, target any) (err error) {
	defer func() {
		closeErr := r.Body.Close()
		if err == nil && closeErr != nil {
			err = fmt.Errorf("close request body: %w", closeErr)
		}
	}()

	if err = decodeStrictJSON(r.Body, target); err != nil {
		return fmt.Errorf("invalid json body: %v", err)
	}
	return nil
}

func decodeStrictJSON(reader io.Reader, target any) error {
	dec := json.NewDecoder(reader)
	dec.DisallowUnknownFields()
	if err := dec.Decode(target); err != nil {
		return err
	}
	var trailing any
	if err := dec.Decode(&trailing); !errors.Is(err, io.EOF) {
		if err == nil {
			return errors.New("trailing data")
		}
		return err
	}
	return nil
}

func decodeStrictJSONBytes(data []byte, target any) error {
	return decodeStrictJSON(strings.NewReader(string(data)), target)
}

func parseAISettingsUpdate(wire aiSettingsUpdateWire) (parsedAISettingsUpdate, error) {
	if wire.ExpectedRevision == nil {
		return parsedAISettingsUpdate{}, errors.New("expectedRevision is required")
	}
	result := parsedAISettingsUpdate{
		Flags:            wire.Flags,
		ExpectedRevision: *wire.ExpectedRevision,
	}
	if wire.DefaultProvider.Present {
		result.DefaultProvider.Present = true
		if string(wire.DefaultProvider.Value) != "null" {
			selector, err := parseProviderSelector(wire.DefaultProvider.Value)
			if err != nil {
				return parsedAISettingsUpdate{}, fmt.Errorf("defaultProvider: %w", err)
			}
			result.DefaultProvider.Value = &selector
		}
	}
	if wire.Provider.Present {
		patch, err := parseProviderPatch(wire.Provider.Value)
		if err != nil {
			return parsedAISettingsUpdate{}, fmt.Errorf("provider: %w", err)
		}
		result.Provider = patch
	}
	return result, nil
}

func parseProviderSelector(raw json.RawMessage) (aiProviderSelector, error) {
	if string(raw) == "null" {
		return aiProviderSelector{}, errors.New("must be a provider selector or null")
	}
	var discriminator struct {
		Type settings.ProviderKind `json:"type"`
	}
	if err := json.Unmarshal(raw, &discriminator); err != nil {
		return aiProviderSelector{}, errors.New("must be an object")
	}
	switch discriminator.Type {
	case settings.ProviderKindAPIKey:
		var value struct {
			Type settings.ProviderKind `json:"type"`
			Name string                `json:"name"`
		}
		if err := decodeStrictJSONBytes(raw, &value); err != nil {
			return aiProviderSelector{}, fmt.Errorf("invalid api_key selector: %v", err)
		}
		provider, ok := settings.ParseAPIKeyProvider(value.Name)
		if !ok {
			return aiProviderSelector{}, fmt.Errorf("unknown api_key provider %q", value.Name)
		}
		return aiProviderSelector{Type: settings.ProviderKindAPIKey, Name: string(provider)}, nil
	case settings.ProviderKindOllama:
		var value struct {
			Type settings.ProviderKind `json:"type"`
		}
		if err := decodeStrictJSONBytes(raw, &value); err != nil {
			return aiProviderSelector{}, fmt.Errorf("invalid ollama selector: %v", err)
		}
		return aiProviderSelector{Type: settings.ProviderKindOllama}, nil
	case settings.ProviderKindManagedCodex:
		var value struct {
			Type settings.ProviderKind `json:"type"`
		}
		if err := decodeStrictJSONBytes(raw, &value); err != nil {
			return aiProviderSelector{}, fmt.Errorf("invalid managed_codex selector: %v", err)
		}
		return aiProviderSelector{Type: settings.ProviderKindManagedCodex}, nil
	default:
		return aiProviderSelector{}, fmt.Errorf("unknown provider type %q", discriminator.Type)
	}
}

func parseProviderPatch(raw json.RawMessage) (providerPatch, error) {
	if string(raw) == "null" {
		return nil, errors.New("must be a provider object; reset individual fields with null")
	}
	var discriminator struct {
		Type settings.ProviderKind `json:"type"`
	}
	if err := json.Unmarshal(raw, &discriminator); err != nil {
		return nil, errors.New("must be an object")
	}
	switch discriminator.Type {
	case settings.ProviderKindAPIKey:
		var value struct {
			Type   settings.ProviderKind `json:"type"`
			Name   string                `json:"name"`
			Model  nullableString        `json:"model"`
			APIKey nullableString        `json:"apiKey"`
		}
		if err := decodeStrictJSONBytes(raw, &value); err != nil {
			return nil, fmt.Errorf("invalid api_key patch: %v", err)
		}
		provider, ok := settings.ParseAPIKeyProvider(value.Name)
		if !ok {
			return nil, fmt.Errorf("unknown api_key provider %q", value.Name)
		}
		if !value.Model.Present && !value.APIKey.Present {
			return nil, errors.New("api_key patch must update model or apiKey")
		}
		if err := refineNullableString("model", &value.Model); err != nil {
			return nil, err
		}
		if err := refineNullableString("apiKey", &value.APIKey); err != nil {
			return nil, err
		}
		return apiKeyProviderPatch{Provider: provider, Model: value.Model, APIKey: value.APIKey}, nil
	case settings.ProviderKindOllama:
		var value struct {
			Type    settings.ProviderKind `json:"type"`
			Enabled nullableBool          `json:"enabled"`
			Model   nullableString        `json:"model"`
		}
		if err := decodeStrictJSONBytes(raw, &value); err != nil {
			return nil, fmt.Errorf("invalid ollama patch: %v", err)
		}
		if !value.Enabled.Present && !value.Model.Present {
			return nil, errors.New("ollama patch must update enabled or model")
		}
		if err := refineNullableString("model", &value.Model); err != nil {
			return nil, err
		}
		return ollamaProviderPatch{Enabled: value.Enabled, Model: value.Model}, nil
	case settings.ProviderKindManagedCodex:
		var value struct {
			Type  settings.ProviderKind `json:"type"`
			Model nullableString        `json:"model"`
		}
		if err := decodeStrictJSONBytes(raw, &value); err != nil {
			return nil, fmt.Errorf("invalid managed_codex patch: %v", err)
		}
		if !value.Model.Present {
			return nil, errors.New("managed_codex patch must update model")
		}
		if err := refineNullableString("model", &value.Model); err != nil {
			return nil, err
		}
		return managedCodexProviderPatch{Model: value.Model}, nil
	default:
		return nil, fmt.Errorf("unknown provider type %q", discriminator.Type)
	}
}

func refineNullableString(field string, value *nullableString) error {
	if !value.Present || value.Value == nil {
		return nil
	}
	trimmed := strings.TrimSpace(*value.Value)
	if trimmed == "" {
		return fmt.Errorf("%s must not be empty", field)
	}
	value.Value = &trimmed
	return nil
}

func providerSelectorName(selector aiProviderSelector) string {
	switch selector.Type {
	case settings.ProviderKindAPIKey:
		return selector.Name
	case settings.ProviderKindOllama:
		return "ollama"
	case settings.ProviderKindManagedCodex:
		return "codex"
	default:
		panic("unreachable provider selector")
	}
}

func providerPatchCanRemoveProvider(patch providerPatch) bool {
	switch patch := patch.(type) {
	case nil:
		return false
	case apiKeyProviderPatch:
		return patch.APIKey.Present && patch.APIKey.Value == nil
	case ollamaProviderPatch:
		return patch.Enabled.Present && (patch.Enabled.Value == nil || !*patch.Enabled.Value)
	case managedCodexProviderPatch:
		return false
	default:
		panic("unreachable provider patch")
	}
}

func applyAISettingsUpdate(st settings.Settings, req parsedAISettingsUpdate) (settings.Settings, error) {
	if req.DefaultProvider.Present {
		st.AI.DefaultProvider = nil
		if req.DefaultProvider.Value != nil {
			name := providerSelectorName(*req.DefaultProvider.Value)
			st.AI.DefaultProvider = &name
		}
	}
	switch patch := req.Provider.(type) {
	case nil:
	case apiKeyProviderPatch:
		if st.AI.Providers.APIKey == nil {
			st.AI.Providers.APIKey = make(map[settings.APIKeyProvider]settings.APIKeyProviderOverride)
		} else {
			st.AI.Providers.APIKey = maps.Clone(st.AI.Providers.APIKey)
		}
		override := st.AI.Providers.APIKey[patch.Provider]
		if patch.Model.Present {
			override.Model = patch.Model.Value
		}
		if override.Model == nil {
			delete(st.AI.Providers.APIKey, patch.Provider)
		} else {
			st.AI.Providers.APIKey[patch.Provider] = override
		}
		if patch.APIKey.Present {
			if st.AI.Credentials == nil {
				st.AI.Credentials = make(map[settings.APIKeyProvider]settings.CredentialOverride)
			} else {
				st.AI.Credentials = maps.Clone(st.AI.Credentials)
			}
			credential := st.AI.Credentials[patch.Provider]
			credential.Value = ""
			credential.Operation = settings.SecretClear
			if patch.APIKey.Value != nil {
				credential.Value = *patch.APIKey.Value
				credential.Operation = settings.SecretReplace
			}
			st.AI.Credentials[patch.Provider] = credential
		}
	case ollamaProviderPatch:
		override := cloneEnabledModelOverride(st.AI.Providers.Ollama)
		if patch.Enabled.Present {
			override.Enabled = patch.Enabled.Value
		}
		if patch.Model.Present {
			override.Model = patch.Model.Value
		}
		if override.Enabled == nil && override.Model == nil {
			st.AI.Providers.Ollama = nil
		} else {
			st.AI.Providers.Ollama = &override
		}
	case managedCodexProviderPatch:
		override := cloneModelOverride(st.AI.Providers.ManagedCodex)
		override.Model = patch.Model.Value
		if override.Model == nil {
			st.AI.Providers.ManagedCodex = nil
		} else {
			st.AI.Providers.ManagedCodex = &override
		}
	default:
		panic("unreachable provider patch")
	}
	if req.Flags.Present {
		// Null values only delete overrides, but their names must still be known.
		overrides := make(map[string]bool, len(req.Flags.Value))
		for name, v := range req.Flags.Value {
			overrides[name] = v != nil && *v
		}
		if _, err := (featureflags.Features{}).WithOverrides(overrides); err != nil {
			return settings.Settings{}, err
		}
		// Copy before writing: st.Flags may alias the caller's settings map.
		flags := make(map[string]bool, len(st.Flags)+len(req.Flags.Value))
		maps.Copy(flags, st.Flags)
		for name, v := range req.Flags.Value {
			if v == nil {
				delete(flags, name)
			} else {
				flags[name] = *v
			}
		}
		st.Flags = flags
	}
	return st, nil
}

func cloneEnabledModelOverride(value *settings.EnabledModelOverride) settings.EnabledModelOverride {
	if value == nil {
		return settings.EnabledModelOverride{}
	}
	return settings.EnabledModelOverride{
		Enabled: cloneBool(value.Enabled),
		Model:   cloneString(value.Model),
	}
}

func cloneModelOverride(value *settings.ModelOverride) settings.ModelOverride {
	if value == nil {
		return settings.ModelOverride{}
	}
	return settings.ModelOverride{Model: cloneString(value.Model)}
}

func cloneString(value *string) *string {
	if value == nil {
		return nil
	}
	copy := *value
	return &copy
}

func cloneBool(value *bool) *bool {
	if value == nil {
		return nil
	}
	copy := *value
	return &copy
}

// buildAISettingsResponse projects the closed provider model without ever
// including credential plaintext or ciphertext.
func buildAISettingsResponse(
	eff settings.EffectiveSettings,
	cfg config.AIConfig,
	managedCodexAvailable bool,
) aiSettingsResponse {
	providers := make([]any, 0, len(settings.APIKeyProviders())+2)
	for _, definition := range settings.APIKeyProviderDefinitions() {
		name := string(definition.ID)
		baseline := eff.Baseline.Providers[name]
		override := eff.Override.Providers[name]
		effective := eff.Effective.Providers[name]
		sources := eff.ProviderSources[name]
		providers = append(providers, aiAPIKeyProviderProjection{
			Type:        settings.ProviderKindAPIKey,
			Name:        name,
			DisplayName: definition.DisplayName,
			Model:       stringProjection(baseline.Model, override.Model, effective.Model, sources.Model),
			Credential: aiCredentialProjection{
				Baseline:  keyStatus(baseline.Credential),
				Override:  keyStatus(override.Credential),
				Effective: keyStatus(effective.Credential),
				Source:    sources.Credential,
				Health:    credentialHealth(eff.CredentialEnvelopes[name]),
			},
			Readiness: providerReadiness(name, eff, cfg, managedCodexAvailable),
		})
	}

	ollamaBaseline := eff.Baseline.Providers["ollama"]
	ollamaOverride := eff.Override.Providers["ollama"]
	ollamaEffective := eff.Effective.Providers["ollama"]
	ollamaSources := eff.ProviderSources["ollama"]
	providers = append(providers, aiOllamaProviderProjection{
		Type: settings.ProviderKindOllama,
		Enabled: aiBoolProjection{
			Baseline:  ollamaBaseline.Enabled,
			Override:  cloneBool(ollamaOverride.Enabled),
			Effective: ollamaEffective.Enabled,
			Source:    ollamaSources.Enabled,
		},
		Model:     stringProjection(ollamaBaseline.Model, ollamaOverride.Model, ollamaEffective.Model, ollamaSources.Model),
		Readiness: providerReadiness("ollama", eff, cfg, managedCodexAvailable),
	})

	codexBaseline := eff.Baseline.Providers["codex"]
	codexOverride := eff.Override.Providers["codex"]
	codexEffective := eff.Effective.Providers["codex"]
	codexSources := eff.ProviderSources["codex"]
	providers = append(providers, aiManagedCodexProviderProjection{
		Type: settings.ProviderKindManagedCodex,
		Enabled: aiBoolProjection{
			Baseline:  codexBaseline.Enabled,
			Override:  nil,
			Effective: codexEffective.Enabled,
			Source:    codexSources.Enabled,
		},
		Model:     stringProjection(codexBaseline.Model, codexOverride.Model, codexEffective.Model, codexSources.Model),
		Readiness: providerReadiness("codex", eff, cfg, managedCodexAvailable),
	})

	return aiSettingsResponse{
		DefaultProvider: aiDefaultProviderProjection{
			Baseline:  providerSelectorForName(eff.Baseline.DefaultProvider),
			Override:  providerSelectorForOptionalName(eff.Override.DefaultProvider),
			Effective: providerSelectorForName(eff.Effective.DefaultProvider),
			Source:    eff.DefaultProviderSource,
		},
		Providers: providers,
		Flags: aiFlagsProjection{
			Baseline:  eff.Baseline.Flags,
			Override:  eff.Override.Flags,
			Effective: eff.Effective.Flags,
			Sources:   eff.FlagSources,
		},
		Revision:        eff.Revision,
		AppliedRevision: eff.AppliedRevision,
		Drift:           eff.Drift,
	}
}

func stringProjection(baseline string, override *string, effective string, source string) aiStringProjection {
	return aiStringProjection{
		Baseline:  optionalNonEmptyString(baseline),
		Override:  cloneString(override),
		Effective: optionalNonEmptyString(effective),
		Source:    source,
	}
}

func optionalNonEmptyString(value string) *string {
	if value == "" {
		return nil
	}
	return &value
}

func keyStatus(value settings.SecretView) aiSettingsKeyStatus {
	return aiSettingsKeyStatus{Set: value.Set, Last4: value.Last4}
}

func credentialHealth(value settings.CredentialEnvelopeStatus) aiCredentialHealth {
	return aiCredentialHealth{
		Stored:          value.Stored,
		Readable:        value.Readable,
		Version:         value.Version,
		Algorithm:       value.Algorithm,
		KeyID:           value.KeyID,
		MigrationNeeded: value.MigrationNeeded,
	}
}

func providerReadiness(
	name string,
	eff settings.EffectiveSettings,
	cfg config.AIConfig,
	managedCodexAvailable bool,
) aiProviderReadiness {
	managedBy := "environment"
	sources := eff.ProviderSources[name]
	if sources.Enabled == settings.SourceDB ||
		sources.Model == settings.SourceDB ||
		sources.Credential == settings.SourceDB {
		managedBy = "runtime"
	}
	if name == "codex" {
		managedBy = "managed_codex"
	}
	return aiProviderReadiness{
		Supported:   true,
		Configured:  airouter.HasProviderConfiguration(name, cfg),
		Registrable: airouter.CanRegister(name, cfg, managedCodexAvailable),
		Effective:   name == eff.DefaultProvider,
		ManagedBy:   managedBy,
	}
}

func providerSelectorForOptionalName(name *string) *aiProviderSelector {
	if name == nil {
		return nil
	}
	return providerSelectorForName(*name)
}

func providerSelectorForName(name string) *aiProviderSelector {
	if provider, ok := settings.ParseAPIKeyProvider(name); ok {
		return &aiProviderSelector{Type: settings.ProviderKindAPIKey, Name: string(provider)}
	}
	switch name {
	case "ollama":
		return &aiProviderSelector{Type: settings.ProviderKindOllama}
	case "codex":
		return &aiProviderSelector{Type: settings.ProviderKindManagedCodex}
	default:
		return nil
	}
}
