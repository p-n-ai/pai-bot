package agent

import (
	"log/slog"
	"regexp"
	"strings"

	"github.com/p-n-ai/pai-bot/internal/chat"
)

var challengeCodePattern = regexp.MustCompile(`^[A-Z0-9]{6}$`)

func (e *Engine) handleChallengeCommand(msg chat.InboundMessage, args []string) (string, error) {
	if e.challenges == nil || e.curriculumLoader == nil {
		return "Peer challenges are not enabled yet.", nil
	}

	conv, err := e.getOrCreateConversation(msg.UserID)
	if err != nil {
		slog.Error("failed to init user for /challenge", "user_id", msg.UserID, "error", err)
		return "I hit a technical issue while setting up your challenge.", nil
	}

	if len(args) == 0 {
		return e.describeChallenge(conv, msg.UserID)
	}

	raw := strings.TrimSpace(strings.Join(args, " "))
	lower := strings.ToLower(raw)
	switch lower {
	case "start":
		return e.startCurrentChallenge(msg, conv)
	case "status":
		return e.describeChallenge(conv, msg.UserID)
	case "cancel":
		return e.cancelChallengeQueue(msg.UserID)
	}

	if topicRequest, ok := parsePrivateChallengeRequest(raw); ok {
		return e.createPrivateChallenge(conv, msg.UserID, strings.TrimSpace(topicRequest))
	}

	if code, ok := parseChallengeCode(raw); ok {
		return e.joinPrivateChallenge(conv, msg.UserID, code)
	}
	return e.enterPublicChallenge(conv, msg.UserID, raw)
}

func (e *Engine) enterPublicChallenge(conv *Conversation, userID, raw string) (string, error) {
	if live, err := e.challenges.GetLiveChallengeForUser(userID); err != nil {
		return "", err
	} else if live != nil {
		return formatChallengeOverview(live, userID, conv, e.now()), nil
	}

	request, emptyResp, err := e.resolveChallengeRequest(userID, raw)
	if err != nil || emptyResp != "" {
		if err != nil {
			return "", err
		}
		return emptyResp, nil
	}

	match, err := e.tryHumanChallengeMatch(userID, request)
	if err != nil {
		if err == ErrChallengeAlreadyActive {
			if live, liveErr := e.challenges.GetLiveChallengeForUser(userID); liveErr == nil && live != nil {
				return formatChallengeOverview(live, userID, conv, e.now()), nil
			}
		}
		return "", err
	}
	if match != nil {
		e.logEventAsync(Event{
			ConversationID: conv.ID,
			UserID:         userID,
			EventType:      "challenge_matched_human",
			Data: map[string]any{
				"challenge_code": match.Code,
				"topic_id":       match.TopicID,
				"source":         match.Source,
			},
		})
		return formatChallengeOverview(match, userID, conv, e.now()), nil
	}

	challenge, err := e.challenges.CreatePublicQueue(userID, ChallengeInput{
		TopicID:    request.Topic.ID,
		TopicName:  request.Topic.Name,
		SubjectID:  request.Topic.SubjectID,
		SyllabusID: request.Topic.SyllabusID,
		Metadata: ChallengeMetadata{
			RequestedText:      raw,
			RequestedTopicID:   request.Topic.ID,
			RequestedTopicName: request.Topic.Name,
			CreatorForm:        request.Form,
		},
	})
	if err != nil {
		if err == ErrChallengeAlreadyActive {
			if live, liveErr := e.challenges.GetLiveChallengeForUser(userID); liveErr == nil && live != nil {
				return formatChallengeOverview(live, userID, conv, e.now()), nil
			}
		}
		return "", err
	}

	e.logEventAsync(Event{
		ConversationID: conv.ID,
		UserID:         userID,
		EventType:      "challenge_queued",
		Data: map[string]any{
			"challenge_code": challenge.Code,
			"requested_text": raw,
			"topic_id":       request.Topic.ID,
		},
	})
	return formatChallengeOverview(challenge, userID, conv, e.now()), nil
}

