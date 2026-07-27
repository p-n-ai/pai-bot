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

	"github.com/p-n-ai/pai-bot/internal/auth"
	"github.com/p-n-ai/pai-bot/internal/chat"
)

type embedConfigStoreStub struct {
	configs        map[string]chat.EmbedConfig
	tenantByOrigin map[string]string
	lastTenantID   string
	lastOrigin     string
}

func (s *embedConfigStoreStub) GetByTenantID(_ context.Context, tenantID string) (chat.EmbedConfig, error) {
	s.lastTenantID = tenantID
	return s.configs[tenantID], nil
}

func (s *embedConfigStoreStub) GetByTenantSlug(_ context.Context, slug string) (chat.EmbedConfig, error) {
	for key, tenantID := range s.tenantByOrigin {
		if strings.HasPrefix(key, slug+"|") {
			return s.configs[tenantID], nil
		}
	}
	return chat.EmbedConfig{}, nil
}

func (s *embedConfigStoreStub) Upsert(_ context.Context, config chat.EmbedConfig) (chat.EmbedConfig, error) {
	s.lastTenantID = config.TenantID
	s.configs[config.TenantID] = config
	return config, nil
}

func (s *embedConfigStoreStub) AddOrigin(_ context.Context, tenantID, origin string) error {
	s.lastTenantID, s.lastOrigin = tenantID, origin
	config := s.configs[tenantID]
	config.TenantID = tenantID
	config.AllowedOrigins = append(config.AllowedOrigins, origin)
	s.configs[tenantID] = config
	return nil
}

func (s *embedConfigStoreStub) RemoveOrigin(_ context.Context, tenantID, origin string) error {
	s.lastTenantID, s.lastOrigin = tenantID, origin
	return nil
}

func (s *embedConfigStoreStub) IsOriginAllowed(_ context.Context, tenantID, origin string) (bool, error) {
	return s.tenantByOrigin[origin] == tenantID, nil
}

func (s *embedConfigStoreStub) FindTenantBySlugAndOrigin(_ context.Context, slug, origin string) (string, error) {
	tenantID := s.tenantByOrigin[slug+"|"+origin]
	if tenantID == "" {
		return "", chat.ErrEmbedNotConfigured
	}
	return tenantID, nil
}

type embedGuestsStub struct {
	parentOrigin string
}

func (s *embedGuestsStub) IssueGuestToken(_ context.Context, tenantID, origin, fingerprint string) (string, string, error) {
	s.parentOrigin = origin
	return "guest-token", "guest-user", nil
}

func (s *embedGuestsStub) UpgradeGuest(_ context.Context, userID, tenantID, parentOrigin, name, email, password string) (string, error) {
	s.parentOrigin = parentOrigin
	return "student-token", nil
}

type embedMessagesStub struct {
	tenantID string
	userID   string
}

type embedIdentityResolverStub struct {
	identity EmbedIdentity
}

type embedAuthenticatorStub struct {
	user auth.UserSession
}

func (s embedAuthenticatorStub) AuthenticatePassword(context.Context, auth.LoginRequest) (auth.UserSession, error) {
	return s.user, nil
}

func (s embedIdentityResolverStub) ResolveEmbedIdentity(context.Context, string, string) (EmbedIdentity, error) {
	return s.identity, nil
}

func (s *embedMessagesStub) ListEmbedMessages(_ context.Context, tenantID, userID, before string, limit int) ([]EmbedMessage, bool, error) {
	s.tenantID, s.userID = tenantID, userID
	return []EmbedMessage{{ID: "message-1", Role: "assistant", Content: "Hello"}}, false, nil
}

