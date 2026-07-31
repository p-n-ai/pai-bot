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
	if !engineCurriculumRuntimeIsNil(engine) {
		t.Fatal("conversation harness should not enable curriculum mastery without --progress")
	}
}

func TestBuildEngineCanEnableProgress(t *testing.T) {
	t.Setenv("LEARN_CURRICULUM_PATH", "../../oss")
	engine, cleanup, err := buildEngine(true, "mock tutor response", true, nil, nil)
	if err != nil {
		t.Fatalf("buildEngine() error = %v", err)
	}
	defer cleanup()

	if engineTrackerIsNil(engine) {
		t.Fatal("conversation harness should enable mastery tracker when --progress is set")
	}
	if engineCurriculumRuntimeIsNil(engine) {
		t.Fatal("conversation harness should enable curriculum runtime when --progress is set")
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

func TestRunConversationPropagatesCharacterOnEveryTurn(t *testing.T) {
	processor := &recordingConversationProcessor{responses: []string{"first reply", "second reply"}}
	conv := conversationSpec{
		ID: "PERSONA",
		Character: characterSpec{
			ID:        "faris-terse",
			FirstName: "Faris",
			Username:  "faris",
			Language:  "ms",
		},
		Turns: []turnSpec{{User: "first turn"}, {User: "second turn"}},
	}

	result := runConversation(processor, conv, time.Second, false, false)
	if !result.Passed {
		t.Fatalf("runConversation() failures = %v", result.Failures)
	}
	if len(processor.messages) != 2 {
		t.Fatalf("ProcessMessage() calls = %d, want 2", len(processor.messages))
	}
	for i, msg := range processor.messages {
		if msg.FirstName != "Faris" || msg.Username != "faris" || msg.Language != "ms" {
			t.Errorf("turn %d persona = (%q, %q, %q), want (Faris, faris, ms)", i+1, msg.FirstName, msg.Username, msg.Language)
		}
	}
	if processor.messages[0].UserID == "" || processor.messages[0].UserID != processor.messages[1].UserID {
		t.Errorf("learner IDs = %q and %q, want one non-empty conversation learner ID", processor.messages[0].UserID, processor.messages[1].UserID)
	}
	if processor.messages[0].DeliveryID == processor.messages[1].DeliveryID {
		t.Errorf("delivery IDs are not unique: %q", processor.messages[0].DeliveryID)
	}
}

func TestRunConversationAppliesTurnLocalResponseLimitOnlyToDeclaredTurn(t *testing.T) {
	conv := conversationSpec{
		ID: "TURN-CHECK",
		Turns: []turnSpec{
			{User: "first turn"},
			{User: "second turn", Checks: behaviorChecks{MaxResponseChars: 4}},
		},
	}

	processor := &recordingConversationProcessor{responses: []string{"long first response", "tiny"}}
	if result := runConversation(processor, conv, time.Second, false, true); !result.Passed {
		t.Fatalf("turn-local limit affected another turn: %v", result.Failures)
	}

	processor = &recordingConversationProcessor{responses: []string{"long first response", "large"}}
	result := runConversation(processor, conv, time.Second, false, true)
	want := []string{"turn 2: response has 5 chars, max 4"}
	if !reflect.DeepEqual(result.Failures, want) {
		t.Fatalf("runConversation() failures = %v, want %v", result.Failures, want)
	}
}

type recordingConversationProcessor struct {
	responses []string
	messages  []chat.InboundMessage
}

func (p *recordingConversationProcessor) ProcessMessage(_ context.Context, msg chat.InboundMessage) (string, error) {
	p.messages = append(p.messages, msg)
	return p.responses[len(p.messages)-1], nil
}

func engineTrackerIsNil(engine any) bool {
	field := reflect.ValueOf(engine).Elem().FieldByName("tracker")
	return field.IsNil()
}

func engineCurriculumRuntimeIsNil(engine any) bool {
	field := reflect.ValueOf(engine).Elem().FieldByName("curriculumRuntime")
	return field.IsNil()
}

const validFixtureYAML = `version: 1
characters:
  - id: " teacher "
    first_name: " Ada "
    username: " ada-teacher "
    language: " BM-my "
conversations:
  - id: " CASE-1 "
    title: " Linear equations "
    character: " teacher "
    turns:
      - user: " learner question "
        checks:
          expected_language: " en_or_mixed "
          max_response_lines: 3
    checks:
      require_response_phrases: [" explain "]
      forbid_final_answer_on_turn: [1]
      max_response_chars: 200
`

func TestLoadFixtureNormalizesAndResolvesCharacter(t *testing.T) {
	fixture, err := loadFixture(writeFixture(t, validFixtureYAML))
	if err != nil {
		t.Fatalf("loadFixture() error = %v", err)
	}

	wantCharacter := characterSpec{ID: "teacher", FirstName: "Ada", Username: "ada-teacher", Language: "ms"}
	if len(fixture.Characters) != 1 || !reflect.DeepEqual(fixture.Characters[0], wantCharacter) {
		t.Fatalf("Characters = %#v, want %#v", fixture.Characters, wantCharacter)
	}
	conversation := fixture.Conversations[0]
	if conversation.ID != "CASE-1" || conversation.Title != "Linear equations" || conversation.CharacterID != "teacher" {
		t.Fatalf("conversation identifiers were not normalized: %#v", conversation)
	}
	if !reflect.DeepEqual(conversation.Character, wantCharacter) {
		t.Fatalf("resolved Character = %#v, want %#v", conversation.Character, wantCharacter)
	}

	if conversation.Turns[0].User != "learner question" || conversation.Turns[0].Checks.ExpectedLanguage != "en_or_mixed" {
		t.Fatalf("turn was not normalized: %#v", conversation.Turns[0])
	}
	if conversation.Checks.RequireResponsePhrases[0] != "explain" {
		t.Fatalf("conversation checks were not normalized: %#v", conversation.Checks)
	}
}

func TestLoadFixtureRejectsInvalidYAMLContracts(t *testing.T) {
	tests := []struct {
		name string
		yaml string
		want string
	}{
		{name: "unknown field", yaml: validFixtureYAML + "typo: true\n", want: "field typo not found"},
		{name: "multiple documents", yaml: validFixtureYAML + "---\nversion: 1\n", want: "exactly one YAML document"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, err := loadFixture(writeFixture(t, test.yaml))
			if err == nil || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("loadFixture() error = %v, want substring %q", err, test.want)
			}
		})
	}
}

