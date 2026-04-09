package agent

import (
	"context"
	"fmt"
	"log/slog"
	"math/rand"
	"strings"

	"github.com/p-n-ai/pai-bot/internal/chat"
	"github.com/p-n-ai/pai-bot/internal/i18n"
	"github.com/p-n-ai/pai-bot/internal/progress"
)

const (
	conversationStateChallengeActive = "challenge_active"
	conversationStateChallengeReview = "challenge_review"
	challengePhasePlaying            = "playing"
	challengePhaseReviewOffered      = "review_offered"
	challengePhaseReviewing          = "reviewing"
)

// challengeOwnsConversation returns true when the conversation is in a challenge state.
func challengeOwnsConversation(conv *Conversation) bool {
	if conv == nil {
		return false
	}
	return conv.State == conversationStateChallengeActive || conv.State == conversationStateChallengeReview
}

// maybeHandleChallengeTurn is the main router for challenge turns.
func (e *Engine) maybeHandleChallengeTurn(ctx context.Context, msg chat.InboundMessage, conv *Conversation) (string, bool) {
	if conv == nil {
		return "", false
	}

	// If already in a challenge state, route to the active phase.
	if conv.ChallengeState != nil && (conv.State == conversationStateChallengeActive || conv.State == conversationStateChallengeReview) {
		state := conv.ChallengeState
		switch state.Phase {
		case challengePhasePlaying:
			return e.handleChallengeAnswer(msg, conv, state), true
		case challengePhaseReviewOffered:
			return e.handleChallengeReviewOffer(msg, conv, state), true
		case challengePhaseReviewing:
			return e.handleChallengeReviewAnswer(msg, conv, state), true
		}
		return "", false
	}

	// Auto-start: if user has a ready challenge and sends a non-command message, start it.
	if e.challenges != nil && !strings.HasPrefix(msg.Text, "/") {
		challenge, err := e.challenges.GetActiveChallengeForUser(msg.UserID)
		if err == nil && challenge != nil && challenge.State == ChallengeStateReady {
			return e.startChallengePlay(ctx, msg, conv, challenge), true
		}
	}

	return "", false
}

// startChallengePlay loads questions and starts the challenge play phase.
func (e *Engine) startChallengePlay(_ context.Context, msg chat.InboundMessage, conv *Conversation, challenge *Challenge) string {
	if e.curriculumLoader == nil {
		return "Challenge mode requires a curriculum loader."
	}

	assessment, ok := e.curriculumLoader.GetAssessment(challenge.TopicID)
	if !ok || len(assessment.Questions) == 0 {
		return "No questions available for this challenge topic."
	}

	allQuestions := questionsFromAssessment(assessment)
	questions := selectChallengeQuestions(allQuestions, challenge.QuestionCount)

	// Transition challenge to active state
	if _, err := e.challenges.StartChallenge(challenge.ID); err != nil {
		slog.Error("failed to start challenge", "challenge_id", challenge.ID, "error", err)
		return "I hit a technical issue while starting the challenge."
	}

	challengeState := ConversationChallengeState{
		ChallengeID:  challenge.ID,
		TopicID:      challenge.TopicID,
		Phase:        challengePhasePlaying,
		Questions:    questions,
		CurrentIndex: 0,
		CorrectCount: 0,
		Answers:      []ChallengeAnswerRecord{},
	}

	if err := e.store.UpdateConversationChallengeState(conv.ID, conversationStateChallengeActive, challengeState); err != nil {
		slog.Error("failed to persist challenge state", "conversation_id", conv.ID, "error", err)
		return "I hit a technical issue while starting the challenge."
	}

	if _, err := e.store.AddMessage(conv.ID, StoredMessage{
		Role:    "user",
		Content: msg.Text,
	}); err != nil {
		slog.Error("failed to store challenge start message", "conversation_id", conv.ID, "error", err)
	}

	topicName := e.lookupTopicName(challenge.TopicID)
	response := renderChallengeQuestion(topicName, 0, len(questions), questions[0])

	if _, err := e.store.AddMessage(conv.ID, StoredMessage{
		Role:    "assistant",
		Content: response,
	}); err != nil {
		slog.Error("failed to store challenge question", "conversation_id", conv.ID, "error", err)
	}

	e.logEventAsync(Event{
		ConversationID: conv.ID,
		UserID:         msg.UserID,
		EventType:      "challenge_started",
		Data: map[string]any{
			"challenge_id":   challenge.ID,
			"topic_id":       challenge.TopicID,
			"question_count": len(questions),
		},
	})

	return response
}

