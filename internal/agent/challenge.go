package agent

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"math/big"
	"strings"
	"sync"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

const (
	ChallengeStateWaiting           = "waiting"
	ChallengeStatePendingAcceptance = "pending_acceptance"
	ChallengeStateReady             = "ready"

	ChallengeMatchSourceInviteCode = "invite_code"
	ChallengeMatchSourceQueue      = "queue"
	ChallengeMatchSourceAIFallback = "ai_fallback"
	ChallengeOpponentKindHuman     = "human"
	ChallengeOpponentKindAI        = "ai"
	defaultChallengeQuestionCount  = 5
	challengeCodeInsertMaxAttempts = 8
	matchmakingWaitTimeout         = 10 * time.Minute
	matchAcceptanceTimeout         = 2 * time.Minute

	MatchmakingStatusSearching = "searching"
	MatchmakingStatusMatched   = "matched"
	MatchmakingStatusCancelled = "cancelled"
	MatchmakingStatusExpired   = "expired"
)

var (
	ErrChallengeNotFound            = errors.New("challenge not found")
	ErrChallengeNotJoinable         = errors.New("challenge not joinable")
	ErrChallengeSelfJoin            = errors.New("challenge self join")
	ErrChallengeSearchAlreadyActive = errors.New("challenge search already active")
	ErrChallengeAlreadyActive       = errors.New("challenge already active")
	ErrChallengeAcceptNotAvailable  = errors.New("challenge accept not available")
)

// Challenge represents a peer challenge.
type Challenge struct {
	ID                 string
	Code               string
	CreatorID          string
	OpponentID         string
	TopicID            string
	TopicName          string
	SyllabusID         string
	QuestionCount      int
	State              string
	MatchSource        string
	OpponentKind       string
	CreatedAt          time.Time
	JoinDeadlineAt     *time.Time
	CreatorAcceptedAt  *time.Time
	OpponentAcceptedAt *time.Time
	ReadyAt            *time.Time
}

// ChallengeCreateInput captures challenge creation input.
type ChallengeCreateInput struct {
	TopicID       string
	TopicName     string
	SyllabusID    string
	QuestionCount int
}

// ChallengeSearch represents one user's active search for a human opponent.
type ChallengeSearch struct {
	ID                 string
	UserID             string
	TopicID            string
	TopicName          string
	SyllabusID         string
	QuestionCount      int
	Status             string
	MatchedChallengeID string
	ExpiresAt          time.Time
	CancelledAt        *time.Time
	CreatedAt          time.Time
}

// StartChallengeSearchResult returns either:
// - Search: the user is still waiting for an opponent
// - Challenge: an opponent was found and the match is ready
type StartChallengeSearchResult struct {
	Search    *ChallengeSearch
	Challenge *Challenge
}

// ChallengeStore persists invite-code challenges and matchmaking searches.
type ChallengeStore interface {
	CreateInviteChallenge(creatorID string, input ChallengeCreateInput) (*Challenge, error)
	JoinChallenge(code, opponentID string) (*Challenge, error)
	GetChallenge(code string) (*Challenge, error)
	StartChallengeSearch(userID string, input ChallengeCreateInput) (*StartChallengeSearchResult, error)
	CancelChallengeSearch(userID string) (bool, error)
	AcceptPendingChallenge(userID string) (*Challenge, error)
	DeclinePendingChallenge(userID string) (bool, error)
	CancelOpenChallenge(userID string) (bool, error)
}

// MemoryChallengeStore stores challenges in memory.
type MemoryChallengeStore struct {
	// mu guards challenges because Go maps are not safe for concurrent writes.
	mu             sync.RWMutex
	challenges     map[string]*Challenge
	challengesByID map[string]*Challenge
	searches       map[string]*ChallengeSearch
}

func NewMemoryChallengeStore() *MemoryChallengeStore {
	return &MemoryChallengeStore{
		challenges:     make(map[string]*Challenge),
		challengesByID: make(map[string]*Challenge),
		searches:       make(map[string]*ChallengeSearch),
	}
}

func (s *MemoryChallengeStore) CreateInviteChallenge(creatorID string, input ChallengeCreateInput) (*Challenge, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.prepareUserStateLocked(creatorID, time.Now())
	if s.userHasBlockingSearchLocked(creatorID) || s.userHasLiveChallengeLocked(creatorID) {
		return nil, ErrChallengeAlreadyActive
	}

	code := GenerateChallengeCode()
	for {
		if _, exists := s.challenges[code]; !exists {
			break
		}
		code = GenerateChallengeCode()
	}

	challenge := &Challenge{
		ID:            generateID(),
		Code:          code,
		CreatorID:     creatorID,
		TopicID:       strings.TrimSpace(input.TopicID),
		TopicName:     strings.TrimSpace(input.TopicName),
		SyllabusID:    strings.TrimSpace(input.SyllabusID),
		QuestionCount: normalizeChallengeQuestionCount(input.QuestionCount),
		State:         ChallengeStateWaiting,
		MatchSource:   ChallengeMatchSourceInviteCode,
		OpponentKind:  ChallengeOpponentKindHuman,
		CreatedAt:     time.Now(),
	}
	s.challenges[code] = challenge
	s.challengesByID[challenge.ID] = challenge
	return cloneChallenge(challenge), nil
}

func (s *MemoryChallengeStore) JoinChallenge(code, opponentID string) (*Challenge, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	challenge, ok := s.challenges[normalizeChallengeCode(code)]
	if !ok {
		return nil, ErrChallengeNotFound
	}
	if challenge.CreatorID == opponentID {
		return nil, ErrChallengeSelfJoin
	}
	s.prepareUserStateLocked(opponentID, time.Now())
	if s.userHasBlockingSearchLocked(opponentID) || s.userHasLiveChallengeLocked(opponentID) {
		return nil, ErrChallengeAlreadyActive
	}
	if challenge.State != ChallengeStateWaiting {
		return nil, ErrChallengeNotJoinable
	}

	now := time.Now()
	challenge.OpponentID = opponentID
	challenge.State = ChallengeStateReady
	challenge.ReadyAt = &now
	return cloneChallenge(challenge), nil
}

func (s *MemoryChallengeStore) GetChallenge(code string) (*Challenge, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	challenge, ok := s.challenges[normalizeChallengeCode(code)]
	if !ok {
		return nil, ErrChallengeNotFound
	}
	if !challengeResumableAt(challenge, time.Now()) {
		return nil, ErrChallengeNotFound
	}
	return cloneChallenge(challenge), nil
}

func (s *MemoryChallengeStore) StartChallengeSearch(userID string, input ChallengeCreateInput) (*StartChallengeSearchResult, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now()
	if result := s.claimTimedOutSearchAsAIFallbackLocked(userID, now); result != nil {
		return result, nil
	}

	s.prepareUserStateLocked(userID, now)

	if search, ok := s.searches[userID]; ok {
		switch search.Status {
		case MatchmakingStatusSearching:
			if search.TopicID != strings.TrimSpace(input.TopicID) {
				return &StartChallengeSearchResult{Search: cloneChallengeSearch(search)}, ErrChallengeSearchAlreadyActive
			}
			if result := s.tryPairLocked(search, input.QuestionCount); result != nil {
				return result, nil
			}
			return &StartChallengeSearchResult{Search: cloneChallengeSearch(search)}, nil
		case MatchmakingStatusMatched:
			if challenge, ok := s.challengesByID[search.MatchedChallengeID]; ok {
				return &StartChallengeSearchResult{
					Search:    cloneChallengeSearch(search),
					Challenge: cloneChallenge(challenge),
				}, nil
			}
		}
	}
	if s.userHasLiveChallengeLocked(userID) {
		return nil, ErrChallengeAlreadyActive
	}

	search := &ChallengeSearch{
		ID:            generateID(),
		UserID:        userID,
		TopicID:       strings.TrimSpace(input.TopicID),
		TopicName:     strings.TrimSpace(input.TopicName),
		SyllabusID:    strings.TrimSpace(input.SyllabusID),
		QuestionCount: normalizeChallengeQuestionCount(input.QuestionCount),
		Status:        MatchmakingStatusSearching,
		ExpiresAt:     time.Now().Add(matchmakingWaitTimeout),
		CreatedAt:     time.Now(),
	}
	s.searches[userID] = search

	if result := s.tryPairLocked(search, input.QuestionCount); result != nil {
		return result, nil
	}
	return &StartChallengeSearchResult{Search: cloneChallengeSearch(search)}, nil
}

