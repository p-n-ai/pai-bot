package ai

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

const defaultGeminiBaseURL = "https://generativelanguage.googleapis.com/v1beta"

// GoogleProvider implements Provider for Google Gemini.
type GoogleProvider struct {
	apiKey  string
	baseURL string
	client  *http.Client
	models  []ModelInfo
}

// GoogleOption configures a GoogleProvider.
type GoogleOption func(*GoogleProvider)

// WithGoogleBaseURL sets the base URL (for testing).
func WithGoogleBaseURL(url string) GoogleOption {
	return func(p *GoogleProvider) {
		p.baseURL = url
	}
}

// WithGoogleHTTPClient sets a custom HTTP client.
func WithGoogleHTTPClient(client *http.Client) GoogleOption {
	return func(p *GoogleProvider) {
		p.client = client
	}
}

// NewGoogleProvider creates a new Google Gemini provider.
func NewGoogleProvider(apiKey string, opts ...GoogleOption) *GoogleProvider {
	p := &GoogleProvider{
		apiKey:  apiKey,
		baseURL: defaultGeminiBaseURL,
		client:  http.DefaultClient,
	}
	for _, opt := range opts {
		opt(p)
	}
	return p
}

// geminiRequest is the request body for the Gemini generateContent API.
type geminiRequest struct {
	Contents         []geminiContent         `json:"contents"`
	GenerationConfig *geminiGenerationConfig `json:"generationConfig,omitempty"`
}

type geminiContent struct {
	Role  string       `json:"role"`
	Parts []geminiPart `json:"parts"`
}

type geminiPart struct {
	Text string `json:"text"`
}

type geminiGenerationConfig struct {
	MaxOutputTokens int      `json:"maxOutputTokens,omitempty"`
	Temperature     *float64 `json:"temperature,omitempty"`
}

// geminiResponse is the response from the Gemini API.
type geminiResponse struct {
	Candidates []struct {
		Content struct {
			Parts []struct {
				Text string `json:"text"`
			} `json:"parts"`
		} `json:"content"`
	} `json:"candidates"`
	UsageMetadata struct {
		PromptTokenCount     int `json:"promptTokenCount"`
		CandidatesTokenCount int `json:"candidatesTokenCount"`
	} `json:"usageMetadata"`
}

func (p *GoogleProvider) Complete(ctx context.Context, req CompletionRequest) (CompletionResponse, error) {
	model := req.Model
	if model == "" {
		model = "gemini-2.5-flash"
	}

	contents := make([]geminiContent, 0, len(req.Messages))
	for _, m := range req.Messages {
		role := m.Role
		// Gemini uses "user" and "model" roles; map "assistant" to "model".
		if role == "assistant" {
			role = "model"
		}
		// Gemini doesn't support "system" as a content role; prepend to first user message.
		if role == "system" {
			continue
		}
		contents = append(contents, geminiContent{
			Role:  role,
			Parts: []geminiPart{{Text: m.Content}},
		})
	}

	gemReq := geminiRequest{Contents: contents}
	if req.MaxTokens > 0 || req.Temperature > 0 {
		config := &geminiGenerationConfig{}
		if req.MaxTokens > 0 {
			config.MaxOutputTokens = req.MaxTokens
		}
		if req.Temperature > 0 {
			temp := req.Temperature
			config.Temperature = &temp
		}
		gemReq.GenerationConfig = config
	}

	body, err := json.Marshal(gemReq)
	if err != nil {
		return CompletionResponse{}, fmt.Errorf("marshal request: %w", err)
	}

	url := fmt.Sprintf("%s/models/%s:generateContent?key=%s", p.baseURL, model, p.apiKey)
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return CompletionResponse{}, fmt.Errorf("create request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := p.client.Do(httpReq)
	if err != nil {
		return CompletionResponse{}, fmt.Errorf("send request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return CompletionResponse{}, fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return CompletionResponse{}, fmt.Errorf("gemini api error (status %d): %s", resp.StatusCode, string(respBody))
	}

	var gemResp geminiResponse
	if err := json.Unmarshal(respBody, &gemResp); err != nil {
		return CompletionResponse{}, fmt.Errorf("unmarshal response: %w", err)
	}

	if len(gemResp.Candidates) == 0 || len(gemResp.Candidates[0].Content.Parts) == 0 {
		return CompletionResponse{}, fmt.Errorf("no content in response")
	}

	return CompletionResponse{
		Content:      gemResp.Candidates[0].Content.Parts[0].Text,
		Model:        model,
		InputTokens:  gemResp.UsageMetadata.PromptTokenCount,
		OutputTokens: gemResp.UsageMetadata.CandidatesTokenCount,
	}, nil
}

func (p *GoogleProvider) StreamComplete(ctx context.Context, req CompletionRequest) (<-chan StreamChunk, error) {
	// TODO: implement SSE streaming via streamGenerateContent
	ch := make(chan StreamChunk, 1)
	resp, err := p.Complete(ctx, req)
	if err != nil {
		close(ch)
		return nil, err
	}
	ch <- StreamChunk{Content: resp.Content, Done: true}
	close(ch)
	return ch, nil
}

func (p *GoogleProvider) Models() []ModelInfo {
	if p.models != nil {
		return p.models
	}
	return []ModelInfo{
		{ID: "gemini-2.5-pro", Name: "Gemini 2.5 Pro", MaxTokens: 1048576, Description: "Most capable Google model"},
		{ID: "gemini-2.5-flash", Name: "Gemini 2.5 Flash", MaxTokens: 1048576, Description: "Fast, affordable Google model"},
	}
}

func (p *GoogleProvider) HealthCheck(ctx context.Context) error {
	url := fmt.Sprintf("%s/models?key=%s", p.baseURL, p.apiKey)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return err
	}

	resp, err := p.client.Do(req)
	if err != nil {
		return fmt.Errorf("health check failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("health check returned status %d", resp.StatusCode)
	}
	return nil
}
