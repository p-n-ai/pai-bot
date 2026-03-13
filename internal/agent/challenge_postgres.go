package agent

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

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

func (s *PostgresChallengeStore) CreatePublicQueue(userID string, input ChallengeInput) (*Challenge, error) {
	return s.insertChallenge(userID, challengeSourcePublicQueue, challengeOpponentHuman, input)
}

func (s *PostgresChallengeStore) CreatePrivateChallenge(userID string, input ChallengeInput) (*Challenge, error) {
	return s.insertChallenge(userID, challengeSourcePrivateCode, challengeOpponentHuman, input)
}

func (s *PostgresChallengeStore) insertChallenge(externalID, source, opponentType string, input ChallengeInput) (*Challenge, error) {
	if live, err := s.GetLiveChallengeForUser(externalID); err != nil {
		return nil, err
	} else if live != nil {
		return nil, ErrChallengeAlreadyActive
	}

	ctx, cancel := context.WithTimeout(context.Background(), dbTimeout)
	defer cancel()

	creatorUserID, err := s.resolveOrCreateUserID(ctx, externalID)
	if err != nil {
		return nil, err
	}
	metadataJSON, err := marshalChallengeMetadata(input.Metadata)
	if err != nil {
		return nil, fmt.Errorf("marshal challenge metadata: %w", err)
	}
	questionsJSON, err := jsonMarshalQuestions(trimChallengeQuestions(input.Questions, input.QuestionCount))
	if err != nil {
		return nil, err
	}
	questionCount := normalizeChallengeQuestionCount(input.QuestionCount, input.Questions)

	for attempt := 0; attempt < 5; attempt++ {
		challenge := &Challenge{
			Code:          generateUniqueChallengeCode(nil),
			Source:        source,
			OpponentType:  opponentType,
			CreatorID:     externalID,
			TopicID:       input.TopicID,
			TopicName:     input.TopicName,
			SubjectID:     input.SubjectID,
			SyllabusID:    input.SyllabusID,
			Questions:     trimChallengeQuestions(input.Questions, input.QuestionCount),
			QuestionCount: questionCount,
			State:         challengeStateWaiting,
			Metadata:      input.Metadata,
		}
		err = s.pool.QueryRow(ctx,
			`INSERT INTO challenges (
			    code, source, opponent_type, creator_user_id, tenant_id,
			    topic_id, topic_name, subject_id, syllabus_id, questions, question_count,
			    state, metadata
			) VALUES (
			    $1, $2, $3, $4::uuid, $5::uuid,
			    $6, $7, $8, $9, $10::jsonb, $11,
			    $12, $13::jsonb
			)
			RETURNING id::text, created_at, updated_at`,
			challenge.Code,
			challenge.Source,
			challenge.OpponentType,
			creatorUserID,
			s.tenantID,
			challenge.TopicID,
			challenge.TopicName,
			challenge.SubjectID,
			challenge.SyllabusID,
			questionsJSON,
			questionCount,
			challenge.State,
			metadataJSON,
		).Scan(&challenge.ID, &challenge.CreatedAt, &challenge.UpdatedAt)
		if err == nil {
			return challenge, nil
		}
		var pgErr *pgconn.PgError
		if !errors.As(err, &pgErr) || pgErr.Code != "23505" {
			return nil, fmt.Errorf("insert challenge: %w", err)
		}
	}
	return nil, fmt.Errorf("failed to generate unique challenge code")
}

func (s *PostgresChallengeStore) GetChallenge(code string) (*Challenge, error) {
	ctx, cancel := context.WithTimeout(context.Background(), dbTimeout)
	defer cancel()
	challenge, err := s.queryChallenge(ctx, s.pool, baseChallengeSelect()+` WHERE c.tenant_id = $1::uuid AND c.code = $2`, s.tenantID, normalizeChallengeCodeValue(code))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrChallengeNotFound
	}
	return challenge, err
}

func (s *PostgresChallengeStore) GetCurrentChallengeForUser(userID string) (*Challenge, error) {
	return s.getChallengeForUser(userID, false)
}

func (s *PostgresChallengeStore) GetLiveChallengeForUser(userID string) (*Challenge, error) {
	return s.getChallengeForUser(userID, true)
}

