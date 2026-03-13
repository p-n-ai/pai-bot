package agent

import (
	"context"
	"fmt"
	"log/slog"
	"sort"
	"strings"
	"time"

	"github.com/p-n-ai/pai-bot/internal/chat"
	"github.com/p-n-ai/pai-bot/internal/i18n"
)

const (
	conversationStateTeaching      = "teaching"
	conversationStateQuizIntensity = "quiz_intensity"
	conversationStateQuizActive    = "quiz_active"
)

func (e *Engine) maybeHandleQuizTurn(_ context.Context, msg chat.InboundMessage, conv *Conversation) (string, bool) {
	if conv.State == conversationStateQuizIntensity && conv.PendingQuizTopicID != "" {
		return e.handleQuizIntensitySelection(msg, conv, conv.PendingQuizTopicID), true
	}
	if conv.State == conversationStateQuizActive && conv.QuizState != nil {
		return e.handleActiveQuizTurn(msg, conv, *conv.QuizState)
	}
	if conv.QuizState != nil && conv.QuizState.RunState == quizRunStatePaused {
		return e.handlePausedQuizTurn(msg, conv, *conv.QuizState)
	}

	topicID, ok := e.resolveQuizStartTopic(msg, conv)
	if !ok {
		return "", false
	}
	return e.startQuiz(msg, conv, topicID), true
}

func (e *Engine) resolveQuizStartTopic(msg chat.InboundMessage, conv *Conversation) (string, bool) {
	if e.curriculumLoader == nil {
		return "", false
	}

	if topicID, ok := parseQuizStartCallback(msg); ok {
		if _, found := e.curriculumLoader.GetAssessment(topicID); found {
			return topicID, true
		}
		return "", false
	}
	if !detectQuizIntent(msg.Text) {
		return "", false
	}

	if topic, _ := e.contextResolver.Resolve(msg.Text); topic != nil {
		if _, found := e.curriculumLoader.GetAssessment(topic.ID); found {
			return topic.ID, true
		}
	}
	if conv.TopicID != "" {
		if _, found := e.curriculumLoader.GetAssessment(conv.TopicID); found {
			return conv.TopicID, true
		}
	}
	for i := len(conv.Messages) - 1; i >= 0; i-- {
		text := sanitizeControlContent(conv.Messages[i].Content)
		if text == "" {
			continue
		}
		topic, _ := e.contextResolver.Resolve(text)
		if topic == nil {
			continue
		}
		if _, found := e.curriculumLoader.GetAssessment(topic.ID); found {
			return topic.ID, true
		}
	}
	return "", false
}

func (e *Engine) startQuiz(msg chat.InboundMessage, conv *Conversation, topicID string) string {
	if intensity := inferQuizStartIntensity(msg.Text); intensity != "" {
		if err := e.store.SetUserPreferredQuizIntensity(msg.UserID, intensity); err != nil {
			slog.Error("failed to persist explicit quiz intensity preference", "user_id", msg.UserID, "error", err)
			return i18n.S(e.messageLocale(msg, conv), i18n.MsgTechnicalIssue)
		}
		return e.startQuizWithIntensity(msg, conv, topicID, intensity, true)
	}

	if intensity, hasIntensity := e.store.GetUserPreferredQuizIntensity(msg.UserID); hasIntensity && normalizeQuizIntensity(intensity) != "" {
		return e.startQuizWithIntensity(msg, conv, topicID, intensity, true)
	}

	return e.startQuizWithIntensity(msg, conv, topicID, defaultQuizIntensity(), true)
}

