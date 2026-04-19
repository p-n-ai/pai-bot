// Copyright 2026 the P&AI authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package chat

import (
	"testing"
	"time"
)

func TestEmbedRateLimiter_Handshake(t *testing.T) {
	// Create limiter with 3 handshakes per minute
	rl := NewEmbedRateLimiter(3, 30, time.Minute)
	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)

	tests := []struct {
		name    string
		ip      string
		allowed bool
	}{
		{"first", "1.2.3.4", true},
		{"second", "1.2.3.4", true},
		{"third", "1.2.3.4", true},
		{"fourth_blocked", "1.2.3.4", false},
		{"different_ip", "5.6.7.8", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := rl.AllowHandshake(tt.ip, now); got != tt.allowed {
				t.Errorf("AllowHandshake(%q) = %v, want %v", tt.ip, got, tt.allowed)
			}
		})
	}
}

func TestEmbedRateLimiter_Message(t *testing.T) {
	rl := NewEmbedRateLimiter(10, 3, time.Minute)
	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)

	// 3 messages allowed, 4th blocked
	for i := 0; i < 3; i++ {
		if !rl.AllowMessage("user-1", now) {
			t.Fatalf("message %d should be allowed", i+1)
		}
	}
	if rl.AllowMessage("user-1", now) {
		t.Fatal("4th message should be blocked")
	}
	// Different user is fine
	if !rl.AllowMessage("user-2", now) {
		t.Fatal("different user should be allowed")
	}
}

func TestEmbedRateLimiter_WindowReset(t *testing.T) {
	rl := NewEmbedRateLimiter(2, 2, time.Minute)
	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)

	rl.AllowHandshake("1.2.3.4", now)
	rl.AllowHandshake("1.2.3.4", now)
	if rl.AllowHandshake("1.2.3.4", now) {
		t.Fatal("should be blocked within window")
	}
	// After window passes, should be allowed again
	later := now.Add(61 * time.Second)
	if !rl.AllowHandshake("1.2.3.4", later) {
		t.Fatal("should be allowed after window reset")
	}
}

func TestEmbedRateLimiter_NilSafe(t *testing.T) {
	var rl *EmbedRateLimiter
	if !rl.AllowHandshake("1.2.3.4", time.Now()) {
		t.Fatal("nil limiter should always allow")
	}
	if !rl.AllowMessage("user", time.Now()) {
		t.Fatal("nil limiter should always allow")
	}
}