// handleChallengeAnswer grades the current question and advances (one shot per question).
func (e *Engine) handleChallengeAnswer(msg chat.InboundMessage, conv *Conversation, state *ConversationChallengeState) string {
	if state.CurrentIndex >= len(state.Questions) {
		return e.completeChallengePlay(msg, conv, state)
	}

	answerText := strings.TrimSpace(msg.Text)
	if answerText == "" {
		topicName := e.lookupTopicName(e.challengeTopicIDFromState(state))
		return renderChallengeQuestion(topicName, state.CurrentIndex, len(state.Questions), state.Questions[state.CurrentIndex])
	}

	if _, err := e.store.AddMessage(conv.ID, StoredMessage{
		Role:    "user",
		Content: answerText,
	}); err != nil {
		slog.Error("failed to store challenge answer", "conversation_id", conv.ID, "error", err)
	}

	question := state.Questions[state.CurrentIndex]
	correct := gradeQuizAnswer(question, answerText)

	record := ChallengeAnswerRecord{
		QuestionIndex: state.CurrentIndex,
		UserAnswer:    answerText,
		Correct:       correct,
	}

	newState := *state
	newState.Answers = append(append([]ChallengeAnswerRecord{}, state.Answers...), record)
	newState.CurrentIndex = state.CurrentIndex + 1
	if correct {
		newState.CorrectCount = state.CorrectCount + 1
	}

	topicName := e.lookupTopicName(e.challengeTopicIDFromState(state))
	locale := e.messageLocale(msg, conv)
	var response string

	// Build feedback
	var feedback string
	if correct {
		feedback = i18n.S(locale, i18n.MsgChallengeCorrect)
	} else {
		feedback = i18n.S(locale, i18n.MsgChallengeIncorrect, question.Answer)
	}

	if newState.CurrentIndex >= len(newState.Questions) {
		// Challenge complete
		if err := e.store.UpdateConversationChallengeState(conv.ID, conversationStateChallengeActive, newState); err != nil {
			slog.Error("failed to update challenge state", "conversation_id", conv.ID, "error", err)
		}
		response = e.completeChallengePlay(msg, conv, &newState)
		response = feedback + "\n\n" + response
	} else {
		if err := e.store.UpdateConversationChallengeState(conv.ID, conversationStateChallengeActive, newState); err != nil {
			slog.Error("failed to update challenge state", "conversation_id", conv.ID, "error", err)
		}
		nextQ := renderChallengeQuestion(topicName, newState.CurrentIndex, len(newState.Questions), newState.Questions[newState.CurrentIndex])
		response = feedback + "\n\n" + nextQ
	}

	if _, err := e.store.AddMessage(conv.ID, StoredMessage{
		Role:    "assistant",
		Content: response,
	}); err != nil {
		slog.Error("failed to store challenge response", "conversation_id", conv.ID, "error", err)
	}

	e.logEventAsync(Event{
		ConversationID: conv.ID,
		UserID:         msg.UserID,
		EventType:      "challenge_answer",
		Data: map[string]any{
			"challenge_id":   state.ChallengeID,
			"question_index": state.CurrentIndex,
			"correct":        correct,
		},
	})

	return response
}

// completeChallengePlay settles the challenge and offers review if needed.
func (e *Engine) completeChallengePlay(msg chat.InboundMessage, conv *Conversation, state *ConversationChallengeState) string {
	// Complete the challenge in the store
	if _, err := e.challenges.CompleteChallenge(state.ChallengeID); err != nil {
		slog.Warn("failed to complete challenge in store", "challenge_id", state.ChallengeID, "error", err)
	}

	// Award XP for winning (score > 50% or AI fallback)
	scorePercent := 0
	if len(state.Questions) > 0 {
		scorePercent = (state.CorrectCount * 100) / len(state.Questions)
	}
	if scorePercent > 50 && e.xp != nil {
		if err := e.xp.Award(msg.UserID, progress.XPSourceChallenge, progress.XPChallengeWin, map[string]any{
			"challenge_id": state.ChallengeID,
			"score":        fmt.Sprintf("%d/%d", state.CorrectCount, len(state.Questions)),
		}); err != nil {
			slog.Error("failed to award challenge XP", "user_id", msg.UserID, "error", err)
		}
	}

	locale := e.messageLocale(msg, conv)
	resultText := renderChallengeResultLocalized(locale, state.CorrectCount, len(state.Questions))

	// Determine missed questions
	var missedIndices []int
	for _, a := range state.Answers {
		if !a.Correct {
			missedIndices = append(missedIndices, a.QuestionIndex)
		}
	}

	e.logEventAsync(Event{
		ConversationID: conv.ID,
		UserID:         msg.UserID,
		EventType:      "challenge_completed",
		Data: map[string]any{
			"challenge_id":   state.ChallengeID,
			"correct_count":  state.CorrectCount,
			"total_questions": len(state.Questions),
			"missed_count":   len(missedIndices),
		},
	})

	if len(missedIndices) == 0 {
		// Perfect score - no review needed
		if err := e.store.ClearConversationChallengeState(conv.ID, conversationStateTeaching); err != nil {
			slog.Error("failed to clear challenge state", "conversation_id", conv.ID, "error", err)
		}
		return resultText
	}

	// Offer review
	newState := *state
	newState.Phase = challengePhaseReviewOffered
	newState.MissedIndices = missedIndices

	if err := e.store.UpdateConversationChallengeState(conv.ID, conversationStateChallengeActive, newState); err != nil {
		slog.Error("failed to update challenge state for review offer", "conversation_id", conv.ID, "error", err)
	}

	return resultText + "\n\n" + renderChallengeReviewOfferLocalized(locale, len(missedIndices))
}

