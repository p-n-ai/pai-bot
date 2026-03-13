package agent

import (
	"sync"
	"time"
)

type MemoryChallengeStore struct {
	mu         sync.RWMutex
	challenges map[string]*Challenge
}

func NewMemoryChallengeStore() *MemoryChallengeStore {
	return &MemoryChallengeStore{challenges: make(map[string]*Challenge)}
}

func (s *MemoryChallengeStore) CreatePublicQueue(userID string, input ChallengeInput) (*Challenge, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if liveChallengeForUserLocked(s.challenges, userID) != nil {
		return nil, ErrChallengeAlreadyActive
	}
	now := time.Now()
	challenge := newChallengeRecord(userID, challengeSourcePublicQueue, challengeOpponentHuman, input, now)
	s.challenges[challenge.Code] = challenge
	return cloneChallenge(challenge), nil
}

func (s *MemoryChallengeStore) CreatePrivateChallenge(userID string, input ChallengeInput) (*Challenge, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if liveChallengeForUserLocked(s.challenges, userID) != nil {
		return nil, ErrChallengeAlreadyActive
	}
	now := time.Now()
	challenge := newChallengeRecord(userID, challengeSourcePrivateCode, challengeOpponentHuman, input, now)
	s.challenges[challenge.Code] = challenge
	return cloneChallenge(challenge), nil
}

func (s *MemoryChallengeStore) GetChallenge(code string) (*Challenge, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	challenge, ok := s.challenges[normalizeChallengeCodeValue(code)]
	if !ok {
		return nil, ErrChallengeNotFound
	}
	return cloneChallenge(challenge), nil
}

func (s *MemoryChallengeStore) GetCurrentChallengeForUser(userID string) (*Challenge, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return cloneChallenge(latestChallengeForUserLocked(s.challenges, userID, false)), nil
}

func (s *MemoryChallengeStore) GetLiveChallengeForUser(userID string) (*Challenge, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return cloneChallenge(liveChallengeForUserLocked(s.challenges, userID)), nil
}

func (s *MemoryChallengeStore) ListWaitingPublicChallenges(excludeUserID string) ([]*Challenge, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var out []*Challenge
	for _, challenge := range s.challenges {
		if challenge.Source != challengeSourcePublicQueue || challenge.State != challengeStateWaiting || challenge.OpponentID != "" {
			continue
		}
		if challenge.CreatorID == excludeUserID {
			continue
		}
		out = append(out, cloneChallenge(challenge))
	}
	sortChallengesByCreatedAt(out)
	return out, nil
}

func (s *MemoryChallengeStore) ActivateHumanMatch(code, opponentID string, input ChallengeInput) (*Challenge, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	challenge, ok := s.challenges[normalizeChallengeCodeValue(code)]
	if !ok {
		return nil, ErrChallengeNotFound
	}
	if live := liveChallengeForUserLocked(s.challenges, opponentID); live != nil {
		return nil, ErrChallengeAlreadyActive
	}
	if challenge.CreatorID == opponentID {
		return nil, ErrChallengeSelfJoin
	}
	if challenge.OpponentID != "" {
		return nil, ErrChallengeFull
	}
	now := time.Now()
	challenge.OpponentID = opponentID
	challenge.OpponentType = challengeOpponentHuman
	challenge.TopicID = input.TopicID
	challenge.TopicName = input.TopicName
	challenge.SubjectID = input.SubjectID
	challenge.SyllabusID = input.SyllabusID
	challenge.Questions = trimChallengeQuestions(input.Questions, input.QuestionCount)
	challenge.QuestionCount = normalizeChallengeQuestionCount(input.QuestionCount, input.Questions)
	challenge.Metadata.OpponentForm = input.Metadata.CreatorForm
	challenge.State = challengeStateWaiting
	challenge.CreatorReadyAt = nil
	challenge.OpponentReadyAt = nil
	challenge.UpdatedAt = now
	return cloneChallenge(challenge), nil
}

