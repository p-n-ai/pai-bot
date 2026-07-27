// Copyright 2026 the P&AI authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package chat

import (
	"sync"
	"time"
)

// EmbedRateLimiter provides rate limiting for embed WebSocket connections.
type EmbedRateLimiter struct {
	handshakeLimit int // max handshakes per IP per window
	messageLimit   int // max messages per user per window
	window         time.Duration
	maxBuckets     int

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
		maxBuckets:     10000,
		handshakes:     make(map[string]rateLimitState),
		messages:       make(map[string]rateLimitState),
	}
}

// AllowHandshake checks if a WebSocket handshake for the authenticated identity is allowed.
func (rl *EmbedRateLimiter) AllowHandshake(identityKey string, now time.Time) bool {
	if rl == nil {
		return true
	}
	return rl.allow(rl.handshakes, identityKey, rl.handshakeLimit, now)
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
		rl.ensureBucketCapacity(buckets, key, now)
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

func (rl *EmbedRateLimiter) ensureBucketCapacity(buckets map[string]rateLimitState, key string, now time.Time) {
	if _, exists := buckets[key]; exists || rl.maxBuckets <= 0 || len(buckets) < rl.maxBuckets {
		return
	}
	for bucketKey, state := range buckets {
		if !state.windowStart.Add(rl.window).After(now) {
			delete(buckets, bucketKey)
		}
	}
	if len(buckets) < rl.maxBuckets {
		return
	}
	var oldestKey string
	var oldestStart time.Time
	for bucketKey, state := range buckets {
		if oldestKey == "" || state.windowStart.Before(oldestStart) {
			oldestKey = bucketKey
			oldestStart = state.windowStart
		}
	}
	delete(buckets, oldestKey)
}
