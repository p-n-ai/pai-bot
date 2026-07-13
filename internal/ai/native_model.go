// Copyright 2026 the P&AI authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package ai

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/p-n-ai/pai-bot/internal/llm"
)

// NativeProvider preserves native assistant and tool messages for agent turns.
type NativeProvider interface {
	CompleteNative(context.Context, string, llm.Context, *llm.StreamOptions) (llm.AssistantMessage, error)
}

// NativeModelConfig fixes routing policy around one provider-neutral native model port.
type NativeModelConfig struct {
	Task  TaskType
	Model string
}

// NativeModel routes native-message calls through the existing provider fallback policy.
type NativeModel struct {
	router *Router
	config NativeModelConfig
}

// NewNativeModel returns a native-message model backed by router.
func NewNativeModel(router *Router, config NativeModelConfig) *NativeModel {
	return &NativeModel{router: router, config: config}
}

// Complete preserves native messages for capable providers and safely projects tool-free calls for legacy providers.
func (m *NativeModel) Complete(ctx context.Context, c llm.Context, opts *llm.StreamOptions) (llm.AssistantMessage, error) {
	if m == nil || m.router == nil {
		return llm.AssistantMessage{}, errors.New("native AI router is required")
	}
	return m.router.completeNative(ctx, m.config, c, opts)
}

func (r *Router) completeNative(ctx context.Context, config NativeModelConfig, c llm.Context, opts *llm.StreamOptions) (llm.AssistantMessage, error) {
	providers, order, gen := r.snapshotProviders()
	if len(order) == 0 {
		return llm.AssistantMessage{}, errors.New("all AI providers failed (no providers registered)")
	}

	legacyRequest, legacyCompatible := projectNativeCompletionRequest(config, c, opts)
	var failures []string
	for _, name := range order {
		provider := providers[name]
		if provider == nil {
			continue
		}
		if r.isCircuitOpen(name) {
			failures = append(failures, name+": circuit open")
			continue
		}

		modelID := strings.TrimSpace(config.Model)
		if modelID == "" {
			modelID = r.defaultModelForProvider(name)
		}
		startedAt := time.Now()
		var response llm.AssistantMessage
		var err error
		if native, ok := provider.(NativeProvider); ok {
			response, err = r.completeNativeWithRetry(ctx, native, modelID, c, opts)
		} else if legacyCompatible {
			req := legacyRequest
			req.Model = modelID
			var legacyResponse CompletionResponse
			legacyResponse, err = r.completeWithRetry(ctx, provider, req)
			if err == nil {
				response = projectLegacyCompletionResponse(name, legacyResponse)
			}
		} else {
			failures = append(failures, name+": native tool messages unsupported")
			continue
		}
		traceRequest := projectNativeTraceRequest(config, modelID, c, opts)
		trace := CompletionTrace{
			Provider:    name,
			Request:     traceRequest,
			Error:       completionErrorString(err),
			StartedAt:   startedAt,
			CompletedAt: time.Now(),
		}
		if err == nil {
			traceResponse := projectNativeCompletionResponse(response)
			trace.Response = &traceResponse
		}
		r.emitTrace(trace)

		if err != nil {
			if ctxErr := ctx.Err(); ctxErr != nil {
				return llm.AssistantMessage{}, ctxErr
			}
			if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
				return llm.AssistantMessage{}, err
			}
			r.markFailure(name, gen)
			slog.Warn("native AI provider failed, trying next",
				"provider", name,
				"duration_ms", time.Since(startedAt).Milliseconds(),
				"error", true,
			)
			failures = append(failures, name+": request failed")
			continue
		}

		r.markSuccess(name, gen)
		slog.Debug("native AI request completed",
			"provider", name,
			"model", response.ResponseModel,
			"input_tokens", response.Usage.Input+response.Usage.CacheRead+response.Usage.CacheWrite,
			"output_tokens", response.Usage.Output,
			"duration_ms", time.Since(startedAt).Milliseconds(),
		)
		return response, nil
	}

	return llm.AssistantMessage{}, fmt.Errorf("all AI providers failed: %s", strings.Join(failures, "; "))
}

