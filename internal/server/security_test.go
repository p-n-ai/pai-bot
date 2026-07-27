// Copyright 2026 the P&AI authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package server

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestFixedWindowLimiter(t *testing.T) {
	limiter := newFixedWindowLimiter(2, time.Minute)
	now := time.Date(2026, 4, 9, 10, 0, 0, 0, time.UTC)

	allowed, retryAfter := limiter.Allow("ip:127.0.0.1", now)
	if !allowed || retryAfter != 0 {
		t.Fatalf("first request = (%v, %v), want (true, 0)", allowed, retryAfter)
	}

	allowed, retryAfter = limiter.Allow("ip:127.0.0.1", now.Add(10*time.Second))
	if !allowed || retryAfter != 0 {
		t.Fatalf("second request = (%v, %v), want (true, 0)", allowed, retryAfter)
	}

	allowed, retryAfter = limiter.Allow("ip:127.0.0.1", now.Add(20*time.Second))
	if allowed {
		t.Fatal("third request should be denied in same window")
	}
	if retryAfter <= 0 {
		t.Fatalf("retryAfter = %v, want > 0", retryAfter)
	}

	allowed, retryAfter = limiter.Allow("ip:127.0.0.1", now.Add(61*time.Second))
	if !allowed || retryAfter != 0 {
		t.Fatalf("request after new window = (%v, %v), want (true, 0)", allowed, retryAfter)
	}
}

func TestFixedWindowLimiterBoundsAndPrunesBuckets(t *testing.T) {
	limiter := newFixedWindowLimiter(2, time.Minute)
	limiter.maxKeys = 2
	now := time.Date(2026, 4, 9, 10, 0, 0, 0, time.UTC)

	limiter.Allow("first", now)
	limiter.Allow("second", now.Add(time.Second))
	limiter.Allow("third", now.Add(2*time.Second))
	if len(limiter.buckets) != 2 {
		t.Fatalf("bucket count = %d, want 2", len(limiter.buckets))
	}
	if _, exists := limiter.buckets["first"]; exists {
		t.Fatal("oldest bucket was not evicted at capacity")
	}

	limiter.Allow("fourth", now.Add(2*time.Minute))
	if len(limiter.buckets) != 1 {
		t.Fatalf("expired bucket count = %d, want 1", len(limiter.buckets))
	}
}

func TestWithAPIRateLimit_AuthEndpointReturns429(t *testing.T) {
	apiLimiter := newFixedWindowLimiter(100, time.Minute)
	authLimiter := newFixedWindowLimiter(2, time.Minute)
	now := time.Date(2026, 4, 9, 10, 0, 0, 0, time.UTC)
	nowFn := func() time.Time { return now }

	handler := withAPIRateLimit(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}), nowFn, apiLimiter, authLimiter)

	for i := 0; i < 2; i++ {
		req := httptest.NewRequest(http.MethodPost, "/api/auth/login", nil)
		req.RemoteAddr = "203.0.113.10:43123"
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("request %d status = %d, want 200", i+1, rec.Code)
		}
	}

	req := httptest.NewRequest(http.MethodPost, "/api/auth/login", nil)
	req.RemoteAddr = "203.0.113.10:43123"
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusTooManyRequests {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusTooManyRequests)
	}
	if rec.Header().Get("Retry-After") == "" {
		t.Fatal("expected Retry-After header")
	}
}

