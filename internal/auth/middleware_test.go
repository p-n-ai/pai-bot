package auth

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestAuthenticateMiddleware(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 3, 13, 10, 0, 0, 0, time.UTC)
	manager := NewTokenManager("test-secret", time.Minute)
	token, err := manager.Issue(TokenClaims{
		Subject:  "user-123",
		TenantID: "tenant-abc",
		Role:     RoleTeacher,
	}, now)
	if err != nil {
		t.Fatalf("Issue() error = %v", err)
	}

	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		claims, ok := ClaimsFromContext(r.Context())
		if !ok {
			t.Fatal("ClaimsFromContext() ok = false, want true")
		}
		if claims.Subject != "user-123" {
			t.Fatalf("Subject = %q, want user-123", claims.Subject)
		}
		w.WriteHeader(http.StatusNoContent)
	})

	handler := Authenticate(manager, func() time.Time { return now.Add(30 * time.Second) })(next)
	req := httptest.NewRequest(http.MethodGet, "/api/admin/students/stu_1", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusNoContent)
	}
}

func TestAuthenticateMiddlewareAcceptsSessionCookie(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 3, 13, 10, 0, 0, 0, time.UTC)
	manager := NewTokenManager("test-secret", time.Minute)
	token, err := manager.Issue(TokenClaims{
		Subject:  "user-123",
		TenantID: "tenant-abc",
		Role:     RoleTeacher,
	}, now)
	if err != nil {
		t.Fatalf("Issue() error = %v", err)
	}

	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		claims, ok := ClaimsFromContext(r.Context())
		if !ok {
			t.Fatal("ClaimsFromContext() ok = false, want true")
		}
		if claims.Subject != "user-123" {
			t.Fatalf("Subject = %q, want user-123", claims.Subject)
		}
		w.WriteHeader(http.StatusNoContent)
	})

	handler := Authenticate(manager, func() time.Time { return now.Add(30 * time.Second) })(next)
	req := httptest.NewRequest(http.MethodGet, "/api/admin/students/stu_1", nil)
	req.AddCookie(&http.Cookie{Name: SessionCookieName, Value: token})
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusNoContent)
	}
}

func TestAuthenticateMiddlewareRejectsInvalidRequests(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 3, 13, 10, 0, 0, 0, time.UTC)
	manager := NewTokenManager("test-secret", time.Minute)

	tests := []struct {
		name         string
		header       string
		wantStatus   int
		wantContains string
	}{
		{
			name:         "missing header",
			header:       "",
			wantStatus:   http.StatusUnauthorized,
			wantContains: "missing auth token",
		},
		{
			name:         "wrong scheme",
			header:       "Basic abc",
			wantStatus:   http.StatusUnauthorized,
			wantContains: "missing auth token",
		},
		{
			name:         "malformed token",
			header:       "Bearer bad-token",
			wantStatus:   http.StatusUnauthorized,
			wantContains: "invalid token",
		},
	}

	handler := Authenticate(manager, func() time.Time { return now })(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/api/admin/students/stu_1", nil)
			if tt.header != "" {
				req.Header.Set("Authorization", tt.header)
			}
			rec := httptest.NewRecorder()

			handler.ServeHTTP(rec, req)

			if rec.Code != tt.wantStatus {
				t.Fatalf("status = %d, want %d", rec.Code, tt.wantStatus)
			}
			if body := rec.Body.String(); body == "" || !strings.Contains(body, tt.wantContains) {
				t.Fatalf("body = %q, want substring %q", body, tt.wantContains)
			}
		})
	}
}

func TestRequireRoles(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		role       Role
		allowed    []Role
		wantStatus int
	}{
		{
			name:       "teacher allowed",
			role:       RoleTeacher,
			allowed:    []Role{RoleTeacher, RoleAdmin},
			wantStatus: http.StatusNoContent,
		},
		{
			name:       "student forbidden",
			role:       RoleStudent,
			allowed:    []Role{RoleTeacher, RoleAdmin},
			wantStatus: http.StatusForbidden,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := RequireRoles(tt.allowed...)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusNoContent)
			}))

			req := httptest.NewRequest(http.MethodGet, "/api/admin/students/stu_1", nil)
			req = req.WithContext(WithClaims(req.Context(), TokenClaims{
				Subject:  "user-123",
				TenantID: "tenant-abc",
				Role:     tt.role,
			}))
			rec := httptest.NewRecorder()

			handler.ServeHTTP(rec, req)

			if rec.Code != tt.wantStatus {
				t.Fatalf("status = %d, want %d", rec.Code, tt.wantStatus)
			}
		})
	}
}
