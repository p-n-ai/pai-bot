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
	pending map[string][]curriculum.Topic // userID → unlocked topics
}

func newPendingUnlocks() *pendingUnlocks {
	return &pendingUnlocks{
		pending: make(map[string][]curriculum.Topic),
	}
}

func (p *pendingUnlocks) add(userID string, topics []curriculum.Topic) {
	if len(topics) == 0 {
		return
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	p.pending[userID] = append(p.pending[userID], topics...)
}

func (p *pendingUnlocks) drain(userID string) []curriculum.Topic {
	p.mu.Lock()
	defer p.mu.Unlock()
	topics := p.pending[userID]
	delete(p.pending, userID)
	return topics
}

// checkTopicUnlocks checks if mastering a topic unlocks any new topics for the user.
func (e *Engine) checkTopicUnlocks(userID, syllabusID string, topic *curriculum.Topic) {
	if e.prereqGraph == nil || e.tracker == nil || topic == nil {
		return
	}

	// Get all mastery scores for the user.
	allProgress, err := e.tracker.GetAllProgress(userID)
	if err != nil {
		slog.Warn("failed to get progress for unlock check", "user_id", userID, "error", err)
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
		"user_id", userID,
		"mastered_topic", topic.ID,
		"unlocked_count", len(unlocked),
	)

	e.unlocks.add(userID, unlocked)

	for _, t := range unlocked {
		e.logEventAsync(Event{
			UserID:    userID,
			EventType: "topic_unlocked",
			Data: map[string]any{
				"topic_id":          t.ID,
				"topic_name":        t.Name,
				"unlocked_by":       topic.ID,
				"syllabus_id":       syllabusID,
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
func (e *Engine) drainUnlockNotification(userID, locale string) string {
	if e.unlocks == nil {
		return ""
	}
	topics := e.unlocks.drain(userID)
	if len(topics) == 0 {
		return ""
	}
	return formatUnlockNotification(locale, topics)
}

// drainMilestoneNotification returns and clears any pending milestone celebration messages for the user.
func (e *Engine) drainMilestoneNotification(userID string) string {
	if e.milestones == nil {
		return ""
	}
	msgs := e.milestones.drain(userID)
	return formatMilestoneBlock(msgs)
}

// resolveUserLocale returns the preferred locale for the given user, falling back to DefaultLocale.
func (e *Engine) resolveUserLocale(userID string) string {
	if lang, ok := e.store.GetUserPreferredLanguage(userID); ok && lang != "" {
		return lang
	}
	return i18n.DefaultLocale
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
