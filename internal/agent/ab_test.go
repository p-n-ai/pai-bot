// Copyright 2026 the P&AI authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package agent

import (
	"context"
	"strings"
	"testing"

	"github.com/p-n-ai/pai-bot/internal/chat"
)

func TestAssignABGroup_ReturnsValidGroup(t *testing.T) {
	for i := 0; i < 100; i++ {
		group := AssignABGroup()
		if group != ABGroupA && group != ABGroupB {
			t.Fatalf("AssignABGroup() = %q, want A or B", group)
		}
	}
}

func TestAssignABGroup_Distribution(t *testing.T) {
	counts := map[string]int{ABGroupA: 0, ABGroupB: 0}
	n := 10000
	for i := 0; i < n; i++ {
		counts[AssignABGroup()]++
	}
	ratioA := float64(counts[ABGroupA]) / float64(n)
	if ratioA < 0.45 || ratioA > 0.55 {
		t.Errorf("AB distribution skewed: A=%d B=%d (%.1f%% A)", counts[ABGroupA], counts[ABGroupB], ratioA*100)
	}
}

func TestMemoryStore_ABGroup(t *testing.T) {
	store := NewMemoryStore()
	group, ok := store.GetUserABGroup("user1")
	if ok || group != "" {
		t.Fatalf("GetUserABGroup() = %q, %v, want empty, false", group, ok)
	}
	if err := store.SetUserABGroup("user1", ABGroupA); err != nil {
		t.Fatal(err)
	}
	group, ok = store.GetUserABGroup("user1")
	if !ok || group != ABGroupA {
		t.Fatalf("GetUserABGroup() = %q, %v, want A, true", group, ok)
	}
	if err := store.SetUserABGroup("user1", ABGroupB); err != nil {
		t.Fatal(err)
	}
	group, ok = store.GetUserABGroup("user1")
	if !ok || group != ABGroupB {
		t.Fatalf("GetUserABGroup() = %q, %v, want B, true", group, ok)
	}
	if err := store.SetUserABGroup("user1", ""); err != nil {
		t.Fatal(err)
	}
	group, ok = store.GetUserABGroup("user1")
	if ok || group != "" {
		t.Fatalf("after clear: GetUserABGroup() = %q, %v, want empty, false", group, ok)
	}
	if err := store.SetUserABGroup("", ABGroupA); err == nil {
		t.Fatal("expected error for empty user_id")
	}
}

func TestDevAB_ShowCurrent(t *testing.T) {
	store := NewMemoryStore()
	engine := NewEngine(EngineConfig{
		Store:       store,
		EventLogger: NewMemoryEventLogger(),
		DevMode:     true,
	})

	msg := chat.InboundMessage{Channel: "telegram", UserID: "ab-user", Text: "/dev-ab"}
	resp, err := engine.ProcessMessage(context.Background(), msg)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(resp, "Current AB group") {
		t.Errorf("expected current group info, got: %s", resp)
	}
}

func TestDevAB_SetGroup(t *testing.T) {
	store := NewMemoryStore()
	engine := NewEngine(EngineConfig{
		Store:       store,
		EventLogger: NewMemoryEventLogger(),
		DevMode:     true,
	})

	msg := chat.InboundMessage{Channel: "telegram", UserID: "ab-user", Text: "/dev-ab B"}
	resp, err := engine.ProcessMessage(context.Background(), msg)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(resp, "AB group set to B") {
		t.Errorf("expected set confirmation, got: %s", resp)
	}

	group, ok := store.GetUserABGroup("ab-user")
	if !ok || group != ABGroupB {
		t.Fatalf("GetUserABGroup() = %q, %v, want B, true", group, ok)
	}
}

func TestDevAB_HiddenWithoutDevMode(t *testing.T) {
	store := NewMemoryStore()
	engine := NewEngine(EngineConfig{
		Store:       store,
		EventLogger: NewMemoryEventLogger(),
		DevMode:     false,
	})

	msg := chat.InboundMessage{Channel: "telegram", UserID: "ab-user", Text: "/dev-ab A"}
	resp, err := engine.ProcessMessage(context.Background(), msg)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(resp, "tidak diketahui") && !strings.Contains(resp, "Unknown") {
		t.Errorf("expected unknown command, got: %s", resp)
	}
}
