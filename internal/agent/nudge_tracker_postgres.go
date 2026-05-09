// Copyright 2026 the P&AI authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package agent

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// PostgresNudgeTracker persists nudge counts and sends in PostgreSQL.
type PostgresNudgeTracker struct {
	pool     *pgxpool.Pool
	tenantID string
}

const nudgeDayTimeZone = "Asia/Kuala_Lumpur"

// NewPostgresNudgeTracker creates a PostgreSQL-backed nudge tracker.
func NewPostgresNudgeTracker(pool *pgxpool.Pool, tenantID string) *PostgresNudgeTracker {
	return &PostgresNudgeTracker{
		pool:     pool,
		tenantID: tenantID,
	}
}

func (t *PostgresNudgeTracker) NudgeCountToday(userID string) (int, error) {
	ctx, cancel := context.WithTimeout(context.Background(), dbTimeout)
	defer cancel()

	var count int
	query, args := buildNudgeCountTodayQuery(t.tenantID, userID)
	err := t.pool.QueryRow(ctx, query, args...).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("count nudges today: %w", err)
	}

	return count, nil
}

func buildNudgeCountTodayQuery(tenantID, userID string) (string, []any) {
	return `WITH target_user AS (
			SELECT id
			FROM users
			WHERE tenant_id = $1::uuid
			  AND external_id = $2
			ORDER BY created_at ASC
			LIMIT 1
		 )
		 SELECT COUNT(*)
		 FROM nudge_log nl
		 JOIN target_user u ON u.id = nl.user_id
		 WHERE nl.tenant_id = $1::uuid
		   AND nl.sent_at >= date_trunc('day', NOW() AT TIME ZONE $3) AT TIME ZONE $3
		   AND nl.sent_at < (date_trunc('day', NOW() AT TIME ZONE $3) + INTERVAL '1 day') AT TIME ZONE $3`,
		[]any{tenantID, userID, nudgeDayTimeZone}
}

func (t *PostgresNudgeTracker) RecordNudge(userID, nudgeType, topicID string) error {
	ctx, cancel := context.WithTimeout(context.Background(), dbTimeout)
	defer cancel()

	dbUserID, err := t.resolveUserID(ctx, userID)
	if err != nil {
		return err
	}

	_, err = t.pool.Exec(ctx,
		`INSERT INTO nudge_log (user_id, tenant_id, nudge_type, topic_id)
		 VALUES ($1::uuid, $2::uuid, $3, $4)`,
		dbUserID,
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
		   AND external_id = $2
		 ORDER BY created_at ASC
		 LIMIT 1`,
		t.tenantID,
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
