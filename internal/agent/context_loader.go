// Copyright 2026 the P&AI authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package agent

import (
	"context"
	"sort"
	"time"

	"github.com/p-n-ai/pai-bot/internal/chat"
	"github.com/p-n-ai/pai-bot/internal/curriculum"
	"github.com/p-n-ai/pai-bot/internal/progress"
)

const (
	maxTurnProgressItems = 5
	maxTurnDueReviews    = 3
	maxTurnGoals         = 3
)

// LoadContextPackets gathers selected learner/runtime state for one tutor turn.
// It returns trust-labeled packets directly, so prompt rendering and tracing do
// not need a second context representation.
func (e *Engine) LoadContextPackets(_ context.Context, turn *AgentTurn, msg chat.InboundMessage, conv *Conversation, topic *curriculum.Topic, teachingNotes string) []ContextPacket {
	var packets []ContextPacket

	profile := LearnerProfile{}
	if name, ok := e.store.GetUserName(msg.UserID); ok && name != "" {
		profile.Name = name
	}
	if form, ok := e.store.GetUserForm(msg.UserID); ok && form != "" {
		profile.Form = form
	}
	if lang, ok := e.store.GetUserPreferredLanguage(msg.UserID); ok && lang != "" {
		profile.Language = lang
	}
	if intensity, ok := e.store.GetUserPreferredQuizIntensity(msg.UserID); ok && intensity != "" {
		profile.QuizIntensity = intensity
	}
	if group, ok := e.store.GetUserABGroup(msg.UserID); ok && group != "" {
		profile.ABGroup = group
	}
	packets = appendProfilePackets(packets, profile)

	if conv != nil {
		packets = append(packets, newContextPacket(ContextPacket{
			ID:       "conversation.state",
			Kind:     ContextKindConversation,
			Trust:    ContextTrustSystemOwned,
			Source:   "conversation",
			Data:     conversationSystemContext(conv),
			RenderAs: ContextRenderSystemData,
		}))
		if conv.Summary != "" {
			packets = append(packets, newContextPacket(ContextPacket{
				ID:       "conversation.summary",
				Kind:     ContextKindConversationSummary,
				Trust:    ContextTrustModelGenerated,
				Source:   "conversation",
				Data:     conv.Summary,
				RenderAs: ContextRenderQuotedData,
			}))
		}
	}

	if topic != nil {
		packets = append(packets, newContextPacket(ContextPacket{
			ID:       "curriculum.topic",
			Kind:     ContextKindCurriculum,
			Trust:    ContextTrustSystemOwned,
			Source:   "topic",
			Data:     curriculumTopicContext(topic),
			RenderAs: ContextRenderSystemData,
		}))
	}
	if teachingNotes != "" {
		packets = append(packets, newContextPacket(ContextPacket{
			ID:       "curriculum.teaching_notes",
			Kind:     ContextKindCurriculum,
			Trust:    ContextTrustSystemOwned,
			Source:   "teaching_notes",
			Data:     teachingNotes,
			RenderAs: ContextRenderSystemData,
		}))
	}

	if e.tracker != nil {
		if items, err := e.tracker.GetAllProgress(msg.UserID); err == nil {
			selected := selectTurnProgress(items, topic, maxTurnProgressItems)
			if len(selected) > 0 {
				packets = append(packets, newContextPacket(ContextPacket{
					ID:       "progress.mastery",
					Kind:     ContextKindProgress,
					Trust:    ContextTrustSystemOwned,
					Source:   "progress",
					Data:     selected,
					RenderAs: ContextRenderSystemData,
				}))
			}
		}
		if due, err := e.tracker.GetDueReviews(msg.UserID); err == nil {
			selected := capProgressItems(sortDueReviews(due), maxTurnDueReviews)
			if len(selected) > 0 {
				packets = append(packets, newContextPacket(ContextPacket{
					ID:       "progress.due_reviews",
					Kind:     ContextKindProgress,
					Trust:    ContextTrustSystemOwned,
					Source:   "due_reviews",
					Data:     selected,
					RenderAs: ContextRenderSystemData,
				}))
			}
		}
	}

	if e.goals != nil {
		if goals, err := e.goals.ListActiveGoals(msg.UserID); err == nil {
			if len(goals) > maxTurnGoals {
				goals = goals[:maxTurnGoals]
			}
			packets = appendGoalPackets(packets, goals)
		}
	}

	if e.streaks != nil {
		if streak, err := e.streaks.GetStreak(msg.UserID); err == nil && (streak.CurrentStreak > 0 || streak.LongestStreak > 0) {
			packets = append(packets, newContextPacket(ContextPacket{
				ID:       "streak.current",
				Kind:     ContextKindStreak,
				Trust:    ContextTrustSystemOwned,
				Source:   "streak",
				Data:     &streak,
				RenderAs: ContextRenderSystemData,
			}))
		}
	}

	if e.xp != nil {
		if total, err := e.xp.GetTotal(msg.UserID); err == nil && total > 0 {
			packets = append(packets, newContextPacket(ContextPacket{
				ID:       "xp.total",
				Kind:     ContextKindXP,
				Trust:    ContextTrustSystemOwned,
				Source:   "xp",
				Data:     total,
				RenderAs: ContextRenderSystemData,
			}))
		}
	}

	if turn.HasReply && turn.ReplyText != "" {
		packets = append(packets, newContextPacket(ContextPacket{
			ID:       "current.reply_to",
			Kind:     ContextKindCurrentInput,
			Trust:    ContextTrustLearnerProvided,
			Source:   "reply_to",
			Data:     turn.ReplyText,
			RenderAs: ContextRenderQuotedData,
		}))
	}

	if turn.ImageDataURL != "" {
		packets = appendImagePackets(packets, turn.ImageDataURL)
	}
	if turn.RatingPromptRequested {
		packets = append(packets, newContextPacket(ContextPacket{
			ID:       "rating.prompt",
			Kind:     ContextKindControlInstruction,
			Trust:    ContextTrustSystemOwned,
			Source:   "rating",
			Data:     ratingPromptInstruction,
			RenderAs: ContextRenderSystemInstruction,
		}))
	}

	return packets
}

func selectTurnProgress(items []progress.ProgressItem, topic *curriculum.Topic, limit int) []progress.ProgressItem {
	if len(items) == 0 || limit <= 0 {
		return nil
	}
	sort.SliceStable(items, func(i, j int) bool {
		if topic != nil {
			if items[i].TopicID == topic.ID && items[j].TopicID != topic.ID {
				return true
			}
			if items[j].TopicID == topic.ID && items[i].TopicID != topic.ID {
				return false
			}
		}
		return items[i].MasteryScore < items[j].MasteryScore
	})
	return capProgressItems(items, limit)
}

func sortDueReviews(items []progress.ProgressItem) []progress.ProgressItem {
	farFuture := time.Now().Add(100 * 365 * 24 * time.Hour)
	sort.SliceStable(items, func(i, j int) bool {
		di := items[i].NextReviewAt
		dj := items[j].NextReviewAt
		if di.IsZero() {
			di = farFuture
		}
		if dj.IsZero() {
			dj = farFuture
		}
		return di.Before(dj)
	})
	return items
}

func capProgressItems(items []progress.ProgressItem, limit int) []progress.ProgressItem {
	if len(items) <= limit {
		return append([]progress.ProgressItem(nil), items...)
	}
	return append([]progress.ProgressItem(nil), items[:limit]...)
}
