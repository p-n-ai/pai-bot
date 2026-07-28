// Copyright 2026 the P&AI authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package server

import (
	"context"
	"errors"
	"sync"
	"time"
)

type cachedHealthCheck struct {
	check   func(context.Context) error
	ttl     time.Duration
	timeout time.Duration
	now     func() time.Time

	mu       sync.Mutex
	valid    bool
	err      error
	expires  time.Time
	inFlight chan struct{}
}

// NewCachedHealthCheck bounds and coalesces an external health check.
func NewCachedHealthCheck(check func(context.Context) error, ttl, timeout time.Duration) func(context.Context) error {
	return newCachedHealthCheck(check, ttl, timeout, time.Now).Check
}

func newCachedHealthCheck(check func(context.Context) error, ttl, timeout time.Duration, now func() time.Time) *cachedHealthCheck {
	return &cachedHealthCheck{
		check:   check,
		ttl:     ttl,
		timeout: timeout,
		now:     now,
	}
}

func (c *cachedHealthCheck) Check(ctx context.Context) error {
	if c == nil || c.check == nil {
		return errors.New("health check is unavailable")
	}
	for {
		c.mu.Lock()
		if c.valid && c.now().Before(c.expires) {
			err := c.err
			c.mu.Unlock()
			return err
		}
		if wait := c.inFlight; wait != nil {
			c.mu.Unlock()
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-wait:
				continue
			}
		}
		c.inFlight = make(chan struct{})
		c.mu.Unlock()

		checkCtx, cancel := context.WithTimeout(context.Background(), c.timeout)
		err := c.check(checkCtx)
		cancel()

		c.mu.Lock()
		c.valid = true
		c.err = err
		c.expires = c.now().Add(c.ttl)
		close(c.inFlight)
		c.inFlight = nil
		c.mu.Unlock()
		return err
	}
}
