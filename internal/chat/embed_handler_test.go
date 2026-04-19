// Copyright 2026 the P&AI authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package chat

import (
	"net/http"
	"net/http/httptest"
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
