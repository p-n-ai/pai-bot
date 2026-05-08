// Copyright 2026 the P&AI authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"os"
	"testing"
)

func TestWriteConversationHistoryTightensExistingFilePermissions(t *testing.T) {
	path := t.TempDir() + "/history.json"
	if err := os.WriteFile(path, []byte("old\n"), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	history := newConversationHistory("u1", "terminal")
	history.Turns = append(history.Turns, conversationTurnJSON{
		UserID:  "u1",
		Channel: "terminal",
		Role:    "student",
		Text:    "hi",
	})
	if err := writeConversationHistory(path, history, 0); err != nil {
		t.Fatalf("writeConversationHistory() error = %v", err)
	}

	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("Stat() error = %v", err)
	}
	if got := info.Mode().Perm(); got != 0o600 {
		t.Fatalf("history permissions = %v, want 0600", got)
	}
}
