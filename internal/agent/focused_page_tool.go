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
	"github.com/p-n-ai/pai-bot/internal/focusedpage"
	"github.com/p-n-ai/pai-bot/internal/llm"
)

const createFocusedPageToolName = "create_focused_page"

type teachingCompletion struct {
	Content      string
	Model        string
	InputTokens  int
	OutputTokens int
}

func (e *Engine) completeTeachingTurn(ctx context.Context, turn *agentTurn, messages []ai.Message, model string) (teachingCompletion, *focusedpage.Artifact, error) {
	if e.focusedPages == nil || turn.Channel != "telegram" || !e.aiRouter.HasNativeProvider() {
		resp, err := e.aiRouter.Complete(ctx, ai.CompletionRequest{Messages: messages, Model: model, Task: ai.TaskTeaching, MaxTokens: 1024})
		return teachingCompletion{Content: resp.Content, Model: resp.Model, InputTokens: resp.InputTokens, OutputTokens: resp.OutputTokens}, nil, err
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
	coreModel := routerAgentModel{router: e.aiRouter, model: model}
	core, err := agentcore.New(coreModel, []agentcore.Tool{tool}, agentcore.DefaultMaxModelCalls)
	if err != nil {
		return teachingCompletion{}, nil, err
	}
	nativeContext, err := nativeTeachingContext(messages)
	if err != nil {
		return teachingCompletion{}, nil, err
	}
	result, err := core.Run(ctx, nativeContext)
	if err != nil {
		return teachingCompletion{}, nil, err
	}
	inputTokens, outputTokens := transcriptUsage(result.Transcript)
	completion := teachingCompletion{
		Content: agentcore.FinalText(result.Final), Model: result.Final.ResponseModel,
		InputTokens: inputTokens, OutputTokens: outputTokens,
	}
	if completion.Model == "" {
		completion.Model = result.Final.Model
	}
	return completion, tool.artifact, nil
}

type routerAgentModel struct {
	router *ai.Router
	model  string
}

func (m routerAgentModel) Complete(ctx context.Context, transcript llm.Context) (llm.AssistantMessage, error) {
	return m.router.CompleteNative(ctx, ai.NativeCompletionRequest{Context: transcript, Model: m.model, Task: ai.TaskTeaching, MaxTokens: 1024})
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
		Description: "Create one private, read-only focused message page for a goal or report. Use at most once in a turn. The server owns its recipient, layout, action, lifetime, capability, URL, and Telegram delivery.",
		Parameters:  json.RawMessage(`{"type":"object","additionalProperties":false,"properties":{"message":{"type":"string","minLength":1,"maxLength":4000}},"required":["message"]}`),
	}
}

func (t *createFocusedPageTool) Execute(ctx context.Context, call llm.ToolCall) llm.ToolResultMessage {
	message, ok := call.Arguments["message"].(string)
	if !ok || len(call.Arguments) != 1 {
		return focusedPageToolResult("Invalid arguments: message must be the only field.", true)
	}
	parsed, err := focusedpage.ParseMessage(message)
	if err != nil {
		return focusedPageToolResult("Invalid focused page message.", true)
	}
	if t.artifact != nil {
		if parsed == t.firstMessage {
			return focusedPageToolResult("Focused page already created for this turn.", false)
		}
		return focusedPageToolResult("Only one focused page may be created per turn.", true)
	}
	t.input.Message = parsed
	artifact, err := t.service.Create(ctx, t.input)
	if err != nil {
		return focusedPageToolResult("Focused page creation failed.", true)
	}
	t.firstMessage = parsed
	t.artifact = &artifact
	return focusedPageToolResult("Focused page created.", false)
}

func focusedPageToolResult(message string, isError bool) llm.ToolResultMessage {
	return llm.ToolResultMessage{Content: []llm.UserContent{llm.TextContent{Text: message}}, IsError: isError, Timestamp: time.Now()}
}

func nativeTeachingContext(messages []ai.Message) (llm.Context, error) {
	var system []string
	context := llm.Context{}
	for _, message := range messages {
		switch message.Role {
		case "system":
			system = append(system, message.Content)
		case "user":
			content := []llm.UserContent{llm.TextContent{Text: message.Content}}
			for _, imageURL := range message.ImageURLs {
				if imageURL != "" {
					content = append(content, llm.ImageURLContent{URL: imageURL})
				}
			}
			context.Messages = append(context.Messages, llm.UserMessage{Content: content})
		case "assistant":
			context.Messages = append(context.Messages, llm.AssistantMessage{Content: []llm.AssistantContent{llm.TextContent{Text: message.Content}}})
		default:
			return llm.Context{}, fmt.Errorf("unsupported teaching message role %q", message.Role)
		}
	}
	context.SystemPrompt = strings.Join(system, "\n\n")
	return context, nil
}

func transcriptUsage(messages []llm.Message) (int, int) {
	var input, output int
	for _, message := range messages {
		if assistant, ok := message.(llm.AssistantMessage); ok {
			input += assistant.Usage.Input + assistant.Usage.CacheRead + assistant.Usage.CacheWrite
			output += assistant.Usage.Output
		}
	}
	return input, output
}
