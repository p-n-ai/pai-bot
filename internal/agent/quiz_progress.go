package agent

import (
	"log/slog"

	"github.com/p-n-ai/pai-bot/internal/progress"
)

func (e *Engine) recordQuizOutcomeAsync(userID, topicID, transport string, question QuizQuestion, correct bool) {
	if e.tracker == nil && e.xp == nil {
		return
	}

	go func() {
		syllabusID := "default"
		if e.curriculumLoader != nil {
			if topic, ok := e.curriculumLoader.GetTopic(topicID); ok && topic.SyllabusID != "" {
				syllabusID = topic.SyllabusID
			}
		}

		if e.tracker != nil {
			masteryBefore, err := e.tracker.GetMastery(userID, syllabusID, topicID)
			if err != nil {
				slog.Warn("failed to read quiz mastery before update", "user_id", userID, "topic_id", topicID, "error", err)
				masteryBefore = 0
			}

			if err := e.tracker.UpdateMastery(userID, syllabusID, topicID, quizMasterySignal(question, correct)); err != nil {
				slog.Warn("failed to update quiz mastery", "user_id", userID, "topic_id", topicID, "error", err)
			} else {
				e.syncGoalProgress(userID, syllabusID, topicID)
				if e.xp != nil {
					masteryAfter, err := e.tracker.GetMastery(userID, syllabusID, topicID)
					if err != nil {
						slog.Warn("failed to read quiz mastery after update", "user_id", userID, "topic_id", topicID, "error", err)
					} else if !progress.IsMastered(masteryBefore) && progress.IsMastered(masteryAfter) {
						if err := e.xp.Award(userID, progress.XPSourceMastery, progress.XPMasteryUp, map[string]any{
							"topic_id":     topicID,
							"syllabus_id":  syllabusID,
							"question_id":  question.ID,
							"difficulty":   question.Difficulty,
							"from_quiz":    true,
							"quiz_correct": correct,
						}); err != nil {
							slog.Warn("failed to award mastery xp from quiz", "user_id", userID, "topic_id", topicID, "error", err)
						}
					}
				}
			}
		}

		if correct && e.xp != nil {
			if err := e.xp.Award(userID, progress.XPSourceQuiz, progress.XPQuizCorrect, map[string]any{
				"topic_id":     topicID,
				"question_id":  question.ID,
				"difficulty":   question.Difficulty,
				"transport":    transport,
				"answer_type":  question.AnswerType,
				"learning_obj": question.LearningObjective,
			}); err != nil {
				slog.Warn("failed to award quiz xp", "user_id", userID, "topic_id", topicID, "error", err)
			}
		}
	}()
}

func quizMasterySignal(question QuizQuestion, correct bool) float64 {
	difficulty := normalizeQuizIntensity(question.Difficulty)
	if correct {
		switch difficulty {
		case "hard":
			return 0.85
		case "medium":
			return 0.75
		default:
			return 0.65
		}
	}

	switch difficulty {
	case "hard":
		return 0.25
	case "medium":
		return 0.20
	default:
		return 0.15
	}
}
