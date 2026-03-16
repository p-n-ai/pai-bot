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
	"github.com/p-n-ai/pai-bot/internal/progress"
)

// capturingProvider captures all requests sent to the AI provider.
type capturingProvider struct {
	requests []ai.CompletionRequest
}

func (p *capturingProvider) Complete(_ context.Context, req ai.CompletionRequest) (ai.CompletionResponse, error) {
	p.requests = append(p.requests, req)
	return ai.CompletionResponse{
		Content: "Faham/Understand: Kita perlu selesaikan persamaan linear.\nRancang/Plan: Langkah 1.\nSemak/Verify: OK.\nKonsep/Connect: Persamaan linear.",
		Model:   "mock",
	}, nil
}
func (p *capturingProvider) StreamComplete(context.Context, ai.CompletionRequest) (<-chan ai.StreamChunk, error) {
	return nil, nil
}
func (p *capturingProvider) Models() []ai.ModelInfo     { return nil }
func (p *capturingProvider) HealthCheck(context.Context) error { return nil }

func TestAdaptiveDepth_BeginnerInSystemPrompt(t *testing.T) {
	provider := &capturingProvider{}
	router := ai.NewRouter()
	router.Register("mock", provider)

	tracker := progress.NewMemoryTracker()
	_ = tracker.UpdateMastery("user1", "default", "F1-06", 0.15)

	loader := createAdaptiveTestLoader(t)
	engine := agent.NewEngine(agent.EngineConfig{
		AIRouter:             router,
		Store:                agent.NewMemoryStore(),
		CurriculumLoader:     loader,
		Tracker:              tracker,
		DisableMultiLanguage: true,
	})

	_, err := engine.ProcessMessage(context.Background(), chat.InboundMessage{
		Channel: "terminal",
		UserID:  "user1",
		Text:    "Ajar saya persamaan linear 2x + 4 = 10",
	})
	if err != nil {
		t.Fatalf("ProcessMessage: %v", err)
	}

	if len(provider.requests) == 0 {
		t.Fatal("no AI requests captured")
	}

	systemPrompt := provider.requests[0].Messages[0].Content
	lower := strings.ToLower(systemPrompt)
	if !strings.Contains(lower, "beginner") {
		t.Errorf("expected system prompt to contain BEGINNER depth")
	}
	if !strings.Contains(lower, "simple") {
		t.Error("expected system prompt to contain 'simple' for beginner level")
	}
}

func TestAdaptiveDepth_ProficientInSystemPrompt(t *testing.T) {
	provider := &capturingProvider{}
	router := ai.NewRouter()
	router.Register("mock", provider)

	tracker := progress.NewMemoryTracker()
	_ = tracker.UpdateMastery("user1", "default", "F1-06", 0.85)

	loader := createAdaptiveTestLoader(t)
	engine := agent.NewEngine(agent.EngineConfig{
		AIRouter:             router,
		Store:                agent.NewMemoryStore(),
		CurriculumLoader:     loader,
		Tracker:              tracker,
		DisableMultiLanguage: true,
	})

	_, err := engine.ProcessMessage(context.Background(), chat.InboundMessage{
		Channel: "terminal",
		UserID:  "user1",
		Text:    "Ajar saya persamaan linear 2x + 4 = 10",
	})
	if err != nil {
		t.Fatalf("ProcessMessage: %v", err)
	}

	systemPrompt := provider.requests[0].Messages[0].Content
	lower := strings.ToLower(systemPrompt)
	if !strings.Contains(lower, "proficient") {
		t.Errorf("expected system prompt to contain PROFICIENT depth")
	}
	if !strings.Contains(lower, "edge case") {
		t.Error("expected system prompt to contain 'edge case' for proficient level")
	}
}

func TestAdaptiveDepth_ProgressContextInPrompt(t *testing.T) {
	provider := &capturingProvider{}
	router := ai.NewRouter()
	router.Register("mock", provider)

	tracker := progress.NewMemoryTracker()
	_ = tracker.UpdateMastery("user1", "default", "F1-05", 0.9)
	_ = tracker.UpdateMastery("user1", "default", "F1-06", 0.5)
	_ = tracker.UpdateMastery("user1", "default", "F1-07", 0.1)

	loader := createAdaptiveTestLoader(t)
	engine := agent.NewEngine(agent.EngineConfig{
		AIRouter:             router,
		Store:                agent.NewMemoryStore(),
		CurriculumLoader:     loader,
		Tracker:              tracker,
		DisableMultiLanguage: true,
	})

	_, err := engine.ProcessMessage(context.Background(), chat.InboundMessage{
		Channel: "terminal",
		UserID:  "user1",
		Text:    "Ajar saya persamaan linear",
	})
	if err != nil {
		t.Fatalf("ProcessMessage: %v", err)
	}

	systemPrompt := provider.requests[0].Messages[0].Content
	if !strings.Contains(systemPrompt, "F1-05") {
		t.Error("expected progress context to include mastered topic F1-05")
	}
	if !strings.Contains(systemPrompt, "F1-07") {
		t.Error("expected progress context to include struggling topic F1-07")
	}
}

func TestAdaptiveDepth_NoTrackerNoBlock(t *testing.T) {
	provider := &capturingProvider{}
	router := ai.NewRouter()
	router.Register("mock", provider)

	engine := agent.NewEngine(agent.EngineConfig{
		AIRouter:             router,
		Store:                agent.NewMemoryStore(),
		Tracker:              nil,
		DisableMultiLanguage: true,
	})

	_, _ = engine.ProcessMessage(context.Background(), chat.InboundMessage{
		Channel: "terminal",
		UserID:  "user1",
		Text:    "Hello",
	})

	if len(provider.requests) == 0 {
		t.Fatal("no AI requests captured")
	}

	systemPrompt := provider.requests[0].Messages[0].Content
	if strings.Contains(strings.ToLower(systemPrompt), "adaptive explanation depth") {
		t.Error("expected NO adaptive depth block when tracker is nil")
	}
}

func createAdaptiveTestLoader(t *testing.T) *curriculum.Loader {
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
