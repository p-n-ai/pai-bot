// Copyright 2026 the P&AI authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package agent

import (
	"context"
	"fmt"
	"log/slog"
	"sort"
	"strings"
	"time"

	"github.com/p-n-ai/pai-bot/internal/chat"
	"github.com/p-n-ai/pai-bot/internal/curriculum"
	"github.com/p-n-ai/pai-bot/internal/progress"
	"github.com/p-n-ai/pai-bot/internal/retrieval"
)

const (
	maxTurnProgressItems = 5
	maxTurnDueReviews    = 3
	maxTurnGoals         = 3
)

// loadContextPackets gathers selected learner/runtime state for one tutor turn.
// It returns trust-labeled packets directly, so prompt rendering and tracing do
// not need a second context representation.
func (e *Engine) loadContextPackets(ctx context.Context, turn *agentTurn, msg chat.InboundMessage, conv *Conversation, topic *curriculum.Topic, teachingNotes string) []contextPacket {
	var packets []contextPacket

	profile := learnerProfile{}
	if identity, err := learnerIdentityForMessage(msg); err == nil {
		if name, ok := e.getUserName(identity); ok && name != "" {
			profile.Name = name
		}
		if form, ok := e.getUserForm(identity); ok && form != "" {
			profile.Form = form
		}
		if lang, ok := e.getUserPreferredLanguage(identity); ok && lang != "" {
			profile.Language = lang
		}
		if intensity, ok := e.getUserPreferredQuizIntensity(identity); ok && intensity != "" {
			profile.QuizIntensity = intensity
		}
		if group, ok := e.getUserABGroup(identity); ok && group != "" {
			profile.ABGroup = group
		}
	}
	packets = appendProfilePackets(packets, profile)

	if conv != nil {
		packets = append(packets, newContextPacket(contextPacket{
			ID:       "conversation.state",
			Kind:     contextKindConversation,
			Trust:    contextTrustSystemOwned,
			Source:   "conversation",
			Data:     conversationSystemContext(conv),
			RenderAs: contextRenderSystemData,
		}))
		if conv.Summary != "" {
			packets = append(packets, newContextPacket(contextPacket{
				ID:       "conversation.summary",
				Kind:     contextKindConversationSummary,
				Trust:    contextTrustModelGenerated,
				Source:   "conversation",
				Data:     conv.Summary,
				RenderAs: contextRenderQuotedData,
			}))
		}
	}

	if topic != nil {
		packets = append(packets, newContextPacket(contextPacket{
			ID:       "curriculum.topic",
			Kind:     contextKindCurriculum,
			Trust:    contextTrustSystemOwned,
			Source:   "topic",
			Data:     curriculumTopicContext(topic),
			RenderAs: contextRenderSystemData,
		}))
	}
	if teachingNotes != "" {
		packets = append(packets, newContextPacket(contextPacket{
			ID:       "curriculum.teaching_notes",
			Kind:     contextKindCurriculum,
			Trust:    contextTrustSystemOwned,
			Source:   "teaching_notes",
			Data:     teachingNotes,
			RenderAs: contextRenderSystemData,
		}))
	}

	identity, identityErr := learnerIdentityForMessage(msg)
	if e.tracker != nil && identityErr == nil {
		if items, err := e.getAllProgress(identity); err == nil {
			selected := selectTurnProgress(items, topic, maxTurnProgressItems)
			if len(selected) > 0 {
				packets = append(packets, newContextPacket(contextPacket{
					ID:       "progress.mastery",
					Kind:     contextKindProgress,
					Trust:    contextTrustSystemOwned,
					Source:   "progress",
					Data:     selected,
					RenderAs: contextRenderSystemData,
				}))
			}
		}
		if due, err := e.getDueReviews(identity); err == nil {
			selected := capProgressItems(sortDueReviews(due), maxTurnDueReviews)
			if len(selected) > 0 {
				packets = append(packets, newContextPacket(contextPacket{
					ID:       "progress.due_reviews",
					Kind:     contextKindProgress,
					Trust:    contextTrustSystemOwned,
					Source:   "due_reviews",
					Data:     selected,
					RenderAs: contextRenderSystemData,
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
			packets = append(packets, newContextPacket(contextPacket{
				ID:       "streak.current",
				Kind:     contextKindStreak,
				Trust:    contextTrustSystemOwned,
				Source:   "streak",
				Data:     &streak,
				RenderAs: contextRenderSystemData,
			}))
		}
	}

	if e.xp != nil {
		if total, err := e.xp.GetTotal(msg.UserID); err == nil && total > 0 {
			packets = append(packets, newContextPacket(contextPacket{
				ID:       "xp.total",
				Kind:     contextKindXP,
				Trust:    contextTrustSystemOwned,
				Source:   "xp",
				Data:     total,
				RenderAs: contextRenderSystemData,
			}))
		}
	}

	if turn.HasReply && turn.ReplyText != "" {
		packets = append(packets, newContextPacket(contextPacket{
			ID:       "current.reply_to",
			Kind:     contextKindCurrentInput,
			Trust:    contextTrustLearnerProvided,
			Source:   "reply_to",
			Data:     turn.ReplyText,
			RenderAs: contextRenderQuotedData,
		}))
	}

	if turn.ImageDataURL != "" {
		packets = appendImagePackets(packets, turn.ImageDataURL)
	}
	if e.evidenceRetriever != nil {
		learnerID, err := e.store.ResolveUserUUID(msg.UserID)
		if err != nil {
			slog.Warn("resolve learner for grounded retrieval", "error", err)
		} else {
			evidence, retrieveErr := e.evidenceRetriever.Retrieve(ctx, retrieval.TutorEvidenceRequest{
				TenantID: e.tenantID, LearnerID: learnerID,
				Query: groundedRetrievalQuery(turn, conv), Limit: 8,
			})
			if retrieveErr != nil {
				slog.Warn("retrieve grounded evidence", "error", retrieveErr)
			} else {
				for i, item := range evidence {
					packets = append(packets, evidenceContextPacket(i+1, item))
				}
			}
		}
	}

	return packets
}

type groundedEvidence struct {
	Label        string
	EvidenceID   string
	Origin       string
	SourceTitle  string
	Filename     string
	LocatorType  string
	LocatorStart int
	LocatorEnd   int
	Locator      string
	Excerpt      string
}

func evidenceContextPacket(index int, item retrieval.TutorEvidence) contextPacket {
	return newContextPacket(contextPacket{
		ID: "evidence." + item.Origin + "." + item.ID, Kind: contextKindEvidence,
		Trust: contextTrustExternal, Source: item.Origin, RenderAs: contextRenderQuotedData,
		Data: groundedEvidence{
			Label: fmt.Sprintf("S%d", index), EvidenceID: item.ID, Origin: item.Origin,
			SourceTitle: item.SourceTitle, Filename: item.Filename,
			LocatorType: item.LocatorType, LocatorStart: item.LocatorStart,
			LocatorEnd: item.LocatorEnd, Locator: item.Locator, Excerpt: item.Excerpt,
		},
	})
}

func groundedRetrievalQuery(turn *agentTurn, conv *Conversation) string {
	query := strings.TrimSpace(turn.InputText)
	if !isVagueContinuation(query) || conv == nil {
		return query
	}
	for i := len(conv.Messages) - 1; i >= 0; i-- {
		message := conv.Messages[i]
		if message.Role == "user" && message.ID != turn.UserMessageID && !isVagueContinuation(message.Content) {
			return strings.TrimSpace(message.Content + " " + query)
		}
	}
	if turn.Topic != nil {
		return strings.TrimSpace(turn.Topic.Name + " " + query)
	}
	return query
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
