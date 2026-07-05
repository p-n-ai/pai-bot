// Copyright 2026 the P&AI authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package server

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/p-n-ai/pai-bot/internal/platform/settings"
	"github.com/p-n-ai/pai-bot/internal/retrieval"
)

type memorySettingsStore struct {
	current settings.Settings
	saves   int
}

func (m *memorySettingsStore) Current() settings.Settings { return m.current }

func (m *memorySettingsStore) Save(_ context.Context, st settings.Settings) error {
	m.current = st
	m.saves++
	return nil
}

func newAISettingsHandler(store runtimeSettingsStore, apply func(settings.Settings)) http.Handler {
	return newMultiTenantAISettingsHandler(store, apply, false)
}

func newMultiTenantAISettingsHandler(store runtimeSettingsStore, apply func(settings.Settings), multiTenant bool) http.Handler {
	return newHandlerWithAdminProvider(fixedAdminDataSourceProvider{source: stubAdminAPI{}}, nil, &chatGatewayStub{}, retrieval.NewMemoryService(), &stubAuthService{}, "change-me-in-production", time.Hour, "", store, apply, multiTenant)
}

func doAISettingsRequest(t *testing.T, handler http.Handler, method, token, body string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(method, "/api/admin/ai/settings", strings.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	return rec
}

type aiSettingsPayload struct {
	DefaultProvider string `json:"defaultProvider"`
	OpenRouterModel string `json:"openrouterModel"`
	OpenRouterKey   struct {
		Set   bool   `json:"set"`
		Last4 string `json:"last4"`
	} `json:"openrouterKey"`
	Flags              map[string]bool `json:"flags"`
	AvailableProviders []string        `json:"availableProviders"`
}

func decodeAISettingsPayload(t *testing.T, rec *httptest.ResponseRecorder) aiSettingsPayload {
	t.Helper()
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d (body %q)", rec.Code, http.StatusOK, rec.Body.String())
	}
	var payload aiSettingsPayload
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	return payload
}

func TestAdminAISettingsGetZeroState(t *testing.T) {
	handler := newAISettingsHandler(&memorySettingsStore{}, nil)
	payload := decodeAISettingsPayload(t, doAISettingsRequest(t, handler, http.MethodGet, mustIssueAdminToken(t), ""))

	if payload.DefaultProvider != "" || payload.OpenRouterModel != "" {
		t.Fatalf("provider/model = %q/%q, want empty", payload.DefaultProvider, payload.OpenRouterModel)
	}
	if payload.OpenRouterKey.Set || payload.OpenRouterKey.Last4 != "" {
		t.Fatalf("openrouterKey = %#v, want unset", payload.OpenRouterKey)
	}
	if enabled, ok := payload.Flags["turn_hooks"]; !ok || enabled {
		t.Fatalf("flags[turn_hooks] = %v, %v; want false, present", enabled, ok)
	}
	for _, name := range []string{"openai", "openrouter", "mock"} {
		if !slices.Contains(payload.AvailableProviders, name) {
			t.Fatalf("availableProviders = %v, want %q included", payload.AvailableProviders, name)
		}
	}
}

func TestAdminAISettingsPutSetsKeyAndNeverReturnsIt(t *testing.T) {
	const secretKey = "sk-or-verysecret-9876"
	store := &memorySettingsStore{}
	handler := newAISettingsHandler(store, nil)

	rec := doAISettingsRequest(t, handler, http.MethodPut, mustIssueAdminToken(t), `{"openrouterApiKey":"`+secretKey+`"}`)
	payload := decodeAISettingsPayload(t, rec)
	if !payload.OpenRouterKey.Set || payload.OpenRouterKey.Last4 != "9876" {
		t.Fatalf("openrouterKey = %#v, want set with last4 9876", payload.OpenRouterKey)
	}
	if strings.Contains(rec.Body.String(), "verysecret") {
		t.Fatalf("PUT response leaked the API key: %q", rec.Body.String())
	}
	if store.current.AI.OpenRouterAPIKey != secretKey {
		t.Fatalf("stored key = %q, want full key", store.current.AI.OpenRouterAPIKey)
	}

	rec = doAISettingsRequest(t, handler, http.MethodGet, mustIssueAdminToken(t), "")
	payload = decodeAISettingsPayload(t, rec)
	if !payload.OpenRouterKey.Set || payload.OpenRouterKey.Last4 != "9876" {
		t.Fatalf("GET openrouterKey = %#v, want set with last4 9876", payload.OpenRouterKey)
	}
	if strings.Contains(rec.Body.String(), "verysecret") {
		t.Fatalf("GET response leaked the API key: %q", rec.Body.String())
	}
}

