// Copyright 2026 the P&AI authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package chat_test

import (
	"testing"

	"github.com/p-n-ai/pai-bot/internal/chat"
)

func TestAllCommandsReturnsCallerOwnedSnapshot(t *testing.T) {
	commands := chat.AllCommands(false)
	if len(commands) == 0 {
		t.Fatal("AllCommands(false) returned no commands")
	}

	original := commands[0]
	commands[0] = chat.BotCommand{Command: "changed", Description: "changed"}

	got := chat.AllCommands(false)[0]
	if got != original {
		t.Fatalf("AllCommands(false)[0] = %#v after caller mutation, want %#v", got, original)
	}
}
