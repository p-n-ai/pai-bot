package agent

import (
	"context"
	"fmt"
	"strings"

	"github.com/p-n-ai/pai-bot/internal/chat"
	"github.com/p-n-ai/pai-bot/internal/curriculum"
)

func (e *Engine) handleChallengeCommand(_ context.Context, msg chat.InboundMessage, args []string) (string, error) {
	if e.challenges == nil {
		return "Challenge mode is not enabled yet.", nil
	}

	conv, err := e.getOrCreateConversation(msg.UserID)
	if err != nil {
		return "I hit a technical issue while opening challenge mode.", nil
	}
	if quizOwnsConversation(conv) {
		return quizMustFinishOrCancelMessage(), nil
	}

	if len(args) == 0 {
		return "Human matchmaking is the next slice.\n\nFor now use:\n- /challenge invite <topic>\n- /challenge <code>", nil
	}

	if strings.EqualFold(args[0], "invite") {
		return e.handleChallengeInvite(msg, conv, args[1:])
	}

	if len(args) == 1 && isChallengeCode(args[0]) {
		return e.handleChallengeJoin(msg, conv, args[0])
	}

	return "Use:\n- /challenge invite <topic>\n- /challenge <code>", nil
}

func (e *Engine) handleChallengeInvite(msg chat.InboundMessage, conv *Conversation, args []string) (string, error) {
	topic, ok := e.resolveChallengeTopic(conv, strings.Join(args, " "))
	if !ok {
		return unresolvedChallengeTopicMessage(), nil
	}

	challenge, err := e.challenges.CreateInviteChallenge(msg.UserID, ChallengeCreateInput{
		TopicID:       topic.ID,
		TopicName:     topic.Name,
		SyllabusID:    topic.SyllabusID,
		QuestionCount: e.challengeQuestionCount(topic.ID),
	})
	if err != nil {
		return "I hit a technical issue while creating that challenge.", nil
	}

	e.logEventAsync(Event{
		ConversationID: conv.ID,
		UserID:         msg.UserID,
		EventType:      "challenge_created",
		Data: map[string]any{
			"challenge_code": challenge.Code,
			"topic_id":       challenge.TopicID,
			"match_source":   challenge.MatchSource,
		},
	})
	return formatChallengeCreatedMessage(challenge), nil
}

func (e *Engine) handleChallengeJoin(msg chat.InboundMessage, conv *Conversation, code string) (string, error) {
	challenge, err := e.challenges.JoinChallenge(code, msg.UserID)
	if err != nil {
		switch err {
		case ErrChallengeSelfJoin:
			return "You can't join your own challenge code.", nil
		case ErrChallengeNotFound, ErrChallengeNotJoinable:
			return invalidChallengeCodeMessage(), nil
		default:
			return "I hit a technical issue while joining that challenge.", nil
		}
	}

	creatorName, ok := e.store.GetUserName(challenge.CreatorID)
	if !ok || strings.TrimSpace(creatorName) == "" {
		creatorName = "Another student"
	}

	e.logEventAsync(Event{
		ConversationID: conv.ID,
		UserID:         msg.UserID,
		EventType:      "challenge_joined",
		Data: map[string]any{
			"challenge_code": challenge.Code,
			"topic_id":       challenge.TopicID,
		},
	})
	return formatChallengeJoinedMessage(challenge, creatorName), nil
}

func (e *Engine) resolveChallengeTopic(conv *Conversation, raw string) (*curriculum.Topic, bool) {
	if e.curriculumLoader == nil {
		return nil, false
	}

	raw = strings.TrimSpace(raw)
	if raw != "" {
		if topic, _ := e.contextResolver.Resolve(raw); challengeTopicAvailable(e.curriculumLoader, topic) {
			return topic, true
		}
	}
	if conv != nil && conv.TopicID != "" {
		if topic, ok := e.curriculumLoader.GetTopic(conv.TopicID); ok {
			if _, ok := e.curriculumLoader.GetAssessment(topic.ID); ok {
				return &topic, true
			}
		}
	}
	if conv != nil {
		for i := len(conv.Messages) - 1; i >= 0; i-- {
			text := sanitizeControlContent(conv.Messages[i].Content)
			if text == "" {
				continue
			}
			topic, _ := e.contextResolver.Resolve(text)
			if challengeTopicAvailable(e.curriculumLoader, topic) {
				return topic, true
			}
		}
	}
	return nil, false
}

func (e *Engine) challengeQuestionCount(topicID string) int {
	if e.curriculumLoader == nil {
		return defaultChallengeQuestionCount
	}
	assessment, ok := e.curriculumLoader.GetAssessment(topicID)
	if !ok || len(assessment.Questions) == 0 {
		return defaultChallengeQuestionCount
	}
	if len(assessment.Questions) < defaultChallengeQuestionCount {
		return len(assessment.Questions)
	}
	return defaultChallengeQuestionCount
}

func challengeTopicAvailable(loader *curriculum.Loader, topic *curriculum.Topic) bool {
	if loader == nil || topic == nil {
		return false
	}
	_, ok := loader.GetAssessment(topic.ID)
	return ok
}

func isChallengeCode(value string) bool {
	value = normalizeChallengeCode(value)
	if len(value) != 6 {
		return false
	}
	for _, r := range value {
		if !strings.ContainsRune("ABCDEFGHJKLMNPQRSTUVWXYZ23456789", r) {
			return false
		}
	}
	return true
}

func invalidChallengeCodeMessage() string {
	return "This challenge code is invalid or unavailable."
}

func unresolvedChallengeTopicMessage() string {
	return "I couldn't map that to a challenge topic yet.\n\nTry something like:\n- /challenge invite linear equations\n- /challenge invite fractions"
}

func formatChallengeCreatedMessage(challenge *Challenge) string {
	return fmt.Sprintf(
		"Challenge created.\n\nCode: %s\nTopic: %s\nQuestions: %d\nShare: /challenge %s",
		challenge.Code,
		challenge.TopicName,
		challenge.QuestionCount,
		challenge.Code,
	)
}

func formatChallengeJoinedMessage(challenge *Challenge, creatorName string) string {
	return fmt.Sprintf(
		"Joined challenge %s.\n\nTopic: %s\nCreator: %s\nQuestions: %d\nState: ready",
		challenge.Code,
		challenge.TopicName,
		creatorName,
		challenge.QuestionCount,
	)
}
