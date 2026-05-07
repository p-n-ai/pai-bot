// Copyright 2026 the P&AI authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package featureflags

import "testing"

func TestParseEmptyFeatureSet(t *testing.T) {
	features, err := Parse(" , ")
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	if features.Enabled(Feature("missing")) {
		t.Fatal("missing feature should not be enabled")
	}
}

func TestParseUnknownFeature(t *testing.T) {
	if _, err := Parse("unknown_feature"); err == nil {
		t.Fatal("Parse() should reject unknown feature flag")
	}
}

func TestParseInvalidBool(t *testing.T) {
	if _, err := Parse("unknown_feature=maybe"); err == nil {
		t.Fatal("Parse() should reject invalid bool override")
	}
}