func (e *Engine) startQuizWithIntensity(msg chat.InboundMessage, conv *Conversation, topicID, intensity string, storeStartMessage bool) string {
	assessment, ok := e.curriculumLoader.GetAssessment(topicID)
	if !ok || len(assessment.Questions) == 0 {
		return quizUnavailableText(e.messageLocale(msg, conv))
	}

	if storeStartMessage {
		if _, err := e.store.AddMessage(conv.ID, StoredMessage{
			Role:    "user",
			Content: msg.Text,
		}); err != nil {
			slog.Error("failed to store quiz start message", "conversation_id", conv.ID, "error", err)
		}
	}

	questions := filterQuizQuestionsByIntensity(questionsFromAssessment(assessment), intensity)
	session := NewQuizSession(msg.UserID, topicID, questions)
	session.Intensity = normalizeQuizIntensity(intensity)
	if err := e.store.UpdateConversationQuizState(conv.ID, conversationStateQuizActive, ConversationQuizState{
		TopicID:        topicID,
		Intensity:      session.Intensity,
		CurrentIndex:   session.CurrentIndex,
		CorrectAnswers: session.CorrectAnswers,
		RunState:       defaultQuizRunState(),
	}); err != nil {
		slog.Error("failed to persist quiz state", "conversation_id", conv.ID, "error", err)
		return i18n.S(e.messageLocale(msg, conv), i18n.MsgTechnicalIssue)
	}

	question, _ := session.NextQuestion()
	response := renderQuizQuestion(e.lookupTopicName(topicID), session, question)
	if _, err := e.store.AddMessage(conv.ID, StoredMessage{
		Role:    "assistant",
		Content: response,
	}); err != nil {
		slog.Error("failed to store quiz prompt", "conversation_id", conv.ID, "error", err)
	}
	e.logEventAsync(Event{
		ConversationID: conv.ID,
		UserID:         msg.UserID,
		EventType:      "quiz_started",
		Data: map[string]any{
			"topic_id":        topicID,
			"intensity":       session.Intensity,
			"question_count":  len(session.Questions),
			"start_transport": quizInputSource(msg),
		},
	})
	return response
}

func (e *Engine) handleQuizIntensitySelection(msg chat.InboundMessage, conv *Conversation, topicID string) string {
	intensity := parseQuizIntensityInput(msg)
	if intensity == "" {
		response := renderQuizIntensityPrompt(e.quizLearnerLabel(msg.UserID))
		if _, err := e.store.AddMessage(conv.ID, StoredMessage{
			Role:    "assistant",
			Content: response,
		}); err != nil {
			slog.Error("failed to store quiz intensity reprompt", "conversation_id", conv.ID, "error", err)
		}
		return response
	}

	if _, err := e.store.AddMessage(conv.ID, StoredMessage{
		Role:    "user",
		Content: strings.TrimSpace(msg.Text),
	}); err != nil {
		slog.Error("failed to store quiz intensity answer", "conversation_id", conv.ID, "error", err)
	}
	if err := e.store.SetUserPreferredQuizIntensity(msg.UserID, intensity); err != nil {
		slog.Error("failed to persist quiz intensity preference", "user_id", msg.UserID, "error", err)
		return i18n.S(e.messageLocale(msg, conv), i18n.MsgTechnicalIssue)
	}
	e.logEventAsync(Event{
		ConversationID: conv.ID,
		UserID:         msg.UserID,
		EventType:      "quiz_intensity_selected",
		Data: map[string]any{
			"topic_id":  topicID,
			"intensity": intensity,
			"transport": quizInputSource(msg),
		},
	})
	return e.startQuizWithIntensity(msg, conv, topicID, intensity, false)
}