func (s *PostgresChallengeStore) getChallengeForUser(userID string, liveOnly bool) (*Challenge, error) {
	ctx, cancel := context.WithTimeout(context.Background(), dbTimeout)
	defer cancel()
	query := baseChallengeSelect() + `
		WHERE c.tenant_id = $1::uuid
		  AND ($2 = cu.external_id OR $2 = ou.external_id)
	`
	if liveOnly {
		query += ` AND c.state IN ('waiting', 'ready', 'active')`
	}
	query += ` ORDER BY c.updated_at DESC LIMIT 1`
	challenge, err := s.queryChallenge(ctx, s.pool, query, s.tenantID, userID)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	return challenge, err
}

func (s *PostgresChallengeStore) ListWaitingPublicChallenges(excludeUserID string) ([]*Challenge, error) {
	ctx, cancel := context.WithTimeout(context.Background(), dbTimeout)
	defer cancel()
	rows, err := s.pool.Query(ctx, baseChallengeSelect()+`
		WHERE c.tenant_id = $1::uuid
		  AND c.source = $2
		  AND c.state = $3
		  AND c.opponent_user_id IS NULL
		  AND cu.external_id <> $4
		ORDER BY c.created_at ASC
	`, s.tenantID, challengeSourcePublicQueue, challengeStateWaiting, excludeUserID)
	if err != nil {
		return nil, fmt.Errorf("list waiting public challenges: %w", err)
	}
	defer rows.Close()
	var out []*Challenge
	for rows.Next() {
		challenge, scanErr := scanChallenge(rows)
		if scanErr != nil {
			return nil, scanErr
		}
		out = append(out, challenge)
	}
	return out, rows.Err()
}

