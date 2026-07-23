// Copyright 2026 the P&AI authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package server

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestFocusedPageCleanupWorkerDoesNotOverlapAndStopsWithContext(t *testing.T) {
	cleaner := &blockingFocusedPageCleaner{started: make(chan struct{}, 1)}
	worker, err := NewFocusedPageCleanupWorker(cleaner, slog.New(slog.NewTextHandler(&bytes.Buffer{}, nil)))
	if err != nil {
		t.Fatal(err)
	}
	ticks := make(chan time.Time, 3)
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		worker.run(ctx, ticks)
		close(done)
	}()

	ticks <- time.Now()
	select {
	case <-cleaner.started:
	case <-time.After(time.Second):
		t.Fatal("cleanup did not start")
	}
	ticks <- time.Now()
	ticks <- time.Now()
	if got := cleaner.calls.Load(); got != 1 {
		t.Fatalf("cleanup calls while first call is active = %d, want 1", got)
	}
	if got := cleaner.maxActive.Load(); got != 1 {
		t.Fatalf("maximum concurrent cleanup calls = %d, want 1", got)
	}

	cancel()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("worker did not stop with its context")
	}
	if got := cleaner.calls.Load(); got != 1 {
		t.Fatalf("cleanup calls after cancellation = %d, want 1", got)
	}
	if got := cleaner.active.Load(); got != 0 {
		t.Fatalf("active cleanup calls after shutdown = %d, want 0", got)
	}
}

func TestFocusedPageCleanupWorkerLogsSafeFailureState(t *testing.T) {
	var output lockedBuffer
	worker, err := NewFocusedPageCleanupWorker(
		failingFocusedPageCleaner{},
		slog.New(slog.NewJSONHandler(&output, nil)),
	)
	if err != nil {
		t.Fatal(err)
	}
	ticks := make(chan time.Time, 1)
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		worker.run(ctx, ticks)
		close(done)
	}()

	ticks <- time.Now()
	deadline := time.Now().Add(time.Second)
	for output.Len() == 0 && time.Now().Before(deadline) {
		time.Sleep(time.Millisecond)
	}
	cancel()
	<-done

	logged := output.String()
	if !strings.Contains(logged, `"failed":true`) {
		t.Fatalf("failure log = %q, want structured failure state", logged)
	}
	if strings.Contains(logged, "private page content") || strings.Contains(logged, "capability-fragment") {
		t.Fatalf("failure log exposed private error content: %q", logged)
	}
}

type blockingFocusedPageCleaner struct {
	calls     atomic.Int32
	active    atomic.Int32
	maxActive atomic.Int32
	started   chan struct{}
}

func (c *blockingFocusedPageCleaner) CleanupExpired(ctx context.Context, _ time.Time) (int64, error) {
	c.calls.Add(1)
	active := c.active.Add(1)
	for {
		current := c.maxActive.Load()
		if active <= current || c.maxActive.CompareAndSwap(current, active) {
			break
		}
	}
	c.started <- struct{}{}
	<-ctx.Done()
	c.active.Add(-1)
	return 0, ctx.Err()
}

type failingFocusedPageCleaner struct{}

func (failingFocusedPageCleaner) CleanupExpired(context.Context, time.Time) (int64, error) {
	return 0, errors.New("private page content capability-fragment")
}

type lockedBuffer struct {
	mu     sync.Mutex
	buffer bytes.Buffer
}

func (b *lockedBuffer) Write(data []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buffer.Write(data)
}

func (b *lockedBuffer) Len() int {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buffer.Len()
}

func (b *lockedBuffer) String() string {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buffer.String()
}
