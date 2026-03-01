package ai

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestOpenAIProvider_Complete(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/chat/completions" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		if r.Header.Get("Authorization") != "Bearer test-key" {
			t.Errorf("unexpected auth header: %s", r.Header.Get("Authorization"))
		}
		if r.Header.Get("Content-Type") != "application/json" {
			t.Errorf("unexpected content type: %s", r.Header.Get("Content-Type"))
		}

		var req openaiRequest
		json.NewDecoder(r.Body).Decode(&req)

		if req.Model != "gpt-4o" {
			t.Errorf("model = %q, want %q", req.Model, "gpt-4o")
		}
		if len(req.Messages) != 1 || req.Messages[0].Content != "hello" {
			t.Errorf("unexpected messages: %+v", req.Messages)
		}

		json.NewEncoder(w).Encode(openaiResponse{
			Choices: []struct {
				Message struct {
					Content string `json:"content"`
				} `json:"message"`
			}{
				{Message: struct {
					Content string `json:"content"`
				}{Content: "Hi there!"}},
			},
			Model: "gpt-4o",
			Usage: struct {
				PromptTokens     int `json:"prompt_tokens"`
				CompletionTokens int `json:"completion_tokens"`
			}{PromptTokens: 10, CompletionTokens: 5},
		})
	}))
	defer server.Close()

	provider := NewOpenAIProvider("test-key", WithBaseURL(server.URL))

	resp, err := provider.Complete(context.Background(), CompletionRequest{
		Messages: []Message{{Role: "user", Content: "hello"}},
		Model:    "gpt-4o",
	})

	if err != nil {
		t.Fatalf("Complete() error = %v", err)
	}
	if resp.Content != "Hi there!" {
		t.Errorf("content = %q, want %q", resp.Content, "Hi there!")
	}
	if resp.InputTokens != 10 {
		t.Errorf("input_tokens = %d, want 10", resp.InputTokens)
	}
	if resp.OutputTokens != 5 {
		t.Errorf("output_tokens = %d, want 5", resp.OutputTokens)
	}
}

func TestOpenAIProvider_Complete_APIError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusTooManyRequests)
		w.Write([]byte(`{"error": "rate limited"}`))
	}))
	defer server.Close()

	provider := NewOpenAIProvider("test-key", WithBaseURL(server.URL))

	_, err := provider.Complete(context.Background(), CompletionRequest{
		Messages: []Message{{Role: "user", Content: "hello"}},
	})

	if err == nil {
		t.Fatal("Complete() should return error on API error")
	}
}

func TestOpenAIProvider_Complete_EmptyChoices(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(openaiResponse{Choices: nil})
	}))
	defer server.Close()

	provider := NewOpenAIProvider("test-key", WithBaseURL(server.URL))

	_, err := provider.Complete(context.Background(), CompletionRequest{
		Messages: []Message{{Role: "user", Content: "hello"}},
	})

	if err == nil {
		t.Fatal("Complete() should return error when no choices")
	}
}

func TestDeepSeekProvider_UsesCorrectBaseURL(t *testing.T) {
	var receivedPath string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedPath = r.URL.Path
		json.NewEncoder(w).Encode(openaiResponse{
			Choices: []struct {
				Message struct {
					Content string `json:"content"`
				} `json:"message"`
			}{
				{Message: struct {
					Content string `json:"content"`
				}{Content: "deepseek response"}},
			},
			Model: "deepseek-chat",
		})
	}))
	defer server.Close()

	provider := NewDeepSeekProvider("ds-key", WithBaseURL(server.URL))

	resp, err := provider.Complete(context.Background(), CompletionRequest{
		Messages: []Message{{Role: "user", Content: "hello"}},
	})

	if err != nil {
		t.Fatalf("Complete() error = %v", err)
	}
	if receivedPath != "/chat/completions" {
		t.Errorf("path = %q, want /chat/completions", receivedPath)
	}
	if resp.Content != "deepseek response" {
		t.Errorf("content = %q, want %q", resp.Content, "deepseek response")
	}
}

func TestOpenAIProvider_HealthCheck(t *testing.T) {
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

			provider := NewOpenAIProvider("test-key", WithBaseURL(server.URL))
			err := provider.HealthCheck(context.Background())

			if (err != nil) != tt.wantErr {
				t.Errorf("HealthCheck() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestOpenAIProvider_Models(t *testing.T) {
	provider := NewOpenAIProvider("test-key")
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

func TestOpenAIProvider_CustomModels(t *testing.T) {
	custom := []ModelInfo{{ID: "custom-model", Name: "Custom"}}
	provider := NewOpenAIProvider("test-key", WithModels(custom))
	models := provider.Models()

	if len(models) != 1 || models[0].ID != "custom-model" {
		t.Errorf("Models() = %+v, want custom models", models)
	}
}

func TestOpenAIProvider_DefaultModel(t *testing.T) {
	var receivedModel string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req openaiRequest
		json.NewDecoder(r.Body).Decode(&req)
		receivedModel = req.Model

		json.NewEncoder(w).Encode(openaiResponse{
			Choices: []struct {
				Message struct {
					Content string `json:"content"`
				} `json:"message"`
			}{
				{Message: struct {
					Content string `json:"content"`
				}{Content: "ok"}},
			},
			Model: req.Model,
		})
	}))
	defer server.Close()

	provider := NewOpenAIProvider("test-key", WithBaseURL(server.URL))

	// No model specified — should use default.
	_, err := provider.Complete(context.Background(), CompletionRequest{
		Messages: []Message{{Role: "user", Content: "hi"}},
	})

	if err != nil {
		t.Fatalf("Complete() error = %v", err)
	}
	if receivedModel != "gpt-4o-mini" {
		t.Errorf("default model = %q, want %q", receivedModel, "gpt-4o-mini")
	}
}