func (s *MemoryChallengeStore) CancelChallengeSearch(userID string) (bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	search, ok := s.searches[userID]
	if !ok || search.Status != MatchmakingStatusSearching {
		return false, nil
	}
	now := time.Now()
	search.Status = MatchmakingStatusCancelled
	search.CancelledAt = &now
	return true, nil
}

func (s *MemoryChallengeStore) tryPairLocked(search *ChallengeSearch, questionCount int) *StartChallengeSearchResult {
	opponentSearch := s.findCompatibleSearchLocked(search.UserID, search.TopicID)
	if opponentSearch == nil {
		return nil
	}

	now := time.Now()
	creatorSearch, opponentSearchForChallenge := opponentSearch, search
	challenge := &Challenge{
		ID:             generateID(),
		CreatorID:      creatorSearch.UserID,
		OpponentID:     opponentSearchForChallenge.UserID,
		TopicID:        creatorSearch.TopicID,
		TopicName:      creatorSearch.TopicName,
		SyllabusID:     creatorSearch.SyllabusID,
		QuestionCount:  normalizeChallengeQuestionCount(creatorSearch.QuestionCount),
		State:          ChallengeStatePendingAcceptance,
		MatchSource:    ChallengeMatchSourceQueue,
		OpponentKind:   ChallengeOpponentKindHuman,
		CreatedAt:      now,
		JoinDeadlineAt: ptrTime(now.Add(matchAcceptanceTimeout)),
	}
	s.challengesByID[challenge.ID] = challenge
	search.Status = MatchmakingStatusMatched
	opponentSearch.Status = MatchmakingStatusMatched
	search.MatchedChallengeID = challenge.ID
	opponentSearch.MatchedChallengeID = challenge.ID

	return &StartChallengeSearchResult{
		Search:    cloneChallengeSearch(search),
		Challenge: cloneChallenge(challenge),
	}
}

func (s *MemoryChallengeStore) claimTimedOutSearchAsAIFallbackLocked(userID string, now time.Time) *StartChallengeSearchResult {
	search, ok := s.searches[userID]
	if !ok || search.Status != MatchmakingStatusSearching || search.ExpiresAt.After(now) {
		return nil
	}

	challenge := &Challenge{
		ID:            generateID(),
		CreatorID:     userID,
		TopicID:       search.TopicID,
		TopicName:     search.TopicName,
		SyllabusID:    search.SyllabusID,
		QuestionCount: normalizeChallengeQuestionCount(search.QuestionCount),
		State:         ChallengeStateReady,
		MatchSource:   ChallengeMatchSourceAIFallback,
		OpponentKind:  ChallengeOpponentKindAI,
		CreatedAt:     now,
		ReadyAt:       ptrTime(now),
	}
	s.challengesByID[challenge.ID] = challenge
	search.Status = MatchmakingStatusMatched
	search.MatchedChallengeID = challenge.ID

	return &StartChallengeSearchResult{
		Search:    cloneChallengeSearch(search),
		Challenge: cloneChallenge(challenge),
	}
}

func (s *MemoryChallengeStore) findCompatibleSearchLocked(userID, topicID string) *ChallengeSearch {
	var candidate *ChallengeSearch
	now := time.Now()
	for _, search := range s.searches {
		if search.UserID == userID || search.TopicID != topicID {
			continue
		}
		if search.Status != MatchmakingStatusSearching {
			continue
		}
		if !search.ExpiresAt.After(now) {
			search.Status = MatchmakingStatusExpired
			continue
		}
		if candidate == nil || search.CreatedAt.Before(candidate.CreatedAt) {
			candidate = search
		}
	}
	return candidate
}

func (s *MemoryChallengeStore) prepareUserStateLocked(userID string, now time.Time) {
	search, ok := s.searches[userID]
	if !ok {
		return
	}
	switch search.Status {
	case MatchmakingStatusSearching:
		if !search.ExpiresAt.After(now) {
			search.Status = MatchmakingStatusExpired
		}
	case MatchmakingStatusMatched:
		challenge, ok := s.challengesByID[search.MatchedChallengeID]
		if !ok || !challengeResumableAt(challenge, now) {
			if ok && challenge.State == ChallengeStatePendingAcceptance {
				challenge.State = "expired"
			}
			search.Status = MatchmakingStatusExpired
		}
	}
}

func (s *MemoryChallengeStore) userHasBlockingSearchLocked(userID string) bool {
	search, ok := s.searches[userID]
	if !ok {
		return false
	}
	switch search.Status {
	case MatchmakingStatusSearching:
		return true
	case MatchmakingStatusMatched:
		challenge, ok := s.challengesByID[search.MatchedChallengeID]
		return ok && challengeResumableAt(challenge, time.Now())
	default:
		return false
	}
}

func (s *MemoryChallengeStore) userHasLiveChallengeLocked(userID string) bool {
	now := time.Now()
	for _, challenge := range s.challengesByID {
		if !challengeResumableAt(challenge, now) {
			continue
		}
		if challenge.CreatorID == userID || challenge.OpponentID == userID {
			return true
		}
	}
	return false
}

func (s *MemoryChallengeStore) AcceptPendingChallenge(userID string) (*Challenge, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	challenge := s.findQueueChallengeForUserLocked(userID)
	if challenge == nil {
		return nil, ErrChallengeAcceptNotAvailable
	}
	if challenge.State == ChallengeStateReady {
		return cloneChallenge(challenge), nil
	}
	if challenge.State != ChallengeStatePendingAcceptance {
		return nil, ErrChallengeAcceptNotAvailable
	}
	now := time.Now()
	if challenge.JoinDeadlineAt != nil && !challenge.JoinDeadlineAt.After(now) {
		challenge.State = "expired"
		s.expireMatchedSearchesForChallengeLocked(challenge.ID)
		return nil, ErrChallengeAcceptNotAvailable
	}
	if challenge.CreatorID == userID && challenge.CreatorAcceptedAt == nil {
		challenge.CreatorAcceptedAt = ptrTime(now)
	}
	if challenge.OpponentID == userID && challenge.OpponentAcceptedAt == nil {
		challenge.OpponentAcceptedAt = ptrTime(now)
	}
	if challenge.CreatorAcceptedAt != nil && challenge.OpponentAcceptedAt != nil {
		challenge.State = ChallengeStateReady
		challenge.ReadyAt = ptrTime(now)
		challenge.JoinDeadlineAt = nil
	}
	return cloneChallenge(challenge), nil
}

func (s *MemoryChallengeStore) DeclinePendingChallenge(userID string) (bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	challenge := s.findQueueChallengeForUserLocked(userID)
	if challenge == nil || challenge.State != ChallengeStatePendingAcceptance {
		return false, nil
	}
	challenge.State = "cancelled"
	challenge.JoinDeadlineAt = nil
	s.cancelMatchedSearchesForChallengeLocked(challenge.ID)
	return true, nil
}

func (s *MemoryChallengeStore) CancelOpenChallenge(userID string) (bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	challenge := s.findOpenChallengeForUserLocked(userID)
	if challenge == nil {
		return false, nil
	}
	challenge.State = "cancelled"
	challenge.JoinDeadlineAt = nil
	s.cancelMatchedSearchesForChallengeLocked(challenge.ID)
	return true, nil
}

func (s *MemoryChallengeStore) findQueueChallengeForUserLocked(userID string) *Challenge {
	now := time.Now()
	for _, challenge := range s.challengesByID {
		if challenge.MatchSource != ChallengeMatchSourceQueue {
			continue
		}
		if challenge.CreatorID != userID && challenge.OpponentID != userID {
			continue
		}
		if !challengeResumableAt(challenge, now) {
			continue
		}
		return challenge
	}
	return nil
}

