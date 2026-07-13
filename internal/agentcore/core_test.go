// Copyright 2026 the P&AI authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package agentcore_test

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"reflect"
	"strings"
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
	err   error
}

type blockingTool struct {
	started chan struct{}
}

func (*blockingTool) Definition() llm.Tool {
	return llm.Tool{
		Name:       "blocking_tool",
		Parameters: []byte(`{"type":"object","additionalProperties":false}`),
	}
}

func (t *blockingTool) Execute(ctx context.Context, _ llm.ToolCall) (llm.ToolResultMessage, error) {
	close(t.started)
	<-ctx.Done()
	return llm.ToolResultMessage{}, ctx.Err()
}

func (*memoryTool) Definition() llm.Tool {
	return llm.Tool{
		Name:       "lookup_curriculum",
		Parameters: []byte(`{"type":"object","properties":{"topic_id":{"type":"string"}},"required":["topic_id"],"additionalProperties":false}`),
	}
}

func (t *memoryTool) Execute(_ context.Context, call llm.ToolCall) (llm.ToolResultMessage, error) {
	t.calls = append(t.calls, call)
	return llm.ToolResultMessage{Content: []llm.UserContent{llm.TextContent{Text: "Linear equations"}}}, t.err
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

func TestRunReturnsToolExecutionErrorToModel(t *testing.T) {
	call := llm.ToolCall{ID: "1", Name: "lookup_curriculum", Arguments: map[string]any{"topic_id": "math-1"}}
	model := &scriptedModel{responses: []llm.AssistantMessage{
		{Content: []llm.AssistantContent{call}},
		{Content: []llm.AssistantContent{llm.TextContent{Text: "Recovered"}}},
	}}
	tool := &memoryTool{err: errors.New("curriculum unavailable")}

	result, err := agentcore.Run(context.Background(), model, llm.Context{SystemPrompt: "Tutor"}, []agentcore.Tool{tool}, agentcore.Config{})
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	toolResult, ok := result.Messages[1].(llm.ToolResultMessage)
	if !ok || !toolResult.IsError || toolResult.ToolCallID != call.ID || toolResult.ToolName != call.Name {
		t.Fatalf("tool result = %#v", result.Messages[1])
	}
	if got := toolResult.Content[0].(llm.TextContent).Text; got != "tool execution failed" {
		t.Fatalf("tool error content = %q", got)
	}
}

func TestRunRecoversAfterRepeatedToolExecutionErrors(t *testing.T) {
	firstCall := llm.ToolCall{ID: "1", Name: "lookup_curriculum", Arguments: map[string]any{"topic_id": "math-1"}}
	secondCall := llm.ToolCall{ID: "2", Name: "lookup_curriculum", Arguments: map[string]any{"topic_id": "math-2"}}
	model := &scriptedModel{responses: []llm.AssistantMessage{
		{Content: []llm.AssistantContent{firstCall}},
		{Content: []llm.AssistantContent{secondCall}},
		{Content: []llm.AssistantContent{llm.TextContent{Text: "Recovered"}}},
	}}

	result, err := agentcore.Run(context.Background(), model, llm.Context{SystemPrompt: "Tutor"}, []agentcore.Tool{&memoryTool{err: errors.New("unavailable")}}, agentcore.Config{})
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	for _, index := range []int{1, 3} {
		toolResult, ok := result.Messages[index].(llm.ToolResultMessage)
		if !ok || !toolResult.IsError {
			t.Fatalf("message[%d] = %#v, want tool error", index, result.Messages[index])
		}
	}
	if result.Termination != agentcore.TerminationCompleted || result.ModelCalls != 3 {
		t.Fatalf("termination = %q calls = %d", result.Termination, result.ModelCalls)
	}
}

func TestRunCancelsDuringToolExecution(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	tool := &blockingTool{started: make(chan struct{})}
	go func() {
		<-tool.started
		cancel()
	}()
	call := llm.ToolCall{ID: "1", Name: "blocking_tool", Arguments: map[string]any{}}
	model := &scriptedModel{responses: []llm.AssistantMessage{{Content: []llm.AssistantContent{call}}}}

	result, err := agentcore.Run(ctx, model, llm.Context{SystemPrompt: "Tutor"}, []agentcore.Tool{tool}, agentcore.Config{})
	if !errors.Is(err, context.Canceled) || result.Termination != agentcore.TerminationCancelled {
		t.Fatalf("Run() = %#v, %v", result, err)
	}
	toolResult, ok := result.Messages[1].(llm.ToolResultMessage)
	if !ok || !toolResult.IsError || toolResult.ToolCallID != call.ID {
		t.Fatalf("tool result = %#v", result.Messages[1])
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

func TestRunLogsPayloadSafeTerminationReason(t *testing.T) {
	var logs bytes.Buffer
	previous := slog.Default()
	slog.SetDefault(slog.New(slog.NewJSONHandler(&logs, &slog.HandlerOptions{Level: slog.LevelDebug})))
	t.Cleanup(func() { slog.SetDefault(previous) })

	_, err := agentcore.Run(context.Background(), &scriptedModel{responses: []llm.AssistantMessage{{
		Content: []llm.AssistantContent{llm.TextContent{Text: "private answer"}},
	}}}, llm.Context{
		SystemPrompt: "private system prompt",
		Messages:     []llm.Message{llm.UserText("private learner text")},
	}, nil, agentcore.Config{RunID: "run-1", ConversationID: "conversation-1"})
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	got := logs.String()
	if !strings.Contains(got, `"termination":"completed"`) {
		t.Fatalf("logs = %s, want completed termination", got)
	}
	for _, private := range []string{"private answer", "private system prompt", "private learner text"} {
		if strings.Contains(got, private) {
			t.Fatalf("logs contain private payload %q: %s", private, got)
		}
	}
}
