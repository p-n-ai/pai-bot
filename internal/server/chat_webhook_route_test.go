// Copyright 2026 the P&AI authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package server

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestTopMuxRoutesEachChatSDKWebhookToItsAdapter(t *testing.T) {
	called := make(map[string]int)
	webhooks := make(map[string]http.Handler)
	for _, name := range []string{"slack", "discord", "teams"} {
		name := name
		webhooks[name] = http.HandlerFunc(func(response http.ResponseWriter, _ *http.Request) {
			called[name]++
			response.WriteHeader(http.StatusNoContent)
		})
	}
	handler := NewTopMux(TopMuxOptions{
		APIHandler:       http.NotFoundHandler(),
		ChatWebhooks:     webhooks,
		JWTSecret:        "test-secret",
		AccessTokenTTL:   time.Hour,
		EmbedConfigStore: nil,
	})

	for _, name := range []string{"slack", "discord", "teams"} {
		request := httptest.NewRequest(http.MethodPost, "/webhook/"+name, nil)
		recorder := httptest.NewRecorder()
		handler.ServeHTTP(recorder, request)
		if recorder.Code != http.StatusNoContent {
			t.Fatalf("%s webhook status = %d, want %d", name, recorder.Code, http.StatusNoContent)
		}
	}
	if called["slack"] != 1 || called["discord"] != 1 || called["teams"] != 1 {
		t.Fatalf("webhook calls = %#v, want each adapter once", called)
	}

	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, httptest.NewRequest(http.MethodPost, "/webhook/unknown", nil))
	if recorder.Code != http.StatusNotFound {
		t.Fatalf("unknown webhook status = %d, want %d", recorder.Code, http.StatusNotFound)
	}
}
