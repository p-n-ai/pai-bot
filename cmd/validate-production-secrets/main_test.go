// Copyright 2026 the P&AI authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"bytes"
	"errors"
	"strings"
	"testing"
)

type failingWriter struct{}

func (failingWriter) Write([]byte) (int, error) {
	return 0, errors.New("write failed")
}

func TestRunAcceptsValidProductionSecrets(t *testing.T) {
	setProductionSecretEnvironment(t)
	var stdout, stderr bytes.Buffer

	if code := run(&stdout, &stderr); code != 0 {
		t.Fatalf("run() code = %d, want 0 (stderr %q)", code, stderr.String())
	}
	if stdout.String() != "Production secrets are valid\n" || stderr.Len() != 0 {
		t.Fatalf("stdout/stderr = %q/%q", stdout.String(), stderr.String())
	}
}

func TestRunRejectsInvalidProductionSecrets(t *testing.T) {
	setProductionSecretEnvironment(t)
	t.Setenv("PAI_CONFIG_ENCRYPTION_KEY", "")
	var stdout, stderr bytes.Buffer

	if code := run(&stdout, &stderr); code != 1 {
		t.Fatalf("run() code = %d, want 1", code)
	}
	if stdout.Len() != 0 || !strings.Contains(stderr.String(), "PAI_CONFIG_ENCRYPTION_KEY") {
		t.Fatalf("stdout/stderr = %q/%q, want safe validation error", stdout.String(), stderr.String())
	}
}

func TestRunFailsWhenSuccessOutputCannotBeWritten(t *testing.T) {
	setProductionSecretEnvironment(t)
	var stderr bytes.Buffer

	if code := run(failingWriter{}, &stderr); code != 1 {
		t.Fatalf("run() code = %d, want 1", code)
	}
	if stderr.Len() != 0 {
		t.Fatalf("stderr = %q, want empty", stderr.String())
	}
}

func setProductionSecretEnvironment(t *testing.T) {
	t.Helper()
	t.Setenv("PAI_AUTH_SECRET", "auth-secret-value-with-enough-variety")
	t.Setenv("PAI_CONFIG_ENCRYPTION_KEY", "active-settings-encryption-key-1234")
	t.Setenv("PAI_CONFIG_PREVIOUS_ENCRYPTION_KEYS", "[]")
	t.Setenv("PAI_AUTH_BOOTSTRAP_ADMIN_PASSWORD", "private-bootstrap-password")
}