func TestWithAPIRateLimit_AuthEndpointIgnoresCallerSelectedKeys(t *testing.T) {
	apiLimiter := newFixedWindowLimiter(100, time.Minute)
	authLimiter := newFixedWindowLimiter(1, time.Minute)
	now := time.Date(2026, 4, 9, 10, 0, 0, 0, time.UTC)
	handler := withAPIRateLimit(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}), func() time.Time { return now }, apiLimiter, authLimiter)

	first := httptest.NewRequest(http.MethodPost, "/api/embed/auth/login", nil)
	first.RemoteAddr = "203.0.113.10:43123"
	first.Header.Set("Authorization", "Bearer attacker-selected-one")
	first.Header.Set("X-Forwarded-For", "198.51.100.1")
	firstResponse := httptest.NewRecorder()
	handler.ServeHTTP(firstResponse, first)
	if firstResponse.Code != http.StatusOK {
		t.Fatalf("first status = %d, want 200", firstResponse.Code)
	}

	second := httptest.NewRequest(http.MethodPost, "/api/embed/auth/login", nil)
	second.RemoteAddr = "203.0.113.10:43123"
	second.Header.Set("Authorization", "Bearer attacker-selected-two")
	second.Header.Set("X-Forwarded-For", "198.51.100.2")
	secondResponse := httptest.NewRecorder()
	handler.ServeHTTP(secondResponse, second)
	if secondResponse.Code != http.StatusTooManyRequests {
		t.Fatalf("second status = %d, want 429", secondResponse.Code)
	}
}

func TestRateLimitClientKeyTrustsOnlySameHostProxy(t *testing.T) {
	direct := httptest.NewRequest(http.MethodGet, "/api/test", nil)
	direct.RemoteAddr = "203.0.113.10:43123"
	direct.Header.Set("X-Forwarded-For", "198.51.100.20")
	if got := rateLimitClientKey(direct, false); got != "ip:203.0.113.10" {
		t.Fatalf("direct client key = %q", got)
	}

	proxied := httptest.NewRequest(http.MethodGet, "/api/test", nil)
	proxied.RemoteAddr = "127.0.0.1:43123"
	proxied.Header.Set("X-Forwarded-For", "198.51.100.30, 203.0.113.30")
	if got := rateLimitClientKey(proxied, false); got != "ip:203.0.113.30" {
		t.Fatalf("same-host proxy client key = %q", got)
	}
}

func TestWithAPIRateLimit_CapabilitiesDoNotConsumeAuthBudget(t *testing.T) {
	apiLimiter := newFixedWindowLimiter(100, time.Minute)
	authLimiter := newFixedWindowLimiter(1, time.Minute)
	now := time.Date(2026, 4, 9, 10, 0, 0, 0, time.UTC)

	handler := withAPIRateLimit(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}), func() time.Time { return now }, apiLimiter, authLimiter)

	capabilitiesReq := httptest.NewRequest(http.MethodGet, "/api/auth/capabilities", nil)
	capabilitiesReq.RemoteAddr = "203.0.113.10:43123"
	capabilitiesRec := httptest.NewRecorder()
	handler.ServeHTTP(capabilitiesRec, capabilitiesReq)
	if capabilitiesRec.Code != http.StatusOK {
		t.Fatalf("capabilities status = %d, want %d", capabilitiesRec.Code, http.StatusOK)
	}

	loginReq := httptest.NewRequest(http.MethodPost, "/api/auth/login", nil)
	loginReq.RemoteAddr = "203.0.113.10:43123"
	loginRec := httptest.NewRecorder()
	handler.ServeHTTP(loginRec, loginReq)
	if loginRec.Code != http.StatusOK {
		t.Fatalf("first login status = %d, want %d", loginRec.Code, http.StatusOK)
	}

	secondLoginReq := httptest.NewRequest(http.MethodPost, "/api/auth/login", nil)
	secondLoginReq.RemoteAddr = "203.0.113.10:43123"
	secondLoginRec := httptest.NewRecorder()
	handler.ServeHTTP(secondLoginRec, secondLoginReq)
	if secondLoginRec.Code != http.StatusTooManyRequests {
		t.Fatalf("second login status = %d, want %d", secondLoginRec.Code, http.StatusTooManyRequests)
	}
}

func TestWithSecurityHeaders(t *testing.T) {
	handler := withSecurityHeaders(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))

	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if got := rec.Header().Get("X-Content-Type-Options"); got != "nosniff" {
		t.Fatalf("X-Content-Type-Options = %q, want nosniff", got)
	}
	if got := rec.Header().Get("X-Frame-Options"); got != "DENY" {
		t.Fatalf("X-Frame-Options = %q, want DENY", got)
	}
	if got := rec.Header().Get("Referrer-Policy"); got != "no-referrer" {
		t.Fatalf("Referrer-Policy = %q, want no-referrer", got)
	}
}
