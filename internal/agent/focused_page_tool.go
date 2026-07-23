// Copyright 2026 the P&AI authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/p-n-ai/pai-bot/internal/agentcore"
	"github.com/p-n-ai/pai-bot/internal/ai"
	"github.com/p-n-ai/pai-bot/internal/chat"
	"github.com/p-n-ai/pai-bot/internal/focusedpage"
	"github.com/p-n-ai/pai-bot/internal/llm"
	"github.com/p-n-ai/pai-bot/internal/platform/featureflags"
)

const createFocusedPageToolName = "create_focused_page"

func (e *Engine) completeTeachingTurn(ctx context.Context, turn *agentTurn, messages []ai.Message, model string) (teachingCompletion, *focusedpage.Artifact, error) {
	focusedConfigured := e.focusedPages != nil && e.focusedPageEnabled(chat.InboundMessage{
		Channel: turn.Channel,
		UserID:  turn.UserID,
	})
	if focusedConfigured && !e.aiRouter.HasNativeProvider() {
		completion, err := e.completeTextTeachingTurn(ctx, messages, model)
		return completion, nil, err
	}
	if !focusedConfigured && !e.featureFlags().Enabled(featureflags.AgentCore) {
		completion, err := e.completeTextTeachingTurn(ctx, messages, model)
		return completion, nil, err
	}
	if !focusedConfigured {
		completion, err := e.completeNativeTeachingTurn(ctx, turn, model)
		return completion, nil, err
	}
	ownerUserID, err := e.store.ResolveUserUUID(turn.UserID)
	if err != nil || ownerUserID == "" {
		return teachingCompletion{}, nil, fmt.Errorf("resolve focused page owner")
	}
	recipientName, _ := e.store.GetUserName(turn.UserID)
	tool := &createFocusedPageTool{
		service: e.focusedPages,
		input: focusedpage.CreateInput{
			TenantID: e.tenantID, OwnerUserID: ownerUserID, ConversationID: turn.ConversationID,
			TurnID: turn.ID, RecipientName: recipientName,
		},
	}
	tools := []agentcore.Tool{tool}
	if e.featureFlags().Enabled(featureflags.AgentCore) {
		tools = append(e.teachingTools(), tools...)
	}
	completion, err := e.completeNativeTeachingTurnWithTools(ctx, turn, model, tools)
	if err != nil {
		return teachingCompletion{}, nil, err
	}
	if tool.artifact != nil && strings.TrimSpace(completion.Content) == "" {
		completion.Content = "Your focused page is ready."
	}
	return completion, tool.artifact, nil
}

func (e *Engine) completeTextTeachingTurn(ctx context.Context, messages []ai.Message, model string) (teachingCompletion, error) {
	response, err := e.aiRouter.Complete(ctx, ai.CompletionRequest{Messages: messages, Model: model, Task: ai.TaskTeaching, MaxTokens: 1024})
	return teachingCompletion{
		Content: response.Content, Model: response.Model,
		InputTokens: response.InputTokens, OutputTokens: response.OutputTokens,
	}, err
}

type createFocusedPageTool struct {
	service      *focusedpage.Service
	input        focusedpage.CreateInput
	firstMessage string
	artifact     *focusedpage.Artifact
}

func (t *createFocusedPageTool) Definition() llm.Tool {
	return llm.Tool{
		Name:        createFocusedPageToolName,
		Description: "Create one private, read-only focused message page for a goal or report. Use at most once in a turn. The server owns its recipient, layout, action, lifetime, capability, URL, and channel delivery.",
		Parameters:  json.RawMessage(`{"type":"object","additionalProperties":false,"properties":{"message":{"type":"string","minLength":1,"maxLength":4000}},"required":["message"]}`),
	}
}

func (t *createFocusedPageTool) Execute(ctx context.Context, call llm.ToolCall) (llm.ToolResultMessage, error) {
	message, ok := call.Arguments["message"].(string)
	if !ok || len(call.Arguments) != 1 {
		return focusedPageToolResult("Invalid arguments: message must be the only field.", true), nil
	}
	parsed, err := focusedpage.ParseMessage(message)
	if err != nil {
		return focusedPageToolResult("Invalid focused page message.", true), nil
	}
	if t.artifact != nil {
		if parsed == t.firstMessage {
			return focusedPageToolResult("Focused page already created for this turn.", false), nil
		}
		return focusedPageToolResult("Only one focused page may be created per turn.", true), nil
	}
	t.input.Message = parsed
	artifact, err := t.service.Create(ctx, t.input)
	if err != nil {
		return focusedPageToolResult("Focused page creation failed.", true), nil
	}
	t.firstMessage = parsed
	t.artifact = &artifact
	return focusedPageToolResult("Focused page created.", false), nil
}

func focusedPageToolResult(message string, isError bool) llm.ToolResultMessage {
	return llm.ToolResultMessage{Content: []llm.UserContent{llm.TextContent{Text: message}}, IsError: isError, Timestamp: time.Now()}
}