func (s *MemoryChallengeStore) findOpenChallengeForUserLocked(userID string) *Challenge {
	now := time.Now()
	for _, challenge := range s.challengesByID {
		if challenge.CreatorID != userID && challenge.OpponentID != userID {
			continue
		}
		if !challengeResumableAt(challenge, now) {
			continue
		}
		if challenge.State == ChallengeStatePendingAcceptance {
			continue
		}
		return challenge
	}
	return nil
}

func (s *MemoryChallengeStore) expireMatchedSearchesForChallengeLocked(challengeID string) {
	for _, search := range s.searches {
		if search.MatchedChallengeID == challengeID && search.Status == MatchmakingStatusMatched {
			search.Status = MatchmakingStatusExpired
		}
	}
}

func (s *MemoryChallengeStore) cancelMatchedSearchesForChallengeLocked(challengeID string) {
	now := time.Now()
	for _, search := range s.searches {
		if search.MatchedChallengeID == challengeID && search.Status == MatchmakingStatusMatched {
			search.Status = MatchmakingStatusCancelled
			search.CancelledAt = &now
		}
	}
}

// PostgresChallengeStore persists challenges in PostgreSQL.
type PostgresChallengeStore struct {
	pool     *pgxpool.Pool
	tenantID string
	channel  string
}

func NewPostgresChallengeStore(pool *pgxpool.Pool, tenantID string) *PostgresChallengeStore {
	return NewPostgresChallengeStoreForChannel(pool, tenantID, defaultChannel)
}

func NewPostgresChallengeStoreForChannel(pool *pgxpool.Pool, tenantID, channel string) *PostgresChallengeStore {
	channel = strings.TrimSpace(channel)
	if channel == "" {
		channel = defaultChannel
	}
	return &PostgresChallengeStore{
		pool:     pool,
		tenantID: tenantID,
		channel:  channel,
	}
}

func (s *PostgresChallengeStore) CreateInviteChallenge(externalCreatorID string, input ChallengeCreateInput) (*Challenge, error) {
	ctx, cancel := context.WithTimeout(context.Background(), dbTimeout)
	defer cancel()

	creatorID, err := s.resolveUserUUID(ctx, externalCreatorID)
	if err != nil {
		return nil, err
	}

	for attempt := 0; attempt < challengeCodeInsertMaxAttempts; attempt++ {
		tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{})
		if err != nil {
			return nil, fmt.Errorf("begin create challenge tx: %w", err)
		}
		defer func() { _ = tx.Rollback(ctx) }()

		if err := s.lockUser(ctx, tx, creatorID); err != nil {
			return nil, err
		}
		if err := s.cleanupExpiredSearchingSearchesForUser(ctx, tx, creatorID); err != nil {
			return nil, err
		}
		if err := s.cleanupStaleMatchedSearchesForUser(ctx, tx, creatorID); err != nil {
			return nil, err
		}
		if err := s.ensureUserCanOpenChallenge(ctx, tx, creatorID); err != nil {
			return nil, err
		}

		code := GenerateChallengeCode()
		challenge := &Challenge{
			Code:          code,
			CreatorID:     externalCreatorID,
			TopicID:       strings.TrimSpace(input.TopicID),
			TopicName:     strings.TrimSpace(input.TopicName),
			SyllabusID:    strings.TrimSpace(input.SyllabusID),
			QuestionCount: normalizeChallengeQuestionCount(input.QuestionCount),
			State:         ChallengeStateWaiting,
			MatchSource:   ChallengeMatchSourceInviteCode,
			OpponentKind:  ChallengeOpponentKindHuman,
		}

		err = tx.QueryRow(ctx,
			`INSERT INTO challenges (
				tenant_id,
				creator_user_id,
				topic_id,
				topic_name,
				syllabus_id,
				match_source,
				opponent_kind,
				invite_code,
				question_count,
				state
			) VALUES ($1::uuid, $2::uuid, $3, $4, $5, $6, $7, $8, $9, $10)
			RETURNING id::text, created_at, ready_at`,
			s.tenantID,
			creatorID,
			challenge.TopicID,
			challenge.TopicName,
			challenge.SyllabusID,
			challenge.MatchSource,
			challenge.OpponentKind,
			challenge.Code,
			challenge.QuestionCount,
			challenge.State,
		).Scan(&challenge.ID, &challenge.CreatedAt, &challenge.ReadyAt)
		if err == nil {
			if err := tx.Commit(ctx); err != nil {
				return nil, fmt.Errorf("commit create challenge: %w", err)
			}
			return challenge, nil
		}
		if isUniqueViolation(err) {
			_ = tx.Rollback(ctx)
			continue
		}
		return nil, fmt.Errorf("insert challenge: %w", err)
	}

	return nil, fmt.Errorf("generate unique challenge code: exhausted retries")
}

func (s *PostgresChallengeStore) JoinChallenge(code, externalOpponentID string) (*Challenge, error) {
	ctx, cancel := context.WithTimeout(context.Background(), dbTimeout)
	defer cancel()

	opponentUUID, err := s.resolveUserUUID(ctx, externalOpponentID)
	if err != nil {
		return nil, err
	}

	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return nil, fmt.Errorf("begin join challenge tx: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	current, creatorUUID, err := s.getChallengeForUpdate(ctx, tx, normalizeChallengeCode(code))
	if err != nil {
		return nil, err
	}
	if creatorUUID == opponentUUID {
		return nil, ErrChallengeSelfJoin
	}
	if err := s.lockUser(ctx, tx, opponentUUID); err != nil {
		return nil, err
	}
	if err := s.cleanupExpiredSearchingSearchesForUser(ctx, tx, opponentUUID); err != nil {
		return nil, err
	}
	if err := s.cleanupStaleMatchedSearchesForUser(ctx, tx, opponentUUID); err != nil {
		return nil, err
	}
	if err := s.ensureUserCanOpenChallenge(ctx, tx, opponentUUID); err != nil {
		return nil, err
	}
	if current.State != ChallengeStateWaiting {
		return nil, ErrChallengeNotJoinable
	}

	var readyAt *time.Time
	err = tx.QueryRow(ctx,
		`UPDATE challenges
		 SET opponent_user_id = $2::uuid,
		     state = $3,
		     ready_at = NOW(),
		     updated_at = NOW()
		 WHERE id = $1::uuid
		   AND state = $4
		 RETURNING ready_at`,
		current.ID,
		opponentUUID,
		ChallengeStateReady,
		ChallengeStateWaiting,
	).Scan(&readyAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrChallengeNotJoinable
		}
		return nil, fmt.Errorf("update challenge join: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit join challenge: %w", err)
	}

	current.OpponentID = externalOpponentID
	current.State = ChallengeStateReady
	current.ReadyAt = readyAt
	return current, nil
}

func (s *PostgresChallengeStore) GetChallenge(code string) (*Challenge, error) {
	ctx, cancel := context.WithTimeout(context.Background(), dbTimeout)
	defer cancel()

	return s.getChallenge(ctx, s.pool, normalizeChallengeCode(code))
}

