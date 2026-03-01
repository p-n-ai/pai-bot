package ai

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestGoogleProvider_Complete(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify Gemini-specific URL pattern.
		if !strings.Contains(r.URL.Path, "/models/gemini-2.5-flash:generateContent") {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		if r.URL.Query().Get("key") != "test-key" {
			t.Errorf("missing or wrong API key in query")
		}

		var req geminiRequest
		json.NewDecoder(r.Body).Decode(&req)

		if len(req.Contents) == 0 {
			t.Error("no contents in request")
		}

		json.NewEncoder(w).Encode(geminiResponse{
			Candidates: []struct {
				Content struct {
					Parts []struct {
						Text string `json:"text"`
					} `json:"parts"`
				} `json:"content"`
			}{
				{Content: struct {
					Parts []struct {
						Text string `json:"text"`
					} `json:"parts"`
				}{Parts: []struct {
					Text string `json:"text"`
				}{{Text: "Gemini response"}}}},
			},
			UsageMetadata: struct {
				PromptTokenCount     int `json:"promptTokenCount"`
				CandidatesTokenCount int `json:"candidatesTokenCount"`
			}{PromptTokenCount: 8, CandidatesTokenCount: 12},
		})
	}))
	defer server.Close()

	provider := NewGoogleProvider("test-key", WithGoogleBaseURL(server.URL))

	resp, err := provider.Complete(context.Background(), CompletionRequest{
		Messages: []Message{{Role: "user", Content: "hello"}},
	})

	if err != nil {
		t.Fatalf("Complete() error = %v", err)
	}
	if resp.Content != "Gemini response" {
		t.Errorf("content = %q, want %q", resp.Content, "Gemini response")
	}
	if resp.InputTokens != 8 {
		t.Errorf("input_tokens = %d, want 8", resp.InputTokens)
	}
}

func TestGoogleProvider_Complete_RoleMappings(t *testing.T) {
	var receivedContents []geminiContent

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req geminiRequest
		json.NewDecoder(r.Body).Decode(&req)
		receivedContents = req.Contents

		json.NewEncoder(w).Encode(geminiResponse{
			Candidates: []struct {
				Content struct {
					Parts []struct {
						Text string `json:"text"`
					} `json:"parts"`
				} `json:"content"`
			}{
				{Content: struct {
					Parts []struct {
						Text string `json:"text"`
					} `json:"parts"`
				}{Parts: []struct {
					Text string `json:"text"`
				}{{Text: "ok"}}}},
			},
		})
	}))
	defer server.Close()

	provider := NewGoogleProvider("test-key", WithGoogleBaseURL(server.URL))

	_, err := provider.Complete(context.Background(), CompletionRequest{
		Messages: []Message{
			{Role: "system", Content: "You are a tutor."},
			{Role: "user", Content: "hello"},
			{Role: "assistant", Content: "hi"},
			{Role: "user", Content: "explain algebra"},
		},
	})

	if err != nil {
		t.Fatalf("Complete() error = %v", err)
	}

	// System messages should be skipped, assistant mapped to "model".
	if len(receivedContents) != 3 {
		t.Fatalf("got %d contents, want 3 (system should be skipped)", len(receivedContents))
	}
	if receivedContents[1].Role != "model" {
		t.Errorf("assistant role mapped to %q, want %q", receivedContents[1].Role, "model")
	}
}

func TestGoogleProvider_Complete_APIError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		w.Write([]byte(`{"error": "forbidden"}`))
	}))
	defer server.Close()

	provider := NewGoogleProvider("test-key", WithGoogleBaseURL(server.URL))

	_, err := provider.Complete(context.Background(), CompletionRequest{
		Messages: []Message{{Role: "user", Content: "hello"}},
	})

	if err == nil {
		t.Fatal("Complete() should return error on API error")
	}
}

func TestGoogleProvider_HealthCheck(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
		wantErr    bool
	}{
		{"healthy", http.StatusOK, false},
		{"unhealthy", http.StatusForbidden, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if !strings.Contains(r.URL.Path, "/models") {
					t.Errorf("unexpected path: %s", r.URL.Path)
				}
				w.WriteHeader(tt.statusCode)
			}))
			defer server.Close()

			provider := NewGoogleProvider("test-key", WithGoogleBaseURL(server.URL))
			err := provider.HealthCheck(context.Background())

			if (err != nil) != tt.wantErr {
				t.Errorf("HealthCheck() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestGoogleProvider_Models(t *testing.T) {
	provider := NewGoogleProvider("test-key")
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
