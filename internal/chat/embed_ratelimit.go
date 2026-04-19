// Copyright 2026 the P&AI authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package chat

import (
	"sync"
	"time"
)

// EmbedRateLimiter provides rate limiting for embed WebSocket connections.
type EmbedRateLimiter struct {
	handshakeLimit int           // max handshakes per IP per window
	messageLimit   int           // max messages per user per window
	window         time.Duration

	mu         sync.Mutex
	handshakes map[string]rateLimitState
	messages   map[string]rateLimitState
}

type rateLimitState struct {
	windowStart time.Time
	count       int
}

// NewEmbedRateLimiter creates a rate limiter with the given limits per window.
func NewEmbedRateLimiter(handshakeLimit, messageLimit int, window time.Duration) *EmbedRateLimiter {
	return &EmbedRateLimiter{
		handshakeLimit: handshakeLimit,
		messageLimit:   messageLimit,
		window:         window,
		handshakes:     make(map[string]rateLimitState),
		messages:       make(map[string]rateLimitState),
	}
}

// AllowHandshake checks if a WebSocket handshake from the given IP is allowed.
func (rl *EmbedRateLimiter) AllowHandshake(ip string, now time.Time) bool {
	if rl == nil {
		return true
	}
	return rl.allow(rl.handshakes, ip, rl.handshakeLimit, now)
}

// AllowMessage checks if a message from the given user is allowed.
func (rl *EmbedRateLimiter) AllowMessage(userID string, now time.Time) bool {
	if rl == nil {
		return true
	}
	return rl.allow(rl.messages, userID, rl.messageLimit, now)
}

func (rl *EmbedRateLimiter) allow(buckets map[string]rateLimitState, key string, limit int, now time.Time) bool {
	if limit <= 0 {
		return true
	}

	rl.mu.Lock()
	defer rl.mu.Unlock()

	state, ok := buckets[key]
	if !ok || now.Sub(state.windowStart) >= rl.window {
		buckets[key] = rateLimitState{windowStart: now, count: 1}
		return true
	}

	if state.count < limit {
		state.count++
		buckets[key] = state
		return true
	}

	return false
}
