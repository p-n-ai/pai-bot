package agent

import (
	"crypto/rand"
	"encoding/json"
	"errors"
	"strings"
	"time"
)

const (
	challengeStateWaiting       = "waiting"
	challengeStateReady         = "ready"
	challengeStateActive        = "active"
	challengeStateCompleted     = "completed"
	challengeSourcePublicQueue  = "public_queue"
	challengeSourcePrivateCode  = "private_code"
	challengeSourceAIFallback   = "ai_fallback"
	challengeOpponentHuman      = "human"
	challengeOpponentAI         = "ai"
	challengeCodeLength         = 6
	defaultChallengeQuestionMax = 5
	challengeAIFallbackWindow   = 30 * time.Second
	challengeFinishXP           = 15
)

var (
	ErrChallengeNotFound      = errors.New("challenge not found")
	ErrChallengeAlreadyActive = errors.New("user already has a live challenge")
	ErrChallengeSelfJoin      = errors.New("cannot join your own challenge")
	ErrChallengeFull          = errors.New("challenge already has two participants")
)

// Challenge stores queue state, private invite state, and live battle state.
type Challenge struct {
	ID                      string
	Code                    string
	Source                  string
	OpponentType            string
	CreatorID               string
	OpponentID              string
	TopicID                 string
	TopicName               string
	SubjectID               string
	SyllabusID              string
	Questions               []QuizQuestion
	QuestionCount           int
	State                   string
	CreatorReadyAt          *time.Time
	OpponentReadyAt         *time.Time
	CreatorCorrectCount     int
	OpponentCorrectCount    int
	CreatorCompletedAt      *time.Time
	OpponentCompletedAt     *time.Time
	CreatorFinishXPGranted  bool
	OpponentFinishXPGranted bool
	WinnerUserID            string
	WinnerXPGranted         bool
	Metadata                ChallengeMetadata
	CreatedAt               time.Time
	UpdatedAt               time.Time
}

type ChallengeMetadata struct {
	RequestedText      string              `json:"requested_text,omitempty"`
	RequestedTopicID   string              `json:"requested_topic_id,omitempty"`
	RequestedTopicName string              `json:"requested_topic_name,omitempty"`
	CreatorForm        string              `json:"creator_form,omitempty"`
	OpponentForm       string              `json:"opponent_form,omitempty"`
	AIProfile          *ChallengeAIProfile `json:"ai_profile,omitempty"`
}

type ChallengeAIProfile struct {
	Label          string  `json:"label"`
	AbilityScore   float64 `json:"ability_score"`
	PlannedCorrect int     `json:"planned_correct"`
}

type ChallengeInput struct {
	TopicID       string
	TopicName     string
	SubjectID     string
	SyllabusID    string
	Questions     []QuizQuestion
	QuestionCount int
	Metadata      ChallengeMetadata
}

type ChallengeCompletion struct {
	Challenge     *Challenge
	AwardFinishXP bool
	WinnerUserID  string
	AwardWinnerXP bool
}

type ChallengeStore interface {
	CreatePublicQueue(userID string, input ChallengeInput) (*Challenge, error)
	CreatePrivateChallenge(userID string, input ChallengeInput) (*Challenge, error)
	GetChallenge(code string) (*Challenge, error)
	GetCurrentChallengeForUser(userID string) (*Challenge, error)
	GetLiveChallengeForUser(userID string) (*Challenge, error)
	ListWaitingPublicChallenges(excludeUserID string) ([]*Challenge, error)
	ActivateHumanMatch(code, opponentID string, input ChallengeInput) (*Challenge, error)
	ActivateAIFallback(code string, input ChallengeInput) (*Challenge, error)
	JoinPrivateChallenge(code, opponentID string) (*Challenge, error)
	MarkReady(code, userID string) (*Challenge, error)
	CancelPublicQueue(userID string) (bool, error)
	CompleteChallenge(code, userID string, correctAnswers int) (*ChallengeCompletion, error)
}

