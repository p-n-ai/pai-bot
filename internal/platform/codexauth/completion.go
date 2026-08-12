// Copyright 2026 the P&AI authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package codexauth

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/p-n-ai/pai-bot/internal/ai"
	"github.com/p-n-ai/pai-bot/internal/jsonobject"
)

const codexChatInstructions = `Act only as PaiBot's chat completion engine. Answer from the supplied conversation and instructions. Do not inspect files, run commands, call tools, ask for approval, or modify the environment. Return only the assistant response.`

type agentMessageItem struct {
	Type  string `json:"type"`
	Text  string `json:"text"`
	Phase string `json:"phase"`
}

// Complete runs one ephemeral, non-interactive Codex thread.
func (m *Manager) Complete(
	ctx context.Context,
	request ai.CompletionRequest,
) (ai.CompletionResponse, error) {
	if err := m.Refresh(ctx); err != nil {
		return ai.CompletionResponse{}, errors.New("codex login is not connected; reconnect it in Admin")
	}

	m.mu.RLock()
	workspace := m.workspace
	m.mu.RUnlock()
	if workspace == "" {
		return ai.CompletionResponse{}, errors.New("codex runtime workspace is unavailable")
	}

	baseInstructions, messages := splitInstructions(request.Messages)
	threadParams := jsonobject.New(
		jsonobject.Member("approvalPolicy", "never"),
		jsonobject.Member("cwd", workspace),
		jsonobject.Member("developerInstructions", codexChatInstructions),
		jsonobject.Member("ephemeral", true),
		jsonobject.Member("model", strings.TrimSpace(request.Model)),
		jsonobject.Member("sandbox", "read-only"),
	)
	if baseInstructions != "" {
		threadParams = threadParams.With(jsonobject.Member("baseInstructions", baseInstructions))
	}
	threadRaw, err := m.call(ctx, "thread/start", threadParams)
	if err != nil {
		return ai.CompletionResponse{}, fmt.Errorf("start Codex chat: %w", err)
	}
	var started struct {
		Thread struct {
			ID string `json:"id"`
		} `json:"thread"`
		Model string `json:"model"`
	}
	if json.Unmarshal(threadRaw, &started) != nil || strings.TrimSpace(started.Thread.ID) == "" {
		return ai.CompletionResponse{}, errors.New("decode Codex chat")
	}

	threadID := started.Thread.ID
	state := &completionState{waiter: make(chan completionResult, 1)}
	m.mu.Lock()
	m.completions[threadID] = state
	m.mu.Unlock()

	turnParams := jsonobject.New(
		jsonobject.Member("approvalPolicy", "never"),
		jsonobject.Member("input", completionInput(messages)),
		jsonobject.Member("threadId", threadID),
	)
	if request.StructuredOutput != nil {
		schema, err := jsonobject.Parse(request.StructuredOutput.JSONSchema)
		if err != nil {
			m.removeCompletion(threadID)
			return ai.CompletionResponse{}, errors.New("invalid Codex output schema")
		}
		turnParams = turnParams.With(jsonobject.Member("outputSchema", schema))
	}
	if _, err := m.call(ctx, "turn/start", turnParams); err != nil {
		m.removeCompletion(threadID)
		return ai.CompletionResponse{}, fmt.Errorf("start Codex response: %w", err)
	}

	select {
	case completed := <-state.waiter:
		if completed.err != nil {
			return ai.CompletionResponse{}, completed.err
		}
		if completed.response.Model == "" {
			completed.response.Model = firstNonEmpty(started.Model, request.Model)
		}
		if request.StructuredOutput != nil {
			completed.response.StructuredOutput = json.RawMessage(completed.response.Content)
		}
		return completed.response, nil
	case <-ctx.Done():
		m.removeCompletion(threadID)
		_, _ = m.callWithTimeout("turn/interrupt", jsonobject.New(jsonobject.Member("threadId", threadID)))
		return ai.CompletionResponse{}, ctx.Err()
	}
}

func (m *Manager) callWithTimeout(method string, params jsonobject.Object) (json.RawMessage, error) {
	ctx, cancel := context.WithTimeout(m.parent, requestTimeout)
	defer cancel()
	return m.call(ctx, method, params)
}

func (m *Manager) removeCompletion(threadID string) {
	m.mu.Lock()
	delete(m.completions, threadID)
	m.mu.Unlock()
}

