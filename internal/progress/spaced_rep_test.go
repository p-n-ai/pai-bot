// Copyright 2026 the P&AI authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package progress

import (
	"testing"
)

func TestSM2_Calculate(t *testing.T) {
	tests := []struct {
		name           string
		quality        int // 0-5 response quality
		repetitions    int
		easeFactor     float64
		interval       int
		wantRepGrow    bool
		wantIntervalUp bool
	}{
		{"perfect-first", 5, 0, 2.5, 1, true, true},
		{"good-second", 4, 1, 2.5, 1, true, true},
		{"fail-reset", 1, 5, 2.5, 10, false, false},
		{"barely-pass", 3, 2, 2.5, 6, true, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := SM2Calculate(tt.quality, tt.repetitions, tt.easeFactor, tt.interval)

			if tt.wantRepGrow && result.Repetitions <= tt.repetitions {
				t.Errorf("Repetitions should grow: got %d, was %d", result.Repetitions, tt.repetitions)
			}
			if !tt.wantRepGrow && result.Repetitions != 0 {
				t.Errorf("Repetitions should reset to 0: got %d", result.Repetitions)
			}
			if result.EaseFactor < 1.3 {
				t.Errorf("EaseFactor should not go below 1.3: got %f", result.EaseFactor)
			}
		})
	}
}

func TestSM2_EaseFactorFloor(t *testing.T) {
	// Repeated failures should not push EF below 1.3.
	ef := 1.4
	for i := 0; i < 10; i++ {
		result := SM2Calculate(0, 0, ef, 1)
		ef = result.EaseFactor
	}
	if ef < 1.3 {
		t.Errorf("EaseFactor went below floor 1.3: got %f", ef)
	}
}

func TestSM2_IntervalProgression(t *testing.T) {
	// rep 0 → interval 1, rep 1 → interval 6, rep 2+ → EF-multiplied.
	r := SM2Calculate(5, 0, 2.5, 1)
	if r.IntervalDays != 1 {
		t.Errorf("rep 0 should give interval 1, got %d", r.IntervalDays)
	}

	r = SM2Calculate(5, 1, 2.5, 1)
	if r.IntervalDays != 6 {
		t.Errorf("rep 1 should give interval 6, got %d", r.IntervalDays)
	}

	r = SM2Calculate(5, 2, 2.5, 6)
	// Expected: round(6 * 2.6) = round(15.6) = 16 (EF increases with quality=5)
	if r.IntervalDays <= 6 {
		t.Errorf("rep 2+ should multiply interval by EF, got %d", r.IntervalDays)
	}
}

func TestSM2_FailureResetsInterval(t *testing.T) {
	r := SM2Calculate(2, 5, 2.5, 30)
	if r.IntervalDays != 1 {
		t.Errorf("failure should reset interval to 1, got %d", r.IntervalDays)
	}
	if r.Repetitions != 0 {
		t.Errorf("failure should reset repetitions to 0, got %d", r.Repetitions)
	}
}
