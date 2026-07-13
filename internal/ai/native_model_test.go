// Copyright 2026 the P&AI authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package ai

import (
	"context"
	"errors"
	"reflect"
	"testing"
	"time"

	"github.com/p-n-ai/pai-bot/internal/llm"
)

type nativeTestProvider struct {
	response llm.AssistantMessage
	err      error
	context  llm.Context
	calls    int
}

func (p *nativeTestProvider) CompleteNative(_ context.Context, _ string, c llm.Context, _ *llm.StreamOptions) (llm.AssistantMessage, error) {
	p.calls++
	p.context = c
	return p.response, p.err
}

func (*nativeTestProvider) Complete(context.Context, CompletionRequest) (CompletionResponse, error) {
	return CompletionResponse{}, errors.New("legacy completion should not run")
}

func (*nativeTestProvider) StreamComplete(context.Context, CompletionRequest) (<-chan StreamChunk, error) {
	return nil, errors.New("legacy stream should not run")
}

func (*nativeTestProvider) Models() []ModelInfo               { return nil }
func (*nativeTestProvider) HealthCheck(context.Context) error { return nil }

func TestNativeModelPreservesContextAndAssistantMessage(t *testing.T) {
	call := llm.ToolCall{ID: "call-1", Name: "lookup", Arguments: map[string]any{"topic_id": "F1-02"}}
	want := llm.AssistantMessage{
		Content:       []llm.AssistantContent{llm.TextContent{Text: "Checking."}, call},
		Provider:      "test",
		Model:         "model-a",
		ResponseModel: "model-a",
		StopReason:    llm.StopReasonToolUse,
	}
	provider := &nativeTestProvider{response: want}
	router := NewRouterWithConfig(RouterConfig{RetryBackoff: []time.Duration{0}})
	router.Register("test", provider)
	model := NewNativeModel(router, NativeModelConfig{Task: TaskTeaching, Model: "model-a"})
	input := llm.Context{
		SystemPrompt: "Tutor policy",
		Messages:     []llm.Message{llm.UserText("Help")},
		Tools:        []llm.Tool{{Name: "lookup", Parameters: []byte(`{"type":"object"}`)}},
	}

	got, err := model.Complete(context.Background(), input, &llm.StreamOptions{MaxTokens: 128})
	if err != nil {
		t.Fatalf("Complete() error = %v", err)
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("assistant = %#v, want %#v", got, want)
	}
	if !reflect.DeepEqual(provider.context, input) {
		t.Fatalf("native context = %#v, want %#v", provider.context, input)
	}
}

func TestNativeModelEmitsSanitizedTraceForNativeToolRequest(t *testing.T) {
	provider := &nativeTestProvider{response: llm.AssistantMessage{
		Content:       []llm.AssistantContent{llm.ToolCall{ID: "call-1", Name: "lookup", Arguments: map[string]any{"topic_id": "F1-02"}}},
		ResponseModel: "model-a",
		Usage:         llm.Usage{Input: 12, Output: 3},
	}}
	router := NewRouterWithConfig(RouterConfig{RetryBackoff: []time.Duration{0}})
	router.Register("native", provider)
	var traces []CompletionTrace
	router.SetTraceFunc(func(trace CompletionTrace) { traces = append(traces, trace) })
	model := NewNativeModel(router, NativeModelConfig{Task: TaskTeaching, Model: "model-a"})

	_, err := model.Complete(context.Background(), llm.Context{
		SystemPrompt: "Tutor policy",
		Messages:     []llm.Message{llm.UserText("Help")},
		Tools:        []llm.Tool{{Name: "lookup", Parameters: []byte(`{"type":"object"}`)}},
	}, &llm.StreamOptions{MaxTokens: 128})
	if err != nil {
		t.Fatalf("Complete() error = %v", err)
	}
	if len(traces) != 1 {
		t.Fatalf("trace count = %d, want 1", len(traces))
	}
	trace := traces[0]
	if trace.Provider != "native" || trace.Request.Model != "model-a" || trace.Request.Task != TaskTeaching || trace.Request.MaxTokens != 128 {
		t.Fatalf("trace routing = %#v", trace)
	}
	if trace.Response == nil || trace.Response.Model != "model-a" || trace.Response.InputTokens != 12 || trace.Response.OutputTokens != 3 {
		t.Fatalf("trace response = %#v", trace.Response)
	}
}

