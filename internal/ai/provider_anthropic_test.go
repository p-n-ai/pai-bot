// Copyright 2026 the P&AI authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package ai

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestNewAnthropicProvider_EmptyKey(t *testing.T) {
	_, err := NewAnthropicProvider("")
	if err == nil {
		t.Fatal("NewAnthropicProvider() should return error for empty key")
	}
}

func TestAnthropicProvider_Complete(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/messages" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		if r.Header.Get("x-api-key") != "test-key" {
			t.Errorf("unexpected x-api-key: %s", r.Header.Get("x-api-key"))
		}
		if r.Header.Get("anthropic-version") != "2023-06-01" {
			t.Errorf("unexpected anthropic-version: %s", r.Header.Get("anthropic-version"))
		}

		var body map[string]interface{}
		_ = json.NewDecoder(r.Body).Decode(&body)

		if body["model"] != "claude-sonnet-4-6" {
			t.Errorf("model = %v, want claude-sonnet-4-6", body["model"])
		}

		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"content": []map[string]string{
				{"type": "text", "text": "Claude response"},
			},
			"model": "claude-sonnet-4-6",
			"usage": map[string]int{
				"input_tokens":  12,
				"output_tokens": 8,
			},
		})
	}))
	defer server.Close()

	provider, err := NewAnthropicProvider("test-key", WithAnthropicBaseURL(server.URL))
	if err != nil {
		t.Fatalf("NewAnthropicProvider() error = %v", err)
	}

	resp, err := provider.Complete(context.Background(), CompletionRequest{
		Messages: []Message{{Role: "user", Content: "hello"}},
	})

	if err != nil {
		t.Fatalf("Complete() error = %v", err)
	}
	if resp.Content != "Claude response" {
		t.Errorf("content = %q, want %q", resp.Content, "Claude response")
	}
	if resp.InputTokens != 12 {
		t.Errorf("input_tokens = %d, want 12", resp.InputTokens)
	}
	if resp.OutputTokens != 8 {
		t.Errorf("output_tokens = %d, want 8", resp.OutputTokens)
	}
}

func TestAnthropicProvider_Complete_SystemMessage(t *testing.T) {
	var receivedBody map[string]interface{}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewDecoder(r.Body).Decode(&receivedBody)

		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"content": []map[string]string{
				{"type": "text", "text": "ok"},
			},
			"model": "claude-sonnet-4-6",
			"usage": map[string]int{"input_tokens": 1, "output_tokens": 1},
		})
	}))
	defer server.Close()

	provider, _ := NewAnthropicProvider("test-key", WithAnthropicBaseURL(server.URL))

	_, err := provider.Complete(context.Background(), CompletionRequest{
		Messages: []Message{
			{Role: "system", Content: "You are a math tutor."},
			{Role: "user", Content: "hello"},
		},
	})

	if err != nil {
		t.Fatalf("Complete() error = %v", err)
	}

	// System message should be extracted to top-level "system" field.
	if receivedBody["system"] != "You are a math tutor." {
		t.Errorf("system = %v, want 'You are a math tutor.'", receivedBody["system"])
	}

	// Messages should NOT include the system message and should use content blocks.
	messages := receivedBody["messages"].([]interface{})
	if len(messages) != 1 {
		t.Fatalf("got %d messages, want 1 (system should be extracted)", len(messages))
	}
	firstMessage, ok := messages[0].(map[string]any)
	if !ok {
		t.Fatalf("first message invalid: %#v", messages[0])
	}
	content, ok := firstMessage["content"].([]any)
	if !ok || len(content) != 1 {
		t.Fatalf("content = %#v, want one text block", firstMessage["content"])
	}
	firstBlock, ok := content[0].(map[string]any)
	if !ok || firstBlock["type"] != "text" || firstBlock["text"] != "hello" {
		t.Fatalf("first block = %#v, want hello text block", content[0])
	}
}

