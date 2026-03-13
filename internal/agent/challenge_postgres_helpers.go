package agent

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
)

func (s *PostgresChallengeStore) saveChallenge(ctx context.Context, tx pgx.Tx, challenge *Challenge, opponentUserID any) error {
	metadataJSON, err := marshalChallengeMetadata(challenge.Metadata)
	if err != nil {
		return fmt.Errorf("marshal challenge metadata: %w", err)
	}
	questionsJSON, err := jsonMarshalQuestions(challenge.Questions)
	if err != nil {
		return err
	}

	var winnerUserID any
	if challenge.WinnerUserID != "" {
		winnerUserID, err = s.lookupUserIDByExternalID(ctx, tx, challenge.WinnerUserID)
		if err != nil {
			return err
		}
	}

	_, err = tx.Exec(ctx, `
		UPDATE challenges
		   SET source = $3,
		       opponent_type = $4,
		       opponent_user_id = COALESCE($5::uuid, opponent_user_id),
		       topic_id = $6,
		       topic_name = $7,
		       subject_id = $8,
		       syllabus_id = $9,
		       questions = $10::jsonb,
		       question_count = $11,
		       state = $12,
		       creator_ready_at = $13,
		       opponent_ready_at = $14,
		       creator_correct_count = $15,
		       opponent_correct_count = $16,
		       creator_completed_at = $17,
		       opponent_completed_at = $18,
		       creator_finish_xp_granted = $19,
		       opponent_finish_xp_granted = $20,
		       winner_user_id = $21,
		       winner_xp_granted = $22,
		       metadata = $23::jsonb,
		       updated_at = NOW()
		 WHERE tenant_id = $1::uuid
		   AND code = $2`,
		s.tenantID,
		challenge.Code,
		challenge.Source,
		challenge.OpponentType,
		opponentUserID,
		challenge.TopicID,
		challenge.TopicName,
		challenge.SubjectID,
		challenge.SyllabusID,
		questionsJSON,
		challenge.QuestionCount,
		challenge.State,
		challenge.CreatorReadyAt,
		challenge.OpponentReadyAt,
		challenge.CreatorCorrectCount,
		challenge.OpponentCorrectCount,
		challenge.CreatorCompletedAt,
		challenge.OpponentCompletedAt,
		challenge.CreatorFinishXPGranted,
		challenge.OpponentFinishXPGranted,
		winnerUserID,
		challenge.WinnerXPGranted,
		metadataJSON,
	)
	if err != nil {
		return fmt.Errorf("save challenge: %w", err)
	}
	return nil
}

func (s *PostgresChallengeStore) queryChallenge(ctx context.Context, q challengeQueryer, query string, args ...any) (*Challenge, error) {
	return scanChallenge(q.QueryRow(ctx, query, args...))
}

func (s *PostgresChallengeStore) resolveOrCreateUserID(ctx context.Context, externalID string) (string, error) {
	store := &PostgresStore{pool: s.pool, tenantID: s.tenantID, channel: s.channel}
	return store.resolveOrCreateUser(ctx, externalID)
}

func (s *PostgresChallengeStore) lookupUserIDByExternalID(ctx context.Context, tx pgx.Tx, externalID string) (string, error) {
	var userID string
	err := tx.QueryRow(ctx, `
		SELECT id::text
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
		return "", fmt.Errorf("lookup challenge user id: %w", err)
	}
	return userID, nil
}

type challengeRow interface {
	Scan(dest ...any) error
}

type challengeQueryer interface {
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}

func baseChallengeSelect() string {
	return `SELECT c.id::text, c.code, c.source, c.opponent_type, cu.external_id, ou.external_id,
	               c.topic_id, c.topic_name, c.subject_id, c.syllabus_id, c.questions, c.question_count,
	               c.state, c.creator_ready_at, c.opponent_ready_at, c.creator_correct_count, c.opponent_correct_count,
	               c.creator_completed_at, c.opponent_completed_at, c.creator_finish_xp_granted, c.opponent_finish_xp_granted,
	               wu.external_id, c.winner_xp_granted, c.metadata, c.created_at, c.updated_at
	          FROM challenges c
	          JOIN users cu ON cu.id = c.creator_user_id
	          LEFT JOIN users ou ON ou.id = c.opponent_user_id
	          LEFT JOIN users wu ON wu.id = c.winner_user_id`
}

func lockingChallengeSelect() string {
	return `SELECT c.id::text, c.code, c.source, c.opponent_type,
	               (SELECT u.external_id FROM users u WHERE u.id = c.creator_user_id),
	               (SELECT u.external_id FROM users u WHERE u.id = c.opponent_user_id),
	               c.topic_id, c.topic_name, c.subject_id, c.syllabus_id, c.questions, c.question_count,
	               c.state, c.creator_ready_at, c.opponent_ready_at, c.creator_correct_count, c.opponent_correct_count,
	               c.creator_completed_at, c.opponent_completed_at, c.creator_finish_xp_granted, c.opponent_finish_xp_granted,
	               (SELECT u.external_id FROM users u WHERE u.id = c.winner_user_id),
	               c.winner_xp_granted, c.metadata, c.created_at, c.updated_at
	          FROM challenges c`
}

func scanChallenge(row challengeRow) (*Challenge, error) {
	var (
		challenge     Challenge
		opponentID    *string
		winnerUserID  *string
		questionsJSON []byte
		metadataJSON  []byte
	)
	err := row.Scan(
		&challenge.ID,
		&challenge.Code,
		&challenge.Source,
		&challenge.OpponentType,
		&challenge.CreatorID,
		&opponentID,
		&challenge.TopicID,
		&challenge.TopicName,
		&challenge.SubjectID,
		&challenge.SyllabusID,
		&questionsJSON,
		&challenge.QuestionCount,
		&challenge.State,
		&challenge.CreatorReadyAt,
		&challenge.OpponentReadyAt,
		&challenge.CreatorCorrectCount,
		&challenge.OpponentCorrectCount,
		&challenge.CreatorCompletedAt,
		&challenge.OpponentCompletedAt,
		&challenge.CreatorFinishXPGranted,
		&challenge.OpponentFinishXPGranted,
		&winnerUserID,
		&challenge.WinnerXPGranted,
		&metadataJSON,
		&challenge.CreatedAt,
		&challenge.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	if opponentID != nil {
		challenge.OpponentID = *opponentID
	}
	if winnerUserID != nil {
		challenge.WinnerUserID = *winnerUserID
	}
	if err := jsonUnmarshalQuestions(questionsJSON, &challenge.Questions); err != nil {
		return nil, err
	}
	meta, err := unmarshalChallengeMetadata(metadataJSON)
	if err != nil {
		return nil, fmt.Errorf("unmarshal challenge metadata: %w", err)
	}
	challenge.Metadata = meta
	return &challenge, nil
}

func mapChallengeQueryError(err error) error {
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrChallengeNotFound
	}
	return err
}

func jsonMarshalQuestions(questions []QuizQuestion) ([]byte, error) {
	data, err := json.Marshal(questions)
	if err != nil {
		return nil, fmt.Errorf("marshal challenge questions: %w", err)
	}
	return data, nil
}

func jsonUnmarshalQuestions(data []byte, out *[]QuizQuestion) error {
	if len(data) == 0 {
		*out = nil
		return nil
	}
	if err := json.Unmarshal(data, out); err != nil {
		return fmt.Errorf("unmarshal challenge questions: %w", err)
	}
	return nil
}
