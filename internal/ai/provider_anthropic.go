package ai

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

const defaultAnthropicBaseURL = "https://api.anthropic.com/v1"

// AnthropicProvider implements Provider for Anthropic Claude.
type AnthropicProvider struct {
	apiKey  string
	baseURL string
	client  *http.Client
}

// AnthropicOption configures an AnthropicProvider.
type AnthropicOption func(*AnthropicProvider)

// WithAnthropicBaseURL sets the base URL (for testing).
func WithAnthropicBaseURL(url string) AnthropicOption {
	return func(p *AnthropicProvider) {
		p.baseURL = url
	}
}

// NewAnthropicProvider creates a new Anthropic provider.
func NewAnthropicProvider(apiKey string, opts ...AnthropicOption) (*AnthropicProvider, error) {
	if apiKey == "" {
		return nil, fmt.Errorf("anthropic API key is required")
	}
	p := &AnthropicProvider{
		apiKey:  apiKey,
		baseURL: defaultAnthropicBaseURL,
		client:  &http.Client{},
	}
	for _, opt := range opts {
		opt(p)
	}
	return p, nil
}

func (p *AnthropicProvider) Complete(ctx context.Context, req CompletionRequest) (CompletionResponse, error) {
	model := req.Model
	if model == "" {
		model = "claude-sonnet-4-6"
	}
	maxTokens := req.MaxTokens
	if maxTokens == 0 {
		maxTokens = 4096
	}

	// Separate system message from user/assistant messages.
	var systemPrompt string
	var messages []map[string]string
	for _, m := range req.Messages {
		if m.Role == "system" {
			systemPrompt = m.Content
			continue
		}
		messages = append(messages, map[string]string{
			"role":    m.Role,
			"content": m.Content,
		})
	}

	body := map[string]interface{}{
		"model":      model,
		"max_tokens": maxTokens,
		"messages":   messages,
	}
	if systemPrompt != "" {
		body["system"] = systemPrompt
	}
	if req.Temperature > 0 {
		body["temperature"] = req.Temperature
	}

	jsonBody, err := json.Marshal(body)
	if err != nil {
		return CompletionResponse{}, fmt.Errorf("marshaling request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, p.baseURL+"/messages", bytes.NewReader(jsonBody))
	if err != nil {
		return CompletionResponse{}, err
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("x-api-key", p.apiKey)
	httpReq.Header.Set("anthropic-version", "2023-06-01")

	resp, err := p.client.Do(httpReq)
	if err != nil {
		return CompletionResponse{}, fmt.Errorf("anthropic API call: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return CompletionResponse{}, fmt.Errorf("reading response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return CompletionResponse{}, fmt.Errorf("anthropic API error %d: %s", resp.StatusCode, string(respBody))
	}

	var result struct {
		Content []struct {
			Text string `json:"text"`
		} `json:"content"`
		Model string `json:"model"`
		Usage struct {
			InputTokens  int `json:"input_tokens"`
			OutputTokens int `json:"output_tokens"`
		} `json:"usage"`
	}
	if err := json.Unmarshal(respBody, &result); err != nil {
		return CompletionResponse{}, fmt.Errorf("parsing response: %w", err)
	}

	if len(result.Content) == 0 {
		return CompletionResponse{}, fmt.Errorf("anthropic returned no content")
	}

	return CompletionResponse{
		Content:      result.Content[0].Text,
		Model:        result.Model,
		InputTokens:  result.Usage.InputTokens,
		OutputTokens: result.Usage.OutputTokens,
	}, nil
}

func (p *AnthropicProvider) StreamComplete(ctx context.Context, req CompletionRequest) (<-chan StreamChunk, error) {
	ch := make(chan StreamChunk, 1)
	go func() {
		defer close(ch)
		resp, err := p.Complete(ctx, req)
		if err != nil {
			ch <- StreamChunk{Error: err}
			return
		}
		ch <- StreamChunk{Content: resp.Content, Done: true}
	}()
	return ch, nil
}

func (p *AnthropicProvider) Models() []ModelInfo {
	return []ModelInfo{
		{ID: "claude-sonnet-4-6", Name: "Claude Sonnet 4.6", MaxTokens: 200000, Description: "Best for teaching"},
		{ID: "claude-haiku-4-5-20251001", Name: "Claude Haiku 4.5", MaxTokens: 200000, Description: "Fast grading"},
	}
}

func (p *AnthropicProvider) HealthCheck(ctx context.Context) error {
	_, err := p.Complete(ctx, CompletionRequest{
		Messages:  []Message{{Role: "user", Content: "ping"}},
		MaxTokens: 1,
	})
	return err
}
