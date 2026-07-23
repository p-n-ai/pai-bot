// Copyright 2026 the P&AI authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package ai

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/p-n-ai/pai-bot/internal/llm"
)

type nativeProviderContractFixture struct {
	name       string
	model      string
	new        func(string) NativeProvider
	writeReply func(http.ResponseWriter, string)
}

func nativeProviderContractFixtures() []nativeProviderContractFixture {
	return []nativeProviderContractFixture{
		{
			name:  "direct OpenAI",
			model: "gpt-test",
			new: func(baseURL string) NativeProvider {
				return NewOpenAIProvider("test-key", WithBaseURL(baseURL)).(NativeProvider)
			},
			writeReply: writeDirectOpenAIContractReply,
		},
		{
			name:  "OpenRouter",
			model: "openai/gpt-test",
			new: func(baseURL string) NativeProvider {
				return newOpenRouterLLMAdapter("test-key", baseURL)
			},
			writeReply: writeOpenRouterContractReply,
		},
	}
}

func TestNativeProviderContinuationContract(t *testing.T) {
	for _, fixture := range nativeProviderContractFixtures() {
		t.Run(fixture.name, func(t *testing.T) {
			var mu sync.Mutex
			var requests []map[string]any
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				var request map[string]any
				if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
					t.Errorf("decode request: %v", err)
					http.Error(w, "bad request", http.StatusBadRequest)
					return
				}
				mu.Lock()
				requests = append(requests, request)
				requestNumber := len(requests)
				mu.Unlock()
				if requestNumber == 1 {
					fixture.writeReply(w, "tool")
					return
				}
				fixture.writeReply(w, "final")
			}))
			t.Cleanup(server.Close)

			provider := fixture.new(server.URL)
			tool := llm.Tool{
				Name:        "create_focused_page",
				Description: "Create one focused page.",
				Parameters:  json.RawMessage(`{"type":"object","additionalProperties":false,"properties":{"message":{"type":"string"}},"required":["message"]}`),
			}
			initial := llm.Context{
				SystemPrompt: "Tutor policy",
				Messages:     []llm.Message{llm.UserText("Show my goal report")},
				Tools:        []llm.Tool{tool},
			}
			first, err := provider.CompleteNative(context.Background(), fixture.model, initial, &llm.StreamOptions{MaxTokens: 128})
			if err != nil {
				t.Fatalf("first CompleteNative() error = %v", err)
			}
			call := requireContractToolCall(t, first)
			if call.ID != "call-page-1" || call.Name != "create_focused_page" || call.Arguments["message"] != "You are making steady progress." {
				t.Fatalf("tool call = %#v", call)
			}

			continuation := llm.Context{
				SystemPrompt: initial.SystemPrompt,
				Messages: append(append([]llm.Message(nil), initial.Messages...),
					first,
					llm.ToolResultMessage{
						ToolCallID: call.ID,
						ToolName:   call.Name,
						Content:    []llm.UserContent{llm.TextContent{Text: "Focused page created."}},
					},
				),
				Tools: []llm.Tool{tool},
			}
			final, err := provider.CompleteNative(context.Background(), fixture.model, continuation, &llm.StreamOptions{MaxTokens: 128})
			if err != nil {
				t.Fatalf("continuation CompleteNative() error = %v", err)
			}
			if text := contractAssistantText(final); text != "Your focused page is ready." {
				t.Fatalf("final text = %q", text)
			}

			mu.Lock()
			captured := append([]map[string]any(nil), requests...)
			mu.Unlock()
			if len(captured) != 2 {
				t.Fatalf("request count = %d, want 2", len(captured))
			}
			assertContractToolDefinition(t, captured[0])
			assertContractContinuation(t, captured[1])
		})
	}
}

func TestNativeProviderRejectsMalformedToolArguments(t *testing.T) {
	for _, fixture := range nativeProviderContractFixtures() {
		t.Run(fixture.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				fixture.writeReply(w, "malformed")
			}))
			t.Cleanup(server.Close)

			_, err := fixture.new(server.URL).CompleteNative(
				context.Background(),
				fixture.model,
				llm.Context{Messages: []llm.Message{llm.UserText("Create a page")}},
				nil,
			)
			if err == nil {
				t.Fatal("CompleteNative() should reject malformed tool arguments")
			}
		})
	}
}