func TestLoadFixtureRejectsInvalidSemantics(t *testing.T) {
	tests := []struct {
		name string
		yaml string
		want string
	}{
		{name: "unsupported version", yaml: strings.Replace(validFixtureYAML, "version: 1", "version: 2", 1), want: "want 1"},

		{name: "duplicate character id", yaml: strings.Replace(validFixtureYAML, "conversations:\n", "  - id: teacher\n    first_name: Other\n    username: other\n    language: en\nconversations:\n", 1), want: `duplicate character id "teacher"`},

		{name: "unknown character", yaml: strings.Replace(validFixtureYAML, `    character: " teacher "`, `    character: missing`, 1), want: `unknown character "missing"`},
		{name: "unsupported character language", yaml: strings.Replace(validFixtureYAML, `    language: " BM-my "`, `    language: klingon`, 1), want: `unsupported language "klingon"`},

		{name: "missing learner text", yaml: strings.Replace(validFixtureYAML, `      - user: " learner question "`, `      - user: "   "`, 1), want: "user text is required"},
		{name: "unsupported language", yaml: strings.Replace(validFixtureYAML, `expected_language: " en_or_mixed "`, `expected_language: klingon`, 1), want: "expected_language"},
		{name: "negative turn limit", yaml: strings.Replace(validFixtureYAML, "max_response_lines: 3", "max_response_lines: -1", 1), want: "max_response_lines must be non-negative"},

		{name: "out of range conversation turn reference", yaml: strings.Replace(validFixtureYAML, "forbid_final_answer_on_turn: [1]", "forbid_final_answer_on_turn: [2]", 1), want: "outside 1..1"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, err := loadFixture(writeFixture(t, test.yaml))
			if err == nil || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("loadFixture() error = %v, want substring %q", err, test.want)
			}
		})
	}
}

func writeFixture(t *testing.T, contents string) string {
	t.Helper()
	path := t.TempDir() + "/fixture.yaml"
	if err := os.WriteFile(path, []byte(contents), 0o600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}
	return path
}
