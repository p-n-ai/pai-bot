// Copyright 2026 the P&AI authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package agent

import (
	"context"
	"log/slog"
	"strings"
	"time"

	"github.com/p-n-ai/pai-bot/internal/ai"
	"github.com/p-n-ai/pai-bot/internal/chat"
	"github.com/p-n-ai/pai-bot/internal/i18n"
)

func (e *Engine) runTeachingTurn(ctx context.Context, msg chat.InboundMessage, conv *Conversation, responsePrefix string) (string, error) {
	userContent := msg.Text
	if msg.HasImage {
		if userContent == "" {
			userContent = "Please help me with the attached image."
		}
	}
	if msg.HasImage && msg.ImageDataURL == "" {
		return i18n.S(e.messageLocale(msg, conv), i18n.MsgImageProcessingFailed), nil
	}
	turn := &agentTurn{
		ID:             generateID(),
		UserID:         msg.UserID,
		ConversationID: conv.ID,
		Channel:        msg.Channel,
		Language:       msg.Language,
		Route:          agentTurnRouteTeaching,
		TaskType:       ai.TaskTeaching,
		InputText:      msg.Text,
		UserContent:    userContent,
		HasImage:       msg.HasImage,
		HasReply:       msg.ReplyToText != "",
		ReplyText:      msg.ReplyToText,
		ImageDataURL:   msg.ImageDataURL,
	}

	// Record user message.
	userMessageID, err := e.store.AddMessage(conv.ID, StoredMessage{
		Role:    "user",
		Content: userContent,
	})
	if err != nil {
		slog.Error("failed to store user message", "error", err)
	}
	turn.UserMessageID = userMessageID
	e.logEventAsync(Event{
		ConversationID: conv.ID,
		UserID:         msg.UserID,
		EventType:      "message_sent",
		Data: map[string]any{
			"channel":   msg.Channel,
			"text_len":  len(msg.Text),
			"has_reply": msg.ReplyToText != "",
			"has_image": msg.HasImage,
			"source":    "chat",
		},
	})

	// Refresh conversation to get latest messages.
	conv, _ = e.store.GetConversation(conv.ID)

	// Compact if needed (summarize older messages).
	e.maybeCompact(ctx, conv)

	matchedTopic, teachingNotes := e.resolveCurriculumContext(msg.UserID, conv.TopicID, msg.Text)

	// Guard: if the message is a vague continuation ("ok", "whats next", etc.)
	// and the conversation already has a stored topic, always prefer the stored
	// topic — even if the retriever matched a different topic (e.g. "next"
	// matching "Patterns and Sequences" via assessment items).
	vague := isVagueContinuation(msg.Text)
	if vague && conv.TopicID != "" && e.curriculumLoader != nil {
		if stored, ok := e.curriculumLoader.GetTopic(conv.TopicID); ok {
			topicCopy := stored
			matchedTopic = &topicCopy
			if notes, ok := e.curriculumLoader.GetTeachingNotes(conv.TopicID); ok {
				teachingNotes = notes
			}
		}
	} else if matchedTopic != nil && matchedTopic.ID != "" && matchedTopic.ID != conv.TopicID {
		// Non-vague message matched a different topic — update the conversation.
		if err := e.store.UpdateConversationTopicID(conv.ID, matchedTopic.ID); err != nil {
			slog.Warn("failed to persist matched topic", "conversation_id", conv.ID, "topic_id", matchedTopic.ID, "error", err)
		} else {
			conv.TopicID = matchedTopic.ID
		}
	}
	replyCount := countTutoringReplies(conv.Messages) + 1
	promptRequested := shouldRequestRatingAfterReply(replyCount, e.ratingPromptEvery)
	turn.RatingPromptRequested = promptRequested
	turn.Conversation = conv
	turn.Topic = matchedTopic
	turn.TeachingNotes = teachingNotes
	turn.Packets = e.loadContextPackets(ctx, turn, msg, conv, matchedTopic, teachingNotes)
	if e.turnHooksEnabled() {
		hookResult, err := e.runTurnHooks(ctx, turn)
		if err != nil {
			turn.Model.Error = err.Error()
			e.logAgentTurnCompleted(turn, "failed")
			slog.Error("turn hook failed", "error", err)
			return i18n.S(e.messageLocale(msg, conv), i18n.MsgTechnicalIssue), nil
		}
		turn.Packets = hookResult.Packets
		if hookResult.Blocked {
			e.logAgentTurnCompleted(turn, "blocked")
			if hookResult.BlockMessage != "" {
				return hookResult.BlockMessage, nil
			}
			return i18n.S(e.messageLocale(msg, conv), i18n.MsgTechnicalIssue), nil
		}
	} else if turn.RatingPromptRequested {
		turn.Packets = appendRatingPromptPacket(turn.Packets)
	}
	messages := e.buildPromptMessagesFromTurn(turn)

	reqModel := ""
	if msg.ImageDataURL != "" {
		// Prefer a vision-capable model for image understanding.
		reqModel = "gpt-4o"
	}

	// Call AI
	modelStartedAt := time.Now()
	resp, err := e.aiRouter.Complete(ctx, ai.CompletionRequest{
		Messages:  messages,
		Model:     reqModel,
		Task:      ai.TaskTeaching,
		MaxTokens: 1024,
	})
	turn.Model.LatencyMS = int(time.Since(modelStartedAt).Milliseconds())
	if err != nil {
		turn.Model.Error = err.Error()
		e.logAgentTurnCompleted(turn, "failed")
		slog.Error("AI completion failed", "error", err)
		return i18n.S(e.messageLocale(msg, conv), i18n.MsgTechnicalIssue), nil
	}
	turn.Model.Model = resp.Model
	turn.Model.InputTokens = resp.InputTokens
	turn.Model.OutputTokens = resp.OutputTokens

	// Telegram does not render LaTeX blocks; keep equations plain.
	plainContent := postProcessTutorResponse(normalizeLegacyExamReferences(normalizeEquationFormatting(resp.Content)), msg.Text)
	finalContent := plainContent
	if promptRequested && !strings.Contains(finalContent, ReviewActionCode) {
		finalContent = strings.TrimSpace(finalContent) + "\n\n" + ReviewActionCode
	}

	// Record assistant response with token metadata.
	assistantMessageID, err := e.store.AddMessage(conv.ID, StoredMessage{
		Role:         "assistant",
		Content:      finalContent,
		Model:        resp.Model,
		InputTokens:  resp.InputTokens,
		OutputTokens: resp.OutputTokens,
	})
	if err != nil {
		slog.Error("failed to store assistant message", "error", err)
	}
	turn.AssistantMessageID = assistantMessageID
	e.logEventAsync(Event{
		ConversationID: conv.ID,
		UserID:         msg.UserID,
		EventType:      "ai_response",
		Data: map[string]any{
			"channel":       msg.Channel,
			"model":         resp.Model,
			"input_tokens":  resp.InputTokens,
			"output_tokens": resp.OutputTokens,
			"text_len":      len(finalContent),
			"has_image":     msg.HasImage,
		},
	})
	e.logAgentTurnCompleted(turn, "completed")
	e.assessMasteryAsync(msg.UserID, matchedTopic, userContent, plainContent)
	e.recordActivityAsync(msg.UserID)

	if promptRequested {
		e.logEventAsync(Event{
			ConversationID: conv.ID,
			UserID:         msg.UserID,
			EventType:      "answer_rating_requested",
			Data: map[string]any{
				"channel":                msg.Channel,
				"after_tutoring_replies": replyCount,
				"rated_message_id":       assistantMessageID,
			},
		})
	}
	responseContent := finalContent
	if promptRequested && assistantMessageID != "" {
		responseContent = injectReviewTokenWithMessageID(finalContent, assistantMessageID)
	}

	if responsePrefix != "" {
		responseContent = responsePrefix + "\n\n" + responseContent
	}

	return responseContent, nil
}