func projectNativeTraceRequest(config NativeModelConfig, modelID string, c llm.Context, opts *llm.StreamOptions) CompletionRequest {
	req := CompletionRequest{Task: config.Task, Model: modelID}
	if c.SystemPrompt != "" {
		req.Messages = append(req.Messages, Message{Role: "system", Content: c.SystemPrompt})
	}
	for _, message := range c.Messages {
		switch typed := message.(type) {
		case llm.UserMessage:
			if projected, ok := projectNativeUserMessage(typed); ok {
				req.Messages = append(req.Messages, projected)
			}
		case llm.AssistantMessage:
			content := nativeAssistantText(typed)
			if content == "" {
				content = "[tool call omitted]"
			}
			req.Messages = append(req.Messages, Message{Role: "assistant", Content: content})
		case llm.ToolResultMessage:
			req.Messages = append(req.Messages, Message{Role: "user", Content: "[tool result omitted]"})
		}
	}
	if opts != nil {
		req.MaxTokens = opts.MaxTokens
		if opts.Temperature != nil {
			req.Temperature = *opts.Temperature
		}
	}
	return req
}

func projectNativeCompletionResponse(response llm.AssistantMessage) CompletionResponse {
	model := response.ResponseModel
	if model == "" {
		model = response.Model
	}
	return CompletionResponse{
		Content:      nativeAssistantText(response),
		Model:        model,
		InputTokens:  response.Usage.Input + response.Usage.CacheRead + response.Usage.CacheWrite,
		OutputTokens: response.Usage.Output,
	}
}

func nativeAssistantText(message llm.AssistantMessage) string {
	var text strings.Builder
	for _, content := range message.Content {
		if block, ok := content.(llm.TextContent); ok {
			text.WriteString(block.Text)
		}
	}
	return text.String()
}

func (r *Router) completeNativeWithRetry(ctx context.Context, provider NativeProvider, model string, c llm.Context, opts *llm.StreamOptions) (llm.AssistantMessage, error) {
	var lastErr error
	attempts := 1 + len(r.retryBackoff)
	for attempt := 1; attempt <= attempts; attempt++ {
		response, err := provider.CompleteNative(ctx, model, c, opts)
		if err == nil {
			return response, nil
		}
		lastErr = err
		if attempt == attempts {
			break
		}
		select {
		case <-ctx.Done():
			return llm.AssistantMessage{}, ctx.Err()
		case <-time.After(r.retryBackoff[attempt-1]):
		}
	}
	return llm.AssistantMessage{}, lastErr
}

func projectNativeCompletionRequest(config NativeModelConfig, c llm.Context, opts *llm.StreamOptions) (CompletionRequest, bool) {
	if len(c.Tools) > 0 {
		return CompletionRequest{}, false
	}
	req := CompletionRequest{Task: config.Task, Model: config.Model}
	if c.SystemPrompt != "" {
		req.Messages = append(req.Messages, Message{Role: "system", Content: c.SystemPrompt})
	}
	for _, message := range c.Messages {
		switch typed := message.(type) {
		case llm.UserMessage:
			projected, ok := projectNativeUserMessage(typed)
			if !ok {
				return CompletionRequest{}, false
			}
			req.Messages = append(req.Messages, projected)
		case llm.AssistantMessage:
			for _, content := range typed.Content {
				if _, ok := content.(llm.TextContent); !ok {
					return CompletionRequest{}, false
				}
			}
			req.Messages = append(req.Messages, Message{Role: "assistant", Content: nativeAssistantText(typed)})
		default:
			return CompletionRequest{}, false
		}
	}
	if opts != nil {
		req.MaxTokens = opts.MaxTokens
		if opts.Temperature != nil {
			req.Temperature = *opts.Temperature
		}
	}
	return req, true
}

func projectNativeUserMessage(message llm.UserMessage) (Message, bool) {
	var projected Message
	projected.Role = "user"
	for _, content := range message.Content {
		switch block := content.(type) {
		case llm.TextContent:
			projected.Content += block.Text
		case llm.ImageURLContent:
			projected.ImageURLs = append(projected.ImageURLs, block.URL)
		case llm.ImageContent:
			if _, err := base64.StdEncoding.DecodeString(block.Data); err != nil {
				return Message{}, false
			}
			projected.ImageURLs = append(projected.ImageURLs, "data:"+block.MimeType+";base64,"+block.Data)
		default:
			return Message{}, false
		}
	}
	return projected, true
}

func projectLegacyCompletionResponse(provider string, response CompletionResponse) llm.AssistantMessage {
	return llm.AssistantMessage{
		Content:       []llm.AssistantContent{llm.TextContent{Text: response.Content}},
		Provider:      provider,
		Model:         response.Model,
		ResponseModel: response.Model,
		Usage: llm.Usage{
			Input:       response.InputTokens,
			Output:      response.OutputTokens,
			TotalTokens: response.TotalTokens(),
		},
		StopReason: llm.StopReasonStop,
		Timestamp:  time.Now(),
	}
}
