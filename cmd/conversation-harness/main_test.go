// Copyright 2026 the P&AI authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"encoding/json"
	"os"
	"reflect"
	"testing"

	"github.com/p-n-ai/pai-bot/internal/ai"
)

func TestBuildEngineLeavesProgressOffByDefault(t *testing.T) {
	engine, cleanup, err := buildEngine(true, "mock tutor response", false, nil)
	if err != nil {
		t.Fatalf("buildEngine() error = %v", err)
	}
	defer cleanup()

	if !engineTrackerIsNil(engine) {
		t.Fatal("conversation harness should not enable mastery tracker by default")
	}
}

func TestBuildEngineCanEnableProgress(t *testing.T) {
	engine, cleanup, err := buildEngine(true, "mock tutor response", true, nil)
	if err != nil {
		t.Fatalf("buildEngine() error = %v", err)
	}
	defer cleanup()

	if engineTrackerIsNil(engine) {
		t.Fatal("conversation harness should enable mastery tracker when --progress is set")
	}
}

func TestRequestDumperWritesCompletionRequestJSONL(t *testing.T) {
	path := t.TempDir() + "/requests.jsonl"
	dumper, err := newRequestDumper(path)
	if err != nil {
		t.Fatalf("newRequestDumper() error = %v", err)
	}
	dumper.Record(ai.CompletionTrace{
		Provider: "mock",
		Request: ai.CompletionRequest{
			Messages:  []ai.Message{{Role: "user", Content: "hi"}},
			Task:      ai.TaskTeaching,
			MaxTokens: 1024,
		},
	})
	if err := dumper.Close(); err != nil {
		t.Fatalf("Close() error = %v", err)
	}

	var record requestDumpRecord
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	if err := json.Unmarshal(b, &record); err != nil {
		t.Fatalf("dumped request is not JSON: %v\n%s", err, b)
	}
	if record.Sequence != 1 {
		t.Fatalf("Sequence = %d, want 1", record.Sequence)
	}
	if record.Provider != "mock" {
		t.Fatalf("Provider = %q, want mock", record.Provider)
	}
	if got := record.Request.Messages[0].Content; got != "hi" {
		t.Fatalf("message content = %q, want hi", got)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("Stat() error = %v", err)
	}
	if got := info.Mode().Perm(); got != 0o600 {
		t.Fatalf("dump permissions = %v, want 0600", got)
	}
}

func TestRequestDumperTightensExistingFilePermissions(t *testing.T) {
	path := t.TempDir() + "/requests.jsonl"
	if err := os.WriteFile(path, []byte("old\n"), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}
	dumper, err := newRequestDumper(path)
	if err != nil {
		t.Fatalf("newRequestDumper() error = %v", err)
	}
	if err := dumper.Close(); err != nil {
		t.Fatalf("Close() error = %v", err)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("Stat() error = %v", err)
	}
	if got := info.Mode().Perm(); got != 0o600 {
		t.Fatalf("dump permissions = %v, want 0600", got)
	}
}

func TestValidateRequestOnlyModeRequiresDumpRequests(t *testing.T) {
	if err := validateRequestOnlyMode(true, ""); err == nil {
		t.Fatal("validateRequestOnlyMode() should reject request-only without dump path")
	}
	if err := validateRequestOnlyMode(true, "requests.jsonl"); err != nil {
		t.Fatalf("validateRequestOnlyMode() error = %v", err)
	}
	if err := validateRequestOnlyMode(false, ""); err != nil {
		t.Fatalf("validateRequestOnlyMode() error = %v", err)
	}
}

func engineTrackerIsNil(engine any) bool {
	field := reflect.ValueOf(engine).Elem().FieldByName("tracker")
	return field.IsNil()
}