func TestAdminEmbedRoutesAuthorizationAndTenantIsolation(t *testing.T) {
	const secret = "embed-admin-test-secret"
	store := &embedConfigStoreStub{configs: map[string]chat.EmbedConfig{
		"tenant-a": {TenantID: "tenant-a", Enabled: true},
	}}
	handler := NewTopMux(TopMuxOptions{
		EmbedConfigStore: store,
		JWTSecret:        secret,
		AccessTokenTTL:   time.Hour,
	})

	for _, test := range []struct {
		name   string
		token  string
		status int
	}{
		{name: "unauthorized", status: http.StatusUnauthorized},
		{name: "student role rejected", token: issueEmbedTestToken(t, secret, auth.RoleStudent, "tenant-a", ""), status: http.StatusForbidden},
		{name: "admin allowed", token: issueEmbedTestToken(t, secret, auth.RoleAdmin, "tenant-a", ""), status: http.StatusOK},
		{name: "platform admin with tenant allowed", token: issueEmbedTestToken(t, secret, auth.RolePlatformAdmin, "tenant-a", ""), status: http.StatusOK},
		{name: "platform admin without tenant rejected", token: issueEmbedTestToken(t, secret, auth.RolePlatformAdmin, "", ""), status: http.StatusForbidden},
	} {
		t.Run(test.name, func(t *testing.T) {
			request := httptest.NewRequest(http.MethodGet, "/api/admin/embed/config", nil)
			if test.token != "" {
				request.Header.Set("Authorization", "Bearer "+test.token)
			}
			response := httptest.NewRecorder()
			handler.ServeHTTP(response, request)
			if response.Code != test.status {
				t.Fatalf("status = %d, want %d: %s", response.Code, test.status, response.Body.String())
			}
			if test.status == http.StatusOK {
				var payload map[string]any
				if err := json.Unmarshal(response.Body.Bytes(), &payload); err != nil {
					t.Fatal(err)
				}
				for _, key := range []string{"tenant_id", "enabled", "allowed_origins", "theme_config", "public_embed_base_url"} {
					if _, ok := payload[key]; !ok {
						t.Errorf("response missing snake_case field %q: %s", key, response.Body.String())
					}
				}
				if store.lastTenantID != "tenant-a" {
					t.Fatalf("store tenant = %q, want tenant-a", store.lastTenantID)
				}
			}
		})
	}
}

func TestAdminEmbedOriginNormalizationAndValidation(t *testing.T) {
	const secret = "embed-origin-test-secret"
	store := &embedConfigStoreStub{configs: map[string]chat.EmbedConfig{}}
	handler := NewTopMux(TopMuxOptions{EmbedConfigStore: store, JWTSecret: secret, AccessTokenTTL: time.Hour})
	token := issueEmbedTestToken(t, secret, auth.RoleAdmin, "tenant-a", "")

	request := httptest.NewRequest(http.MethodPost, "/api/admin/embed/origins", strings.NewReader(`{"origin":"HTTPS://School.Example/"}`))
	request.Header.Set("Authorization", "Bearer "+token)
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusCreated {
		t.Fatalf("status = %d, want 201: %s", response.Code, response.Body.String())
	}
	if store.lastTenantID != "tenant-a" || store.lastOrigin != "https://school.example" {
		t.Fatalf("stored tenant/origin = %q/%q", store.lastTenantID, store.lastOrigin)
	}

	for _, origin := range []string{"javascript:alert(1)", "https://school.example/path", "https://user@school.example", "school.example"} {
		request = httptest.NewRequest(http.MethodDelete, "/api/admin/embed/origins", strings.NewReader(`{"origin":`+strconvQuote(origin)+`}`))
		request.Header.Set("Authorization", "Bearer "+token)
		response = httptest.NewRecorder()
		handler.ServeHTTP(response, request)
		if response.Code != http.StatusBadRequest {
			t.Errorf("origin %q status = %d, want 400", origin, response.Code)
		}
	}
}