func (e *Engine) handleActiveQuizTurn(msg chat.InboundMessage, conv *Conversation, state ConversationQuizState) (string, bool) {
	session, topicName, challenge, ok := e.quizSessionFromState(msg.UserID, state)
	if !ok {
		_ = e.store.ClearConversationQuizState(conv.ID, conversationStateTeaching)
		return quizUnavailableText(e.messageLocale(msg, conv)), true
	}
	question, hasQuestion := session.NextQuestion()
	action := classifyActiveQuizTurn(msg.Text)

	switch action {
	case quizTurnActionExit:
		if state.ChallengeCode != "" {
			state.RunState = quizRunStatePaused
			state.SuspendedBy = quizPauseReasonManual
			if err := e.store.UpdateConversationQuizState(conv.ID, conversationStateTeaching, state); err != nil {
				slog.Error("failed to pause challenge state on exit", "conversation_id", conv.ID, "error", err)
				return i18n.S(e.messageLocale(msg, conv), i18n.MsgTechnicalIssue), true
			}
			response := renderChallengePaused()
			if _, err := e.store.AddMessage(conv.ID, StoredMessage{Role: "assistant", Content: response}); err != nil {
				slog.Error("failed to store challenge pause response", "conversation_id", conv.ID, "error", err)
			}
			return response, true
		}
		if err := e.store.ClearConversationQuizState(conv.ID, conversationStateTeaching); err != nil {
			slog.Error("failed to clear quiz state on exit", "conversation_id", conv.ID, "error", err)
			return i18n.S(e.messageLocale(msg, conv), i18n.MsgTechnicalIssue), true
		}
		response := renderQuizExit()
		if _, err := e.store.AddMessage(conv.ID, StoredMessage{Role: "assistant", Content: response}); err != nil {
			slog.Error("failed to store quiz exit response", "conversation_id", conv.ID, "error", err)
		}
		e.logEventAsync(Event{
			ConversationID: conv.ID,
			UserID:         msg.UserID,
			EventType:      "quiz_exited",
			Data: map[string]any{
				"topic_id":       state.TopicID,
				"question_index": state.CurrentIndex,
			},
		})
		return response, true
	case quizTurnActionPause:
		return e.pauseQuizTurn(msg, conv, state, hasQuestion, question, quizPauseReasonManual)
	case quizTurnActionTeachFirst:
		return e.pauseQuizTurn(msg, conv, state, hasQuestion, question, quizPauseReasonTeachFirst)
	case quizTurnActionSideQuestion:
		return e.pauseQuizTurn(msg, conv, state, hasQuestion, question, quizPauseReasonSideQuestion)
	case quizTurnActionHint:
		response := renderQuizHint(question)
		if _, err := e.store.AddMessage(conv.ID, StoredMessage{Role: "assistant", Content: response}); err != nil {
			slog.Error("failed to store quiz hint response", "conversation_id", conv.ID, "error", err)
		}
		return response, true
	case quizTurnActionRepeat, quizTurnActionShowQuestion:
		response := renderQuestionForState(state, topicName, session, question)
		if _, err := e.store.AddMessage(conv.ID, StoredMessage{Role: "assistant", Content: response}); err != nil {
			slog.Error("failed to store quiz repeat response", "conversation_id", conv.ID, "error", err)
		}
		return response, true
	case quizTurnActionRestart:
		if state.ChallengeCode != "" {
			response := renderQuestionForState(state, topicName, session, question)
			if _, err := e.store.AddMessage(conv.ID, StoredMessage{Role: "assistant", Content: response}); err != nil {
				slog.Error("failed to store challenge repeat response", "conversation_id", conv.ID, "error", err)
			}
			return response, true
		}
		topicID, resolved := e.resolveQuizStartTopic(msg, conv)
		if !resolved {
			topicID = state.TopicID
		}
		return e.startQuiz(msg, conv, topicID), true
	}

	answerText := parseQuizAnswerInput(msg)
	if answerText == "" {
		if !hasQuestion {
			_ = e.store.ClearConversationQuizState(conv.ID, conversationStateTeaching)
			if state.ChallengeCode != "" {
				return formatChallengeOverview(challenge, msg.UserID, nil, time.Now()), true
			}
			return quizCompletedText(e.messageLocale(msg, conv), session.Summary()), true
		}
		return renderQuestionForState(state, topicName, session, question), true
	}

	if _, err := e.store.AddMessage(conv.ID, StoredMessage{
		Role:    "user",
		Content: answerText,
	}); err != nil {
		slog.Error("failed to store quiz answer", "conversation_id", conv.ID, "error", err)
	}

	var result QuizAnswerResult
	if state.ChallengeCode != "" {
		result = submitChallengeAnswer(session, answerText)
	} else {
		result = session.SubmitAnswer(answerText)
	}
	e.recordQuizOutcomeAsync(msg.UserID, state.TopicID, quizInputSource(msg), question, result.Correct, state.ChallengeCode == "")
	if !result.Correct && state.ChallengeCode == "" {
		response := renderQuizRetry(e.messageLocale(msg, conv), result)
		if _, err := e.store.AddMessage(conv.ID, StoredMessage{
			Role:    "assistant",
			Content: response,
		}); err != nil {
			slog.Error("failed to store quiz retry response", "conversation_id", conv.ID, "error", err)
		}
		e.logEventAsync(Event{
			ConversationID: conv.ID,
			UserID:         msg.UserID,
			EventType:      "quiz_answer_incorrect",
			Data: map[string]any{
				"topic_id":         state.TopicID,
				"question_index":   state.CurrentIndex,
				"answer_transport": quizInputSource(msg),
			},
		})
		return response, true
	}

	nextState := ConversationQuizState{
		TopicID:        state.TopicID,
		Intensity:      state.Intensity,
		CurrentIndex:   session.CurrentIndex,
		CorrectAnswers: session.CorrectAnswers,
		ChallengeCode:  state.ChallengeCode,
		RunState:       defaultQuizRunState(),
	}

	var response string
	if session.IsComplete() {
		if state.ChallengeCode != "" {
			response = e.completeChallengeQuiz(msg.UserID, conv, challenge, result, session.Summary())
		} else {
			if err := e.store.ClearConversationQuizState(conv.ID, conversationStateTeaching); err != nil {
				slog.Error("failed to restore teaching state after quiz", "conversation_id", conv.ID, "error", err)
			}
			response = renderQuizCompletion(e.messageLocale(msg, conv), result, session.Summary())
			e.logEventAsync(Event{
				ConversationID: conv.ID,
				UserID:         msg.UserID,
				EventType:      "quiz_completed",
				Data: map[string]any{
					"topic_id":        state.TopicID,
					"correct_answers": session.CorrectAnswers,
					"total_questions": len(session.Questions),
				},
			})
		}
	} else {
		if err := e.store.UpdateConversationQuizState(conv.ID, conversationStateQuizActive, nextState); err != nil {
			slog.Error("failed to update quiz state", "conversation_id", conv.ID, "error", err)
			return i18n.S(e.messageLocale(msg, conv), i18n.MsgTechnicalIssue), true
		}
		question, _ := session.NextQuestion()
		if state.ChallengeCode != "" {
			response = renderChallengeAdvance(state.ChallengeCode, topicName, session, question, result)
		} else {
			response = renderQuizAdvance(topicName, session, question, result)
		}
	}

	if _, err := e.store.AddMessage(conv.ID, StoredMessage{
		Role:    "assistant",
		Content: response,
	}); err != nil {
		slog.Error("failed to store quiz response", "conversation_id", conv.ID, "error", err)
	}
	return response, true
}

