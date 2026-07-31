// Copyright 2026 the P&AI authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package agent

import (
	"context"
	"errors"
	"log/slog"
	"strconv"
	"strings"

	"github.com/p-n-ai/pai-bot/internal/chat"
	"github.com/p-n-ai/pai-bot/internal/curriculum"
	"github.com/p-n-ai/pai-bot/internal/i18n"
	"github.com/p-n-ai/pai-bot/internal/progress"
)

func (e *Engine) maybeHandleCurriculumAttempt(
	ctx context.Context,
	msg chat.InboundMessage,
	conv *Conversation,
) (string, bool) {
	if e.curriculumRuntime == nil || conv == nil || conv.CurriculumState == nil {
		return "", false
	}

	state := *conv.CurriculumState
	attemptID, hasAttemptID := curriculumAttemptID(msg)
	if hasAttemptID && state.LastAttempt != nil && state.LastAttempt.AttemptID == attemptID {
		return state.LastAttempt.Response, true
	}
	answer, isAnswer := e.curriculumAttemptAnswer(msg, state)
	if state.ActiveQuestionID == "" || !isAnswer {
		return "", false
	}
	if !hasAttemptID {
		return curriculumAttemptNeedsDeliveryID(e.messageLocale(msg, conv)), true
	}

	identity, err := learnerIdentityForMessage(msg)
	if err != nil {
		return i18n.S(e.messageLocale(msg, conv), i18n.MsgTechnicalIssue), true
	}
	learnerID, err := e.progressLearnerID(identity)
	if err != nil {
		slog.Error("resolve curriculum attempt learner", "conversation_id", conv.ID, "error", err)
		return i18n.S(e.messageLocale(msg, conv), i18n.MsgTechnicalIssue), true
	}
	result, err := e.curriculumRuntime.RecordAttempt(ctx, curriculum.AttemptInput{
		AttemptID:  attemptID,
		LearnerID:  learnerID,
		TopicID:    state.ActiveTopicID,
		QuestionID: state.ActiveQuestionID,
		Answer:     answer,
		Locale:     e.messageLocale(msg, conv),
	})
	if err != nil {
		if errors.Is(err, curriculum.ErrAssessmentNotGradeable) {
			slog.Error(
				"planned curriculum check is not gradeable",
				"conversation_id", conv.ID,
				"topic_id", state.ActiveTopicID,
				"question_id", state.ActiveQuestionID,
				"error", err,
			)
		} else {
			slog.Error("record curriculum attempt", "conversation_id", conv.ID, "error", err)
		}
		return i18n.S(e.messageLocale(msg, conv), i18n.MsgTechnicalIssue), true
	}
	if !result.Applied {
		return curriculumAttemptAlreadyRecorded(e.messageLocale(msg, conv)), true
	}

	response := renderCurriculumAttemptResponse(e.messageLocale(msg, conv), result.Correct)
	questionID := state.ActiveQuestionID
	attempt := ConversationCurriculumAttempt{
		AttemptID:     attemptID,
		LearnerAnswer: answer,
		Response:      response,
		Applied:       true,
		Correct:       result.Correct,
		Score:         result.Score,
		MasteryBefore: result.MasteryBefore,
		MasteryAfter:  result.MasteryAfter,
	}
	state.ActiveQuestionID = ""
	state.LastAttempt = &attempt
	if err := e.store.SetConversationCurriculumState(conv.ID, state); err != nil {
		slog.Error("persist curriculum attempt state", "conversation_id", conv.ID, "error", err)
		return i18n.S(e.messageLocale(msg, conv), i18n.MsgTechnicalIssue), true
	}
	if _, err := e.store.AddMessage(conv.ID, StoredMessage{Role: "user", Content: msg.Text}); err != nil {
		slog.Error("store curriculum attempt", "conversation_id", conv.ID, "error", err)
	}
	if _, err := e.store.AddMessage(conv.ID, StoredMessage{Role: "assistant", Content: response}); err != nil {
		slog.Error("store curriculum attempt response", "conversation_id", conv.ID, "error", err)
	}

	e.applyCurriculumAttemptSideEffects(identity, state.ActiveTopicID, questionID, result)
	e.logEventAsync(Event{
		ConversationID: conv.ID,
		UserID:         identity.ExternalID(),
		EventType:      "curriculum_attempt_recorded",
		Data: map[string]any{
			"topic_id":     state.ActiveTopicID,
			"question_id":  result.Evidence.QuestionID,
			"correct":      result.Correct,
			"attempt_id":   attemptID,
			"mastery_from": result.MasteryBefore,
			"mastery_to":   result.MasteryAfter,
		},
	})
	e.recordActivityAsync(identity)
	return response, true
}