func TestAdminEmbedConfigUpdatePreservesTenantScope(t *testing.T) {
	const secret = "embed-update-test-secret"
	store := &embedConfigStoreStub{configs: map[string]chat.EmbedConfig{
		"tenant-a": {
			TenantID:       "tenant-a",
			AllowedOrigins: []string{"https://school.example"},
		},
	}}
	handler := NewTopMux(TopMuxOptions{EmbedConfigStore: store, JWTSecret: secret, AccessTokenTTL: time.Hour})
	request := httptest.NewRequest(http.MethodPut, "/api/admin/embed/config", strings.NewReader(
		`{"enabled":true,"theme_config":{"color":"#123456","language":"ms","position":"bottom-left"}}`,
	))
	request.Header.Set("Authorization", "Bearer "+issueEmbedTestToken(t, secret, auth.RoleAdmin, "tenant-a", ""))
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200: %s", response.Code, response.Body.String())
	}
	updated := store.configs["tenant-a"]
	if !updated.Enabled || updated.ThemeConfig["language"] != "ms" {
		t.Fatalf("updated config = %#v", updated)
	}
	if len(updated.AllowedOrigins) != 1 {
		t.Fatalf("allowed origins were not preserved: %#v", updated.AllowedOrigins)
	}
}

func TestEmbedGuestAuthBindsValidatedParentOrigin(t *testing.T) {
	store := &embedConfigStoreStub{
		configs: map[string]chat.EmbedConfig{},
		tenantByOrigin: map[string]string{
			"school|https://school.example": "tenant-a",
		},
	}
	guests := &embedGuestsStub{}
	handler := NewTopMux(TopMuxOptions{
		EmbedConfigStore:  store,
		EmbedGuestService: guests,
		JWTSecret:         "guest-secret",
		AccessTokenTTL:    15 * time.Minute,
		EmbedTokenTTL:     2 * time.Hour,
	})

	request := httptest.NewRequest(http.MethodPost, "https://api.example/api/embed/auth/guest", strings.NewReader(
		`{"tenant":"school","parent_origin":"https://school.example","fingerprint":"abc"}`,
	))
	request.Header.Set("Origin", "https://api.example")
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200: %s", response.Code, response.Body.String())
	}
	if guests.parentOrigin != "https://school.example" {
		t.Fatalf("bound origin = %q", guests.parentOrigin)
	}
	var guestPayload struct {
		ExpiresIn int `json:"expires_in"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &guestPayload); err != nil {
		t.Fatal(err)
	}
	if guestPayload.ExpiresIn != 7200 {
		t.Fatalf("guest expires_in = %d, want 7200", guestPayload.ExpiresIn)
	}

	request = httptest.NewRequest(http.MethodPost, "https://api.example/api/embed/auth/guest", strings.NewReader(
		`{"tenant":"school","parent_origin":"https://school.example"}`,
	))
	request.Header.Set("Origin", "https://evil.example")
	response = httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusForbidden {
		t.Fatalf("spoofed parent status = %d, want 403", response.Code)
	}

	request = httptest.NewRequest(http.MethodPost, "https://api.example/api/embed/auth/guest", strings.NewReader(
		`{"tenant":"school","parent_origin":"https://school.example","fingerprint":"`+strings.Repeat("x", 129)+`"}`,
	))
	request.Header.Set("Origin", "https://api.example")
	response = httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusBadRequest {
		t.Fatalf("long fingerprint status = %d, want 400", response.Code)
	}

	request = httptest.NewRequest(http.MethodPost, "https://api.example/api/embed/auth/guest", strings.NewReader(
		`{"tenant":"school","parent_origin":"https://school.example","fingerprint":"`+strings.Repeat("x", maxEmbedRequestBytes)+`"}`,
	))
	request.Header.Set("Origin", "https://api.example")
	response = httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("oversized body status = %d, want 413", response.Code)
	}
}

func TestPublicEmbedConfigOnlyReturnsEnabledAllowedTenant(t *testing.T) {
	store := &embedConfigStoreStub{
		configs: map[string]chat.EmbedConfig{
			"tenant-a": {
				TenantID: "tenant-a",
				Enabled:  true,
				ThemeConfig: map[string]any{
					"color": "#123456", "language": "ms",
				},
			},
		},
		tenantByOrigin: map[string]string{
			"school|https://school.example": "tenant-a",
		},
	}
	handler := NewTopMux(TopMuxOptions{
		EmbedConfigStore: store,
		JWTSecret:        "config-secret",
		AccessTokenTTL:   time.Hour,
	})
	request := httptest.NewRequest(http.MethodGet,
		"https://api.example/api/embed/config?tenant=school&parent_origin=https%3A%2F%2Fschool.example", nil)
	request.Header.Set("Origin", "https://school.example")
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("status = %d: %s", response.Code, response.Body.String())
	}
	var payload struct {
		Enabled bool           `json:"enabled"`
		Theme   map[string]any `json:"theme_config"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &payload); err != nil {
		t.Fatal(err)
	}
	if !payload.Enabled || payload.Theme["language"] != "ms" {
		t.Fatalf("payload = %#v", payload)
	}

	store.configs["tenant-a"] = chat.EmbedConfig{TenantID: "tenant-a", Enabled: false}
	response = httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusForbidden {
		t.Fatalf("disabled status = %d, want 403", response.Code)
	}
}

