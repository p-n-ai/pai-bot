package ai

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

// OllamaProvider implements Provider for self-hosted Ollama.
// Ollama exposes an OpenAI-compatible API at /v1/chat/completions.
type OllamaProvider struct {
	baseURL string
	client  *http.Client
	models  []ModelInfo
}

// OllamaOption configures an OllamaProvider.
type OllamaOption func(*OllamaProvider)

// WithOllamaHTTPClient sets a custom HTTP client.
func WithOllamaHTTPClient(client *http.Client) OllamaOption {
	return func(p *OllamaProvider) {
		p.client = client
	}
}

// NewOllamaProvider creates a new Ollama provider.
func NewOllamaProvider(baseURL string, opts ...OllamaOption) *OllamaProvider {
	p := &OllamaProvider{
		baseURL: baseURL,
		client:  http.DefaultClient,
	}
	for _, opt := range opts {
		opt(p)
	}
	return p
}

func (p *OllamaProvider) Complete(ctx context.Context, req CompletionRequest) (CompletionResponse, error) {
	model := req.Model
	if model == "" {
		model = "llama3:8b"
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

	body, err := json.Marshal(oaiReq)
	if err != nil {
		return CompletionResponse{}, fmt.Errorf("marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, p.baseURL+"/v1/chat/completions", bytes.NewReader(body))
	if err != nil {
		return CompletionResponse{}, fmt.Errorf("create request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := p.client.Do(httpReq)
	if err != nil {
		return CompletionResponse{}, fmt.Errorf("send request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return CompletionResponse{}, fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return CompletionResponse{}, fmt.Errorf("ollama api error (status %d): %s", resp.StatusCode, string(respBody))
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

func (p *OllamaProvider) StreamComplete(ctx context.Context, req CompletionRequest) (<-chan StreamChunk, error) {
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

func (p *OllamaProvider) Models() []ModelInfo {
	if p.models != nil {
		return p.models
	}
	return []ModelInfo{
		{ID: "llama3:8b", Name: "Llama 3 8B", MaxTokens: 8192, Description: "Free self-hosted model via Ollama"},
	}
}

func (p *OllamaProvider) HealthCheck(ctx context.Context) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, p.baseURL+"/api/tags", nil)
	if err != nil {
		return err
	}

	resp, err := p.client.Do(req)
	if err != nil {
		return fmt.Errorf("health check failed: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("health check returned status %d", resp.StatusCode)
	}
	return nil
}
