package agent

import (
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/p-n-ai/pai-bot/internal/progress"
)

func (e *Engine) quizSessionFromState(userID string, state ConversationQuizState) (*QuizSession, string, *Challenge, bool) {
	if state.ChallengeCode != "" {
		if e.challenges == nil {
			return nil, "", nil, false
		}
		challenge, err := e.challenges.GetChallenge(state.ChallengeCode)
		if err != nil || challenge == nil {
			return nil, "", nil, false
		}
		session := NewQuizSession(userID, challenge.TopicID, challenge.Questions)
		session.Intensity = "challenge"
		session.CurrentIndex = state.CurrentIndex
		session.CorrectAnswers = state.CorrectAnswers
		return session, challenge.TopicName, challenge, true
	}

	assessment, ok := e.curriculumLoader.GetAssessment(state.TopicID)
	if !ok {
		return nil, "", nil, false
	}
	questions := filterQuizQuestionsByIntensity(questionsFromAssessment(assessment), state.Intensity)
	session := NewQuizSession(userID, state.TopicID, questions)
	session.Intensity = state.Intensity
	session.CurrentIndex = state.CurrentIndex
	session.CorrectAnswers = state.CorrectAnswers
	return session, e.lookupTopicName(state.TopicID), nil, true
}

func submitChallengeAnswer(session *QuizSession, answer string) QuizAnswerResult {
	question, ok := session.NextQuestion()
	if !ok {
		return QuizAnswerResult{}
	}

	correct := gradeQuizAnswer(question, answer)
	result := QuizAnswerResult{
		Correct:          correct,
		ExpectedAnswer:   question.Answer,
		Explanation:      question.Working,
		QuestionComplete: true,
	}
	if correct {
		result.Feedback = "Correct."
		session.CorrectAnswers++
	} else {
		result.Feedback = matchingDistractorFeedback(question, answer)
		if result.Feedback == "" {
			result.Feedback = "Not quite."
		}
	}
	session.CurrentIndex++
	return result
}

func renderChallengeQuestion(code, topicName string, session *QuizSession, question QuizQuestion) string {
	body := renderQuizQuestion(topicName, session, question)
	return fmt.Sprintf("Challenge %s\n%s", code, body)
}

func renderChallengeAdvance(code, topicName string, session *QuizSession, question QuizQuestion, result QuizAnswerResult) string {
	return result.Feedback + "\n\n" + renderChallengeQuestion(code, topicName, session, question)
}

func renderChallengePaused() string {
	return "Okay, I paused your challenge run. Send `/challenge start` when you want to continue."
}

func renderChallengeResumed(code, topicName string, session *QuizSession, question QuizQuestion) string {
	return "Resuming your challenge.\n\n" + renderChallengeQuestion(code, topicName, session, question)
}

func renderChallengeResumeWithHint(code, topicName string, session *QuizSession, question QuizQuestion) string {
	if len(question.Hints) == 0 {
		return renderChallengeResumed(code, topicName, session, question)
	}
	return "Resuming your challenge.\nHint: " + question.Hints[0].Text + "\n\n" + renderChallengeQuestion(code, topicName, session, question)
}

func (e *Engine) completeChallengeQuiz(userID string, conv *Conversation, challenge *Challenge, result QuizAnswerResult, summary QuizSummary) string {
	if err := e.store.ClearConversationQuizState(conv.ID, conversationStateTeaching); err != nil {
		slog.Error("failed to clear challenge quiz state", "conversation_id", conv.ID, "error", err)
	}

	completion, err := e.challenges.CompleteChallenge(challenge.Code, userID, summary.CorrectAnswers)
	if err != nil {
		slog.Error("failed to complete challenge", "challenge_code", challenge.Code, "user_id", userID, "error", err)
		return "I couldn't finalize that challenge cleanly."
	}
	if completion.AwardFinishXP && e.xp != nil {
		if err := e.xp.Award(userID, progress.XPSourceChallenge, challengeFinishXP, map[string]any{
			"challenge_code": challenge.Code,
			"topic_id":       challenge.TopicID,
			"kind":           "finish",
		}); err != nil {
			slog.Warn("failed to award challenge finish xp", "challenge_code", challenge.Code, "user_id", userID, "error", err)
		}
	}
	if completion.AwardWinnerXP && e.xp != nil && completion.WinnerUserID != "" {
		if err := e.xp.Award(completion.WinnerUserID, progress.XPSourceChallenge, progress.XPChallengeWin, map[string]any{
			"challenge_code": challenge.Code,
			"topic_id":       challenge.TopicID,
			"kind":           "win",
		}); err != nil {
			slog.Warn("failed to award challenge winner xp", "challenge_code", challenge.Code, "winner_user_id", completion.WinnerUserID, "error", err)
		}
	}

	e.logEventAsync(Event{
		ConversationID: conv.ID,
		UserID:         userID,
		EventType:      "challenge_completed",
		Data: map[string]any{
			"challenge_code":  challenge.Code,
			"topic_id":        challenge.TopicID,
			"correct_answers": summary.CorrectAnswers,
			"total_questions": summary.TotalQuestions,
		},
	})
	return renderChallengeCompletion(result, completion.Challenge, userID)
}

func renderChallengeCompletion(result QuizAnswerResult, challenge *Challenge, userID string) string {
	var builder strings.Builder
	builder.WriteString(result.Feedback)
	builder.WriteString("\n\n")
	builder.WriteString(formatChallengeOverview(challenge, userID, nil, time.Now()))
	return builder.String()
}