func newChallengeRecord(userID, source, opponentType string, input ChallengeInput, now time.Time) *Challenge {
	return &Challenge{
		ID:            generateID(),
		Code:          generateUniqueChallengeCode(nil),
		Source:        source,
		OpponentType:  opponentType,
		CreatorID:     userID,
		TopicID:       input.TopicID,
		TopicName:     input.TopicName,
		SubjectID:     input.SubjectID,
		SyllabusID:    input.SyllabusID,
		Questions:     trimChallengeQuestions(input.Questions, input.QuestionCount),
		QuestionCount: normalizeChallengeQuestionCount(input.QuestionCount, input.Questions),
		State:         challengeStateWaiting,
		Metadata:      input.Metadata,
		CreatedAt:     now,
		UpdatedAt:     now,
	}
}

func completeChallengeRecord(challenge *Challenge, userID string, correctAnswers int) *ChallengeCompletion {
	now := time.Now()
	result := &ChallengeCompletion{Challenge: cloneChallenge(challenge)}
	correctAnswers = clampChallengeCorrectAnswers(correctAnswers, challenge.QuestionCount)

	if challenge.CreatorID == userID {
		if challenge.CreatorCompletedAt == nil {
			challenge.CreatorCorrectCount = correctAnswers
			challenge.CreatorCompletedAt = ptrTime(now)
			if !challenge.CreatorFinishXPGranted {
				challenge.CreatorFinishXPGranted = true
				result.AwardFinishXP = true
			}
		}
	} else if challenge.OpponentID == userID || challenge.OpponentType == challengeOpponentAI {
		if challenge.OpponentID == userID && challenge.OpponentCompletedAt == nil {
			challenge.OpponentCorrectCount = correctAnswers
			challenge.OpponentCompletedAt = ptrTime(now)
			if !challenge.OpponentFinishXPGranted {
				challenge.OpponentFinishXPGranted = true
				result.AwardFinishXP = true
			}
		}
	}

	if challenge.OpponentType == challengeOpponentAI {
		applyAIResultIfNeeded(challenge, now)
	}

	if challenge.CreatorCompletedAt == nil || challenge.OpponentCompletedAt == nil {
		challenge.State = challengeStateActive
		challenge.UpdatedAt = now
		result.Challenge = cloneChallenge(challenge)
		return result
	}

	challenge.State = challengeStateCompleted
	challenge.WinnerUserID = challengeWinnerUserID(challenge)
	if challenge.WinnerUserID != "" && !challenge.WinnerXPGranted {
		result.WinnerUserID = challenge.WinnerUserID
		result.AwardWinnerXP = true
		challenge.WinnerXPGranted = true
	}
	challenge.UpdatedAt = now
	result.Challenge = cloneChallenge(challenge)
	return result
}

func applyAIResultIfNeeded(challenge *Challenge, now time.Time) {
	if challenge.OpponentCompletedAt != nil || challenge.Metadata.AIProfile == nil {
		return
	}
	challenge.OpponentCorrectCount = clampChallengeCorrectAnswers(challenge.Metadata.AIProfile.PlannedCorrect, challenge.QuestionCount)
	challenge.OpponentCompletedAt = ptrTime(now)
}

func challengeWinnerUserID(challenge *Challenge) string {
	switch {
	case challenge.CreatorCorrectCount > challenge.OpponentCorrectCount:
		return challenge.CreatorID
	case challenge.OpponentCorrectCount > challenge.CreatorCorrectCount:
		if challenge.OpponentType == challengeOpponentAI {
			return ""
		}
		return challenge.OpponentID
	default:
		return ""
	}
}

func liveChallengeForUserLocked(challenges map[string]*Challenge, userID string) *Challenge {
	return latestChallengeForUserLocked(challenges, userID, true)
}

