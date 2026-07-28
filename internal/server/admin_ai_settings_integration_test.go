// Copyright 2026 the P&AI authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

//go:build integration

package server

import (
	"context"
	"encoding/json"
	"fmt"
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
	)
	if _, err := pool.Exec(ctx, `
		INSERT INTO runtime_settings (id, ai, flags, secrets)
		VALUES (1, '{}', '{}', jsonb_build_object('openrouter_api_key', $1::text))
	`, legacyOpenRouterBlob); err != nil {
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
	token, err := auth.NewTokenManager(jwtSecret, time.Hour).Issue(auth.TokenClaims{
		Subject:  "admin-1",
		TenantID: "tenant-1",
		Role:     auth.RoleAdmin,
	}, time.Now().UTC())
	if err != nil {
		t.Fatalf("Issue() error = %v", err)
	}

	body := `{"defaultProvider":"ollama","openrouterModel":"unused-test-model"}`
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
			handler.ServeHTTP(rec, req)
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
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("PUT status = %d, want %d (body %q)", rec.Code, http.StatusOK, rec.Body.String())
	}
	if strings.Contains(rec.Body.String(), openRouterKey) || strings.Contains(rec.Body.String(), "integration-secret") {
		t.Fatalf("PUT response leaked API key: %q", rec.Body.String())
	}
	var response aiSettingsPayload
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode PUT response: %v", err)
	}
	if response.DefaultProvider != "ollama" || !response.OpenRouterKey.Set || response.OpenRouterKey.Last4 != "9876" {
		t.Fatalf("PUT response = %#v, want ollama and redacted key ending 9876", response)
	}
	if response.Revision == 0 || response.AppliedRevision != response.Revision || response.Drift {
		t.Fatalf("PUT revision state = desired %d applied %d drift %v, want synchronously applied revision",
			response.Revision, response.AppliedRevision, response.Drift)
	}
	if keyHealth := response.Health.OpenRouterKey; !keyHealth.Stored || !keyHealth.Readable ||
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
			body: `{"expectedRevision":0,"openrouterModel":"stale/model"}`,
			want: http.StatusConflict,
		},
		{
			name: "unknown provider",
			body: `{"expectedRevision":` + fmt.Sprint(response.Revision) + `,"defaultProvider":"unknown-provider"}`,
			want: http.StatusBadRequest,
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPut, "/api/admin/ai/settings", strings.NewReader(tt.body))
			req.Header.Set("Authorization", "Bearer "+token)
			rec := httptest.NewRecorder()
			handler.ServeHTTP(rec, req)
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
	var persistedRevision int64
	if err := pool.QueryRow(ctx, `SELECT revision FROM runtime_settings WHERE id = 1`).Scan(&persistedRevision); err != nil {
		t.Fatalf("read runtime settings revision: %v", err)
	}
	if persistedRevision != response.Revision {
		t.Fatalf("persisted revision = %d, want rejected writes to preserve %d", persistedRevision, response.Revision)
	}

	getReq := httptest.NewRequest(http.MethodGet, "/api/admin/ai/settings", nil)
	getReq.Header.Set("Authorization", "Bearer "+token)
	getRec := httptest.NewRecorder()
	handler.ServeHTTP(getRec, getReq)
	if getRec.Code != http.StatusOK {
		t.Fatalf("GET status = %d, want %d", getRec.Code, http.StatusOK)
	}
	if strings.Contains(getRec.Body.String(), openRouterKey) ||
		strings.Contains(getRec.Body.String(), "integration-secret") ||
		strings.Contains(getRec.Body.String(), rawSecrets) {
		t.Fatalf("GET response leaked secret material: %q", getRec.Body.String())
	}
	if got := router.ProviderOrder(); len(got) == 0 || got[0] != "ollama" {
		t.Fatalf("live provider order = %v, want ollama first", got)
	}
	assertLocalCompletion(t, ctx, router)

	restartedStore := settings.New(pool, encryptionKey, "", envAI, featureflags.Features{})
	if err := restartedStore.Start(ctx); err != nil {
		t.Fatalf("restarted settings.Store.Start() error = %v", err)
	}
	if got := restartedStore.Current().AI.OpenRouterAPIKey; got != openRouterKey {
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

	healthReq := httptest.NewRequest(http.MethodGet, "/health/ai", nil)
	healthReq.Header.Set("Authorization", "Bearer integration-health-token")
	healthRec := httptest.NewRecorder()
	restartedTopMux.ServeHTTP(healthRec, healthReq)
	if healthRec.Code != http.StatusOK || healthRec.Body.String() != `{"status":"ok"}` {
		t.Fatalf("restarted AI health = %d %q, want redacted ok", healthRec.Code, healthRec.Body.String())
	}
	if got := completions.Load(); got != 3 {
		t.Fatalf("local completion requests = %d, want 3", got)
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
