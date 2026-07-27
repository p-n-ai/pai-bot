// Copyright 2026 the P&AI authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package ai

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/p-n-ai/pai-bot/internal/llm"
)

func TestCodexProviderCompleteUsesCodexResponsesProtocol(t *testing.T) {
	var captured map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/codex/responses" {
			t.Errorf("path = %q", r.URL.Path)
		}
		assertCodexHeaders(t, r, "access-token", "account-123")
		if err := json.NewDecoder(r.Body).Decode(&captured); err != nil {
			t.Errorf("decode request: %v", err)
		}
		writeCodexSSE(w,
			`{"type":"response.output_item.added","output_index":0,"item":{"type":"message","id":"msg-1","content":[]}}`,
			`{"type":"response.output_text.delta","output_index":0,"delta":"{\"answer\":"}`,
			`{"type":"response.output_text.delta","output_index":0,"delta":"\"Use the diagram.\"}"}`,
			`{"type":"response.output_item.done","output_index":0,"item":{"type":"message","id":"msg-1","content":[{"type":"output_text","text":"{\"answer\":\"Use the diagram.\"}"}]}}`,
			`{"type":"response.completed","response":{"id":"resp-1","model":"gpt-codex-test","status":"completed","usage":{"input_tokens":12,"output_tokens":4,"total_tokens":16,"input_tokens_details":{"cached_tokens":3}}}}`,
		)
	}))
	t.Cleanup(server.Close)

	provider, err := NewCodexProvider(
		"access-token",
		WithCodexAccountID("account-123"),
		WithCodexBaseURL(server.URL),
		WithCodexHTTPClient(server.Client()),
	)
	if err != nil {
		t.Fatalf("NewCodexProvider() error = %v", err)
	}
	response, err := provider.Complete(context.Background(), CompletionRequest{
		Model: "gpt-codex-test",
		Messages: []Message{
			{Role: "system", Content: "Tutor policy"},
			{Role: "user", Content: "Explain this", ImageURLs: []string{"https://example.test/diagram.png"}},
		},
		MaxTokens:   64,
		Temperature: 0.25,
		StructuredOutput: &StructuredOutputSpec{
			Name:       "tutor_response",
			JSONSchema: json.RawMessage(`{"type":"object","properties":{"answer":{"type":"string"}},"required":["answer"],"additionalProperties":false}`),
			Strict:     true,
		},
	})
	if err != nil {
		t.Fatalf("Complete() error = %v", err)
	}
	if response.Content != `{"answer":"Use the diagram."}` || string(response.StructuredOutput) != response.Content {
		t.Fatalf("response = %#v", response)
	}
	if response.Model != "gpt-codex-test" || response.InputTokens != 12 || response.OutputTokens != 4 {
		t.Fatalf("response metadata = %#v", response)
	}

	if captured["store"] != false || captured["stream"] != true || captured["instructions"] != "Tutor policy" {
		t.Fatalf("request flags = %#v", captured)
	}
	if captured["max_output_tokens"] != float64(64) || captured["temperature"] != 0.25 {
		t.Fatalf("request generation options = %#v", captured)
	}
	text := captured["text"].(map[string]any)
	format := text["format"].(map[string]any)
	if text["verbosity"] != "low" || format["type"] != "json_schema" || format["name"] != "tutor_response" || format["strict"] != true {
		t.Fatalf("text options = %#v", text)
	}
	input := captured["input"].([]any)
	if len(input) != 1 {
		t.Fatalf("input = %#v", input)
	}
	content := input[0].(map[string]any)["content"].([]any)
	if content[0].(map[string]any)["type"] != "input_text" ||
		content[1].(map[string]any)["type"] != "input_image" ||
		content[1].(map[string]any)["image_url"] != "https://example.test/diagram.png" {
		t.Fatalf("multimodal content = %#v", content)
	}
}