func TestAdminEmbedConfigUsesConfiguredOrForwardedPublicBaseURL(t *testing.T) {
	const secret = "embed-base-test"
	store := &embedConfigStoreStub{configs: map[string]chat.EmbedConfig{"tenant-a": {TenantID: "tenant-a"}}}
	token := issueEmbedTestToken(t, secret, auth.RoleAdmin, "tenant-a", "")
	for _, test := range []struct {
		name       string
		configured string
		forwarded  string
		want       string
	}{
		{name: "configured", configured: "https://chat.example/", forwarded: "admin.example", want: "https://chat.example"},
		{name: "forwarded request", forwarded: "admin.example", want: "https://admin.example"},
	} {
		t.Run(test.name, func(t *testing.T) {
			handler := NewTopMux(TopMuxOptions{
				EmbedConfigStore: store,
				EmbedBaseURL:     test.configured,
				JWTSecret:        secret,
				AccessTokenTTL:   time.Hour,
			})
			request := httptest.NewRequest(http.MethodGet, "http://internal/api/admin/embed/config", nil)
			request.Header.Set("Authorization", "Bearer "+token)
			request.Header.Set("X-Forwarded-Host", test.forwarded)
			request.Header.Set("X-Forwarded-Proto", "https")
			response := httptest.NewRecorder()
			handler.ServeHTTP(response, request)
			var payload struct {
				PublicEmbedBaseURL string `json:"public_embed_base_url"`
			}
			if err := json.Unmarshal(response.Body.Bytes(), &payload); err != nil {
				t.Fatal(err)
			}
			if payload.PublicEmbedBaseURL != test.want {
				t.Fatalf("public base URL = %q, want %q", payload.PublicEmbedBaseURL, test.want)
			}
		})
	}
}

func TestEmbedMessagesUsesTokenTenantAndUser(t *testing.T) {
	const secret = "embed-message-secret"
	messages := &embedMessagesStub{}
	store := &embedConfigStoreStub{
		configs: map[string]chat.EmbedConfig{},
		tenantByOrigin: map[string]string{
			"https://school.example": "tenant-a",
		},
	}
	handler := NewTopMux(TopMuxOptions{
		EmbedConfigStore:  store,
		EmbedMessageStore: messages,
		JWTSecret:         secret,
		AccessTokenTTL:    time.Hour,
	})
	token := issueEmbedTestToken(t, secret, auth.RoleGuest, "tenant-a", "https://school.example")
	request := httptest.NewRequest(http.MethodGet, "https://api.example/api/embed/messages", nil)
	request.Header.Set("Authorization", "Bearer "+token)
	request.Header.Set("Origin", "https://api.example")
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("status = %d: %s", response.Code, response.Body.String())
	}
	if messages.tenantID != "tenant-a" || messages.userID != "user-a" {
		t.Fatalf("history scope = %q/%q", messages.tenantID, messages.userID)
	}

	for _, test := range []struct {
		name    string
		origin  string
		referer string
		status  int
		revoke  bool
	}{
		{name: "same-origin browser GET without origin", status: http.StatusOK},
		{
			name:    "browser referer with embed path",
			referer: "https://api.example/embed/chat?tenant=school",
			status:  http.StatusOK,
		},
		{name: "mismatched request origin", origin: "https://evil.example", status: http.StatusForbidden},
		{name: "revoked parent origin", origin: "https://api.example", status: http.StatusForbidden, revoke: true},
	} {
		t.Run(test.name, func(t *testing.T) {
			if test.revoke {
				delete(store.tenantByOrigin, "https://school.example")
				defer func() {
					store.tenantByOrigin["https://school.example"] = "tenant-a"
				}()
			}
			request := httptest.NewRequest(http.MethodGet, "https://api.example/api/embed/messages", nil)
			request.Header.Set("Authorization", "Bearer "+token)
			if test.origin != "" {
				request.Header.Set("Origin", test.origin)
			}
			if test.referer != "" {
				request.Header.Set("Referer", test.referer)
			}
			response := httptest.NewRecorder()
			handler.ServeHTTP(response, request)
			if response.Code != test.status {
				t.Fatalf("status = %d, want %d: %s", response.Code, test.status, response.Body.String())
			}
		})
	}
}

