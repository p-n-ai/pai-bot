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
	"github.com/p-n-ai/pai-bot/internal/platform/config"
	"github.com/p-n-ai/pai-bot/internal/platform/featureflags"
	"github.com/p-n-ai/pai-bot/internal/platform/settings"
)

// runtimeSettingsStore is the seam the AI settings handlers need; the
// concrete *settings.Store satisfies it.
type runtimeSettingsStore interface {
	Effective() settings.EffectiveSettings
	MergedAI(settings.Settings) config.AIConfig
	// Update must run apply (nil ok) before releasing its write lock so live re-applies happen in save order.
	Update(ctx context.Context, mutate func(settings.Settings) (settings.Settings, error), apply func(settings.Settings)) (settings.Settings, error)
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
	DefaultProvider    string              `json:"defaultProvider"`
	OpenRouterModel    string              `json:"openrouterModel"`
	OpenRouterKey      aiSettingsKeyStatus `json:"openrouterKey"`
	Flags              map[string]bool     `json:"flags"`
	Sources            aiSettingsSources   `json:"sources"`
	AvailableProviders []string            `json:"availableProviders"`
}

// A null flag value deletes the DB override so the flag returns to env control.
type aiSettingsUpdateRequest struct {
	DefaultProvider  *string          `json:"defaultProvider"`
	OpenRouterModel  *string          `json:"openrouterModel"`
	OpenRouterAPIKey *string          `json:"openrouterApiKey"`
	Flags            map[string]*bool `json:"flags"`
}

func handleAdminGetAISettings(store runtimeSettingsStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, buildAISettingsResponse(store.Effective()))
	}
}

func handleAdminUpdateAISettings(store runtimeSettingsStore, applySettings func(settings.Settings)) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var body aiSettingsUpdateRequest
		if err := decodeStrictJSONBody(r, &body); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		var badReq error
		_, err := store.Update(r.Context(), func(cur settings.Settings) (settings.Settings, error) {
			next, err := applyAISettingsUpdate(cur, body)
			// Only a request that sets defaultProvider is checked against the
			// merged config: clearing a key under a stale default must still work.
			if err == nil && body.DefaultProvider != nil && next.AI.DefaultProvider != "" &&
				!airouter.WouldRegister(next.AI.DefaultProvider, store.MergedAI(next)) {
				err = fmt.Errorf("provider %q has no usable configuration", next.AI.DefaultProvider)
			}
			// Clearing the key is the only update that can remove a provider;
			// an empty router would crash-loop the next boot outside dev mode,
			// taking down the admin UI that could repair it.
			if err == nil && body.OpenRouterAPIKey != nil && next.AI.OpenRouterAPIKey == "" &&
				!anyProviderRegistrable(store.MergedAI(next)) {
				err = errors.New("clearing the API key would leave no AI providers configured")
			}
			badReq = err
			if err != nil {
				return settings.Settings{}, err
			}
			return next, nil
		}, applySettings)
		if badReq != nil {
			http.Error(w, badReq.Error(), http.StatusBadRequest)
			return
		}
		if err != nil {
			if errors.Is(err, settings.ErrDefaultAuthSecret) {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}
		writeJSON(w, http.StatusOK, buildAISettingsResponse(store.Effective()))
	}
}

func anyProviderRegistrable(cfg config.AIConfig) bool {
	return slices.ContainsFunc(airouter.ProviderNames(), func(name string) bool {
		return airouter.WouldRegister(name, cfg)
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
	if req.DefaultProvider != nil {
		name := strings.ToLower(strings.TrimSpace(*req.DefaultProvider))
		if name != "" && !slices.Contains(airouter.ProviderNames(), name) {
			return settings.Settings{}, fmt.Errorf("unknown provider %q", name)
		}
		st.AI.DefaultProvider = name
	}
	if req.OpenRouterModel != nil {
		st.AI.OpenRouterModel = strings.TrimSpace(*req.OpenRouterModel)
	}
	if req.OpenRouterAPIKey != nil {
		st.AI.OpenRouterAPIKey = strings.TrimSpace(*req.OpenRouterAPIKey)
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
func buildAISettingsResponse(eff settings.EffectiveSettings) aiSettingsResponse {
	return aiSettingsResponse{
		DefaultProvider: eff.DefaultProvider,
		OpenRouterModel: eff.OpenRouterModel,
		OpenRouterKey:   aiSettingsKeyStatus{Set: eff.OpenRouterKeySet, Last4: eff.OpenRouterKeyLast4},
		Flags:           eff.Flags,
		Sources: aiSettingsSources{
			DefaultProvider: eff.DefaultProviderSource,
			OpenRouterModel: eff.OpenRouterModelSource,
			OpenRouterKey:   eff.OpenRouterKeySource,
			Flags:           eff.FlagSources,
		},
		AvailableProviders: airouter.ProviderNames(),
	}
}
