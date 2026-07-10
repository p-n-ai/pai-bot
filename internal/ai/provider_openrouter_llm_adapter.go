// Copyright 2026 the P&AI authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package ai

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/p-n-ai/pai-bot/internal/llm"
)

const (
	openRouterLLMDefaultBaseURL = "https://openrouter.ai/api/v1"
	openRouterLLMDefaultModel   = "qwen/qwen3-max"
)

var errOpenRouterLLMCompletion = errors.New("openrouter completion failed")

type openRouterLLMAdapter struct {
	apiKey  string
	baseURL string
}

var _ Provider = (*openRouterLLMAdapter)(nil)

// NewOpenRouterLLMAdapter adapts the native llm OpenRouter path to Provider.
func NewOpenRouterLLMAdapter(apiKey string) Provider {
	return newOpenRouterLLMAdapter(apiKey, openRouterLLMDefaultBaseURL)
}

func newOpenRouterLLMAdapter(apiKey, baseURL string) *openRouterLLMAdapter {
	return &openRouterLLMAdapter{apiKey: apiKey, baseURL: baseURL}
}

func (p *openRouterLLMAdapter) Complete(ctx context.Context, req CompletionRequest) (CompletionResponse, error) {
	modelID := req.Model
	if modelID == "" {
		modelID = openRouterLLMDefaultModel
	}

	completionContext, err := projectOpenRouterLLMContext(req.Messages)
	if err != nil {
		return CompletionResponse{}, err
	}
	options := projectOpenRouterLLMOptions(p.apiKey, req)
	message, err := llm.StreamOpenRouterChat(ctx, llm.Model{
		ID:       modelID,
		API:      llm.APIOpenRouterChat,
		Provider: "openrouter",
		BaseURL:  p.baseURL,
	}, completionContext, &options).Result()
	if err != nil {
		if ctxErr := ctx.Err(); ctxErr != nil {
			return CompletionResponse{}, ctxErr
		}
		return CompletionResponse{}, errOpenRouterLLMCompletion
	}

	responseModel := message.ResponseModel
	if responseModel == "" {
		responseModel = message.Model
	}
	return CompletionResponse{
		Content:      openRouterLLMText(message),
		Model:        responseModel,
		InputTokens:  message.Usage.Input + message.Usage.CacheRead + message.Usage.CacheWrite,
		OutputTokens: message.Usage.Output,
	}, nil
}

func projectOpenRouterLLMContext(messages []Message) (llm.Context, error) {
	projected := llm.Context{Messages: make([]llm.Message, 0, len(messages))}
	for i, message := range messages {
		hasNonEmptyImage := false
		for _, rawImage := range message.ImageURLs {
			if rawImage != "" {
				hasNonEmptyImage = true
				break
			}
		}
		switch message.Role {
		case "system":
			if hasNonEmptyImage {
				return llm.Context{}, errors.New("system message cannot contain images")
			}
			projected.Messages = append(projected.Messages, llm.SystemMessage{Content: message.Content})
		case "user":
			content := make([]llm.UserContent, 0, 1+len(message.ImageURLs))
			if len(message.ImageURLs) == 0 || message.Content != "" {
				content = append(content, llm.TextContent{Text: message.Content})
			}
			for _, rawImage := range message.ImageURLs {
				if rawImage == "" {
					continue
				}
				image, err := projectOpenRouterLLMImage(rawImage)
				if err != nil {
					return llm.Context{}, err
				}
				content = append(content, image)
			}
			projected.Messages = append(projected.Messages, llm.UserMessage{Content: content})
		case "assistant":
			if hasNonEmptyImage {
				return llm.Context{}, errors.New("assistant message cannot contain images")
			}
			projected.Messages = append(projected.Messages, llm.AssistantMessage{
				Content: []llm.AssistantContent{llm.TextContent{Text: message.Content}},
			})
		default:
			return llm.Context{}, fmt.Errorf("unsupported message role at index %d", i)
		}
	}
	return projected, nil
}

func projectOpenRouterLLMImage(raw string) (llm.UserContent, error) {
	if !strings.HasPrefix(raw, "data:") {
		return llm.ImageURLContent{URL: raw}, nil
	}
	metadata, _, hasPayload := strings.Cut(raw, ",")
	if !hasPayload || !strings.HasSuffix(metadata, ";base64") {
		return nil, errors.New("invalid or unsupported image data URL")
	}
	image, err := parseDataURLImage(raw)
	if err != nil {
		return nil, errors.New("invalid or unsupported image data URL")
	}
	return llm.ImageContent{
		Data:     base64.StdEncoding.EncodeToString(image.Data),
		MimeType: image.MIMEType,
	}, nil
}

func projectOpenRouterLLMOptions(apiKey string, req CompletionRequest) llm.StreamOptions {
	options := llm.StreamOptions{
		APIKey:    apiKey,
		MaxTokens: req.MaxTokens,
	}
	if req.Temperature > 0 {
		temperature := req.Temperature
		options.Temperature = &temperature
	}
	if req.StructuredOutput != nil {
		options.StructuredOutput = &llm.StructuredOutputSpec{
			Name:       req.StructuredOutput.Name,
			JSONSchema: append([]byte(nil), req.StructuredOutput.JSONSchema...),
			Strict:     req.StructuredOutput.Strict,
		}
	}
	return options
}

func openRouterLLMText(message llm.AssistantMessage) string {
	var text strings.Builder
	for _, content := range message.Content {
		if block, ok := content.(llm.TextContent); ok {
			text.WriteString(block.Text)
		}
	}
	return text.String()
}

func (p *openRouterLLMAdapter) StreamComplete(ctx context.Context, req CompletionRequest) (<-chan StreamChunk, error) {
	response, err := p.Complete(ctx, req)
	if err != nil {
		return nil, err
	}
	chunks := make(chan StreamChunk, 1)
	chunks <- StreamChunk{Content: response.Content, Done: true}
	close(chunks)
	return chunks, nil
}

func (p *openRouterLLMAdapter) Models() []ModelInfo {
	return []ModelInfo{{
		ID:          openRouterLLMDefaultModel,
		Name:        "Qwen3 Max",
		MaxTokens:   262144,
		Description: "Current general-purpose OpenRouter default",
	}}
}

func (p *openRouterLLMAdapter) HealthCheck(ctx context.Context) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, strings.TrimRight(p.baseURL, "/")+"/models", nil)
	if err != nil {
		return errors.New("openrouter health check request is invalid")
	}
	req.Header.Set("Authorization", "Bearer "+p.apiKey)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		if ctxErr := ctx.Err(); ctxErr != nil {
			return ctxErr
		}
		return errors.New("openrouter health check failed")
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("openrouter health check returned status %d", resp.StatusCode)
	}
	return nil
}