// handleChallengeReviewOffer processes the student's response to the review offer.
func (e *Engine) handleChallengeReviewOffer(msg chat.InboundMessage, conv *Conversation, state *ConversationChallengeState) string {
	// Persist the user's decision message.
	if _, err := e.store.AddMessage(conv.ID, StoredMessage{
		Role:    "user",
		Content: strings.TrimSpace(msg.Text),
	}); err != nil {
		slog.Error("failed to store review decision", "conversation_id", conv.ID, "error", err)
	}

	text := strings.ToLower(strings.TrimSpace(msg.Text))

	if isReviewAccept(text) {
		// Bounds check: MissedIndices must be non-empty and point to valid questions.
		if len(state.MissedIndices) == 0 || state.MissedIndices[0] >= len(state.Questions) {
			if err := e.store.ClearConversationChallengeState(conv.ID, conversationStateTeaching); err != nil {
				slog.Error("failed to clear invalid challenge review state", "conversation_id", conv.ID, "error", err)
			}
			return i18n.S(e.messageLocale(msg, conv), i18n.MsgChallengeReviewSkip)
		}

		// Start review
		newState := *state
		newState.Phase = challengePhaseReviewing
		newState.ReviewIndex = 0
		newState.ReviewCorrect = 0

		if err := e.store.UpdateConversationChallengeState(conv.ID, conversationStateChallengeReview, newState); err != nil {
			slog.Error("failed to update challenge state for review", "conversation_id", conv.ID, "error", err)
			return i18n.S(e.messageLocale(msg, conv), i18n.MsgTechnicalIssue)
		}

		topicName := e.lookupTopicName(e.challengeTopicIDFromState(state))
		questionIdx := state.MissedIndices[0]
		question := state.Questions[questionIdx]
		response := renderChallengeReviewQuestion(topicName, 0, len(state.MissedIndices), question)

		if _, err := e.store.AddMessage(conv.ID, StoredMessage{
			Role:    "assistant",
			Content: response,
		}); err != nil {
			slog.Error("failed to store review question", "conversation_id", conv.ID, "error", err)
		}

		e.logEventAsync(Event{
			ConversationID: conv.ID,
			UserID:         msg.UserID,
			EventType:      "challenge_review_started",
			Data: map[string]any{
				"challenge_id": state.ChallengeID,
				"missed_count": len(state.MissedIndices),
			},
		})

		return response
	}

	// Skip review - return to teaching
	if err := e.store.ClearConversationChallengeState(conv.ID, conversationStateTeaching); err != nil {
		slog.Error("failed to clear challenge state on review skip", "conversation_id", conv.ID, "error", err)
	}

	locale := e.messageLocale(msg, conv)
	response := i18n.S(locale, i18n.MsgChallengeReviewSkip)
	if _, err := e.store.AddMessage(conv.ID, StoredMessage{
		Role:    "assistant",
		Content: response,
	}); err != nil {
		slog.Error("failed to store review skip response", "conversation_id", conv.ID, "error", err)
	}

	e.logEventAsync(Event{
		ConversationID: conv.ID,
		UserID:         msg.UserID,
		EventType:      "challenge_review_skipped",
		Data: map[string]any{
			"challenge_id": state.ChallengeID,
		},
	})

	return response
}

