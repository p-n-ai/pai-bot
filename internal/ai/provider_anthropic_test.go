package ai

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
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
		json.NewDecoder(r.Body).Decode(&body)

		if body["model"] != "claude-sonnet-4-6" {
			t.Errorf("model = %v, want claude-sonnet-4-6", body["model"])
		}

		json.NewEncoder(w).Encode(map[string]interface{}{
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
		json.NewDecoder(r.Body).Decode(&receivedBody)

		json.NewEncoder(w).Encode(map[string]interface{}{
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

	// Messages should NOT include the system message.
	messages := receivedBody["messages"].([]interface{})
	if len(messages) != 1 {
		t.Fatalf("got %d messages, want 1 (system should be extracted)", len(messages))
	}
}

func TestAnthropicProvider_Complete_APIError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte(`{"error": {"message": "invalid api key"}}`))
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
				json.NewEncoder(w).Encode(map[string]interface{}{
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
