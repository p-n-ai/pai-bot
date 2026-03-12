package main

import "testing"

func TestSeedModeString(t *testing.T) {
	if demoSeedMode.String() != "demo" {
		t.Fatalf("demoSeedMode.String() = %q, want demo", demoSeedMode.String())
	}
}
