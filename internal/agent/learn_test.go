// Copyright 2026 the P&AI authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package agent_test

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/p-n-ai/pai-bot/internal/agent"
	"github.com/p-n-ai/pai-bot/internal/ai"
	"github.com/p-n-ai/pai-bot/internal/chat"
	"github.com/p-n-ai/pai-bot/internal/curriculum"
)

func TestLearnCommand_SetsTopicAndResponds(t *testing.T) {
	provider := &echoProvider{}
	router := ai.NewRouter()
	router.Register("mock", provider)

	loader := createLearnTestLoader(t)
	engine := agent.NewEngine(agent.EngineConfig{
		AIRouter:             router,
		Store:                agent.NewMemoryStore(),
		CurriculumLoader:     loader,
		DisableMultiLanguage: true,
	})

	resp, err := engine.ProcessMessage(context.Background(), chat.InboundMessage{
		Channel: "terminal",
		UserID:  "user1",
		Text:    "/learn persamaan linear",
	})
	if err != nil {
		t.Fatalf("ProcessMessage: %v", err)
	}

	lower := strings.ToLower(resp)
	if !strings.Contains(lower, "persamaan linear") && !strings.Contains(lower, "linear equation") {
		t.Errorf("expected response to mention the topic name, got: %s", resp)
	}
}

func TestLearnCommand_NoArgs_ShowsUsage(t *testing.T) {
	engine := agent.NewEngine(agent.EngineConfig{
		AIRouter:             ai.NewRouter(),
		Store:                agent.NewMemoryStore(),
		DisableMultiLanguage: true,
	})

	resp, err := engine.ProcessMessage(context.Background(), chat.InboundMessage{
		Channel: "terminal",
		UserID:  "user1",
		Text:    "/learn",
	})
	if err != nil {
		t.Fatalf("ProcessMessage: %v", err)
	}

	if !strings.Contains(strings.ToLower(resp), "/learn") {
		t.Errorf("expected usage hint with /learn, got: %s", resp)
	}
}

func TestLearnCommand_TopicNotFound(t *testing.T) {
	loader := createLearnTestLoader(t)
	engine := agent.NewEngine(agent.EngineConfig{
		AIRouter:             ai.NewRouter(),
		Store:                agent.NewMemoryStore(),
		CurriculumLoader:     loader,
		DisableMultiLanguage: true,
	})

	resp, err := engine.ProcessMessage(context.Background(), chat.InboundMessage{
		Channel: "terminal",
		UserID:  "user1",
		Text:    "/learn quantum physics",
	})
	if err != nil {
		t.Fatalf("ProcessMessage: %v", err)
	}

	lower := strings.ToLower(resp)
	if !strings.Contains(lower, "tidak dijumpai") && !strings.Contains(lower, "not found") && !strings.Contains(lower, "topik") {
		t.Errorf("expected topic-not-found message, got: %s", resp)
	}
}

func TestLearnCommand_RejectsTopicBelowAITeachingQuality(t *testing.T) {
	loader := createLearnTestLoaderAtQuality(t, 2)
	store := agent.NewMemoryStore()
	engine := agent.NewEngine(agent.EngineConfig{
		AIRouter:             ai.NewRouter(),
		Store:                store,
		CurriculumLoader:     loader,
		DisableMultiLanguage: true,
	})

	resp, err := engine.ProcessMessage(context.Background(), chat.InboundMessage{
		Channel: "terminal",
		UserID:  "user1",
		Text:    "/learn persamaan linear",
	})
	if err != nil {
		t.Fatalf("ProcessMessage: %v", err)
	}
	identity, err := agent.NewLearnerIdentity("terminal", "user1")
	if err != nil {
		t.Fatalf("NewLearnerIdentity: %v", err)
	}
	if conversation, found := store.GetActiveConversationFor(identity); found && conversation.TopicID != "" {
		t.Fatalf("conversation.TopicID = %q after response %q, want unset", conversation.TopicID, resp)
	}
}

