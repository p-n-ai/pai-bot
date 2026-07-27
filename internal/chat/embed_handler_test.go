// Copyright 2026 the P&AI authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package chat

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestHandleWidgetJS_ReturnsJavaScript(t *testing.T) {
	handler := HandleWidgetJS()
	req := httptest.NewRequest(http.MethodGet, "/embed/pai-chat.js", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rec.Code)
	}
	ct := rec.Header().Get("Content-Type")
	if ct != "application/javascript; charset=utf-8" {
		t.Errorf("expected javascript content type, got %q", ct)
	}
	if rec.Body.Len() == 0 {
		t.Error("expected non-empty body")
	}
}

func TestHandleWidgetJS_AccessibleIdempotentAndParentOriginBound(t *testing.T) {
	handler := HandleWidgetJS()
	req := httptest.NewRequest(http.MethodGet, "/embed/pai-chat.js", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)
	body := rec.Body.String()
	for _, expected := range []string{
		"document.createElement('button')",
		"aria-expanded",
		"document.getElementById('pai-chat-toggle')",
		"max-width:calc(100vw - 40px)",
		"parent_origin: parentOrigin",
		"}, baseOrigin)",
		"fetch(configURL",
		"config.enabled !== true",
		"Buka sembang P&AI",
		"打开 P&AI 聊天",
		"readableForeground(color)",
	} {
		if !strings.Contains(body, expected) {
			t.Errorf("widget script missing %q", expected)
		}
	}
}

func TestHandleChatPage_ValidTenant(t *testing.T) {
	handler := HandleChatPage(nil)
	req := httptest.NewRequest(http.MethodGet, "/embed/chat?tenant=demo", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rec.Code)
	}
	ct := rec.Header().Get("Content-Type")
	if ct != "text/html; charset=utf-8" {
		t.Errorf("expected html content type, got %q", ct)
	}
	for _, expected := range []string{
		"event.source !== window.parent",
		"event.origin !== normalized",
		"parent_origin: parentOrigin",
		"savedSession.parentOrigin === parentOrigin",
		"tokenHasEmbedIdentity(savedSession.token)",
		"'/ws/embed'",
		"if (!historyLoaded)",
		"document.documentElement.lang = lang",
		"Simpan kemajuan anda",
		"保存学习进度",
		"readableForeground(color)",
	} {
		if !strings.Contains(rec.Body.String(), expected) {
			t.Errorf("chat page missing %q", expected)
		}
	}
	for _, forbidden := range []string{"var storageKey = 'pai-chat-'", "function loadHistory()", "function saveHistory()"} {
		if strings.Contains(rec.Body.String(), forbidden) {
			t.Errorf("chat page retains duplicate-prone local history code %q", forbidden)
		}
	}
}

func TestHandleChatPage_MissingTenant(t *testing.T) {
	handler := HandleChatPage(nil)
	req := httptest.NewRequest(http.MethodGet, "/embed/chat", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rec.Code)
	}
}

func TestHandleChatPage_CSPFailsClosedUntilEnabledConfigLoads(t *testing.T) {
	for _, test := range []struct {
		name  string
		store EmbedConfigStore
		want  string
	}{
		{name: "unwired", store: nil, want: "frame-ancestors 'none'"},
		{
			name: "disabled",
			store: &MockEmbedConfigStore{
				Tenants: map[string]string{"school": "tenant-1"},
				Configs: map[string]EmbedConfig{
					"tenant-1": {Enabled: false, AllowedOrigins: []string{"https://school.example"}},
				},
			},
			want: "frame-ancestors 'none'",
		},
		{
			name: "enabled",
			store: &MockEmbedConfigStore{
				Tenants: map[string]string{"school": "tenant-1"},
				Configs: map[string]EmbedConfig{
					"tenant-1": {Enabled: true, AllowedOrigins: []string{"https://school.example"}},
				},
			},
			want: "frame-ancestors https://school.example",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			rec := httptest.NewRecorder()
			HandleChatPage(test.store).ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/embed/chat?tenant=school", nil))
			if got := rec.Header().Get("Content-Security-Policy"); got != test.want {
				t.Fatalf("CSP = %q, want %q", got, test.want)
			}
		})
	}
}
