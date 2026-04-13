// Copyright 2026 the P&AI authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package agent_test

import (
	"testing"

	"github.com/p-n-ai/pai-bot/internal/agent"
	"github.com/p-n-ai/pai-bot/internal/curriculum"
)

func TestDetectTopic(t *testing.T) {
	topics := []curriculum.Topic{
		{
			ID:   "F1-01",
			Name: "Variables and Algebraic Expressions",
			LearningObjectives: []curriculum.LearningObjective{
				{ID: "LO1", Text: "Identify variables and constants"},
			},
		},
		{
			ID:   "F1-02",
			Name: "Linear Equations",
			LearningObjectives: []curriculum.LearningObjective{
				{ID: "LO1", Text: "Solve linear equations in one variable"},
			},
		},
	}

	tests := []struct {
		name   string
		text   string
		wantID string
		wantOK bool
	}{
		{
			name:   "matches topic name keywords",
			text:   "Can you teach me linear equations?",
			wantID: "F1-02",
			wantOK: true,
		},
		{
			name:   "matches learning objective keywords",
			text:   "I need help to identify variables in algebra",
			wantID: "F1-01",
			wantOK: true,
		},
		{
			name:   "no topic match",
			text:   "hello what is your name",
			wantID: "",
			wantOK: false,
		},
		{
			name:   "does not match topic word inside larger token",
			text:   "Can you explain nonlinear patterns?",
			wantID: "",
			wantOK: false,
		},
		{
			name:   "does not match learning objective word inside larger token",
			text:   "I am confused about invariable costs",
			wantID: "",
			wantOK: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotID, gotOK := agent.DetectTopic(tt.text, topics)
			if gotOK != tt.wantOK {
				t.Fatalf("DetectTopic() ok = %v, want %v", gotOK, tt.wantOK)
			}
			if gotID != tt.wantID {
				t.Fatalf("DetectTopic() id = %q, want %q", gotID, tt.wantID)
			}
		})
	}
}