func (e *Engine) persistCurriculumPlanState(
	conv *Conversation,
	turnID string,
	goalTopicID string,
	plan curriculum.TeachingPlan,
) error {
	state := ConversationCurriculumState{
		GoalTopicID:       goalTopicID,
		ActiveTopicID:     plan.Target.TopicID,
		ActiveObjectiveID: plan.Target.ObjectiveID,
		RunID:             turnID,
	}
	if plan.Check != nil {
		state.ActiveQuestionID = plan.Check.QuestionID
		state.ActiveAnswerType = plan.Check.AnswerType
		for _, option := range plan.Check.Options {
			state.ActiveOptions = append(state.ActiveOptions, ConversationCurriculumOption{
				ID:   option.ID,
				Text: option.Text,
			})
		}
	}
	if conv != nil && conv.CurriculumState != nil {
		state.LastAttempt = conv.CurriculumState.LastAttempt
	}
	return e.store.SetConversationCurriculumState(conv.ID, state)
}

func (e *Engine) retirePendingCurriculumCheck(conv *Conversation, topicID, runID string) error {
	if conv == nil || conv.CurriculumState == nil {
		return nil
	}
	return e.store.SetConversationCurriculumState(conv.ID, ConversationCurriculumState{
		GoalTopicID:   topicID,
		ActiveTopicID: topicID,
		RunID:         runID,
		LastAttempt:   conv.CurriculumState.LastAttempt,
	})
}

func (e *Engine) curriculumAttemptAnswer(msg chat.InboundMessage, state ConversationCurriculumState) (string, bool) {
	if classifyActiveQuizTurn(msg.Text) != quizTurnActionAnswer {
		return "", false
	}
	answer := strings.TrimSpace(msg.Text)
	if answer == "" {
		return "", false
	}
	if topic, _ := e.resolveCurriculumContext(msg.UserID, "", answer); topic != nil && topic.ID != state.ActiveTopicID {
		return "", false
	}
	if explicitAnswer, explicit := stripCurriculumAnswerPrefix(answer); explicit {
		return explicitAnswer, explicitAnswer != ""
	}

	switch state.ActiveAnswerType {
	case "multiple_choice":
		for _, option := range state.ActiveOptions {
			if strings.EqualFold(answer, option.ID) || strings.EqualFold(answer, option.Text) {
				return answer, true
			}
		}
		if len(state.ActiveOptions) > 0 {
			return "", false
		}
		return answer, isPlausibleExactCurriculumAnswer(answer)
	case "exact":
		return answer, isPlausibleExactCurriculumAnswer(answer)
	default:
		return "", false
	}
}

func isPlausibleExactCurriculumAnswer(answer string) bool {
	if strings.Contains(answer, "?") || len(strings.Fields(answer)) > 16 {
		return false
	}
	_, err := strconv.ParseFloat(strings.TrimSpace(answer), 64)
	return err == nil
}

func stripCurriculumAnswerPrefix(answer string) (string, bool) {
	lower := strings.ToLower(answer)
	for _, prefix := range []string{"answer:", "jawapan:", "jawab:"} {
		if strings.HasPrefix(lower, prefix) {
			return strings.TrimSpace(answer[len(prefix):]), true
		}
	}
	return answer, false
}

