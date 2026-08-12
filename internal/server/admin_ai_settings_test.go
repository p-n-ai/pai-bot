// Copyright 2026 the P&AI authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package server

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
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

func (m *memorySettingsStore) Update(
	_ context.Context,
	mutate func(settings.Settings) (settings.Settings, error),
	prepare settings.PrepareApply,
) (settings.Settings, error) {
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
	for provider, credential := range st.AI.Credentials {
		credential.Operation = settings.SecretPreserve
		st.AI.Credentials[provider] = credential
	}
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

func newMultiTenantAISettingsHandler(
	store runtimeSettingsStore,
	apply func(settings.Settings),
	multiTenant bool,
) http.Handler {
	var prepare settings.PrepareApply
	if apply != nil {
		prepare = func(st settings.Settings) (settings.PreparedApply, error) {
			return func() { apply(st) }, nil
		}
	}
	return newHandlerWithAdminProvider(
		fixedAdminDataSourceProvider{source: stubAdminAPI{}},
		nil,
		&chatGatewayStub{},
		retrieval.NewMemoryService(),
		&stubAuthService{},
		"change-me-in-production",
		time.Hour,
		"",
		store,
		prepare,
		multiTenant,
	)
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

func doAISettingsRequest(
	t *testing.T,
	handler http.Handler,
	method string,
	token string,
	body string,
) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(method, "/api/admin/ai/settings", strings.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	return rec
}

type aiSettingsTestPayload struct {
	DefaultProvider aiDefaultProviderProjection `json:"defaultProvider"`
	Providers       []json.RawMessage           `json:"providers"`
	Flags           aiFlagsProjection           `json:"flags"`
	Revision        int64                       `json:"revision"`
	AppliedRevision int64                       `json:"appliedRevision"`
	Drift           bool                        `json:"drift"`
}

func decodeAISettingsPayload(t *testing.T, rec *httptest.ResponseRecorder) aiSettingsTestPayload {
	t.Helper()
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d (body %q)", rec.Code, http.StatusOK, rec.Body.String())
	}
	var payload aiSettingsTestPayload
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	return payload
}

func decodeProviderProjection(
	t *testing.T,
	payload aiSettingsTestPayload,
	kind settings.ProviderKind,
	name string,
) map[string]any {
	t.Helper()
	for _, raw := range payload.Providers {
		var header struct {
			Type settings.ProviderKind `json:"type"`
			Name string                `json:"name"`
		}
		if err := json.Unmarshal(raw, &header); err != nil {
			t.Fatal(err)
		}
		if header.Type == kind && header.Name == name {
			var result map[string]any
			if err := json.Unmarshal(raw, &result); err != nil {
				t.Fatal(err)
			}
			return result
		}
	}
	t.Fatalf("provider %s/%s not found in %s", kind, name, payload.Providers)
	return nil
}

func stringPointer(value string) *string { return &value }
func boolPointer(value bool) *bool       { return &value }

func apiKeyCredential(value string) map[settings.APIKeyProvider]settings.CredentialOverride {
	return map[settings.APIKeyProvider]settings.CredentialOverride{
		settings.APIKeyProviderOpenRouter: {Value: value},
	}
}

func TestAdminAISettingsGetReturnsClosedRedactedProviderProjections(t *testing.T) {
	const secret = "sk-or-super-secret-9876"
	env := config.AIConfig{
		DefaultProvider: "openrouter",
		OpenRouter: config.OpenRouterConfig{
			APIKey: secret,
			Model:  "env/model",
		},
		Ollama: config.OllamaConfig{URL: "http://127.0.0.1:11434"},
		Codex:  config.CodexConfig{Enabled: true, Model: "codex-env"},
		CatalogProviders: map[string]config.CatalogProviderConfig{
			"cerebras": {APIKey: "cerebras-env-key"},
		},
	}
	store := &memorySettingsStore{envAI: env}
	handler := newAISettingsHandler(store, nil)

	rec := doAISettingsRequest(t, handler, http.MethodGet, mustIssueAdminToken(t), "")
	payload := decodeAISettingsPayload(t, rec)

	if strings.Contains(rec.Body.String(), secret) ||
		strings.Contains(rec.Body.String(), "openrouterModel") ||
		strings.Contains(rec.Body.String(), "openrouterKey") {
		t.Fatalf("response leaked a secret or legacy flat field: %s", rec.Body.String())
	}
	if payload.DefaultProvider.Effective == nil ||
		payload.DefaultProvider.Effective.Type != settings.ProviderKindAPIKey ||
		payload.DefaultProvider.Effective.Name != "openrouter" {
		t.Fatalf("defaultProvider = %#v, want api_key/openrouter", payload.DefaultProvider)
	}
	if len(payload.Providers) != len(settings.APIKeyProviders())+2 {
		t.Fatalf("provider count = %d, want %d", len(payload.Providers), len(settings.APIKeyProviders())+2)
	}
	openRouter := decodeProviderProjection(t, payload, settings.ProviderKindAPIKey, "openrouter")
	credential := openRouter["credential"].(map[string]any)
	effective := credential["effective"].(map[string]any)
	if effective["set"] != true || effective["last4"] != "9876" || credential["source"] != settings.SourceEnv {
		t.Fatalf("credential projection = %#v, want redacted env key", credential)
	}
	cerebras := decodeProviderProjection(t, payload, settings.ProviderKindAPIKey, "cerebras")
	model := cerebras["model"].(map[string]any)
	if cerebras["displayName"] != "Cerebras" || model["baseline"] != "gpt-oss-120b" ||
		model["effective"] != "gpt-oss-120b" || model["source"] != settings.SourceDefault {
		t.Fatalf("Cerebras projection = %#v, want catalog default metadata", cerebras)
	}
	decodeProviderProjection(t, payload, settings.ProviderKindOllama, "")
	decodeProviderProjection(t, payload, settings.ProviderKindManagedCodex, "")
}

func TestAdminAISettingsGetReportsUnreadableStoredCredential(t *testing.T) {
	store := &memorySettingsStore{
		envAI: config.AIConfig{
			DefaultProvider: "openrouter",
			OpenRouter:      config.OpenRouterConfig{APIKey: "sk-or-env-9876"},
		},
		current: settings.Settings{AI: settings.AISettings{
			Credentials: map[settings.APIKeyProvider]settings.CredentialOverride{
				settings.APIKeyProviderOpenRouter: {
					Envelope: settings.CredentialEnvelopeStatus{Stored: true},
				},
			},
		}},
	}
	handler := newAISettingsHandler(store, nil)

	payload := decodeAISettingsPayload(t, doAISettingsRequest(
		t, handler, http.MethodGet, mustIssueAdminToken(t), "",
	))
	openRouter := decodeProviderProjection(t, payload, settings.ProviderKindAPIKey, "openrouter")
	credential := openRouter["credential"].(map[string]any)
	effective := credential["effective"].(map[string]any)
	health := credential["health"].(map[string]any)
	readiness := openRouter["readiness"].(map[string]any)
	if effective["set"] != false || credential["source"] != settings.SourceDB ||
		health["stored"] != true || health["readable"] != false ||
		readiness["registrable"] != false {
		t.Fatalf("unreadable credential projection = %#v, readiness = %#v", credential, readiness)
	}
}

func TestAdminAISettingsPutAppliesEachClosedVariant(t *testing.T) {
	tests := []struct {
		name  string
		env   config.AIConfig
		body  string
		check func(*testing.T, settings.Settings)
	}{
		{
			name: "api_key",
			env:  config.AIConfig{OpenAI: config.OpenAIConfig{APIKey: "env-openai"}},
			body: `{
				"expectedRevision":0,
				"defaultProvider":{"type":"api_key","name":"openrouter"},
				"provider":{"type":"api_key","name":"openrouter","model":"db/model","apiKey":"sk-or-db-1234"}
			}`,
			check: func(t *testing.T, st settings.Settings) {
				if st.AI.DefaultProvider == nil || *st.AI.DefaultProvider != "openrouter" {
					t.Fatalf("default provider = %#v", st.AI.DefaultProvider)
				}
				override := st.AI.Providers.APIKey[settings.APIKeyProviderOpenRouter]
				if override.Model == nil || *override.Model != "db/model" {
					t.Fatalf("OpenRouter override = %#v", override)
				}
				if st.AI.Credentials[settings.APIKeyProviderOpenRouter].Value != "sk-or-db-1234" {
					t.Fatal("OpenRouter credential was not stored in private state")
				}
			},
		},
		{
			name: "ollama",
			env:  config.AIConfig{OpenAI: config.OpenAIConfig{APIKey: "env-openai"}},
			body: `{
				"expectedRevision":0,
				"defaultProvider":{"type":"ollama"},
				"provider":{"type":"ollama","enabled":true,"model":"qwen:test"}
			}`,
			check: func(t *testing.T, st settings.Settings) {
				if st.AI.DefaultProvider == nil || *st.AI.DefaultProvider != "ollama" ||
					st.AI.Providers.Ollama == nil ||
					st.AI.Providers.Ollama.Enabled == nil || !*st.AI.Providers.Ollama.Enabled ||
					st.AI.Providers.Ollama.Model == nil || *st.AI.Providers.Ollama.Model != "qwen:test" {
					t.Fatalf("stored settings = %#v", st.AI)
				}
			},
		},
		{
			name: "managed_codex",
			env: config.AIConfig{
				OpenAI: config.OpenAIConfig{APIKey: "env-openai"},
				Codex:  config.CodexConfig{Enabled: true},
			},
			body: `{"expectedRevision":0,"provider":{"type":"managed_codex","model":"gpt-5-codex"}}`,
			check: func(t *testing.T, st settings.Settings) {
				if st.AI.Providers.ManagedCodex == nil ||
					st.AI.Providers.ManagedCodex.Model == nil ||
					*st.AI.Providers.ManagedCodex.Model != "gpt-5-codex" {
					t.Fatalf("managed Codex override = %#v", st.AI.Providers.ManagedCodex)
				}
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			store := &memorySettingsStore{envAI: test.env}
			var applied []settings.Settings
			handler := newAISettingsHandler(store, func(st settings.Settings) {
				applied = append(applied, st)
			})

			rec := doAISettingsRequest(t, handler, http.MethodPut, mustIssueAdminToken(t), test.body)
			decodeAISettingsPayload(t, rec)

			if store.saves != 1 || len(applied) != 1 {
				t.Fatalf("saves/applies = %d/%d, want 1/1", store.saves, len(applied))
			}
			test.check(t, store.current)
		})
	}
}

func TestAdminAISettingsPutRejectsInvalidAndCrossVariantFields(t *testing.T) {
	tests := []struct {
		name string
		body string
	}{
		{"legacy flat field", `{"openrouterApiKey":"secret"}`},
		{"unknown top level", `{"providers":[]}`},
		{"missing type", `{"provider":{"model":"x"}}`},
		{"unknown type", `{"provider":{"type":"bedrock","model":"x"}}`},
		{"provider null", `{"provider":null}`},
		{"empty api key patch", `{"provider":{"type":"api_key","name":"openrouter"}}`},
		{"unknown api provider", `{"provider":{"type":"api_key","name":"skynet","model":"x"}}`},
		{"api key with enabled", `{"provider":{"type":"api_key","name":"openrouter","enabled":true}}`},
		{"api key with url", `{"provider":{"type":"api_key","name":"openrouter","url":"http://localhost"}}`},
		{"ollama with api key", `{"provider":{"type":"ollama","apiKey":"secret"}}`},
		{"ollama with name", `{"provider":{"type":"ollama","name":"ollama","model":"x"}}`},
		{"ollama with url", `{"provider":{"type":"ollama","url":"http://localhost"}}`},
		{"managed codex with enabled", `{"provider":{"type":"managed_codex","enabled":true}}`},
		{"managed codex with token", `{"provider":{"type":"managed_codex","apiKey":"secret"}}`},
		{"selector cross field", `{"defaultProvider":{"type":"ollama","name":"ollama"}}`},
		{"empty model", `{"provider":{"type":"ollama","model":""}}`},
		{"whitespace model", `{"provider":{"type":"managed_codex","model":"   "}}`},
		{"empty key", `{"provider":{"type":"api_key","name":"openrouter","apiKey":""}}`},
		{"flags null", `{"flags":null}`},
		{"trailing JSON", `{} {}`},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			store := &memorySettingsStore{}
			handler := newAISettingsHandler(store, nil)
			body := `{"expectedRevision":0,` + strings.TrimPrefix(test.body, "{")
			if test.name == "trailing JSON" {
				body = `{"expectedRevision":0} {}`
			}
			rec := doAISettingsRequest(t, handler, http.MethodPut, mustIssueAdminToken(t), body)
			if rec.Code != http.StatusBadRequest {
				t.Fatalf("status = %d, want 400 (body %q)", rec.Code, rec.Body.String())
			}
			if store.saves != 0 {
				t.Fatalf("saves = %d, want 0", store.saves)
			}
		})
	}
}