func (e *Engine) createPrivateChallenge(conv *Conversation, userID, raw string) (string, error) {
	if live, err := e.challenges.GetLiveChallengeForUser(userID); err != nil {
		return "", err
	} else if live != nil {
		return formatChallengeOverview(live, userID, conv, e.now()), nil
	}

	request, emptyResp, err := e.resolveChallengeRequest(userID, raw)
	if err != nil || emptyResp != "" {
		if err != nil {
			return "", err
		}
		return emptyResp, nil
	}

	topic, input, err := e.buildPrivateChallengeInput(userID, request)
	if err != nil {
		return "", err
	}
	challenge, err := e.challenges.CreatePrivateChallenge(userID, input)
	if err != nil {
		if err == ErrChallengeAlreadyActive {
			if live, liveErr := e.challenges.GetLiveChallengeForUser(userID); liveErr == nil && live != nil {
				return formatChallengeOverview(live, userID, conv, e.now()), nil
			}
		}
		return "", err
	}

	e.logEventAsync(Event{
		ConversationID: conv.ID,
		UserID:         userID,
		EventType:      "challenge_created_private",
		Data: map[string]any{
			"challenge_code": challenge.Code,
			"topic_id":       topic.ID,
		},
	})
	return formatChallengeOverview(challenge, userID, conv, e.now()), nil
}

func (e *Engine) joinPrivateChallenge(conv *Conversation, userID, code string) (string, error) {
	challenge, err := e.challenges.JoinPrivateChallenge(code, userID)
	if err != nil {
		switch err {
		case ErrChallengeAlreadyActive:
			if live, liveErr := e.challenges.GetLiveChallengeForUser(userID); liveErr == nil && live != nil {
				return formatChallengeOverview(live, userID, conv, e.now()), nil
			}
		case ErrChallengeNotFound:
			return "I couldn't find that private challenge code.\n\nTry `/challenge private linear equations` to create one or `/challenge ABC123` to join.", nil
		case ErrChallengeSelfJoin:
			return "That's your own private challenge code. Share it with another student, then both of you send `/challenge start`.", nil
		case ErrChallengeFull:
			return "That private challenge already has two students.", nil
		}
		return "", err
	}

	e.logEventAsync(Event{
		ConversationID: conv.ID,
		UserID:         userID,
		EventType:      "challenge_joined_private",
		Data: map[string]any{
			"challenge_code": challenge.Code,
			"topic_id":       challenge.TopicID,
		},
	})
	return formatChallengeOverview(challenge, userID, conv, e.now()), nil
}

func (e *Engine) startCurrentChallenge(msg chat.InboundMessage, conv *Conversation) (string, error) {
	challenge, err := e.currentChallengeForUser(msg.UserID)
	if err != nil {
		return "", err
	}
	if challenge == nil {
		return challengeEmptyStateMessage(), nil
	}
	challenge, err = e.maybePromoteChallengeForStatus(msg.UserID, challenge)
	if err != nil {
		return "", err
	}
	if challenge == nil {
		return challengeEmptyStateMessage(), nil
	}
	if challenge.State == challengeStateCompleted || challengeUserCompleted(challenge, msg.UserID) {
		return formatChallengeOverview(challenge, msg.UserID, conv, e.now()), nil
	}

	if challenge.OpponentType == challengeOpponentHuman {
		challenge, err = e.challenges.MarkReady(challenge.Code, msg.UserID)
		if err != nil {
			return "", err
		}
		if !challengeReadyToOpen(challenge) {
			return formatChallengeOverview(challenge, msg.UserID, conv, e.now()), nil
		}
	}

	if conv.QuizState != nil && conv.QuizState.ChallengeCode == challenge.Code {
		if conv.QuizState.RunState == quizRunStatePaused {
			return e.resumePausedQuizTurn(msg, conv, *conv.QuizState, quizTurnActionResume), nil
		}
		session, topicName, _, ok := e.quizSessionFromState(msg.UserID, *conv.QuizState)
		if !ok {
			return "I couldn't reload that challenge quiz state.", nil
		}
		question, hasQuestion := session.NextQuestion()
		if !hasQuestion {
			return formatChallengeOverview(challenge, msg.UserID, conv, e.now()), nil
		}
		return renderChallengeQuestion(challenge.Code, topicName, session, question), nil
	}

	state := ConversationQuizState{
		TopicID:        challenge.TopicID,
		Intensity:      "challenge",
		CurrentIndex:   0,
		CorrectAnswers: 0,
		ChallengeCode:  challenge.Code,
		RunState:       defaultQuizRunState(),
	}
	if err := e.store.UpdateConversationQuizState(conv.ID, conversationStateQuizActive, state); err != nil {
		return "", err
	}

	session := NewQuizSession(msg.UserID, challenge.TopicID, challenge.Questions)
	session.Intensity = "challenge"
	question, _ := session.NextQuestion()
	response := renderChallengeQuestion(challenge.Code, challenge.TopicName, session, question)
	if _, err := e.store.AddMessage(conv.ID, StoredMessage{Role: "assistant", Content: response}); err != nil {
		slog.Error("failed to store challenge prompt", "conversation_id", conv.ID, "error", err)
	}
	e.logEventAsync(Event{
		ConversationID: conv.ID,
		UserID:         msg.UserID,
		EventType:      "challenge_started",
		Data: map[string]any{
			"challenge_code": challenge.Code,
			"topic_id":       challenge.TopicID,
			"question_count": challenge.QuestionCount,
			"opponent_type":  challenge.OpponentType,
		},
	})
	return response, nil
}

