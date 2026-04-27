// Copyright 2026 the P&AI authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package agent

import (
	"github.com/p-n-ai/pai-bot/internal/ai"
	"github.com/p-n-ai/pai-bot/internal/curriculum"
)

const (
	agentTurnRouteTeaching = "teaching"
)

// AgentTurn is the runtime boundary for one inbound message that reaches the
// generic tutor model path.
type AgentTurn struct {
	ID             string
	UserID         string
	ConversationID string
	Channel        string
	Language       string
	Route          string
	TaskType       ai.TaskType

	InputText             string
	UserContent           string
	UserMessageID         string
	AssistantMessageID    string
	HasImage              bool
	HasReply              bool
	ReplyText             string
	ImageDataURL          string
	RatingPromptRequested bool

	Conversation  *Conversation
	Topic         *curriculum.Topic
	TeachingNotes string
	Packets       []ContextPacket
	Prompt        PromptManifest
	Model         ModelResult
}

// LearnerProfile is the small educational profile that can be shown to the
// tutor model.
type LearnerProfile struct {
	Name          string
	Form          string
	Language      string
	QuizIntensity string
	ABGroup       string
}

type ContextKind string

const (
	ContextKindProfile             ContextKind = "profile"
	ContextKindConversation        ContextKind = "conversation"
	ContextKindConversationSummary ContextKind = "conversation_summary"
	ContextKindCurriculum          ContextKind = "curriculum"
	ContextKindProgress            ContextKind = "progress"
	ContextKindGoal                ContextKind = "goal"
	ContextKindStreak              ContextKind = "streak"
	ContextKindXP                  ContextKind = "xp"
	ContextKindCurrentInput        ContextKind = "current_input"
	ContextKindImage               ContextKind = "image"
	ContextKindControlInstruction  ContextKind = "control_instruction"
)

type ContextTrust string

const (
	ContextTrustSystemOwned     ContextTrust = "system_owned"
	ContextTrustModelGenerated  ContextTrust = "model_generated"
	ContextTrustLearnerProvided ContextTrust = "learner_provided"
	ContextTrustExternal        ContextTrust = "external"
)

type ContextRenderMode string

const (
	ContextRenderSystemInstruction ContextRenderMode = "system_instruction"
	ContextRenderSystemData        ContextRenderMode = "system_data"
	ContextRenderQuotedData        ContextRenderMode = "quoted_data"
	ContextRenderAttachment        ContextRenderMode = "attachment"
)

type ContextTraceMode string

const (
	ContextTraceMetadataOnly ContextTraceMode = "metadata_only"
	ContextTraceOmit         ContextTraceMode = "omit"
)

type ContextPacket struct {
	ID        string
	Kind      ContextKind
	Trust     ContextTrust
	Source    string
	Data      any
	RenderAs  ContextRenderMode
	TraceMode ContextTraceMode
}

// ContextSource is trace metadata. It should not contain raw private data.
type ContextSource struct {
	Name     string
	Included bool
}

// PromptManifest records the shape of the model input without storing the full
// prompt body.
type PromptManifest struct {
	MessageCount    int
	HasSystemPrompt bool
	HasSummary      bool
	HasImage        bool
	ContextSources  []ContextSource
}

// ModelResult records the model call result for tracing.
type ModelResult struct {
	Model        string
	InputTokens  int
	OutputTokens int
	LatencyMS    int
	Error        string
}
