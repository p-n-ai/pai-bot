// Copyright 2026 the P&AI authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package agent_test

import (
	"context"
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
	conv, found := store.GetActiveConversation("user1")
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
func (p *echoProvider) Models() []ai.ModelInfo          { return nil }
func (p *echoProvider) HealthCheck(context.Context) error { return nil }

func createLearnTestLoader(t *testing.T) *curriculum.Loader {
	t.Helper()
	dir := t.TempDir()

	topicYAML := `id: F1-06
name: "Persamaan Linear (Linear Equations)"
subject_id: algebra
syllabus_id: default
keywords:
  - persamaan linear
  - linear equation
`
	if err := os.WriteFile(filepath.Join(dir, "F1-06.yaml"), []byte(topicYAML), 0o644); err != nil {
		t.Fatalf("write topic: %v", err)
	}

	teachingMD := "# Persamaan Linear\nTeaching notes for linear equations."
	if err := os.WriteFile(filepath.Join(dir, "F1-06.teaching.md"), []byte(teachingMD), 0o644); err != nil {
		t.Fatalf("write teaching notes: %v", err)
	}

	loader, err := curriculum.NewLoader(dir)
	if err != nil {
		t.Fatalf("NewLoader: %v", err)
	}
	return loader
}