func TestCodexProviderCompleteNativePreservesToolContinuation(t *testing.T) {
	var mu sync.Mutex
	var requests []map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var request map[string]any
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			t.Errorf("decode request: %v", err)
		}
		mu.Lock()
		requests = append(requests, request)
		requestNumber := len(requests)
		mu.Unlock()
		if requestNumber == 1 {
			writeCodexSSE(w,
				`{"type":"response.output_item.done","output_index":0,"item":{"type":"reasoning","id":"rs-1","status":"completed","summary":[{"type":"summary_text","text":"I should create the focused page."}]}}`,
				`{"type":"response.output_item.added","output_index":1,"item":{"type":"function_call","id":"fc-1","call_id":"call-page-1","name":"create_focused_page","arguments":""}}`,
				`{"type":"response.function_call_arguments.delta","output_index":1,"delta":"{\"message\":\"Steady progress.\"}"}`,
				`{"type":"response.output_item.done","output_index":1,"item":{"type":"function_call","id":"fc-1","call_id":"call-page-1","name":"create_focused_page","arguments":"{\"message\":\"Steady progress.\"}"}}`,
				`{"type":"response.completed","response":{"id":"resp-tool","model":"gpt-codex-test","status":"completed","output":[{"type":"reasoning","id":"rs-1","encrypted_content":"encrypted-reasoning"},{"type":"function_call","id":"fc-1","call_id":"call-page-1","name":"create_focused_page","arguments":"{\"message\":\"Steady progress.\"}"}],"usage":{"input_tokens":15,"output_tokens":5,"total_tokens":20,"input_tokens_details":{"cached_tokens":4,"cache_write_tokens":2}}}}`,
			)
			return
		}
		writeCodexSSE(w,
			`{"type":"response.output_item.done","output_index":0,"item":{"type":"message","id":"msg-final","content":[{"type":"output_text","text":"Page ready."}]}}`,
			`{"type":"response.completed","response":{"id":"resp-final","model":"gpt-codex-test","status":"completed","usage":{"input_tokens":20,"output_tokens":3}}}`,
		)
	}))
	t.Cleanup(server.Close)

	provider, err := NewCodexProvider("opaque-token", WithCodexAccountID("account-123"), WithCodexBaseURL(server.URL))
	if err != nil {
		t.Fatalf("NewCodexProvider() error = %v", err)
	}
	tool := llm.Tool{
		Name:        "create_focused_page",
		Description: "Create one focused page.",
		Parameters:  json.RawMessage(`{"type":"object","properties":{"message":{"type":"string"}},"required":["message"],"additionalProperties":false}`),
	}
	initial := llm.Context{
		SystemPrompt: "Tutor policy",
		Messages: []llm.Message{llm.UserMessage{Content: []llm.UserContent{
			llm.TextContent{Text: "Show progress"},
			llm.ImageContent{MimeType: "image/png", Data: "AAEC"},
		}}},
		Tools: []llm.Tool{tool},
	}
	first, err := provider.CompleteNative(context.Background(), "gpt-codex-test", initial, &llm.StreamOptions{MaxTokens: 128})
	if err != nil {
		t.Fatalf("first CompleteNative() error = %v", err)
	}
	call := requireContractToolCall(t, first)
	if call.ID != "call-page-1|fc-1" || call.Name != tool.Name || call.Arguments["message"] != "Steady progress." {
		t.Fatalf("tool call = %#v", call)
	}
	if len(first.Content) != 2 {
		t.Fatalf("first content = %#v", first.Content)
	}
	thinking, ok := first.Content[0].(llm.ThinkingContent)
	if !ok || thinking.Thinking != "I should create the focused page." {
		t.Fatalf("thinking = %#v", first.Content[0])
	}
	var reasoningSignature map[string]any
	if err := json.Unmarshal([]byte(thinking.Signature), &reasoningSignature); err != nil {
		t.Fatalf("decode reasoning signature: %v", err)
	}
	if reasoningSignature["type"] != "reasoning" || reasoningSignature["id"] != "rs-1" ||
		reasoningSignature["encrypted_content"] != "encrypted-reasoning" ||
		reasoningSignature["status"] != "completed" {
		t.Fatalf("reasoning signature = %#v", reasoningSignature)
	}
	if first.StopReason != llm.StopReasonToolUse ||
		first.Usage.Input != 9 ||
		first.Usage.CacheRead != 4 ||
		first.Usage.CacheWrite != 2 ||
		first.Usage.Output != 5 ||
		first.Usage.TotalTokens != 20 {
		t.Fatalf("first metadata = %#v", first)
	}

	continuation := llm.Context{
		SystemPrompt: initial.SystemPrompt,
		Messages: append(append([]llm.Message(nil), initial.Messages...),
			first,
			llm.ToolResultMessage{
				ToolCallID: call.ID,
				ToolName:   call.Name,
				Content:    []llm.UserContent{llm.TextContent{Text: "Created."}},
			},
		),
		Tools: []llm.Tool{tool},
	}
	final, err := provider.CompleteNative(context.Background(), "gpt-codex-test", continuation, nil)
	if err != nil {
		t.Fatalf("continuation CompleteNative() error = %v", err)
	}
	if contractAssistantText(final) != "Page ready." {
		t.Fatalf("final = %#v", final)
	}

	mu.Lock()
	captured := append([]map[string]any(nil), requests...)
	mu.Unlock()
	tools := captured[0]["tools"].([]any)
	if tools[0].(map[string]any)["name"] != tool.Name || tools[0].(map[string]any)["strict"] != nil {
		t.Fatalf("tools = %#v", tools)
	}
	userContent := captured[0]["input"].([]any)[0].(map[string]any)["content"].([]any)
	if userContent[1].(map[string]any)["image_url"] != "data:image/png;base64,AAEC" {
		t.Fatalf("image = %#v", userContent[1])
	}
	input := captured[1]["input"].([]any)
	var reasoning, functionCall, functionOutput map[string]any
	var reasoningIndex, functionCallIndex int
	for index, raw := range input {
		item := raw.(map[string]any)
		switch item["type"] {
		case "reasoning":
			reasoning = item
			reasoningIndex = index
		case "function_call":
			functionCall = item
			functionCallIndex = index
		case "function_call_output":
			functionOutput = item
		}
	}
	if reasoning["id"] != "rs-1" || reasoning["encrypted_content"] != "encrypted-reasoning" || reasoningIndex >= functionCallIndex {
		t.Fatalf("reasoning replay = %#v at %d; function call at %d", reasoning, reasoningIndex, functionCallIndex)
	}
	if functionCall["call_id"] != "call-page-1" || functionCall["id"] != "fc-1" || functionCall["name"] != tool.Name {
		t.Fatalf("function call replay = %#v", functionCall)
	}
	if functionOutput["call_id"] != "call-page-1" || functionOutput["output"] != "Created." {
		t.Fatalf("function output replay = %#v", functionOutput)
	}
}