func TestAdminAISettingsPutOmissionLeavesStateAndNullResetsOverrides(t *testing.T) {
	store := &memorySettingsStore{
		envAI: config.AIConfig{
			DefaultProvider: "openai",
			OpenAI:          config.OpenAIConfig{APIKey: "env-openai"},
			OpenRouter:      config.OpenRouterConfig{APIKey: "env-openrouter"},
		},
		current: settings.Settings{
			AI: settings.AISettings{
				DefaultProvider: stringPointer("openrouter"),
				Providers: settings.ProviderOverrides{
					APIKey: map[settings.APIKeyProvider]settings.APIKeyProviderOverride{
						settings.APIKeyProviderOpenRouter: {Model: stringPointer("old-model")},
					},
					Ollama:       &settings.EnabledModelOverride{Enabled: boolPointer(true), Model: stringPointer("old-ollama")},
					ManagedCodex: &settings.ModelOverride{Model: stringPointer("old-codex")},
				},
				Credentials: apiKeyCredential("sk-or-old-1234"),
			},
		},
	}
	handler := newAISettingsHandler(store, nil)

	decodeAISettingsPayload(t, doAISettingsRequest(
		t, handler, http.MethodPut, mustIssueAdminToken(t),
		`{"expectedRevision":0,"provider":{"type":"ollama","model":"new-ollama"}}`,
	))
	if store.current.AI.Providers.Ollama.Enabled == nil ||
		!*store.current.AI.Providers.Ollama.Enabled ||
		*store.current.AI.Providers.APIKey[settings.APIKeyProviderOpenRouter].Model != "old-model" ||
		store.current.AI.Credentials[settings.APIKeyProviderOpenRouter].Value != "sk-or-old-1234" {
		t.Fatalf("omitted fields changed: %#v", store.current.AI)
	}

	decodeAISettingsPayload(t, doAISettingsRequest(
		t, handler, http.MethodPut, mustIssueAdminToken(t),
		`{
			"expectedRevision":1,
			"defaultProvider":null,
			"provider":{"type":"api_key","name":"openrouter","model":null,"apiKey":null}
		}`,
	))
	if store.current.AI.DefaultProvider != nil {
		t.Fatalf("default override = %#v, want nil", store.current.AI.DefaultProvider)
	}
	if _, ok := store.current.AI.Providers.APIKey[settings.APIKeyProviderOpenRouter]; ok {
		t.Fatalf("OpenRouter model override survived reset: %#v", store.current.AI.Providers.APIKey)
	}
	credential := store.current.AI.Credentials[settings.APIKeyProviderOpenRouter]
	if credential.Value != "" {
		t.Fatal("credential reset retained private value")
	}
}

