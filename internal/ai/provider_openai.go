package ai

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

const (
	defaultOpenAIBaseURL   = "https://api.openai.com/v1"
	defaultDeepSeekBaseURL = "https://api.deepseek.com"
)

// OpenAIProvider implements Provider for OpenAI and OpenAI-compatible APIs
// (DeepSeek, Groq, Together AI, etc.) via a configurable base URL.
type OpenAIProvider struct {
	apiKey  string
	baseURL string
	client  *http.Client
	name    string
	models  []ModelInfo
}

// OpenAIOption configures an OpenAIProvider.
type OpenAIOption func(*OpenAIProvider)

// WithBaseURL sets the base URL for the OpenAI-compatible API.
func WithBaseURL(url string) OpenAIOption {
	return func(p *OpenAIProvider) {
		p.baseURL = url
	}
}

// WithHTTPClient sets a custom HTTP client.
func WithHTTPClient(client *http.Client) OpenAIOption {
	return func(p *OpenAIProvider) {
		p.client = client
	}
}

// WithModels sets the available models for this provider.
func WithModels(models []ModelInfo) OpenAIOption {
	return func(p *OpenAIProvider) {
		p.models = models
	}
}

// WithProviderName sets the provider name (for multi-instance use, e.g. "deepseek").
func WithProviderName(name string) OpenAIOption {
	return func(p *OpenAIProvider) {
		p.name = name
	}
}

// NewOpenAIProvider creates a new OpenAI-compatible provider.
func NewOpenAIProvider(apiKey string, opts ...OpenAIOption) *OpenAIProvider {
	p := &OpenAIProvider{
		apiKey:  apiKey,
		baseURL: defaultOpenAIBaseURL,
		client:  http.DefaultClient,
		name:    "openai",
	}
	for _, opt := range opts {
		opt(p)
	}
	return p
}

// NewDeepSeekProvider creates a provider for the DeepSeek API (OpenAI-compatible).
func NewDeepSeekProvider(apiKey string, opts ...OpenAIOption) *OpenAIProvider {
	opts = append([]OpenAIOption{
		WithBaseURL(defaultDeepSeekBaseURL),
		WithProviderName("deepseek"),
	}, opts...)
	return NewOpenAIProvider(apiKey, opts...)
}

// openaiRequest is the request body for the OpenAI chat completions API.
type openaiRequest struct {
	Model       string          `json:"model"`
	Messages    []openaiMessage `json:"messages"`
	MaxTokens   int             `json:"max_tokens,omitempty"`
	Temperature *float64        `json:"temperature,omitempty"`
}

type openaiMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// openaiResponse is the response from the OpenAI chat completions API.
type openaiResponse struct {
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
	Model string `json:"model"`
	Usage struct {
		PromptTokens     int `json:"prompt_tokens"`
		CompletionTokens int `json:"completion_tokens"`
	} `json:"usage"`
}

func (p *OpenAIProvider) Complete(ctx context.Context, req CompletionRequest) (CompletionResponse, error) {
	model := req.Model
	if model == "" {
		model = "gpt-4o-mini" // sensible default
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
		return CompletionResponse{}, fmt.Errorf("openai api error (status %d): %s", resp.StatusCode, string(respBody))
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

func (p *OpenAIProvider) StreamComplete(ctx context.Context, req CompletionRequest) (<-chan StreamChunk, error) {
	// TODO: implement SSE streaming
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

func (p *OpenAIProvider) Models() []ModelInfo {
	if p.models != nil {
		return p.models
	}
	return []ModelInfo{
		{ID: "gpt-4o", Name: "GPT-4o", MaxTokens: 128000, Description: "Most capable OpenAI model"},
		{ID: "gpt-4o-mini", Name: "GPT-4o Mini", MaxTokens: 128000, Description: "Fast, affordable OpenAI model"},
	}
}

func (p *OpenAIProvider) HealthCheck(ctx context.Context) error {
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