func TestAdminAISettingsPutAbsentFieldsUnchanged(t *testing.T) {
	store := &memorySettingsStore{current: settings.Settings{
		AI:    settings.AISettings{DefaultProvider: "openrouter", OpenRouterModel: "old-model", OpenRouterAPIKey: "sk-1234"},
		Flags: map[string]bool{"turn_hooks": true},
	}}
	handler := newAISettingsHandler(store, nil)

	payload := decodeAISettingsPayload(t, doAISettingsRequest(t, handler, http.MethodPut, mustIssueAdminToken(t), `{"openrouterModel":"deepseek/deepseek-chat"}`))
	if payload.OpenRouterModel != "deepseek/deepseek-chat" {
		t.Fatalf("openrouterModel = %q, want deepseek/deepseek-chat", payload.OpenRouterModel)
	}
	if payload.DefaultProvider != "openrouter" || !payload.OpenRouterKey.Set || payload.OpenRouterKey.Last4 != "1234" {
		t.Fatalf("payload = %#v, want provider and key untouched", payload)
	}
	if store.current.AI.OpenRouterAPIKey != "sk-1234" || !store.current.Flags["turn_hooks"] {
		t.Fatalf("stored settings = %#v, want key and flags untouched", store.current)
	}
}

func TestAdminAISettingsPutEmptyKeyClears(t *testing.T) {
	store := &memorySettingsStore{current: settings.Settings{
		AI: settings.AISettings{OpenRouterAPIKey: "sk-1234"},
	}}
	handler := newAISettingsHandler(store, nil)

	payload := decodeAISettingsPayload(t, doAISettingsRequest(t, handler, http.MethodPut, mustIssueAdminToken(t), `{"openrouterApiKey":""}`))
	if payload.OpenRouterKey.Set || payload.OpenRouterKey.Last4 != "" {
		t.Fatalf("openrouterKey = %#v, want cleared", payload.OpenRouterKey)
	}
	if store.current.AI.OpenRouterAPIKey != "" {
		t.Fatalf("stored key = %q, want empty", store.current.AI.OpenRouterAPIKey)
	}
}

func TestAdminAISettingsPutRejectsUnknownProvider(t *testing.T) {
	store := &memorySettingsStore{}
	handler := newAISettingsHandler(store, nil)

	rec := doAISettingsRequest(t, handler, http.MethodPut, mustIssueAdminToken(t), `{"defaultProvider":"skynet"}`)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
	if store.saves != 0 {
		t.Fatalf("saves = %d, want 0", store.saves)
	}
}

func TestAdminAISettingsPutRejectsUnknownFlag(t *testing.T) {
	store := &memorySettingsStore{}
	handler := newAISettingsHandler(store, nil)

	rec := doAISettingsRequest(t, handler, http.MethodPut, mustIssueAdminToken(t), `{"flags":{"warp_drive":true}}`)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
	if store.saves != 0 {
		t.Fatalf("saves = %d, want 0", store.saves)
	}
}

func TestAdminAISettingsPutAppliesSettings(t *testing.T) {
	store := &memorySettingsStore{}
	var applied []settings.Settings
	handler := newAISettingsHandler(store, func(st settings.Settings) { applied = append(applied, st) })

	payload := decodeAISettingsPayload(t, doAISettingsRequest(t, handler, http.MethodPut, mustIssueAdminToken(t), `{"defaultProvider":"openrouter","flags":{"turn_hooks":true}}`))
	if payload.DefaultProvider != "openrouter" || !payload.Flags["turn_hooks"] {
		t.Fatalf("payload = %#v, want provider openrouter and turn_hooks on", payload)
	}
	if store.saves != 1 {
		t.Fatalf("saves = %d, want 1", store.saves)
	}
	if len(applied) != 1 || applied[0].AI.DefaultProvider != "openrouter" || !applied[0].Flags["turn_hooks"] {
		t.Fatalf("applied = %#v, want one call with saved settings", applied)
	}
}

func TestAdminAISettingsRejectsTeacherRole(t *testing.T) {
	handler := newAISettingsHandler(&memorySettingsStore{}, nil)

	for _, method := range []string{http.MethodGet, http.MethodPut} {
		rec := doAISettingsRequest(t, handler, method, mustIssueTeacherToken(t), `{}`)
		if rec.Code != http.StatusForbidden {
			t.Fatalf("%s status = %d, want %d", method, rec.Code, http.StatusForbidden)
		}
	}
}

func TestAdminAISettingsAllowsPlatformAdmin(t *testing.T) {
	handler := newAISettingsHandler(&memorySettingsStore{}, nil)

	decodeAISettingsPayload(t, doAISettingsRequest(t, handler, http.MethodGet, mustIssuePlatformAdminToken(t), ""))
	payload := decodeAISettingsPayload(t, doAISettingsRequest(t, handler, http.MethodPut, mustIssuePlatformAdminToken(t), `{"defaultProvider":"openrouter"}`))
	if payload.DefaultProvider != "openrouter" {
		t.Fatalf("defaultProvider = %q, want openrouter", payload.DefaultProvider)
	}
}

func TestAdminAISettingsMultiTenantRejectsTenantAdmin(t *testing.T) {
	handler := newMultiTenantAISettingsHandler(&memorySettingsStore{}, nil, true)

	for _, method := range []string{http.MethodGet, http.MethodPut} {
		rec := doAISettingsRequest(t, handler, method, mustIssueAdminToken(t), `{}`)
		if rec.Code != http.StatusForbidden {
			t.Fatalf("tenant admin %s status = %d, want %d", method, rec.Code, http.StatusForbidden)
		}
	}
	decodeAISettingsPayload(t, doAISettingsRequest(t, handler, http.MethodGet, mustIssuePlatformAdminToken(t), ""))
}