func TestAdminAISettingsPutNullKeyFallsBackToEnvironment(t *testing.T) {
	store := &memorySettingsStore{
		envAI: config.AIConfig{OpenRouter: config.OpenRouterConfig{APIKey: "sk-or-env-9876"}},
		current: settings.Settings{AI: settings.AISettings{
			Credentials: apiKeyCredential("sk-or-db-1234"),
		}},
	}
	handler := newAISettingsHandler(store, nil)
	payload := decodeAISettingsPayload(t, doAISettingsRequest(
		t, handler, http.MethodPut, mustIssueAdminToken(t),
		`{"expectedRevision":0,"provider":{"type":"api_key","name":"openrouter","apiKey":null}}`,
	))
	openRouter := decodeProviderProjection(t, payload, settings.ProviderKindAPIKey, "openrouter")
	credential := openRouter["credential"].(map[string]any)
	effective := credential["effective"].(map[string]any)
	if effective["set"] != true || effective["last4"] != "9876" ||
		credential["source"] != settings.SourceEnv {
		t.Fatalf("credential = %#v, want environment fallback", credential)
	}
}

func TestAdminAISettingsPutRejectsRemovingLastProvider(t *testing.T) {
	store := &memorySettingsStore{current: settings.Settings{AI: settings.AISettings{
		Credentials: apiKeyCredential("sk-or-db-1234"),
	}}}
	handler := newAISettingsHandler(store, nil)
	rec := doAISettingsRequest(
		t, handler, http.MethodPut, mustIssueAdminToken(t),
		`{"expectedRevision":0,"provider":{"type":"api_key","name":"openrouter","apiKey":null}}`,
	)
	if rec.Code != http.StatusBadRequest || !strings.Contains(rec.Body.String(), "no AI providers") {
		t.Fatalf("status/body = %d/%q, want no-provider 400", rec.Code, rec.Body.String())
	}
	if store.saves != 0 {
		t.Fatalf("saves = %d, want 0", store.saves)
	}
}

