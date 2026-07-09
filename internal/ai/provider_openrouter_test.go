// Copyright 2026 the P&AI authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package ai

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"
)

func TestOpenRouterProvider_Complete_MapsTeachingRequestAndResponse(t *testing.T) {
	type observedRequest struct {
		method        string
		path          string
		authorization string
		contentType   string
		body          map[string]any
	}
	observed := make(chan observedRequest, 1)
	decodeErrors := make(chan error, 1)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			decodeErrors <- err
			http.Error(w, "invalid request", http.StatusBadRequest)
			return
		}
		observed <- observedRequest{
			method:        r.Method,
			path:          r.URL.Path,
			authorization: r.Header.Get("Authorization"),
			contentType:   r.Header.Get("Content-Type"),
			body:          body,
		}

		writeOpenAITextResponse(t, w, "OpenRouter response", "qwen/qwen3-max", 7, 15)
	}))
	defer server.Close()

	provider := NewOpenRouterProvider("test-or-key", WithOpenRouterBaseURL(server.URL))

	resp, err := provider.Complete(context.Background(), CompletionRequest{
		Messages: []Message{
			{Role: "system", Content: "Teach with hints before answers."},
			{Role: "user", Content: "How do I solve 2x + 3 = 11?"},
			{Role: "assistant", Content: "What could you subtract first?"},
			{Role: "user", Content: "Is my working correct?", ImageURLs: []string{"data:image/png;base64,AAEC"}},
		},
		MaxTokens:   1024,
		Temperature: 0.3,
	})

	select {
	case decodeErr := <-decodeErrors:
		t.Fatalf("decode request: %v", decodeErr)
	default:
	}
	if err != nil {
		t.Fatalf("Complete() error = %v", err)
	}
	got := <-observed
	if got.method != http.MethodPost || got.path != "/chat/completions" {
		t.Errorf("request = %s %s, want POST /chat/completions", got.method, got.path)
	}
	if got.authorization != "Bearer test-or-key" {
		t.Errorf("authorization = %q, want Bearer test-or-key", got.authorization)
	}
	if got.contentType != "application/json" {
		t.Errorf("content type = %q, want application/json", got.contentType)
	}
	wantBody := map[string]any{
		"model":       "qwen/qwen3-max",
		"max_tokens":  float64(1024),
		"temperature": 0.3,
		"messages": []any{
			map[string]any{"role": "system", "content": "Teach with hints before answers."},
			map[string]any{"role": "user", "content": "How do I solve 2x + 3 = 11?"},
			map[string]any{"role": "assistant", "content": "What could you subtract first?"},
			map[string]any{"role": "user", "content": []any{
				map[string]any{"type": "text", "text": "Is my working correct?"},
				map[string]any{"type": "image_url", "image_url": map[string]any{"url": "data:image/png;base64,AAEC"}},
			}},
		},
	}
	if !reflect.DeepEqual(got.body, wantBody) {
		t.Errorf("request body = %#v, want %#v", got.body, wantBody)
	}
	if resp.Content != "OpenRouter response" {
		t.Errorf("content = %q, want %q", resp.Content, "OpenRouter response")
	}
	if resp.InputTokens != 7 {
		t.Errorf("input_tokens = %d, want 7", resp.InputTokens)
	}
	if resp.OutputTokens != 15 {
		t.Errorf("output_tokens = %d, want 15", resp.OutputTokens)
	}
	if resp.Model != "qwen/qwen3-max" {
		t.Errorf("model = %q, want qwen/qwen3-max", resp.Model)
	}
}

func TestOpenRouterProvider_Complete_StructuredOutput_AddsResponseFormat(t *testing.T) {
	var captured map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewDecoder(r.Body).Decode(&captured)

		writeOpenAITextResponse(t, w, `{"final_answer":"ok"}`, "qwen/qwen3-max", 7, 15)
	}))
	defer server.Close()

	provider := NewOpenRouterProvider("test-or-key", WithOpenRouterBaseURL(server.URL))

	resp, err := provider.Complete(context.Background(), CompletionRequest{
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

	assertJSONSchemaResponseFormat(t, captured)

	if resp.Content != `{"final_answer":"ok"}` {
		t.Fatalf("content = %q, want %q", resp.Content, `{"final_answer":"ok"}`)
	}
}

func TestOpenRouterProvider_ExtraHeaders(t *testing.T) {
	var gotReferer, gotTitle string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotReferer = r.Header.Get("HTTP-Referer")
		gotTitle = r.Header.Get("X-Title")

		writeOpenAITextResponse(t, w, "ok", "", 0, 0)
	}))
	defer server.Close()

	provider := NewOpenRouterProvider("test-key", WithOpenRouterBaseURL(server.URL))

	_, err := provider.Complete(context.Background(), CompletionRequest{
		Messages: []Message{{Role: "user", Content: "hi"}},
	})

	if err != nil {
		t.Fatalf("Complete() error = %v", err)
	}
	if gotReferer != "https://pandai.org" {
		t.Errorf("HTTP-Referer = %q, want %q", gotReferer, "https://pandai.org")
	}
	if gotTitle != "P&AI Bot" {
		t.Errorf("X-Title = %q, want %q", gotTitle, "P&AI Bot")
	}
}

