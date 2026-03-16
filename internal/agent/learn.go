package agent

import (
	"context"
	"log/slog"
	"strings"

	"github.com/p-n-ai/pai-bot/internal/chat"
	"github.com/p-n-ai/pai-bot/internal/i18n"
)

func (e *Engine) handleLearnCommand(_ context.Context, msg chat.InboundMessage, args []string) (string, error) {
	locale := e.messageLocale(msg, nil)

	if len(args) == 0 {
		return i18n.S(locale, i18n.MsgLearnUsage), nil
	}

	raw := strings.Join(args, " ")

	// Resolve topic from text.
	topic, _ := e.contextResolver.Resolve(raw)
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

	return i18n.S(locale, i18n.MsgLearnTopicSet, topic.Name), nil
}
