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

	"github.com/p-n-ai/pai-bot/internal/platform/codexauth"
	"github.com/p-n-ai/pai-bot/internal/platform/config"
	"github.com/p-n-ai/pai-bot/internal/platform/featureflags"
	"github.com/p-n-ai/pai-bot/internal/platform/settings"
	"github.com/p-n-ai/pai-bot/internal/retrieval"
)

type memorySettingsStore struct {
	envAI    config.AIConfig
	envFlags featureflags.Features
	current  settings.Settings
	saves    int
}

type stubCodexDeviceAuth struct {
	status codexauth.Status
	starts int
}

func (m *memorySettingsStore) Current() settings.Settings { return m.current }

func (s *stubCodexDeviceAuth) Status() codexauth.Status { return s.status }
func (s *stubCodexDeviceAuth) Available() bool {
	return s.status.State == codexauth.StateConnected
}
func (s *stubCodexDeviceAuth) Start() (codexauth.Status, error) {
	s.starts++
	return s.status, nil
}

func (m *memorySettingsStore) Effective() settings.EffectiveSettings {
	effective := settings.Effective(m.envAI, m.envFlags, m.current)
	effective.AppliedRevision = m.current.Revision
	return effective
}

func (m *memorySettingsStore) EffectiveFor(st settings.Settings) settings.EffectiveSettings {
	effective := settings.Effective(m.envAI, m.envFlags, st)
	effective.AppliedRevision = st.Revision
	return effective
}

func (m *memorySettingsStore) MergedAI(st settings.Settings) config.AIConfig {
	return settings.MergeAI(m.envAI, st)
}

func (m *memorySettingsStore) Update(_ context.Context, mutate func(settings.Settings) (settings.Settings, error), prepare settings.PrepareApply) (settings.Settings, error) {
	st, err := mutate(m.current)
	if err != nil {
		return settings.Settings{}, err
	}
	var apply settings.PreparedApply
	if prepare != nil {
		apply, err = prepare(st)
		if err != nil {
			return settings.Settings{}, err
		}
	}
	st.Revision = m.current.Revision + 1
	st.AI.OpenRouterAPIKeyOperation = settings.SecretPreserve
	m.current = st
	m.saves++
	if apply != nil {
		apply()
	}
	return st, nil
}

func newAISettingsHandler(store runtimeSettingsStore, apply func(settings.Settings)) http.Handler {
	return newMultiTenantAISettingsHandler(store, apply, false)
}

func newMultiTenantAISettingsHandler(store runtimeSettingsStore, apply func(settings.Settings), multiTenant bool) http.Handler {
	var prepare settings.PrepareApply
	if apply != nil {
		prepare = func(st settings.Settings) (settings.PreparedApply, error) {
			return func() { apply(st) }, nil
		}
	}
	return newHandlerWithAdminProvider(fixedAdminDataSourceProvider{source: stubAdminAPI{}}, nil, &chatGatewayStub{}, retrieval.NewMemoryService(), &stubAuthService{}, "change-me-in-production", time.Hour, "", store, prepare, multiTenant)
}

func newCodexAuthHandler(store runtimeSettingsStore, deviceAuth codexDeviceAuth) http.Handler {
	return NewHandlerWithAdminProviderAndTeacherResourcesAndCodexAuth(
		fixedAdminDataSourceProvider{source: stubAdminAPI{}},
		nil,
		&chatGatewayStub{},
		retrieval.NewMemoryService(),
		nil,
		&stubAuthService{},
		"change-me-in-production",
		time.Hour,
		"",
		store,
		nil,
		false,
		deviceAuth,
	)
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
	Flags   map[string]bool `json:"flags"`
	Sources struct {
		DefaultProvider string            `json:"defaultProvider"`
		OpenRouterModel string            `json:"openrouterModel"`
		OpenRouterKey   string            `json:"openrouterKey"`
		Flags           map[string]string `json:"flags"`
	} `json:"sources"`
	Baseline struct {
		DefaultProvider string          `json:"defaultProvider"`
		OpenRouterModel string          `json:"openrouterModel"`
		Flags           map[string]bool `json:"flags"`
		OpenRouterKey   struct {
			Set   bool   `json:"set"`
			Last4 string `json:"last4"`
		} `json:"openrouterKey"`
	} `json:"baseline"`
	Override struct {
		DefaultProvider *string         `json:"defaultProvider"`
		OpenRouterModel *string         `json:"openrouterModel"`
		Flags           map[string]bool `json:"flags"`
		OpenRouterKey   struct {
			Set   bool   `json:"set"`
			Last4 string `json:"last4"`
		} `json:"openrouterKey"`
	} `json:"override"`
	Effective struct {
		DefaultProvider string          `json:"defaultProvider"`
		OpenRouterModel string          `json:"openrouterModel"`
		Flags           map[string]bool `json:"flags"`
		OpenRouterKey   struct {
			Set   bool   `json:"set"`
			Last4 string `json:"last4"`
		} `json:"openrouterKey"`
	} `json:"effective"`
	Revision           int64                 `json:"revision"`
	AppliedRevision    int64                 `json:"appliedRevision"`
	Drift              bool                  `json:"drift"`
	AvailableProviders []string              `json:"availableProviders"`
	Providers          []aiProviderReadiness `json:"providers"`
	Health             aiSettingsHealth      `json:"health"`
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
	if payload.Sources.DefaultProvider != "none" || payload.Sources.OpenRouterKey != "none" || payload.Sources.Flags["turn_hooks"] != "none" {
		t.Fatalf("sources = %#v, want all none", payload.Sources)
	}
}

