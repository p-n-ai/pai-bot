// Copyright 2026 the P&AI authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package agent

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/p-n-ai/pai-bot/internal/agentskills"
	"github.com/p-n-ai/pai-bot/internal/ai"
	"github.com/p-n-ai/pai-bot/internal/llm"
)

type skillsNativeProvider struct {
	response llm.AssistantMessage
}

func (p *skillsNativeProvider) CompleteNative(context.Context, string, llm.Context, *llm.StreamOptions) (llm.AssistantMessage, error) {
	return p.response, nil
}

func (*skillsNativeProvider) Complete(context.Context, ai.CompletionRequest) (ai.CompletionResponse, error) {
	return ai.CompletionResponse{}, errors.New("legacy completion should not run")
}

func (*skillsNativeProvider) StreamComplete(context.Context, ai.CompletionRequest) (<-chan ai.StreamChunk, error) {
	return nil, errors.New("legacy stream should not run")
}

func (*skillsNativeProvider) Models() []ai.ModelInfo            { return nil }
func (*skillsNativeProvider) HealthCheck(context.Context) error { return nil }

func TestEngineAdvertisesAndRegistersConfiguredSkills(t *testing.T) {
	root := t.TempDir()
	skillRoot := filepath.Join(root, "study-planner")
	if err := os.MkdirAll(skillRoot, 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	contents := "---\nname: study-planner\ndescription: Build a study plan when a learner asks for one.\n---\nPlan one week at a time."
	if err := os.WriteFile(filepath.Join(skillRoot, "SKILL.md"), []byte(contents), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}
	registry, err := agentskills.Load(root)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	engine := NewEngine(EngineConfig{Skills: registry})
	context := llm.Context{SystemPrompt: "Tutor policy"}

	engine.addSkillCatalog(&context)
	if !strings.Contains(context.SystemPrompt, "study-planner: Build a study plan") || strings.Contains(context.SystemPrompt, "Plan one week") {
		t.Fatalf("system prompt = %q", context.SystemPrompt)
	}
	tools := engine.teachingTools()
	if len(tools) != 2 || tools[0].Definition().Name != "load_skill" || tools[1].Definition().Name != "read_skill_resource" {
		t.Fatalf("teaching tools = %#v", tools)
	}
}

func TestOnDemandSkillsEnableNativeTeachingWithoutFeatureFlag(t *testing.T) {
	root := t.TempDir()
	skillRoot := filepath.Join(root, "study-planner")
	if err := os.MkdirAll(skillRoot, 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	contents := "---\nname: study-planner\ndescription: Build a study plan.\n---\nPlan one week."
	if err := os.WriteFile(filepath.Join(skillRoot, "SKILL.md"), []byte(contents), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}
	registry, err := agentskills.Load(root)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	router := ai.NewRouter()
	router.Register("native", &skillsNativeProvider{response: llm.AssistantMessage{
		Content: []llm.AssistantContent{llm.TextContent{Text: "native response"}},
	}})
	engine := NewEngine(EngineConfig{AIRouter: router, Skills: registry})
	turn := &agentTurn{UserContent: "Plan my week", Conversation: &Conversation{}}

	completion, _, err := engine.completeTeachingTurn(t.Context(), turn, engine.buildPromptMessagesFromTurn(turn), "")
	if err != nil {
		t.Fatalf("completeTeachingTurn() error = %v", err)
	}
	if completion.Content != "native response" {
		t.Fatalf("completion = %q, want native response", completion.Content)
	}
}

func TestEngineOmitsSkillsWhenNotConfigured(t *testing.T) {
	engine := NewEngine(EngineConfig{})
	context := llm.Context{SystemPrompt: "Tutor policy"}
	engine.addSkillCatalog(&context)
	if context.SystemPrompt != "Tutor policy" || len(engine.teachingTools()) != 0 {
		t.Fatalf("unconfigured skills changed runtime: context=%q tools=%d", context.SystemPrompt, len(engine.teachingTools()))
	}
}

func TestEngineAddsAlwaysActiveSkillToLegacyPrompt(t *testing.T) {
	root := t.TempDir()
	skillRoot := filepath.Join(root, "teacher")
	if err := os.MkdirAll(skillRoot, 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	contents := "---\nname: teacher\ndescription: Teach as a learning buddy.\nmetadata:\n  activation: always\n---\nTeach one useful move at a time."
	if err := os.WriteFile(filepath.Join(skillRoot, "SKILL.md"), []byte(contents), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}
	registry, err := agentskills.Load(root)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	engine := NewEngine(EngineConfig{Skills: registry})
	turn := &agentTurn{UserContent: "Help me", Conversation: &Conversation{}}

	messages := engine.buildPromptMessagesFromTurn(turn)
	if len(messages) == 0 || !strings.Contains(messages[0].Content, "ACTIVE SKILL: teacher") || !strings.Contains(messages[0].Content, "Teach one useful move") {
		t.Fatalf("system prompt = %#v", messages)
	}
}