func formatChallengeOverview(challenge *Challenge, userID string, conv *Conversation, now time.Time) string {
	if challenge == nil {
		return challengeEmptyStateMessage()
	}

	if conv != nil && conv.QuizState != nil && conv.QuizState.ChallengeCode == challenge.Code {
		switch conv.QuizState.RunState {
		case quizRunStatePaused:
			return fmt.Sprintf(
				"Challenge paused.\nCode: %s\nTopic: %s\n\nSend `/challenge start` to resume.",
				challenge.Code,
				challenge.TopicName,
			)
		default:
			return fmt.Sprintf(
				"Challenge in progress.\nCode: %s\nTopic: %s\nQuestion: %d/%d\n\nReply with your next answer, or send `/challenge start` to show the current question again.",
				challenge.Code,
				challenge.TopicName,
				conv.QuizState.CurrentIndex+1,
				maxChallengeQuestionCount(challenge),
			)
		}
	}

	switch {
	case challenge.Source == challengeSourcePublicQueue && challenge.OpponentID == "" && challenge.State == challengeStateWaiting:
		remaining := challengeAIFallbackWindow - now.Sub(challenge.CreatedAt)
		if remaining < 0 {
			remaining = 0
		}
		return fmt.Sprintf(
			"Looking for a challenge opponent.\nRequested topic: %s\nMatch style: same level first, same or nearby topic in the same subject.\nAI fallback: about %ds\n\nSend `/challenge cancel` to leave the queue.",
			fallbackChallengeTopicName(challenge),
			int(remaining.Round(time.Second).Seconds()),
		)
	case challenge.State == challengeStateCompleted:
		return formatCompletedChallenge(challenge, userID)
	case challengeUserCompleted(challenge, userID):
		userScore, _ := challengeScoresForUser(challenge, userID)
		return fmt.Sprintf(
			"Challenge submitted.\nCode: %s\nTopic: %s\nYour score: %d/%d\n\nWaiting for your opponent to finish.",
			challenge.Code,
			challenge.TopicName,
			userScore,
			maxChallengeQuestionCount(challenge),
		)
	case challenge.OpponentType == challengeOpponentAI:
		return fmt.Sprintf(
			"Adaptive AI rival ready.\nCode: %s\nTopic: %s\nQuestions: %d\n\nSend `/challenge start` to play.",
			challenge.Code,
			challenge.TopicName,
			maxChallengeQuestionCount(challenge),
		)
	case challengeReadyToOpen(challenge):
		return fmt.Sprintf(
			"Challenge ready.\nCode: %s\nTopic: %s\nQuestions: %d\nYou: ready\nOpponent: ready\n\nSend `/challenge start` to play.",
			challenge.Code,
			challenge.TopicName,
			maxChallengeQuestionCount(challenge),
		)
	default:
		return fmt.Sprintf(
			"Challenge matched.\nCode: %s\nTopic: %s\nQuestions: %d\nYou: %s\nOpponent: %s\n\nBoth students need to send `/challenge start`.",
			challenge.Code,
			challenge.TopicName,
			maxChallengeQuestionCount(challenge),
			challengeReadinessLabel(challenge, userID),
			challengeOpponentReadinessLabel(challenge, userID),
		)
	}
}

func formatCompletedChallenge(challenge *Challenge, userID string) string {
	userScore, opponentScore := challengeScoresForUser(challenge, userID)
	resultLine := "Result: draw."
	if userScore > opponentScore {
		resultLine = "Result: you win."
	} else if userScore < opponentScore {
		if challenge.OpponentType == challengeOpponentAI {
			resultLine = "Result: AI rival wins."
		} else {
			resultLine = "Result: your opponent wins."
		}
	}
	return fmt.Sprintf(
		"Challenge result.\nCode: %s\nTopic: %s\nYour score: %d/%d\nOpponent score: %d/%d\n%s",
		challenge.Code,
		challenge.TopicName,
		userScore,
		maxChallengeQuestionCount(challenge),
		opponentScore,
		maxChallengeQuestionCount(challenge),
		resultLine,
	)
}

func challengeScoresForUser(challenge *Challenge, userID string) (int, int) {
	if challenge.CreatorID == userID {
		return challenge.CreatorCorrectCount, challenge.OpponentCorrectCount
	}
	return challenge.OpponentCorrectCount, challenge.CreatorCorrectCount
}

func challengeUserCompleted(challenge *Challenge, userID string) bool {
	if challenge.CreatorID == userID {
		return challenge.CreatorCompletedAt != nil
	}
	return challenge.OpponentCompletedAt != nil
}

func challengeReadinessLabel(challenge *Challenge, userID string) string {
	if challenge.CreatorID == userID {
		return boolLabel(challenge.CreatorReadyAt != nil)
	}
	return boolLabel(challenge.OpponentReadyAt != nil)
}

func challengeOpponentReadinessLabel(challenge *Challenge, userID string) string {
	if challenge.CreatorID == userID {
		if challenge.OpponentType == challengeOpponentAI {
			return "ready"
		}
		return boolLabel(challenge.OpponentReadyAt != nil)
	}
	return boolLabel(challenge.CreatorReadyAt != nil)
}

func boolLabel(v bool) string {
	if v {
		return "ready"
	}
	return "not ready"
}

func maxChallengeQuestionCount(challenge *Challenge) int {
	if challenge == nil || challenge.QuestionCount <= 0 {
		return len(challenge.Questions)
	}
	return challenge.QuestionCount
}

func fallbackChallengeTopicName(challenge *Challenge) string {
	if challenge == nil {
		return ""
	}
	if challenge.Metadata.RequestedTopicName != "" {
		return challenge.Metadata.RequestedTopicName
	}
	return challenge.TopicName
}

func challengeEmptyStateMessage() string {
	return "You don't have an active challenge.\n\nTry something like:\n- /challenge linear equations\n- /challenge private fractions\n- /challenge ABC123"
}
