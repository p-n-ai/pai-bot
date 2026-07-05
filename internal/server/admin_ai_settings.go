// Copyright 2026 the P&AI authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package server

import (
	"context"
	"fmt"
	"net/http"
	"slices"
	"strings"

	"github.com/p-n-ai/pai-bot/internal/platform/airouter"
	"github.com/p-n-ai/pai-bot/internal/platform/featureflags"
	"github.com/p-n-ai/pai-bot/internal/platform/settings"
)

// runtimeSettingsStore is the seam the AI settings handlers need; the
// concrete *settings.Store satisfies it.
type runtimeSettingsStore interface {
	Effective() settings.EffectiveSettings
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

type aiSettingsUpdateRequest struct {
	DefaultProvider  *string         `json:"defaultProvider"`
	OpenRouterModel  *string         `json:"openrouterModel"`
	OpenRouterAPIKey *string         `json:"openrouterApiKey"`
	Flags            map[string]bool `json:"flags"`
}

func handleAdminGetAISettings(store runtimeSettingsStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, buildAISettingsResponse(store.Effective()))
	}
}

func handleAdminUpdateAISettings(store runtimeSettingsStore, applySettings func(settings.Settings)) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var body aiSettingsUpdateRequest
		if err := decodeJSONBody(r, &body); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		var badReq error
		_, err := store.Update(r.Context(), func(cur settings.Settings) (settings.Settings, error) {
			next, err := applyAISettingsUpdate(cur, body)
			badReq = err
			return next, err
		}, applySettings)
		if badReq != nil {
			http.Error(w, badReq.Error(), http.StatusBadRequest)
			return
		}
		if err != nil {
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}
		writeJSON(w, http.StatusOK, buildAISettingsResponse(store.Effective()))
	}
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
		if _, err := (featureflags.Features{}).WithOverrides(req.Flags); err != nil {
			return settings.Settings{}, err
		}
		// Copy before writing: st.Flags may alias the caller's settings map.
		flags := make(map[string]bool, len(st.Flags)+len(req.Flags))
		for name, enabled := range st.Flags {
			flags[name] = enabled
		}
		for name, enabled := range req.Flags {
			flags[name] = enabled
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
		OpenRouterKey:   aiSettingsKeyStatus{Set: eff.OpenRouterAPIKey != "", Last4: settings.KeyLast4(eff.OpenRouterAPIKey)},
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