func TestAnthropicProvider_Complete_StructuredOutput_UsesStableOutputConfigWithoutBetaHeader(t *testing.T) {
	var receivedBody map[string]any
	var receivedBeta string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedBeta = r.Header.Get("anthropic-beta")
		_ = json.NewDecoder(r.Body).Decode(&receivedBody)

		_ = json.NewEncoder(w).Encode(map[string]any{
			"content": []map[string]string{
				{"type": "text", "text": `{"final_answer":"ok"}`},
			},
			"model": "claude-sonnet-4-6",
			"usage": map[string]int{"input_tokens": 1, "output_tokens": 1},
		})
	}))
	defer server.Close()

	provider, _ := NewAnthropicProvider("test-key", WithAnthropicBaseURL(server.URL))

	_, err := provider.Complete(context.Background(), CompletionRequest{
		Messages: []Message{{Role: "user", Content: "hello"}},
		StructuredOutput: &StructuredOutputSpec{
			Name:       "tutor_response",
			JSONSchema: testStructuredSchema,
			Strict:     true,
		},
	})
	if err != nil {
		t.Fatalf("Complete() error = %v", err)
	}

	if receivedBeta != "" {
		t.Fatalf("anthropic-beta = %q, want empty", receivedBeta)
	}

	outputConfig, ok := receivedBody["output_config"].(map[string]any)
	if !ok {
		t.Fatalf("output_config missing or invalid: %#v", receivedBody["output_config"])
	}
	format, ok := outputConfig["format"].(map[string]any)
	if !ok {
		t.Fatalf("output_config.format missing or invalid: %#v", outputConfig["format"])
	}
	if format["type"] != "json_schema" {
		t.Fatalf("output_config.format.type = %#v, want json_schema", format["type"])
	}
	schema, ok := format["schema"].(map[string]any)
	if !ok {
		t.Fatalf("output_config.format.schema missing or invalid: %#v", format["schema"])
	}
	if schema["type"] != "object" {
		t.Fatalf("schema.type = %#v, want object", schema["type"])
	}
}

func TestAnthropicProvider_Complete_PlainTextRequestOmitsBetaHeader(t *testing.T) {
	var receivedBeta string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedBeta = r.Header.Get("anthropic-beta")

		_ = json.NewEncoder(w).Encode(map[string]any{
			"content": []map[string]string{
				{"type": "text", "text": "ok"},
			},
			"model": "claude-sonnet-4-6",
			"usage": map[string]int{"input_tokens": 1, "output_tokens": 1},
		})
	}))
	defer server.Close()

	provider, _ := NewAnthropicProvider("test-key", WithAnthropicBaseURL(server.URL))

	_, err := provider.Complete(context.Background(), CompletionRequest{
		Messages: []Message{{Role: "user", Content: "hello"}},
	})
	if err != nil {
		t.Fatalf("Complete() error = %v", err)
	}

	if receivedBeta != "" {
		t.Fatalf("anthropic-beta = %q, want empty", receivedBeta)
	}
}

func TestAnthropicProvider_Complete_WithDataURLImageUsesImageBlock(t *testing.T) {
	var receivedBody map[string]any

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewDecoder(r.Body).Decode(&receivedBody)

		_ = json.NewEncoder(w).Encode(map[string]any{
			"content": []map[string]string{
				{"type": "text", "text": "ok"},
			},
			"model": "claude-sonnet-4-6",
			"usage": map[string]int{"input_tokens": 1, "output_tokens": 1},
		})
	}))
	defer server.Close()

	provider, _ := NewAnthropicProvider("test-key", WithAnthropicBaseURL(server.URL))

	_, err := provider.Complete(context.Background(), CompletionRequest{
		Messages: []Message{{
			Role:      "user",
			Content:   "what is in this image?",
			ImageURLs: []string{"data:image/png;base64,AAEC"},
		}},
	})
	if err != nil {
		t.Fatalf("Complete() error = %v", err)
	}

	messages, ok := receivedBody["messages"].([]any)
	if !ok || len(messages) != 1 {
		t.Fatalf("messages = %#v, want one message", receivedBody["messages"])
	}
	firstMessage, ok := messages[0].(map[string]any)
	if !ok {
		t.Fatalf("first message invalid: %#v", messages[0])
	}
	content, ok := firstMessage["content"].([]any)
	if !ok || len(content) != 2 {
		t.Fatalf("content = %#v, want text + image blocks", firstMessage["content"])
	}
	imageBlock, ok := content[1].(map[string]any)
	if !ok {
		t.Fatalf("image block invalid: %#v", content[1])
	}
	source, ok := imageBlock["source"].(map[string]any)
	if !ok {
		t.Fatalf("image source invalid: %#v", imageBlock["source"])
	}
	if source["type"] != "base64" {
		t.Fatalf("source.type = %#v, want base64", source["type"])
	}
	if source["media_type"] != "image/png" {
		t.Fatalf("source.media_type = %#v, want image/png", source["media_type"])
	}
	if source["data"] != base64.StdEncoding.EncodeToString([]byte{0x00, 0x01, 0x02}) {
		t.Fatalf("source.data = %#v, want base64 payload", source["data"])
	}
}