func TestAdminAISettingsGetReportsEffectiveState(t *testing.T) {
	envFlags, err := featureflags.Parse("turn_hooks=true")
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	envAI := config.AIConfig{DefaultProvider: "openai"}
	envAI.OpenRouter.APIKey = "sk-or-env-5678"
	envAI.OpenRouter.Model = "env-model"
	store := &memorySettingsStore{envAI: envAI, envFlags: envFlags}
	handler := newAISettingsHandler(store, nil)

	payload := decodeAISettingsPayload(t, doAISettingsRequest(t, handler, http.MethodGet, mustIssueAdminToken(t), ""))
	if payload.DefaultProvider != "openai" || payload.Sources.DefaultProvider != "env" {
		t.Fatalf("defaultProvider = %q (%s), want openai (env)", payload.DefaultProvider, payload.Sources.DefaultProvider)
	}
	if !payload.OpenRouterKey.Set || payload.OpenRouterKey.Last4 != "5678" || payload.Sources.OpenRouterKey != "env" {
		t.Fatalf("openrouterKey = %#v (%s), want env key set with last4 5678", payload.OpenRouterKey, payload.Sources.OpenRouterKey)
	}
	if !payload.Flags["turn_hooks"] || payload.Sources.Flags["turn_hooks"] != "env" {
		t.Fatalf("turn_hooks = %v (%s), want true (env)", payload.Flags["turn_hooks"], payload.Sources.Flags["turn_hooks"])
	}
	if payload.Baseline.DefaultProvider != "openai" || payload.Baseline.OpenRouterModel != "env-model" {
		t.Fatalf("baseline = %+v, want env provider/model", payload.Baseline)
	}
	if payload.Baseline.OpenRouterKey.Last4 != "" || !payload.Baseline.OpenRouterKey.Set {
		t.Fatalf("baseline.openrouterKey = %+v, want set without env key hint", payload.Baseline.OpenRouterKey)
	}
	if !payload.Baseline.Flags["turn_hooks"] || !payload.Effective.Flags["turn_hooks"] {
		t.Fatalf("flag projections = baseline:%v effective:%v, want true/true",
			payload.Baseline.Flags, payload.Effective.Flags)
	}
	if payload.Override.DefaultProvider != nil || payload.Override.OpenRouterModel != nil || payload.Override.OpenRouterKey.Set {
		t.Fatalf("override = %+v, want no DB overrides", payload.Override)
	}
	if len(payload.Override.Flags) != 0 {
		t.Fatalf("override.flags = %v, want no DB overrides", payload.Override.Flags)
	}
	if payload.Effective.DefaultProvider != payload.DefaultProvider || payload.Effective.OpenRouterModel != payload.OpenRouterModel {
		t.Fatalf("effective = %+v, want top-level compatibility aliases", payload.Effective)
	}
	if payload.Revision != 0 || payload.AppliedRevision != 0 || payload.Drift {
		t.Fatalf("revision state = %d/%d drift=%v, want synced zero state", payload.Revision, payload.AppliedRevision, payload.Drift)
	}

	store.current = settings.Settings{
		AI:    settings.AISettings{DefaultProvider: "openrouter"},
		Flags: map[string]bool{"turn_hooks": false},
	}
	payload = decodeAISettingsPayload(t, doAISettingsRequest(t, handler, http.MethodGet, mustIssueAdminToken(t), ""))
	if payload.DefaultProvider != "openrouter" || payload.Sources.DefaultProvider != "db" {
		t.Fatalf("defaultProvider = %q (%s), want openrouter (db)", payload.DefaultProvider, payload.Sources.DefaultProvider)
	}
	if payload.Flags["turn_hooks"] || payload.Sources.Flags["turn_hooks"] != "db" {
		t.Fatalf("turn_hooks = %v (%s), want false (db)", payload.Flags["turn_hooks"], payload.Sources.Flags["turn_hooks"])
	}
	if override, ok := payload.Override.Flags["turn_hooks"]; !ok || override {
		t.Fatalf("override.flags[turn_hooks] = %v, %v; want false, true",
			override, ok)
	}
	if payload.Effective.Flags["turn_hooks"] {
		t.Fatal("effective.flags[turn_hooks] = true, want false")
	}
	if payload.OpenRouterModel != "env-model" || payload.Sources.OpenRouterModel != "env" {
		t.Fatalf("openrouterModel = %q (%s), want env-model (env)", payload.OpenRouterModel, payload.Sources.OpenRouterModel)
	}
}

