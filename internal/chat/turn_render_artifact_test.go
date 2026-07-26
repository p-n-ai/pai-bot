// Copyright 2026 the P&AI authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package chat

import "testing"

func TestRenderTurnPreservesFocusedPageForChannelDelivery(t *testing.T) {
	pageURL := "https://pages.example/a/page-1#private-capability"
	out, ok := RenderTurn(
		InboundMessage{Channel: "websocket", UserID: "learner-1"},
		"Your report is ready.",
		pageURL,
		TelegramInlineKeyboardContext{},
	)
	if !ok {
		t.Fatal("RenderTurn() dropped a non-empty tutor response")
	}
	if out.Text != "Your report is ready." || out.FocusedPageURL != pageURL {
		t.Fatalf("text = %q, focused page = %q", out.Text, out.FocusedPageURL)
	}
}

func TestRenderTurnPlainTextHasNoFocusedPage(t *testing.T) {
	out, ok := RenderTurn(
		InboundMessage{Channel: "websocket", UserID: "learner-1"},
		"Plain tutor reply",
		"",
		TelegramInlineKeyboardContext{},
	)
	if !ok {
		t.Fatal("RenderTurn() dropped a non-empty tutor response")
	}
	if out.Text != "Plain tutor reply" || out.FocusedPageURL != "" {
		t.Fatalf("text = %q, focused page = %q", out.Text, out.FocusedPageURL)
	}
}

func TestRenderTurnUsesNormalizedFocusedPageForTelegramButton(t *testing.T) {
	const pageURL = "https://pages.example/a/page-1#private-capability"
	out, ok := RenderTurn(
		InboundMessage{Channel: "telegram", UserID: "learner-1"},
		"Your report is ready.",
		"  "+pageURL+"\n",
		TelegramInlineKeyboardContext{},
	)
	if !ok {
		t.Fatal("RenderTurn() dropped a non-empty tutor response")
	}
	if out.FocusedPageURL != pageURL {
		t.Fatalf("FocusedPageURL = %q, want %q", out.FocusedPageURL, pageURL)
	}
	if len(out.InlineKeyboard) != 1 || len(out.InlineKeyboard[0]) != 1 {
		t.Fatalf("InlineKeyboard = %#v, want one focused-page button", out.InlineKeyboard)
	}
	if got := out.InlineKeyboard[0][0].URL; got != pageURL {
		t.Fatalf("focused-page button URL = %q, want %q", got, pageURL)
	}
}
