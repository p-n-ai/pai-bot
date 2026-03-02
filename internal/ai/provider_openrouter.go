package ai

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

const defaultOpenRouterBaseURL = "https://openrouter.ai/api/v1"

// OpenRouterProvider implements Provider for OpenRouter.
// OpenRouter uses an OpenAI-compatible API with extra HTTP headers.
type OpenRouterProvider struct {
	apiKey  string
	baseURL string
	client  *http.Client
	models  []ModelInfo
}

// OpenRouterOption configures an OpenRouterProvider.
type OpenRouterOption func(*OpenRouterProvider)

// WithOpenRouterBaseURL sets the base URL (for testing).
func WithOpenRouterBaseURL(url string) OpenRouterOption {
	return func(p *OpenRouterProvider) {
		p.baseURL = url
	}
}

// WithOpenRouterHTTPClient sets a custom HTTP client.
func WithOpenRouterHTTPClient(client *http.Client) OpenRouterOption {
	return func(p *OpenRouterProvider) {
		p.client = client
	}
}

// NewOpenRouterProvider creates a new OpenRouter provider.
func NewOpenRouterProvider(apiKey string, opts ...OpenRouterOption) *OpenRouterProvider {
	p := &OpenRouterProvider{
		apiKey:  apiKey,
		baseURL: defaultOpenRouterBaseURL,
		client:  http.DefaultClient,
	}
	for _, opt := range opts {
		opt(p)
	}
	return p
}

func (p *OpenRouterProvider) Complete(ctx context.Context, req CompletionRequest) (CompletionResponse, error) {
	model := req.Model
	if model == "" {
		model = "qwen/qwen-2.5-72b-instruct"
	}

	messages := make([]openaiMessage, len(req.Messages))
	for i, m := range req.Messages {
		messages[i] = openaiMessage(m)
	}

	oaiReq := openaiRequest{
		Model:    model,
		Messages: messages,
	}
	if req.MaxTokens > 0 {
		oaiReq.MaxTokens = req.MaxTokens
	}
	if req.Temperature > 0 {
		temp := req.Temperature
		oaiReq.Temperature = &temp
	}

	body, err := json.Marshal(oaiReq)
	if err != nil {
		return CompletionResponse{}, fmt.Errorf("marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, p.baseURL+"/chat/completions", bytes.NewReader(body))
	if err != nil {
		return CompletionResponse{}, fmt.Errorf("create request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+p.apiKey)
	httpReq.Header.Set("HTTP-Referer", "https://pandai.org")
	httpReq.Header.Set("X-Title", "P&AI Bot")

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
		return CompletionResponse{}, fmt.Errorf("openrouter api error (status %d): %s", resp.StatusCode, string(respBody))
	}

	var oaiResp openaiResponse
	if err := json.Unmarshal(respBody, &oaiResp); err != nil {
		return CompletionResponse{}, fmt.Errorf("unmarshal response: %w", err)
	}

	if len(oaiResp.Choices) == 0 {
		return CompletionResponse{}, fmt.Errorf("no choices in response")
	}

	return CompletionResponse{
		Content:      oaiResp.Choices[0].Message.Content,
		Model:        oaiResp.Model,
		InputTokens:  oaiResp.Usage.PromptTokens,
		OutputTokens: oaiResp.Usage.CompletionTokens,
	}, nil
}

func (p *OpenRouterProvider) StreamComplete(ctx context.Context, req CompletionRequest) (<-chan StreamChunk, error) {
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

func (p *OpenRouterProvider) Models() []ModelInfo {
	if p.models != nil {
		return p.models
	}
	return []ModelInfo{
		{ID: "qwen/qwen-2.5-72b-instruct", Name: "Qwen 2.5 72B", MaxTokens: 32768, Description: "Large open-weight model via OpenRouter"},
	}
}

func (p *OpenRouterProvider) HealthCheck(ctx context.Context) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, p.baseURL+"/models", nil)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+p.apiKey)

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
