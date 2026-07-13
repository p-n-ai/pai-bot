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

// agentTurn is the runtime boundary for one inbound message that reaches the
// generic tutor model path.
type agentTurn struct {
	ID             string
	UserID         string
	ConversationID string
	Channel        string
	Language       string
	Route          string
	TaskType       ai.TaskType

	InputText          string
	UserContent        string
	UserMessageID      string
	AssistantMessageID string
	HasImage           bool
	HasReply           bool
	ReplyText          string
	ImageDataURL       string
	Conversation       *Conversation
	Topic              *curriculum.Topic
	TeachingNotes      string
	Packets            []contextPacket
	Prompt             promptManifest
	Model              modelResult
}

// learnerProfile is the small educational profile that can be shown to the
// tutor model.
type learnerProfile struct {
	Name          string
	Form          string
	Language      string
	QuizIntensity string
	ABGroup       string
}

type contextKind string

const (
	contextKindProfile             contextKind = "profile"
	contextKindConversation        contextKind = "conversation"
	contextKindConversationSummary contextKind = "conversation_summary"
	contextKindCurriculum          contextKind = "curriculum"
	contextKindProgress            contextKind = "progress"
	contextKindGoal                contextKind = "goal"
	contextKindStreak              contextKind = "streak"
	contextKindXP                  contextKind = "xp"
	contextKindCurrentInput        contextKind = "current_input"
	contextKindImage               contextKind = "image"
	contextKindControlInstruction  contextKind = "control_instruction"
)

type contextTrust string

const (
	contextTrustSystemOwned     contextTrust = "system_owned"
	contextTrustModelGenerated  contextTrust = "model_generated"
	contextTrustLearnerProvided contextTrust = "learner_provided"
	contextTrustExternal        contextTrust = "external"
)

type contextRenderMode string

const (
	contextRenderSystemInstruction contextRenderMode = "system_instruction"
	contextRenderSystemData        contextRenderMode = "system_data"
	contextRenderQuotedData        contextRenderMode = "quoted_data"
	contextRenderAttachment        contextRenderMode = "attachment"
)

type contextTraceMode string

const (
	contextTraceMetadataOnly contextTraceMode = "metadata_only"
	contextTraceOmit         contextTraceMode = "omit"
)

type contextPacket struct {
	ID        string
	Kind      contextKind
	Trust     contextTrust
	Source    string
	Data      any
	RenderAs  contextRenderMode
	TraceMode contextTraceMode
}

// contextSource is trace metadata. It should not contain raw private data.
type contextSource struct {
	Name     string
	Included bool
}

// promptManifest records the shape of the model input without storing the full
// prompt body.
type promptManifest struct {
	MessageCount    int
	HasSystemPrompt bool
	HasSummary      bool
	HasImage        bool
	ContextSources  []contextSource
}

// modelResult records the model call result for tracing.
type modelResult struct {
	Model        string
	InputTokens  int
	OutputTokens int
	LatencyMS    int
	Error        string
}