func (m *Manager) handleTokenUsage(params json.RawMessage) {
	var updated struct {
		ThreadID   string `json:"threadId"`
		TokenUsage struct {
			Last struct {
				InputTokens  int `json:"inputTokens"`
				OutputTokens int `json:"outputTokens"`
			} `json:"last"`
		} `json:"tokenUsage"`
	}
	if json.Unmarshal(params, &updated) != nil {
		return
	}
	m.mu.Lock()
	if state := m.completions[updated.ThreadID]; state != nil {
		state.inputTokens = updated.TokenUsage.Last.InputTokens
		state.outputTokens = updated.TokenUsage.Last.OutputTokens
	}
	m.mu.Unlock()
}

func (m *Manager) handleItemCompleted(params json.RawMessage) {
	var completed struct {
		ThreadID string           `json:"threadId"`
		Item     agentMessageItem `json:"item"`
	}
	if json.Unmarshal(params, &completed) != nil ||
		completed.ThreadID == "" ||
		completed.Item.Type != "agentMessage" ||
		strings.TrimSpace(completed.Item.Text) == "" {
		return
	}
	m.mu.Lock()
	if state := m.completions[completed.ThreadID]; state != nil {
		state.messages = append(state.messages, completed.Item)
	}
	m.mu.Unlock()
}

func (m *Manager) handleTurnCompleted(params json.RawMessage) {
	var completed struct {
		ThreadID string `json:"threadId"`
		Turn     struct {
			Status string             `json:"status"`
			Items  []agentMessageItem `json:"items"`
		} `json:"turn"`
	}
	if json.Unmarshal(params, &completed) != nil || completed.ThreadID == "" {
		return
	}

	m.mu.Lock()
	state := m.completions[completed.ThreadID]
	delete(m.completions, completed.ThreadID)
	m.mu.Unlock()
	if state == nil {
		return
	}
	if completed.Turn.Status != "completed" {
		state.waiter <- completionResult{err: errors.New("codex response failed")}
		return
	}
	messages := completed.Turn.Items
	if len(messages) == 0 {
		messages = state.messages
	}
	content := finalAgentMessage(messages)
	if content == "" {
		state.waiter <- completionResult{err: errors.New("codex returned an empty response")}
		return
	}
	state.waiter <- completionResult{response: ai.CompletionResponse{
		Content:      content,
		InputTokens:  state.inputTokens,
		OutputTokens: state.outputTokens,
	}}
}

func splitInstructions(messages []ai.Message) (string, []ai.Message) {
	instructions := make([]string, 0)
	conversation := make([]ai.Message, 0, len(messages))
	for _, message := range messages {
		if strings.EqualFold(strings.TrimSpace(message.Role), "system") {
			if content := strings.TrimSpace(message.Content); content != "" {
				instructions = append(instructions, content)
			}
			continue
		}
		conversation = append(conversation, message)
	}
	return strings.Join(instructions, "\n\n"), conversation
}

func completionInput(messages []ai.Message) []jsonobject.Object {
	var transcript strings.Builder
	input := make([]jsonobject.Object, 0, 1+len(messages))
	for _, message := range messages {
		role := strings.ToUpper(strings.TrimSpace(message.Role))
		if role == "" {
			role = "USER"
		}
		if transcript.Len() > 0 {
			transcript.WriteString("\n\n")
		}
		_, _ = fmt.Fprintf(&transcript, "%s:\n%s", role, strings.TrimSpace(message.Content))
		for _, imageURL := range message.ImageURLs {
			if strings.TrimSpace(imageURL) != "" {
				input = append(input, jsonobject.New(jsonobject.Member("type", "image"), jsonobject.Member("url", imageURL)))
			}
		}
	}
	if transcript.Len() == 0 {
		transcript.WriteString("USER:\n")
	}
	textInput := jsonobject.New(jsonobject.Member("type", "text"), jsonobject.Member("text", transcript.String()))
	return append([]jsonobject.Object{textInput}, input...)
}

func finalAgentMessage(items []agentMessageItem) string {
	var final string
	var messages []string
	for _, item := range items {
		if item.Type != "agentMessage" || strings.TrimSpace(item.Text) == "" {
			continue
		}
		messages = append(messages, strings.TrimSpace(item.Text))
		if item.Phase == "final_answer" {
			final = strings.TrimSpace(item.Text)
		}
	}
	if final != "" {
		return final
	}
	return strings.Join(messages, "\n")
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if value = strings.TrimSpace(value); value != "" {
			return value
		}
	}
	return "gpt-5.4"
}