func TestLearnCommand_SetsConversationTopicID(t *testing.T) {
	provider := &echoProvider{}
	router := ai.NewRouter()
	router.Register("mock", provider)

	store := agent.NewMemoryStore()
	loader := createLearnTestLoader(t)
	engine := agent.NewEngine(agent.EngineConfig{
		AIRouter:             router,
		Store:                store,
		CurriculumLoader:     loader,
		DisableMultiLanguage: true,
	})

	_, err := engine.ProcessMessage(context.Background(), chat.InboundMessage{
		Channel: "terminal",
		UserID:  "user1",
		Text:    "/learn persamaan linear",
	})
	if err != nil {
		t.Fatalf("ProcessMessage: %v", err)
	}

	// Verify the conversation's topic was set.
	identity, err := agent.NewLearnerIdentity("terminal", "user1")
	if err != nil {
		t.Fatalf("NewLearnerIdentity: %v", err)
	}
	conv, found := store.GetActiveConversationFor(identity)
	if !found {
		t.Fatal("expected active conversation")
	}
	if conv.TopicID != "F1-06" {
		t.Errorf("expected TopicID=F1-06, got %q", conv.TopicID)
	}
}

// echoProvider returns a fixed response for testing.
type echoProvider struct{}

func (p *echoProvider) Complete(_ context.Context, req ai.CompletionRequest) (ai.CompletionResponse, error) {
	return ai.CompletionResponse{
		Content: "Baik, mari kita belajar topik ini bersama.",
		Model:   "mock",
	}, nil
}
func (p *echoProvider) StreamComplete(context.Context, ai.CompletionRequest) (<-chan ai.StreamChunk, error) {
	return nil, nil
}
func (p *echoProvider) Models() []ai.ModelInfo            { return nil }
func (p *echoProvider) HealthCheck(context.Context) error { return nil }

func createLearnTestLoader(t *testing.T) *curriculum.Loader {
	return createLearnTestLoaderAtQuality(t, 3)
}

func createLearnTestLoaderAtQuality(t *testing.T, qualityLevel int) *curriculum.Loader {
	t.Helper()
	dir := t.TempDir()
	curriculumDir := filepath.Join(dir, "curricula", "default")
	topicsDir := filepath.Join(curriculumDir, "topics")
	if err := os.MkdirAll(topicsDir, 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	for name, content := range map[string]string{
		"syllabus.yaml": `id: default
name: Default
subjects:
  - algebra
`,
		"subject.yaml": `id: algebra
name: Algebra
syllabus_id: default
`,
		"subject-grade.yaml": `id: algebra-f1
name: Algebra Form 1
subject_id: algebra
syllabus_id: default
grade_id: form-1
topics:
  - F1-06
`,
	} {
		path := filepath.Join(curriculumDir, name)
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			t.Fatalf("write metadata %s: %v", name, err)
		}
	}

	topicYAML := fmt.Sprintf(`id: F1-06
name: "Persamaan Linear (Linear Equations)"
subject_grade_id: algebra-f1
subject_id: algebra
syllabus_id: default
difficulty: beginner
content_standards:
  - id: "1.1"
    text: Linear equations
learning_objectives:
  - id: LO1
    content_standard_id: "1.1"
    text: Solve linear equations.
    bloom: apply
quality_level: %d
provenance: human
`, qualityLevel)
	if err := os.WriteFile(filepath.Join(topicsDir, "F1-06.yaml"), []byte(topicYAML), 0o644); err != nil {
		t.Fatalf("write topic: %v", err)
	}

	teachingMD := "# Persamaan Linear\nTeaching notes for linear equations."
	if err := os.WriteFile(filepath.Join(topicsDir, "F1-06.teaching.md"), []byte(teachingMD), 0o644); err != nil {
		t.Fatalf("write teaching notes: %v", err)
	}
	if err := os.WriteFile(filepath.Join(topicsDir, "F1-06.examples.yaml"), []byte(`
topic_id: F1-06
worked_examples:
  - id: WE-01
    topic: Solving linear equations
    difficulty: easy
    learning_objective: LO1
    misconception_alert: Moving a term changes nothing.
    scenario: Solve x plus 1 equals 2.
    working: Subtract 1 from both sides.
`), 0o644); err != nil {
		t.Fatalf("write examples: %v", err)
	}
	if err := os.WriteFile(filepath.Join(topicsDir, "F1-06.assessments.yaml"), []byte(`
topic_id: F1-06
questions:
  - id: Q1
    text: Solve x plus 1 equals 2.
    difficulty: easy
    learning_objective: LO1
    answer:
      type: exact
      value: "1"
`), 0o644); err != nil {
		t.Fatalf("write assessment: %v", err)
	}

	loader, err := curriculum.NewLoader(dir)
	if err != nil {
		t.Fatalf("NewLoader: %v", err)
	}
	return loader
}
