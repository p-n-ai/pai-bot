// Copyright 2026 the P&AI authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package terminalnudge

import (
	"context"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/p-n-ai/pai-bot/internal/chat"
)

// Triggerer checks a specific user for due-review nudges.
type Triggerer interface {
	CheckUserForNudge(ctx context.Context, userID string, now time.Time) error
}

// MessageSource returns captured outbound messages.
type MessageSource interface {
	Messages() []chat.OutboundMessage
}

// Config controls the nudge trigger run.
type Config struct {
	UserID string
	Now    func() time.Time
}

// Run triggers a nudge check for one user and prints any generated nudges.
func Run(ctx context.Context, out io.Writer, cfg Config, triggerer Triggerer, source MessageSource) error {
	if triggerer == nil {
		return fmt.Errorf("triggerer is required")
	}
	if source == nil {
		return fmt.Errorf("message source is required")
	}

	userID := strings.TrimSpace(cfg.UserID)
	if userID == "" {
		return fmt.Errorf("user_id is required")
	}

	nowFn := cfg.Now
	if nowFn == nil {
		nowFn = time.Now
	}

	before := len(source.Messages())
	if err := triggerer.CheckUserForNudge(ctx, userID, nowFn()); err != nil {
		return err
	}
	after := source.Messages()

	if _, err := fmt.Fprintf(out, "Triggered nudge check for %s\n", userID); err != nil {
		return err
	}

	if len(after) <= before {
		_, err := fmt.Fprintln(out, "No nudge was sent.")
		return err
	}

	for _, msg := range after[before:] {
		if _, err := fmt.Fprintf(out, "\nNudge:\n%s\n", strings.TrimSpace(msg.Text)); err != nil {
			return err
		}
	}

	return nil
}
