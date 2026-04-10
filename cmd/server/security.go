package main

import (
	"crypto/sha256"
	"encoding/hex"
	"math"
	"net"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/p-n-ai/pai-bot/internal/auth"
)

const (
	defaultAPIRateLimitPerMinute  = 240
	defaultAuthRateLimitPerMinute = 20
)

type fixedWindowLimiter struct {
	limit  int
	window time.Duration

	mu      sync.Mutex
	buckets map[string]fixedWindowState
}

type fixedWindowState struct {
	windowStart time.Time
	count       int
}

func newFixedWindowLimiter(limit int, window time.Duration) *fixedWindowLimiter {
	return &fixedWindowLimiter{
		limit:   limit,
		window:  window,
		buckets: make(map[string]fixedWindowState),
	}
}

func (l *fixedWindowLimiter) Allow(key string, now time.Time) (bool, time.Duration) {
	if l == nil || l.limit <= 0 || l.window <= 0 {
		return true, 0
	}
	if strings.TrimSpace(key) == "" {
		key = "anonymous"
	}

	l.mu.Lock()
	defer l.mu.Unlock()

	state, ok := l.buckets[key]
	if !ok || now.Sub(state.windowStart) >= l.window {
		l.buckets[key] = fixedWindowState{windowStart: now, count: 1}
		return true, 0
	}

	if state.count < l.limit {
		state.count++
		l.buckets[key] = state
		return true, 0
	}

	retryAfter := l.window - now.Sub(state.windowStart)
	if retryAfter < 0 {
		retryAfter = 0
	}
	return false, retryAfter
}

func withAPIRateLimit(next http.Handler, now func() time.Time, apiLimiter, authLimiter *fixedWindowLimiter) http.Handler {
	if now == nil {
		now = time.Now
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasPrefix(r.URL.Path, "/api/") || r.Method == http.MethodOptions {
			next.ServeHTTP(w, r)
			return
		}

		limiter := apiLimiter
		keyPrefix := "api"
		if strings.HasPrefix(r.URL.Path, "/api/auth/") {
			limiter = authLimiter
			keyPrefix = "auth"
		}

		key := keyPrefix + ":" + rateLimitClientKey(r)
		allowed, retryAfter := limiter.Allow(key, now().UTC())
		if !allowed {
			seconds := int(math.Ceil(retryAfter.Seconds()))
			if seconds < 1 {
				seconds = 1
			}
			w.Header().Set("Retry-After", strconv.Itoa(seconds))
			http.Error(w, http.StatusText(http.StatusTooManyRequests), http.StatusTooManyRequests)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func withSecurityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("Referrer-Policy", "no-referrer")
		w.Header().Set("Cross-Origin-Resource-Policy", "same-site")
		next.ServeHTTP(w, r)
	})
}

func rateLimitClientKey(r *http.Request) string {
	sessionToken := readCookieValue(r, auth.SessionCookieName)
	if sessionToken != "" {
		return "session:" + shortTokenHash(sessionToken)
	}
	if token, err := bearerToken(r.Header.Get("Authorization")); err == nil {
		return "bearer:" + shortTokenHash(token)
	}

	ip := strings.TrimSpace(firstForwardedFor(r.Header.Get("X-Forwarded-For")))
	if ip == "" {
		ip = strings.TrimSpace(r.Header.Get("X-Real-IP"))
	}
	if ip == "" {
		host, _, err := net.SplitHostPort(strings.TrimSpace(r.RemoteAddr))
		if err == nil {
			ip = host
		}
	}
	if ip == "" {
		ip = "unknown"
	}

	return "ip:" + ip
}

func firstForwardedFor(v string) string {
	if strings.TrimSpace(v) == "" {
		return ""
	}
	parts := strings.Split(v, ",")
	if len(parts) == 0 {
		return ""
	}
	return strings.TrimSpace(parts[0])
}

func shortTokenHash(v string) string {
	sum := sha256.Sum256([]byte(v))
	return hex.EncodeToString(sum[:6])
}
