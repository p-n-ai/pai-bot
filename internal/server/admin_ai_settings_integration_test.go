// Copyright 2026 the P&AI authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

//go:build integration

package server

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/p-n-ai/pai-bot/internal/ai"
	"github.com/p-n-ai/pai-bot/internal/auth"
	"github.com/p-n-ai/pai-bot/internal/platform/airouter"
	"github.com/p-n-ai/pai-bot/internal/platform/config"
	"github.com/p-n-ai/pai-bot/internal/platform/featureflags"
	"github.com/p-n-ai/pai-bot/internal/platform/settings"
	"github.com/p-n-ai/pai-bot/internal/retrieval"
)

func TestAdminAISettingsPutAppliesToLiveRouterAndSurvivesRestart(t *testing.T) {
	ctx, pool := serverSettingsTestPool(t)

	var completions atomic.Int32
	fakeOllama := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/v1/chat/completions" {
			http.Error(w, "unexpected request", http.StatusNotFound)
			return
		}
		var request struct {
			Model    string       `json:"model"`
			Messages []ai.Message `json:"messages"`
		}
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			http.Error(w, "invalid request", http.StatusBadRequest)
			return
		}
		if request.Model != "local-test" || len(request.Messages) != 1 ||
			request.Messages[0].Role != "user" {
			http.Error(w, "unexpected completion payload", http.StatusBadRequest)
			return
		}
		completions.Add(1)
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"model": "local-test",
			"choices": []map[string]any{{
				"message": map[string]string{"role": "assistant", "content": "local answer"},
			}},
			"usage": map[string]int{"prompt_tokens": 3, "completion_tokens": 2},
		})
	}))
	t.Cleanup(fakeOllama.Close)

	envAI := config.AIConfig{
		DefaultProvider: "mock",
		Mock:            config.MockAIConfig{Response: "wrong provider"},
		OpenRouter: config.OpenRouterConfig{
			APIKey: "sk-or-env-fallback-2468",
			Model:  "env/openrouter-model",
		},
		Ollama: config.OllamaConfig{
			Enabled: true,
			URL:     fakeOllama.URL,
			Model:   "local-test",
		},
	}
	const (
		jwtSecret     = "integration-jwt-secret"
		encryptionKey = "integration-settings-encryption-key"
		openRouterKey = "sk-or-integration-secret-9876"
		// Fixed pre-PR #224 base64(nonce || AES-GCM ciphertext || tag)
		// encrypted with jwtSecret. The HTTP update must migrate it.
		legacyOpenRouterBlob = "ABEiM0RVZneImaq7QQF+vEphZow1zsXjvEnSz0txnpYvitq0NVWKsOVPCjK4gtdd9cm8bJUSqkiS"
		unknownSecretName    = "future_provider_token"
		unknownSecretValue   = "opaque-future-secret-value"
	)
	var logs bytes.Buffer
	previousLogger := slog.Default()
	slog.SetDefault(slog.New(slog.NewTextHandler(&logs, nil)))
	t.Cleanup(func() { slog.SetDefault(previousLogger) })
	if _, err := pool.Exec(ctx, `
		INSERT INTO runtime_settings (id, ai, flags, secrets)
		VALUES (
			1,
			'{}',
			'{}',
			jsonb_build_object(
				'openrouter_api_key', $1::text,
				$2::text, $3::text
			)
		)
	`, legacyOpenRouterBlob, unknownSecretName, unknownSecretValue); err != nil {
		t.Fatalf("seed legacy runtime credential: %v", err)
	}
	store := settings.New(pool, encryptionKey, jwtSecret, envAI, featureflags.Features{})
	if err := store.Start(ctx); err != nil {
		t.Fatalf("settings.Store.Start() error = %v", err)
	}
	router := airouter.Setup(envAI)
	store.MarkApplied(store.Current().Revision)
	if got := router.ProviderOrder(); len(got) == 0 || got[0] != "mock" {
		t.Fatalf("initial provider order = %v, want mock first", got)
	}
	prepareSettings := func(st settings.Settings) (settings.PreparedApply, error) {
		plan, err := airouter.PrepareWithCodexAuth(settings.MergeAI(envAI, st), nil)
		if err != nil {
			return nil, err
		}
		return func() { plan.Apply(router) }, nil
	}
	handler := newHandlerWithAdminProvider(
		fixedAdminDataSourceProvider{source: stubAdminAPI{}},
		nil,
		&chatGatewayStub{},
		retrieval.NewMemoryService(),
		&stubAuthService{},
		jwtSecret,
		time.Hour,
		"",
		store,
		prepareSettings,
		false,
	)
	topMux := NewTopMux(TopMuxOptions{
		APIHandler:      handler,
		AIHealthEnabled: func() bool { return true },
		AIHealthToken:   "integration-health-token",
		AIHealthCheck:   NewAIHealthCheck(router),
	})
	token, err := auth.NewTokenManager(jwtSecret, time.Hour).Issue(auth.TokenClaims{
		Subject:  "admin-1",
		TenantID: "tenant-1",
		Role:     auth.RoleAdmin,
	}, time.Now().UTC())
	if err != nil {
		t.Fatalf("Issue() error = %v", err)
	}

	body := `{"expectedRevision":0,"defaultProvider":{"type":"ollama"}}`
	for _, tt := range []struct {
		name  string
		token string
		want  int
	}{
		{name: "unauthenticated", want: http.StatusUnauthorized},
		{
			name: "wrong role",
			token: mustIssueAISettingsIntegrationToken(
				t,
				jwtSecret,
				auth.RoleStudent,
			),
			want: http.StatusForbidden,
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPut, "/api/admin/ai/settings", strings.NewReader(body))
			if tt.token != "" {
				req.Header.Set("Authorization", "Bearer "+tt.token)
			}
			rec := httptest.NewRecorder()
			topMux.ServeHTTP(rec, req)
			if rec.Code != tt.want {
				t.Fatalf("PUT status = %d, want %d", rec.Code, tt.want)
			}
			var persistedBlob string
			if err := pool.QueryRow(ctx,
				`SELECT secrets->>'openrouter_api_key' FROM runtime_settings WHERE id = 1`,
			).Scan(&persistedBlob); err != nil {
				t.Fatalf("read runtime settings credential: %v", err)
			}
			if persistedBlob != legacyOpenRouterBlob {
				t.Fatal("unauthorized request mutated the runtime settings credential")
			}
		})
	}

	req := httptest.NewRequest(http.MethodPut, "/api/admin/ai/settings", strings.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	topMux.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("PUT status = %d, want %d (body %q)", rec.Code, http.StatusOK, rec.Body.String())
	}
	if strings.Contains(rec.Body.String(), openRouterKey) || strings.Contains(rec.Body.String(), "integration-secret") {
		t.Fatalf("PUT response leaked API key: %q", rec.Body.String())
	}
	var response aiSettingsTestPayload
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode PUT response: %v", err)
	}
	openRouter := integrationAPIKeyProvider(t, response, settings.APIKeyProviderOpenRouter)
	if response.DefaultProvider.Effective == nil ||
		response.DefaultProvider.Effective.Type != settings.ProviderKindOllama ||
		!openRouter.Credential.Effective.Set ||
		openRouter.Credential.Effective.Last4 != "9876" {
		t.Fatalf("PUT response = %#v, want ollama and redacted key ending 9876", response)
	}
	if response.Revision == 0 || response.AppliedRevision != response.Revision || response.Drift {
		t.Fatalf("PUT revision state = desired %d applied %d drift %v, want synchronously applied revision",
			response.Revision, response.AppliedRevision, response.Drift)
	}
	if keyHealth := openRouter.Credential.Health; !keyHealth.Stored || !keyHealth.Readable ||
		keyHealth.Version != "v1" || keyHealth.Algorithm != "a256gcm" ||
		keyHealth.KeyID == "" || keyHealth.MigrationNeeded {
		t.Fatalf("credential health = %+v, want readable current envelope metadata", keyHealth)
	}

	for _, tt := range []struct {
		name string
		body string
		want int
	}{
		{
			name: "stale revision",
			body: `{"expectedRevision":0,"provider":{"type":"api_key","name":"openrouter","model":"stale/model"}}`,
			want: http.StatusConflict,
		},
		{
			name: "unknown provider",
			body: `{"expectedRevision":` + fmt.Sprint(response.Revision) + `,"defaultProvider":{"type":"api_key","name":"unknown-provider"}}`,
			want: http.StatusBadRequest,
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPut, "/api/admin/ai/settings", strings.NewReader(tt.body))
			req.Header.Set("Authorization", "Bearer "+token)
			rec := httptest.NewRecorder()
			topMux.ServeHTTP(rec, req)
			if rec.Code != tt.want {
				t.Fatalf("PUT status = %d, want %d (body %q)", rec.Code, tt.want, rec.Body.String())
			}
		})
	}

	var rawAI, rawSecrets string
	if err := pool.QueryRow(ctx,
		`SELECT ai::text, secrets::text FROM runtime_settings WHERE id = 1`,
	).Scan(&rawAI, &rawSecrets); err != nil {
		t.Fatalf("read runtime_settings row: %v", err)
	}
	if !strings.Contains(rawAI, `"default_provider": "ollama"`) {
		t.Fatalf("stored ai = %q, want ollama default provider", rawAI)
	}
	if strings.Contains(rawAI, openRouterKey) || strings.Contains(rawSecrets, openRouterKey) ||
		strings.Contains(rawSecrets, "integration-secret") {
		t.Fatalf("runtime_settings stored API key in plaintext: ai=%q secrets=%q", rawAI, rawSecrets)
	}
	if rawSecrets == "{}" {
		t.Fatal("runtime_settings secrets = {}, want encrypted OpenRouter key")
	}
	if strings.Contains(rawSecrets, legacyOpenRouterBlob) {
		t.Fatal("HTTP update did not migrate the legacy credential envelope")
	}
	var persistedSecrets map[string]string
	if err := json.Unmarshal([]byte(rawSecrets), &persistedSecrets); err != nil {
		t.Fatalf("decode persisted secrets: %v", err)
	}
	migratedOpenRouterBlob := persistedSecrets["provider/openrouter/api_key"]
	if !strings.HasPrefix(migratedOpenRouterBlob, "pai:v1:a256gcm:") {
		t.Fatalf("migrated OpenRouter credential has unexpected envelope")
	}
	var preservedUnknownSecret string
	if err := pool.QueryRow(
		ctx,
		`SELECT secrets->>$1 FROM runtime_settings WHERE id = 1`,
		unknownSecretName,
	).Scan(&preservedUnknownSecret); err != nil {
		t.Fatalf("read preserved unknown secret: %v", err)
	}
	if preservedUnknownSecret != unknownSecretValue {
		t.Fatalf("unknown secret = %q, want byte-for-byte preserved value", preservedUnknownSecret)
	}
	var persistedRevision int64
	if err := pool.QueryRow(ctx, `SELECT revision FROM runtime_settings WHERE id = 1`).Scan(&persistedRevision); err != nil {
		t.Fatalf("read runtime settings revision: %v", err)
	}
	if persistedRevision != response.Revision {
		t.Fatalf("persisted revision = %d, want rejected writes to preserve %d", persistedRevision, response.Revision)
	}

	if got := router.ProviderOrder(); len(got) == 0 || got[0] != "ollama" {
		t.Fatalf("live provider order = %v, want ollama first", got)
	}
	assertLocalCompletion(t, ctx, router)

	getReq := httptest.NewRequest(http.MethodGet, "/api/admin/ai/settings", nil)
	getReq.Header.Set("Authorization", "Bearer "+token)
	getRec := httptest.NewRecorder()
	topMux.ServeHTTP(getRec, getReq)
	if getRec.Code != http.StatusOK {
		t.Fatalf("GET status = %d, want %d", getRec.Code, http.StatusOK)
	}
	if strings.Contains(getRec.Body.String(), openRouterKey) ||
		strings.Contains(getRec.Body.String(), "integration-secret") ||
		strings.Contains(getRec.Body.String(), rawSecrets) ||
		strings.Contains(getRec.Body.String(), unknownSecretValue) {
		t.Fatalf("GET response leaked secret material: %q", getRec.Body.String())
	}
	assertAIHealthOK(t, topMux)

	restartedStore := settings.New(pool, encryptionKey, "", envAI, featureflags.Features{})
	if err := restartedStore.Start(ctx); err != nil {
		t.Fatalf("restarted settings.Store.Start() error = %v", err)
	}
	if got := restartedStore.Current().AI.Credentials[settings.APIKeyProviderOpenRouter].Value; got != openRouterKey {
		t.Fatalf("restarted store API key = %q, want decrypted original", got)
	}
	restartedRouter := airouter.Setup(settings.MergeAI(envAI, restartedStore.Current()))
	restartedStore.MarkApplied(restartedStore.Current().Revision)
	if got := restartedRouter.ProviderOrder(); len(got) == 0 || got[0] != "ollama" {
		t.Fatalf("restarted provider order = %v, want ollama first", got)
	}
	assertLocalCompletion(t, ctx, restartedRouter)

	restartedPrepare := func(st settings.Settings) (settings.PreparedApply, error) {
		plan, err := airouter.PrepareWithCodexAuth(settings.MergeAI(envAI, st), nil)
		if err != nil {
			return nil, err
		}
		return func() { plan.Apply(restartedRouter) }, nil
	}
	restartedHandler := newHandlerWithAdminProvider(
		fixedAdminDataSourceProvider{source: stubAdminAPI{}},
		nil,
		&chatGatewayStub{},
		retrieval.NewMemoryService(),
		&stubAuthService{},
		jwtSecret,
		time.Hour,
		"",
		restartedStore,
		restartedPrepare,
		false,
	)
	restartedTopMux := NewTopMux(TopMuxOptions{
		APIHandler:      restartedHandler,
		AIHealthEnabled: func() bool { return true },
		AIHealthToken:   "integration-health-token",
		AIHealthCheck:   NewAIHealthCheck(restartedRouter),
	})

	restartedGet := httptest.NewRequest(http.MethodGet, "/api/admin/ai/settings", nil)
	restartedGet.Header.Set("Authorization", "Bearer "+token)
	restartedGetRec := httptest.NewRecorder()
	restartedTopMux.ServeHTTP(restartedGetRec, restartedGet)
	if restartedGetRec.Code != http.StatusOK ||
		strings.Contains(restartedGetRec.Body.String(), openRouterKey) ||
		strings.Contains(restartedGetRec.Body.String(), legacyOpenRouterBlob) {
		t.Fatalf("restarted GET was not safe: status=%d body=%q", restartedGetRec.Code, restartedGetRec.Body.String())
	}

	assertAIHealthOK(t, restartedTopMux)

	resetReq := httptest.NewRequest(
		http.MethodPut,
		"/api/admin/ai/settings",
		strings.NewReader(
			`{"expectedRevision":`+fmt.Sprint(response.Revision)+`,"provider":{"type":"api_key","name":"openrouter","apiKey":null}}`,
		),
	)
	resetReq.Header.Set("Authorization", "Bearer "+token)
	resetRec := httptest.NewRecorder()
	restartedTopMux.ServeHTTP(resetRec, resetReq)
	if resetRec.Code != http.StatusOK {
		t.Fatalf("reset PUT status = %d, want %d (body %q)", resetRec.Code, http.StatusOK, resetRec.Body.String())
	}
	var resetResponse aiSettingsTestPayload
	if err := json.Unmarshal(resetRec.Body.Bytes(), &resetResponse); err != nil {
		t.Fatalf("decode reset response: %v", err)
	}
	resetOpenRouter := integrationAPIKeyProvider(t, resetResponse, settings.APIKeyProviderOpenRouter)
	if resetOpenRouter.Credential.Source != "env" ||
		!resetOpenRouter.Credential.Effective.Set ||
		resetOpenRouter.Credential.Effective.Last4 != "2468" ||
		resetOpenRouter.Credential.Override.Set {
		t.Fatalf("reset response = %#v, want environment credential fallback", resetResponse)
	}
	var storedOpenRouterKey, storedUnknownSecret *string
	if err := pool.QueryRow(ctx, `
		SELECT secrets->>'provider/openrouter/api_key', secrets->>$1
		FROM runtime_settings
		WHERE id = 1
	`, unknownSecretName).Scan(&storedOpenRouterKey, &storedUnknownSecret); err != nil {
		t.Fatalf("read secrets after reset: %v", err)
	}
	if storedOpenRouterKey != nil || storedUnknownSecret == nil || *storedUnknownSecret != unknownSecretValue {
		t.Fatalf("reset persisted openrouter=%v unknown=%v, want only unknown entry preserved",
			storedOpenRouterKey, storedUnknownSecret)
	}
	assertLocalCompletion(t, ctx, restartedRouter)

	if got := completions.Load(); got != 5 {
		t.Fatalf("local completion requests = %d, want three explicit completions and two health probes", got)
	}
	assertCredentialFailureResponsesAreSafe(
		t,
		ctx,
		pool,
		envAI,
		jwtSecret,
		token,
		encryptionKey,
		migratedOpenRouterBlob,
		unknownSecretName,
		unknownSecretValue,
	)
	for _, sensitive := range []string{
		openRouterKey,
		envAI.OpenRouter.APIKey,
		legacyOpenRouterBlob,
		unknownSecretValue,
		token,
		"Bearer " + token,
	} {
		if sensitive != "" && strings.Contains(logs.String(), sensitive) {
			t.Fatalf("logs leaked sensitive runtime settings material")
		}
	}
}

