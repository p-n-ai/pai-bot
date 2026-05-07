// Copyright 2026 the P&AI authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package featureflags

import "testing"

func TestParseEmptyFeatureSet(t *testing.T) {
	features, err := Parse(" , ")
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	if features.Enabled(TurnHooks) {
		t.Fatal("turn_hooks should default to disabled")
	}
	if features.Enabled(Feature("missing")) {
		t.Fatal("missing feature should not be enabled")
	}
}

func TestParseTurnHooksFeature(t *testing.T) {
	tests := []struct {
		name    string
		value   string
		enabled bool
	}{
		{name: "bare name enables", value: "turn_hooks", enabled: true},
		{name: "explicit true enables", value: "turn_hooks=true", enabled: true},
		{name: "explicit false disables", value: "turn_hooks=false", enabled: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			features, err := Parse(tt.value)
			if err != nil {
				t.Fatalf("Parse() error = %v", err)
			}
			if got := features.Enabled(TurnHooks); got != tt.enabled {
				t.Fatalf("turn_hooks enabled = %v, want %v", got, tt.enabled)
			}
		})
	}
}

func TestParseUnknownFeature(t *testing.T) {
	if _, err := Parse("unknown_feature"); err == nil {
		t.Fatal("Parse() should reject unknown feature flag")
	}
}

func TestParseDuplicateFeature(t *testing.T) {
	if _, err := Parse("turn_hooks,turn_hooks=false"); err == nil {
		t.Fatal("Parse() should reject duplicate feature flag override")
	}
}

func TestParseInvalidBool(t *testing.T) {
	if _, err := Parse("turn_hooks=maybe"); err == nil {
		t.Fatal("Parse() should reject invalid bool override")
	}
}
