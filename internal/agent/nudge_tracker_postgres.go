// Copyright 2026 the P&AI authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package agent

import (
	"context"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/p-n-ai/pai-bot/internal/progress"
)

// PostgresNudgeTracker persists nudge counts and sends in PostgreSQL.
type PostgresNudgeTracker struct {
	pool     *pgxpool.Pool
	tenantID string
	channel  string
}

const nudgeDayTimeZone = "Asia/Kuala_Lumpur"

// NewPostgresNudgeTracker creates a PostgreSQL-backed nudge tracker.
func NewPostgresNudgeTracker(pool *pgxpool.Pool, tenantID string) *PostgresNudgeTracker {
	return NewPostgresNudgeTrackerForChannel(pool, tenantID, defaultChannel)
}

// NewPostgresNudgeTrackerForChannel creates a legacy external-ID adapter scoped
// to one provider; learner-ID methods bypass external-ID resolution entirely.
func NewPostgresNudgeTrackerForChannel(
	pool *pgxpool.Pool,
	tenantID string,
	channel string,
) *PostgresNudgeTracker {
	channel = strings.TrimSpace(channel)
	if channel == "" {
		channel = defaultChannel
	}
	return &PostgresNudgeTracker{
		pool:     pool,
		tenantID: tenantID,
		channel:  channel,
	}
}

func (t *PostgresNudgeTracker) NudgeCountToday(userID string) (int, error) {
	ctx, cancel := context.WithTimeout(context.Background(), dbTimeout)
	defer cancel()

	learnerID, err := t.resolveUserID(ctx, userID)
	if err != nil {
		return 0, err
	}
	refinedLearnerID, err := progress.NewLearnerID(learnerID)
	if err != nil {
		return 0, err
	}
	return t.nudgeCountTodayForLearner(ctx, refinedLearnerID)
}

func (t *PostgresNudgeTracker) NudgeCountTodayForLearner(learnerID progress.LearnerID) (int, error) {
	ctx, cancel := context.WithTimeout(context.Background(), dbTimeout)
	defer cancel()
	return t.nudgeCountTodayForLearner(ctx, learnerID)
}

func (t *PostgresNudgeTracker) nudgeCountTodayForLearner(
	ctx context.Context,
	learnerID progress.LearnerID,
) (int, error) {
	var count int
	query, args := buildNudgeCountTodayForLearnerQuery(t.tenantID, learnerID.String())
	err := t.pool.QueryRow(ctx, query, args...).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("count nudges today: %w", err)
	}

	return count, nil
}

func buildNudgeCountTodayForLearnerQuery(tenantID, learnerID string) (string, []any) {
	return `SELECT COUNT(*)
		 FROM nudge_log nl
		 WHERE nl.tenant_id = $1::uuid
		   AND nl.user_id = $2::uuid
		   AND nl.sent_at >= date_trunc('day', NOW() AT TIME ZONE $3) AT TIME ZONE $3
		   AND nl.sent_at < (date_trunc('day', NOW() AT TIME ZONE $3) + INTERVAL '1 day') AT TIME ZONE $3`,
		[]any{tenantID, learnerID, nudgeDayTimeZone}
}

func (t *PostgresNudgeTracker) RecordNudge(userID, nudgeType, topicID string) error {
	ctx, cancel := context.WithTimeout(context.Background(), dbTimeout)
	defer cancel()

	dbUserID, err := t.resolveUserID(ctx, userID)
	if err != nil {
		return err
	}

	learnerID, err := progress.NewLearnerID(dbUserID)
	if err != nil {
		return err
	}
	return t.recordNudgeForLearner(ctx, learnerID, nudgeType, topicID)
}

func (t *PostgresNudgeTracker) RecordNudgeForLearner(
	learnerID progress.LearnerID,
	nudgeType,
	topicID string,
) error {
	ctx, cancel := context.WithTimeout(context.Background(), dbTimeout)
	defer cancel()
	return t.recordNudgeForLearner(ctx, learnerID, nudgeType, topicID)
}

func (t *PostgresNudgeTracker) recordNudgeForLearner(
	ctx context.Context,
	learnerID progress.LearnerID,
	nudgeType,
	topicID string,
) error {
	_, err := t.pool.Exec(ctx,
		`INSERT INTO nudge_log (user_id, tenant_id, nudge_type, topic_id)
		 VALUES ($1::uuid, $2::uuid, $3, $4)`,
		learnerID.String(),
		t.tenantID,
		nudgeType,
		nullIfEmpty(topicID),
	)
	if err != nil {
		return fmt.Errorf("record nudge: %w", err)
	}

	return nil
}

func (t *PostgresNudgeTracker) resolveUserID(ctx context.Context, externalID string) (string, error) {
	var dbUserID string
	err := t.pool.QueryRow(ctx,
		`SELECT id::text
		 FROM users
		 WHERE tenant_id = $1::uuid
		   AND channel = $2
		   AND external_id = $3
		 ORDER BY created_at ASC
		 LIMIT 1`,
		t.tenantID,
		t.channel,
		externalID,
	).Scan(&dbUserID)
	if err == nil {
		return dbUserID, nil
	}
	if err == pgx.ErrNoRows {
		return "", fmt.Errorf("resolve user for nudge %q: %w", externalID, err)
	}
	return "", fmt.Errorf("resolve user for nudge %q: %w", externalID, err)
}