func integrationAPIKeyProvider(
	t *testing.T,
	payload aiSettingsTestPayload,
	name settings.APIKeyProvider,
) aiAPIKeyProviderProjection {
	t.Helper()
	for _, raw := range payload.Providers {
		var header struct {
			Type settings.ProviderKind `json:"type"`
			Name string                `json:"name"`
		}
		if err := json.Unmarshal(raw, &header); err != nil {
			t.Fatalf("decode provider discriminator: %v", err)
		}
		if header.Type != settings.ProviderKindAPIKey || header.Name != string(name) {
			continue
		}
		var provider aiAPIKeyProviderProjection
		if err := json.Unmarshal(raw, &provider); err != nil {
			t.Fatalf("decode API-key provider %q: %v", name, err)
		}
		return provider
	}
	t.Fatalf("API-key provider %q missing from response", name)
	return aiAPIKeyProviderProjection{}
}

func assertCredentialFailureResponsesAreSafe(
	t *testing.T,
	ctx context.Context,
	pool *pgxpool.Pool,
	envAI config.AIConfig,
	jwtSecret, token, encryptionKey, validBlob, unknownSecretName, unknownSecretValue string,
) {
	t.Helper()

	unknownKeyParts := strings.Split(validBlob, ":")
	if len(unknownKeyParts) != 5 {
		t.Fatalf("valid credential envelope has %d fields, want 5", len(unknownKeyParts))
	}
	unknownKeyParts[3] = "AAAAAAAAAAAAAAAAAAAAAA"
	unknownKeyBlob := strings.Join(unknownKeyParts, ":")

	tamperedBlob := validBlob
	last := len(tamperedBlob) - 1
	if tamperedBlob[last] == 'A' {
		tamperedBlob = tamperedBlob[:last] + "B"
	} else {
		tamperedBlob = tamperedBlob[:last] + "A"
	}

	for _, tt := range []struct {
		name      string
		activeKey string
		blob      string
	}{
		{name: "missing active key", blob: validBlob},
		{name: "unknown key id", activeKey: encryptionKey, blob: unknownKeyBlob},
		{name: "tampered ciphertext", activeKey: encryptionKey, blob: tamperedBlob},
	} {
		t.Run(tt.name, func(t *testing.T) {
			if _, err := pool.Exec(ctx, `
				UPDATE runtime_settings
				SET secrets = jsonb_build_object(
					'openrouter_api_key', $1::text,
					$2::text, $3::text
				)
				WHERE id = 1
			`, tt.blob, unknownSecretName, unknownSecretValue); err != nil {
				t.Fatalf("seed unreadable credential: %v", err)
			}

			failureStore := settings.New(
				pool,
				tt.activeKey,
				"",
				envAI,
				featureflags.Features{},
			)
			if err := failureStore.Start(ctx); err != nil {
				t.Fatalf("start with unreadable credential: %v", err)
			}
			failureStore.MarkApplied(failureStore.Current().Revision)
			failureHandler := newHandlerWithAdminProvider(
				fixedAdminDataSourceProvider{source: stubAdminAPI{}},
				nil,
				&chatGatewayStub{},
				retrieval.NewMemoryService(),
				&stubAuthService{},
				jwtSecret,
				time.Hour,
				"",
				failureStore,
				nil,
				false,
			)
			failureMux := NewTopMux(TopMuxOptions{
				APIHandler: failureHandler,
			})
			req := httptest.NewRequest(http.MethodGet, "/api/admin/ai/settings", nil)
			req.Header.Set("Authorization", "Bearer "+token)
			rec := httptest.NewRecorder()
			failureMux.ServeHTTP(rec, req)
			if rec.Code != http.StatusOK {
				t.Fatalf("GET unreadable credential status = %d, want %d", rec.Code, http.StatusOK)
			}
			var got aiSettingsTestPayload
			if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
				t.Fatalf("decode unreadable credential response: %v", err)
			}
			openRouter := integrationAPIKeyProvider(t, got, settings.APIKeyProviderOpenRouter)
			if !openRouter.Credential.Health.Stored || openRouter.Credential.Health.Readable {
				t.Fatalf("credential health = %+v, want stored and unreadable", openRouter.Credential.Health)
			}
			for _, sensitive := range []string{tt.blob, unknownSecretValue, token} {
				if strings.Contains(rec.Body.String(), sensitive) {
					t.Fatal("unreadable credential response leaked sensitive material")
				}
			}
			var preserved string
			if err := pool.QueryRow(
				ctx,
				`SELECT secrets->>$1 FROM runtime_settings WHERE id = 1`,
				unknownSecretName,
			).Scan(&preserved); err != nil {
				t.Fatalf("read unknown secret after failure: %v", err)
			}
			if preserved != unknownSecretValue {
				t.Fatalf("unknown secret = %q, want preserved", preserved)
			}
		})
	}
}

