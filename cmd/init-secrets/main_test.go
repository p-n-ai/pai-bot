// Copyright 2026 the P&AI authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRunRequiresOutputPath(t *testing.T) {
	var stdout, stderr bytes.Buffer

	if code := run(nil, &stdout, &stderr); code != 2 {
		t.Fatalf("run() code = %d, want 2", code)
	}
	if stdout.Len() != 0 || !strings.Contains(stderr.String(), "-out is required") {
		t.Fatalf("stdout/stderr = %q/%q, want required flag error", stdout.String(), stderr.String())
	}
}

func TestRunCreatesPrivateSecretFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "production-secrets.env")
	var stdout, stderr bytes.Buffer

	if code := run([]string{"-out", path}, &stdout, &stderr); code != 0 {
		t.Fatalf("run() code = %d, want 0 (stderr %q)", code, stderr.String())
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat generated secret file: %v", err)
	}
	if info.Mode().Perm() != 0o600 {
		t.Fatalf("generated secret file permissions = %o, want 600", info.Mode().Perm())
	}
	if !strings.Contains(stdout.String(), path) || stderr.Len() != 0 {
		t.Fatalf("stdout/stderr = %q/%q, want created path and no error", stdout.String(), stderr.String())
	}
}

func TestRunRefusesExistingOutputPath(t *testing.T) {
	path := filepath.Join(t.TempDir(), "production-secrets.env")
	if err := os.WriteFile(path, []byte("keep-me\n"), 0o600); err != nil {
		t.Fatalf("seed existing file: %v", err)
	}
	var stdout, stderr bytes.Buffer

	if code := run([]string{"-out", path}, &stdout, &stderr); code != 1 {
		t.Fatalf("run() code = %d, want 1", code)
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read existing file: %v", err)
	}
	if string(raw) != "keep-me\n" || stdout.Len() != 0 || stderr.Len() == 0 {
		t.Fatalf("file/stdout/stderr = %q/%q/%q, want preserved file and safe error", raw, stdout.String(), stderr.String())
	}
}
