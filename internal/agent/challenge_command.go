package agent

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/p-n-ai/pai-bot/internal/chat"
	"github.com/p-n-ai/pai-bot/internal/curriculum"
	"github.com/p-n-ai/pai-bot/internal/i18n"
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
	if challengeOwnsConversation(conv) {
		return i18n.S(e.messageLocale(msg, conv), i18n.MsgChallengeFinishFirst), nil
	}

	if len(args) == 0 {
		return e.handleChallengeSearch(msg, conv)
	}

	if strings.EqualFold(args[0], "cancel") {
		return e.handleChallengeCancel(msg, conv)
	}

	if strings.EqualFold(args[0], "accept") {
		return e.handleChallengeAccept(msg, conv)
	}

	if strings.EqualFold(args[0], "invite") {
		return e.handleChallengeInvite(msg, conv, args[1:])
	}

	if len(args) == 1 && isChallengeCode(args[0]) {
		return e.handleChallengeJoin(msg, conv, args[0])
	}

	return "Use:\n- /challenge\n- /challenge accept\n- /challenge cancel\n- /challenge invite <topic>\n- /challenge <code>", nil
}

func (e *Engine) handleChallengeSearch(msg chat.InboundMessage, conv *Conversation) (string, error) {
	topic, ok := e.resolveChallengeTopic(conv, "")
	if !ok {
		return unresolvedChallengeSearchTopicMessage(), nil
	}

	result, err := e.challenges.StartChallengeSearch(msg.UserID, ChallengeCreateInput{
		TopicID:       topic.ID,
		TopicName:     topic.Name,
		SyllabusID:    topic.SyllabusID,
		QuestionCount: e.challengeQuestionCount(topic.ID),
	})
	if err != nil {
		if err == ErrChallengeSearchAlreadyActive && result != nil && result.Search != nil {
			return formatChallengeSearchAlreadyActiveMessage(result.Search), nil
		}
		if err == ErrChallengeAlreadyActive {
			return challengeAlreadyActiveMessage(), nil
		}
		return "I hit a technical issue while starting challenge search.", nil
	}
	if result == nil {
		return "I hit a technical issue while starting challenge search.", nil
	}
	if result.Challenge != nil {
		if result.Challenge.MatchSource == ChallengeMatchSourceAIFallback || result.Challenge.OpponentKind == ChallengeOpponentKindAI {
			return formatAIFallbackChallengeReadyMessage(result.Challenge), nil
		}
		otherUserID := result.Challenge.CreatorID
		if otherUserID == msg.UserID {
			otherUserID = result.Challenge.OpponentID
		}
		opponentName, ok := e.store.GetUserName(otherUserID)
		if !ok || strings.TrimSpace(opponentName) == "" {
			opponentName = "Another student"
		}
		e.logEventAsync(Event{
			ConversationID: conv.ID,
			UserID:         msg.UserID,
			EventType:      "challenge_matchmaking_paired",
			Data: map[string]any{
				"topic_id":     result.Challenge.TopicID,
				"match_source": result.Challenge.MatchSource,
				"state":        result.Challenge.State,
			},
		})
		if result.Challenge.State == ChallengeStatePendingAcceptance {
			return formatQueueChallengePendingAcceptanceMessage(result.Challenge, opponentName), nil
		}
		return formatQueueChallengeReadyMessage(result.Challenge, opponentName), nil
	}
	if result.Search == nil {
		return "I hit a technical issue while starting challenge search.", nil
	}

	e.logEventAsync(Event{
		ConversationID: conv.ID,
		UserID:         msg.UserID,
		EventType:      "challenge_matchmaking_queued",
		Data: map[string]any{
			"topic_id": result.Search.TopicID,
			"status":   result.Search.Status,
		},
	})
	return formatChallengeSearchingMessage(result.Search), nil
}

func (e *Engine) handleChallengeCancel(msg chat.InboundMessage, conv *Conversation) (string, error) {
	cancelled, err := e.challenges.CancelChallengeSearch(msg.UserID)
	if err != nil {
		return "I hit a technical issue while cancelling challenge search.", nil
	}
	if cancelled {
		e.logEventAsync(Event{
			ConversationID: conv.ID,
			UserID:         msg.UserID,
			EventType:      "challenge_matchmaking_cancelled",
		})
		return "Challenge search cancelled.", nil
	}

	declined, err := e.challenges.DeclinePendingChallenge(msg.UserID)
	if err != nil {
		return "I hit a technical issue while declining that challenge.", nil
	}
	if declined {
		e.logEventAsync(Event{
			ConversationID: conv.ID,
			UserID:         msg.UserID,
			EventType:      "challenge_matchmaking_declined",
		})
		return "Challenge declined.", nil
	}

	cancelled, err = e.challenges.CancelOpenChallenge(msg.UserID)
	if err != nil {
		return "I hit a technical issue while cancelling that challenge.", nil
	}
	if cancelled {
		e.logEventAsync(Event{
			ConversationID: conv.ID,
			UserID:         msg.UserID,
			EventType:      "challenge_cancelled",
		})
		return "Challenge cancelled.", nil
	}
	return "You do not have a cancellable challenge right now.", nil
}

