// Copyright 2026 the P&AI authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package agent

import (
	"context"
	"errors"
	"strings"

	"github.com/p-n-ai/pai-bot/internal/agentcore"
	"github.com/p-n-ai/pai-bot/internal/ai"
	"github.com/p-n-ai/pai-bot/internal/llm"
)

type teachingCompletion struct {
	Content      string
	Model        string
	InputTokens  int
	OutputTokens int
}

func (e *Engine) completeNativeTeachingTurn(ctx context.Context, turn *agentTurn, modelID string) (teachingCompletion, error) {
	return e.completeNativeTeachingTurnWithTools(ctx, turn, modelID, e.teachingTools())
}

func (e *Engine) completeNativeTeachingTurnWithTools(ctx context.Context, turn *agentTurn, modelID string, tools []agentcore.Tool) (teachingCompletion, error) {
	nativeContext, err := e.buildNativeContextFromTurn(turn)
	if err != nil {
		return teachingCompletion{}, err
	}
	model := ai.NewNativeModel(e.aiRouter, ai.NativeModelConfig{Task: ai.TaskTeaching, Model: modelID})
	result, err := agentcore.Run(ctx, model, nativeContext, tools, agentcore.Config{
		MaxModelCalls:  agentcore.DefaultMaxModelCalls,
		StreamOptions:  &llm.StreamOptions{MaxTokens: 1024},
		RunID:          turn.ID,
		ConversationID: turn.ConversationID,
	})
	if err != nil {
		return teachingCompletion{}, err
	}
	if result.Termination != agentcore.TerminationCompleted {
		return teachingCompletion{}, errors.New("native teaching turn did not complete")
	}

	var content strings.Builder
	completion := teachingCompletion{}
	for _, block := range result.Final.Content {
		if text, ok := block.(llm.TextContent); ok {
			content.WriteString(text.Text)
		}
	}
	for _, message := range result.Messages {
		assistant, ok := message.(llm.AssistantMessage)
		if !ok {
			continue
		}
		completion.InputTokens += assistant.Usage.Input + assistant.Usage.CacheRead + assistant.Usage.CacheWrite
		completion.OutputTokens += assistant.Usage.Output
	}
	completion.Content = content.String()
	completion.Model = result.Final.ResponseModel
	if completion.Model == "" {
		completion.Model = result.Final.Model
	}
	return completion, nil
}