func TestCodexProviderRejectsMalformedReasoningSignatures(t *testing.T) {
	var requests atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		requests.Add(1)
		http.Error(w, "unexpected request", http.StatusInternalServerError)
	}))
	t.Cleanup(server.Close)
	provider, err := NewCodexProvider("opaque-token", WithCodexAccountID("account-123"), WithCodexBaseURL(server.URL))
	if err != nil {
		t.Fatalf("NewCodexProvider() error = %v", err)
	}

	for name, signature := range map[string]string{
		"invalid JSON":    "{not-json",
		"wrong item type": `{"type":"message","id":"msg-1"}`,
		"missing item ID": `{"type":"reasoning"}`,
	} {
		t.Run(name, func(t *testing.T) {
			_, err := provider.CompleteNative(context.Background(), "gpt-codex-test", llm.Context{
				Messages: []llm.Message{llm.AssistantMessage{
					Content: []llm.AssistantContent{llm.ThinkingContent{
						Thinking:  "prior reasoning",
						Signature: signature,
					}},
				}},
			}, nil)
			if err == nil {
				t.Fatal("CompleteNative() should reject malformed reasoning signature")
			}
		})
	}
	if requests.Load() != 0 {
		t.Fatalf("HTTP requests = %d, want 0", requests.Load())
	}
}

func TestCodexProviderRefreshesExpiredTokenOnceConcurrently(t *testing.T) {
	expired := codexTestJWT(t, "account-old", time.Now().Add(-time.Hour))
	refreshed := codexTestJWT(t, "account-new", time.Now().Add(time.Hour))
	var refreshes atomic.Int32
	var completions atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/oauth/token":
			refreshes.Add(1)
			if err := r.ParseForm(); err != nil {
				t.Errorf("parse refresh form: %v", err)
			}
			if r.Form.Get("grant_type") != "refresh_token" || r.Form.Get("refresh_token") != "refresh-old" || r.Form.Get("client_id") != codexOAuthClientID {
				t.Errorf("refresh form = %v", r.Form)
			}
			time.Sleep(30 * time.Millisecond)
			_ = json.NewEncoder(w).Encode(map[string]any{
				"access_token":  refreshed,
				"refresh_token": "refresh-new",
				"expires_in":    3600,
			})
		case "/codex/responses":
			completions.Add(1)
			assertCodexHeaders(t, r, refreshed, "account-new")
			writeCodexSSE(w, `{"type":"response.completed","response":{"id":"resp","model":"gpt-codex-test","status":"completed","usage":{"input_tokens":1,"output_tokens":1}}}`)
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(server.Close)

	provider, err := NewCodexProvider(
		expired,
		WithCodexRefreshToken("refresh-old"),
		WithCodexBaseURL(server.URL),
		WithCodexAuthBaseURL(server.URL),
	)
	if err != nil {
		t.Fatalf("NewCodexProvider() error = %v", err)
	}
	const workers = 8
	start := make(chan struct{})
	errs := make(chan error, workers)
	for range workers {
		go func() {
			<-start
			_, err := provider.Complete(context.Background(), CompletionRequest{
				Model:    "gpt-codex-test",
				Messages: []Message{{Role: "user", Content: "Hi"}},
			})
			errs <- err
		}()
	}
	close(start)
	for range workers {
		if err := <-errs; err != nil {
			t.Fatalf("Complete() error = %v", err)
		}
	}
	if refreshes.Load() != 1 || completions.Load() != workers {
		t.Fatalf("refreshes = %d, completions = %d", refreshes.Load(), completions.Load())
	}
}

