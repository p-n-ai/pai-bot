// Copyright 2026 the P&AI authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/p-n-ai/pai-bot/internal/auth"
	"github.com/p-n-ai/pai-bot/internal/chat"
)

// mockEmbedStore implements chat.EmbedConfigStore for testing.
type mockEmbedStore struct {
	configs         map[string]chat.EmbedConfig
	origins         map[string][]string
	addOriginErr    error
	removeOriginErr error
	findErr         error
	findResult      string
}

func newMockEmbedStore() *mockEmbedStore {
	return &mockEmbedStore{
		configs: make(map[string]chat.EmbedConfig),
		origins: make(map[string][]string),
	}
}

func (m *mockEmbedStore) GetByTenantID(_ context.Context, tenantID string) (chat.EmbedConfig, error) {
	if cfg, ok := m.configs[tenantID]; ok {
		return cfg, nil
	}
	return chat.EmbedConfig{TenantID: tenantID}, nil
}

func (m *mockEmbedStore) Upsert(_ context.Context, cfg chat.EmbedConfig) (chat.EmbedConfig, error) {
	m.configs[cfg.TenantID] = cfg
	return cfg, nil
}

func (m *mockEmbedStore) AddOrigin(_ context.Context, tenantID, origin string) error {
	if m.addOriginErr != nil {
		return m.addOriginErr
	}
	m.origins[tenantID] = append(m.origins[tenantID], origin)
	return nil
}

func (m *mockEmbedStore) RemoveOrigin(_ context.Context, tenantID, origin string) error {
	if m.removeOriginErr != nil {
		return m.removeOriginErr
	}
	filtered := m.origins[tenantID][:0]
	for _, o := range m.origins[tenantID] {
		if o != origin {
			filtered = append(filtered, o)
		}
	}
	m.origins[tenantID] = filtered
	return nil
}

func (m *mockEmbedStore) IsOriginAllowed(_ context.Context, tenantID, origin string) (bool, error) {
	for _, o := range m.origins[tenantID] {
		if o == origin {
			return true, nil
		}
	}
	return false, nil
}

func (m *mockEmbedStore) GetByTenantSlug(_ context.Context, _ string) (chat.EmbedConfig, error) {
	return chat.EmbedConfig{}, nil
}

func (m *mockEmbedStore) FindTenantBySlugAndOrigin(_ context.Context, _, _ string) (string, error) {
	if m.findErr != nil {
		return "", m.findErr
	}
	return m.findResult, nil
}

// TestHandleEmbedGuestAuth tests various cases for the guest auth endpoint.
func TestHandleEmbedGuestAuth(t *testing.T) {
	tests := []struct {
		name       string
		body       string
		origin     string
		storeErr   error
		wantStatus int
	}{
		{
			name:       "missing tenant in body",
			body:       `{}`,
			origin:     "https://example.com",
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "empty tenant field",
			body:       `{"tenant": "  "}`,
			origin:     "https://example.com",
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "missing origin header",
			body:       `{"tenant": "school-a"}`,
			origin:     "",
			wantStatus: http.StatusForbidden,
		},
		{
			name:       "embed not configured for tenant/origin",
			body:       `{"tenant": "school-a"}`,
			origin:     "https://example.com",
			storeErr:   chat.ErrEmbedNotConfigured,
			wantStatus: http.StatusForbidden,
		},
		{
			name:       "invalid json body",
			body:       `not-json`,
			origin:     "https://example.com",
			wantStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := newMockEmbedStore()
			store.findErr = tt.storeErr

			// handleEmbedGuestAuth needs a GuestService; pass nil — these cases
			// all return before reaching IssueGuestToken.
			handler := handleEmbedGuestAuth(store, nil)

			req := httptest.NewRequest(http.MethodPost, "/api/embed/auth/guest",
				strings.NewReader(tt.body))
			if tt.origin != "" {
				req.Header.Set("Origin", tt.origin)
			}
			rec := httptest.NewRecorder()
			handler.ServeHTTP(rec, req)

			if rec.Code != tt.wantStatus {
				t.Errorf("status = %d, want %d (body: %q)", rec.Code, tt.wantStatus, rec.Body.String())
			}
		})
	}
}

// TestHandleAdminGetEmbedConfig_NoAuth verifies 401 is returned without auth context.
func TestHandleAdminGetEmbedConfig_NoAuth(t *testing.T) {
	store := newMockEmbedStore()
	handler := handleAdminGetEmbedConfig(store)

	req := httptest.NewRequest(http.MethodGet, "/api/admin/embed/config", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusUnauthorized)
	}
}

