package auth

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestTokenManagerRoundTrip(t *testing.T) {
	t.Parallel()

	manager := NewTokenManager("test-secret", time.Minute)
	now := time.Date(2026, 3, 13, 10, 0, 0, 0, time.UTC)

	token, err := manager.Issue(TokenClaims{
		Subject:  "user-123",
		TenantID: "tenant-abc",
		Role:     RoleTeacher,
	}, now)
	if err != nil {
		t.Fatalf("Issue() error = %v", err)
	}

	claims, err := manager.Parse(token, now.Add(30*time.Second))
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}

	if claims.Subject != "user-123" {
		t.Fatalf("Subject = %q, want user-123", claims.Subject)
	}
	if claims.TenantID != "tenant-abc" {
		t.Fatalf("TenantID = %q, want tenant-abc", claims.TenantID)
	}
	if claims.Role != RoleTeacher {
		t.Fatalf("Role = %q, want %q", claims.Role, RoleTeacher)
	}
	if claims.IssuedAt.IsZero() {
		t.Fatal("IssuedAt should be set")
	}
	if claims.ExpiresAt.IsZero() {
		t.Fatal("ExpiresAt should be set")
	}
}

func TestTokenManagerParseErrors(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 3, 13, 10, 0, 0, 0, time.UTC)
	validManager := NewTokenManager("test-secret", time.Minute)
	wrongSecretManager := NewTokenManager("wrong-secret", time.Minute)

	token, err := validManager.Issue(TokenClaims{
		Subject:  "user-123",
		TenantID: "tenant-abc",
		Role:     RoleAdmin,
	}, now)
	if err != nil {
		t.Fatalf("Issue() error = %v", err)
	}

	tests := []struct {
		name    string
		token   string
		manager *TokenManager
		now     time.Time
		wantErr error
	}{
		{
			name:    "expired token",
			token:   token,
			manager: validManager,
			now:     now.Add(2 * time.Minute),
			wantErr: ErrExpiredToken,
		},
		{
			name:    "invalid signature",
			token:   token,
			manager: wrongSecretManager,
			now:     now.Add(30 * time.Second),
			wantErr: ErrInvalidToken,
		},
		{
			name:    "malformed token",
			token:   "bad-token",
			manager: validManager,
			now:     now.Add(30 * time.Second),
			wantErr: ErrInvalidToken,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := tt.manager.Parse(tt.token, tt.now)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("Parse() error = %v, want %v", err, tt.wantErr)
			}
		})
	}
}

func TestClaimsFromContext(t *testing.T) {
	t.Parallel()

	ctx := WithClaims(context.Background(), TokenClaims{
		Subject:  "user-123",
		TenantID: "tenant-abc",
		Role:     RoleParent,
	})

	claims, ok := ClaimsFromContext(ctx)
	if !ok {
		t.Fatal("ClaimsFromContext() ok = false, want true")
	}
	if claims.Role != RoleParent {
		t.Fatalf("Role = %q, want %q", claims.Role, RoleParent)
	}
}