func (e *Engine) handleChallengeAccept(msg chat.InboundMessage, conv *Conversation) (string, error) {
	challenge, err := e.challenges.AcceptPendingChallenge(msg.UserID)
	if err != nil {
		if err == ErrChallengeAcceptNotAvailable {
			return "You do not have a queue match waiting for acceptance right now.", nil
		}
		return "I hit a technical issue while accepting that challenge.", nil
	}

	otherUserID := challenge.CreatorID
	if otherUserID == msg.UserID {
		otherUserID = challenge.OpponentID
	}
	opponentName, ok := e.store.GetUserName(otherUserID)
	if !ok || strings.TrimSpace(opponentName) == "" {
		opponentName = "Another student"
	}

	e.logEventAsync(Event{
		ConversationID: conv.ID,
		UserID:         msg.UserID,
		EventType:      "challenge_matchmaking_accepted",
		Data: map[string]any{
			"topic_id": challenge.TopicID,
			"state":    challenge.State,
		},
	})
	if challenge.State == ChallengeStateReady {
		return formatQueueChallengeReadyMessage(challenge, opponentName), nil
	}
	return formatQueueChallengeAcceptedMessage(challenge, opponentName), nil
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
		if err == ErrChallengeAlreadyActive {
			return challengeAlreadyActiveMessage(), nil
		}
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
		case ErrChallengeAlreadyActive:
			return challengeAlreadyActiveMessage(), nil
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
		userID := ""
		topicID := ""
		if conv != nil {
			userID = conv.UserID
			topicID = conv.TopicID
		}
		if topic, _ := e.resolveCurriculumContext(userID, topicID, raw); challengeTopicAvailable(e.curriculumLoader, topic) {
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
			topic, _ := e.resolveCurriculumContext(conv.UserID, conv.TopicID, text)
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

func unresolvedChallengeSearchTopicMessage() string {
	return "I need a topic before I can find an opponent.\n\nTry talking about the topic first, or use:\n- /challenge invite linear equations\n- /challenge invite fractions"
}

func challengeAlreadyActiveMessage() string {
	return "You already have a live challenge or challenge search. Finish it first, or use /challenge cancel if it has not started yet."
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

func formatChallengeSearchingMessage(search *ChallengeSearch) string {
	return fmt.Sprintf(
		"Searching for an opponent.\n\nTopic: %s\nStatus: %s\nTimeout: %d minutes\n\nTip: Cancel and use /challenge invite to create a code for a friend.\n\nCancel: /challenge cancel",
		search.TopicName,
		search.Status,
		int(matchmakingWaitTimeout/time.Minute),
	)
}

func formatChallengeSearchAlreadyActiveMessage(search *ChallengeSearch) string {
	return fmt.Sprintf(
		"You already have a challenge search in progress.\n\nTopic: %s\nStatus: %s\nCancel: /challenge cancel",
		search.TopicName,
		search.Status,
	)
}

func formatQueueChallengeReadyMessage(challenge *Challenge, opponentName string) string {
	return fmt.Sprintf(
		"Opponent found.\n\nTopic: %s\nOpponent: %s\nQuestions: %d\nState: ready",
		challenge.TopicName,
		opponentName,
		challenge.QuestionCount,
	)
}

func formatQueueChallengePendingAcceptanceMessage(challenge *Challenge, opponentName string) string {
	return fmt.Sprintf(
		"Opponent found.\n\nTopic: %s\nOpponent: %s\nQuestions: %d\nState: pending_acceptance\nUse: /challenge accept\nDecline: /challenge cancel\nAccept window: %d minutes",
		challenge.TopicName,
		opponentName,
		challenge.QuestionCount,
		int(matchAcceptanceTimeout/time.Minute),
	)
}

func formatQueueChallengeAcceptedMessage(challenge *Challenge, opponentName string) string {
	return fmt.Sprintf(
		"Accepted.\n\nTopic: %s\nOpponent: %s\nQuestions: %d\nState: pending_acceptance\nWaiting for the other player.\nDecline: /challenge cancel",
		challenge.TopicName,
		opponentName,
		challenge.QuestionCount,
	)
}

func formatAIFallbackChallengeReadyMessage(challenge *Challenge) string {
	return fmt.Sprintf(
		"No human opponent found in time.\n\nTopic: %s\nOpponent: AI\nQuestions: %d\nState: ready",
		challenge.TopicName,
		challenge.QuestionCount,
	)
}
