package ai

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestOpenRouterProvider_Complete(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/chat/completions" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		if r.Header.Get("Authorization") != "Bearer test-or-key" {
			t.Errorf("unexpected auth header: %s", r.Header.Get("Authorization"))
		}

		writeOpenAITextResponse(t, w, "OpenRouter response", "qwen/qwen-2.5-72b-instruct", 7, 15)
	}))
	defer server.Close()

	provider := NewOpenRouterProvider("test-or-key", WithOpenRouterBaseURL(server.URL))

	resp, err := provider.Complete(context.Background(), CompletionRequest{
		Messages: []Message{{Role: "user", Content: "hello"}},
	})

	if err != nil {
		t.Fatalf("Complete() error = %v", err)
	}
	if resp.Content != "OpenRouter response" {
		t.Errorf("content = %q, want %q", resp.Content, "OpenRouter response")
	}
	if resp.InputTokens != 7 {
		t.Errorf("input_tokens = %d, want 7", resp.InputTokens)
	}
}

func TestOpenRouterProvider_Complete_StructuredOutput_AddsResponseFormat(t *testing.T) {
	var captured map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewDecoder(r.Body).Decode(&captured)

		writeOpenAITextResponse(t, w, `{"final_answer":"ok"}`, "qwen/qwen-2.5-72b-instruct", 7, 15)
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
