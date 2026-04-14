// Copyright 2026 the P&AI authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package terminalnudge

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/p-n-ai/pai-bot/internal/chat"
)

type fakeTriggerer struct {
	err error
}

func (f fakeTriggerer) CheckUserForNudge(_ context.Context, _ string, _ time.Time) error {
	return f.err
}

type fakeMessageSource struct {
	reads   [][]chat.OutboundMessage
	readIdx int
}

func (f *fakeMessageSource) Messages() []chat.OutboundMessage {
	if len(f.reads) == 0 {
		return nil
	}
	idx := f.readIdx
	if idx >= len(f.reads) {
		idx = len(f.reads) - 1
	}
	f.readIdx++
	return append([]chat.OutboundMessage(nil), f.reads[idx]...)
}

func TestRun_PrintsNewNudges(t *testing.T) {
	var out strings.Builder
	err := Run(context.Background(), &out, Config{
		UserID: "student-1",
		Now: func() time.Time {
			return time.Date(2026, 3, 9, 10, 0, 0, 0, time.UTC)
		},
	}, fakeTriggerer{}, &fakeMessageSource{
		reads: [][]chat.OutboundMessage{
			nil,
			{
				{Text: "Nudge text 1"},
				{Text: "Nudge text 2"},
			},
		},
	})
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	rendered := out.String()
	if !strings.Contains(rendered, "Triggered nudge check for student-1") {
		t.Fatalf("output = %q, want trigger header", rendered)
	}
	if !strings.Contains(rendered, "Nudge text 1") || !strings.Contains(rendered, "Nudge text 2") {
		t.Fatalf("output = %q, want nudge texts", rendered)
	}
}

func TestRun_PrintsNoNudgeMessage(t *testing.T) {
	var out strings.Builder
	err := Run(context.Background(), &out, Config{
		UserID: "student-1",
		Now:    func() time.Time { return time.Now() },
	}, fakeTriggerer{}, &fakeMessageSource{})
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	if !strings.Contains(out.String(), "No nudge was sent.") {
		t.Fatalf("output = %q, want no nudge text", out.String())
	}
}

func TestRun_PropagatesTriggerError(t *testing.T) {
	err := Run(context.Background(), &strings.Builder{}, Config{
		UserID: "student-1",
		Now:    time.Now,
	}, fakeTriggerer{err: errors.New("boom")}, &fakeMessageSource{})
	if err == nil || !strings.Contains(err.Error(), "boom") {
		t.Fatalf("Run() error = %v, want boom", err)
	}
}
