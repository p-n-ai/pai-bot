// Copyright 2026 the P&AI authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package progress

import (
	"context"
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
	ORDER BY created_at ASC
	LIMIT 1
)`

// PostgresTracker is a PostgreSQL-backed implementation of Tracker.
type PostgresTracker struct {
	pool     *pgxpool.Pool
	tenantID string
}

// NewPostgresTracker creates a new PostgreSQL-backed tracker.
// tenantID is the UUID of the tenant for row-level isolation.
func NewPostgresTracker(pool *pgxpool.Pool, tenantID string) *PostgresTracker {
	return &PostgresTracker{pool: pool, tenantID: tenantID}
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
	).Scan(&userID)
	if err != nil {
		return LearnerID{}, err
	}
	return NewLearnerID(userID)
}