func (s *PostgresChallengeStore) StartChallengeSearch(externalUserID string, input ChallengeCreateInput) (*StartChallengeSearchResult, error) {
	ctx, cancel := context.WithTimeout(context.Background(), dbTimeout)
	defer cancel()

	userUUID, err := s.resolveUserUUID(ctx, externalUserID)
	if err != nil {
		return nil, err
	}

	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return nil, fmt.Errorf("begin start matchmaking tx: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	if err := s.lockUser(ctx, tx, userUUID); err != nil {
		return nil, err
	}
	if err := s.cleanupStaleMatchedSearchesForUser(ctx, tx, userUUID); err != nil {
		return nil, err
	}
	if result, err := s.claimTimedOutSearchAsAIFallback(ctx, tx, userUUID, externalUserID); err != nil {
		return nil, err
	} else if result != nil {
		if err := tx.Commit(ctx); err != nil {
			return nil, fmt.Errorf("commit start matchmaking: %w", err)
		}
		return result, nil
	}
	if err := s.cleanupExpiredSearchingSearchesForUser(ctx, tx, userUUID); err != nil {
		return nil, err
	}

	search, err := s.getSearchingSearchForUser(ctx, tx, userUUID, externalUserID)
	if err != nil {
		return nil, err
	}
	if search != nil {
		if search.TopicID != strings.TrimSpace(input.TopicID) {
			return &StartChallengeSearchResult{Search: search}, ErrChallengeSearchAlreadyActive
		}
	} else {
		matchedSearch, challenge, err := s.getMatchedSearchForUser(ctx, tx, userUUID, externalUserID)
		if err != nil {
			return nil, err
		}
		if matchedSearch != nil {
			if err := tx.Commit(ctx); err != nil {
				return nil, fmt.Errorf("commit start matchmaking: %w", err)
			}
			return &StartChallengeSearchResult{
				Search:    matchedSearch,
				Challenge: challenge,
			}, nil
		}
		if err := s.ensureUserCanOpenChallenge(ctx, tx, userUUID); err != nil {
			return nil, err
		}

		search, err = s.insertChallengeSearch(ctx, tx, userUUID, externalUserID, input)
		if err != nil {
			return nil, err
		}
		if search.TopicID != strings.TrimSpace(input.TopicID) {
			return &StartChallengeSearchResult{Search: search}, ErrChallengeSearchAlreadyActive
		}
	}

	opponentSearch, matchUUID, err := s.findCompatibleSearch(ctx, tx, userUUID, search.TopicID)
	if err != nil {
		return nil, err
	}
	if opponentSearch == nil {
		if err := tx.Commit(ctx); err != nil {
			return nil, fmt.Errorf("commit start matchmaking: %w", err)
		}
		return &StartChallengeSearchResult{Search: search}, nil
	}

	creatorUUID, creatorExternalID := userUUID, externalUserID
	opponentUUID, opponentExternalID := matchUUID, opponentSearch.UserID
	if opponentSearch.CreatedAt.Before(search.CreatedAt) {
		creatorUUID, creatorExternalID = matchUUID, opponentSearch.UserID
		opponentUUID, opponentExternalID = userUUID, externalUserID
	}
	challenge := &Challenge{
		CreatorID:     creatorExternalID,
		OpponentID:    opponentExternalID,
		TopicID:       search.TopicID,
		TopicName:     search.TopicName,
		SyllabusID:    search.SyllabusID,
		QuestionCount: normalizeChallengeQuestionCount(input.QuestionCount),
		State:         ChallengeStatePendingAcceptance,
		MatchSource:   ChallengeMatchSourceQueue,
		OpponentKind:  ChallengeOpponentKindHuman,
	}

	err = tx.QueryRow(ctx,
		`INSERT INTO challenges (
			tenant_id,
			creator_user_id,
			opponent_user_id,
			topic_id,
			topic_name,
			syllabus_id,
			match_source,
			opponent_kind,
			question_count,
			state,
			join_deadline_at
		) VALUES ($1::uuid, $2::uuid, $3::uuid, $4, $5, $6, $7, $8, $9, $10, NOW() + $11 * INTERVAL '1 second')
		RETURNING id::text, created_at, join_deadline_at`,
		s.tenantID,
		creatorUUID,
		opponentUUID,
		challenge.TopicID,
		challenge.TopicName,
		challenge.SyllabusID,
		challenge.MatchSource,
		challenge.OpponentKind,
		challenge.QuestionCount,
		challenge.State,
		int(matchAcceptanceTimeout/time.Second),
	).Scan(&challenge.ID, &challenge.CreatedAt, &challenge.JoinDeadlineAt)
	if err != nil {
		return nil, fmt.Errorf("insert queue challenge: %w", err)
	}

	_, err = tx.Exec(ctx,
		`UPDATE challenge_matchmaking_tickets
		 SET status = $2,
		     matched_challenge_id = $3::uuid,
		     updated_at = NOW()
		 WHERE id = ANY($1::uuid[])`,
		[]string{search.ID, opponentSearch.ID},
		MatchmakingStatusMatched,
		challenge.ID,
	)
	if err != nil {
		return nil, fmt.Errorf("mark tickets matched: %w", err)
	}

	search.Status = MatchmakingStatusMatched
	search.MatchedChallengeID = challenge.ID
	opponentSearch.Status = MatchmakingStatusMatched
	opponentSearch.MatchedChallengeID = challenge.ID

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit start matchmaking: %w", err)
	}
	return &StartChallengeSearchResult{
		Search:    search,
		Challenge: challenge,
	}, nil
}

func (s *PostgresChallengeStore) CancelChallengeSearch(externalUserID string) (bool, error) {
	ctx, cancel := context.WithTimeout(context.Background(), dbTimeout)
	defer cancel()

	userUUID, err := s.resolveUserUUID(ctx, externalUserID)
	if err != nil {
		return false, err
	}

	cmd, err := s.pool.Exec(ctx,
		`UPDATE challenge_matchmaking_tickets
		 SET status = $3,
		     cancelled_at = NOW(),
		     updated_at = NOW()
		 WHERE tenant_id = $1::uuid
		   AND user_id = $2::uuid
		   AND status = $4`,
		s.tenantID,
		userUUID,
		MatchmakingStatusCancelled,
		MatchmakingStatusSearching,
	)
	if err != nil {
		return false, fmt.Errorf("cancel matchmaking: %w", err)
	}
	return cmd.RowsAffected() > 0, nil
}

func (s *PostgresChallengeStore) AcceptPendingChallenge(externalUserID string) (*Challenge, error) {
	ctx, cancel := context.WithTimeout(context.Background(), dbTimeout)
	defer cancel()

	userUUID, err := s.resolveUserUUID(ctx, externalUserID)
	if err != nil {
		return nil, err
	}

	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return nil, fmt.Errorf("begin accept challenge tx: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	if err := s.lockUser(ctx, tx, userUUID); err != nil {
		return nil, err
	}
	if err := s.cleanupExpiredSearchingSearchesForUser(ctx, tx, userUUID); err != nil {
		return nil, err
	}
	if err := s.cleanupStaleMatchedSearchesForUser(ctx, tx, userUUID); err != nil {
		return nil, err
	}

	challenge, role, err := s.getPendingQueueChallengeForUserForUpdate(ctx, tx, userUUID)
	if err != nil {
		return nil, err
	}
	if challenge == nil {
		challenge, err = s.getReadyQueueChallengeForUser(ctx, tx, userUUID)
		if err != nil {
			return nil, err
		}
		if challenge == nil {
			return nil, ErrChallengeAcceptNotAvailable
		}
		if err := tx.Commit(ctx); err != nil {
			return nil, fmt.Errorf("commit accept challenge: %w", err)
		}
		return challenge, nil
	}

	now := time.Now()
	if challenge.JoinDeadlineAt != nil && !challenge.JoinDeadlineAt.After(now) {
		if err := s.expirePendingQueueChallenge(ctx, tx, challenge.ID); err != nil {
			return nil, err
		}
		if err := tx.Commit(ctx); err != nil {
			return nil, fmt.Errorf("commit expired acceptance: %w", err)
		}
		return nil, ErrChallengeAcceptNotAvailable
	}

	var query string
	switch role {
	case "creator":
		query = `UPDATE challenges
		            SET creator_accepted_at = COALESCE(creator_accepted_at, NOW()),
		                updated_at = NOW()
		          WHERE id = $1::uuid`
	case "opponent":
		query = `UPDATE challenges
		            SET opponent_accepted_at = COALESCE(opponent_accepted_at, NOW()),
		                updated_at = NOW()
		          WHERE id = $1::uuid`
	default:
		return nil, ErrChallengeAcceptNotAvailable
	}
	if _, err := tx.Exec(ctx, query, challenge.ID); err != nil {
		return nil, fmt.Errorf("mark challenge accepted: %w", err)
	}

	challenge, err = s.getChallengeByID(ctx, tx, challenge.ID)
	if err != nil {
		return nil, err
	}
	if challenge.CreatorAcceptedAt != nil && challenge.OpponentAcceptedAt != nil && challenge.State == ChallengeStatePendingAcceptance {
		var readyAt time.Time
		err = tx.QueryRow(ctx,
			`UPDATE challenges
			    SET state = $2,
			        ready_at = NOW(),
			        join_deadline_at = NULL,
			        updated_at = NOW()
			  WHERE id = $1::uuid
			    AND state = $3
			RETURNING ready_at`,
			challenge.ID,
			ChallengeStateReady,
			ChallengeStatePendingAcceptance,
		).Scan(&readyAt)
		if err != nil {
			return nil, fmt.Errorf("mark challenge ready: %w", err)
		}
		challenge.State = ChallengeStateReady
		challenge.ReadyAt = &readyAt
		challenge.JoinDeadlineAt = nil
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit accept challenge: %w", err)
	}
	return challenge, nil
}