func (s *PostgresChallengeStore) ActivateHumanMatch(code, opponentID string, input ChallengeInput) (*Challenge, error) {
	if live, err := s.GetLiveChallengeForUser(opponentID); err != nil {
		return nil, err
	} else if live != nil {
		return nil, ErrChallengeAlreadyActive
	}
	ctx, cancel := context.WithTimeout(context.Background(), dbTimeout)
	defer cancel()

	opponentUserID, err := s.resolveOrCreateUserID(ctx, opponentID)
	if err != nil {
		return nil, err
	}

	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return nil, fmt.Errorf("begin activate human match tx: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	challenge, err := s.queryChallenge(ctx, tx, lockingChallengeSelect()+`
		WHERE c.tenant_id = $1::uuid
		  AND c.code = $2
		FOR UPDATE
	`, s.tenantID, normalizeChallengeCodeValue(code))
	if err != nil {
		return nil, mapChallengeQueryError(err)
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
	if err := s.saveChallenge(ctx, tx, challenge, opponentUserID); err != nil {
		return nil, err
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit activate human match: %w", err)
	}
	return challenge, nil
}

func (s *PostgresChallengeStore) ActivateAIFallback(code string, input ChallengeInput) (*Challenge, error) {
	ctx, cancel := context.WithTimeout(context.Background(), dbTimeout)
	defer cancel()

	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return nil, fmt.Errorf("begin activate ai tx: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	challenge, err := s.queryChallenge(ctx, tx, lockingChallengeSelect()+`
		WHERE c.tenant_id = $1::uuid
		  AND c.code = $2
		FOR UPDATE
	`, s.tenantID, normalizeChallengeCodeValue(code))
	if err != nil {
		return nil, mapChallengeQueryError(err)
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
	if err := s.saveChallenge(ctx, tx, challenge, nil); err != nil {
		return nil, err
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit activate ai fallback: %w", err)
	}
	return challenge, nil
}

func (s *PostgresChallengeStore) JoinPrivateChallenge(code, opponentID string) (*Challenge, error) {
	if live, err := s.GetLiveChallengeForUser(opponentID); err != nil {
		return nil, err
	} else if live != nil {
		return nil, ErrChallengeAlreadyActive
	}
	ctx, cancel := context.WithTimeout(context.Background(), dbTimeout)
	defer cancel()
	opponentUserID, err := s.resolveOrCreateUserID(ctx, opponentID)
	if err != nil {
		return nil, err
	}

	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return nil, fmt.Errorf("begin private join tx: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	challenge, err := s.queryChallenge(ctx, tx, lockingChallengeSelect()+`
		WHERE c.tenant_id = $1::uuid
		  AND c.code = $2
		FOR UPDATE
	`, s.tenantID, normalizeChallengeCodeValue(code))
	if err != nil {
		return nil, mapChallengeQueryError(err)
	}
	if challenge.Source != challengeSourcePrivateCode {
		return nil, ErrChallengeNotFound
	}
	if challenge.CreatorID == opponentID {
		return nil, ErrChallengeSelfJoin
	}
	if challenge.OpponentID != "" {
		if challenge.OpponentID == opponentID {
			return challenge, nil
		}
		return nil, ErrChallengeFull
	}

	now := time.Now()
	challenge.OpponentID = opponentID
	challenge.OpponentType = challengeOpponentHuman
	challenge.OpponentReadyAt = nil
	challenge.State = challengeStateWaiting
	challenge.UpdatedAt = now
	if err := s.saveChallenge(ctx, tx, challenge, opponentUserID); err != nil {
		return nil, err
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit private join: %w", err)
	}
	return challenge, nil
}

func (s *PostgresChallengeStore) MarkReady(code, userID string) (*Challenge, error) {
	ctx, cancel := context.WithTimeout(context.Background(), dbTimeout)
	defer cancel()
	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return nil, fmt.Errorf("begin mark ready tx: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	challenge, err := s.queryChallenge(ctx, tx, lockingChallengeSelect()+`
		WHERE c.tenant_id = $1::uuid
		  AND c.code = $2
		FOR UPDATE
	`, s.tenantID, normalizeChallengeCodeValue(code))
	if err != nil {
		return nil, mapChallengeQueryError(err)
	}

	now := time.Now()
	switch userID {
	case challenge.CreatorID:
		if challenge.CreatorReadyAt == nil {
			challenge.CreatorReadyAt = ptrTime(now)
		}
	case challenge.OpponentID:
		if challenge.OpponentReadyAt == nil {
			challenge.OpponentReadyAt = ptrTime(now)
		}
	default:
		return nil, ErrChallengeNotFound
	}
	if challengeReadyToOpen(challenge) && challenge.State != challengeStateCompleted {
		challenge.State = challengeStateReady
	}
	challenge.UpdatedAt = now
	if err := s.saveChallenge(ctx, tx, challenge, nil); err != nil {
		return nil, err
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit mark ready: %w", err)
	}
	return challenge, nil
}

func (s *PostgresChallengeStore) CancelPublicQueue(userID string) (bool, error) {
	ctx, cancel := context.WithTimeout(context.Background(), dbTimeout)
	defer cancel()
	cmd, err := s.pool.Exec(ctx, `
		DELETE FROM challenges
		 WHERE tenant_id = $1::uuid
		   AND source = $2
		   AND state = $3
		   AND opponent_user_id IS NULL
		   AND creator_user_id = (
		     SELECT id FROM users
		      WHERE tenant_id = $1::uuid
		        AND channel = $4
		        AND external_id = $5
		      ORDER BY created_at ASC
		      LIMIT 1
		   )`,
		s.tenantID,
		challengeSourcePublicQueue,
		challengeStateWaiting,
		s.channel,
		userID,
	)
	if err != nil {
		return false, fmt.Errorf("cancel public queue: %w", err)
	}
	return cmd.RowsAffected() > 0, nil
}

func (s *PostgresChallengeStore) CompleteChallenge(code, userID string, correctAnswers int) (*ChallengeCompletion, error) {
	ctx, cancel := context.WithTimeout(context.Background(), dbTimeout)
	defer cancel()
	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return nil, fmt.Errorf("begin complete challenge tx: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	challenge, err := s.queryChallenge(ctx, tx, lockingChallengeSelect()+`
		WHERE c.tenant_id = $1::uuid
		  AND c.code = $2
		FOR UPDATE
	`, s.tenantID, normalizeChallengeCodeValue(code))
	if err != nil {
		return nil, mapChallengeQueryError(err)
	}
	if !challengeHasUser(challenge, userID) {
		return nil, ErrChallengeNotFound
	}

	completion := completeChallengeRecord(challenge, userID, correctAnswers)

	var opponentUserID any
	if challenge.OpponentID != "" {
		id, lookupErr := s.lookupUserIDByExternalID(ctx, tx, challenge.OpponentID)
		if lookupErr != nil {
			return nil, lookupErr
		}
		opponentUserID = id
	}
	if err := s.saveChallenge(ctx, tx, challenge, opponentUserID); err != nil {
		return nil, err
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit complete challenge: %w", err)
	}
	return completion, nil
}