func TestOpenRouterProvider_Complete_OmitsZeroMaxTokensAndTemperature(t *testing.T) {
	requestBodies := make(chan map[string]any, 1)
	decodeErrors := make(chan error, 1)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			decodeErrors <- err
			http.Error(w, "invalid request", http.StatusBadRequest)
			return
		}
		requestBodies <- body
		writeOpenAITextResponse(t, w, "ok", "qwen/qwen3-max", 0, 0)
	}))
	defer server.Close()
	provider := NewOpenRouterProvider("test-key", WithOpenRouterBaseURL(server.URL))

	_, err := provider.Complete(context.Background(), CompletionRequest{
		Messages: []Message{{Role: "user", Content: "hi"}},
	})

	select {
	case decodeErr := <-decodeErrors:
		t.Fatalf("decode request: %v", decodeErr)
	default:
	}
	if err != nil {
		t.Fatalf("Complete() error = %v", err)
	}
	body := <-requestBodies
	if _, ok := body["max_tokens"]; ok {
		t.Errorf("zero max_tokens should be omitted, request = %#v", body)
	}
	if _, ok := body["temperature"]; ok {
		t.Errorf("zero temperature should be omitted, request = %#v", body)
	}
}

func TestOpenRouterProvider_Complete_UsesCallerModel(t *testing.T) {
	models := make(chan string, 1)
	decodeErrors := make(chan error, 1)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			Model string `json:"model"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			decodeErrors <- err
			http.Error(w, "invalid request", http.StatusBadRequest)
			return
		}
		models <- body.Model
		writeOpenAITextResponse(t, w, "ok", body.Model, 0, 0)
	}))
	defer server.Close()
	provider := NewOpenRouterProvider("test-key", WithOpenRouterBaseURL(server.URL))

	resp, err := provider.Complete(context.Background(), CompletionRequest{
		Messages: []Message{{Role: "user", Content: "hi"}},
		Model:    "anthropic/claude-sonnet-4",
	})

	select {
	case decodeErr := <-decodeErrors:
		t.Fatalf("decode request: %v", decodeErr)
	default:
	}
	if err != nil {
		t.Fatalf("Complete() error = %v", err)
	}
	if got := <-models; got != "anthropic/claude-sonnet-4" {
		t.Errorf("request model = %q, want anthropic/claude-sonnet-4", got)
	}
	if resp.Model != "anthropic/claude-sonnet-4" {
		t.Errorf("response model = %q, want anthropic/claude-sonnet-4", resp.Model)
	}
}

func TestOpenRouterProvider_Complete_APIError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusTooManyRequests)
		_, _ = w.Write([]byte(`{"error": "rate limited"}`))
	}))
	defer server.Close()

	provider := NewOpenRouterProvider("test-key", WithOpenRouterBaseURL(server.URL))

	_, err := provider.Complete(context.Background(), CompletionRequest{
		Messages: []Message{{Role: "user", Content: "hello"}},
	})

	if err == nil {
		t.Fatal("Complete() should return error on API error")
	}
}

func TestOpenRouterProvider_HealthCheck(t *testing.T) {
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
				if r.URL.Path != "/models" {
					t.Errorf("unexpected path: %s", r.URL.Path)
				}
				w.WriteHeader(tt.statusCode)
			}))
			defer server.Close()

			provider := NewOpenRouterProvider("test-key", WithOpenRouterBaseURL(server.URL))
			err := provider.HealthCheck(context.Background())

			if (err != nil) != tt.wantErr {
				t.Errorf("HealthCheck() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestOpenRouterProvider_Models(t *testing.T) {
	provider := NewOpenRouterProvider("test-key")
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