func TestNativeProviderToolFreeContract(t *testing.T) {
	for _, fixture := range nativeProviderContractFixtures() {
		t.Run(fixture.name, func(t *testing.T) {
			var captured map[string]any
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if err := json.NewDecoder(r.Body).Decode(&captured); err != nil {
					t.Errorf("decode request: %v", err)
				}
				fixture.writeReply(w, "plain")
			}))
			t.Cleanup(server.Close)

			temperature := 0.25
			response, err := fixture.new(server.URL).CompleteNative(
				context.Background(),
				fixture.model,
				llm.Context{
					SystemPrompt: "Tutor policy",
					Messages: []llm.Message{llm.UserMessage{Content: []llm.UserContent{
						llm.TextContent{Text: "Explain this image"},
						llm.ImageContent{MimeType: "image/png", Data: "AAEC"},
					}}},
				},
				&llm.StreamOptions{
					MaxTokens:   64,
					Temperature: &temperature,
					StructuredOutput: &llm.StructuredOutputSpec{
						Name:       "tutor_response",
						JSONSchema: json.RawMessage(`{"type":"object","properties":{"answer":{"type":"string"}},"required":["answer"],"additionalProperties":false}`),
						Strict:     true,
					},
				},
			)
			if err != nil {
				t.Fatalf("CompleteNative() error = %v", err)
			}
			if text := contractAssistantText(response); text != `{"answer":"Use the diagram."}` {
				t.Fatalf("tool-free text = %q", text)
			}
			if _, present := captured["tools"]; present {
				t.Fatalf("tool-free request contains tools: %#v", captured["tools"])
			}
			assertContractImageAndStructuredOutput(t, captured)
		})
	}
}

func TestDeepSeekProviderDoesNotImplementNativeContract(t *testing.T) {
	if _, ok := any(NewDeepSeekProvider("test-key")).(NativeProvider); ok {
		t.Fatal("DeepSeek must remain outside the direct OpenAI native contract")
	}
	if _, ok := NewOpenAIProvider("test-key", WithProviderName("compatible")).(NativeProvider); ok {
		t.Fatal("arbitrary OpenAI-compatible providers must remain outside the direct OpenAI native contract")
	}
}

func requireContractToolCall(t *testing.T, message llm.AssistantMessage) llm.ToolCall {
	t.Helper()
	for _, content := range message.Content {
		if call, ok := content.(llm.ToolCall); ok {
			return call
		}
	}
	t.Fatalf("assistant content has no tool call: %#v", message.Content)
	return llm.ToolCall{}
}

func contractAssistantText(message llm.AssistantMessage) string {
	var text strings.Builder
	for _, content := range message.Content {
		if block, ok := content.(llm.TextContent); ok {
			text.WriteString(block.Text)
		}
	}
	return text.String()
}

func assertContractToolDefinition(t *testing.T, request map[string]any) {
	t.Helper()
	tools, ok := request["tools"].([]any)
	if !ok || len(tools) != 1 {
		t.Fatalf("tools = %#v", request["tools"])
	}
	function := tools[0].(map[string]any)["function"].(map[string]any)
	parameters := function["parameters"].(map[string]any)
	if function["name"] != "create_focused_page" || function["description"] != "Create one focused page." || parameters["type"] != "object" {
		t.Fatalf("tool definition = %#v", function)
	}
}