func (e *Engine) handlePausedQuizTurn(msg chat.InboundMessage, conv *Conversation, state ConversationQuizState) (string, bool) {
	action := classifyPausedQuizTurn(msg.Text)
	switch action {
	case quizTurnActionExit:
		if state.ChallengeCode != "" {
			response := renderChallengePaused()
			if _, err := e.store.AddMessage(conv.ID, StoredMessage{Role: "assistant", Content: response}); err != nil {
				slog.Error("failed to store paused challenge exit response", "conversation_id", conv.ID, "error", err)
			}
			return response, true
		}
		if err := e.store.ClearConversationQuizState(conv.ID, conversationStateTeaching); err != nil {
			slog.Error("failed to clear paused quiz state on exit", "conversation_id", conv.ID, "error", err)
			return i18n.S(e.messageLocale(msg, conv), i18n.MsgTechnicalIssue), true
		}
		response := renderQuizExit()
		if _, err := e.store.AddMessage(conv.ID, StoredMessage{Role: "assistant", Content: response}); err != nil {
			slog.Error("failed to store paused quiz exit response", "conversation_id", conv.ID, "error", err)
		}
		return response, true
	case quizTurnActionResume, quizTurnActionHint, quizTurnActionRepeat:
		return e.resumePausedQuizTurn(msg, conv, state, action), true
	case quizTurnActionRestart:
		topicID, resolved := e.resolveQuizStartTopic(msg, conv)
		if !resolved {
			topicID = state.TopicID
		}
		return e.startQuiz(msg, conv, topicID), true
	default:
		return "", false
	}
}

