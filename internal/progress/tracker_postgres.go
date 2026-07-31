// Copyright 2026 the P&AI authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package progress

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

const dbTimeout = 5 * time.Second

const targetUserCTE = `WITH target_user AS (
	SELECT id
	FROM users
	WHERE external_id = $1
	  AND tenant_id = $2::uuid
	  AND channel = $3
	ORDER BY created_at ASC
	LIMIT 1
)`

// PostgresTracker is a PostgreSQL-backed implementation of Tracker.
type PostgresTracker struct {
	pool     *pgxpool.Pool
	tenantID string
	channel  string
}

// NewPostgresTracker creates a new PostgreSQL-backed tracker.
// tenantID is the UUID of the tenant for row-level isolation.
func NewPostgresTracker(pool *pgxpool.Pool, tenantID string) *PostgresTracker {
	return NewPostgresTrackerForChannel(pool, tenantID, "telegram")
}

// NewPostgresTrackerForChannel creates a legacy external-ID adapter scoped to
// one provider; LearnerTracker methods use internal IDs directly.
func NewPostgresTrackerForChannel(pool *pgxpool.Pool, tenantID, channel string) *PostgresTracker {
	channel = strings.TrimSpace(channel)
	if channel == "" {
		channel = "telegram"
	}
	return &PostgresTracker{pool: pool, tenantID: tenantID, channel: channel}
}

func (p *PostgresTracker) UpdateMastery(userID, syllabusID, topicID string, delta float64) error {
	learnerID, err := p.resolveLegacyLearnerID(userID)
	if err != nil {
		return err
	}
	return p.UpdateMasteryForLearner(learnerID, syllabusID, topicID, delta)
}

func (p *PostgresTracker) UpdateMasteryForLearner(learnerID LearnerID, syllabusID, topicID string, delta float64) error {
	delta = clamp(delta, 0.0, 1.0)

	ctx, cancel := context.WithTimeout(context.Background(), dbTimeout)
	defer cancel()

	// Try to get existing item.
	var existing ProgressItem
	var found bool
	err := p.pool.QueryRow(ctx,
		`SELECT mastery_score, ease_factor, interval_days, repetitions
		 FROM learning_progress
		 WHERE user_id = $1::uuid
		   AND tenant_id = $2::uuid
		   AND syllabus_id = $3 AND topic_id = $4`,
		learnerID.String(), p.tenantID, syllabusID, topicID,
	).Scan(&existing.MasteryScore, &existing.EaseFactor, &existing.IntervalDays, &existing.Repetitions)

	if err == nil {
		found = true
	} else if err != pgx.ErrNoRows {
		return err
	}

	now := time.Now()
	var score float64
	var sm2 SM2Result

	if !found {
		score = delta
		quality := DeltaToQuality(delta)
		sm2 = SM2Calculate(quality, 0, 2.5, 1)
	} else {
		score = clamp(existing.MasteryScore*0.7+delta*0.3, 0.0, 1.0)
		quality := DeltaToQuality(delta)
		sm2 = SM2Calculate(quality, existing.Repetitions, existing.EaseFactor, existing.IntervalDays)
	}

	nextReview := now.Add(time.Duration(sm2.IntervalDays*24) * time.Hour)

	_, err = p.pool.Exec(ctx,
		`INSERT INTO learning_progress (user_id, tenant_id, syllabus_id, topic_id, mastery_score, ease_factor, interval_days, repetitions, next_review_at, last_studied_at)
		 VALUES ($1::uuid, $2::uuid, $3, $4, $5, $6, $7, $8, $9, $10)
		 ON CONFLICT (user_id, syllabus_id, topic_id)
		 DO UPDATE SET mastery_score = $5, ease_factor = $6, interval_days = $7, repetitions = $8, next_review_at = $9, last_studied_at = $10, updated_at = NOW()`,
		learnerID.String(), p.tenantID, syllabusID, topicID, score, sm2.EaseFactor, sm2.IntervalDays, sm2.Repetitions, nextReview, now,
	)
	if err != nil {
		return err
	}

	// Record daily mastery snapshot for leaderboard history.
	_, err = p.pool.Exec(ctx,
		`INSERT INTO mastery_snapshots (user_id, tenant_id, topic_id, mastery_score, snapshot_date)
		 VALUES ($1::uuid, $2::uuid, $3, $4, CURRENT_DATE)
		 ON CONFLICT (user_id, topic_id, snapshot_date)
		 DO UPDATE SET mastery_score = $4`,
		learnerID.String(), p.tenantID, topicID, score,
	)
	return err
}

