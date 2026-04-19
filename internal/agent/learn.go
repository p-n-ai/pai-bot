// Copyright 2026 the P&AI authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"

	"github.com/p-n-ai/pai-bot/internal/ai"
	"github.com/p-n-ai/pai-bot/internal/chat"
	"github.com/p-n-ai/pai-bot/internal/curriculum"
	"github.com/p-n-ai/pai-bot/internal/i18n"
)

func (e *Engine) handleLearnCommand(ctx context.Context, msg chat.InboundMessage, args []string) (string, error) {
	locale := e.messageLocale(msg, nil)

	if len(args) == 0 {
		usage := i18n.S(locale, i18n.MsgLearnUsage)
		if e.curriculumLoader != nil {
			topics := e.curriculumLoader.AllTopics()
			if len(topics) > 0 {
				usage += "\n\nTopik tersedia:"
				for _, t := range topics {
					usage += "\n- " + t.Name + " (" + t.ID + ")"
				}
			}
		}
		return usage, nil
	}

	raw := strings.Join(args, " ")

	// Resolve topic from text via lexical retrieval.
	topic, _ := e.resolveCurriculumContext(msg.UserID, "", raw)

	// Fallback: if lexical retrieval missed (e.g. typos), ask AI to fuzzy-match.
	if topic == nil && e.curriculumLoader != nil && e.aiRouter != nil {
		if matched := e.aiMatchTopic(ctx, raw); matched != nil {
			topic = matched
		}
	}

	if topic == nil {
		return i18n.S(locale, i18n.MsgLearnTopicNotFound, raw), nil
	}

	// Get or create conversation and set topic.
	conv, err := e.getOrCreateConversation(msg.UserID)
	if err != nil {
		slog.Error("failed to get conversation for /learn", "user_id", msg.UserID, "error", err)
		return i18n.S(locale, i18n.MsgTechnicalIssue), nil
	}

	if err := e.store.UpdateConversationTopicID(conv.ID, topic.ID); err != nil {
		slog.Error("failed to set topic on conversation", "conversation_id", conv.ID, "topic_id", topic.ID, "error", err)
		return i18n.S(locale, i18n.MsgTechnicalIssue), nil
	}

	// Reset state to teaching if in a different mode.
	if conv.State != "teaching" {
		if err := e.store.UpdateConversationState(conv.ID, "teaching"); err != nil {
			slog.Error("failed to reset state to teaching", "conversation_id", conv.ID, "error", err)
		}
	}

	slog.Info("topic set via /learn",
		"user_id", msg.UserID,
		"topic_id", topic.ID,
		"topic_name", topic.Name,
	)

	e.logEventAsync(Event{
		ConversationID: conv.ID,
		UserID:         msg.UserID,
		EventType:      "learn_topic_set",
		Data: map[string]any{
			"topic_id":   topic.ID,
			"topic_name": topic.Name,
		},
	})

	// Store the /learn exchange in conversation history so subsequent AI
	// calls see that a topic was just set and a learning session started.
	response := i18n.S(locale, i18n.MsgLearnTopicSet, topic.Name)
	if _, err := e.store.AddMessage(conv.ID, StoredMessage{
		Role:    "user",
		Content: msg.Text,
	}); err != nil {
		slog.Error("failed to store /learn user message", "error", err)
	}
	if _, err := e.store.AddMessage(conv.ID, StoredMessage{
		Role:    "assistant",
		Content: response,
	}); err != nil {
		slog.Error("failed to store /learn response", "error", err)
	}

	return response, nil
}

// aiMatchTopic uses a cheap AI model to fuzzy-match user input (possibly with
// typos) against the list of available topic IDs and names.
func (e *Engine) aiMatchTopic(ctx context.Context, userInput string) *curriculum.Topic {
	topics := e.curriculumLoader.AllTopics()
	if len(topics) == 0 {
		return nil
	}

	// Build a compact topic list for the prompt.
	var topicList strings.Builder
	for _, t := range topics {
		fmt.Fprintf(&topicList, "- %s: %s\n", t.ID, t.Name)
	}

	type matchResult struct {
		TopicID string `json:"topic_id"`
	}

	prompt := fmt.Sprintf(`The user typed: %q

Available topics:
%s
Match the user's input to the most likely topic from the list above.
The user may have typos or use abbreviations. Pick the best match.
If nothing is remotely close, return an empty topic_id.

Return JSON: {"topic_id": "<ID>"}`, userInput, topicList.String())

	schema, _ := json.Marshal(map[string]any{
		"type": "object",
		"properties": map[string]any{
			"topic_id": map[string]any{"type": "string"},
		},
		"required":             []string{"topic_id"},
		"additionalProperties": false,
	})

	var result matchResult
	_, err := e.aiRouter.CompleteJSON(ctx, ai.CompletionRequest{
		Messages:  []ai.Message{{Role: "user", Content: prompt}},
		Task:      ai.TaskGrading, // cheap model
		MaxTokens: 64,
		StructuredOutput: &ai.StructuredOutputSpec{
			Name:       "topic_match",
			JSONSchema: schema,
			Strict:     true,
		},
	}, &result)
	if err != nil {
		slog.Warn("AI topic match failed", "error", err, "input", userInput)
		return nil
	}

	if result.TopicID == "" {
		return nil
	}

	matched, ok := e.curriculumLoader.GetTopic(result.TopicID)
	if !ok {
		slog.Warn("AI returned unknown topic ID", "topic_id", result.TopicID, "input", userInput)
		return nil
	}

	slog.Info("AI fuzzy-matched topic",
		"input", userInput,
		"matched_id", matched.ID,
		"matched_name", matched.Name,
	)
	return &matched
}