func assertContractContinuation(t *testing.T, request map[string]any) {
	t.Helper()
	messages := request["messages"].([]any)
	var assistant, tool map[string]any
	for _, raw := range messages {
		message := raw.(map[string]any)
		switch message["role"] {
		case "assistant":
			assistant = message
		case "tool":
			tool = message
		}
	}
	if assistant == nil || tool == nil {
		t.Fatalf("continuation messages = %#v", messages)
	}
	calls := assistant["tool_calls"].([]any)
	call := calls[0].(map[string]any)
	function := call["function"].(map[string]any)
	var arguments map[string]any
	if err := json.Unmarshal([]byte(function["arguments"].(string)), &arguments); err != nil {
		t.Fatalf("decode assistant tool arguments: %v", err)
	}
	if call["id"] != "call-page-1" || function["name"] != "create_focused_page" || arguments["message"] != "You are making steady progress." {
		t.Fatalf("assistant tool call = %#v", call)
	}
	if tool["tool_call_id"] != "call-page-1" || tool["content"] != "Focused page created." {
		t.Fatalf("tool result = %#v", tool)
	}
}

func assertContractImageAndStructuredOutput(t *testing.T, request map[string]any) {
	t.Helper()
	messages := request["messages"].([]any)
	user := messages[1].(map[string]any)
	parts := user["content"].([]any)
	image := parts[1].(map[string]any)["image_url"].(map[string]any)
	if image["url"] != "data:image/png;base64,AAEC" {
		t.Fatalf("image = %#v", image)
	}
	responseFormat := request["response_format"].(map[string]any)
	jsonSchema := responseFormat["json_schema"].(map[string]any)
	if responseFormat["type"] != "json_schema" || jsonSchema["name"] != "tutor_response" || jsonSchema["strict"] != true {
		t.Fatalf("response_format = %#v", responseFormat)
	}
}

func writeDirectOpenAIContractReply(w http.ResponseWriter, kind string) {
	w.Header().Set("Content-Type", "application/json")
	switch kind {
	case "tool":
		_, _ = w.Write([]byte(`{"id":"oa-tool","model":"gpt-test","choices":[{"message":{"content":"Checking.","tool_calls":[{"id":"call-page-1","type":"function","function":{"name":"create_focused_page","arguments":"{\"message\":\"You are making steady progress.\"}"}}]},"finish_reason":"tool_calls"}],"usage":{"prompt_tokens":12,"completion_tokens":4}}`))
	case "final":
		_, _ = w.Write([]byte(`{"id":"oa-final","model":"gpt-test","choices":[{"message":{"content":"Your focused page is ready."},"finish_reason":"stop"}],"usage":{"prompt_tokens":20,"completion_tokens":6}}`))
	case "malformed":
		_, _ = w.Write([]byte(`{"id":"oa-bad","model":"gpt-test","choices":[{"message":{"tool_calls":[{"id":"call-bad","type":"function","function":{"name":"create_focused_page","arguments":"{not-json"}}]},"finish_reason":"tool_calls"}]}`))
	case "plain":
		_, _ = w.Write([]byte(`{"id":"oa-plain","model":"gpt-test","choices":[{"message":{"content":"{\"answer\":\"Use the diagram.\"}"},"finish_reason":"stop"}]}`))
	}
}

func writeOpenRouterContractReply(w http.ResponseWriter, kind string) {
	switch kind {
	case "tool":
		writeOpenRouterLLMStream(w, `{"id":"or-tool","model":"openai/gpt-test","choices":[{"delta":{"content":"Checking.","tool_calls":[{"index":0,"id":"call-page-1","type":"function","function":{"name":"create_focused_page","arguments":"{\"message\":\"You are making steady progress.\"}"}}]},"finish_reason":"tool_calls"}]}`)
	case "final":
		writeOpenRouterLLMStream(w, `{"id":"or-final","model":"openai/gpt-test","choices":[{"delta":{"content":"Your focused page is ready."},"finish_reason":"stop"}]}`)
	case "malformed":
		writeOpenRouterLLMStream(w, `{"id":"or-bad","model":"openai/gpt-test","choices":[{"delta":{"tool_calls":[{"index":0,"id":"call-bad","type":"function","function":{"name":"create_focused_page","arguments":"{not-json"}}]},"finish_reason":"tool_calls"}]}`)
	case "plain":
		writeOpenRouterLLMStream(w, `{"id":"or-plain","model":"openai/gpt-test","choices":[{"delta":{"content":"{\"answer\":\"Use the diagram.\"}"},"finish_reason":"stop"}]}`)
	}
}