// TestHandleAdminGetEmbedConfig_WithAuth verifies the config is returned when authenticated.
func TestHandleAdminGetEmbedConfig_WithAuth(t *testing.T) {
	store := newMockEmbedStore()
	store.configs["tenant-1"] = chat.EmbedConfig{
		TenantID: "tenant-1",
		Enabled:  true,
	}

	handler := handleAdminGetEmbedConfig(store)

	req := httptest.NewRequest(http.MethodGet, "/api/admin/embed/config", nil)
	ctx := auth.WithClaims(req.Context(), auth.TokenClaims{
		Subject:  "user-1",
		TenantID: "tenant-1",
		Role:     auth.RoleAdmin,
	})
	req = req.WithContext(ctx)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	var cfg chat.EmbedConfig
	if err := json.NewDecoder(rec.Body).Decode(&cfg); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if cfg.TenantID != "tenant-1" {
		t.Errorf("tenant_id = %q, want tenant-1", cfg.TenantID)
	}
}

// TestHandleAdminAddEmbedOrigin covers invalid format and valid origin cases.
func TestHandleAdminAddEmbedOrigin(t *testing.T) {
	tests := []struct {
		name       string
		body       string
		withAuth   bool
		wantStatus int
	}{
		{
			name:       "no auth",
			body:       `{"origin": "https://example.com"}`,
			withAuth:   false,
			wantStatus: http.StatusUnauthorized,
		},
		{
			name:       "invalid origin format — no scheme",
			body:       `{"origin": "not-a-url"}`,
			withAuth:   true,
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "invalid origin format — ftp scheme",
			body:       `{"origin": "ftp://example.com"}`,
			withAuth:   true,
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "missing origin field",
			body:       `{}`,
			withAuth:   true,
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "valid https origin",
			body:       `{"origin": "https://example.com"}`,
			withAuth:   true,
			wantStatus: http.StatusCreated,
		},
		{
			name:       "valid http origin",
			body:       `{"origin": "http://localhost:3000"}`,
			withAuth:   true,
			wantStatus: http.StatusCreated,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := newMockEmbedStore()
			handler := handleAdminAddEmbedOrigin(store)

			req := httptest.NewRequest(http.MethodPost, "/api/admin/embed/origins",
				strings.NewReader(tt.body))
			if tt.withAuth {
				ctx := auth.WithClaims(req.Context(), auth.TokenClaims{
					Subject:  "user-1",
					TenantID: "tenant-1",
					Role:     auth.RoleAdmin,
				})
				req = req.WithContext(ctx)
			}
			rec := httptest.NewRecorder()
			handler.ServeHTTP(rec, req)

			if rec.Code != tt.wantStatus {
				t.Errorf("status = %d, want %d (body: %q)", rec.Code, tt.wantStatus, rec.Body.String())
			}
		})
	}
}

// TestHandleAdminDeleteEmbedOrigin covers missing origin and successful deletion.
func TestHandleAdminDeleteEmbedOrigin(t *testing.T) {
	tests := []struct {
		name       string
		body       string
		withAuth   bool
		wantStatus int
	}{
		{
			name:       "no auth",
			body:       `{"origin": "https://example.com"}`,
			withAuth:   false,
			wantStatus: http.StatusUnauthorized,
		},
		{
			name:       "missing origin field",
			body:       `{}`,
			withAuth:   true,
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "empty origin",
			body:       `{"origin": "   "}`,
			withAuth:   true,
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "valid delete",
			body:       `{"origin": "https://example.com"}`,
			withAuth:   true,
			wantStatus: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := newMockEmbedStore()
			handler := handleAdminDeleteEmbedOrigin(store)

			req := httptest.NewRequest(http.MethodDelete, "/api/admin/embed/origins",
				strings.NewReader(tt.body))
			if tt.withAuth {
				ctx := auth.WithClaims(req.Context(), auth.TokenClaims{
					Subject:  "user-1",
					TenantID: "tenant-1",
					Role:     auth.RoleAdmin,
				})
				req = req.WithContext(ctx)
			}
			rec := httptest.NewRecorder()
			handler.ServeHTTP(rec, req)

			if rec.Code != tt.wantStatus {
				t.Errorf("status = %d, want %d (body: %q)", rec.Code, tt.wantStatus, rec.Body.String())
			}
		})
	}
}

// TestHandleEmbedUpgradeGuest_MissingFields verifies 400 when required fields are absent.
func TestHandleEmbedUpgradeGuest_MissingFields(t *testing.T) {
	tm := auth.NewTokenManager("test-secret-32-bytes-long-enough!", 0)
	// Issue a valid guest token so validation gets past auth.
	guestToken, err := tm.Issue(auth.TokenClaims{
		Subject:  "user-guest-1",
		TenantID: "tenant-1",
		Role:     auth.RoleGuest,
	}, time.Now())
	if err != nil {
		t.Fatalf("issue guest token: %v", err)
	}

	tests := []struct {
		name       string
		body       string
		wantStatus int
	}{
		{
			name:       "missing name",
			body:       `{"email":"a@b.com","password":"supersecret"}`,
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "missing email",
			body:       `{"name":"Alice","password":"supersecret"}`,
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "invalid email — no at sign",
			body:       `{"name":"Alice","email":"notanemail","password":"supersecret"}`,
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "password too short",
			body:       `{"name":"Alice","email":"a@b.com","password":"short"}`,
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "invalid json",
			body:       `not-json`,
			wantStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// handleEmbedUpgradeGuest with nil guestSvc — all cases fail before UpgradeGuest is called.
			handler := handleEmbedUpgradeGuest(nil, tm)
			req := httptest.NewRequest(http.MethodPost, "/api/embed/auth/upgrade",
				strings.NewReader(tt.body))
			req.Header.Set("Authorization", "Bearer "+guestToken)
			rec := httptest.NewRecorder()
			handler.ServeHTTP(rec, req)

			if rec.Code != tt.wantStatus {
				t.Errorf("status = %d, want %d (body: %q)", rec.Code, tt.wantStatus, rec.Body.String())
			}
		})
	}
}