func (e *Engine) persistQuizCurriculumAttempt(
	msg chat.InboundMessage,
	conv *Conversation,
	quizState ConversationQuizState,
	question QuizQuestion,
	attemptID string,
	result curriculum.AttemptResult,
	response string,
) error {
	attempt := ConversationCurriculumAttempt{
		AttemptID:     attemptID,
		LearnerAnswer: msg.Text,
		Response:      response,
		Applied:       result.Applied,
		Correct:       result.Correct,
		Score:         result.Score,
		MasteryBefore: result.MasteryBefore,
		MasteryAfter:  result.MasteryAfter,
	}
	state := ConversationCurriculumState{
		GoalTopicID:       quizState.TopicID,
		ActiveTopicID:     quizState.TopicID,
		ActiveObjectiveID: question.LearningObjective,
		RunID:             "quiz:" + conv.ID,
		LastAttempt:       &attempt,
	}
	if err := e.store.SetConversationCurriculumState(conv.ID, state); err != nil {
		return err
	}
	identity, err := learnerIdentityForMessage(msg)
	if err != nil {
		return err
	}
	e.applyCurriculumAttemptSideEffects(identity, quizState.TopicID, question.ID, result)
	e.logEventAsync(Event{
		ConversationID: conv.ID,
		UserID:         identity.ExternalID(),
		EventType:      "curriculum_attempt_recorded",
		Data: map[string]any{
			"topic_id":     quizState.TopicID,
			"question_id":  question.ID,
			"correct":      result.Correct,
			"attempt_id":   attemptID,
			"mastery_from": result.MasteryBefore,
			"mastery_to":   result.MasteryAfter,
			"from_quiz":    true,
		},
	})
	e.recordActivityAsync(identity)
	return nil
}

func (e *Engine) applyCurriculumAttemptSideEffects(
	identity LearnerIdentity,
	topicID string,
	questionID string,
	result curriculum.AttemptResult,
) {
	if !result.Applied || e.curriculumLoader == nil {
		return
	}
	topic, found := e.curriculumLoader.GetTopic(topicID)
	if !found {
		return
	}
	syllabusID := topic.SyllabusID
	if syllabusID == "" {
		syllabusID = "default"
	}
	e.syncGoalProgress(identity, syllabusID, topicID)
	e.checkTopicUnlocks(identity, syllabusID, &topic)

	if result.Correct && e.xp != nil {
		if err := e.xp.Award(identity.ExternalID(), progress.XPSourceQuiz, progress.XPQuizCorrect, map[string]any{
			"topic_id":    topicID,
			"question_id": questionID,
			"source":      "curriculum_engine",
		}); err != nil {
			slog.Warn("award curriculum check xp", "topic_id", topicID, "error", err)
		}
	}
	if !progress.IsMastered(result.MasteryBefore) && progress.IsMastered(result.MasteryAfter) {
		if e.xp != nil {
			if err := e.xp.Award(identity.ExternalID(), progress.XPSourceMastery, progress.XPMasteryUp, map[string]any{
				"topic_id":    topicID,
				"syllabus_id": syllabusID,
				"question_id": questionID,
			}); err != nil {
				slog.Warn("award curriculum mastery xp", "topic_id", topicID, "error", err)
			}
		}
		if e.milestones != nil && e.userABGroup(identity) == ABGroupA {
			e.milestones.add(
				identity,
				FormatTopicMasteredCelebration(e.resolveUserLocale(identity), topic.Name, progress.XPMasteryUp),
			)
		}
	}
}

func renderCurriculumAttemptResponse(locale string, correct bool) string {
	if strings.HasPrefix(strings.ToLower(locale), "ms") {
		if correct {
			return "Betul — jawapan itu direkodkan. Bagus."
		}
		return "Belum tepat. Tak apa — saya akan laraskan langkah seterusnya."
	}
	if correct {
		return "Correct — that answer is recorded. Nice work."
	}
	return "Not quite. That’s okay — I’ll adjust the next step."
}

func curriculumAttemptNeedsDeliveryID(locale string) string {
	if strings.HasPrefix(strings.ToLower(locale), "ms") {
		return "Saya tak dapat merekod jawapan ini dengan selamat. Hantar semula sekali lagi."
	}
	return "I couldn’t record that answer safely. Please send it once more."
}

func curriculumAttemptAlreadyRecorded(locale string) string {
	if strings.HasPrefix(strings.ToLower(locale), "ms") {
		return "Jawapan itu sudah direkodkan."
	}
	return "That answer was already recorded."
}