func mustIssueAISettingsIntegrationToken(t *testing.T, secret string, role auth.Role) string {
	t.Helper()
	token, err := auth.NewTokenManager(secret, time.Hour).Issue(auth.TokenClaims{
		Subject:  "integration-user",
		TenantID: "tenant-1",
		Role:     role,
	}, time.Now().UTC())
	if err != nil {
		t.Fatalf("Issue(%s) error = %v", role, err)
	}
	return token
}

func assertLocalCompletion(t *testing.T, ctx context.Context, router *ai.Router) {
	t.Helper()
	response, err := router.Complete(ctx, ai.CompletionRequest{
		Messages: []ai.Message{{Role: "user", Content: "hello"}},
	})
	if err != nil {
		t.Fatalf("Router.Complete() error = %v", err)
	}
	if response.Content != "local answer" || response.Model != "local-test" ||
		response.InputTokens != 3 || response.OutputTokens != 2 {
		t.Fatalf("Router.Complete() = %#v, want local fake response", response)
	}
}

func assertAIHealthOK(t *testing.T, handler http.Handler) {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, "/health/ai", nil)
	req.Header.Set("Authorization", "Bearer integration-health-token")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK || rec.Body.String() != `{"status":"ok"}` {
		t.Fatalf("AI health = %d %q, want redacted ok", rec.Code, rec.Body.String())
	}
}