func TestAdminAISettingsPutRejectsRemovingCurrentDefaultWithFallback(t *testing.T) {
	store := &memorySettingsStore{
		envAI: config.AIConfig{
			DefaultProvider: "openai",
			OpenAI:          config.OpenAIConfig{APIKey: "env-openai"},
		},
		current: settings.Settings{
			AI: settings.AISettings{
				DefaultProvider: stringPointer("openrouter"),
				Credentials:     apiKeyCredential("sk-or-db-1234"),
			},
			Revision: 2,
		},
	}
	handler := newAISettingsHandler(store, nil)
	rec := doAISettingsRequest(
		t, handler, http.MethodPut, mustIssueAdminToken(t),
		`{"expectedRevision":2,"provider":{"type":"api_key","name":"openrouter","apiKey":null}}`,
	)
	if rec.Code != http.StatusBadRequest ||
		!strings.Contains(rec.Body.String(), `default provider "openrouter"`) {
		t.Fatalf("status/body = %d/%q, want unusable-default 400", rec.Code, rec.Body.String())
	}
	if store.saves != 0 {
		t.Fatalf("saves = %d, want 0", store.saves)
	}
}

func TestAdminAISettingsPutRejectsUnavailableDefaultSelector(t *testing.T) {
	for _, body := range []string{
		`{"expectedRevision":0,"defaultProvider":{"type":"api_key","name":"anthropic"}}`,
		`{"expectedRevision":0,"defaultProvider":{"type":"managed_codex"}}`,
	} {
		store := &memorySettingsStore{envAI: config.AIConfig{Codex: config.CodexConfig{Enabled: true}}}
		handler := newAISettingsHandler(store, nil)
		rec := doAISettingsRequest(t, handler, http.MethodPut, mustIssueAdminToken(t), body)
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("body %s: status = %d, want 400", body, rec.Code)
		}
	}
}