func TestNativeModelEmitsFailureTraceBeforeFallback(t *testing.T) {
	failing := &nativeTestProvider{err: errors.New("provider unavailable")}
	succeeding := &nativeTestProvider{response: llm.AssistantMessage{
		Content:       []llm.AssistantContent{llm.TextContent{Text: "fallback"}},
		ResponseModel: "model-b",
	}}
	router := NewRouterWithConfig(RouterConfig{RetryBackoff: []time.Duration{0}})
	router.Register("failing", failing)
	router.Register("succeeding", succeeding)
	var traces []CompletionTrace
	router.SetTraceFunc(func(trace CompletionTrace) { traces = append(traces, trace) })

	_, err := NewNativeModel(router, NativeModelConfig{Task: TaskTeaching}).Complete(
		context.Background(),
		llm.Context{SystemPrompt: "Tutor", Messages: []llm.Message{llm.UserText("Help")}},
		nil,
	)
	if err != nil {
		t.Fatalf("Complete() error = %v", err)
	}
	if len(traces) != 2 || traces[0].Provider != "failing" || traces[0].Error == "" || traces[0].Response != nil {
		t.Fatalf("failure trace = %#v", traces)
	}
	if traces[1].Provider != "succeeding" || traces[1].Response == nil || traces[1].Response.Content != "fallback" {
		t.Fatalf("fallback trace = %#v", traces[1])
	}
}

func TestNativeModelCancellationDoesNotFallbackOrTripCircuit(t *testing.T) {
	for _, cancelErr := range []error{context.Canceled, context.DeadlineExceeded} {
		t.Run(cancelErr.Error(), func(t *testing.T) {
			cancelled := &nativeTestProvider{err: cancelErr}
			fallback := &nativeTestProvider{response: llm.AssistantMessage{
				Content: []llm.AssistantContent{llm.TextContent{Text: "must not run"}},
			}}
			router := NewRouterWithConfig(RouterConfig{
				RetryBackoff:            []time.Duration{0},
				BreakerFailureThreshold: 1,
			})
			router.Register("cancelled", cancelled)
			router.Register("fallback", fallback)

			_, err := NewNativeModel(router, NativeModelConfig{Task: TaskTeaching}).Complete(
				context.Background(),
				llm.Context{SystemPrompt: "Tutor", Messages: []llm.Message{llm.UserText("Help")}},
				nil,
			)
			if !errors.Is(err, cancelErr) {
				t.Fatalf("Complete() error = %v, want %v", err, cancelErr)
			}
			if fallback.calls != 0 {
				t.Fatalf("fallback calls = %d, want 0", fallback.calls)
			}
			if router.isCircuitOpen("cancelled") {
				t.Fatal("cancellation must not open the provider circuit")
			}
		})
	}
}

func TestNativeModelDoesNotTripLegacyProviderCircuitForUnsupportedTools(t *testing.T) {
	legacy := NewMockProvider("legacy")
	native := &nativeTestProvider{response: llm.AssistantMessage{Content: []llm.AssistantContent{llm.TextContent{Text: "native"}}}}
	router := NewRouterWithConfig(RouterConfig{RetryBackoff: []time.Duration{0}, BreakerFailureThreshold: 1})
	router.Register("legacy", legacy)
	router.Register("native", native)
	model := NewNativeModel(router, NativeModelConfig{Task: TaskTeaching})

	_, err := model.Complete(context.Background(), llm.Context{
		SystemPrompt: "Tutor",
		Tools:        []llm.Tool{{Name: "lookup", Parameters: []byte(`{"type":"object"}`)}},
	}, nil)
	if err != nil {
		t.Fatalf("Complete() error = %v", err)
	}
	if router.isCircuitOpen("legacy") {
		t.Fatal("unsupported native tools must not count as a legacy provider failure")
	}
}