// RecordMasteryEvidence atomically deduplicates one attempt, updates mastery,
// and records the daily snapshot.
func (p *PostgresTracker) RecordMasteryEvidence(ctx context.Context, evidence MasteryEvidence) (MasteryEvidenceResult, error) {
	evidence, err := validateMasteryEvidence(evidence)
	if err != nil {
		return MasteryEvidenceResult{}, err
	}
	ctx, cancel := context.WithTimeout(ctx, dbTimeout)
	defer cancel()

	tx, err := p.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return MasteryEvidenceResult{}, err
	}
	defer func() {
		_ = tx.Rollback(context.Background())
	}()

	attemptLock := strings.Join([]string{p.tenantID, evidence.LearnerID.String(), evidence.AttemptID}, "\x1f")
	if _, err := tx.Exec(ctx, `SELECT pg_advisory_xact_lock(hashtextextended($1, 0))`, attemptLock); err != nil {
		return MasteryEvidenceResult{}, fmt.Errorf("lock mastery attempt: %w", err)
	}

	recorded, found, err := loadMasteryEvidence(ctx, tx, p.tenantID, evidence.LearnerID, evidence.AttemptID)
	if err != nil {
		return MasteryEvidenceResult{}, err
	}
	if found {
		if !sameMasteryEvidence(recorded.evidence, evidence) {
			return MasteryEvidenceResult{}, ErrMasteryEvidenceConflict
		}
		result := recorded.result
		result.Applied = false
		return result, nil
	}

	var learnerExists bool
	if err := tx.QueryRow(ctx,
		`SELECT EXISTS (
			SELECT 1
			FROM users
			WHERE id = $1::uuid
			  AND tenant_id = $2::uuid
		)`,
		evidence.LearnerID.String(), p.tenantID,
	).Scan(&learnerExists); err != nil {
		return MasteryEvidenceResult{}, fmt.Errorf("validate mastery learner: %w", err)
	}
	if !learnerExists {
		return MasteryEvidenceResult{}, fmt.Errorf("learner %q does not belong to tracker tenant", evidence.LearnerID.String())
	}

	topicLock := strings.Join([]string{
		p.tenantID,
		evidence.LearnerID.String(),
		evidence.SyllabusID,
		evidence.TopicID,
	}, "\x1f")
	if _, err := tx.Exec(ctx, `SELECT pg_advisory_xact_lock(hashtextextended($1, 1))`, topicLock); err != nil {
		return MasteryEvidenceResult{}, fmt.Errorf("lock mastery topic: %w", err)
	}

	var existing ProgressItem
	found = true
	err = tx.QueryRow(ctx,
		`SELECT mastery_score, ease_factor, interval_days, repetitions
		 FROM learning_progress
		 WHERE user_id = $1::uuid
		   AND tenant_id = $2::uuid
		   AND syllabus_id = $3
		   AND topic_id = $4
		 FOR UPDATE`,
		evidence.LearnerID.String(), p.tenantID, evidence.SyllabusID, evidence.TopicID,
	).Scan(&existing.MasteryScore, &existing.EaseFactor, &existing.IntervalDays, &existing.Repetitions)
	if err == pgx.ErrNoRows {
		found = false
	} else if err != nil {
		return MasteryEvidenceResult{}, fmt.Errorf("read mastery transition: %w", err)
	}

	var previous *ProgressItem
	if found {
		previous = &existing
	}
	now := time.Now()
	before, next := calculateMasteryTransition(
		evidence.LearnerID.String(),
		evidence.SyllabusID,
		evidence.TopicID,
		evidence.Score,
		previous,
		now,
	)

	tag, err := tx.Exec(ctx,
		`INSERT INTO learning_progress (
			user_id, tenant_id, syllabus_id, topic_id, mastery_score,
			ease_factor, interval_days, repetitions, next_review_at, last_studied_at
		)
		 VALUES ($1::uuid, $2::uuid, $3, $4, $5, $6, $7, $8, $9, $10)
		 ON CONFLICT (user_id, syllabus_id, topic_id)
		 DO UPDATE SET
			mastery_score = EXCLUDED.mastery_score,
			ease_factor = EXCLUDED.ease_factor,
			interval_days = EXCLUDED.interval_days,
			repetitions = EXCLUDED.repetitions,
			next_review_at = EXCLUDED.next_review_at,
			last_studied_at = EXCLUDED.last_studied_at,
			updated_at = NOW()
		 WHERE learning_progress.tenant_id = EXCLUDED.tenant_id`,
		evidence.LearnerID.String(), p.tenantID, evidence.SyllabusID, evidence.TopicID,
		next.MasteryScore, next.EaseFactor, next.IntervalDays, next.Repetitions,
		next.NextReviewAt, next.LastStudied,
	)
	if err != nil {
		return MasteryEvidenceResult{}, fmt.Errorf("write mastery transition: %w", err)
	}
	if tag.RowsAffected() != 1 {
		return MasteryEvidenceResult{}, fmt.Errorf("write mastery transition: tenant scope mismatch")
	}

	tag, err = tx.Exec(ctx,
		`INSERT INTO mastery_snapshots (user_id, tenant_id, topic_id, mastery_score, snapshot_date)
		 VALUES ($1::uuid, $2::uuid, $3, $4, CURRENT_DATE)
		 ON CONFLICT (user_id, topic_id, snapshot_date)
		 DO UPDATE SET mastery_score = EXCLUDED.mastery_score
		 WHERE mastery_snapshots.tenant_id = EXCLUDED.tenant_id`,
		evidence.LearnerID.String(), p.tenantID, evidence.TopicID, next.MasteryScore,
	)
	if err != nil {
		return MasteryEvidenceResult{}, fmt.Errorf("write mastery snapshot: %w", err)
	}
	if tag.RowsAffected() != 1 {
		return MasteryEvidenceResult{}, fmt.Errorf("write mastery snapshot: tenant scope mismatch")
	}

	result := MasteryEvidenceResult{
		Applied:       true,
		MasteryBefore: before,
		MasteryAfter:  next.MasteryScore,
	}
	if _, err := tx.Exec(ctx,
		`INSERT INTO learning_attempts (
			tenant_id, user_id, attempt_id, syllabus_id, topic_id,
			source_kind, source_id, source_revision, score, payload_hash,
			mastery_before, mastery_after
		)
		 VALUES ($1::uuid, $2::uuid, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)`,
		p.tenantID, evidence.LearnerID.String(), evidence.AttemptID,
		evidence.SyllabusID, evidence.TopicID, evidence.SourceKind, evidence.SourceID,
		evidence.SourceRevision, evidence.Score, evidence.PayloadHash[:],
		result.MasteryBefore, result.MasteryAfter,
	); err != nil {
		return MasteryEvidenceResult{}, fmt.Errorf("write mastery attempt: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return MasteryEvidenceResult{}, err
	}
	return result, nil
}

