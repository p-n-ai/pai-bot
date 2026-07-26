// Copyright 2026 the P&AI authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package agent

import "testing"

func TestGroupMemberRecipientRequiresSavedRouteForThreadedChannels(t *testing.T) {
	withoutRoute := groupMemberRecipient(GroupMemberDelivery{
		ExternalID: "U123",
		Channel:    "slack",
	})
	if _, ok := withoutRoute.outbound("leaderboard", "Markdown"); ok {
		t.Fatal("Slack destination without a saved thread route should not be deliverable")
	}

	withRoute := groupMemberRecipient(GroupMemberDelivery{
		ExternalID: "U123",
		Channel:    "slack",
		ThreadID:   "slack:C456:1712345678.000100",
	})
	out, ok := withRoute.outbound("leaderboard", "Markdown")
	if !ok {
		t.Fatal("Slack destination with a saved thread route should be deliverable")
	}
	if out.ThreadID != "slack:C456:1712345678.000100" {
		t.Fatalf("ThreadID = %q, want saved route", out.ThreadID)
	}
}

func TestGroupMemberRecipientAllowsDirectChannelsWithoutThreadRoute(t *testing.T) {
	recipient := groupMemberRecipient(GroupMemberDelivery{
		ExternalID: "telegram-user",
		Channel:    "telegram",
	})
	if _, ok := recipient.outbound("leaderboard", "Markdown"); !ok {
		t.Fatal("Telegram direct destination should not require a saved thread route")
	}
}