func TestAdminAISettingsGetReportsExplicitFalseEnvironmentFlag(t *testing.T) {
	envFlags, err := featureflags.Parse("turn_hooks=false")
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	handler := newAISettingsHandler(&memorySettingsStore{envFlags: envFlags}, nil)

	payload := decodeAISettingsPayload(t, doAISettingsRequest(
		t,
		handler,
		http.MethodGet,
		mustIssueAdminToken(t),
		"",
	))

	if payload.Flags["turn_hooks"] || payload.Sources.Flags["turn_hooks"] != "env" {
		t.Fatalf("turn_hooks = %v (%s), want false from env",
			payload.Flags["turn_hooks"], payload.Sources.Flags["turn_hooks"])
	}
	if payload.Baseline.Flags["turn_hooks"] || payload.Effective.Flags["turn_hooks"] {
		t.Fatalf("flag projections = baseline:%v effective:%v, want false/false",
			payload.Baseline.Flags, payload.Effective.Flags)
	}
	if len(payload.Override.Flags) != 0 {
		t.Fatalf("override.flags = %v, want no DB overrides", payload.Override.Flags)
	}
}

func TestAdminAISettingsPutNullScalarsRemoveOverrides(t *testing.T) {
	envAI := config.AIConfig{DefaultProvider: "openai"}
	envAI.OpenRouter.Model = "env-model"
	envAI.OpenRouter.APIKey = "sk-or-env-5678"
	store := &memorySettingsStore{
		envAI: envAI,
		current: settings.Settings{AI: settings.AISettings{
			DefaultProvider:  "openrouter",
			OpenRouterModel:  "db-model",
			OpenRouterAPIKey: "sk-or-db-1234",
		}},
	}
	handler := newAISettingsHandler(store, nil)

	payload := decodeAISettingsPayload(t, doAISettingsRequest(
		t,
		handler,
		http.MethodPut,
		mustIssueAdminToken(t),
		`{"defaultProvider":null,"openrouterModel":null,"openrouterApiKey":null}`,
	))

	if payload.DefaultProvider != "openai" || payload.OpenRouterModel != "env-model" {
		t.Fatalf("effective provider/model = %q/%q, want env values", payload.DefaultProvider, payload.OpenRouterModel)
	}
	if !payload.OpenRouterKey.Set || payload.OpenRouterKey.Last4 != "5678" {
		t.Fatalf("effective key = %+v, want redacted env fallback", payload.OpenRouterKey)
	}
	if payload.Override.DefaultProvider != nil || payload.Override.OpenRouterModel != nil || payload.Override.OpenRouterKey.Set {
		t.Fatalf("override = %+v, want all DB overrides deleted", payload.Override)
	}
	if store.current.AI != (settings.AISettings{}) {
		t.Fatalf("stored AI = %+v, want zero overrides", store.current.AI)
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
	if payload.Revision != 1 || payload.AppliedRevision != 1 || payload.Drift {
		t.Fatalf("revision state = %d/%d drift=%v, want applied revision 1", payload.Revision, payload.AppliedRevision, payload.Drift)
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
		AI:    settings.AISettings{DefaultProvider: "openrouter", OpenRouterModel: "old-model", OpenRouterAPIKey: "sk-or-1234"},
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
	if store.current.AI.OpenRouterAPIKey != "sk-or-1234" || !store.current.Flags["turn_hooks"] {
		t.Fatalf("stored settings = %#v, want key and flags untouched", store.current)
	}
}

func TestAdminAISettingsPutRejectsStaleRevision(t *testing.T) {
	store := &memorySettingsStore{current: settings.Settings{
		AI:       settings.AISettings{DefaultProvider: "openai"},
		Revision: 4,
	}}
	handler := newAISettingsHandler(store, nil)

	rec := doAISettingsRequest(
		t,
		handler,
		http.MethodPut,
		mustIssueAdminToken(t),
		`{"expectedRevision":3,"openrouterModel":"new-model"}`,
	)
	if rec.Code != http.StatusConflict {
		t.Fatalf("status = %d, want %d (body %q)", rec.Code, http.StatusConflict, rec.Body.String())
	}
	if store.saves != 0 || store.current.AI.OpenRouterModel != "" || store.current.Revision != 4 {
		t.Fatalf("store = %+v saves=%d, want unchanged revision 4", store.current, store.saves)
	}
}

func TestAdminAISettingsPutEmptyKeyClears(t *testing.T) {
	store := &memorySettingsStore{
		envAI: config.AIConfig{OpenAI: config.OpenAIConfig{APIKey: "sk-env"}},
		current: settings.Settings{
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

func TestAdminAISettingsPutNullFlagRemovesOverride(t *testing.T) {
	envFlags, err := featureflags.Parse("turn_hooks=true")
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	store := &memorySettingsStore{envFlags: envFlags, current: settings.Settings{
		Flags: map[string]bool{"turn_hooks": false},
	}}
	handler := newAISettingsHandler(store, nil)

	payload := decodeAISettingsPayload(t, doAISettingsRequest(t, handler, http.MethodPut, mustIssueAdminToken(t), `{"flags":{"turn_hooks":null}}`))
	if !payload.Flags["turn_hooks"] || payload.Sources.Flags["turn_hooks"] != "env" {
		t.Fatalf("turn_hooks = %v (%s), want true (env) after override removal", payload.Flags["turn_hooks"], payload.Sources.Flags["turn_hooks"])
	}
	if _, ok := store.current.Flags["turn_hooks"]; ok {
		t.Fatalf("stored flags = %#v, want turn_hooks override deleted", store.current.Flags)
	}
}

func TestAdminAISettingsPutNullUnknownFlagRejected(t *testing.T) {
	store := &memorySettingsStore{}
	handler := newAISettingsHandler(store, nil)

	rec := doAISettingsRequest(t, handler, http.MethodPut, mustIssueAdminToken(t), `{"flags":{"warp_drive":null}}`)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
	if store.saves != 0 {
		t.Fatalf("saves = %d, want 0", store.saves)
	}
}

func TestAdminAISettingsPutRejectsUnconfiguredDefaultProvider(t *testing.T) {
	store := &memorySettingsStore{}
	handler := newAISettingsHandler(store, nil)

	rec := doAISettingsRequest(t, handler, http.MethodPut, mustIssueAdminToken(t), `{"defaultProvider":"anthropic"}`)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d (body %q)", rec.Code, http.StatusBadRequest, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "no usable configuration") {
		t.Fatalf("body = %q, want no-usable-configuration error", rec.Body.String())
	}
	if store.saves != 0 {
		t.Fatalf("saves = %d, want 0", store.saves)
	}
}

func TestAdminAISettingsPutRejectsManagedCodexWhenRuntimeUnavailable(t *testing.T) {
	store := &memorySettingsStore{envAI: config.AIConfig{
		Codex: config.CodexConfig{Enabled: true},
	}}
	handler := newAISettingsHandler(store, nil)

	rec := doAISettingsRequest(t, handler, http.MethodPut, mustIssueAdminToken(t), `{"defaultProvider":"codex"}`)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d (body %q)", rec.Code, http.StatusBadRequest, rec.Body.String())
	}
	if store.saves != 0 {
		t.Fatalf("saves = %d, want 0", store.saves)
	}
}

func TestAdminAISettingsPutClearKeyKeepsStaleDefault(t *testing.T) {
	store := &memorySettingsStore{
		envAI: config.AIConfig{OpenAI: config.OpenAIConfig{APIKey: "sk-env"}},
		current: settings.Settings{
			AI: settings.AISettings{DefaultProvider: "openrouter", OpenRouterAPIKey: "sk-or-1234"},
		}}
	handler := newAISettingsHandler(store, nil)

	// The request does not set defaultProvider, so the stale default must not
	// block clearing the key.
	payload := decodeAISettingsPayload(t, doAISettingsRequest(t, handler, http.MethodPut, mustIssueAdminToken(t), `{"openrouterApiKey":""}`))
	if payload.OpenRouterKey.Set {
		t.Fatalf("openrouterKey = %#v, want cleared", payload.OpenRouterKey)
	}
	if store.current.AI.DefaultProvider != "openrouter" || store.current.AI.OpenRouterAPIKey != "" {
		t.Fatalf("stored settings = %#v, want default kept and key cleared", store.current.AI)
	}
}

func TestAdminAISettingsPutRejectsClearingLastProvider(t *testing.T) {
	// No env providers: clearing the only key would empty the router and
	// crash-loop the next non-dev boot.
	store := &memorySettingsStore{current: settings.Settings{
		AI: settings.AISettings{OpenRouterAPIKey: "sk-or-1234"},
	}}
	handler := newAISettingsHandler(store, nil)

	rec := doAISettingsRequest(t, handler, http.MethodPut, mustIssueAdminToken(t), `{"openrouterApiKey":""}`)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d (body %q)", rec.Code, http.StatusBadRequest, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "no AI providers") {
		t.Fatalf("body = %q, want no-AI-providers error", rec.Body.String())
	}
	if store.saves != 0 || store.current.AI.OpenRouterAPIKey != "sk-or-1234" {
		t.Fatalf("saves = %d, stored key = %q; want no save and key kept", store.saves, store.current.AI.OpenRouterAPIKey)
	}
}

func TestAdminAISettingsPutRejectsUnknownField(t *testing.T) {
	store := &memorySettingsStore{}
	handler := newAISettingsHandler(store, nil)

	rec := doAISettingsRequest(t, handler, http.MethodPut, mustIssueAdminToken(t), `{"openrouterKey":"x"}`)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
	if store.saves != 0 {
		t.Fatalf("saves = %d, want 0", store.saves)
	}
}

// failingSettingsStore succeeds mutate but fails the save, like the Postgres
// store refusing to encrypt under the default auth secret.
type failingSettingsStore struct {
	memorySettingsStore
	saveErr error
}

func (f *failingSettingsStore) Update(_ context.Context, mutate func(settings.Settings) (settings.Settings, error), _ settings.PrepareApply) (settings.Settings, error) {
	if _, err := mutate(f.current); err != nil {
		return settings.Settings{}, err
	}
	return settings.Settings{}, f.saveErr
}

func TestAdminAISettingsPutMapsMissingConfigEncryptionKeyTo400(t *testing.T) {
	store := &failingSettingsStore{saveErr: settings.ErrConfigEncryptionKey}
	handler := newAISettingsHandler(store, nil)

	rec := doAISettingsRequest(t, handler, http.MethodPut, mustIssueAdminToken(t), `{"openrouterApiKey":"sk-or-new-key"}`)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d (body %q)", rec.Code, http.StatusBadRequest, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "PAI_CONFIG_ENCRYPTION_KEY") {
		t.Fatalf("body = %q, want PAI_CONFIG_ENCRYPTION_KEY message", rec.Body.String())
	}
}

func TestAdminAISettingsPutAppliesSettings(t *testing.T) {
	envAI := config.AIConfig{}
	envAI.OpenRouter.APIKey = "sk-or-env-1234"
	store := &memorySettingsStore{envAI: envAI}
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
	envAI := config.AIConfig{}
	envAI.OpenRouter.APIKey = "sk-or-env-1234"
	handler := newAISettingsHandler(&memorySettingsStore{envAI: envAI}, nil)

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

func TestAdminCodexDeviceAuthReturnsOnlySafeParsedState(t *testing.T) {
	deviceAuth := &stubCodexDeviceAuth{status: codexauth.Status{
		State:           codexauth.StateAwaiting,
		VerificationURL: "https://auth.openai.com/codex/device",
		UserCode:        "ABCD-1234",
	}}
	handler := newCodexAuthHandler(&memorySettingsStore{}, deviceAuth)

	req := httptest.NewRequest(http.MethodPost, "/api/admin/ai/codex/auth/device", nil)
	req.Header.Set("Authorization", "Bearer "+mustIssueAdminToken(t))
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK || deviceAuth.starts != 1 {
		t.Fatalf("status/starts = %d/%d, want 200/1", rec.Code, deviceAuth.starts)
	}
	var payload codexauth.Status
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	if payload != deviceAuth.status {
		t.Fatalf("payload = %#v, want %#v", payload, deviceAuth.status)
	}
}

func TestAdminCodexDeviceAuthRejectsTeacher(t *testing.T) {
	deviceAuth := &stubCodexDeviceAuth{}
	handler := newCodexAuthHandler(&memorySettingsStore{}, deviceAuth)
	req := httptest.NewRequest(http.MethodPost, "/api/admin/ai/codex/auth/device", nil)
	req.Header.Set("Authorization", "Bearer "+mustIssueTeacherToken(t))
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden || deviceAuth.starts != 0 {
		t.Fatalf("status/starts = %d/%d, want 403/0", rec.Code, deviceAuth.starts)
	}
}