// GetMasteryEvidenceCounts returns durable attempt counts grouped by topic.
func (p *PostgresTracker) GetMasteryEvidenceCounts(ctx context.Context, learnerID LearnerID) ([]MasteryEvidenceCount, error) {
	if learnerID.String() == "" {
		return nil, fmt.Errorf("mastery evidence learner ID is required")
	}
	ctx, cancel := context.WithTimeout(ctx, dbTimeout)
	defer cancel()

	rows, err := p.pool.Query(ctx,
		`SELECT syllabus_id, topic_id, COUNT(*)::integer
		 FROM learning_attempts
		 WHERE tenant_id = $1::uuid
		   AND user_id = $2::uuid
		 GROUP BY syllabus_id, topic_id
		 ORDER BY syllabus_id, topic_id`,
		p.tenantID, learnerID.String(),
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var counts []MasteryEvidenceCount
	for rows.Next() {
		var count MasteryEvidenceCount
		if err := rows.Scan(&count.SyllabusID, &count.TopicID, &count.Count); err != nil {
			return nil, err
		}
		counts = append(counts, count)
	}
	return counts, rows.Err()
}

// GetMasterySnapshot returns scores and evidence counts from one PostgreSQL
// statement, so every item belongs to the same MVCC read snapshot.
func (p *PostgresTracker) GetMasterySnapshot(ctx context.Context, learnerID LearnerID) ([]MasterySnapshotItem, error) {
	if learnerID.String() == "" {
		return nil, fmt.Errorf("mastery snapshot learner ID is required")
	}
	ctx, cancel := context.WithTimeout(ctx, dbTimeout)
	defer cancel()

	rows, err := p.pool.Query(ctx,
		`WITH progress_items AS (
			SELECT syllabus_id, topic_id, mastery_score
			FROM learning_progress
			WHERE tenant_id = $1::uuid
			  AND user_id = $2::uuid
		),
		evidence_counts AS (
			SELECT syllabus_id, topic_id, COUNT(*)::integer AS evidence_count
			FROM learning_attempts
			WHERE tenant_id = $1::uuid
			  AND user_id = $2::uuid
			GROUP BY syllabus_id, topic_id
		),
		latest_evidence AS (
			SELECT DISTINCT ON (syllabus_id, topic_id)
				syllabus_id, topic_id, source_kind, source_id, score
			FROM learning_attempts
			WHERE tenant_id = $1::uuid
			  AND user_id = $2::uuid
			ORDER BY syllabus_id, topic_id, created_at DESC, attempt_id DESC
		),
		topic_keys AS (
			SELECT syllabus_id, topic_id FROM progress_items
			UNION
			SELECT syllabus_id, topic_id FROM evidence_counts
		)
		SELECT
			topic_keys.syllabus_id,
			topic_keys.topic_id,
			COALESCE(progress_items.mastery_score, 0),
			progress_items.topic_id IS NOT NULL,
			COALESCE(evidence_counts.evidence_count, 0),
			latest_evidence.source_kind,
			latest_evidence.source_id,
			latest_evidence.score
		FROM topic_keys
		LEFT JOIN progress_items USING (syllabus_id, topic_id)
		LEFT JOIN evidence_counts USING (syllabus_id, topic_id)
		LEFT JOIN latest_evidence USING (syllabus_id, topic_id)
		ORDER BY 1, 2`,
		p.tenantID, learnerID.String(),
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var snapshot []MasterySnapshotItem
	for rows.Next() {
		var item MasterySnapshotItem
		var latestSourceKind, latestSourceID *string
		var latestScore *float64
		if err := rows.Scan(
			&item.SyllabusID,
			&item.TopicID,
			&item.MasteryScore,
			&item.MasteryKnown,
			&item.EvidenceCount,
			&latestSourceKind,
			&latestSourceID,
			&latestScore,
		); err != nil {
			return nil, err
		}
		if latestSourceKind != nil && latestSourceID != nil && latestScore != nil {
			item.LatestEvidence = &MasteryEvidenceSummary{
				SourceKind: *latestSourceKind,
				SourceID:   *latestSourceID,
				Score:      *latestScore,
			}
		}
		snapshot = append(snapshot, item)
	}
	return snapshot, rows.Err()
}

func loadMasteryEvidence(
	ctx context.Context,
	tx pgx.Tx,
	tenantID string,
	learnerID LearnerID,
	attemptID string,
) (recordedMasteryEvidence, bool, error) {
	var recorded recordedMasteryEvidence
	var payloadHash []byte
	err := tx.QueryRow(ctx,
		`SELECT syllabus_id, topic_id, source_kind, source_id, source_revision, score,
		        payload_hash, mastery_before, mastery_after
		 FROM learning_attempts
		 WHERE tenant_id = $1::uuid
		   AND user_id = $2::uuid
		   AND attempt_id = $3`,
		tenantID, learnerID.String(), attemptID,
	).Scan(
		&recorded.evidence.SyllabusID,
		&recorded.evidence.TopicID,
		&recorded.evidence.SourceKind,
		&recorded.evidence.SourceID,
		&recorded.evidence.SourceRevision,
		&recorded.evidence.Score,
		&payloadHash,
		&recorded.result.MasteryBefore,
		&recorded.result.MasteryAfter,
	)
	if err == pgx.ErrNoRows {
		return recordedMasteryEvidence{}, false, nil
	}
	if err != nil {
		return recordedMasteryEvidence{}, false, fmt.Errorf("read mastery attempt: %w", err)
	}
	if len(payloadHash) != len(recorded.evidence.PayloadHash) {
		return recordedMasteryEvidence{}, false, fmt.Errorf("stored mastery attempt has invalid payload hash")
	}
	recorded.evidence.AttemptID = attemptID
	recorded.evidence.LearnerID = learnerID
	copy(recorded.evidence.PayloadHash[:], payloadHash)
	recorded.result.Applied = true
	return recorded, true, nil
}

// SetMastery directly sets a topic's mastery score (dev/testing only).
func (p *PostgresTracker) SetMastery(userID, syllabusID, topicID string, score float64) error {
	learnerID, err := p.resolveLegacyLearnerID(userID)
	if err != nil {
		return err
	}
	return p.SetMasteryForLearner(learnerID, syllabusID, topicID, score)
}

func (p *PostgresTracker) SetMasteryForLearner(learnerID LearnerID, syllabusID, topicID string, score float64) error {
	ctx, cancel := context.WithTimeout(context.Background(), dbTimeout)
	defer cancel()

	now := time.Now()
	_, err := p.pool.Exec(ctx,
		`INSERT INTO learning_progress (user_id, tenant_id, syllabus_id, topic_id, mastery_score, last_studied_at)
		 VALUES ($1::uuid, $2::uuid, $3, $4, $5, $6)
		 ON CONFLICT (user_id, syllabus_id, topic_id)
		 DO UPDATE SET mastery_score = $5, last_studied_at = $6, updated_at = NOW()`,
		learnerID.String(), p.tenantID, syllabusID, topicID, score, now,
	)
	return err
}

func (p *PostgresTracker) GetMastery(userID, syllabusID, topicID string) (float64, error) {
	learnerID, err := p.resolveLegacyLearnerID(userID)
	if err != nil {
		return 0, err
	}
	return p.GetMasteryForLearner(learnerID, syllabusID, topicID)
}

func (p *PostgresTracker) GetMasteryForLearner(learnerID LearnerID, syllabusID, topicID string) (float64, error) {
	ctx, cancel := context.WithTimeout(context.Background(), dbTimeout)
	defer cancel()

	var score float64
	err := p.pool.QueryRow(ctx,
		`SELECT lp.mastery_score
		 FROM learning_progress lp
		 WHERE lp.user_id = $1::uuid
		   AND lp.tenant_id = $2::uuid
		   AND lp.syllabus_id = $3
		   AND lp.topic_id = $4`,
		learnerID.String(), p.tenantID, syllabusID, topicID,
	).Scan(&score)

	if err == pgx.ErrNoRows {
		return 0, nil
	}
	return score, err
}

func (p *PostgresTracker) GetAllProgress(userID string) ([]ProgressItem, error) {
	learnerID, err := p.resolveLegacyLearnerID(userID)
	if err != nil {
		return nil, err
	}
	return p.GetAllProgressForLearner(learnerID)
}

func (p *PostgresTracker) GetAllProgressForLearner(learnerID LearnerID) ([]ProgressItem, error) {
	ctx, cancel := context.WithTimeout(context.Background(), dbTimeout)
	defer cancel()

	rows, err := p.pool.Query(ctx,
		`SELECT lp.syllabus_id, lp.topic_id, lp.mastery_score, lp.ease_factor, lp.interval_days, lp.repetitions, lp.next_review_at, lp.last_studied_at
		 FROM learning_progress lp
		 WHERE lp.user_id = $1::uuid
		   AND lp.tenant_id = $2::uuid
		 ORDER BY lp.last_studied_at DESC`,
		learnerID.String(), p.tenantID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []ProgressItem
	for rows.Next() {
		var item ProgressItem
		item.UserID = learnerID.String()
		if err := rows.Scan(&item.SyllabusID, &item.TopicID, &item.MasteryScore, &item.EaseFactor, &item.IntervalDays, &item.Repetitions, &item.NextReviewAt, &item.LastStudied); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (p *PostgresTracker) GetDueReviews(userID string) ([]ProgressItem, error) {
	learnerID, err := p.resolveLegacyLearnerID(userID)
	if err != nil {
		return nil, err
	}
	return p.GetDueReviewsForLearner(learnerID)
}

func (p *PostgresTracker) GetDueReviewsForLearner(learnerID LearnerID) ([]ProgressItem, error) {
	ctx, cancel := context.WithTimeout(context.Background(), dbTimeout)
	defer cancel()

	rows, err := p.pool.Query(ctx,
		`SELECT lp.syllabus_id, lp.topic_id, lp.mastery_score, lp.ease_factor, lp.interval_days, lp.repetitions, lp.next_review_at, lp.last_studied_at
		 FROM learning_progress lp
		 WHERE lp.user_id = $1::uuid
		   AND lp.tenant_id = $2::uuid
		   AND lp.next_review_at <= NOW()
		 ORDER BY lp.next_review_at ASC`,
		learnerID.String(), p.tenantID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []ProgressItem
	for rows.Next() {
		var item ProgressItem
		item.UserID = learnerID.String()
		if err := rows.Scan(&item.SyllabusID, &item.TopicID, &item.MasteryScore, &item.EaseFactor, &item.IntervalDays, &item.Repetitions, &item.NextReviewAt, &item.LastStudied); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (p *PostgresTracker) resolveLegacyLearnerID(externalID string) (LearnerID, error) {
	ctx, cancel := context.WithTimeout(context.Background(), dbTimeout)
	defer cancel()

	var userID string
	err := p.pool.QueryRow(ctx,
		targetUserCTE+` SELECT id::text FROM target_user`,
		externalID,
		p.tenantID,
		p.channel,
	).Scan(&userID)
	if err != nil {
		return LearnerID{}, err
	}
	return NewLearnerID(userID)
}