func TestAdminAISettingsPutRevisionAndFlagSemantics(t *testing.T) {
	envFlags, err := featureflags.Parse("turn_hooks=true")
	if err != nil {
		t.Fatal(err)
	}
	store := &memorySettingsStore{
		envAI:    config.AIConfig{OpenAI: config.OpenAIConfig{APIKey: "env-openai"}},
		envFlags: envFlags,
		current: settings.Settings{
			Flags:    map[string]bool{"turn_hooks": false},
			Revision: 4,
		},
	}
	handler := newAISettingsHandler(store, nil)

	missing := doAISettingsRequest(
		t, handler, http.MethodPut, mustIssueAdminToken(t),
		`{"flags":{"turn_hooks":null}}`,
	)
	if missing.Code != http.StatusBadRequest ||
		!strings.Contains(missing.Body.String(), "expectedRevision") ||
		store.saves != 0 {
		t.Fatalf("missing revision status/body/saves = %d/%q/%d, want 400/required/0",
			missing.Code, missing.Body.String(), store.saves)
	}

	stale := doAISettingsRequest(
		t, handler, http.MethodPut, mustIssueAdminToken(t),
		`{"expectedRevision":3,"flags":{"turn_hooks":null}}`,
	)
	if stale.Code != http.StatusConflict || store.saves != 0 {
		t.Fatalf("stale status/saves = %d/%d, want 409/0", stale.Code, store.saves)
	}

	payload := decodeAISettingsPayload(t, doAISettingsRequest(
		t, handler, http.MethodPut, mustIssueAdminToken(t),
		`{"expectedRevision":4,"flags":{"turn_hooks":null}}`,
	))
	if !payload.Flags.Effective["turn_hooks"] ||
		payload.Flags.Sources["turn_hooks"] != settings.SourceEnv {
		t.Fatalf("flags = %#v, want environment fallback", payload.Flags)
	}

	unknown := doAISettingsRequest(
		t, handler, http.MethodPut, mustIssueAdminToken(t),
		`{"expectedRevision":5,"flags":{"warp_drive":null}}`,
	)
	if unknown.Code != http.StatusBadRequest {
		t.Fatalf("unknown flag status = %d, want 400", unknown.Code)
	}
}

type failingSettingsStore struct {
	memorySettingsStore
	saveErr error
}

func (f *failingSettingsStore) Update(
	_ context.Context,
	mutate func(settings.Settings) (settings.Settings, error),
	_ settings.PrepareApply,
) (settings.Settings, error) {
	if _, err := mutate(f.current); err != nil {
		return settings.Settings{}, err
	}
	return settings.Settings{}, f.saveErr
}

