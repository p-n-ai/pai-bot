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

func TestNormalizeImageInputAcceptsExactLimitDataURL(t *testing.T) {
	data := make([]byte, maxImageInputBytes)
	payload := base64.StdEncoding.EncodeToString(data)

	image, err := normalizeImageInput("data:image/png;base64," + payload)
	if err != nil {
		t.Fatalf("normalizeImageInput() error = %v, want exact-limit image accepted", err)
	}
	if len(image.Data) != maxImageInputBytes {
		t.Fatalf("len(image.Data) = %d, want %d", len(image.Data), maxImageInputBytes)
	}
}
