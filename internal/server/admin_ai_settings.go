// Copyright 2026 the P&AI authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package server

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
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

// aiSettingsSources tags each effective field: "db" | "env" | "none".
type aiSettingsSources struct {
	DefaultProvider string            `json:"defaultProvider"`
	OpenRouterModel string            `json:"openrouterModel"`
	OpenRouterKey   string            `json:"openrouterKey"`
	Flags           map[string]string `json:"flags"`
}

type aiSettingsResponse struct {
	DefaultProvider    string                `json:"defaultProvider"`
	OpenRouterModel    string                `json:"openrouterModel"`
	OpenRouterKey      aiSettingsKeyStatus   `json:"openrouterKey"`
	Flags              map[string]bool       `json:"flags"`
	Sources            aiSettingsSources     `json:"sources"`
	Baseline           aiSettingsView        `json:"baseline"`
	Override           aiSettingsOverride    `json:"override"`
	Effective          aiSettingsView        `json:"effective"`
	Revision           int64                 `json:"revision"`
	AppliedRevision    int64                 `json:"appliedRevision"`
	Drift              bool                  `json:"drift"`
	AvailableProviders []string              `json:"availableProviders"`
	Providers          []aiProviderReadiness `json:"providers"`
	Health             aiSettingsHealth      `json:"health"`
}

