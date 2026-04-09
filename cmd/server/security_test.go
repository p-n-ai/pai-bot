package main

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
