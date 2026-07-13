// Copyright 2026 the P&AI authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package agent

import (
	"encoding/base64"
	"errors"
	"fmt"
	"strings"

	"github.com/p-n-ai/pai-bot/internal/ai"
	"github.com/p-n-ai/pai-bot/internal/llm"
)

func (e *Engine) buildNativeContextFromTurn(turn *agentTurn) (llm.Context, error) {
	messages := e.buildPromptMessagesFromTurn(turn)
	native := llm.Context{Messages: make([]llm.Message, 0, len(messages))}
	var systemPrompt strings.Builder
	for _, message := range messages {
		switch message.Role {
		case "system":
			if message.Content == "" {
				continue
			}
			if systemPrompt.Len() > 0 {
				systemPrompt.WriteString("\n\n")
			}
			systemPrompt.WriteString(message.Content)
		case "user":
			content, err := nativeUserContent(message)
			if err != nil {
				return llm.Context{}, err
			}
			native.Messages = append(native.Messages, llm.UserMessage{Content: content})
		case "assistant":
			native.Messages = append(native.Messages, llm.AssistantMessage{Content: []llm.AssistantContent{llm.TextContent{Text: message.Content}}})
		default:
			return llm.Context{}, errors.New("unsupported tutor prompt role")
		}
	}
	native.SystemPrompt = systemPrompt.String()
	if native.SystemPrompt == "" {
		return llm.Context{}, errors.New("tutor system prompt is required")
	}
	return native, nil
}

func nativeUserContent(message ai.Message) ([]llm.UserContent, error) {
	content := make([]llm.UserContent, 0, 1+len(message.ImageURLs))
	if len(message.ImageURLs) == 0 || message.Content != "" {
		content = append(content, llm.TextContent{Text: message.Content})
	}
	for _, rawImage := range message.ImageURLs {
		if rawImage == "" {
			continue
		}
		if !strings.HasPrefix(rawImage, "data:") {
			content = append(content, llm.ImageURLContent{URL: rawImage})
			continue
		}
		metadata, payload, ok := strings.Cut(rawImage, ",")
		if !ok || !strings.HasSuffix(metadata, ";base64") {
			return nil, errors.New("invalid image data URL")
		}
		mimeType := strings.TrimSuffix(strings.TrimPrefix(metadata, "data:"), ";base64")
		if mimeType == "" {
			return nil, errors.New("image MIME type is required")
		}
		if _, err := base64.StdEncoding.DecodeString(payload); err != nil {
			return nil, fmt.Errorf("decode image data: %w", err)
		}
		content = append(content, llm.ImageContent{Data: payload, MimeType: mimeType})
	}
	return content, nil
}
