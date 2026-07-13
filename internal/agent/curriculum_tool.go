// Copyright 2026 the P&AI authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package agent

import (
	"context"
	"encoding/json"
	"time"

	"github.com/p-n-ai/pai-bot/internal/agentcore"
	"github.com/p-n-ai/pai-bot/internal/curriculum"
	"github.com/p-n-ai/pai-bot/internal/llm"
)

const curriculumLookupToolName = "lookup_curriculum_topic"

type curriculumLookupTool struct {
	loader *curriculum.Loader
}

func (curriculumLookupTool) Definition() llm.Tool {
	return llm.Tool{
		Name:        curriculumLookupToolName,
		Description: "Look up the official curriculum topic and teaching notes for an exact topic ID when more curriculum detail is needed.",
		Parameters: json.RawMessage(`{
			"type":"object",
			"properties":{"topic_id":{"type":"string","minLength":1}},
			"required":["topic_id"],
			"additionalProperties":false
		}`),
	}
}

func (t curriculumLookupTool) Execute(_ context.Context, call llm.ToolCall) (llm.ToolResultMessage, error) {
	topicID, _ := call.Arguments["topic_id"].(string)
	topic, ok := t.loader.GetTopic(topicID)
	if !ok {
		return llm.ToolResultMessage{
			Content:   []llm.UserContent{llm.TextContent{Text: "curriculum topic not found"}},
			IsError:   true,
			Timestamp: time.Now(),
		}, nil
	}
	notes, _ := t.loader.GetTeachingNotes(topicID)
	payload, err := json.Marshal(struct {
		Topic         curriculum.Topic `json:"topic"`
		TeachingNotes string           `json:"teaching_notes,omitempty"`
	}{Topic: topic, TeachingNotes: notes})
	if err != nil {
		return llm.ToolResultMessage{
			Content:   []llm.UserContent{llm.TextContent{Text: "curriculum topic could not be encoded"}},
			IsError:   true,
			Timestamp: time.Now(),
		}, nil
	}
	return llm.ToolResultMessage{
		Content:   []llm.UserContent{llm.TextContent{Text: string(payload)}},
		Timestamp: time.Now(),
	}, nil
}

func (e *Engine) teachingTools() []agentcore.Tool {
	if e.curriculumLoader == nil {
		return nil
	}
	return []agentcore.Tool{curriculumLookupTool{loader: e.curriculumLoader}}
}
