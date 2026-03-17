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
	ChallengeStateWaiting = "waiting"
	ChallengeStateReady   = "ready"

	ChallengeMatchSourceInviteCode = "invite_code"
	defaultChallengeQuestionCount  = 5
	challengeCodeInsertMaxAttempts = 8
)

var (
	ErrChallengeNotFound    = errors.New("challenge not found")
	ErrChallengeNotJoinable = errors.New("challenge not joinable")
	ErrChallengeSelfJoin    = errors.New("challenge self join")
)

// Challenge represents a peer challenge.
type Challenge struct {
	ID            string
	Code          string
	CreatorID     string
	OpponentID    string
	TopicID       string
	TopicName     string
	SyllabusID    string
	QuestionCount int
	State         string
	MatchSource   string
	CreatedAt     time.Time
	ReadyAt       *time.Time
}

// ChallengeCreateInput captures challenge creation input.
type ChallengeCreateInput struct {
	TopicID       string
	TopicName     string
	SyllabusID    string
	QuestionCount int
}

// ChallengeStore persists invite-code challenges.
type ChallengeStore interface {
	CreateInviteChallenge(creatorID string, input ChallengeCreateInput) (*Challenge, error)
	JoinChallenge(code, opponentID string) (*Challenge, error)
	GetChallenge(code string) (*Challenge, error)
}

// MemoryChallengeStore stores challenges in memory.
type MemoryChallengeStore struct {
	// mu guards challenges because Go maps are not safe for concurrent writes.
	mu         sync.RWMutex
	challenges map[string]*Challenge
}

func NewMemoryChallengeStore() *MemoryChallengeStore {
	return &MemoryChallengeStore{
		challenges: make(map[string]*Challenge),
	}
}

func (s *MemoryChallengeStore) CreateInviteChallenge(creatorID string, input ChallengeCreateInput) (*Challenge, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

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
		CreatedAt:     time.Now(),
	}
	s.challenges[code] = challenge
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
	return cloneChallenge(challenge), nil
}

// PostgresChallengeStore persists challenges in PostgreSQL.
type PostgresChallengeStore struct {
	pool     *pgxpool.Pool
	tenantID string
	channel  string
}

func NewPostgresChallengeStore(pool *pgxpool.Pool, tenantID string) *PostgresChallengeStore {
	return &PostgresChallengeStore{
		pool:     pool,
		tenantID: tenantID,
		channel:  defaultChannel,
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
		}

		err := s.pool.QueryRow(ctx,
			`INSERT INTO challenges (
				tenant_id,
				creator_user_id,
				topic_id,
				topic_name,
				syllabus_id,
				match_source,
				invite_code,
				question_count,
				state
			) VALUES ($1::uuid, $2::uuid, $3, $4, $5, $6, $7, $8, $9)
			RETURNING id::text, created_at, ready_at`,
			s.tenantID,
			creatorID,
			challenge.TopicID,
			challenge.TopicName,
			challenge.SyllabusID,
			challenge.MatchSource,
			challenge.Code,
			challenge.QuestionCount,
			challenge.State,
		).Scan(&challenge.ID, &challenge.CreatedAt, &challenge.ReadyAt)
		if err == nil {
			return challenge, nil
		}
		if isUniqueViolation(err) {
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

func (s *PostgresChallengeStore) getChallengeForUpdate(ctx context.Context, tx pgx.Tx, code string) (*Challenge, string, error) {
	var challenge Challenge
	var readyAt *time.Time
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
		        c.created_at,
		        c.ready_at,
		        c.creator_user_id::text
		   FROM challenges c
		   JOIN users creator ON creator.id = c.creator_user_id
		   LEFT JOIN users opponent ON opponent.id = c.opponent_user_id
		  WHERE c.tenant_id = $1::uuid
		    AND c.invite_code = $2
		  FOR UPDATE`,
		s.tenantID,
		code,
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
		&challenge.CreatedAt,
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
	challenge.ReadyAt = readyAt
	return &challenge, creatorUUID, nil
}

func (s *PostgresChallengeStore) getChallenge(ctx context.Context, querier interface {
	QueryRow(context.Context, string, ...any) pgx.Row
}, code string) (*Challenge, error) {
	var challenge Challenge
	var creatorID string
	var opponentID *string
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
		        c.created_at,
		        c.ready_at
		   FROM challenges c
		   JOIN users creator ON creator.id = c.creator_user_id
		   LEFT JOIN users opponent ON opponent.id = c.opponent_user_id
		  WHERE c.tenant_id = $1::uuid
		    AND c.invite_code = $2`,
		s.tenantID,
		code,
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
		&challenge.CreatedAt,
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
	if challenge.ReadyAt != nil {
		readyAt := *challenge.ReadyAt
		cloned.ReadyAt = &readyAt
	}
	return &cloned
}

func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505"
}
