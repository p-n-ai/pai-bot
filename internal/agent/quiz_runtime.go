// Copyright 2026 the P&AI authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package agent

import "strings"

const (
	quizRunStateActive = "active"
	quizRunStatePaused = "paused"
)

const (
	quizPauseReasonManual         = "manual_pause"
	quizPauseReasonSideQuestion   = "side_question"
	quizPauseReasonTeachFirst     = "teach_first"
	quizPauseReasonRestartPending = "restart_pending"
)

type quizTurnAction string

const (
	quizTurnActionAnswer       quizTurnAction = "answer"
	quizTurnActionExit         quizTurnAction = "exit"
	quizTurnActionHint         quizTurnAction = "hint"
	quizTurnActionPause        quizTurnAction = "pause"
	quizTurnActionRepeat       quizTurnAction = "repeat"
	quizTurnActionRestart      quizTurnAction = "restart"
	quizTurnActionResume       quizTurnAction = "resume"
	quizTurnActionShowQuestion quizTurnAction = "show_question"
	quizTurnActionSideQuestion quizTurnAction = "side_question"
	quizTurnActionTeachFirst   quizTurnAction = "teach_first"
	quizTurnActionUnclassified quizTurnAction = "unclassified"
)

func defaultQuizRunState() string {
	return quizRunStateActive
}

func classifyActiveQuizTurn(text string) quizTurnAction {
	normalized := normalizeQuizControlText(text)
	switch {
	case normalized == "":
		return quizTurnActionShowQuestion
	case isQuizExitIntent(normalized):
		return quizTurnActionExit
	case isQuizPauseIntent(normalized):
		return quizTurnActionPause
	case isQuizHintIntent(normalized):
		return quizTurnActionHint
	case isQuizRepeatIntent(normalized):
		return quizTurnActionRepeat
	case isQuizTeachIntent(normalized):
		return quizTurnActionTeachFirst
	case detectQuizIntent(text):
		return quizTurnActionRestart
	case isQuizSideQuestionIntent(normalized):
		return quizTurnActionSideQuestion
	default:
		return quizTurnActionAnswer
	}
}

func classifyPausedQuizTurn(text string) quizTurnAction {
	normalized := normalizeQuizControlText(text)
	switch {
	case normalized == "":
		return quizTurnActionResume
	case isQuizExitIntent(normalized):
		return quizTurnActionExit
	case isQuizResumeIntent(normalized):
		return quizTurnActionResume
	case isQuizHintIntent(normalized):
		return quizTurnActionHint
	case isQuizRepeatIntent(normalized):
		return quizTurnActionRepeat
	case detectQuizIntent(text):
		return quizTurnActionRestart
	default:
		return quizTurnActionUnclassified
	}
}

func normalizeQuizControlText(text string) string {
	return " " + strings.ToLower(strings.TrimSpace(text)) + " "
}

func isQuizHintIntent(text string) bool {
	return containsQuizControlPhrase(text,
		" hint ",
		" petunjuk ",
		" clue ",
		" help ",
		" help me ",
		" tolong ",
		" bantu ",
		" bagi hint ",
		" beri hint ",
		" idk ",
		" i dont know ",
		" i don't know ",
		" not sure ",
		" tak tahu ",
	)
}

func isQuizRepeatIntent(text string) bool {
	return containsQuizControlPhrase(text,
		" repeat ",
		" ulang ",
		" ulang soalan ",
		" repeat question ",
		" show question ",
		" what was the question ",
		" show me again ",
		" say that again ",
		" repeat quiz ",
	)
}

func isQuizPauseIntent(text string) bool {
	return containsQuizControlPhrase(text,
		" pause ",
		" pause quiz ",
		" pause this ",
		" hold on ",
		" hold up ",
		" hang on ",
		" later ",
		" brb ",
		" be right back ",
		" sebentar ",
		" kejap ",
		" kejap ya ",
		" nanti dulu ",
		" nanti ",
		" stop for now ",
	)
}

func isQuizExitIntent(text string) bool {
	return containsQuizControlPhrase(text,
		" stop ",
		" stop quiz ",
		" enough ",
		" done ",
		" done with quiz ",
		" cancel quiz ",
		" cancel ",
		" end quiz ",
		" quit quiz ",
		" quit ",
		" exit quiz ",
		" keluar quiz ",
		" tak nak quiz ",
		" taknak quiz ",
		" never mind quiz ",
	)
}

func isQuizResumeIntent(text string) bool {
	return containsQuizControlPhrase(text,
		" continue quiz ",
		" resume quiz ",
		" sambung quiz ",
		" teruskan quiz ",
		" lanjut quiz ",
		" back to quiz ",
		" continue ",
		" resume ",
		" sambung ",
		" teruskan ",
		" lanjut ",
	)
}

func isQuizTeachIntent(text string) bool {
	return containsQuizControlPhrase(text,
		" teach me ",
		" teach me first ",
		" explain this first ",
		" explain this ",
		" can you explain ",
		" explain dulu ",
		" teach this ",
		" ajar dulu ",
		" jelas dulu ",
		" tak faham ",
		" i dont get it ",
		" i don't get it ",
		" dont understand ",
		" don't understand ",
	)
}

func isQuizSideQuestionIntent(text string) bool {
	return containsQuizControlPhrase(text,
		" weather ",
		" cuaca ",
		" how are you ",
		" who are you ",
		" what time ",
		" tell me a joke ",
		" joke ",
	)
}

func containsQuizControlPhrase(text string, phrases ...string) bool {
	for _, phrase := range phrases {
		if strings.Contains(text, phrase) {
			return true
		}
	}
	return false
}