func (s *PostgresChallengeStore) DeclinePendingChallenge(externalUserID string) (bool, error) {
	ctx, cancel := context.WithTimeout(context.Background(), dbTimeout)
	defer cancel()

	userUUID, err := s.resolveUserUUID(ctx, externalUserID)
	if err != nil {
		return false, err
	}

	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return false, fmt.Errorf("begin decline challenge tx: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	if err := s.lockUser(ctx, tx, userUUID); err != nil {
		return false, err
	}
	challenge, _, err := s.getPendingQueueChallengeForUserForUpdate(ctx, tx, userUUID)
	if err != nil {
		return false, err
	}
	if challenge == nil {
		return false, nil
	}

	if _, err := tx.Exec(ctx,
		`UPDATE challenges
		    SET state = 'cancelled',
		        join_deadline_at = NULL,
		        updated_at = NOW()
		  WHERE id = $1::uuid`,
		challenge.ID,
	); err != nil {
		return false, fmt.Errorf("cancel pending challenge: %w", err)
	}
	if _, err := tx.Exec(ctx,
		`UPDATE challenge_matchmaking_tickets
		    SET status = $2,
		        cancelled_at = NOW(),
		        updated_at = NOW()
		  WHERE tenant_id = $1::uuid
		    AND matched_challenge_id = $3::uuid
		    AND status = $4`,
		s.tenantID,
		MatchmakingStatusCancelled,
		challenge.ID,
		MatchmakingStatusMatched,
	); err != nil {
		return false, fmt.Errorf("cancel pending challenge tickets: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return false, fmt.Errorf("commit decline challenge: %w", err)
	}
	return true, nil
}

func (s *PostgresChallengeStore) CancelOpenChallenge(externalUserID string) (bool, error) {
	ctx, cancel := context.WithTimeout(context.Background(), dbTimeout)
	defer cancel()

	userUUID, err := s.resolveUserUUID(ctx, externalUserID)
	if err != nil {
		return false, err
	}

	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return false, fmt.Errorf("begin cancel open challenge tx: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	if err := s.lockUser(ctx, tx, userUUID); err != nil {
		return false, err
	}

	var challengeID string
	err = tx.QueryRow(ctx,
		`SELECT id::text
		   FROM challenges
		  WHERE tenant_id = $1::uuid
		    AND state IN ('waiting', 'ready', 'active')
		    AND (creator_user_id = $2::uuid OR opponent_user_id = $2::uuid)
		  ORDER BY created_at DESC
		  LIMIT 1
		  FOR UPDATE`,
		s.tenantID,
		userUUID,
	).Scan(&challengeID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return false, nil
		}
		return false, fmt.Errorf("get open challenge: %w", err)
	}

	if _, err := tx.Exec(ctx,
		`UPDATE challenges
		    SET state = 'cancelled',
		        join_deadline_at = NULL,
		        updated_at = NOW()
		  WHERE id = $1::uuid`,
		challengeID,
	); err != nil {
		return false, fmt.Errorf("cancel open challenge: %w", err)
	}
	if _, err := tx.Exec(ctx,
		`UPDATE challenge_matchmaking_tickets
		    SET status = $2,
		        cancelled_at = NOW(),
		        updated_at = NOW()
		  WHERE tenant_id = $1::uuid
		    AND matched_challenge_id = $3::uuid
		    AND status = $4`,
		s.tenantID,
		MatchmakingStatusCancelled,
		challengeID,
		MatchmakingStatusMatched,
	); err != nil {
		return false, fmt.Errorf("cancel open challenge tickets: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return false, fmt.Errorf("commit cancel open challenge: %w", err)
	}
	return true, nil
}

func (s *PostgresChallengeStore) getChallengeForUpdate(ctx context.Context, tx pgx.Tx, code string) (*Challenge, string, error) {
	var challenge Challenge
	var readyAt *time.Time
	var joinDeadlineAt *time.Time
	var creatorAcceptedAt *time.Time
	var opponentAcceptedAt *time.Time
	var creatorExternalID string
	var opponentExternalID *string
	var creatorUUID string

	err := tx.QueryRow(ctx,
		`SELECT c.id::text,
		        c.invite_code,
		        creator.external_id,
		        opponent.external_id,
		        c.topic_id,
		        c.topic_name,
		        c.syllabus_id,
		        c.question_count,
		        c.state,
		        c.match_source,
		        c.opponent_kind,
		        c.created_at,
		        c.join_deadline_at,
		        c.creator_accepted_at,
		        c.opponent_accepted_at,
		        c.ready_at,
		        c.creator_user_id::text
		   FROM challenges c
		   JOIN users creator ON creator.id = c.creator_user_id
		   LEFT JOIN users opponent ON opponent.id = c.opponent_user_id
		  WHERE c.tenant_id = $1::uuid
		    AND c.invite_code = $2
		    AND creator.channel = $3
		    AND c.state IN ('waiting', 'pending_acceptance', 'ready', 'active')
		  FOR UPDATE OF c`,
		s.tenantID,
		code,
		s.channel,
	).Scan(
		&challenge.ID,
		&challenge.Code,
		&creatorExternalID,
		&opponentExternalID,
		&challenge.TopicID,
		&challenge.TopicName,
		&challenge.SyllabusID,
		&challenge.QuestionCount,
		&challenge.State,
		&challenge.MatchSource,
		&challenge.OpponentKind,
		&challenge.CreatedAt,
		&joinDeadlineAt,
		&creatorAcceptedAt,
		&opponentAcceptedAt,
		&readyAt,
		&creatorUUID,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, "", ErrChallengeNotFound
		}
		return nil, "", fmt.Errorf("get challenge for update: %w", err)
	}
	challenge.CreatorID = creatorExternalID
	if opponentExternalID != nil {
		challenge.OpponentID = *opponentExternalID
	}
	challenge.JoinDeadlineAt = joinDeadlineAt
	challenge.CreatorAcceptedAt = creatorAcceptedAt
	challenge.OpponentAcceptedAt = opponentAcceptedAt
	challenge.ReadyAt = readyAt
	return &challenge, creatorUUID, nil
}

