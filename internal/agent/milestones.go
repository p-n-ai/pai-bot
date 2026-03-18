package agent

import (
	"fmt"
	"strings"
	"sync"

	"github.com/p-n-ai/pai-bot/internal/i18n"
)

var xpMilestones = []int{100, 500, 1000, 2500, 5000, 10000}

func CheckXPMilestone(before, after int) (bool, int) {
	hit := 0
	for _, m := range xpMilestones {
		if before < m && after >= m {
			hit = m
		}
	}
	return hit > 0, hit
}

func FormatTopicMasteredCelebration(locale, topicName string, xpAwarded int) string {
	return fmt.Sprintf(i18n.S(locale, i18n.MsgMilestoneTopicMastered), topicName, xpAwarded)
}

func FormatXPMilestoneCelebration(locale string, xpTotal int) string {
	return fmt.Sprintf(i18n.S(locale, i18n.MsgMilestoneXP), xpTotal)
}

func FormatSubjectCompleteCelebration(locale, subjectName string) string {
	return fmt.Sprintf(i18n.S(locale, i18n.MsgMilestoneSubjectDone), subjectName)
}

func FormatStreakRecordCelebration(locale string, days int) string {
	return fmt.Sprintf(i18n.S(locale, i18n.MsgMilestoneStreakRecord), days)
}

type pendingMilestones struct {
	mu      sync.Mutex
	pending map[string][]string
}

func newPendingMilestones() *pendingMilestones {
	return &pendingMilestones{pending: make(map[string][]string)}
}

func (p *pendingMilestones) add(userID, msg string) {
	if msg == "" {
		return
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	p.pending[userID] = append(p.pending[userID], msg)
}

func (p *pendingMilestones) drain(userID string) []string {
	p.mu.Lock()
	defer p.mu.Unlock()
	msgs := p.pending[userID]
	delete(p.pending, userID)
	return msgs
}

func formatMilestoneBlock(msgs []string) string { //nolint:unused // will be called by engine in Task 2
	if len(msgs) == 0 {
		return ""
	}
	return strings.Join(msgs, "\n\n") + "\n\n"
}
