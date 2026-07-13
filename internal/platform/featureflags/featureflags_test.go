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
	if features.Enabled(AgentCore) {
		t.Fatal("agent_core should default to disabled")
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

func TestParseAgentCoreFeature(t *testing.T) {
	features, err := Parse("agent_core")
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	if !features.Enabled(AgentCore) {
		t.Fatal("agent_core should be enabled")
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

func TestWithOverrides(t *testing.T) {
	base, err := Parse("")
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}

	on, err := base.WithOverrides(map[string]bool{"turn_hooks": true})
	if err != nil {
		t.Fatalf("WithOverrides() error = %v", err)
	}
	if !on.Enabled(TurnHooks) {
		t.Fatal("turn_hooks should be enabled after override")
	}
	if base.Enabled(TurnHooks) {
		t.Fatal("WithOverrides must not mutate the receiver")
	}

	off, err := on.WithOverrides(map[string]bool{"turn_hooks": false})
	if err != nil {
		t.Fatalf("WithOverrides(false) error = %v", err)
	}
	if off.Enabled(TurnHooks) {
		t.Fatal("turn_hooks should be disabled after override")
	}

	if _, err := base.WithOverrides(map[string]bool{"unknown_flag": true}); err == nil {
		t.Fatal("WithOverrides should reject unknown feature flag")
	}
}

func TestParseInvalidBool(t *testing.T) {
	if _, err := Parse("turn_hooks=maybe"); err == nil {
		t.Fatal("Parse() should reject invalid bool override")
	}
}

func TestDefaults(t *testing.T) {
	defaults := Defaults()
	if enabled, ok := defaults["turn_hooks"]; !ok || enabled {
		t.Fatalf("Defaults()[turn_hooks] = %v, %v; want false, present", enabled, ok)
	}
	if enabled, ok := defaults["agent_core"]; !ok || enabled {
		t.Fatalf("Defaults()[agent_core] = %v, %v; want false, present", enabled, ok)
	}
}
