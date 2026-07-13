// Copyright 2026 the P&AI authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

// Package agentcore runs provider-neutral model and tool continuation turns.
package agentcore

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/p-n-ai/pai-bot/internal/llm"
	"github.com/xeipuuv/gojsonschema"
)

const DefaultMaxModelCalls = 8

var ErrModelCallLimit = errors.New("agent core model call limit reached")

type Termination string

const (
	TerminationCompleted      Termination = "completed"
	TerminationCancelled      Termination = "cancelled"
	TerminationModelCallLimit Termination = "model_call_limit"
)

type Model interface {
	Complete(context.Context, llm.Context, *llm.StreamOptions) (llm.AssistantMessage, error)
}

type Tool interface {
	Definition() llm.Tool
	Execute(context.Context, llm.ToolCall) (llm.ToolResultMessage, error)
}

type Config struct {
	MaxModelCalls  int
	StreamOptions  *llm.StreamOptions
	RunID          string
	ConversationID string
}

type Result struct {
	Final       llm.AssistantMessage
	Messages    []llm.Message
	Termination Termination
	ModelCalls  int
}

func Run(ctx context.Context, model Model, initial llm.Context, tools []Tool, cfg Config) (Result, error) {
	if model == nil {
		return Result{}, errors.New("agent core model is required")
	}
	if initial.SystemPrompt == "" {
		return Result{}, errors.New("agent core system prompt is required")
	}
	for _, message := range initial.Messages {
		if _, ok := message.(llm.SystemMessage); ok {
			return Result{}, errors.New("system messages are not allowed; use Context.SystemPrompt")
		}
	}

	registry, definitions, err := buildToolRegistry(tools)
	if err != nil {
		return Result{}, err
	}
	transcript := append([]llm.Message(nil), initial.Messages...)
	modelContext := initial
	modelContext.Messages = transcript
	modelContext.Tools = definitions
	maxCalls := cfg.MaxModelCalls
	if maxCalls <= 0 {
		maxCalls = DefaultMaxModelCalls
	}
	result := Result{Messages: transcript}

	for result.ModelCalls < maxCalls {
		if err := ctx.Err(); err != nil {
			terminate(&result, cfg, TerminationCancelled)
			return result, err
		}

		started := time.Now()
		reply, err := model.Complete(ctx, modelContext, cfg.StreamOptions)
		result.ModelCalls++
		slog.Debug("agent core model call completed",
			"run_id", cfg.RunID,
			"conversation_id", cfg.ConversationID,
			"model_call", result.ModelCalls,
			"duration_ms", time.Since(started).Milliseconds(),
			"error", err != nil,
		)
		if err != nil {
			if ctxErr := ctx.Err(); ctxErr != nil {
				terminate(&result, cfg, TerminationCancelled)
				return result, ctxErr
			}
			return result, err
		}

		transcript = append(transcript, reply)
		result.Messages = append([]llm.Message(nil), transcript...)
		result.Final = reply
		calls := toolCalls(reply)
		if len(calls) == 0 {
			terminate(&result, cfg, TerminationCompleted)
			return result, nil
		}

		for _, call := range calls {
			toolStarted := time.Now()
			toolResult := executeTool(ctx, registry, call)
			transcript = append(transcript, toolResult)
			result.Messages = append([]llm.Message(nil), transcript...)
			slog.Debug("agent core tool call completed",
				"run_id", cfg.RunID,
				"conversation_id", cfg.ConversationID,
				"tool_name", call.Name,
				"tool_call_id", call.ID,
				"duration_ms", time.Since(toolStarted).Milliseconds(),
				"error", toolResult.IsError,
			)
			if err := ctx.Err(); err != nil {
				terminate(&result, cfg, TerminationCancelled)
				return result, err
			}
		}
		modelContext.Messages = transcript
	}

	terminate(&result, cfg, TerminationModelCallLimit)
	return result, ErrModelCallLimit
}

func terminate(result *Result, cfg Config, termination Termination) {
	result.Termination = termination
	slog.Debug("agent core run terminated",
		"run_id", cfg.RunID,
		"conversation_id", cfg.ConversationID,
		"termination", termination,
		"model_calls", result.ModelCalls,
	)
}

type registeredTool struct {
	tool   Tool
	schema *gojsonschema.Schema
}

func buildToolRegistry(tools []Tool) (map[string]registeredTool, []llm.Tool, error) {
	registry := make(map[string]registeredTool, len(tools))
	definitions := make([]llm.Tool, 0, len(tools))
	for _, tool := range tools {
		if tool == nil {
			return nil, nil, errors.New("agent core tool is nil")
		}
		definition := tool.Definition()
		if definition.Name == "" {
			return nil, nil, errors.New("agent core tool name is required")
		}
		if _, exists := registry[definition.Name]; exists {
			return nil, nil, fmt.Errorf("duplicate agent core tool %q", definition.Name)
		}
		if len(definition.Parameters) == 0 {
			definition.Parameters = json.RawMessage(`{"type":"object","additionalProperties":false}`)
		}
		schema, err := gojsonschema.NewSchema(gojsonschema.NewBytesLoader(definition.Parameters))
		if err != nil {
			return nil, nil, fmt.Errorf("compile schema for tool %q: %w", definition.Name, err)
		}
		registry[definition.Name] = registeredTool{tool: tool, schema: schema}
		definitions = append(definitions, definition)
	}
	return registry, definitions, nil
}

func executeTool(ctx context.Context, registry map[string]registeredTool, call llm.ToolCall) llm.ToolResultMessage {
	if err := ctx.Err(); err != nil {
		return toolError(call, "tool execution cancelled")
	}
	registered, ok := registry[call.Name]
	if !ok {
		return toolError(call, "unknown tool")
	}
	validation, err := registered.schema.Validate(gojsonschema.NewGoLoader(call.Arguments))
	if err != nil || !validation.Valid() {
		return toolError(call, "invalid tool arguments")
	}
	result, err := registered.tool.Execute(ctx, call)
	if err != nil {
		return toolError(call, "tool execution failed")
	}
	result.ToolCallID = call.ID
	result.ToolName = call.Name
	return result
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

func toolCalls(message llm.AssistantMessage) []llm.ToolCall {
	var calls []llm.ToolCall
	for _, content := range message.Content {
		if call, ok := content.(llm.ToolCall); ok {
			calls = append(calls, call)
		}
	}
	return calls
}
