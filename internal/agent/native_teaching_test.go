// Copyright 2026 the P&AI authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package agent_test

import (
	"context"
	"errors"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/p-n-ai/pai-bot/internal/agent"
	"github.com/p-n-ai/pai-bot/internal/ai"
	"github.com/p-n-ai/pai-bot/internal/chat"
	"github.com/p-n-ai/pai-bot/internal/llm"
	"github.com/p-n-ai/pai-bot/internal/platform/featureflags"
)

type scriptedNativeProvider struct {
	mu        sync.Mutex
	responses []llm.AssistantMessage
	contexts  []llm.Context
}

func (p *scriptedNativeProvider) CompleteNative(_ context.Context, _ string, c llm.Context, _ *llm.StreamOptions) (llm.AssistantMessage, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.contexts = append(p.contexts, c)
	if len(p.responses) == 0 {
		return llm.AssistantMessage{}, errors.New("unexpected native model call")
	}
	response := p.responses[0]
	p.responses = p.responses[1:]
	return response, nil
}

func (*scriptedNativeProvider) Complete(context.Context, ai.CompletionRequest) (ai.CompletionResponse, error) {
	return ai.CompletionResponse{}, errors.New("legacy completion should not run")
}

func (*scriptedNativeProvider) StreamComplete(context.Context, ai.CompletionRequest) (<-chan ai.StreamChunk, error) {
	return nil, errors.New("legacy stream should not run")
}

func (*scriptedNativeProvider) Models() []ai.ModelInfo            { return nil }
func (*scriptedNativeProvider) HealthCheck(context.Context) error { return nil }

type serialNativeProvider struct {
	started chan struct{}
	release chan struct{}
	mu      sync.Mutex
	calls   int
	active  int
	max     int
}

func (p *serialNativeProvider) CompleteNative(ctx context.Context, _ string, _ llm.Context, _ *llm.StreamOptions) (llm.AssistantMessage, error) {
	p.mu.Lock()
	p.calls++
	call := p.calls
	p.active++
	if p.active > p.max {
		p.max = p.active
	}
	p.mu.Unlock()
	defer func() {
		p.mu.Lock()
		p.active--
		p.mu.Unlock()
	}()
	if call == 1 {
		close(p.started)
		select {
		case <-ctx.Done():
			return llm.AssistantMessage{}, ctx.Err()
		case <-p.release:
		}
	}
	return llm.AssistantMessage{
		Content:       []llm.AssistantContent{llm.TextContent{Text: "Tutor response"}},
		Model:         "serial-test",
		ResponseModel: "serial-test",
		StopReason:    llm.StopReasonStop,
	}, nil
}

func (*serialNativeProvider) Complete(context.Context, ai.CompletionRequest) (ai.CompletionResponse, error) {
	return ai.CompletionResponse{}, errors.New("legacy completion should not run")
}

func (*serialNativeProvider) StreamComplete(context.Context, ai.CompletionRequest) (<-chan ai.StreamChunk, error) {
	return nil, errors.New("legacy stream should not run")
}

func (*serialNativeProvider) Models() []ai.ModelInfo            { return nil }
func (*serialNativeProvider) HealthCheck(context.Context) error { return nil }

