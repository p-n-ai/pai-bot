package ai

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestOllamaProvider_Complete(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/chat/completions" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		// Ollama doesn't require an Authorization header.
		if r.Header.Get("Authorization") != "" {
			t.Error("Ollama should not send Authorization header")
		}

		var req openaiRequest
		json.NewDecoder(r.Body).Decode(&req)

		json.NewEncoder(w).Encode(openaiResponse{
			Choices: []struct {
				Message struct {
					Content string `json:"content"`
				} `json:"message"`
			}{
				{Message: struct {
					Content string `json:"content"`
				}{Content: "Ollama response"}},
			},
			Model: "llama3:8b",
			Usage: struct {
				PromptTokens     int `json:"prompt_tokens"`
				CompletionTokens int `json:"completion_tokens"`
			}{PromptTokens: 5, CompletionTokens: 10},
		})
	}))
	defer server.Close()

	provider := NewOllamaProvider(server.URL)

	resp, err := provider.Complete(context.Background(), CompletionRequest{
		Messages: []Message{{Role: "user", Content: "hello"}},
	})

	if err != nil {
		t.Fatalf("Complete() error = %v", err)
	}
	if resp.Content != "Ollama response" {
		t.Errorf("content = %q, want %q", resp.Content, "Ollama response")
	}
	if resp.InputTokens != 5 {
		t.Errorf("input_tokens = %d, want 5", resp.InputTokens)
	}
}

func TestOllamaProvider_Complete_APIError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("model not found"))
	}))
	defer server.Close()

	provider := NewOllamaProvider(server.URL)

	_, err := provider.Complete(context.Background(), CompletionRequest{
		Messages: []Message{{Role: "user", Content: "hello"}},
	})

	if err == nil {
		t.Fatal("Complete() should return error on API error")
	}
}

func TestOllamaProvider_HealthCheck(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
		wantErr    bool
	}{
		{"healthy", http.StatusOK, false},
		{"unhealthy", http.StatusServiceUnavailable, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path != "/api/tags" {
					t.Errorf("unexpected path: %s", r.URL.Path)
				}
				w.WriteHeader(tt.statusCode)
			}))
			defer server.Close()

			provider := NewOllamaProvider(server.URL)
			err := provider.HealthCheck(context.Background())

			if (err != nil) != tt.wantErr {
				t.Errorf("HealthCheck() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestOllamaProvider_Models(t *testing.T) {
	provider := NewOllamaProvider("http://localhost:11434")
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