func TestAdminAISettingsPutMapsCredentialPersistenceErrorsTo400(t *testing.T) {
	for _, test := range []struct {
		name    string
		err     error
		message string
	}{
		{
			name:    "missing encryption key",
			err:     settings.ErrConfigEncryptionKey,
			message: "PAI_CONFIG_ENCRYPTION_KEY",
		},
		{
			name:    "credential too large",
			err:     settings.ErrCredentialTooLarge,
			message: "4 KiB",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			store := &failingSettingsStore{
				memorySettingsStore: memorySettingsStore{
					envAI: config.AIConfig{OpenAI: config.OpenAIConfig{APIKey: "env-openai"}},
				},
				saveErr: test.err,
			}
			handler := newAISettingsHandler(store, nil)
			rec := doAISettingsRequest(
				t, handler, http.MethodPut, mustIssueAdminToken(t),
				`{"expectedRevision":0,"provider":{"type":"api_key","name":"openrouter","apiKey":"sk-or-new-key"}}`,
			)
			if rec.Code != http.StatusBadRequest ||
				!strings.Contains(rec.Body.String(), test.message) {
				t.Fatalf("status/body = %d/%q", rec.Code, rec.Body.String())
			}
		})
	}
}

func TestAdminAISettingsAuthorization(t *testing.T) {
	handler := newAISettingsHandler(&memorySettingsStore{}, nil)
	for _, method := range []string{http.MethodGet, http.MethodPut} {
		rec := doAISettingsRequest(t, handler, method, mustIssueTeacherToken(t), `{}`)
		if rec.Code != http.StatusForbidden {
			t.Fatalf("teacher %s status = %d, want 403", method, rec.Code)
		}
	}

	env := config.AIConfig{OpenRouter: config.OpenRouterConfig{APIKey: "env-key"}}
	handler = newAISettingsHandler(&memorySettingsStore{envAI: env}, nil)
	decodeAISettingsPayload(t, doAISettingsRequest(
		t,
		handler,
		http.MethodPut,
		mustIssuePlatformAdminToken(t),
		`{"expectedRevision":0,"defaultProvider":{"type":"api_key","name":"openrouter"}}`,
	))
}

func TestAdminAISettingsMultiTenantRejectsTenantAdmin(t *testing.T) {
	handler := newMultiTenantAISettingsHandler(&memorySettingsStore{}, nil, true)
	for _, method := range []string{http.MethodGet, http.MethodPut} {
		rec := doAISettingsRequest(t, handler, method, mustIssueAdminToken(t), `{}`)
		if rec.Code != http.StatusForbidden {
			t.Fatalf("tenant admin %s status = %d, want 403", method, rec.Code)
		}
	}
	decodeAISettingsPayload(t, doAISettingsRequest(
		t, handler, http.MethodGet, mustIssuePlatformAdminToken(t), "",
	))
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
		t.Fatal(err)
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

func TestParseProviderSelectorAcceptsEveryAPIKeyProvider(t *testing.T) {
	for _, provider := range settings.APIKeyProviders() {
		raw := json.RawMessage(`{"type":"api_key","name":"` + provider + `"}`)
		selector, err := parseProviderSelector(raw)
		if err != nil {
			t.Fatalf("%s: %v", provider, err)
		}
		if selector.Name != string(provider) || selector.Type != settings.ProviderKindAPIKey {
			t.Fatalf("%s: selector = %#v", provider, selector)
		}
	}
}

func TestAdminAISettingsStoreFailureDoesNotExposeCause(t *testing.T) {
	store := &failingSettingsStore{
		memorySettingsStore: memorySettingsStore{
			envAI: config.AIConfig{OpenAI: config.OpenAIConfig{APIKey: "env-openai"}},
		},
		saveErr: errors.New("upstream included a sensitive credential"),
	}
	handler := newAISettingsHandler(store, nil)
	rec := doAISettingsRequest(
		t, handler, http.MethodPut, mustIssueAdminToken(t),
		`{"expectedRevision":0,"provider":{"type":"ollama","model":"qwen"}}`,
	)
	if rec.Code != http.StatusInternalServerError ||
		strings.Contains(rec.Body.String(), "sensitive credential") {
		t.Fatalf("status/body = %d/%q, want redacted 500", rec.Code, rec.Body.String())
	}
}
