// Copyright 2026 the P&AI authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package chat

import "testing"

func TestRenderTurnPreservesThreadRoute(t *testing.T) {
	out, ok := RenderTurn(InboundMessage{
		Channel:  "slack",
		UserID:   "U123",
		ThreadID: "slack:C123:1725000000.000100",
	}, "Tutor reply", "", TelegramInlineKeyboardContext{})
	if !ok {
		t.Fatal("RenderTurn() dropped non-empty tutor reply")
	}
	if out.UserID != "U123" {
		t.Fatalf("UserID = %q, want learner identity U123", out.UserID)
	}
	if out.ThreadID != "slack:C123:1725000000.000100" {
		t.Fatalf("ThreadID = %q, want preserved Slack route", out.ThreadID)
	}
}