// TestHandleEmbedUpgradeGuest_NoAuth verifies 401 when no JWT is provided.
func TestHandleEmbedUpgradeGuest_NoAuth(t *testing.T) {
	tm := auth.NewTokenManager("test-secret-32-bytes-long-enough!", 0)
	handler := handleEmbedUpgradeGuest(nil, tm)

	req := httptest.NewRequest(http.MethodPost, "/api/embed/auth/upgrade",
		strings.NewReader(`{"name":"Alice","email":"a@b.com","password":"supersecret"}`))
	// No Authorization header.
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusUnauthorized)
	}
}

// TestHandleEmbedMessages_NoAuth verifies 401 when no JWT is provided.
func TestHandleEmbedMessages_NoAuth(t *testing.T) {
	handler := handleEmbedMessages(nil, auth.NewTokenManager("test-secret-32-bytes-long-enough!", 0))

	req := httptest.NewRequest(http.MethodGet, "/api/embed/messages", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusUnauthorized)
	}
}

// TestHandleEmbedMessages_InvalidToken verifies 401 when an invalid JWT is provided.
func TestHandleEmbedMessages_InvalidToken(t *testing.T) {
	handler := handleEmbedMessages(nil, auth.NewTokenManager("test-secret-32-bytes-long-enough!", 0))

	req := httptest.NewRequest(http.MethodGet, "/api/embed/messages", nil)
	req.Header.Set("Authorization", "Bearer invalid-token-value")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusUnauthorized)
	}
}

// TestHandleEmbedMessages_ValidToken verifies 200 with empty messages when pool is nil.
func TestHandleEmbedMessages_ValidToken(t *testing.T) {
	tm := auth.NewTokenManager("test-secret-32-bytes-long-enough!", 0)
	token, err := tm.Issue(auth.TokenClaims{
		Subject:  "user-1",
		TenantID: "tenant-1",
		Role:     auth.RoleGuest,
	}, time.Now())
	if err != nil {
		t.Fatalf("issue token: %v", err)
	}

	handler := handleEmbedMessages(nil, tm)

	req := httptest.NewRequest(http.MethodGet, "/api/embed/messages", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d (body: %q)", rec.Code, http.StatusOK, rec.Body.String())
	}

	var resp struct {
		Messages []any `json:"messages"`
		HasMore  bool  `json:"has_more"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if len(resp.Messages) != 0 {
		t.Errorf("messages length = %d, want 0", len(resp.Messages))
	}
	if resp.HasMore {
		t.Errorf("has_more = true, want false")
	}
}

// TestHandleAdminUpdateEmbedConfig_NoAuth verifies 401 without auth.
func TestHandleAdminUpdateEmbedConfig_NoAuth(t *testing.T) {
	store := newMockEmbedStore()
	handler := handleAdminUpdateEmbedConfig(store)

	body := `{"enabled": true}`
	req := httptest.NewRequest(http.MethodPut, "/api/admin/embed/config", strings.NewReader(body))
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusUnauthorized)
	}
}

// TestHandleAdminUpdateEmbedConfig_UpdatesEnabled verifies enabled flag is persisted.
func TestHandleAdminUpdateEmbedConfig_UpdatesEnabled(t *testing.T) {
	store := newMockEmbedStore()
	enabled := true
	handler := handleAdminUpdateEmbedConfig(store)

	body, _ := json.Marshal(map[string]any{"enabled": enabled})
	req := httptest.NewRequest(http.MethodPut, "/api/admin/embed/config", bytes.NewReader(body))
	ctx := auth.WithClaims(req.Context(), auth.TokenClaims{
		Subject:  "user-1",
		TenantID: "tenant-1",
		Role:     auth.RoleAdmin,
	})
	req = req.WithContext(ctx)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d (body: %q)", rec.Code, http.StatusOK, rec.Body.String())
	}

	var cfg chat.EmbedConfig
	if err := json.NewDecoder(rec.Body).Decode(&cfg); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if !cfg.Enabled {
		t.Errorf("enabled = %v, want true", cfg.Enabled)
	}
}

