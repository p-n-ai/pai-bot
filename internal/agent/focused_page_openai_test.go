// Copyright 2026 the P&AI authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package agent

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/p-n-ai/pai-bot/internal/ai"
	"github.com/p-n-ai/pai-bot/internal/chat"
	"github.com/p-n-ai/pai-bot/internal/focusedpage"
)

func TestDirectOpenAIFocusedPageTurnContinuesAfterToolResult(t *testing.T) {
	var calls atomic.Int32
	var continuation map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var request map[string]any
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			t.Errorf("decode request: %v", err)
			http.Error(w, "bad request", http.StatusBadRequest)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		if calls.Add(1) == 1 {
			_, _ = w.Write([]byte(`{"id":"oa-tool","model":"gpt-test","choices":[{"message":{"tool_calls":[{"id":"page-1","type":"function","function":{"name":"create_focused_page","arguments":"{\"message\":\"You completed your goal report.\"}"}}]},"finish_reason":"tool_calls"}]}`))
			return
		}
		continuation = request
		_, _ = w.Write([]byte(`{"id":"oa-final","model":"gpt-test","choices":[{"message":{"content":"I made a focused page for your report."},"finish_reason":"stop"}]}`))
	}))
	t.Cleanup(server.Close)

	store := NewMemoryStore()
	if err := store.SetUserName("learner-1", "Aina"); err != nil {
		t.Fatal(err)
	}
	pageService, err := focusedpage.NewService(
		focusedpage.NewMemoryStore(),
		"https://pages.example",
		[]byte("0123456789abcdef0123456789abcdef"),
		time.Now,
	)
	if err != nil {
		t.Fatal(err)
	}
	router := ai.NewRouterWithConfig(ai.RouterConfig{RetryBackoff: []time.Duration{time.Millisecond}})
	router.Register("openai", ai.NewOpenAIProvider("test-key", ai.WithBaseURL(server.URL)))
	engine := NewEngine(EngineConfig{
		AIRouter:           router,
		Store:              store,
		TenantID:           "tenant-1",
		FocusedPages:       pageService,
		FocusedPageEnabled: telegramFocusedPageEnabled,
	})

	result, err := engine.ProcessTurn(context.Background(), chat.InboundMessage{
		Channel: "telegram",
		UserID:  "learner-1",
		Text:    "Give me a report on my goal",
	})
	if err != nil {
		t.Fatalf("ProcessTurn() error = %v", err)
	}
	if result.Text != "I made a focused page for your report." || result.FocusedPage == nil {
		t.Fatalf("text = %q, focused page present = %t", result.Text, result.FocusedPage != nil)
	}
	if calls.Load() != 2 {
		t.Fatalf("OpenAI calls = %d, want 2", calls.Load())
	}
	assertOpenAIFocusedPageContinuation(t, continuation)
}

func assertOpenAIFocusedPageContinuation(t *testing.T, request map[string]any) {
	t.Helper()
	messages := request["messages"].([]any)
	var assistant, toolResult map[string]any
	for _, raw := range messages {
		message := raw.(map[string]any)
		switch message["role"] {
		case "assistant":
			assistant = message
		case "tool":
			toolResult = message
		}
	}
	if assistant == nil || toolResult == nil {
		t.Fatalf("continuation messages = %#v", messages)
	}
	call := assistant["tool_calls"].([]any)[0].(map[string]any)
	function := call["function"].(map[string]any)
	if call["id"] != "page-1" || function["name"] != "create_focused_page" {
		t.Fatalf("assistant tool call = %#v", call)
	}
	if toolResult["tool_call_id"] != "page-1" || toolResult["content"] != "Focused page created." {
		t.Fatalf("tool result = %#v", toolResult)
	}
}
