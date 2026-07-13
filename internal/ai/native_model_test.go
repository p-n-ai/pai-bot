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
}

func (p *nativeTestProvider) CompleteNative(_ context.Context, _ string, c llm.Context, _ *llm.StreamOptions) (llm.AssistantMessage, error) {
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
