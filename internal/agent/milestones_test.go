// Copyright 2026 the P&AI authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package agent

import (
	"strings"
	"testing"
)

func TestXPMilestones(t *testing.T) {
	tests := []struct {
		name    string
		before  int
		after   int
		wantHit bool
		wantAt  int
	}{
		{"no crossing", 90, 95, false, 0},
		{"cross 100", 90, 110, true, 100},
		{"cross 500", 480, 520, true, 500},
		{"cross 1000", 990, 1010, true, 1000},
		{"cross 2500", 2400, 2600, true, 2500},
		{"cross 5000", 4900, 5100, true, 5000},
		{"exact boundary", 99, 100, true, 100},
		{"already past", 110, 120, false, 0},
		{"cross multiple returns highest", 90, 550, true, 500},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hit, at := CheckXPMilestone(tt.before, tt.after)
			if hit != tt.wantHit {
				t.Errorf("CheckXPMilestone(%d, %d) hit = %v, want %v", tt.before, tt.after, hit, tt.wantHit)
			}
			if at != tt.wantAt {
				t.Errorf("CheckXPMilestone(%d, %d) at = %d, want %d", tt.before, tt.after, at, tt.wantAt)
			}
		})
	}
}

func TestFormatTopicMasteredCelebration(t *testing.T) {
	msg := FormatTopicMasteredCelebration("en", "Linear Equations", 50)
	if !strings.Contains(msg, "Linear Equations") {
		t.Errorf("expected topic name in message, got: %s", msg)
	}
	if !strings.Contains(msg, "50") {
		t.Errorf("expected XP amount in message, got: %s", msg)
	}
	if !strings.Contains(msg, "🏆") {
		t.Errorf("expected trophy emoji in message, got: %s", msg)
	}
}

func TestFormatXPMilestoneCelebration(t *testing.T) {
	msg := FormatXPMilestoneCelebration("ms", 1000)
	if !strings.Contains(msg, "1000") {
		t.Errorf("expected XP amount in message, got: %s", msg)
	}
	if !strings.Contains(msg, "🌟") {
		t.Errorf("expected star emoji in message, got: %s", msg)
	}
}

func TestFormatSubjectCompleteCelebration(t *testing.T) {
	msg := FormatSubjectCompleteCelebration("zh", "KSSM Matematik Tingkatan 1")
	if !strings.Contains(msg, "KSSM Matematik Tingkatan 1") {
		t.Errorf("expected subject name in message, got: %s", msg)
	}
	if !strings.Contains(msg, "🎓") {
		t.Errorf("expected graduation emoji in message, got: %s", msg)
	}
}

func TestFormatStreakRecordCelebration(t *testing.T) {
	msg := FormatStreakRecordCelebration("en", 14)
	if !strings.Contains(msg, "14") {
		t.Errorf("expected streak days in message, got: %s", msg)
	}
	if !strings.Contains(msg, "🔥") {
		t.Errorf("expected fire emoji in message, got: %s", msg)
	}
}

func TestPendingMilestones(t *testing.T) {
	pm := newPendingMilestones()
	pm.add("user1", "🏆 You mastered topic X!")
	pm.add("user1", "🌟 You reached 500 XP!")
	msgs := pm.drain("user1")
	if len(msgs) != 2 {
		t.Fatalf("drain() = %d messages, want 2", len(msgs))
	}
	msgs = pm.drain("user1")
	if len(msgs) != 0 {
		t.Fatalf("second drain() = %d messages, want 0", len(msgs))
	}
	msgs = pm.drain("nobody")
	if len(msgs) != 0 {
		t.Fatalf("drain(nobody) = %d messages, want 0", len(msgs))
	}
}
