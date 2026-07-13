// Copyright 2026 the P&AI authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

// Package agentcore executes a provider-neutral model and tool continuation loop.
package agentcore

import (
	"context"
	"fmt"
	"time"

	"github.com/p-n-ai/pai-bot/internal/llm"
)

const DefaultMaxModelCalls = 3

type Model interface {
	Complete(context.Context, llm.Context) (llm.AssistantMessage, error)
}

type Tool interface {
	Definition() llm.Tool
	Execute(context.Context, llm.ToolCall) llm.ToolResultMessage
}

type Result struct {
	Final      llm.AssistantMessage
	Transcript []llm.Message
	ModelCalls int
	ToolCalls  int
}

type Core struct {
	model         Model
	tools         map[string]Tool
	definitions   []llm.Tool
	maxModelCalls int
}

func New(model Model, tools []Tool, maxModelCalls int) (*Core, error) {
	if model == nil {
		return nil, fmt.Errorf("agent core model is required")
	}
	if maxModelCalls <= 0 {
		maxModelCalls = DefaultMaxModelCalls
	}
	registered := make(map[string]Tool, len(tools))
	definitions := make([]llm.Tool, 0, len(tools))
	for _, tool := range tools {
		if tool == nil || tool.Definition().Name == "" {
			return nil, fmt.Errorf("agent core tool name is required")
		}
		name := tool.Definition().Name
		if _, exists := registered[name]; exists {
			return nil, fmt.Errorf("duplicate agent core tool %q", name)
		}
		registered[name] = tool
		definitions = append(definitions, tool.Definition())
	}
	return &Core{model: model, tools: registered, definitions: definitions, maxModelCalls: maxModelCalls}, nil
}

func (c *Core) Run(ctx context.Context, initial llm.Context) (Result, error) {
	for _, message := range initial.Messages {
		if _, ok := message.(llm.SystemMessage); ok {
			return Result{}, fmt.Errorf("agent core system instructions must use Context.SystemPrompt")
		}
	}
	transcript := append([]llm.Message(nil), initial.Messages...)

	result := Result{}
	for result.ModelCalls < c.maxModelCalls {
		if err := ctx.Err(); err != nil {
			return Result{}, err
		}
		reply, err := c.model.Complete(ctx, llm.Context{
			SystemPrompt: initial.SystemPrompt,
			Messages:     append([]llm.Message(nil), transcript...),
			Tools:        append([]llm.Tool(nil), c.definitions...),
		})
		if err != nil {
			return Result{}, err
		}
		result.ModelCalls++
		transcript = append(transcript, reply)
		calls := toolCalls(reply)
		if len(calls) == 0 {
			result.Final = reply
			result.Transcript = transcript
			return result, nil
		}
		for _, call := range calls {
			result.ToolCalls++
			tool, ok := c.tools[call.Name]
			if !ok {
				transcript = append(transcript, toolError(call, "unknown tool"))
				continue
			}
			toolResult := tool.Execute(ctx, call)
			toolResult.ToolCallID = call.ID
			toolResult.ToolName = call.Name
			if toolResult.Timestamp.IsZero() {
				toolResult.Timestamp = time.Now()
			}
			transcript = append(transcript, toolResult)
		}
	}
	return Result{}, fmt.Errorf("agent core exceeded %d model calls", c.maxModelCalls)
}

func toolCalls(message llm.AssistantMessage) []llm.ToolCall {
	var calls []llm.ToolCall
	for _, content := range message.Content {
		if call, ok := content.(llm.ToolCall); ok {
			calls = append(calls, call)
		}
	}
	return calls
}

func toolError(call llm.ToolCall, message string) llm.ToolResultMessage {
	return llm.ToolResultMessage{
		ToolCallID: call.ID,
		ToolName:   call.Name,
		Content:    []llm.UserContent{llm.TextContent{Text: message}},
		IsError:    true,
		Timestamp:  time.Now(),
	}
}

func FinalText(message llm.AssistantMessage) string {
	var text string
	for _, content := range message.Content {
		if part, ok := content.(llm.TextContent); ok {
			text += part.Text
		}
	}
	return text
}
