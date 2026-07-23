// Copyright 2026 the P&AI authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package ai

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/p-n-ai/pai-bot/internal/llm"
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

type directOpenAIProvider struct {
	*OpenAIProvider
}

var _ Provider = (*directOpenAIProvider)(nil)
var _ NativeProvider = (*directOpenAIProvider)(nil)

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
func NewOpenAIProvider(apiKey string, opts ...OpenAIOption) Provider {
	provider := newOpenAIProvider(apiKey, opts...)
	if provider.name != "openai" {
		return provider
	}
	return &directOpenAIProvider{OpenAIProvider: provider}
}

func newOpenAIProvider(apiKey string, opts ...OpenAIOption) *OpenAIProvider {
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
	return newOpenAIProvider(apiKey, opts...)
}

// openaiRequest is the request body for the OpenAI chat completions API.
type openaiRequest struct {
	Model               string                `json:"model"`
	Messages            []openaiMessage       `json:"messages"`
	Tools               []openaiTool          `json:"tools,omitempty"`
	ResponseFormat      *openaiResponseFormat `json:"response_format,omitempty"`
	MaxTokens           int                   `json:"max_tokens,omitempty"`
	MaxCompletionTokens int                   `json:"max_completion_tokens,omitempty"`
	Temperature         *float64              `json:"temperature,omitempty"`
}

type openaiMessage struct {
	Role       string           `json:"role"`
	Content    any              `json:"content,omitempty"`
	ToolCalls  []openaiToolCall `json:"tool_calls,omitempty"`
	ToolCallID string           `json:"tool_call_id,omitempty"`
}

type openaiToolCall struct {
	ID       string             `json:"id"`
	Type     string             `json:"type"`
	Function openaiToolFunction `json:"function"`
}

type openaiToolFunction struct {
	Name      string `json:"name"`
	Arguments string `json:"arguments"`
}

type openaiTool struct {
	Type     string `json:"type"`
	Function struct {
		Name        string          `json:"name"`
		Description string          `json:"description"`
		Parameters  json.RawMessage `json:"parameters"`
	} `json:"function"`
}

type openaiResponseFormat struct {
	Type       string                      `json:"type"`
	JSONSchema *openaiResponseFormatSchema `json:"json_schema,omitempty"`
}

type openaiResponseFormatSchema struct {
	Name   string          `json:"name"`
	Schema json.RawMessage `json:"schema"`
	Strict bool            `json:"strict,omitempty"`
}

type openaiContentPart struct {
	Type     string           `json:"type"`
	Text     string           `json:"text,omitempty"`
	ImageURL *openaiImagePart `json:"image_url,omitempty"`
}

