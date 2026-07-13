// Copyright 2026 the P&AI authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package focusedpage

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type PostgresDeliveryStore struct{ pool *pgxpool.Pool }

func NewPostgresDeliveryStore(pool *pgxpool.Pool) *PostgresDeliveryStore {
	return &PostgresDeliveryStore{pool: pool}
}

func (s *PostgresDeliveryStore) Enqueue(ctx context.Context, delivery Delivery) error {
	if s.pool == nil {
		return fmt.Errorf("focused page delivery pool is nil")
	}
	cmd, err := s.pool.Exec(ctx, `
		INSERT INTO focused_page_deliveries (
			tenant_id, page_public_id, turn_id, channel, recipient_external_id,
			tutor_text, status, next_attempt_at, expires_at
		)
		SELECT p.tenant_id, p.public_id, p.turn_id, $4, $5, $6, 'pending', $7, p.expires_at
		FROM focused_pages p
		WHERE p.tenant_id = $1::uuid AND p.public_id = $2 AND p.turn_id = $3
		  AND p.status = 'active' AND p.expires_at = $8
		ON CONFLICT (tenant_id, page_public_id) DO NOTHING`,
		delivery.TenantID, delivery.PublicID, delivery.TurnID, delivery.Channel, delivery.RecipientID,
		delivery.TutorText, delivery.NextAttempt, delivery.ExpiresAt)
	if err != nil {
		return fmt.Errorf("enqueue focused page delivery: %w", err)
	}
	if cmd.RowsAffected() == 1 {
		return nil
	}
	existing, err := s.get(ctx, delivery.TenantID, delivery.PublicID)
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrForbidden
	}
	if err != nil {
		return err
	}
	if !sameDeliveryPayload(existing, delivery) {
		return ErrDeliveryConflict
	}
	return nil
}

func (s *PostgresDeliveryStore) ClaimDue(ctx context.Context, now time.Time, lease time.Duration, limit int) ([]Delivery, error) {
	if limit <= 0 {
		return nil, nil
	}
	if _, err := s.pool.Exec(ctx, `
		UPDATE focused_page_deliveries d
		SET status = CASE WHEN p.status = 'revoked' THEN 'cancelled' ELSE 'expired' END, updated_at = $1
		FROM focused_pages p
		WHERE d.tenant_id = p.tenant_id AND d.page_public_id = p.public_id AND d.status = 'pending'
		  AND (p.status <> 'active' OR d.expires_at <= $1)`, now); err != nil {
		return nil, fmt.Errorf("expire focused page deliveries: %w", err)
	}
	rows, err := s.pool.Query(ctx, `
		WITH candidates AS (
			SELECT d.tenant_id, d.page_public_id
			FROM focused_page_deliveries d
			JOIN focused_pages p ON p.tenant_id = d.tenant_id AND p.public_id = d.page_public_id
			WHERE d.status = 'pending' AND d.next_attempt_at <= $1 AND d.expires_at > $1 AND p.status = 'active'
			ORDER BY d.next_attempt_at, d.created_at
			FOR UPDATE OF d SKIP LOCKED
			LIMIT $2
		)
		UPDATE focused_page_deliveries d
		SET attempt_count = d.attempt_count + 1, last_attempt_at = $1,
		    next_attempt_at = $1 + ($3 * interval '1 millisecond'), updated_at = $1
		FROM candidates c
		WHERE d.tenant_id = c.tenant_id AND d.page_public_id = c.page_public_id
		RETURNING d.tenant_id::text, d.page_public_id, d.turn_id, d.channel,
		          d.recipient_external_id, d.tutor_text, d.status, d.attempt_count,
		          d.next_attempt_at, d.expires_at`, now, limit, lease.Milliseconds())
	if err != nil {
		return nil, fmt.Errorf("claim focused page deliveries: %w", err)
	}
	defer rows.Close()
	deliveries := make([]Delivery, 0, limit)
	for rows.Next() {
		var delivery Delivery
		if err := rows.Scan(&delivery.TenantID, &delivery.PublicID, &delivery.TurnID, &delivery.Channel,
			&delivery.RecipientID, &delivery.TutorText, &delivery.Status, &delivery.Attempts,
			&delivery.NextAttempt, &delivery.ExpiresAt); err != nil {
			return nil, fmt.Errorf("scan focused page delivery: %w", err)
		}
		deliveries = append(deliveries, delivery)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate focused page deliveries: %w", err)
	}
	return deliveries, nil
}

func (s *PostgresDeliveryStore) MarkSent(ctx context.Context, tenantID, publicID string, now time.Time) error {
	cmd, err := s.pool.Exec(ctx, `UPDATE focused_page_deliveries
		SET status = 'sent', sent_at = $3, updated_at = $3
		WHERE tenant_id = $1::uuid AND page_public_id = $2 AND status = 'pending'`, tenantID, publicID, now)
	if err != nil {
		return fmt.Errorf("mark focused page delivery sent: %w", err)
	}
	if cmd.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (s *PostgresDeliveryStore) Reschedule(ctx context.Context, tenantID, publicID string, nextAttempt, now time.Time) error {
	cmd, err := s.pool.Exec(ctx, `UPDATE focused_page_deliveries
		SET status = CASE WHEN expires_at <= $4 OR expires_at <= $3 THEN 'expired' ELSE 'pending' END,
		    next_attempt_at = LEAST($3, expires_at), updated_at = $4
		WHERE tenant_id = $1::uuid AND page_public_id = $2 AND status = 'pending'`,
		tenantID, publicID, nextAttempt, now)
	if err != nil {
		return fmt.Errorf("reschedule focused page delivery: %w", err)
	}
	if cmd.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (s *PostgresDeliveryStore) get(ctx context.Context, tenantID, publicID string) (Delivery, error) {
	var delivery Delivery
	err := s.pool.QueryRow(ctx, `SELECT tenant_id::text, page_public_id, turn_id, channel,
		recipient_external_id, tutor_text, status, attempt_count, next_attempt_at, expires_at
		FROM focused_page_deliveries WHERE tenant_id = $1::uuid AND page_public_id = $2`, tenantID, publicID).
		Scan(&delivery.TenantID, &delivery.PublicID, &delivery.TurnID, &delivery.Channel,
			&delivery.RecipientID, &delivery.TutorText, &delivery.Status, &delivery.Attempts,
			&delivery.NextAttempt, &delivery.ExpiresAt)
	return delivery, err
}