func TestAnthropicProvider_Complete_WithRemoteImageURLUsesURLSource(t *testing.T) {
	var receivedBody map[string]any

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewDecoder(r.Body).Decode(&receivedBody)

		_ = json.NewEncoder(w).Encode(map[string]any{
			"content": []map[string]string{
				{"type": "text", "text": "ok"},
			},
			"model": "claude-sonnet-4-6",
			"usage": map[string]int{"input_tokens": 1, "output_tokens": 1},
		})
	}))
	defer server.Close()

	provider, _ := NewAnthropicProvider("test-key", WithAnthropicBaseURL(server.URL))

	_, err := provider.Complete(context.Background(), CompletionRequest{
		Messages: []Message{{
			Role:      "user",
			Content:   "what is in this image?",
			ImageURLs: []string{"https://example.com/image.png"},
		}},
	})
	if err != nil {
		t.Fatalf("Complete() error = %v", err)
	}

	messages, ok := receivedBody["messages"].([]any)
	if !ok || len(messages) != 1 {
		t.Fatalf("messages = %#v, want one message", receivedBody["messages"])
	}
	firstMessage, ok := messages[0].(map[string]any)
	if !ok {
		t.Fatalf("first message invalid: %#v", messages[0])
	}
	content, ok := firstMessage["content"].([]any)
	if !ok || len(content) != 2 {
		t.Fatalf("content = %#v, want text + image blocks", firstMessage["content"])
	}
	imageBlock, ok := content[1].(map[string]any)
	if !ok {
		t.Fatalf("image block invalid: %#v", content[1])
	}
	source, ok := imageBlock["source"].(map[string]any)
	if !ok {
		t.Fatalf("image source invalid: %#v", imageBlock["source"])
	}
	if source["type"] != "url" {
		t.Fatalf("source.type = %#v, want url", source["type"])
	}
	if source["url"] != "https://example.com/image.png" {
		t.Fatalf("source.url = %#v, want https URL", source["url"])
	}
}

func TestAnthropicProvider_Complete_RejectsUnsupportedImageInput(t *testing.T) {
	provider, _ := NewAnthropicProvider("test-key", WithAnthropicBaseURL("http://example.com"))

	_, err := provider.Complete(context.Background(), CompletionRequest{
		Messages: []Message{{
			Role:      "user",
			Content:   "what is in this image?",
			ImageURLs: []string{"ftp://example.com/image.png"},
		}},
	})
	if err == nil {
		t.Fatal("Complete() should reject unsupported image scheme")
	}
	if !strings.Contains(err.Error(), "unsupported image") {
		t.Fatalf("error = %v, want unsupported image error", err)
	}
}

func TestAnthropicProvider_Complete_APIError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"error": {"message": "invalid api key"}}`))
	}))
	defer server.Close()

	provider, _ := NewAnthropicProvider("bad-key", WithAnthropicBaseURL(server.URL))

	_, err := provider.Complete(context.Background(), CompletionRequest{
		Messages: []Message{{Role: "user", Content: "hello"}},
	})

	if err == nil {
		t.Fatal("Complete() should return error on API error")
	}
}

func TestAnthropicProvider_HealthCheck(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
		wantErr    bool
	}{
		{"healthy", http.StatusOK, false},
		{"unhealthy", http.StatusUnauthorized, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if tt.statusCode != http.StatusOK {
					w.WriteHeader(tt.statusCode)
					return
				}
				_ = json.NewEncoder(w).Encode(map[string]interface{}{
					"content": []map[string]string{
						{"type": "text", "text": "pong"},
					},
					"model": "claude-sonnet-4-6",
					"usage": map[string]int{"input_tokens": 1, "output_tokens": 1},
				})
			}))
			defer server.Close()

			provider, _ := NewAnthropicProvider("test-key", WithAnthropicBaseURL(server.URL))
			err := provider.HealthCheck(context.Background())

			if (err != nil) != tt.wantErr {
				t.Errorf("HealthCheck() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestAnthropicProvider_Models(t *testing.T) {
	provider, _ := NewAnthropicProvider("test-key")
	models := provider.Models()

	if len(models) == 0 {
		t.Fatal("Models() returned empty list")
	}
	for _, m := range models {
		if m.Name == "" {
			t.Errorf("model %q has empty name", m.ID)
		}
	}
}