type openaiImagePart struct {
	URL string `json:"url"`
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

type openaiNativeResponse struct {
	ID      string `json:"id"`
	Choices []struct {
		Message struct {
			Content   string           `json:"content"`
			ToolCalls []openaiToolCall `json:"tool_calls"`
		} `json:"message"`
		FinishReason string `json:"finish_reason"`
	} `json:"choices"`
	Model string `json:"model"`
	Usage struct {
		PromptTokens        int `json:"prompt_tokens"`
		CompletionTokens    int `json:"completion_tokens"`
		PromptTokensDetails *struct {
			CachedTokens int `json:"cached_tokens"`
		} `json:"prompt_tokens_details"`
	} `json:"usage"`
}

func (p *OpenAIProvider) Complete(ctx context.Context, req CompletionRequest) (CompletionResponse, error) {
	model := req.Model
	if model == "" {
		model = "gpt-5.4-mini" // current low-latency default
	}

	oaiReq := openaiRequest{
		Model:    model,
		Messages: buildOpenAIMessages(req.Messages),
	}
	if req.MaxTokens > 0 {
		// Newer OpenAI models (o1+, gpt-5+) reject max_tokens and require
		// max_completion_tokens instead.
		if needsMaxCompletionTokens(model) {
			oaiReq.MaxCompletionTokens = req.MaxTokens
		} else {
			oaiReq.MaxTokens = req.MaxTokens
		}
	}
	if req.Temperature > 0 {
		temp := req.Temperature
		oaiReq.Temperature = &temp
	}
	if err := applyOpenAIStructuredOutput(p.name, &oaiReq, req.StructuredOutput); err != nil {
		return CompletionResponse{}, err
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
	defer func() { _ = resp.Body.Close() }()

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

func (p *directOpenAIProvider) CompleteNative(ctx context.Context, model string, c llm.Context, opts *llm.StreamOptions) (llm.AssistantMessage, error) {
	if model == "" {
		model = "gpt-5.4-mini"
	}
	messages, err := buildNativeOpenAIMessages(c)
	if err != nil {
		return llm.AssistantMessage{}, err
	}
	tools, err := buildNativeOpenAITools(c.Tools)
	if err != nil {
		return llm.AssistantMessage{}, err
	}
	request := openaiRequest{
		Model:    model,
		Messages: messages,
		Tools:    tools,
	}
	if opts != nil {
		if opts.MaxTokens > 0 {
			if needsMaxCompletionTokens(model) {
				request.MaxCompletionTokens = opts.MaxTokens
			} else {
				request.MaxTokens = opts.MaxTokens
			}
		}
		request.Temperature = opts.Temperature
		if opts.StructuredOutput != nil {
			spec := &StructuredOutputSpec{
				Name:       opts.StructuredOutput.Name,
				JSONSchema: append(json.RawMessage(nil), opts.StructuredOutput.JSONSchema...),
				Strict:     opts.StructuredOutput.Strict,
			}
			if err := applyOpenAIStructuredOutput(p.name, &request, spec); err != nil {
				return llm.AssistantMessage{}, err
			}
		}
	}

	body, err := json.Marshal(request)
	if err != nil {
		return llm.AssistantMessage{}, fmt.Errorf("marshal native OpenAI request: %w", err)
	}
	httpRequest, err := http.NewRequestWithContext(ctx, http.MethodPost, strings.TrimRight(p.baseURL, "/")+"/chat/completions", bytes.NewReader(body))
	if err != nil {
		return llm.AssistantMessage{}, fmt.Errorf("create native OpenAI request: %w", err)
	}
	httpRequest.Header.Set("Content-Type", "application/json")
	httpRequest.Header.Set("Authorization", "Bearer "+p.apiKey)
	if opts != nil {
		for name, value := range opts.Headers {
			httpRequest.Header.Set(name, value)
		}
	}

	response, err := p.client.Do(httpRequest)
	if err != nil {
		if ctxErr := ctx.Err(); ctxErr != nil {
			return llm.AssistantMessage{}, ctxErr
		}
		return llm.AssistantMessage{}, fmt.Errorf("send native OpenAI request: %w", err)
	}
	defer func() { _ = response.Body.Close() }()

	responseBody, err := io.ReadAll(response.Body)
	if err != nil {
		return llm.AssistantMessage{}, fmt.Errorf("read native OpenAI response: %w", err)
	}
	if response.StatusCode != http.StatusOK {
		return llm.AssistantMessage{}, fmt.Errorf("native OpenAI API returned status %d", response.StatusCode)
	}

	var decoded openaiNativeResponse
	if err := json.Unmarshal(responseBody, &decoded); err != nil {
		return llm.AssistantMessage{}, fmt.Errorf("decode native OpenAI response: %w", err)
	}
	if len(decoded.Choices) == 0 {
		return llm.AssistantMessage{}, fmt.Errorf("native OpenAI response contains no choices")
	}
	return projectNativeOpenAIResponse(model, decoded)
}

func buildNativeOpenAIMessages(c llm.Context) ([]openaiMessage, error) {
	messages := make([]openaiMessage, 0, len(c.Messages)+1)
	if c.SystemPrompt != "" {
		messages = append(messages, openaiMessage{Role: "system", Content: c.SystemPrompt})
	}
	for _, message := range c.Messages {
		switch typed := message.(type) {
		case llm.SystemMessage:
			messages = append(messages, openaiMessage{Role: "system", Content: typed.Content})
		case llm.UserMessage:
			projected, ok := projectNativeUserMessage(typed)
			if !ok {
				return nil, fmt.Errorf("native OpenAI user message contains invalid image data")
			}
			messages = append(messages, buildOpenAIMessages([]Message{projected})...)
		case llm.AssistantMessage:
			projected, err := buildNativeOpenAIAssistantMessage(typed)
			if err != nil {
				return nil, err
			}
			if projected.Content != nil || len(projected.ToolCalls) > 0 {
				messages = append(messages, projected)
			}
		case llm.ToolResultMessage:
			content, err := nativeOpenAIToolResultText(typed)
			if err != nil {
				return nil, err
			}
			messages = append(messages, openaiMessage{
				Role:       "tool",
				Content:    content,
				ToolCallID: typed.ToolCallID,
			})
		default:
			return nil, fmt.Errorf("native OpenAI message type %T is unsupported", message)
		}
	}
	return messages, nil
}

func buildNativeOpenAIAssistantMessage(message llm.AssistantMessage) (openaiMessage, error) {
	var text strings.Builder
	projected := openaiMessage{Role: "assistant"}
	for _, content := range message.Content {
		switch block := content.(type) {
		case llm.TextContent:
			text.WriteString(block.Text)
		case llm.ToolCall:
			arguments, err := json.Marshal(block.Arguments)
			if err != nil {
				return openaiMessage{}, fmt.Errorf("native OpenAI tool call %q arguments: %w", block.Name, err)
			}
			if block.Arguments == nil {
				arguments = []byte("{}")
			}
			projected.ToolCalls = append(projected.ToolCalls, openaiToolCall{
				ID:   block.ID,
				Type: "function",
				Function: openaiToolFunction{
					Name:      block.Name,
					Arguments: string(arguments),
				},
			})
		case llm.ThinkingContent:
		default:
			return openaiMessage{}, fmt.Errorf("native OpenAI assistant content type %T is unsupported", content)
		}
	}
	if text.Len() > 0 {
		projected.Content = text.String()
	}
	return projected, nil
}

func nativeOpenAIToolResultText(message llm.ToolResultMessage) (string, error) {
	var text []string
	for _, content := range message.Content {
		block, ok := content.(llm.TextContent)
		if !ok {
			return "", fmt.Errorf("native OpenAI tool result content type %T is unsupported", content)
		}
		text = append(text, block.Text)
	}
	if len(text) == 0 {
		return "(no text result)", nil
	}
	return strings.Join(text, "\n"), nil
}

func buildNativeOpenAITools(tools []llm.Tool) ([]openaiTool, error) {
	projected := make([]openaiTool, len(tools))
	for index, tool := range tools {
		var parameters map[string]any
		if err := json.Unmarshal(tool.Parameters, &parameters); err != nil {
			return nil, fmt.Errorf("native OpenAI tool %q parameters: %w", tool.Name, err)
		}
		if parameters == nil {
			return nil, fmt.Errorf("native OpenAI tool %q parameters must be a JSON object", tool.Name)
		}
		projected[index].Type = "function"
		projected[index].Function.Name = tool.Name
		projected[index].Function.Description = tool.Description
		projected[index].Function.Parameters = append(json.RawMessage(nil), tool.Parameters...)
	}
	return projected, nil
}

func projectNativeOpenAIResponse(requestModel string, response openaiNativeResponse) (llm.AssistantMessage, error) {
	choice := response.Choices[0]
	content := make([]llm.AssistantContent, 0, 1+len(choice.Message.ToolCalls))
	if choice.Message.Content != "" {
		content = append(content, llm.TextContent{Text: choice.Message.Content})
	}
	for _, toolCall := range choice.Message.ToolCalls {
		var arguments map[string]any
		encoded := strings.TrimSpace(toolCall.Function.Arguments)
		if encoded == "" {
			arguments = map[string]any{}
		} else if err := json.Unmarshal([]byte(encoded), &arguments); err != nil {
			return llm.AssistantMessage{}, fmt.Errorf("native OpenAI tool call %q arguments: %w", toolCall.Function.Name, err)
		} else if arguments == nil {
			return llm.AssistantMessage{}, fmt.Errorf("native OpenAI tool call %q arguments must be a JSON object", toolCall.Function.Name)
		}
		content = append(content, llm.ToolCall{
			ID:        toolCall.ID,
			Name:      toolCall.Function.Name,
			Arguments: arguments,
		})
	}
	stopReason, err := nativeOpenAIStopReason(choice.FinishReason)
	if err != nil {
		return llm.AssistantMessage{}, err
	}
	cacheRead := 0
	if response.Usage.PromptTokensDetails != nil {
		cacheRead = response.Usage.PromptTokensDetails.CachedTokens
	}
	input := max(0, response.Usage.PromptTokens-cacheRead)
	responseModel := response.Model
	if responseModel == "" {
		responseModel = requestModel
	}
	return llm.AssistantMessage{
		Content:       content,
		API:           llm.APIOpenAICompletions,
		Provider:      "openai",
		Model:         requestModel,
		ResponseModel: responseModel,
		ResponseID:    response.ID,
		Usage: llm.Usage{
			Input:       input,
			Output:      response.Usage.CompletionTokens,
			CacheRead:   cacheRead,
			TotalTokens: response.Usage.PromptTokens + response.Usage.CompletionTokens,
		},
		StopReason: stopReason,
		Timestamp:  time.Now(),
	}, nil
}

func nativeOpenAIStopReason(reason string) (llm.StopReason, error) {
	switch reason {
	case "stop":
		return llm.StopReasonStop, nil
	case "length":
		return llm.StopReasonLength, nil
	case "tool_calls", "function_call":
		return llm.StopReasonToolUse, nil
	default:
		return llm.StopReasonError, fmt.Errorf("native OpenAI finish reason %q is unsupported", reason)
	}
}

func applyOpenAIStructuredOutput(providerName string, oaiReq *openaiRequest, spec *StructuredOutputSpec) error {
	if spec == nil {
		return nil
	}
	if spec.Name == "" {
		return fmt.Errorf("structured output name is required")
	}
	if len(spec.JSONSchema) == 0 {
		return fmt.Errorf("structured output JSON schema is required")
	}

	if providerName == "deepseek" {
		oaiReq.ResponseFormat = &openaiResponseFormat{Type: "json_object"}
		oaiReq.Messages = append([]openaiMessage{{
			Role:    "system",
			Content: fmt.Sprintf("Return a JSON object only that matches this schema: %s", string(spec.JSONSchema)),
		}}, oaiReq.Messages...)
		return nil
	}

	oaiReq.ResponseFormat = &openaiResponseFormat{
		Type: "json_schema",
		JSONSchema: &openaiResponseFormatSchema{
			Name:   spec.Name,
			Schema: spec.JSONSchema,
			Strict: spec.Strict,
		},
	}
	return nil
}

// needsMaxCompletionTokens returns true for model families that reject the
// legacy max_tokens parameter and require max_completion_tokens instead.
func needsMaxCompletionTokens(model string) bool {
	switch {
	case strings.HasPrefix(model, "o1"),
		strings.HasPrefix(model, "o3"),
		strings.HasPrefix(model, "o4"),
		strings.HasPrefix(model, "gpt-4.1"),
		strings.HasPrefix(model, "gpt-5"):
		return true
	default:
		return false
	}
}

func buildOpenAIMessages(messages []Message) []openaiMessage {
	out := make([]openaiMessage, 0, len(messages))
	for _, m := range messages {
		if len(m.ImageURLs) == 0 {
			out = append(out, openaiMessage{
				Role:    m.Role,
				Content: m.Content,
			})
			continue
		}

		parts := make([]openaiContentPart, 0, 1+len(m.ImageURLs))
		if m.Content != "" {
			parts = append(parts, openaiContentPart{
				Type: "text",
				Text: m.Content,
			})
		}
		for _, imageURL := range m.ImageURLs {
			if imageURL == "" {
				continue
			}
			parts = append(parts, openaiContentPart{
				Type: "image_url",
				ImageURL: &openaiImagePart{
					URL: imageURL,
				},
			})
		}

		out = append(out, openaiMessage{
			Role:    m.Role,
			Content: parts,
		})
	}
	return out
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
		{ID: "gpt-5.4", Name: "GPT-5.4", MaxTokens: 128000, Description: "Frontier OpenAI model for complex work"},
		{ID: "gpt-5.4-mini", Name: "GPT-5.4 mini", MaxTokens: 128000, Description: "Current fast OpenAI default"},
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
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("health check returned status %d", resp.StatusCode)
	}
	return nil
}
