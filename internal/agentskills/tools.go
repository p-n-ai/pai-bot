// Copyright 2026 the P&AI authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package agentskills

import (
	"context"
	"encoding/json"
	"time"

	"github.com/p-n-ai/pai-bot/internal/agentcore"
	"github.com/p-n-ai/pai-bot/internal/llm"
)

const (
	loadSkillToolName    = "load_skill"
	readResourceToolName = "read_skill_resource"
)

// Tools returns the read-only tools used to progressively load skill content.
func (r *Registry) Tools() []agentcore.Tool {
	if r == nil || !r.hasOnDemandSkills() {
		return nil
	}
	return []agentcore.Tool{loadSkillTool{registry: r}, readResourceTool{registry: r}}
}

type loadSkillTool struct{ registry *Registry }

func (loadSkillTool) Definition() llm.Tool {
	return llm.Tool{
		Name:        loadSkillToolName,
		Description: "Load the full instructions for one available skill before applying it.",
		Parameters:  json.RawMessage(`{"type":"object","properties":{"name":{"type":"string","minLength":1}},"required":["name"],"additionalProperties":false}`),
	}
}

func (t loadSkillTool) Execute(_ context.Context, call llm.ToolCall) (llm.ToolResultMessage, error) {
	name, _ := call.Arguments["name"].(string)
	skill, ok := t.registry.skill(name)
	if !ok {
		return skillToolResult("skill not found", true), nil
	}
	return skillToolResult(skill.document, false), nil
}

type readResourceTool struct{ registry *Registry }

func (readResourceTool) Definition() llm.Tool {
	return llm.Tool{
		Name:        readResourceToolName,
		Description: "Read one file referenced by a loaded skill. Paths are relative to that skill directory.",
		Parameters:  json.RawMessage(`{"type":"object","properties":{"skill":{"type":"string","minLength":1},"path":{"type":"string","minLength":1}},"required":["skill","path"],"additionalProperties":false}`),
	}
}

func (t readResourceTool) Execute(_ context.Context, call llm.ToolCall) (llm.ToolResultMessage, error) {
	skill, _ := call.Arguments["skill"].(string)
	path, _ := call.Arguments["path"].(string)
	contents, err := t.registry.readResource(skill, path)
	if err != nil {
		return skillToolResult("skill resource unavailable", true), nil
	}
	return skillToolResult(contents, false), nil
}

func skillToolResult(content string, isError bool) llm.ToolResultMessage {
	return llm.ToolResultMessage{
		Content:   []llm.UserContent{llm.TextContent{Text: content}},
		IsError:   isError,
		Timestamp: time.Now(),
	}
}
