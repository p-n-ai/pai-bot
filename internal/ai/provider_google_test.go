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
		_ = json.NewDecoder(r.Body).Decode(&req)

		if len(req.Contents) == 0 {
			t.Error("no contents in request")
		}

		_ = json.NewEncoder(w).Encode(geminiResponse{
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
	var receivedBody map[string]any

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewDecoder(r.Body).Decode(&receivedBody)

		_ = json.NewEncoder(w).Encode(geminiResponse{
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

	systemInstruction, ok := receivedBody["systemInstruction"].(map[string]any)
	if !ok {
		t.Fatalf("systemInstruction missing or invalid: %#v", receivedBody["systemInstruction"])
	}
	systemParts, ok := systemInstruction["parts"].([]any)
	if !ok || len(systemParts) != 1 {
		t.Fatalf("systemInstruction.parts = %#v, want one part", systemInstruction["parts"])
	}
	firstSystemPart, ok := systemParts[0].(map[string]any)
	if !ok || firstSystemPart["text"] != "You are a tutor." {
		t.Fatalf("systemInstruction first part = %#v, want tutor text", systemParts[0])
	}

	receivedContents, ok := receivedBody["contents"].([]any)
	if !ok {
		t.Fatalf("contents missing or invalid: %#v", receivedBody["contents"])
	}
	if len(receivedContents) != 3 {
		t.Fatalf("got %d contents, want 3", len(receivedContents))
	}
	secondContent, ok := receivedContents[1].(map[string]any)
	if !ok {
		t.Fatalf("second content invalid: %#v", receivedContents[1])
	}
	if secondContent["role"] != "model" {
		t.Errorf("assistant role mapped to %q, want %q", secondContent["role"], "model")
	}
}

func TestGoogleProvider_Complete_WithDataURLImageUsesInlineData(t *testing.T) {
	var receivedBody map[string]any

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewDecoder(r.Body).Decode(&receivedBody)

		_ = json.NewEncoder(w).Encode(geminiResponse{
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
		Messages: []Message{{
			Role:      "user",
			Content:   "what is in this image?",
			ImageURLs: []string{"data:image/png;base64,AAEC"},
		}},
	})
	if err != nil {
		t.Fatalf("Complete() error = %v", err)
	}

	contents, ok := receivedBody["contents"].([]any)
	if !ok || len(contents) != 1 {
		t.Fatalf("contents = %#v, want one entry", receivedBody["contents"])
	}
	content, ok := contents[0].(map[string]any)
	if !ok {
		t.Fatalf("content invalid: %#v", contents[0])
	}
	parts, ok := content["parts"].([]any)
	if !ok || len(parts) != 2 {
		t.Fatalf("parts = %#v, want text + image", content["parts"])
	}
	imagePart, ok := parts[1].(map[string]any)
	if !ok {
		t.Fatalf("image part invalid: %#v", parts[1])
	}
	inlineData, ok := imagePart["inlineData"].(map[string]any)
	if !ok {
		t.Fatalf("inlineData missing or invalid: %#v", imagePart["inlineData"])
	}
	if inlineData["mimeType"] != "image/png" {
		t.Fatalf("inlineData.mimeType = %#v, want image/png", inlineData["mimeType"])
	}
	if inlineData["data"] != base64.StdEncoding.EncodeToString([]byte{0x00, 0x01, 0x02}) {
		t.Fatalf("inlineData.data = %#v, want base64 payload", inlineData["data"])
	}
}

func TestGoogleProvider_Complete_WithRemoteImageURLDownloadsToInlineData(t *testing.T) {
	imageServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "image/png")
		_, _ = w.Write([]byte{0x00, 0x01, 0x02})
	}))
	defer imageServer.Close()

	var receivedBody map[string]any
	apiServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewDecoder(r.Body).Decode(&receivedBody)

		_ = json.NewEncoder(w).Encode(geminiResponse{
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
	defer apiServer.Close()

	provider := NewGoogleProvider("test-key", WithGoogleBaseURL(apiServer.URL))

	_, err := provider.Complete(context.Background(), CompletionRequest{
		Messages: []Message{{
			Role:      "user",
			Content:   "what is in this image?",
			ImageURLs: []string{imageServer.URL + "/image.png"},
		}},
	})
	if err != nil {
		t.Fatalf("Complete() error = %v", err)
	}

	contents, ok := receivedBody["contents"].([]any)
	if !ok || len(contents) != 1 {
		t.Fatalf("contents = %#v, want one entry", receivedBody["contents"])
	}
	content, ok := contents[0].(map[string]any)
	if !ok {
		t.Fatalf("content invalid: %#v", contents[0])
	}
	parts, ok := content["parts"].([]any)
	if !ok || len(parts) != 2 {
		t.Fatalf("parts = %#v, want text + image", content["parts"])
	}
	imagePart, ok := parts[1].(map[string]any)
	if !ok {
		t.Fatalf("image part invalid: %#v", parts[1])
	}
	inlineData, ok := imagePart["inlineData"].(map[string]any)
	if !ok {
		t.Fatalf("inlineData missing or invalid: %#v", imagePart["inlineData"])
	}
	if inlineData["mimeType"] != "image/png" {
		t.Fatalf("inlineData.mimeType = %#v, want image/png", inlineData["mimeType"])
	}
	if inlineData["data"] != base64.StdEncoding.EncodeToString([]byte{0x00, 0x01, 0x02}) {
		t.Fatalf("inlineData.data = %#v, want base64 payload", inlineData["data"])
	}
}

func TestGoogleProvider_Complete_RejectsUnsupportedImageInput(t *testing.T) {
	provider := NewGoogleProvider("test-key", WithGoogleBaseURL("http://example.com"))

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

func TestGoogleProvider_Complete_StructuredOutput_AddsJSONSchemaConfig(t *testing.T) {
	var captured geminiRequest

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewDecoder(r.Body).Decode(&captured)

		_ = json.NewEncoder(w).Encode(geminiResponse{
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
				}{{Text: `{"final_answer":"ok"}`}}}},
			},
		})
	}))
	defer server.Close()

	provider := NewGoogleProvider("test-key", WithGoogleBaseURL(server.URL))

	_, err := provider.Complete(context.Background(), CompletionRequest{
		Messages: []Message{{Role: "user", Content: "hello"}},
		StructuredOutput: &StructuredOutputSpec{
			Name:       "tutor_response",
			JSONSchema: testStructuredSchema,
			Strict:     true,
		},
		MaxTokens:   123,
		Temperature: 0.4,
	})
	if err != nil {
		t.Fatalf("Complete() error = %v", err)
	}

	if captured.GenerationConfig == nil {
		t.Fatal("GenerationConfig should be set for structured output")
	}
	if captured.GenerationConfig.ResponseMIMEType != "application/json" {
		t.Fatalf("ResponseMIMEType = %q, want application/json", captured.GenerationConfig.ResponseMIMEType)
	}
	if string(captured.GenerationConfig.ResponseJSONSchema) != string(testStructuredSchema) {
		t.Fatalf("ResponseJSONSchema = %s, want %s", string(captured.GenerationConfig.ResponseJSONSchema), string(testStructuredSchema))
	}
	if captured.GenerationConfig.MaxOutputTokens != 123 {
		t.Fatalf("MaxOutputTokens = %d, want 123", captured.GenerationConfig.MaxOutputTokens)
	}
	if captured.GenerationConfig.Temperature == nil || *captured.GenerationConfig.Temperature != 0.4 {
		t.Fatalf("Temperature = %#v, want 0.4", captured.GenerationConfig.Temperature)
	}
}

func TestGoogleProvider_Complete_APIError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		_, _ = w.Write([]byte(`{"error": "forbidden"}`))
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
