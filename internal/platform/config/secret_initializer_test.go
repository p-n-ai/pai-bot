// Copyright 2026 the P&AI authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestWriteNewSecretEnvFileCreatesIndependentPrivateRoots(t *testing.T) {
	path := filepath.Join(t.TempDir(), "secrets.env")
	if err := WriteNewSecretEnvFile(path); err != nil {
		t.Fatalf("WriteNewSecretEnvFile() error = %v", err)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat secret env file: %v", err)
	}
	if got := info.Mode().Perm(); got != 0o600 {
		t.Fatalf("secret env permissions = %o, want 600", got)
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read secret env file: %v", err)
	}
	lines := strings.Split(strings.TrimSpace(string(raw)), "\n")
	if len(lines) != 2 {
		t.Fatalf("secret env lines = %d, want 2", len(lines))
	}
	auth := strings.TrimPrefix(lines[0], "PAI_AUTH_SECRET=")
	encryption := strings.TrimPrefix(lines[1], "PAI_CONFIG_ENCRYPTION_KEY=")
	if auth == lines[0] || encryption == lines[1] || len(auth) < 43 || len(encryption) < 43 {
		t.Fatal("secret env file does not contain two full random roots")
	}
	if auth == encryption {
		t.Fatal("auth and configuration encryption roots must be independent")
	}
}

func TestWriteNewSecretEnvFileRefusesOverwrite(t *testing.T) {
	path := filepath.Join(t.TempDir(), "secrets.env")
	if err := os.WriteFile(path, []byte("keep-me\n"), 0o600); err != nil {
		t.Fatalf("seed existing file: %v", err)
	}
	if err := WriteNewSecretEnvFile(path); err == nil {
		t.Fatal("WriteNewSecretEnvFile() error = nil, want overwrite refusal")
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read existing file: %v", err)
	}
	if string(raw) != "keep-me\n" {
		t.Fatalf("existing file changed to %q", raw)
	}
}