func (e *Engine) cancelChallengeQueue(userID string) (string, error) {
	challenge, err := e.challenges.GetLiveChallengeForUser(userID)
	if err != nil {
		return "", err
	}
	if challenge == nil {
		return challengeEmptyStateMessage(), nil
	}
	if challenge.Source != challengeSourcePublicQueue || challenge.OpponentID != "" || challenge.State != challengeStateWaiting {
		return "You're already matched or mid-battle, so there's no public queue to cancel.\n\nSend `/challenge` to see the current challenge status.", nil
	}
	cancelled, err := e.challenges.CancelPublicQueue(userID)
	if err != nil {
		return "", err
	}
	if !cancelled {
		return challengeEmptyStateMessage(), nil
	}
	return "Okay, I removed you from the public challenge queue.", nil
}

func (e *Engine) describeChallenge(conv *Conversation, userID string) (string, error) {
	challenge, err := e.currentChallengeForUser(userID)
	if err != nil {
		return "", err
	}
	if challenge == nil {
		return challengeEmptyStateMessage(), nil
	}
	challenge, err = e.maybePromoteChallengeForStatus(userID, challenge)
	if err != nil {
		return "", err
	}
	if challenge == nil {
		return challengeEmptyStateMessage(), nil
	}
	return formatChallengeOverview(challenge, userID, conv, e.now()), nil
}

func (e *Engine) currentChallengeForUser(userID string) (*Challenge, error) {
	if e.challenges == nil {
		return nil, nil
	}
	challenge, err := e.challenges.GetLiveChallengeForUser(userID)
	if err != nil {
		return nil, err
	}
	if challenge != nil {
		return challenge, nil
	}
	return e.challenges.GetCurrentChallengeForUser(userID)
}

func (e *Engine) maybePromoteChallengeForStatus(userID string, challenge *Challenge) (*Challenge, error) {
	if challenge == nil {
		return nil, nil
	}
	if challenge.Source != challengeSourcePublicQueue || challenge.State != challengeStateWaiting || challenge.OpponentID != "" {
		return challenge, nil
	}
	if e.now().Sub(challenge.CreatedAt) < challengeAIFallbackWindow {
		return challenge, nil
	}
	return e.activateAIChallenge(userID, challenge)
}

func parseChallengeCode(raw string) (string, bool) {
	code := normalizeChallengeCodeValue(raw)
	return code, challengeCodePattern.MatchString(code)
}

func parsePrivateChallengeRequest(raw string) (string, bool) {
	fields := strings.Fields(strings.TrimSpace(raw))
	if len(fields) < 2 || !strings.EqualFold(fields[0], "private") {
		return "", false
	}
	return strings.Join(fields[1:], " "), true
}