type aiProviderReadiness struct {
	Name        string `json:"name"`
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

type aiSettingsHealth struct {
	Revision        int64              `json:"revision"`
	AppliedRevision int64              `json:"appliedRevision"`
	Drift           bool               `json:"drift"`
	OpenRouterKey   aiCredentialHealth `json:"openrouterKey"`
}

type aiSettingsView struct {
	DefaultProvider string              `json:"defaultProvider"`
	OpenRouterModel string              `json:"openrouterModel"`
	OpenRouterKey   aiSettingsKeyStatus `json:"openrouterKey"`
	Flags           map[string]bool     `json:"flags"`
}

type aiSettingsOverride struct {
	DefaultProvider *string             `json:"defaultProvider"`
	OpenRouterModel *string             `json:"openrouterModel"`
	OpenRouterKey   aiSettingsKeyStatus `json:"openrouterKey"`
	Flags           map[string]bool     `json:"flags"`
}

// nullableString distinguishes an omitted update field from an explicit null,
// which deletes the database override and restores the environment baseline.
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

// A null scalar or flag value deletes the DB override so the field returns to
// environment/default control. Omitted fields remain unchanged.
type aiSettingsUpdateRequest struct {
	DefaultProvider  nullableString   `json:"defaultProvider"`
	OpenRouterModel  nullableString   `json:"openrouterModel"`
	OpenRouterAPIKey nullableString   `json:"openrouterApiKey"`
	Flags            map[string]*bool `json:"flags"`
	ExpectedRevision *int64           `json:"expectedRevision"`
}

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
		var body aiSettingsUpdateRequest
		if err := decodeStrictJSONBody(r, &body); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		var badReq error
		saved, err := store.Update(r.Context(), func(cur settings.Settings) (settings.Settings, error) {
			if body.ExpectedRevision != nil && *body.ExpectedRevision != cur.Revision {
				return settings.Settings{}, settings.ErrRevisionConflict
			}
			next, err := applyAISettingsUpdate(cur, body)
			// Only a request that sets defaultProvider is checked against the
			// merged config: clearing a key under a stale default must still work.
			if err == nil && body.DefaultProvider.Present && next.AI.DefaultProvider != "" &&
				!airouter.CanRegister(next.AI.DefaultProvider, store.MergedAI(next), codexAvailable(deviceAuth)) {
				err = fmt.Errorf("provider %q has no usable configuration", next.AI.DefaultProvider)
			}
			// Clearing the key is the only update that can remove a provider;
			// an empty router would crash-loop the next boot outside dev mode,
			// taking down the admin UI that could repair it.
			if err == nil && body.OpenRouterAPIKey.Present && next.AI.OpenRouterAPIKey == "" &&
				!anyProviderRegistrable(store.MergedAI(next), codexAvailable(deviceAuth)) {
				err = errors.New("clearing the API key would leave no AI providers configured")
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
			if errors.Is(err, settings.ErrConfigEncryptionKey) {
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

	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err = dec.Decode(target); err != nil {
		return fmt.Errorf("invalid json body: %v", err)
	}
	if dec.More() {
		return fmt.Errorf("invalid json body: trailing data")
	}
	return nil
}

// applyAISettingsUpdate merges the request onto current settings: absent
// fields stay unchanged, an empty openrouterApiKey clears the stored key.
func applyAISettingsUpdate(st settings.Settings, req aiSettingsUpdateRequest) (settings.Settings, error) {
	if req.DefaultProvider.Present {
		name := ""
		if req.DefaultProvider.Value != nil {
			name = strings.ToLower(strings.TrimSpace(*req.DefaultProvider.Value))
		}
		if name != "" && !slices.Contains(airouter.ProviderNames(), name) {
			return settings.Settings{}, fmt.Errorf("unknown provider %q", name)
		}
		st.AI.DefaultProvider = name
	}
	if req.OpenRouterModel.Present {
		st.AI.OpenRouterModel = ""
		if req.OpenRouterModel.Value != nil {
			st.AI.OpenRouterModel = strings.TrimSpace(*req.OpenRouterModel.Value)
		}
	}
	if req.OpenRouterAPIKey.Present {
		st.AI.OpenRouterAPIKey = ""
		st.AI.OpenRouterAPIKeyOperation = settings.SecretClear
		if req.OpenRouterAPIKey.Value != nil {
			st.AI.OpenRouterAPIKey = strings.TrimSpace(*req.OpenRouterAPIKey.Value)
			if st.AI.OpenRouterAPIKey != "" {
				st.AI.OpenRouterAPIKeyOperation = settings.SecretReplace
			}
		}
	}
	if req.Flags != nil {
		// Null values only delete overrides, but their names must still be known.
		overrides := make(map[string]bool, len(req.Flags))
		for name, v := range req.Flags {
			overrides[name] = v != nil && *v
		}
		if _, err := (featureflags.Features{}).WithOverrides(overrides); err != nil {
			return settings.Settings{}, err
		}
		// Copy before writing: st.Flags may alias the caller's settings map.
		flags := make(map[string]bool, len(st.Flags)+len(req.Flags))
		maps.Copy(flags, st.Flags)
		for name, v := range req.Flags {
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

// buildAISettingsResponse never includes the API key itself — only set/last4.
func buildAISettingsResponse(
	eff settings.EffectiveSettings,
	cfg config.AIConfig,
	managedCodexAvailable bool,
) aiSettingsResponse {
	providers := make([]aiProviderReadiness, 0, len(airouter.ProviderNames()))
	for _, name := range airouter.ProviderNames() {
		managedBy := "environment"
		if name == "openrouter" &&
			(eff.DefaultProviderSource == settings.SourceDB ||
				eff.OpenRouterModelSource == settings.SourceDB ||
				eff.OpenRouterKeySource == settings.SourceDB) {
			managedBy = "runtime"
		}
		if name == "codex" && cfg.Codex.Enabled {
			managedBy = "managed_codex"
		}
		providers = append(providers, aiProviderReadiness{
			Name:        name,
			Supported:   true,
			Configured:  airouter.HasProviderConfiguration(name, cfg),
			Registrable: airouter.CanRegister(name, cfg, managedCodexAvailable),
			Effective:   name == eff.DefaultProvider,
			ManagedBy:   managedBy,
		})
	}
	return aiSettingsResponse{
		DefaultProvider: eff.DefaultProvider,
		OpenRouterModel: eff.OpenRouterModel,
		OpenRouterKey:   aiSettingsKeyStatus{Set: eff.OpenRouterKeySet, Last4: eff.OpenRouterKeyLast4},
		Flags:           eff.Flags,
		Baseline: aiSettingsView{
			DefaultProvider: eff.Baseline.DefaultProvider,
			OpenRouterModel: eff.Baseline.OpenRouterModel,
			Flags:           eff.Baseline.Flags,
			OpenRouterKey: aiSettingsKeyStatus{
				Set:   eff.Baseline.OpenRouterKey.Set,
				Last4: eff.Baseline.OpenRouterKey.Last4,
			},
		},
		Override: aiSettingsOverride{
			DefaultProvider: eff.Override.DefaultProvider,
			OpenRouterModel: eff.Override.OpenRouterModel,
			Flags:           eff.Override.Flags,
			OpenRouterKey: aiSettingsKeyStatus{
				Set:   eff.Override.OpenRouterKey.Set,
				Last4: eff.Override.OpenRouterKey.Last4,
			},
		},
		Effective: aiSettingsView{
			DefaultProvider: eff.Effective.DefaultProvider,
			OpenRouterModel: eff.Effective.OpenRouterModel,
			Flags:           eff.Effective.Flags,
			OpenRouterKey: aiSettingsKeyStatus{
				Set:   eff.Effective.OpenRouterKey.Set,
				Last4: eff.Effective.OpenRouterKey.Last4,
			},
		},
		Revision:        eff.Revision,
		AppliedRevision: eff.AppliedRevision,
		Drift:           eff.Drift,
		Providers:       providers,
		Health: aiSettingsHealth{
			Revision:        eff.Revision,
			AppliedRevision: eff.AppliedRevision,
			Drift:           eff.Drift,
			OpenRouterKey: aiCredentialHealth{
				Stored:          eff.OpenRouterEnvelope.Stored,
				Readable:        eff.OpenRouterEnvelope.Readable,
				Version:         eff.OpenRouterEnvelope.Version,
				Algorithm:       eff.OpenRouterEnvelope.Algorithm,
				KeyID:           eff.OpenRouterEnvelope.KeyID,
				MigrationNeeded: eff.OpenRouterEnvelope.MigrationNeeded,
			},
		},
		Sources: aiSettingsSources{
			DefaultProvider: eff.DefaultProviderSource,
			OpenRouterModel: eff.OpenRouterModelSource,
			OpenRouterKey:   eff.OpenRouterKeySource,
			Flags:           eff.FlagSources,
		},
		AvailableProviders: airouter.ProviderNames(),
	}
}