func TestEngineAgentCoreRunsCurriculumToolAndPersistsOnlyFinalText(t *testing.T) {
	features, err := featureflags.Parse("agent_core")
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	provider := &scriptedNativeProvider{responses: []llm.AssistantMessage{
		{
			Content: []llm.AssistantContent{llm.ToolCall{
				ID:        "curriculum-1",
				Name:      "lookup_curriculum_topic",
				Arguments: map[string]any{"topic_id": "F1-02"},
			}},
			StopReason: llm.StopReasonToolUse,
		},
		{
			Content:       []llm.AssistantContent{llm.TextContent{Text: "Subtract 1 from both sides first."}},
			Model:         "native-test",
			ResponseModel: "native-test",
			Usage:         llm.Usage{Input: 12, Output: 7},
			StopReason:    llm.StopReasonStop,
		},
	}}
	router := ai.NewRouterWithConfig(ai.RouterConfig{RetryBackoff: []time.Duration{0}})
	router.Register("native-test", provider)
	store := agent.NewMemoryStore()
	engine := agent.NewEngine(agent.EngineConfig{
		AIRouter:         router,
		Store:            store,
		CurriculumLoader: createTestCurriculumLoader(t),
		FeatureFlags:     func() featureflags.Features { return features },
	})

	response, err := engine.ProcessMessage(context.Background(), chat.InboundMessage{
		Channel: "telegram",
		UserID:  "native-tool-user",
		Text:    "Help me solve a linear equation",
	})
	if err != nil {
		t.Fatalf("ProcessMessage() error = %v", err)
	}
	if response != "Subtract 1 from both sides first." {
		t.Fatalf("response = %q", response)
	}
	if len(provider.contexts) != 2 {
		t.Fatalf("native calls = %d, want 2", len(provider.contexts))
	}
	if provider.contexts[0].SystemPrompt == "" || len(provider.contexts[0].Tools) != 1 {
		t.Fatalf("initial native context = %#v", provider.contexts[0])
	}
	for _, message := range provider.contexts[0].Messages {
		if _, ok := message.(llm.SystemMessage); ok {
			t.Fatal("native context must use SystemPrompt instead of system messages")
		}
	}
	continuation := provider.contexts[1].Messages
	if len(continuation) < 3 {
		t.Fatalf("continuation transcript = %#v", continuation)
	}
	toolResult, ok := continuation[len(continuation)-1].(llm.ToolResultMessage)
	if !ok || toolResult.ToolCallID != "curriculum-1" || toolResult.ToolName != "lookup_curriculum_topic" || toolResult.IsError {
		t.Fatalf("tool result = %#v", continuation[len(continuation)-1])
	}
	resultText := toolResult.Content[0].(llm.TextContent).Text
	if !strings.Contains(resultText, `"ID":"F1-02"`) || !strings.Contains(resultText, "Treat the equation like a balance") {
		t.Fatalf("curriculum result = %q", resultText)
	}

	conv, found := store.GetActiveConversation("native-tool-user")
	if !found || len(conv.Messages) != 2 {
		t.Fatalf("stored conversation = %#v", conv)
	}
	if conv.Messages[0].Role != "user" || conv.Messages[1].Role != "assistant" || conv.Messages[1].Content != response {
		t.Fatalf("stored messages = %#v", conv.Messages)
	}
}

func TestEngineSerializesTeachingTurnsPerConversation(t *testing.T) {
	features, err := featureflags.Parse("agent_core")
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	provider := &serialNativeProvider{started: make(chan struct{}), release: make(chan struct{})}
	router := ai.NewRouterWithConfig(ai.RouterConfig{RetryBackoff: []time.Duration{0}})
	router.Register("serial-test", provider)
	engine := agent.NewEngine(agent.EngineConfig{
		AIRouter:     router,
		Store:        agent.NewMemoryStore(),
		FeatureFlags: func() featureflags.Features { return features },
	})

	firstDone := make(chan error, 1)
	go func() {
		_, err := engine.ProcessMessage(context.Background(), chat.InboundMessage{Channel: "telegram", UserID: "serial-user", Text: "first"})
		firstDone <- err
	}()
	select {
	case <-provider.started:
	case <-time.After(time.Second):
		t.Fatal("first teaching turn did not reach model")
	}

	secondDone := make(chan error, 1)
	go func() {
		_, err := engine.ProcessMessage(context.Background(), chat.InboundMessage{Channel: "telegram", UserID: "serial-user", Text: "second"})
		secondDone <- err
	}()
	time.Sleep(50 * time.Millisecond)
	provider.mu.Lock()
	callsBeforeRelease := provider.calls
	provider.mu.Unlock()
	if callsBeforeRelease != 1 {
		t.Fatalf("model calls before first turn released = %d, want 1", callsBeforeRelease)
	}

	close(provider.release)
	for i, done := range []<-chan error{firstDone, secondDone} {
		select {
		case err := <-done:
			if err != nil {
				t.Fatalf("turn %d error = %v", i+1, err)
			}
		case <-time.After(time.Second):
			t.Fatalf("turn %d did not complete", i+1)
		}
	}
	provider.mu.Lock()
	defer provider.mu.Unlock()
	if provider.max != 1 || provider.calls != 2 {
		t.Fatalf("provider max concurrency = %d calls = %d, want 1 and 2", provider.max, provider.calls)
	}
}