// handleChallengeReviewAnswer grades a review answer (allows retries like quiz).
func (e *Engine) handleChallengeReviewAnswer(msg chat.InboundMessage, conv *Conversation, state *ConversationChallengeState) string {
	if state.ReviewIndex >= len(state.MissedIndices) {
		return e.completeChallengeReview(msg, conv, state)
	}

	// Bounds check: ensure MissedIndices point to valid questions.
	questionIdx := state.MissedIndices[state.ReviewIndex]
	if questionIdx >= len(state.Questions) {
		if err := e.store.ClearConversationChallengeState(conv.ID, conversationStateTeaching); err != nil {
			slog.Error("failed to clear invalid review state", "conversation_id", conv.ID, "error", err)
		}
		return i18n.S(e.messageLocale(msg, conv), i18n.MsgChallengeReviewSkip)
	}

	answerText := strings.TrimSpace(msg.Text)
	if answerText == "" {
		topicName := e.lookupTopicName(e.challengeTopicIDFromState(state))
		question := state.Questions[questionIdx]
		return renderChallengeReviewQuestion(topicName, state.ReviewIndex, len(state.MissedIndices), question)
	}

	// Check for exit/stop intents
	if isChallengeReviewExit(answerText) {
		if _, err := e.store.AddMessage(conv.ID, StoredMessage{
			Role:    "user",
			Content: answerText,
		}); err != nil {
			slog.Error("failed to store review exit message", "conversation_id", conv.ID, "error", err)
		}
		if err := e.store.ClearConversationChallengeState(conv.ID, conversationStateTeaching); err != nil {
			slog.Error("failed to clear challenge state on review exit", "conversation_id", conv.ID, "error", err)
		}

		e.logEventAsync(Event{
			ConversationID: conv.ID,
			UserID:         msg.UserID,
			EventType:      "challenge_review_exited",
			Data: map[string]any{
				"challenge_id":  state.ChallengeID,
				"review_index":  state.ReviewIndex,
				"review_total":  len(state.MissedIndices),
			},
		})

		return i18n.S(e.messageLocale(msg, conv), i18n.MsgChallengeReviewSkip)
	}

	if _, err := e.store.AddMessage(conv.ID, StoredMessage{
		Role:    "user",
		Content: answerText,
	}); err != nil {
		slog.Error("failed to store review answer", "conversation_id", conv.ID, "error", err)
	}

	question := state.Questions[questionIdx]
	correct := gradeQuizAnswer(question, answerText)

	topicName := e.lookupTopicName(e.challengeTopicIDFromState(state))
	var response string

	if !correct {
		// Allow retry (like quiz)
		locale := e.messageLocale(msg, conv)
		feedback := i18n.S(locale, i18n.MsgChallengeReviewRetry)
		if len(question.Hints) > 0 {
			feedback += "\nHint: " + question.Hints[0].Text
		}
		response = feedback

		if _, err := e.store.AddMessage(conv.ID, StoredMessage{
			Role:    "assistant",
			Content: response,
		}); err != nil {
			slog.Error("failed to store review retry", "conversation_id", conv.ID, "error", err)
		}
		return response
	}

	// Correct - advance review
	locale := e.messageLocale(msg, conv)
	correctFeedback := i18n.S(locale, i18n.MsgChallengeCorrect)

	newState := *state
	newState.ReviewIndex = state.ReviewIndex + 1
	newState.ReviewCorrect = state.ReviewCorrect + 1

	if newState.ReviewIndex >= len(newState.MissedIndices) {
		// Review complete
		response = correctFeedback + "\n\n" + e.completeChallengeReview(msg, conv, &newState)
	} else {
		if err := e.store.UpdateConversationChallengeState(conv.ID, conversationStateChallengeReview, newState); err != nil {
			slog.Error("failed to update review state", "conversation_id", conv.ID, "error", err)
		}
		nextQuestionIdx := newState.MissedIndices[newState.ReviewIndex]
		// Bounds check for next review question.
		if nextQuestionIdx >= len(newState.Questions) {
			response = correctFeedback + "\n\n" + e.completeChallengeReview(msg, conv, &newState)
		} else {
			nextQuestion := newState.Questions[nextQuestionIdx]
			nextQ := renderChallengeReviewQuestion(topicName, newState.ReviewIndex, len(newState.MissedIndices), nextQuestion)
			response = correctFeedback + "\n\n" + nextQ
		}
	}

	if _, err := e.store.AddMessage(conv.ID, StoredMessage{
		Role:    "assistant",
		Content: response,
	}); err != nil {
		slog.Error("failed to store review response", "conversation_id", conv.ID, "error", err)
	}

	return response
}