func (e *Engine) resumePausedQuizTurn(msg chat.InboundMessage, conv *Conversation, state ConversationQuizState, action quizTurnAction) string {
	session, topicName, _, ok := e.quizSessionFromState(msg.UserID, state)
	if !ok {
		_ = e.store.ClearConversationQuizState(conv.ID, conversationStateTeaching)
		return quizUnavailableText(e.messageLocale(msg, conv))
	}
	question, hasQuestion := session.NextQuestion()
	if !hasQuestion {
		_ = e.store.ClearConversationQuizState(conv.ID, conversationStateTeaching)
		if state.ChallengeCode != "" {
			challenge, err := e.challenges.GetChallenge(state.ChallengeCode)
			if err == nil && challenge != nil {
				return formatChallengeOverview(challenge, msg.UserID, nil, time.Now())
			}
		}
		return quizCompletedText(e.messageLocale(msg, conv), session.Summary())
	}

	suspendedBy := state.SuspendedBy
	state.RunState = defaultQuizRunState()
	state.SuspendedBy = ""
	if err := e.store.UpdateConversationQuizState(conv.ID, conversationStateQuizActive, state); err != nil {
		slog.Error("failed to resume quiz state", "conversation_id", conv.ID, "error", err)
		return i18n.S(e.messageLocale(msg, conv), i18n.MsgTechnicalIssue)
	}

	var response string
	switch action {
	case quizTurnActionHint:
		if state.ChallengeCode != "" {
			response = renderChallengeResumeWithHint(state.ChallengeCode, topicName, session, question)
		} else {
			response = renderQuizResumeWithHint(topicName, session, question)
		}
	case quizTurnActionRepeat:
		if state.ChallengeCode != "" {
			response = renderChallengeResumed(state.ChallengeCode, topicName, session, question)
		} else {
			response = renderQuizResumed(topicName, session, question)
		}
	default:
		if state.ChallengeCode != "" {
			response = renderChallengeResumed(state.ChallengeCode, topicName, session, question)
		} else {
			response = renderQuizResumed(topicName, session, question)
		}
	}
	if _, err := e.store.AddMessage(conv.ID, StoredMessage{Role: "assistant", Content: response}); err != nil {
		slog.Error("failed to store quiz resume response", "conversation_id", conv.ID, "error", err)
	}
	e.logEventAsync(Event{
		ConversationID: conv.ID,
		UserID:         msg.UserID,
		EventType:      "quiz_resumed",
		Data: map[string]any{
			"topic_id":     state.TopicID,
			"suspended_by": suspendedBy,
		},
	})
	return response
}

func (e *Engine) pauseQuizTurn(msg chat.InboundMessage, conv *Conversation, state ConversationQuizState, hasQuestion bool, question QuizQuestion, reason string) (string, bool) {
	state.RunState = quizRunStatePaused
	state.SuspendedBy = reason
	if err := e.store.UpdateConversationQuizState(conv.ID, conversationStateTeaching, state); err != nil {
		slog.Error("failed to pause quiz state", "conversation_id", conv.ID, "error", err)
		return i18n.S(e.messageLocale(msg, conv), i18n.MsgTechnicalIssue), true
	}

	e.logEventAsync(Event{
		ConversationID: conv.ID,
		UserID:         msg.UserID,
		EventType:      "quiz_paused",
		Data: map[string]any{
			"topic_id":       state.TopicID,
			"question_index": state.CurrentIndex,
			"reason":         reason,
		},
	})

	if reason == quizPauseReasonManual {
		response := renderQuizPaused()
		if _, err := e.store.AddMessage(conv.ID, StoredMessage{Role: "assistant", Content: response}); err != nil {
			slog.Error("failed to store quiz pause response", "conversation_id", conv.ID, "error", err)
		}
		return response, true
	}

	if reason == quizPauseReasonTeachFirst && hasQuestion {
		if _, err := e.store.AddMessage(conv.ID, StoredMessage{
			Role: "user",
			Content: fmt.Sprintf("[Quiz paused for teaching]\nTopic: %s\nCurrent question: %s\nStudent wants teaching help before continuing the quiz.",
				e.lookupTopicName(state.TopicID),
				question.Text,
			),
		}); err != nil {
			slog.Error("failed to store quiz teaching bridge", "conversation_id", conv.ID, "error", err)
		}
	}

	return "", false
}

