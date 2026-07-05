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
	Current() settings.Settings
	Save(ctx context.Context, st settings.Settings) error
}

type aiSettingsKeyStatus struct {
	Set   bool   `json:"set"`
	Last4 string `json:"last4"`
}

type aiSettingsResponse struct {
	DefaultProvider    string              `json:"defaultProvider"`
	OpenRouterModel    string              `json:"openrouterModel"`
	OpenRouterKey      aiSettingsKeyStatus `json:"openrouterKey"`
	Flags              map[string]bool     `json:"flags"`
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
		writeJSON(w, http.StatusOK, buildAISettingsResponse(store.Current()))
	}
}

func handleAdminUpdateAISettings(store runtimeSettingsStore, applySettings func(settings.Settings)) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var body aiSettingsUpdateRequest
		if err := decodeJSONBody(r, &body); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		st, err := applyAISettingsUpdate(store.Current(), body)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		if err := store.Save(r.Context(), st); err != nil {
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}
		if applySettings != nil {
			applySettings(st)
		}
		writeJSON(w, http.StatusOK, buildAISettingsResponse(st))
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
		if _, err := settings.MergeFlags(featureflags.Features{}, req.Flags); err != nil {
			return settings.Settings{}, err
		}
		// Copy before writing: st.Flags aliases the store's live snapshot map.
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
func buildAISettingsResponse(st settings.Settings) aiSettingsResponse {
	flags := featureflags.Defaults()
	for name, enabled := range st.Flags {
		flags[name] = enabled
	}
	key := st.AI.OpenRouterAPIKey
	return aiSettingsResponse{
		DefaultProvider:    st.AI.DefaultProvider,
		OpenRouterModel:    st.AI.OpenRouterModel,
		OpenRouterKey:      aiSettingsKeyStatus{Set: key != "", Last4: settings.KeyLast4(key)},
		Flags:              flags,
		AvailableProviders: airouter.ProviderNames(),
	}
}