func (s *PostgresChallengeStore) getChallenge(ctx context.Context, querier interface {
	QueryRow(context.Context, string, ...any) pgx.Row
}, code string) (*Challenge, error) {
	var challenge Challenge
	var creatorID string
	var opponentID *string
	var joinDeadlineAt *time.Time
	var creatorAcceptedAt *time.Time
	var opponentAcceptedAt *time.Time
	err := querier.QueryRow(ctx,
		`SELECT c.id::text,
		        c.invite_code,
		        creator.external_id,
		        opponent.external_id,
		        c.topic_id,
		        c.topic_name,
		        c.syllabus_id,
		        c.question_count,
		        c.state,
		        c.match_source,
		        c.opponent_kind,
		        c.created_at,
		        c.join_deadline_at,
		        c.creator_accepted_at,
		        c.opponent_accepted_at,
		        c.ready_at
		   FROM challenges c
		   JOIN users creator ON creator.id = c.creator_user_id
		   LEFT JOIN users opponent ON opponent.id = c.opponent_user_id
		  WHERE c.tenant_id = $1::uuid
		    AND c.invite_code = $2
		    AND creator.channel = $3
		    AND c.state IN ('waiting', 'pending_acceptance', 'ready', 'active')`,
		s.tenantID,
		code,
		s.channel,
	).Scan(
		&challenge.ID,
		&challenge.Code,
		&creatorID,
		&opponentID,
		&challenge.TopicID,
		&challenge.TopicName,
		&challenge.SyllabusID,
		&challenge.QuestionCount,
		&challenge.State,
		&challenge.MatchSource,
		&challenge.OpponentKind,
		&challenge.CreatedAt,
		&joinDeadlineAt,
		&creatorAcceptedAt,
		&opponentAcceptedAt,
		&challenge.ReadyAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrChallengeNotFound
		}
		return nil, fmt.Errorf("get challenge: %w", err)
	}
	challenge.CreatorID = creatorID
	if opponentID != nil {
		challenge.OpponentID = *opponentID
	}
	challenge.JoinDeadlineAt = joinDeadlineAt
	challenge.CreatorAcceptedAt = creatorAcceptedAt
	challenge.OpponentAcceptedAt = opponentAcceptedAt
	return &challenge, nil
}

func (s *PostgresChallengeStore) resolveUserUUID(ctx context.Context, externalID string) (string, error) {
	var userID string
	err := s.pool.QueryRow(ctx,
		`SELECT id::text
		   FROM users
		  WHERE tenant_id = $1::uuid
		    AND channel = $2
		    AND external_id = $3
		  ORDER BY created_at ASC
		  LIMIT 1`,
		s.tenantID,
		s.channel,
		externalID,
	).Scan(&userID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return "", fmt.Errorf("user not found: %s", externalID)
		}
		return "", fmt.Errorf("resolve challenge user: %w", err)
	}
	return userID, nil
}

func (s *PostgresChallengeStore) getSearchingSearchForUser(ctx context.Context, tx pgx.Tx, userUUID, externalUserID string) (*ChallengeSearch, error) {
	var search ChallengeSearch
	err := tx.QueryRow(ctx,
		`SELECT id::text,
		        topic_id,
		        topic_name,
		        syllabus_id,
		        question_count,
		        status,
		        COALESCE(matched_challenge_id::text, ''),
		        expires_at,
		        cancelled_at,
		        created_at
		   FROM challenge_matchmaking_tickets
		  WHERE tenant_id = $1::uuid
		    AND user_id = $2::uuid
		    AND status = $3
		    AND expires_at > NOW()
		  ORDER BY created_at ASC
		  LIMIT 1
		  FOR UPDATE`,
		s.tenantID,
		userUUID,
		MatchmakingStatusSearching,
	).Scan(
		&search.ID,
		&search.TopicID,
		&search.TopicName,
		&search.SyllabusID,
		&search.QuestionCount,
		&search.Status,
		&search.MatchedChallengeID,
		&search.ExpiresAt,
		&search.CancelledAt,
		&search.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("get searching ticket: %w", err)
	}
	search.UserID = externalUserID
	return &search, nil
}

func (s *PostgresChallengeStore) getTimedOutSearchingSearchForUser(ctx context.Context, tx pgx.Tx, userUUID, externalUserID string) (*ChallengeSearch, error) {
	var search ChallengeSearch
	err := tx.QueryRow(ctx,
		`SELECT id::text,
		        topic_id,
		        topic_name,
		        syllabus_id,
		        question_count,
		        status,
		        COALESCE(matched_challenge_id::text, ''),
		        expires_at,
		        cancelled_at,
		        created_at
		   FROM challenge_matchmaking_tickets
		  WHERE tenant_id = $1::uuid
		    AND user_id = $2::uuid
		    AND status = $3
		    AND expires_at <= NOW()
		  ORDER BY created_at ASC
		  LIMIT 1
		  FOR UPDATE`,
		s.tenantID,
		userUUID,
		MatchmakingStatusSearching,
	).Scan(
		&search.ID,
		&search.TopicID,
		&search.TopicName,
		&search.SyllabusID,
		&search.QuestionCount,
		&search.Status,
		&search.MatchedChallengeID,
		&search.ExpiresAt,
		&search.CancelledAt,
		&search.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("get timed-out ticket: %w", err)
	}
	search.UserID = externalUserID
	return &search, nil
}

func (s *PostgresChallengeStore) claimTimedOutSearchAsAIFallback(ctx context.Context, tx pgx.Tx, userUUID, externalUserID string) (*StartChallengeSearchResult, error) {
	search, err := s.getTimedOutSearchingSearchForUser(ctx, tx, userUUID, externalUserID)
	if err != nil || search == nil {
		return nil, err
	}

	challenge := &Challenge{
		CreatorID:     externalUserID,
		TopicID:       search.TopicID,
		TopicName:     search.TopicName,
		SyllabusID:    search.SyllabusID,
		QuestionCount: normalizeChallengeQuestionCount(search.QuestionCount),
		State:         ChallengeStateReady,
		MatchSource:   ChallengeMatchSourceAIFallback,
		OpponentKind:  ChallengeOpponentKindAI,
	}

	err = tx.QueryRow(ctx,
		`INSERT INTO challenges (
			tenant_id,
			creator_user_id,
			topic_id,
			topic_name,
			syllabus_id,
			match_source,
			opponent_kind,
			question_count,
			state,
			ready_at
		) VALUES ($1::uuid, $2::uuid, $3, $4, $5, $6, $7, $8, $9, NOW())
		RETURNING id::text, created_at, ready_at`,
		s.tenantID,
		userUUID,
		challenge.TopicID,
		challenge.TopicName,
		challenge.SyllabusID,
		challenge.MatchSource,
		challenge.OpponentKind,
		challenge.QuestionCount,
		challenge.State,
	).Scan(&challenge.ID, &challenge.CreatedAt, &challenge.ReadyAt)
	if err != nil {
		return nil, fmt.Errorf("insert AI fallback challenge: %w", err)
	}

	cmd, err := tx.Exec(ctx,
		`UPDATE challenge_matchmaking_tickets
		    SET status = $2,
		        matched_challenge_id = $3::uuid,
		        updated_at = NOW()
		  WHERE id = $1::uuid
		    AND status = $4`,
		search.ID,
		MatchmakingStatusMatched,
		challenge.ID,
		MatchmakingStatusSearching,
	)
	if err != nil {
		return nil, fmt.Errorf("mark timed-out ticket matched: %w", err)
	}
	if cmd.RowsAffected() != 1 {
		return nil, fmt.Errorf("mark timed-out ticket matched: rows affected = %d", cmd.RowsAffected())
	}

	search.Status = MatchmakingStatusMatched
	search.MatchedChallengeID = challenge.ID
	return &StartChallengeSearchResult{
		Search:    search,
		Challenge: challenge,
	}, nil
}

func (s *PostgresChallengeStore) getMatchedSearchForUser(ctx context.Context, tx pgx.Tx, userUUID, externalUserID string) (*ChallengeSearch, *Challenge, error) {
	var search ChallengeSearch
	err := tx.QueryRow(ctx,
		`SELECT t.id::text,
		        t.topic_id,
		        t.topic_name,
		        t.syllabus_id,
		        t.question_count,
		        t.status,
		        t.matched_challenge_id::text,
		        t.expires_at,
		        t.cancelled_at,
		        t.created_at
		   FROM challenge_matchmaking_tickets t
		   JOIN challenges c ON c.id = t.matched_challenge_id
		  WHERE t.tenant_id = $1::uuid
		    AND t.user_id = $2::uuid
		    AND t.status = $3
		    AND (
		        c.state IN ('waiting', 'ready', 'active')
		        OR (c.state = 'pending_acceptance' AND (c.join_deadline_at IS NULL OR c.join_deadline_at > NOW()))
		    )
		  ORDER BY t.created_at DESC
		  LIMIT 1
		  FOR UPDATE`,
		s.tenantID,
		userUUID,
		MatchmakingStatusMatched,
	).Scan(
		&search.ID,
		&search.TopicID,
		&search.TopicName,
		&search.SyllabusID,
		&search.QuestionCount,
		&search.Status,
		&search.MatchedChallengeID,
		&search.ExpiresAt,
		&search.CancelledAt,
		&search.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil, nil
		}
		return nil, nil, fmt.Errorf("get matched ticket: %w", err)
	}
	search.UserID = externalUserID

	challenge, err := s.getChallengeByID(ctx, tx, search.MatchedChallengeID)
	if err != nil {
		return nil, nil, err
	}
	return &search, challenge, nil
}