func parseQuizStartCallback(msg chat.InboundMessage) (string, bool) {
	if msg.CallbackQueryID == "" || !strings.HasPrefix(msg.Text, "quiz:start:") {
		return "", false
	}
	topicID := strings.TrimSpace(strings.TrimPrefix(msg.Text, "quiz:start:"))
	return topicID, topicID != ""
}

func parseQuizAnswerInput(msg chat.InboundMessage) string {
	if msg.CallbackQueryID != "" && strings.HasPrefix(msg.Text, "quiz:answer:") {
		return strings.TrimSpace(strings.TrimPrefix(msg.Text, "quiz:answer:"))
	}
	return strings.TrimSpace(msg.Text)
}

func parseQuizIntensityInput(msg chat.InboundMessage) string {
	if msg.CallbackQueryID != "" && strings.HasPrefix(msg.Text, "quiz:intensity:") {
		return normalizeQuizIntensity(strings.TrimSpace(strings.TrimPrefix(msg.Text, "quiz:intensity:")))
	}
	return normalizeQuizIntensity(msg.Text)
}

func detectQuizIntent(text string) bool {
	normalized := strings.ToLower(strings.TrimSpace(text))
	if normalized == "" {
		return false
	}
	switch normalized {
	case "quiz", "kuiz", "latihan":
		return true
	}
	phrases := []string{
		"quiz me",
		"give me a quiz",
		"start a quiz",
		"start quiz",
		"can you quiz me",
		"lets do a quiz",
		"let's do a quiz",
		"another quiz",
		"new quiz",
		"practice questions",
		"give me practice questions",
		"give me some practice questions",
		"test me",
		"test me on",
		"give me a test",
		"start a test",
		"kuiz saya",
		"beri saya kuiz",
		"bagi saya kuiz",
		"bagi saya quiz",
		"uji saya",
		"uji saya tentang",
		"soalan latihan",
		"beri saya latihan",
		"bagi saya latihan",
	}
	for _, phrase := range phrases {
		if strings.Contains(normalized, phrase) {
			return true
		}
	}
	if containsAny(normalized, "quiz", "kuiz") && containsAny(normalized,
		"give me",
		"another",
		"new",
		"lets do",
		"let's do",
		"bagi saya",
		"beri saya",
		"nak",
		"mahu",
		"mau",
		"uji saya",
		"test me",
	) {
		return true
	}
	if strings.Contains(normalized, "latihan") && containsAny(normalized,
		"beri saya",
		"bagi saya",
		"soalan",
		"nak",
		"mahu",
		"mau",
	) {
		return true
	}
	return false
}

func containsAny(text string, parts ...string) bool {
	for _, part := range parts {
		if strings.Contains(text, part) {
			return true
		}
	}
	return false
}

func renderQuizQuestion(topicName string, session *QuizSession, question QuizQuestion) string {
	var builder strings.Builder
	if topicName != "" {
		builder.WriteString("Quiz mode: ")
		builder.WriteString(topicName)
		builder.WriteString("\n")
	}
	fmt.Fprintf(&builder, "Question %d/%d\n", session.CurrentIndex+1, len(session.Questions))
	builder.WriteString(question.Text)

	options := quizOptions(question)
	if len(options) > 0 {
		builder.WriteString("\nOptions:")
		for _, option := range options {
			builder.WriteString("\n- ")
			builder.WriteString(option)
		}
	}

	switch question.AnswerType {
	case "free_text":
		builder.WriteString("\nReply with a short explanation.")
	default:
		builder.WriteString("\nReply with your answer.")
	}
	return builder.String()
}

