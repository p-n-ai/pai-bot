// Copyright 2026 the P&AI authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package server

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestCachedHealthCheckCachesResultUntilTTL(t *testing.T) {
	now := time.Unix(100, 0)
	var calls atomic.Int32
	checker := newCachedHealthCheck(func(context.Context) error {
		calls.Add(1)
		return nil
	}, time.Minute, time.Second, func() time.Time { return now })

	if err := checker.Check(context.Background()); err != nil {
		t.Fatalf("first check error = %v", err)
	}
	if err := checker.Check(context.Background()); err != nil {
		t.Fatalf("cached check error = %v", err)
	}
	if got := calls.Load(); got != 1 {
		t.Fatalf("calls within TTL = %d, want 1", got)
	}

	now = now.Add(time.Minute)
	if err := checker.Check(context.Background()); err != nil {
		t.Fatalf("expired check error = %v", err)
	}
	if got := calls.Load(); got != 2 {
		t.Fatalf("calls after TTL = %d, want 2", got)
	}
}

func TestCachedHealthCheckCoalescesConcurrentRequests(t *testing.T) {
	var calls atomic.Int32
	started := make(chan struct{})
	release := make(chan struct{})
	checker := NewCachedHealthCheck(func(context.Context) error {
		if calls.Add(1) == 1 {
			close(started)
		}
		<-release
		return nil
	}, time.Minute, time.Second)

	const requestCount = 8
	var wg sync.WaitGroup
	errs := make(chan error, requestCount)
	for range requestCount {
		wg.Add(1)
		go func() {
			defer wg.Done()
			errs <- checker(context.Background())
		}()
	}
	<-started
	close(release)
	wg.Wait()
	close(errs)
	for err := range errs {
		if err != nil {
			t.Fatalf("concurrent check error = %v", err)
		}
	}
	if got := calls.Load(); got != 1 {
		t.Fatalf("concurrent calls = %d, want 1", got)
	}
}

func TestCachedHealthCheckBoundsUpstreamTimeout(t *testing.T) {
	checker := NewCachedHealthCheck(func(ctx context.Context) error {
		<-ctx.Done()
		return ctx.Err()
	}, time.Minute, 10*time.Millisecond)

	err := checker(context.Background())
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("check error = %v, want deadline exceeded", err)
	}
}