func (s *PostgresChallengeStore) getPendingQueueChallengeForUserForUpdate(ctx context.Context, tx pgx.Tx, userUUID string) (*Challenge, string, error) {
	var role string
	var challengeID string
	err := tx.QueryRow(ctx,
		`SELECT c.id::text,
		        CASE
		            WHEN c.creator_user_id = $2::uuid THEN 'creator'
		            WHEN c.opponent_user_id = $2::uuid THEN 'opponent'
		        END
		   FROM challenges c
		  WHERE c.tenant_id = $1::uuid
		    AND c.match_source = $3
		    AND c.state = $4
		    AND (c.join_deadline_at IS NULL OR c.join_deadline_at > NOW())
		    AND (c.creator_user_id = $2::uuid OR c.opponent_user_id = $2::uuid)
		  ORDER BY c.created_at DESC
		  LIMIT 1
		  FOR UPDATE`,
		s.tenantID,
		userUUID,
		ChallengeMatchSourceQueue,
		ChallengeStatePendingAcceptance,
	).Scan(&challengeID, &role)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, "", nil
		}
		return nil, "", fmt.Errorf("get pending queue challenge: %w", err)
	}
	challenge, err := s.getChallengeByID(ctx, tx, challengeID)
	if err != nil {
		return nil, "", err
	}
	return challenge, role, nil
}

func (s *PostgresChallengeStore) getReadyQueueChallengeForUser(ctx context.Context, tx pgx.Tx, userUUID string) (*Challenge, error) {
	var challengeID string
	err := tx.QueryRow(ctx,
		`SELECT c.id::text
		   FROM challenges c
		  WHERE c.tenant_id = $1::uuid
		    AND c.match_source = $3
		    AND c.state = $4
		    AND (c.creator_user_id = $2::uuid OR c.opponent_user_id = $2::uuid)
		  ORDER BY c.created_at DESC
		  LIMIT 1`,
		s.tenantID,
		userUUID,
		ChallengeMatchSourceQueue,
		ChallengeStateReady,
	).Scan(&challengeID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("get ready queue challenge: %w", err)
	}
	return s.getChallengeByID(ctx, tx, challengeID)
}

func (s *PostgresChallengeStore) expirePendingQueueChallenge(ctx context.Context, tx pgx.Tx, challengeID string) error {
	if _, err := tx.Exec(ctx,
		`UPDATE challenges
		    SET state = 'expired',
		        updated_at = NOW()
		  WHERE id = $1::uuid`,
		challengeID,
	); err != nil {
		return fmt.Errorf("expire pending challenge: %w", err)
	}
	if _, err := tx.Exec(ctx,
		`UPDATE challenge_matchmaking_tickets
		    SET status = $2,
		        updated_at = NOW()
		  WHERE tenant_id = $1::uuid
		    AND matched_challenge_id = $3::uuid
		    AND status = $4`,
		s.tenantID,
		MatchmakingStatusExpired,
		challengeID,
		MatchmakingStatusMatched,
	); err != nil {
		return fmt.Errorf("expire pending challenge tickets: %w", err)
	}
	return nil
}

func (s *PostgresChallengeStore) insertChallengeSearch(ctx context.Context, tx pgx.Tx, userUUID, externalUserID string, input ChallengeCreateInput) (*ChallengeSearch, error) {
	var search ChallengeSearch
	err := tx.QueryRow(ctx,
		`INSERT INTO challenge_matchmaking_tickets (
			tenant_id,
			user_id,
			topic_id,
			topic_name,
			syllabus_id,
			question_count,
			status,
			expires_at
		) VALUES ($1::uuid, $2::uuid, $3, $4, $5, $6, $7, NOW() + $8 * INTERVAL '1 second')
		RETURNING id::text, expires_at, created_at`,
		s.tenantID,
		userUUID,
		strings.TrimSpace(input.TopicID),
		strings.TrimSpace(input.TopicName),
		strings.TrimSpace(input.SyllabusID),
		normalizeChallengeQuestionCount(input.QuestionCount),
		MatchmakingStatusSearching,
		int(matchmakingWaitTimeout/time.Second),
	).Scan(&search.ID, &search.ExpiresAt, &search.CreatedAt)
	if err != nil {
		if isUniqueViolation(err) {
			search, getErr := s.getSearchingSearchForUser(ctx, tx, userUUID, externalUserID)
			if getErr != nil {
				return nil, getErr
			}
			if search != nil {
				return search, nil
			}
		}
		return nil, fmt.Errorf("insert matchmaking ticket: %w", err)
	}
	search.UserID = externalUserID
	search.TopicID = strings.TrimSpace(input.TopicID)
	search.TopicName = strings.TrimSpace(input.TopicName)
	search.SyllabusID = strings.TrimSpace(input.SyllabusID)
	search.QuestionCount = normalizeChallengeQuestionCount(input.QuestionCount)
	search.Status = MatchmakingStatusSearching
	return &search, nil
}

func (s *PostgresChallengeStore) getChallengeByID(ctx context.Context, querier interface {
	QueryRow(context.Context, string, ...any) pgx.Row
}, challengeID string) (*Challenge, error) {
	var challenge Challenge
	var creatorID string
	var opponentID *string
	var joinDeadlineAt *time.Time
	var creatorAcceptedAt *time.Time
	var opponentAcceptedAt *time.Time
	err := querier.QueryRow(ctx,
		`SELECT c.id::text,
		        COALESCE(c.invite_code, ''),
		        creator.external_id,
		        opponent.external_id,
		        c.topic_id,
		        c.topic_name,
		        c.syllabus_id,
		        c.question_count,
		        c.state,
		        c.match_source,
		        c.opponent_kind,
		        c.created_at,
		        c.join_deadline_at,
		        c.creator_accepted_at,
		        c.opponent_accepted_at,
		        c.ready_at
		   FROM challenges c
		   JOIN users creator ON creator.id = c.creator_user_id
		   LEFT JOIN users opponent ON opponent.id = c.opponent_user_id
		  WHERE c.tenant_id = $1::uuid
		    AND c.id = $2::uuid
		    AND c.state IN ('waiting', 'pending_acceptance', 'ready', 'active')`,
		s.tenantID,
		challengeID,
	).Scan(
		&challenge.ID,
		&challenge.Code,
		&creatorID,
		&opponentID,
		&challenge.TopicID,
		&challenge.TopicName,
		&challenge.SyllabusID,
		&challenge.QuestionCount,
		&challenge.State,
		&challenge.MatchSource,
		&challenge.OpponentKind,
		&challenge.CreatedAt,
		&joinDeadlineAt,
		&creatorAcceptedAt,
		&opponentAcceptedAt,
		&challenge.ReadyAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrChallengeNotFound
		}
		return nil, fmt.Errorf("get challenge by id: %w", err)
	}
	challenge.CreatorID = creatorID
	if opponentID != nil {
		challenge.OpponentID = *opponentID
	}
	challenge.JoinDeadlineAt = joinDeadlineAt
	challenge.CreatorAcceptedAt = creatorAcceptedAt
	challenge.OpponentAcceptedAt = opponentAcceptedAt
	return &challenge, nil
}