func serverSettingsTestPool(t *testing.T) (context.Context, *pgxpool.Pool) {
	t.Helper()

	databaseURL := strings.TrimSpace(os.Getenv("LEARN_TEST_DATABASE_URL"))
	if databaseURL == "" {
		t.Skip("LEARN_TEST_DATABASE_URL is not set")
	}
	ctx := context.Background()
	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		t.Fatalf("pgxpool.New() error = %v", err)
	}
	t.Cleanup(pool.Close)

	var databaseName string
	if err := pool.QueryRow(ctx, `SELECT current_database()`).Scan(&databaseName); err != nil {
		t.Fatalf("identify test database: %v", err)
	}
	normalizedName := strings.ToLower(strings.TrimSpace(databaseName))
	if normalizedName == "postgres" || !strings.Contains(normalizedName, "test") {
		t.Skipf("refusing to modify database %q: its name does not identify it as a test database", databaseName)
	}

	if _, err := pool.Exec(ctx, `DROP TABLE IF EXISTS runtime_settings`); err != nil {
		t.Fatalf("drop test runtime_settings: %v", err)
	}
	for _, name := range []string{
		"20260705090000_runtime_settings.sql",
		"20260729100000_runtime_settings_revision.sql",
	} {
		migrationPath := filepath.Join("..", "..", "migrations", name)
		migrationBytes, err := os.ReadFile(migrationPath)
		if err != nil {
			t.Fatalf("read production migration %s: %v", migrationPath, err)
		}
		upSQL := string(migrationBytes)
		if marker := strings.Index(upSQL, "-- +goose Up"); marker >= 0 {
			upSQL = upSQL[marker+len("-- +goose Up"):]
		}
		if marker := strings.Index(upSQL, "-- +goose Down"); marker >= 0 {
			upSQL = upSQL[:marker]
		}
		if _, err := pool.Exec(ctx, upSQL); err != nil {
			t.Fatalf("apply production migration %s: %v", migrationPath, err)
		}
	}
	t.Cleanup(func() {
		if _, err := pool.Exec(context.Background(), `DROP TABLE IF EXISTS runtime_settings`); err != nil {
			t.Errorf("drop test runtime_settings during cleanup: %v", err)
		}
	})
	return ctx, pool
}
