package progress

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

const dbTimeout = 5 * time.Second

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
	delta = clamp(delta, 0.0, 1.0)

	ctx, cancel := context.WithTimeout(context.Background(), dbTimeout)
	defer cancel()

	// Try to get existing item.
	var existing ProgressItem
	var found bool
	err := p.pool.QueryRow(ctx,
		`SELECT mastery_score, ease_factor, interval_days, repetitions
		 FROM learning_progress
		 WHERE user_id = (SELECT id FROM users WHERE external_id = $1 AND tenant_id = $2)
		   AND syllabus_id = $3 AND topic_id = $4`,
		userID, p.tenantID, syllabusID, topicID,
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
		 VALUES ((SELECT id FROM users WHERE external_id = $1 AND tenant_id = $2), $2, $3, $4, $5, $6, $7, $8, $9, $10)
		 ON CONFLICT (user_id, syllabus_id, topic_id)
		 DO UPDATE SET mastery_score = $5, ease_factor = $6, interval_days = $7, repetitions = $8, next_review_at = $9, last_studied_at = $10, updated_at = NOW()`,
		userID, p.tenantID, syllabusID, topicID, score, sm2.EaseFactor, sm2.IntervalDays, sm2.Repetitions, nextReview, now,
	)
	if err != nil {
		return err
	}

	// Record daily mastery snapshot for leaderboard history.
	_, err = p.pool.Exec(ctx,
		`INSERT INTO mastery_snapshots (user_id, tenant_id, topic_id, mastery_score, snapshot_date)
		 VALUES ((SELECT id FROM users WHERE external_id = $1 AND tenant_id = $2), $2, $3, $4, CURRENT_DATE)
		 ON CONFLICT (user_id, topic_id, snapshot_date)
		 DO UPDATE SET mastery_score = $4`,
		userID, p.tenantID, topicID, score,
	)
	return err
}

// SetMastery directly sets a topic's mastery score (dev/testing only).
func (p *PostgresTracker) SetMastery(userID, syllabusID, topicID string, score float64) error {
	ctx, cancel := context.WithTimeout(context.Background(), dbTimeout)
	defer cancel()

	now := time.Now()
	_, err := p.pool.Exec(ctx,
		`INSERT INTO learning_progress (user_id, tenant_id, syllabus_id, topic_id, mastery_score, last_studied_at)
		 VALUES ((SELECT id FROM users WHERE external_id = $1 AND tenant_id = $2), $2, $3, $4, $5, $6)
		 ON CONFLICT (user_id, syllabus_id, topic_id)
		 DO UPDATE SET mastery_score = $5, last_studied_at = $6, updated_at = NOW()`,
		userID, p.tenantID, syllabusID, topicID, score, now,
	)
	return err
}

func (p *PostgresTracker) GetMastery(userID, syllabusID, topicID string) (float64, error) {
	ctx, cancel := context.WithTimeout(context.Background(), dbTimeout)
	defer cancel()

	var score float64
	err := p.pool.QueryRow(ctx,
		`SELECT mastery_score FROM learning_progress
		 WHERE user_id = (SELECT id FROM users WHERE external_id = $1 AND tenant_id = $2)
		   AND syllabus_id = $3 AND topic_id = $4`,
		userID, p.tenantID, syllabusID, topicID,
	).Scan(&score)

	if err == pgx.ErrNoRows {
		return 0, nil
	}
	return score, err
}

func (p *PostgresTracker) GetAllProgress(userID string) ([]ProgressItem, error) {
	ctx, cancel := context.WithTimeout(context.Background(), dbTimeout)
	defer cancel()

	rows, err := p.pool.Query(ctx,
		`SELECT syllabus_id, topic_id, mastery_score, ease_factor, interval_days, repetitions, next_review_at, last_studied_at
		 FROM learning_progress
		 WHERE user_id = (SELECT id FROM users WHERE external_id = $1 AND tenant_id = $2)
		 ORDER BY last_studied_at DESC`,
		userID, p.tenantID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []ProgressItem
	for rows.Next() {
		var item ProgressItem
		item.UserID = userID
		if err := rows.Scan(&item.SyllabusID, &item.TopicID, &item.MasteryScore, &item.EaseFactor, &item.IntervalDays, &item.Repetitions, &item.NextReviewAt, &item.LastStudied); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (p *PostgresTracker) GetDueReviews(userID string) ([]ProgressItem, error) {
	ctx, cancel := context.WithTimeout(context.Background(), dbTimeout)
	defer cancel()

	rows, err := p.pool.Query(ctx,
		`SELECT syllabus_id, topic_id, mastery_score, ease_factor, interval_days, repetitions, next_review_at, last_studied_at
		 FROM learning_progress
		 WHERE user_id = (SELECT id FROM users WHERE external_id = $1 AND tenant_id = $2)
		   AND next_review_at <= NOW()
		 ORDER BY next_review_at ASC`,
		userID, p.tenantID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []ProgressItem
	for rows.Next() {
		var item ProgressItem
		item.UserID = userID
		if err := rows.Scan(&item.SyllabusID, &item.TopicID, &item.MasteryScore, &item.EaseFactor, &item.IntervalDays, &item.Repetitions, &item.NextReviewAt, &item.LastStudied); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}