func TestCodexProviderRefreshesAndRetriesAfterAuthRejection(t *testing.T) {
	oldToken := codexTestJWT(t, "account-old", time.Now().Add(time.Hour))
	newToken := codexTestJWT(t, "account-new", time.Now().Add(2*time.Hour))
	var refreshes atomic.Int32
	var requests atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/oauth/token":
			refreshes.Add(1)
			_ = json.NewEncoder(w).Encode(map[string]any{
				"access_token":  newToken,
				"refresh_token": "refresh-new",
				"expires_in":    7200,
			})
		case "/codex/responses":
			requests.Add(1)
			if r.Header.Get("Authorization") == "Bearer "+oldToken {
				http.Error(w, "expired", http.StatusUnauthorized)
				return
			}
			assertCodexHeaders(t, r, newToken, "account-new")
			writeCodexSSE(w, `{"type":"response.completed","response":{"id":"resp","model":"gpt-codex-test","status":"completed"}}`)
		}
	}))
	t.Cleanup(server.Close)

	provider, err := NewCodexProvider(
		oldToken,
		WithCodexRefreshToken("refresh-old"),
		WithCodexBaseURL(server.URL),
		WithCodexAuthBaseURL(server.URL),
	)
	if err != nil {
		t.Fatalf("NewCodexProvider() error = %v", err)
	}
	if _, err := provider.Complete(context.Background(), CompletionRequest{Messages: []Message{{Role: "user", Content: "Hi"}}}); err != nil {
		t.Fatalf("Complete() error = %v", err)
	}
	if refreshes.Load() != 1 || requests.Load() != 2 {
		t.Fatalf("refreshes = %d, requests = %d", refreshes.Load(), requests.Load())
	}
}

func TestCodexProviderStreamCompleteEmitsSSEDeltas(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		writeCodexSSE(w,
			`{"type":"response.output_text.delta","output_index":0,"delta":"Hello"}`,
			`{"type":"response.output_text.delta","output_index":0,"delta":" world"}`,
			`{"type":"response.completed","response":{"id":"resp","model":"gpt-codex-test","status":"completed"}}`,
		)
	}))
	t.Cleanup(server.Close)
	provider, err := NewCodexProvider("opaque", WithCodexAccountID("account-123"), WithCodexBaseURL(server.URL))
	if err != nil {
		t.Fatalf("NewCodexProvider() error = %v", err)
	}
	chunks, err := provider.StreamComplete(context.Background(), CompletionRequest{Messages: []Message{{Role: "user", Content: "Hi"}}})
	if err != nil {
		t.Fatalf("StreamComplete() error = %v", err)
	}
	var text strings.Builder
	var done bool
	for chunk := range chunks {
		if chunk.Error != nil {
			t.Fatalf("stream error = %v", chunk.Error)
		}
		text.WriteString(chunk.Content)
		done = done || chunk.Done
	}
	if text.String() != "Hello world" || !done {
		t.Fatalf("text = %q, done = %v", text.String(), done)
	}
}

func assertCodexHeaders(t *testing.T, r *http.Request, token, accountID string) {
	t.Helper()
	want := map[string]string{
		"Authorization":      "Bearer " + token,
		"chatgpt-account-id": accountID,
		"originator":         "pi",
		"OpenAI-Beta":        "responses=experimental",
		"Accept":             "text/event-stream",
	}
	for name, value := range want {
		if got := r.Header.Get(name); got != value {
			t.Errorf("%s = %q, want %q", name, got, value)
		}
	}
}

func writeCodexSSE(w http.ResponseWriter, events ...string) {
	w.Header().Set("Content-Type", "text/event-stream")
	for _, event := range events {
		_, _ = fmt.Fprintf(w, "data: %s\n\n", event)
	}
}

func codexTestJWT(t *testing.T, accountID string, expiresAt time.Time) string {
	t.Helper()
	header, _ := json.Marshal(map[string]any{"alg": "none"})
	payload, _ := json.Marshal(map[string]any{
		"https://api.openai.com/auth": map[string]any{"chatgpt_account_id": accountID},
		"exp":                         expiresAt.Unix(),
	})
	return base64.RawURLEncoding.EncodeToString(header) + "." +
		base64.RawURLEncoding.EncodeToString(payload) + ".signature"
}
