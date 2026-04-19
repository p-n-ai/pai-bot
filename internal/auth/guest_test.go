// Copyright 2026 the P&AI authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package auth

import (
	"testing"
	"time"
)

func TestGuestService_RoleGuestConstant(t *testing.T) {
	if RoleGuest != "guest" {
		t.Errorf("RoleGuest = %q, want %q", RoleGuest, "guest")
	}
}

func TestGuestTokenIssuance(t *testing.T) {
	tm := NewTokenManager("test-secret-for-guest", 0)

	tests := []struct {
		name     string
		claims   TokenClaims
		wantRole Role
	}{
		{
			name: "guest role round-trip",
			claims: TokenClaims{
				Subject:  "user-uuid-1234",
				TenantID: "tenant-abc",
				Role:     RoleGuest,
			},
			wantRole: RoleGuest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			now := time.Now().UTC()
			token, err := tm.Issue(tt.claims, now)
			if err != nil {
				t.Fatalf("Issue() error = %v", err)
			}

			parsed, err := tm.Parse(token, now.Add(time.Second))
			if err != nil {
				t.Fatalf("Parse() error = %v", err)
			}

			if parsed.Subject != tt.claims.Subject {
				t.Errorf("Subject = %q, want %q", parsed.Subject, tt.claims.Subject)
			}
			if parsed.TenantID != tt.claims.TenantID {
				t.Errorf("TenantID = %q, want %q", parsed.TenantID, tt.claims.TenantID)
			}
			if parsed.Role != tt.wantRole {
				t.Errorf("Role = %q, want %q", parsed.Role, tt.wantRole)
			}
		})
	}
}

func TestGuestTokenRequiresTenantID(t *testing.T) {
	tm := NewTokenManager("test-secret-for-guest", 0)

	_, err := tm.Issue(TokenClaims{
		Subject:  "user-uuid-5678",
		TenantID: "",
		Role:     RoleGuest,
	}, time.Now().UTC())

	if err == nil {
		t.Fatal("Issue() with empty TenantID should return error, got nil")
	}
}

func TestNewGuestService_Construction(t *testing.T) {
	tm := NewTokenManager("test-secret", 0)
	gs := NewGuestService(nil, tm)
	if gs == nil {
		t.Fatal("NewGuestService() returned nil")
	}
	if gs.tokenManager != tm {
		t.Error("tokenManager not set correctly")
	}
	if gs.pool != nil {
		t.Error("pool should be nil in this test")
	}
}

func TestGuestService_FingerprintReuse(t *testing.T) {
	// findGuestByFingerprint with an empty fingerprint must return empty string without error.
	// This exercises the early-return guard that prevents unnecessary DB queries.
	tm := NewTokenManager("test-secret", 0)
	gs := NewGuestService(nil, tm)

	userID, err := gs.findGuestByFingerprint(t.Context(), "tenant-abc", "")
	if err != nil {
		t.Fatalf("findGuestByFingerprint() with empty fingerprint returned error: %v", err)
	}
	if userID != "" {
		t.Errorf("findGuestByFingerprint() with empty fingerprint = %q, want empty string", userID)
	}
}