func TestEmbedStudentLoginTokenCarriesInternalAndChannelIdentity(t *testing.T) {
	const secret = "embed-login-secret"
	store := &embedConfigStoreStub{
		configs: map[string]chat.EmbedConfig{},
		tenantByOrigin: map[string]string{
			"school|https://school.example": "tenant-a",
		},
	}
	authSvc := embedAuthenticatorStub{user: auth.UserSession{
		UserID: "internal-student-id", TenantID: "tenant-a", Role: auth.RoleStudent, Name: "Student",
	}}
	handler := NewTopMux(TopMuxOptions{
		EmbedConfigStore:   store,
		EmbedAuthenticator: authSvc,
		EmbedIdentityResolver: embedIdentityResolverStub{identity: EmbedIdentity{
			Channel: "telegram", ExternalID: "telegram-student-id",
		}},
		JWTSecret:      secret,
		AccessTokenTTL: 15 * time.Minute,
		EmbedTokenTTL:  2 * time.Hour,
	})
	request := httptest.NewRequest(http.MethodPost, "https://api.example/api/embed/auth/login", strings.NewReader(
		`{"tenant":"school","parent_origin":"https://school.example","email":"student@example.com","password":"password"}`,
	))
	request.Header.Set("Origin", "https://api.example")
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("status = %d: %s", response.Code, response.Body.String())
	}
	var payload struct {
		Token     string `json:"token"`
		ExpiresIn int    `json:"expires_in"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &payload); err != nil {
		t.Fatal(err)
	}
	claims, err := auth.NewTokenManager(secret, time.Hour).Parse(payload.Token, time.Now().UTC())
	if err != nil {
		t.Fatal(err)
	}
	if claims.Subject != "internal-student-id" || claims.TenantID != "tenant-a" {
		t.Fatalf("internal identity = %q/%q", claims.TenantID, claims.Subject)
	}
	if claims.Channel != "telegram" || claims.ExternalID != "telegram-student-id" {
		t.Fatalf("channel identity = %q/%q", claims.Channel, claims.ExternalID)
	}
	if payload.ExpiresIn != 7200 {
		t.Fatalf("login expires_in = %d, want 7200", payload.ExpiresIn)
	}
	if got := claims.ExpiresAt.Sub(claims.IssuedAt); got != 2*time.Hour {
		t.Fatalf("login token lifetime = %s, want 2h", got)
	}
}

func issueEmbedTestToken(t *testing.T, secret string, role auth.Role, tenantID, parentOrigin string) string {
	t.Helper()
	token, err := auth.NewTokenManager(secret, time.Hour).Issue(auth.TokenClaims{
		Subject: "user-a", TenantID: tenantID, Role: role, ParentOrigin: parentOrigin,
	}, time.Now().UTC())
	if err != nil {
		t.Fatal(err)
	}
	return token
}

func strconvQuote(value string) string {
	bytes, err := json.Marshal(value)
	if err != nil {
		panic(errors.New("quote string"))
	}
	return string(bytes)
}
