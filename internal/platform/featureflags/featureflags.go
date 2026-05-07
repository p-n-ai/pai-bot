// Copyright 2026 the P&AI authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

// Package featureflags owns deploy-time product and internal rollout gates.
package featureflags

import (
	"fmt"
	"strconv"
	"strings"
)

// Feature names a known deploy-time feature gate.
type Feature string

// Status describes feature maturity, not whether the feature is enabled.
type Status string

const (
	InDevelopment Status = "in_development"
	Stable        Status = "stable"
)

const (
	// TurnHooks enables the internal Tutor Turn hook seam.
	TurnHooks Feature = "turn_hooks"
)

// Spec describes a known feature flag.
type Spec struct {
	Feature        Feature
	Status         Status
	DefaultEnabled bool
}

// Features is the effective feature set for this process.
type Features struct {
	enabled map[Feature]struct{}
}

var registry = map[Feature]Spec{
	TurnHooks: {
		Feature:        TurnHooks,
		Status:         InDevelopment,
		DefaultEnabled: false,
	},
}

// Parse builds an effective feature set from comma-separated overrides.
func Parse(value string) (Features, error) {
	features := withDefaults()
	seen := map[Feature]struct{}{}

	for _, part := range strings.Split(value, ",") {
		name, enabled, err := parseOverride(part)
		if err != nil {
			return Features{}, err
		}
		if name == "" {
			continue
		}
		feature := Feature(name)
		if _, ok := registry[feature]; !ok {
			return Features{}, fmt.Errorf("unknown feature flag %q", name)
		}
		if _, ok := seen[feature]; ok {
			return Features{}, fmt.Errorf("duplicate feature flag override %q", name)
		}
		seen[feature] = struct{}{}
		features.set(feature, enabled)
	}

	return features, nil
}

func withDefaults() Features {
	features := Features{enabled: map[Feature]struct{}{}}
	for feature, spec := range registry {
		if spec.DefaultEnabled {
			features.enabled[feature] = struct{}{}
		}
	}
	return features
}

func parseOverride(value string) (string, bool, error) {
	raw := strings.TrimSpace(value)
	if raw == "" {
		return "", false, nil
	}

	name, rawEnabled, ok := strings.Cut(raw, "=")
	name = strings.TrimSpace(name)
	if name == "" {
		return "", false, fmt.Errorf("feature flag name is required")
	}
	if !ok {
		return name, true, nil
	}

	enabled, err := strconv.ParseBool(strings.TrimSpace(rawEnabled))
	if err != nil {
		return "", false, fmt.Errorf("invalid value for feature flag %q: %q", name, rawEnabled)
	}
	return name, enabled, nil
}

func (f Features) set(feature Feature, enabled bool) {
	if enabled {
		f.enabled[feature] = struct{}{}
		return
	}
	delete(f.enabled, feature)
}

// Enabled reports whether feature is active.
func (f Features) Enabled(feature Feature) bool {
	_, ok := f.enabled[feature]
	return ok
}