func renderQuestionForState(state ConversationQuizState, topicName string, session *QuizSession, question QuizQuestion) string {
	if state.ChallengeCode != "" {
		return renderChallengeQuestion(state.ChallengeCode, topicName, session, question)
	}
	return renderQuizQuestion(topicName, session, question)
}

func renderQuizIntensityPrompt(learnerLabel string) string {
	if learnerLabel != "" {
		return fmt.Sprintf("%s, what intensity do you want for this quiz?\nReply with: easy, medium, hard, or mixed.", learnerLabel)
	}
	return "What intensity do you want for this quiz?\nReply with: easy, medium, hard, or mixed."
}

func renderQuizAdvance(topicName string, session *QuizSession, question QuizQuestion, result QuizAnswerResult) string {
	return result.Feedback + "\n\n" + renderQuizQuestion(topicName, session, question)
}

func renderQuizRetry(locale string, result QuizAnswerResult) string {
	var builder strings.Builder
	builder.WriteString(result.Feedback)
	if result.Hint != "" {
		builder.WriteString("\nHint: ")
		builder.WriteString(result.Hint)
	}
	builder.WriteString("\nTry the same question again.")
	return builder.String()
}

func renderQuizHint(question QuizQuestion) string {
	if len(question.Hints) == 0 {
		return "No extra hint for this question yet. Try the same question again."
	}
	return "Hint: " + question.Hints[0].Text + "\nTry the same question again."
}

func renderQuizPaused() string {
	return "Okay, I paused the quiz. We can talk first. Say continue quiz when you want to resume."
}

func renderQuizExit() string {
	return "Okay, we can stop the quiz here. If you want another one later, just ask naturally."
}

func renderQuizResumed(topicName string, session *QuizSession, question QuizQuestion) string {
	return "Resuming your quiz.\n\n" + renderQuizQuestion(topicName, session, question)
}

func renderQuizResumeWithHint(topicName string, session *QuizSession, question QuizQuestion) string {
	if len(question.Hints) == 0 {
		return renderQuizResumed(topicName, session, question)
	}
	return "Resuming your quiz.\nHint: " + question.Hints[0].Text + "\n\n" + renderQuizQuestion(topicName, session, question)
}

func renderQuizCompletion(locale string, result QuizAnswerResult, summary QuizSummary) string {
	var builder strings.Builder
	builder.WriteString(result.Feedback)
	builder.WriteString("\n\n")
	builder.WriteString(quizCompletedText(locale, summary))
	return builder.String()
}

func quizCompletedText(_ string, summary QuizSummary) string {
	return fmt.Sprintf(
		"Quiz complete.\nScore: %d/%d (%d%%)\nSend another topic whenever you want the next quiz.",
		summary.CorrectAnswers,
		summary.TotalQuestions,
		summary.ScorePercentage,
	)
}

func quizUnavailableText(_ string) string {
	return "I can't start a quiz for that topic yet because the assessment set is not available."
}

func quizOptions(question QuizQuestion) []string {
	if question.AnswerType != "multiple_choice" {
		return nil
	}
	options := []string{question.Answer}
	for _, distractor := range question.Distractors {
		options = append(options, distractor.Value)
	}
	sort.Strings(options)
	return options
}

func quizInputSource(msg chat.InboundMessage) string {
	if msg.CallbackQueryID != "" {
		return "button"
	}
	return "text"
}

func (e *Engine) lookupTopicName(topicID string) string {
	if e.curriculumLoader == nil {
		return ""
	}
	topic, ok := e.curriculumLoader.GetTopic(topicID)
	if !ok {
		return ""
	}
	return topic.Name
}

func (e *Engine) quizLearnerLabel(userID string) string {
	name, hasName := e.store.GetUserName(userID)
	form, hasForm := e.store.GetUserForm(userID)
	name = strings.TrimSpace(name)
	form = strings.TrimSpace(form)

	switch {
	case hasName && name != "" && hasForm && form != "":
		return fmt.Sprintf("%s (Form %s)", name, form)
	case hasName && name != "":
		return name
	case hasForm && form != "":
		return fmt.Sprintf("Form %s learner", form)
	default:
		return ""
	}
}
