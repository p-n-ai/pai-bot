package auth

import (
	"context"
	"errors"
	"net/http"
	"slices"
	"strings"
	"time"
)

type contextKey string

const claimsContextKey contextKey = "auth_claims"

func WithClaims(ctx context.Context, claims TokenClaims) context.Context {
	return context.WithValue(ctx, claimsContextKey, claims)
}

func ClaimsFromContext(ctx context.Context) (TokenClaims, bool) {
	claims, ok := ctx.Value(claimsContextKey).(TokenClaims)
	return claims, ok
}

func Authenticate(manager *TokenManager, now func() time.Time) func(http.Handler) http.Handler {
	if now == nil {
		now = time.Now
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			token, err := requestToken(r)
			if err != nil {
				http.Error(w, err.Error(), http.StatusUnauthorized)
				return
			}

			claims, err := manager.Parse(token, now().UTC())
			if err != nil {
				if errors.Is(err, ErrExpiredToken) {
					http.Error(w, "expired token", http.StatusUnauthorized)
					return
				}
				http.Error(w, "invalid token", http.StatusUnauthorized)
				return
			}

			next.ServeHTTP(w, r.WithContext(WithClaims(r.Context(), claims)))
		})
	}
}

func RequireRoles(allowed ...Role) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			claims, ok := ClaimsFromContext(r.Context())
			if !ok {
				http.Error(w, "missing auth claims", http.StatusUnauthorized)
				return
			}
			if !slices.Contains(allowed, claims.Role) {
				http.Error(w, "forbidden", http.StatusForbidden)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

func bearerToken(header string) (string, error) {
	parts := strings.Fields(header)
	if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") || strings.TrimSpace(parts[1]) == "" {
		return "", errors.New("missing auth token")
	}
	return parts[1], nil
}

func requestToken(r *http.Request) (string, error) {
	if token, err := bearerToken(r.Header.Get("Authorization")); err == nil {
		return token, nil
	}

	cookie, err := r.Cookie(AccessTokenCookieName)
	if err != nil || strings.TrimSpace(cookie.Value) == "" {
		return "", errors.New("missing auth token")
	}

	return cookie.Value, nil
}
