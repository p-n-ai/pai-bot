// Copyright 2026 the P&AI authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"context"
	"encoding/json"
	"os"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/p-n-ai/pai-bot/internal/ai"
	"github.com/p-n-ai/pai-bot/internal/chat"
	"github.com/p-n-ai/pai-bot/internal/conversationharness"
	"github.com/p-n-ai/pai-bot/internal/retrieval"
)

func TestBuildEngineLeavesProgressOffByDefault(t *testing.T) {
	engine, cleanup, err := buildEngine(true, "mock tutor response", false, nil, nil)
	if err != nil {
		t.Fatalf("buildEngine() error = %v", err)
	}
	defer cleanup()

	if !engineTrackerIsNil(engine) {
		t.Fatal("conversation harness should not enable mastery tracker by default")
	}
}

func TestBuildEngineCanEnableProgress(t *testing.T) {
	engine, cleanup, err := buildEngine(true, "mock tutor response", true, nil, nil)
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

func TestHarnessEvidenceRetrieverIsolatesIdenticalPromptsByCase(t *testing.T) {
	retriever := newHarnessEvidenceRetriever([]conversationSpec{
		{
			ID: "CASE-A",
			Evidence: []evidenceSpec{{
				ID: "evidence-a", Origin: "teacher", Excerpt: "Method A",
			}},
			Turns: []turnSpec{{User: "Explain this method"}},
		},
		{
			ID: "CASE-B",
			Evidence: []evidenceSpec{{
				ID: "evidence-b", Origin: "teacher", Excerpt: "Method B",
			}},
			Turns: []turnSpec{{User: "Explain this method"}},
		},
	})

	for _, test := range []struct {
		learnerID string
		wantID    string
	}{
		{learnerID: "harness-case-a-1001", wantID: "evidence-a"},
		{learnerID: "harness-case-b-1002", wantID: "evidence-b"},
	} {
		items, err := retriever.Retrieve(context.Background(), retrieval.TutorEvidenceRequest{
			LearnerID: test.learnerID,
			Query:     "Explain this method",
		})
		if err != nil {
			t.Fatal(err)
		}
		if len(items) != 1 || items[0].ID != test.wantID {
			t.Fatalf("Retrieve(%q) = %#v, want evidence %q", test.learnerID, items, test.wantID)
		}
	}
}

func TestHarnessEvidenceRetrieverRejectsUnscopedLearner(t *testing.T) {
	retriever := newHarnessEvidenceRetriever([]conversationSpec{{
		ID:       "CASE-A",
		Evidence: []evidenceSpec{{ID: "evidence-a", Origin: "teacher"}},
		Turns:    []turnSpec{{User: "Same prompt"}},
	}})

	items, err := retriever.Retrieve(context.Background(), retrieval.TutorEvidenceRequest{
		LearnerID: "another-case-a-1001",
		Query:     "Same prompt",
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 0 {
		t.Fatalf("unscoped learner received evidence: %#v", items)
	}
}

func TestValidateFixtureResolvesCharacterAndDynamics(t *testing.T) {
	fixture := fixtureFile{
		Version: 2,
		Characters: []characterSpec{{
			ID: "aina", FirstName: "Aina", Language: "en",
		}},
		Conversations: []conversationSpec{{
			ID: "NAT01", Title: "Natural interruption", CharacterID: "aina",
			Turns: []turnSpec{
				{User: "Explain everything", ExpectStatus: conversationharness.StatusInterrupted},
				{
					User: "Wait, one point only", Delivery: conversationharness.DeliveryInterrupt,
					ExpectStatus: conversationharness.StatusDelivered,
				},
			},
		}},
	}

	if err := validateFixture(&fixture); err != nil {
		t.Fatalf("validateFixture() error = %v", err)
	}
	if got := fixture.Conversations[0].resolvedCharacter.FirstName; got != "Aina" {
		t.Fatalf("resolved character name = %q, want Aina", got)
	}
}

func TestValidateFixtureRejectsUnknownCharacter(t *testing.T) {
	fixture := fixtureFile{
		Version: 2,
		Conversations: []conversationSpec{{
			ID: "NAT01", Title: "Unknown character", CharacterID: "missing",
			Turns: []turnSpec{{User: "hello"}},
		}},
	}
	if err := validateFixture(&fixture); err == nil {
		t.Fatal("validateFixture() should reject an unknown character")
	}
}

func TestLoadFixtureRejectsUnknownFields(t *testing.T) {
	path := t.TempDir() + "/fixture.yaml"
	fixture := `version: 2
provider: mock
conversations:
  - id: NAT01
    title: Typo must fail
    turns:
      - user: hello
        delivry: queue
`
	if err := os.WriteFile(path, []byte(fixture), 0o600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	_, err := loadFixture(path)
	if err == nil {
		t.Fatal("loadFixture() should reject an unknown field")
	}
	if !strings.Contains(err.Error(), "field delivry not found") {
		t.Fatalf("loadFixture() error = %q, want unknown-field detail", err)
	}
}

func TestLoadFixtureParsesTurnDelay(t *testing.T) {
	path := t.TempDir() + "/fixture.yaml"
	fixture := `version: 2
provider: mock
conversations:
  - id: NAT01
    title: Delayed follow-up
    turns:
      - user: hello
        after: 25ms
`
	if err := os.WriteFile(path, []byte(fixture), 0o600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	loaded, err := loadFixture(path)
	if err != nil {
		t.Fatalf("loadFixture() error = %v", err)
	}
	if got := loaded.Conversations[0].Turns[0].After.Duration; got != 25*time.Millisecond {
		t.Fatalf("turn delay = %v, want 25ms", got)
	}
}

func TestLoadFixtureRejectsInvalidTurnDelay(t *testing.T) {
	for _, test := range []struct {
		name  string
		after string
		want  string
	}{
		{name: "malformed", after: "soon", want: `parse duration "soon"`},
		{name: "negative", after: "-1s", want: "duration must not be negative"},
	} {
		t.Run(test.name, func(t *testing.T) {
			path := t.TempDir() + "/fixture.yaml"
			fixture := `version: 2
provider: mock
conversations:
  - id: NAT01
    title: Invalid delay
    turns:
      - user: hello
        after: ` + test.after + "\n"
			if err := os.WriteFile(path, []byte(fixture), 0o600); err != nil {
				t.Fatalf("WriteFile() error = %v", err)
			}

			_, err := loadFixture(path)
			if err == nil {
				t.Fatal("loadFixture() should reject an invalid turn delay")
			}
			if !strings.Contains(err.Error(), test.want) {
				t.Fatalf("loadFixture() error = %q, want %q", err, test.want)
			}
		})
	}
}

func TestLoadFixtureKeepsVersionOneCompatibility(t *testing.T) {
	path := t.TempDir() + "/fixture.yaml"
	fixture := `version: 1
provider: mock
conversations:
  - id: LEGACY01
    title: Supported legacy fixture
    turns:
      - user: hello
    checks:
      require_non_empty_replies: true
`
	if err := os.WriteFile(path, []byte(fixture), 0o600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	loaded, err := loadFixture(path)
	if err != nil {
		t.Fatalf("loadFixture() error = %v", err)
	}
	if loaded.Version != 1 || len(loaded.Conversations) != 1 {
		t.Fatalf("loaded fixture = %#v, want one version 1 conversation", loaded)
	}
}

func TestRunConversationPassesCharacterProfileToEveryTurn(t *testing.T) {
	var messages []chat.InboundMessage
	result := runConversation(
		func(_ context.Context, message chat.InboundMessage) (string, error) {
			messages = append(messages, message)
			return "short reply", nil
		},
		conversationSpec{
			ID:    "NAT01",
			Title: "Character profile",
			resolvedCharacter: characterSpec{
				FirstName: "Aina",
				Username:  "aina",
				Language:  "en",
			},
			Turns: []turnSpec{
				{User: "first"},
				{User: "second"},
			},
		},
		testTurnTimeout,
		false,
		false,
	)

	if !result.Passed {
		t.Fatalf("runConversation() failures = %#v", result.Failures)
	}
	if len(messages) != 2 {
		t.Fatalf("processed messages = %d, want 2", len(messages))
	}
	for index, message := range messages {
		if message.FirstName != "Aina" || message.Username != "aina" || message.Language != "en" {
			t.Fatalf("message %d profile = %#v, want Aina/aina/en", index+1, message)
		}
	}
	if len(result.Outcomes) != 2 {
		t.Fatalf("outcomes = %d, want 2", len(result.Outcomes))
	}
	for index, outcome := range result.Outcomes {
		if outcome.Turn != index+1 ||
			outcome.Delivery != conversationharness.DeliveryWait ||
			outcome.Status != conversationharness.StatusDelivered {
			t.Fatalf("outcome %d = %#v", index+1, outcome)
		}
	}
}

func TestRunConversationSuppressesInterruptedResponse(t *testing.T) {
	engine, cleanup, err := buildEngine(true, "stale mock response", false, nil, nil)
	if err != nil {
		t.Fatalf("buildEngine() error = %v", err)
	}
	defer cleanup()

	result := runConversation(engine.ProcessMessage, conversationSpec{
		ID:    "NAT01",
		Title: "Interrupt stale response",
		Turns: []turnSpec{
			{User: "long request", ExpectStatus: conversationharness.StatusInterrupted},
			{
				User: "replacement", Delivery: conversationharness.DeliveryInterrupt,
				ExpectStatus: conversationharness.StatusDelivered,
			},
		},
	}, testTurnTimeout, false, false)

	if !result.Passed {
		t.Fatalf("runConversation() failures = %#v", result.Failures)
	}
	if result.Delivered != 1 || result.Interrupted != 1 {
		t.Fatalf(
			"delivered/interrupted = %d/%d, want 1/1",
			result.Delivered,
			result.Interrupted,
		)
	}
	if len(result.Outcomes) != 2 ||
		result.Outcomes[0].Status != conversationharness.StatusInterrupted ||
		result.Outcomes[1].Status != conversationharness.StatusDelivered {
		t.Fatalf("outcomes = %#v, want interrupted then delivered", result.Outcomes)
	}
}

const testTurnTimeout = 5 * time.Second

func engineTrackerIsNil(engine any) bool {
	field := reflect.ValueOf(engine).Elem().FieldByName("tracker")
	return field.IsNil()
}