func latestChallengeForUserLocked(challenges map[string]*Challenge, userID string, liveOnly bool) *Challenge {
	var latest *Challenge
	for _, challenge := range challenges {
		if !challengeHasUser(challenge, userID) {
			continue
		}
		if liveOnly && challenge.State == challengeStateCompleted {
			continue
		}
		if latest == nil || challenge.UpdatedAt.After(latest.UpdatedAt) {
			latest = challenge
		}
	}
	return latest
}

func challengeHasUser(challenge *Challenge, userID string) bool {
	if challenge == nil {
		return false
	}
	return challenge.CreatorID == userID || challenge.OpponentID == userID
}

func challengeReadyToOpen(challenge *Challenge) bool {
	return challenge != nil && challenge.CreatorReadyAt != nil && challenge.OpponentReadyAt != nil
}

func normalizeChallengeQuestionCount(requested int, questions []QuizQuestion) int {
	if requested <= 0 || requested > defaultChallengeQuestionMax {
		requested = defaultChallengeQuestionMax
	}
	if len(questions) < requested {
		return len(questions)
	}
	return requested
}

func trimChallengeQuestions(questions []QuizQuestion, requested int) []QuizQuestion {
	count := normalizeChallengeQuestionCount(requested, questions)
	if count <= 0 {
		return nil
	}
	return cloneQuizQuestions(questions[:count])
}

func cloneChallenge(challenge *Challenge) *Challenge {
	if challenge == nil {
		return nil
	}
	copy := *challenge
	copy.Questions = cloneQuizQuestions(challenge.Questions)
	copy.Metadata = cloneChallengeMetadata(challenge.Metadata)
	return &copy
}

func cloneChallengeMetadata(meta ChallengeMetadata) ChallengeMetadata {
	copy := meta
	if meta.AIProfile != nil {
		profile := *meta.AIProfile
		copy.AIProfile = &profile
	}
	return copy
}

func cloneQuizQuestions(questions []QuizQuestion) []QuizQuestion {
	if len(questions) == 0 {
		return nil
	}
	cloned := make([]QuizQuestion, len(questions))
	copy(cloned, questions)
	return cloned
}

func sortChallengesByCreatedAt(challenges []*Challenge) {
	for i := 0; i < len(challenges)-1; i++ {
		for j := i + 1; j < len(challenges); j++ {
			if challenges[j].CreatedAt.Before(challenges[i].CreatedAt) {
				challenges[i], challenges[j] = challenges[j], challenges[i]
			}
		}
	}
}

func clampChallengeCorrectAnswers(correctAnswers, total int) int {
	if correctAnswers < 0 {
		return 0
	}
	if total > 0 && correctAnswers > total {
		return total
	}
	return correctAnswers
}

func normalizeChallengeCodeValue(value string) string {
	return strings.ToUpper(strings.TrimSpace(value))
}

func generateUniqueChallengeCode(exists func(code string) bool) string {
	for {
		code := randomChallengeCode(challengeCodeLength)
		if exists == nil || !exists(code) {
			return code
		}
	}
}

func randomChallengeCode(length int) string {
	const alphabet = "ABCDEFGHJKLMNPQRSTUVWXYZ23456789"
	buf := make([]byte, length)
	random := make([]byte, length)
	if _, err := rand.Read(random); err != nil {
		return strings.Repeat("A", length)
	}
	for i := range buf {
		buf[i] = alphabet[int(random[i])%len(alphabet)]
	}
	return string(buf)
}

func marshalChallengeMetadata(meta ChallengeMetadata) ([]byte, error) {
	return json.Marshal(meta)
}

func unmarshalChallengeMetadata(data []byte) (ChallengeMetadata, error) {
	if len(data) == 0 {
		return ChallengeMetadata{}, nil
	}
	var meta ChallengeMetadata
	err := json.Unmarshal(data, &meta)
	return meta, err
}

func ptrTime(v time.Time) *time.Time {
	value := v
	return &value
}
