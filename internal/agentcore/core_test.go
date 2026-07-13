// Copyright 2026 the P&AI authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package agentcore_test

import (
	"context"
	"errors"
	"reflect"
	"testing"

	"github.com/p-n-ai/pai-bot/internal/agentcore"
	"github.com/p-n-ai/pai-bot/internal/llm"
)

type scriptedModel struct {
	responses []llm.AssistantMessage
	contexts  []llm.Context
}

func (m *scriptedModel) Complete(_ context.Context, c llm.Context, _ *llm.StreamOptions) (llm.AssistantMessage, error) {
	m.contexts = append(m.contexts, c)
	if len(m.responses) == 0 {
		return llm.AssistantMessage{}, errors.New("unexpected model call")
	}
	response := m.responses[0]
	m.responses = m.responses[1:]
	return response, nil
}

type memoryTool struct {
	calls []llm.ToolCall
}

func (*memoryTool) Definition() llm.Tool {
	return llm.Tool{
		Name:       "lookup_curriculum",
		Parameters: []byte(`{"type":"object","properties":{"topic_id":{"type":"string"}},"required":["topic_id"],"additionalProperties":false}`),
	}
}

func (t *memoryTool) Execute(_ context.Context, call llm.ToolCall) llm.ToolResultMessage {
	t.calls = append(t.calls, call)
	return llm.ToolResultMessage{Content: []llm.UserContent{llm.TextContent{Text: "Linear equations"}}}
}

func TestRunPreservesExactToolTranscript(t *testing.T) {
	call := llm.ToolCall{ID: "call-1", Name: "lookup_curriculum", Arguments: map[string]any{"topic_id": "math-1"}}
	first := llm.AssistantMessage{Content: []llm.AssistantContent{llm.TextContent{Text: "Checking."}, call}, StopReason: llm.StopReasonToolUse}
	final := llm.AssistantMessage{Content: []llm.AssistantContent{llm.TextContent{Text: "Let us solve x + 2 = 5."}}, StopReason: llm.StopReasonStop}
	model := &scriptedModel{responses: []llm.AssistantMessage{first, final}}
	tool := &memoryTool{}
	user := llm.UserText("Help with equations")

	result, err := agentcore.Run(context.Background(), model, llm.Context{
		SystemPrompt: "You are a tutor.",
		Messages:     []llm.Message{user},
	}, []agentcore.Tool{tool}, agentcore.Config{MaxModelCalls: 3})
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	want := []llm.Message{
		user,
		first,
		llm.ToolResultMessage{ToolCallID: "call-1", ToolName: "lookup_curriculum", Content: []llm.UserContent{llm.TextContent{Text: "Linear equations"}}},
		final,
	}
	if !reflect.DeepEqual(result.Messages, want) {
		t.Fatalf("transcript = %#v, want %#v", result.Messages, want)
	}
	if result.Termination != agentcore.TerminationCompleted || result.ModelCalls != 2 {
		t.Fatalf("termination = %q calls = %d", result.Termination, result.ModelCalls)
	}
	if len(model.contexts) != 2 || !reflect.DeepEqual(model.contexts[1].Messages, want[:3]) {
		t.Fatalf("continuation context = %#v, want %#v", model.contexts, want[:3])
	}
}

func TestRunDirectAnswer(t *testing.T) {
	final := llm.AssistantMessage{Content: []llm.AssistantContent{llm.TextContent{Text: "Answer"}}}
	result, err := agentcore.Run(context.Background(), &scriptedModel{responses: []llm.AssistantMessage{final}}, llm.Context{
		SystemPrompt: "Tutor",
		Messages:     []llm.Message{llm.UserText("Question")},
	}, nil, agentcore.Config{})
	if err != nil || result.Termination != agentcore.TerminationCompleted || result.ModelCalls != 1 {
		t.Fatalf("Run() = %#v, %v", result, err)
	}
}

func TestRunReturnsToolErrorsToModel(t *testing.T) {
	tests := []struct {
		name string
		call llm.ToolCall
	}{
		{name: "unknown", call: llm.ToolCall{ID: "1", Name: "missing", Arguments: map[string]any{}}},
		{name: "invalid arguments", call: llm.ToolCall{ID: "2", Name: "lookup_curriculum", Arguments: map[string]any{}}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			first := llm.AssistantMessage{Content: []llm.AssistantContent{tt.call}}
			final := llm.AssistantMessage{Content: []llm.AssistantContent{llm.TextContent{Text: "Recovered"}}}
			model := &scriptedModel{responses: []llm.AssistantMessage{first, final}}
			result, err := agentcore.Run(context.Background(), model, llm.Context{SystemPrompt: "Tutor"}, []agentcore.Tool{&memoryTool{}}, agentcore.Config{})
			if err != nil {
				t.Fatalf("Run() error = %v", err)
			}
			toolResult, ok := result.Messages[1].(llm.ToolResultMessage)
			if !ok || !toolResult.IsError || toolResult.ToolCallID != tt.call.ID || toolResult.ToolName != tt.call.Name {
				t.Fatalf("tool result = %#v", result.Messages[1])
			}
		})
	}
}

func TestRunExecutesToolCallsSequentially(t *testing.T) {
	firstCall := llm.ToolCall{ID: "1", Name: "lookup_curriculum", Arguments: map[string]any{"topic_id": "first"}}
	secondCall := llm.ToolCall{ID: "2", Name: "lookup_curriculum", Arguments: map[string]any{"topic_id": "second"}}
	tool := &memoryTool{}
	model := &scriptedModel{responses: []llm.AssistantMessage{
		{Content: []llm.AssistantContent{firstCall, secondCall}},
		{Content: []llm.AssistantContent{llm.TextContent{Text: "Done"}}},
	}}

	_, err := agentcore.Run(context.Background(), model, llm.Context{SystemPrompt: "Tutor"}, []agentcore.Tool{tool}, agentcore.Config{})
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if !reflect.DeepEqual(tool.calls, []llm.ToolCall{firstCall, secondCall}) {
		t.Fatalf("tool calls = %#v", tool.calls)
	}
}

func TestRunCancellationAndLimit(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	result, err := agentcore.Run(ctx, &scriptedModel{}, llm.Context{SystemPrompt: "Tutor"}, nil, agentcore.Config{})
	if !errors.Is(err, context.Canceled) || result.Termination != agentcore.TerminationCancelled {
		t.Fatalf("cancelled Run() = %#v, %v", result, err)
	}

	toolCall := llm.AssistantMessage{Content: []llm.AssistantContent{llm.ToolCall{ID: "1", Name: "lookup_curriculum", Arguments: map[string]any{"topic_id": "math-1"}}}}
	result, err = agentcore.Run(context.Background(), &scriptedModel{responses: []llm.AssistantMessage{toolCall}}, llm.Context{SystemPrompt: "Tutor"}, []agentcore.Tool{&memoryTool{}}, agentcore.Config{MaxModelCalls: 1})
	if !errors.Is(err, agentcore.ErrModelCallLimit) || result.Termination != agentcore.TerminationModelCallLimit {
		t.Fatalf("limited Run() = %#v, %v", result, err)
	}
}

func TestRunRejectsSystemMessages(t *testing.T) {
	_, err := agentcore.Run(context.Background(), &scriptedModel{}, llm.Context{
		SystemPrompt: "Tutor",
		Messages:     []llm.Message{llm.SystemMessage{Content: "duplicate"}},
	}, nil, agentcore.Config{})
	if err == nil {
		t.Fatal("Run() should reject system messages")
	}
}
