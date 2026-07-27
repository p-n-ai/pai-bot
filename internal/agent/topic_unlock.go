// Copyright 2026 the P&AI authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package agent

import (
	"fmt"
	"log/slog"
	"strings"
	"sync"

	"github.com/p-n-ai/pai-bot/internal/curriculum"
	"github.com/p-n-ai/pai-bot/internal/i18n"
)

// pendingUnlocks tracks topics that were unlocked but not yet notified to the user.
type pendingUnlocks struct {
	mu      sync.Mutex
	pending map[LearnerIdentity][]curriculum.Topic
}

func newPendingUnlocks() *pendingUnlocks {
	return &pendingUnlocks{
		pending: make(map[LearnerIdentity][]curriculum.Topic),
	}
}

func (p *pendingUnlocks) add(identity LearnerIdentity, topics []curriculum.Topic) {
	if len(topics) == 0 {
		return
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	p.pending[identity] = append(p.pending[identity], topics...)
}

func (p *pendingUnlocks) drain(identity LearnerIdentity) []curriculum.Topic {
	p.mu.Lock()
	defer p.mu.Unlock()
	topics := p.pending[identity]
	delete(p.pending, identity)
	return topics
}

// checkTopicUnlocks checks if mastering a topic unlocks any new topics for the user.
func (e *Engine) checkTopicUnlocks(identity LearnerIdentity, syllabusID string, topic *curriculum.Topic) {
	if e.prereqGraph == nil || e.tracker == nil || topic == nil {
		return
	}

	// Get all mastery scores for the user.
	allProgress, err := e.getAllProgress(identity)
	if err != nil {
		slog.Warn("failed to get progress for unlock check", "user_id", identity.ExternalID(), "error", err)
		return
	}

	scores := make(map[string]float64, len(allProgress))
	for _, p := range allProgress {
		scores[p.TopicID] = p.MasteryScore
	}

	unlocked := e.prereqGraph.UnlockableTopics(topic.ID, scores)
	if len(unlocked) == 0 {
		return
	}

	slog.Info("topics unlocked",
		"user_id", identity.ExternalID(),
		"mastered_topic", topic.ID,
		"unlocked_count", len(unlocked),
	)

	e.unlocks.add(identity, unlocked)

	for _, t := range unlocked {
		e.logEventAsync(Event{
			UserID:    identity.ExternalID(),
			EventType: "topic_unlocked",
			Data: map[string]any{
				"channel":     identity.Channel(),
				"topic_id":    t.ID,
				"topic_name":  t.Name,
				"unlocked_by": topic.ID,
				"syllabus_id": syllabusID,
			},
		})
	}
}

// formatUnlockNotification builds a notification message for newly unlocked topics.
func formatUnlockNotification(locale string, topics []curriculum.Topic) string {
	if len(topics) == 0 {
		return ""
	}

	var names []string
	for _, t := range topics {
		names = append(names, t.Name)
	}

	return i18n.S(locale, i18n.MsgTopicUnlocked, strings.Join(names, "\n- "))
}

// drainUnlockNotification returns and clears any pending unlock notification for the user.
func (e *Engine) drainUnlockNotification(identity LearnerIdentity, locale string) string {
	if e.unlocks == nil {
		return ""
	}
	topics := e.unlocks.drain(identity)
	if len(topics) == 0 {
		return ""
	}
	return formatUnlockNotification(locale, topics)
}

// drainMilestoneNotification returns and clears any pending milestone celebration messages for the user.
func (e *Engine) drainMilestoneNotification(identity LearnerIdentity) string {
	if e.milestones == nil {
		return ""
	}
	msgs := e.milestones.drain(identity)
	return formatMilestoneBlock(msgs)
}

// resolveUserLocale returns the preferred locale for the given user, falling back to DefaultLocale.
func (e *Engine) resolveUserLocale(identity LearnerIdentity) string {
	if lang, ok := e.getUserPreferredLanguage(identity); ok && lang != "" {
		return lang
	}
	return i18n.DefaultLocale
}

// userABGroup returns the A/B group for the given user, defaulting to ABGroupA.
func (e *Engine) userABGroup(identity LearnerIdentity) string {
	if group, ok := e.getUserABGroup(identity); ok && group != "" {
		return group
	}
	return ABGroupA
}

// buildPrereqGraph creates the prerequisite graph from loaded curriculum topics.
func buildPrereqGraph(loader *curriculum.Loader) *curriculum.PrereqGraph {
	if loader == nil {
		return nil
	}
	topics := loader.AllTopics()
	if len(topics) == 0 {
		return nil
	}
	graph := curriculum.NewPrereqGraph(topics)
	slog.Info("prerequisite graph built", "topics", len(topics))

	// Log topics with prerequisites for visibility.
	for _, t := range topics {
		if len(t.Prerequisites.Required) > 0 {
			slog.Debug("topic prerequisites",
				"topic_id", t.ID,
				"requires", fmt.Sprintf("%v", t.Prerequisites.Required),
			)
		}
	}
	return graph
}
