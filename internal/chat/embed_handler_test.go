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
		"parent_origin: window.location.origin",
		"}, baseOrigin)",
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
	} {
		if !strings.Contains(rec.Body.String(), expected) {
			t.Errorf("chat page missing %q", expected)
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
