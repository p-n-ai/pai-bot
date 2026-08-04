// Copyright 2026 the P&AI authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package ai

import (
	"encoding/base64"
	"strings"
	"testing"
)

func TestNormalizeImageInputRejectsOversizedDataURL(t *testing.T) {
	payload := strings.Repeat("A", base64.StdEncoding.EncodedLen(maxImageInputBytes+1))
	_, err := normalizeImageInput("data:image/png;base64," + payload)
	if err == nil || !strings.Contains(err.Error(), "exceeds") {
		t.Fatalf("normalizeImageInput() error = %v, want oversized image rejection", err)
	}
}
