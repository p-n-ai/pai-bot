// Copyright 2026 the P&AI authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package server

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestPublicHealthFeature(t *testing.T) {
	enabled := false
	handler := NewTopMux(TopMuxOptions{
		APIHandler: http.HandlerFunc(handleHealthz),
		PublicHealthEnabled: func() bool {
			return enabled
		},
	})

	assertResponse := func(path string, wantStatus int, wantBody string) *httptest.ResponseRecorder {
		t.Helper()
		request := httptest.NewRequest(http.MethodGet, path, nil)
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, request)
		if response.Code != wantStatus {
			t.Fatalf("%s status = %d, want %d", path, response.Code, wantStatus)
		}
		if response.Body.String() != wantBody {
			t.Fatalf("%s body = %q, want %q", path, response.Body.String(), wantBody)
		}
		return response
	}

	assertResponse("/health/api", http.StatusNotFound, "404 page not found\n")
	assertResponse("/health/status", http.StatusNotFound, "404 page not found\n")
	assertResponse("/healthz", http.StatusOK, `{"status":"ok"}`)

	enabled = true
	assertResponse("/health/api", http.StatusOK, `{"status":"ok"}`)
	assertResponse(
		"/health/status",
		http.StatusOK,
		"{\"status\":\"ok\",\"components\":[{\"id\":\"application\",\"status\":\"operational\"}]}\n",
	)
	assertResponse("/healthz", http.StatusOK, `{"status":"ok"}`)
}

func TestPublicStatusPageIncludesCoarseAIState(t *testing.T) {
	checkErr := error(nil)
	handler := NewTopMux(TopMuxOptions{
		PublicHealthEnabled: func() bool { return true },
		AIHealthEnabled:     func() bool { return true },
		AIHealthCheck: func(context.Context) error {
			return checkErr
		},
	})

	request := func() string {
		t.Helper()
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/health/status", nil))
		if response.Code != http.StatusOK {
			t.Fatalf("/health/status status = %d, want 200", response.Code)
		}
		return response.Body.String()
	}

	healthyBody := request()
	for _, expected := range []string{`"status":"ok"`, `"id":"ai"`, `"status":"operational"`} {
		if !strings.Contains(healthyBody, expected) {
			t.Fatalf("healthy public status missing %q", expected)
		}
	}

	checkErr = errors.New("private provider detail")
	unavailableBody := request()
	for _, expected := range []string{`"status":"degraded"`, `"status":"unavailable"`} {
		if !strings.Contains(unavailableBody, expected) {
			t.Fatalf("unavailable public status missing %q", expected)
		}
	}
	if strings.Contains(unavailableBody, checkErr.Error()) {
		t.Fatal("public status exposed provider error")
	}
}

func TestAIHealthFeatureAndAuthentication(t *testing.T) {
	enabled := false
	checkCalls := 0
	checkErr := error(nil)
	handler := NewTopMux(TopMuxOptions{
		AIHealthEnabled: func() bool {
			return enabled
		},
		AIHealthToken: "monitor-secret",
		AIHealthCheck: func(context.Context) error {
			checkCalls++
			return checkErr
		},
	})

	request := func(token string) *httptest.ResponseRecorder {
		t.Helper()
		req := httptest.NewRequest(http.MethodGet, "/health/ai", nil)
		if token != "" {
			req.Header.Set("Authorization", "Bearer "+token)
		}
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		return rec
	}

	if rec := request("monitor-secret"); rec.Code != http.StatusNotFound {
		t.Fatalf("disabled status = %d, want 404", rec.Code)
	}
	enabled = true
	for _, token := range []string{"", "wrong-secret"} {
		if rec := request(token); rec.Code != http.StatusNotFound {
			t.Fatalf("token %q status = %d, want 404", token, rec.Code)
		}
	}
	if checkCalls != 0 {
		t.Fatalf("unauthorized check calls = %d, want 0", checkCalls)
	}

	rec := request("monitor-secret")
	if rec.Code != http.StatusOK || rec.Body.String() != `{"status":"ok"}` {
		t.Fatalf("healthy response = %d %q", rec.Code, rec.Body.String())
	}

	checkErr = errors.New("provider secret detail")
	rec = request("monitor-secret")
	if rec.Code != http.StatusServiceUnavailable || rec.Body.String() != `{"status":"unavailable"}` {
		t.Fatalf("unavailable response = %d %q", rec.Code, rec.Body.String())
	}
	if got := rec.Body.String(); strings.Contains(got, checkErr.Error()) {
		t.Fatal("public response exposed provider error")
	}
}