func (s *MemoryChallengeStore) ActivateAIFallback(code string, input ChallengeInput) (*Challenge, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	challenge, ok := s.challenges[normalizeChallengeCodeValue(code)]
	if !ok {
		return nil, ErrChallengeNotFound
	}
	now := time.Now()
	challenge.Source = challengeSourceAIFallback
	challenge.OpponentType = challengeOpponentAI
	challenge.TopicID = input.TopicID
	challenge.TopicName = input.TopicName
	challenge.SubjectID = input.SubjectID
	challenge.SyllabusID = input.SyllabusID
	challenge.Questions = trimChallengeQuestions(input.Questions, input.QuestionCount)
	challenge.QuestionCount = normalizeChallengeQuestionCount(input.QuestionCount, input.Questions)
	challenge.Metadata.AIProfile = input.Metadata.AIProfile
	challenge.State = challengeStateReady
	challenge.CreatorReadyAt = ptrTime(challenge.CreatedAt)
	challenge.OpponentReadyAt = ptrTime(now)
	challenge.UpdatedAt = now
	return cloneChallenge(challenge), nil
}

func (s *MemoryChallengeStore) JoinPrivateChallenge(code, opponentID string) (*Challenge, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	challenge, ok := s.challenges[normalizeChallengeCodeValue(code)]
	if !ok {
		return nil, ErrChallengeNotFound
	}
	if live := liveChallengeForUserLocked(s.challenges, opponentID); live != nil {
		return nil, ErrChallengeAlreadyActive
	}
	if challenge.CreatorID == opponentID {
		return nil, ErrChallengeSelfJoin
	}
	if challenge.Source != challengeSourcePrivateCode {
		return nil, ErrChallengeNotFound
	}
	if challenge.OpponentID != "" {
		if challenge.OpponentID == opponentID {
			return cloneChallenge(challenge), nil
		}
		return nil, ErrChallengeFull
	}
	now := time.Now()
	challenge.OpponentID = opponentID
	challenge.OpponentType = challengeOpponentHuman
	challenge.OpponentReadyAt = nil
	challenge.State = challengeStateWaiting
	challenge.UpdatedAt = now
	return cloneChallenge(challenge), nil
}

func (s *MemoryChallengeStore) MarkReady(code, userID string) (*Challenge, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	challenge, ok := s.challenges[normalizeChallengeCodeValue(code)]
	if !ok {
		return nil, ErrChallengeNotFound
	}
	if !challengeHasUser(challenge, userID) {
		return nil, ErrChallengeNotFound
	}
	now := time.Now()
	if challenge.CreatorID == userID && challenge.CreatorReadyAt == nil {
		challenge.CreatorReadyAt = ptrTime(now)
	}
	if challenge.OpponentID == userID && challenge.OpponentReadyAt == nil {
		challenge.OpponentReadyAt = ptrTime(now)
	}
	if challengeReadyToOpen(challenge) && challenge.State != challengeStateCompleted {
		challenge.State = challengeStateReady
	}
	challenge.UpdatedAt = now
	return cloneChallenge(challenge), nil
}

func (s *MemoryChallengeStore) CancelPublicQueue(userID string) (bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for code, challenge := range s.challenges {
		if challenge.CreatorID != userID || challenge.Source != challengeSourcePublicQueue || challenge.State != challengeStateWaiting || challenge.OpponentID != "" {
			continue
		}
		delete(s.challenges, code)
		return true, nil
	}
	return false, nil
}

func (s *MemoryChallengeStore) CompleteChallenge(code, userID string, correctAnswers int) (*ChallengeCompletion, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	challenge, ok := s.challenges[normalizeChallengeCodeValue(code)]
	if !ok {
		return nil, ErrChallengeNotFound
	}
	return completeChallengeRecord(challenge, userID, correctAnswers), nil
}