// completeChallengeReview awards XP and clears state.
func (e *Engine) completeChallengeReview(msg chat.InboundMessage, conv *Conversation, state *ConversationChallengeState) string {
	// Award review XP
	if e.xp != nil {
		if err := e.xp.Award(msg.UserID, progress.XPSourceReview, progress.XPReviewCompleted, map[string]any{
			"challenge_id":  state.ChallengeID,
			"review_correct": state.ReviewCorrect,
			"review_total":  len(state.MissedIndices),
		}); err != nil {
			slog.Error("failed to award review XP", "user_id", msg.UserID, "error", err)
		}
	}

	if err := e.store.ClearConversationChallengeState(conv.ID, conversationStateTeaching); err != nil {
		slog.Error("failed to clear challenge state after review", "conversation_id", conv.ID, "error", err)
	}

	e.logEventAsync(Event{
		ConversationID: conv.ID,
		UserID:         msg.UserID,
		EventType:      "challenge_review_completed",
		Data: map[string]any{
			"challenge_id":  state.ChallengeID,
			"review_correct": state.ReviewCorrect,
			"review_total":  len(state.MissedIndices),
		},
	})

	locale := e.messageLocale(msg, conv)
	return renderChallengeReviewCompleteLocalized(locale, state.ReviewCorrect, len(state.MissedIndices))
}

// challengeTopicIDFromState returns the topic ID from the challenge state.
func (e *Engine) challengeTopicIDFromState(state *ConversationChallengeState) string {
	if state == nil {
		return ""
	}
	return state.TopicID
}

// --- Rendering functions ---

func renderChallengeQuestion(topicName string, index, total int, question QuizQuestion) string {
	var builder strings.Builder
	if topicName != "" {
		builder.WriteString("Challenge: ")
		builder.WriteString(topicName)
		builder.WriteString("\n")
	}
	fmt.Fprintf(&builder, "Question %d/%d\n", index+1, total)
	builder.WriteString(question.Text)

	options := quizOptions(question)
	if len(options) > 0 {
		builder.WriteString("\nOptions:")
		for _, option := range options {
			builder.WriteString("\n- ")
			builder.WriteString(option)
		}
	}

	builder.WriteString("\nReply with your answer.")
	return builder.String()
}

func renderChallengeResultLocalized(locale string, correct, total int) string {
	scorePercent := 0
	if total > 0 {
		scorePercent = (correct * 100) / total
	}
	return i18n.S(locale, i18n.MsgChallengeComplete, correct, total, scorePercent)
}

func renderChallengeReviewOfferLocalized(locale string, missedCount int) string {
	return i18n.S(locale, i18n.MsgChallengeReviewOffer, missedCount)
}

func renderChallengeReviewQuestion(topicName string, reviewIndex, reviewTotal int, question QuizQuestion) string {
	var builder strings.Builder
	if topicName != "" {
		builder.WriteString("Review: ")
		builder.WriteString(topicName)
		builder.WriteString("\n")
	}
	fmt.Fprintf(&builder, "Review Question %d/%d\n", reviewIndex+1, reviewTotal)
	builder.WriteString(question.Text)

	options := quizOptions(question)
	if len(options) > 0 {
		builder.WriteString("\nOptions:")
		for _, option := range options {
			builder.WriteString("\n- ")
			builder.WriteString(option)
		}
	}

	builder.WriteString("\nReply with your answer.")
	return builder.String()
}

func renderChallengeReviewCompleteLocalized(locale string, reviewCorrect, reviewTotal int) string {
	return i18n.S(locale, i18n.MsgChallengeReviewDone, reviewCorrect, reviewTotal)
}

// --- Helper functions ---

func selectChallengeQuestions(questions []QuizQuestion, count int) []QuizQuestion {
	if len(questions) <= count {
		return append([]QuizQuestion(nil), questions...)
	}
	// Shuffle and take first count
	shuffled := make([]QuizQuestion, len(questions))
	copy(shuffled, questions)
	rand.Shuffle(len(shuffled), func(i, j int) {
		shuffled[i], shuffled[j] = shuffled[j], shuffled[i]
	})
	return shuffled[:count]
}

func isReviewAccept(text string) bool {
	normalized := strings.ToLower(strings.TrimSpace(text))
	switch normalized {
	case "review", "yes", "ulang", "ya", "ok", "okay", "sure", "y", "challenge:review":
		return true
	}
	return false
}

func isReviewSkip(text string) bool {
	normalized := strings.ToLower(strings.TrimSpace(text))
	return normalized == "challenge:skip"
}

func isChallengeReviewExit(text string) bool {
	normalized := strings.ToLower(strings.TrimSpace(text))
	switch normalized {
	case "stop", "cancel", "exit", "quit", "keluar", "berhenti":
		return true
	}
	return false
}