func (s *PostgresChallengeStore) findCompatibleSearch(ctx context.Context, tx pgx.Tx, currentUserUUID, topicID string) (*ChallengeSearch, string, error) {
	var search ChallengeSearch
	var matchUUID string
	err := tx.QueryRow(ctx,
		`SELECT t.id::text,
		        u.external_id,
		        t.topic_id,
		        t.topic_name,
		        t.syllabus_id,
		        t.status,
		        COALESCE(t.matched_challenge_id::text, ''),
		        t.expires_at,
		        t.cancelled_at,
		        t.created_at,
		        t.user_id::text
		   FROM challenge_matchmaking_tickets t
		   JOIN users u ON u.id = t.user_id
		  WHERE t.tenant_id = $1::uuid
		    AND t.topic_id = $2
		    AND t.status = $3
		    AND t.expires_at > NOW()
		    AND t.user_id <> $4::uuid
		    AND u.channel = $5
		  ORDER BY t.created_at ASC
		  LIMIT 1
		  FOR UPDATE SKIP LOCKED`,
		s.tenantID,
		topicID,
		MatchmakingStatusSearching,
		currentUserUUID,
		s.channel,
	).Scan(
		&search.ID,
		&search.UserID,
		&search.TopicID,
		&search.TopicName,
		&search.SyllabusID,
		&search.Status,
		&search.MatchedChallengeID,
		&search.ExpiresAt,
		&search.CancelledAt,
		&search.CreatedAt,
		&matchUUID,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, "", nil
		}
		return nil, "", fmt.Errorf("find compatible ticket: %w", err)
	}
	return &search, matchUUID, nil
}

// GenerateChallengeCode generates a 6-character uppercase code.
func GenerateChallengeCode() string {
	const charset = "ABCDEFGHJKLMNPQRSTUVWXYZ23456789"
	code := make([]byte, 6)
	for i := range code {
		n, _ := rand.Int(rand.Reader, big.NewInt(int64(len(charset))))
		code[i] = charset[n.Int64()]
	}
	return string(code)
}

func normalizeChallengeCode(code string) string {
	return strings.ToUpper(strings.TrimSpace(code))
}

func normalizeChallengeQuestionCount(n int) int {
	if n <= 0 {
		return defaultChallengeQuestionCount
	}
	return n
}

func cloneChallenge(challenge *Challenge) *Challenge {
	if challenge == nil {
		return nil
	}
	cloned := *challenge
	if challenge.JoinDeadlineAt != nil {
		joinDeadlineAt := *challenge.JoinDeadlineAt
		cloned.JoinDeadlineAt = &joinDeadlineAt
	}
	if challenge.CreatorAcceptedAt != nil {
		creatorAcceptedAt := *challenge.CreatorAcceptedAt
		cloned.CreatorAcceptedAt = &creatorAcceptedAt
	}
	if challenge.OpponentAcceptedAt != nil {
		opponentAcceptedAt := *challenge.OpponentAcceptedAt
		cloned.OpponentAcceptedAt = &opponentAcceptedAt
	}
	if challenge.ReadyAt != nil {
		readyAt := *challenge.ReadyAt
		cloned.ReadyAt = &readyAt
	}
	return &cloned
}

func cloneChallengeSearch(search *ChallengeSearch) *ChallengeSearch {
	if search == nil {
		return nil
	}
	cloned := *search
	if search.CancelledAt != nil {
		cancelledAt := *search.CancelledAt
		cloned.CancelledAt = &cancelledAt
	}
	return &cloned
}

func challengeResumableAt(challenge *Challenge, now time.Time) bool {
	if challenge == nil {
		return false
	}
	switch challenge.State {
	case ChallengeStateWaiting, ChallengeStateReady, "active":
		return true
	case ChallengeStatePendingAcceptance:
		return challenge.JoinDeadlineAt == nil || challenge.JoinDeadlineAt.After(now)
	default:
		return false
	}
}

func ptrTime(t time.Time) *time.Time {
	return &t
}

func (s *PostgresChallengeStore) lockUser(ctx context.Context, tx pgx.Tx, userUUID string) error {
	_, err := tx.Exec(ctx,
		`SELECT 1
		   FROM users
		  WHERE tenant_id = $1::uuid
		    AND id = $2::uuid
		  FOR UPDATE`,
		s.tenantID,
		userUUID,
	)
	if err != nil {
		return fmt.Errorf("lock challenge user: %w", err)
	}
	return nil
}

func (s *PostgresChallengeStore) cleanupExpiredSearchingSearchesForUser(ctx context.Context, tx pgx.Tx, userUUID string) error {
	_, err := tx.Exec(ctx,
		`UPDATE challenge_matchmaking_tickets
		    SET status = $3,
		        updated_at = NOW()
		  WHERE tenant_id = $1::uuid
		    AND user_id = $2::uuid
		    AND status = $4
		    AND expires_at <= NOW()`,
		s.tenantID,
		userUUID,
		MatchmakingStatusExpired,
		MatchmakingStatusSearching,
	)
	if err != nil {
		return fmt.Errorf("expire stale searching tickets: %w", err)
	}
	return nil
}

func (s *PostgresChallengeStore) cleanupStaleMatchedSearchesForUser(ctx context.Context, tx pgx.Tx, userUUID string) error {
	_, err := tx.Exec(ctx,
		`UPDATE challenge_matchmaking_tickets t
		    SET status = $3,
		        updated_at = NOW()
		  WHERE t.tenant_id = $1::uuid
		    AND t.user_id = $2::uuid
		    AND t.status = $4
		    AND NOT EXISTS (
		        SELECT 1
		          FROM challenges c
		         WHERE c.id = t.matched_challenge_id
		           AND (
		               c.state IN ('waiting', 'ready', 'active')
		               OR (c.state = 'pending_acceptance' AND (c.join_deadline_at IS NULL OR c.join_deadline_at > NOW()))
		           )
		    )`,
		s.tenantID,
		userUUID,
		MatchmakingStatusExpired,
		MatchmakingStatusMatched,
	)
	if err != nil {
		return fmt.Errorf("expire stale matched tickets: %w", err)
	}
	return nil
}

func (s *PostgresChallengeStore) ensureUserCanOpenChallenge(ctx context.Context, tx pgx.Tx, userUUID string) error {
	var hasBlocking bool
	err := tx.QueryRow(ctx,
		`SELECT EXISTS (
		    SELECT 1
		      FROM challenge_matchmaking_tickets t
		      LEFT JOIN challenges c
		        ON c.id = t.matched_challenge_id
		       AND (
		           c.state IN ('waiting', 'ready', 'active')
		           OR (c.state = 'pending_acceptance' AND (c.join_deadline_at IS NULL OR c.join_deadline_at > NOW()))
		       )
		     WHERE t.tenant_id = $1::uuid
		       AND t.user_id = $2::uuid
		       AND (
		           (t.status = $3 AND t.expires_at > NOW())
		           OR (t.status = $4 AND c.id IS NOT NULL)
		       )
		)`,
		s.tenantID,
		userUUID,
		MatchmakingStatusSearching,
		MatchmakingStatusMatched,
	).Scan(&hasBlocking)
	if err != nil {
		return fmt.Errorf("check blocking challenge search: %w", err)
	}
	if hasBlocking {
		return ErrChallengeAlreadyActive
	}

	err = tx.QueryRow(ctx,
		`SELECT EXISTS (
		    SELECT 1
		      FROM challenges
		     WHERE tenant_id = $1::uuid
		       AND (
		           state IN ('waiting', 'ready', 'active')
		           OR (state = 'pending_acceptance' AND (join_deadline_at IS NULL OR join_deadline_at > NOW()))
		       )
		       AND (creator_user_id = $2::uuid OR opponent_user_id = $2::uuid)
		)`,
		s.tenantID,
		userUUID,
	).Scan(&hasBlocking)
	if err != nil {
		return fmt.Errorf("check live challenge: %w", err)
	}
	if hasBlocking {
		return ErrChallengeAlreadyActive
	}
	return nil
}

func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505"
}
