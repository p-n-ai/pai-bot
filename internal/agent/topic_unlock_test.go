// Copyright 2026 the P&AI authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package agent

import (
	"strings"
	"testing"

	"github.com/p-n-ai/pai-bot/internal/curriculum"
	"github.com/p-n-ai/pai-bot/internal/i18n"
)

func TestPendingUnlocks_AddAndDrain(t *testing.T) {
	pu := newPendingUnlocks()

	topics := []curriculum.Topic{
		{ID: "F1-06", Name: "Linear Equations"},
	}

	pu.add("user1", topics)

	drained := pu.drain("user1")
	if len(drained) != 1 {
		t.Fatalf("expected 1 drained topic, got %d", len(drained))
	}
	if drained[0].ID != "F1-06" {
		t.Errorf("expected F1-06, got %s", drained[0].ID)
	}

	// Second drain should be empty.
	drained = pu.drain("user1")
	if len(drained) != 0 {
		t.Errorf("expected 0 drained topics after second drain, got %d", len(drained))
	}
}

func TestPendingUnlocks_AddEmpty(t *testing.T) {
	pu := newPendingUnlocks()
	pu.add("user1", nil)

	drained := pu.drain("user1")
	if len(drained) != 0 {
		t.Errorf("expected 0 drained topics, got %d", len(drained))
	}
}

func TestFormatUnlockNotification(t *testing.T) {
	topics := []curriculum.Topic{
		{ID: "F1-06", Name: "Persamaan Linear"},
		{ID: "F1-07", Name: "Ketaksamaan Linear"},
	}

	msg := formatUnlockNotification("ms", topics)
	if !strings.Contains(msg, "Persamaan Linear") {
		t.Errorf("expected topic name in notification, got: %s", msg)
	}
	if !strings.Contains(msg, "Ketaksamaan Linear") {
		t.Errorf("expected second topic name in notification, got: %s", msg)
	}
	if !strings.Contains(msg, "/learn") {
		t.Errorf("expected /learn hint in notification, got: %s", msg)
	}
}

func TestFormatUnlockNotification_Empty(t *testing.T) {
	msg := formatUnlockNotification("ms", nil)
	if msg != "" {
		t.Errorf("expected empty notification for nil topics, got: %s", msg)
	}
}

func TestFormatUnlockNotification_English(t *testing.T) {
	topics := []curriculum.Topic{
		{ID: "F1-06", Name: "Linear Equations"},
	}
	msg := formatUnlockNotification("en", topics)
	if !strings.Contains(msg, "Congratulations") {
		t.Errorf("expected English notification, got: %s", msg)
	}
}

func TestFormatUnlockNotification_Chinese(t *testing.T) {
	_ = i18n.S("zh", i18n.MsgTopicUnlocked, "test") // ensure key exists
	topics := []curriculum.Topic{
		{ID: "F1-06", Name: "线性方程"},
	}
	msg := formatUnlockNotification("zh", topics)
	if !strings.Contains(msg, "线性方程") {
		t.Errorf("expected Chinese topic name, got: %s", msg)
	}
}
