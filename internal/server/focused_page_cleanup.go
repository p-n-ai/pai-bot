// Copyright 2026 the P&AI authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package server

import (
	"context"
	"fmt"
	"log/slog"
	"time"
)

const FocusedPageCleanupInterval = 15 * time.Minute

type focusedPageCleaner interface {
	CleanupExpired(context.Context, time.Time) (int64, error)
}

type FocusedPageCleanupWorker struct {
	cleaner focusedPageCleaner
	logger  *slog.Logger
	now     func() time.Time
}

func NewFocusedPageCleanupWorker(cleaner focusedPageCleaner, logger *slog.Logger) (*FocusedPageCleanupWorker, error) {
	if cleaner == nil {
		return nil, fmt.Errorf("focused page cleaner is required")
	}
	if logger == nil {
		logger = slog.Default()
	}
	return &FocusedPageCleanupWorker{cleaner: cleaner, logger: logger, now: time.Now}, nil
}

func (w *FocusedPageCleanupWorker) Run(ctx context.Context) {
	ticker := time.NewTicker(FocusedPageCleanupInterval)
	defer ticker.Stop()
	w.run(ctx, ticker.C)
}

func (w *FocusedPageCleanupWorker) run(ctx context.Context, ticks <-chan time.Time) {
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticks:
			deleted, err := w.cleaner.CleanupExpired(ctx, w.now().UTC())
			if ctx.Err() != nil {
				return
			}
			if err != nil {
				w.logger.Warn("focused page cleanup failed", "deleted", 0, "failed", true)
				continue
			}
			w.logger.Info("focused page cleanup completed", "deleted", deleted, "failed", false)
		}
	}
}
