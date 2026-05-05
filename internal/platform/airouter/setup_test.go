// Copyright 2026 the P&AI authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package airouter

import "testing"

func TestProviderOrderSkipsMockByDefault(t *testing.T) {
	for _, provider := range providerOrder("") {
		if provider == "mock" {
			t.Fatal("mock provider should require explicit LEARN_AI_DEFAULT_PROVIDER=mock")
		}
	}
}

func TestProviderOrderAllowsExplicitMock(t *testing.T) {
	order := providerOrder("mock")
	if len(order) == 0 || order[0] != "mock" {
		t.Fatalf("providerOrder(mock) = %#v, want mock first", order)
	}
}
