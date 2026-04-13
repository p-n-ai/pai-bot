// Copyright 2026 the P&AI authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package curriculum

import (
	"testing"
)

func TestPrereqGraph_DependentsOf(t *testing.T) {
	topics := []Topic{
		{ID: "F1-05", Prerequisites: Prerequisites{}},
		{ID: "F1-06", Prerequisites: Prerequisites{Required: []string{"F1-05"}}},
		{ID: "F1-07", Prerequisites: Prerequisites{Required: []string{"F1-05", "F1-06"}}},
		{ID: "F1-08", Prerequisites: Prerequisites{Required: []string{"F1-07"}}},
	}

	g := NewPrereqGraph(topics)

	// F1-05 is required by F1-06 and F1-07
	deps := g.DependentsOf("F1-05")
	if len(deps) != 2 {
		t.Fatalf("DependentsOf(F1-05) = %v, want 2 items", deps)
	}
	has06, has07 := false, false
	for _, d := range deps {
		if d == "F1-06" {
			has06 = true
		}
		if d == "F1-07" {
			has07 = true
		}
	}
	if !has06 || !has07 {
		t.Errorf("DependentsOf(F1-05) = %v, want F1-06 and F1-07", deps)
	}

	// F1-07 is required by F1-08
	deps = g.DependentsOf("F1-07")
	if len(deps) != 1 || deps[0] != "F1-08" {
		t.Errorf("DependentsOf(F1-07) = %v, want [F1-08]", deps)
	}

	// F1-08 has no dependents
	deps = g.DependentsOf("F1-08")
	if len(deps) != 0 {
		t.Errorf("DependentsOf(F1-08) = %v, want []", deps)
	}
}

func TestPrereqGraph_RequiredPrereqs(t *testing.T) {
	topics := []Topic{
		{ID: "F1-05", Prerequisites: Prerequisites{}},
		{ID: "F1-06", Prerequisites: Prerequisites{Required: []string{"F1-05"}}},
		{ID: "F1-07", Prerequisites: Prerequisites{Required: []string{"F1-05", "F1-06"}}},
	}

	g := NewPrereqGraph(topics)

	prereqs := g.RequiredPrereqs("F1-07")
	if len(prereqs) != 2 {
		t.Fatalf("RequiredPrereqs(F1-07) = %v, want 2 items", prereqs)
	}

	prereqs = g.RequiredPrereqs("F1-05")
	if len(prereqs) != 0 {
		t.Errorf("RequiredPrereqs(F1-05) = %v, want []", prereqs)
	}
}

func TestPrereqGraph_UnlockableTopics(t *testing.T) {
	topics := []Topic{
		{ID: "F1-05", Name: "Algebra Basics", Prerequisites: Prerequisites{}},
		{ID: "F1-06", Name: "Linear Equations", Prerequisites: Prerequisites{Required: []string{"F1-05"}}},
		{ID: "F1-07", Name: "Inequalities", Prerequisites: Prerequisites{Required: []string{"F1-05", "F1-06"}}},
		{ID: "F1-08", Name: "Simultaneous Eq", Prerequisites: Prerequisites{Required: []string{"F1-07"}}},
	}

	g := NewPrereqGraph(topics)

	// masteryScores: F1-05 mastered, nothing else
	scores := map[string]float64{
		"F1-05": 0.85,
	}

	// Mastering F1-05 should unlock F1-06 (only prereq is F1-05)
	// but NOT F1-07 (needs F1-05 AND F1-06)
	unlocked := g.UnlockableTopics("F1-05", scores)
	if len(unlocked) != 1 {
		t.Fatalf("UnlockableTopics after F1-05 = %v, want 1", unlocked)
	}
	if unlocked[0].ID != "F1-06" {
		t.Errorf("expected F1-06, got %s", unlocked[0].ID)
	}

	// Now F1-06 is also mastered
	scores["F1-06"] = 0.82
	unlocked = g.UnlockableTopics("F1-06", scores)
	if len(unlocked) != 1 {
		t.Fatalf("UnlockableTopics after F1-06 = %v, want 1", unlocked)
	}
	if unlocked[0].ID != "F1-07" {
		t.Errorf("expected F1-07, got %s", unlocked[0].ID)
	}
}

func TestPrereqGraph_UnlockableTopics_NoPrereqs(t *testing.T) {
	topics := []Topic{
		{ID: "F1-05", Prerequisites: Prerequisites{}},
		{ID: "F1-06", Prerequisites: Prerequisites{}},
	}

	g := NewPrereqGraph(topics)

	// Topics with no prereqs are always unlocked, so mastering one
	// shouldn't trigger unlock notifications for the other.
	unlocked := g.UnlockableTopics("F1-05", map[string]float64{"F1-05": 0.9})
	if len(unlocked) != 0 {
		t.Errorf("expected no unlockable topics, got %v", unlocked)
	}
}

func TestPrereqGraph_UnlockableTopics_AlreadyMastered(t *testing.T) {
	topics := []Topic{
		{ID: "F1-05", Prerequisites: Prerequisites{}},
		{ID: "F1-06", Prerequisites: Prerequisites{Required: []string{"F1-05"}}},
	}

	g := NewPrereqGraph(topics)

	// F1-06 already mastered — don't re-notify
	scores := map[string]float64{
		"F1-05": 0.9,
		"F1-06": 0.85,
	}
	unlocked := g.UnlockableTopics("F1-05", scores)
	if len(unlocked) != 0 {
		t.Errorf("expected no unlockable topics (already mastered), got %v", unlocked)
	}
}
