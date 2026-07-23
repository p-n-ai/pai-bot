// Copyright 2026 the P&AI authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package focusedpagedelivery

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type PostgresStore struct{ pool *pgxpool.Pool }

func NewPostgresStore(pool *pgxpool.Pool) *PostgresStore {
	return &PostgresStore{pool: pool}
}

func (s *PostgresStore) Enqueue(ctx context.Context, input EnqueueInput, now time.Time) (Delivery, error) {
	if s.pool == nil {
		return Delivery{}, fmt.Errorf("focused-page delivery pool is nil")
	}
	if err := validateInput(input); err != nil {
		return Delivery{}, err
	}
	_, err := s.pool.Exec(ctx, `
		INSERT INTO focused_page_deliveries (
			tenant_id, turn_id, channel, recipient_id, final_text,
			focused_page_public_id, status, next_attempt_at, created_at, updated_at
		) VALUES ($1::uuid, $2, $3, $4, $5, $6, 'pending', $7, $7, $7)
		ON CONFLICT (tenant_id, turn_id, channel) DO NOTHING`,
		input.TenantID, input.TurnID, input.Channel, input.RecipientID, input.FinalText,
		input.FocusedPagePublicID, now)
	if err != nil {
		return Delivery{}, fmt.Errorf("enqueue focused-page delivery: %w", err)
	}
	delivery, err := s.getByKey(ctx, input.TenantID, input.TurnID, input.Channel)
	if err != nil {
		return Delivery{}, err
	}
	return delivery, nil
}

func (s *PostgresStore) Claim(ctx context.Context, id, token string, now, leaseExpiry time.Time) (Delivery, bool, error) {
	return s.claim(ctx, `id = $1::uuid`, []any{id}, token, now, leaseExpiry)
}

func (s *PostgresStore) ClaimDue(ctx context.Context, token string, now, leaseExpiry time.Time) (Delivery, bool, error) {
	return s.claim(ctx, `TRUE`, nil, token, now, leaseExpiry)
}

func (s *PostgresStore) claim(ctx context.Context, selector string, selectorArgs []any, token string, now, leaseExpiry time.Time) (Delivery, bool, error) {
	args := append(selectorArgs, token, now, leaseExpiry)
	tokenArg := len(selectorArgs) + 1
	nowArg := tokenArg + 1
	expiryArg := nowArg + 1
	query := fmt.Sprintf(`
		WITH candidate AS (
			SELECT id
			FROM focused_page_deliveries
			WHERE %s
			  AND (
				(status = 'pending' AND next_attempt_at <= $%d)
				OR (status = 'leased' AND lease_expires_at <= $%d)
			  )
			ORDER BY next_attempt_at, created_at
			FOR UPDATE SKIP LOCKED
			LIMIT 1
		)
		UPDATE focused_page_deliveries AS delivery
		SET status = 'leased', lease_token = $%d, lease_expires_at = $%d, updated_at = $%d
		FROM candidate
		WHERE delivery.id = candidate.id
		RETURNING delivery.id::text, delivery.tenant_id::text, delivery.turn_id,
		          delivery.channel, delivery.recipient_id, delivery.final_text,
		          delivery.focused_page_public_id, delivery.status, delivery.attempt_count,
		          delivery.next_attempt_at, delivery.lease_token, delivery.lease_expires_at,
		          delivery.delivered_at, delivery.created_at, delivery.updated_at`,
		selector, nowArg, nowArg, tokenArg, expiryArg, nowArg)
	delivery, err := scanDelivery(s.pool.QueryRow(ctx, query, args...))
	if errors.Is(err, pgx.ErrNoRows) {
		return Delivery{}, false, nil
	}
	if err != nil {
		return Delivery{}, false, fmt.Errorf("claim focused-page delivery: %w", err)
	}
	return delivery, true, nil
}

func (s *PostgresStore) MarkDelivered(ctx context.Context, id, token string, now time.Time) error {
	tag, err := s.pool.Exec(ctx, `
		UPDATE focused_page_deliveries
		SET status = 'delivered', lease_token = NULL, lease_expires_at = NULL,
		    delivered_at = $3, updated_at = $3
		WHERE id = $1::uuid AND status = 'leased' AND lease_token = $2`,
		id, token, now)
	if err != nil {
		return fmt.Errorf("mark focused-page delivery delivered: %w", err)
	}
	if tag.RowsAffected() != 1 {
		return ErrLeaseLost
	}
	return nil
}

func (s *PostgresStore) ScheduleRetry(ctx context.Context, id, token string, nextAttempt, now time.Time) error {
	tag, err := s.pool.Exec(ctx, `
		UPDATE focused_page_deliveries
		SET status = 'pending', attempt_count = attempt_count + 1,
		    next_attempt_at = $3, lease_token = NULL, lease_expires_at = NULL, updated_at = $4
		WHERE id = $1::uuid AND status = 'leased' AND lease_token = $2`,
		id, token, nextAttempt, now)
	if err != nil {
		return fmt.Errorf("schedule focused-page delivery retry: %w", err)
	}
	if tag.RowsAffected() != 1 {
		return ErrLeaseLost
	}
	return nil
}

func (s *PostgresStore) getByKey(ctx context.Context, tenantID, turnID, channel string) (Delivery, error) {
	delivery, err := scanDelivery(s.pool.QueryRow(ctx, `
		SELECT id::text, tenant_id::text, turn_id, channel, recipient_id, final_text,
		       focused_page_public_id, status, attempt_count, next_attempt_at,
		       COALESCE(lease_token, ''), lease_expires_at, delivered_at, created_at, updated_at
		FROM focused_page_deliveries
		WHERE tenant_id = $1::uuid AND turn_id = $2 AND channel = $3`,
		tenantID, turnID, channel))
	if err != nil {
		return Delivery{}, fmt.Errorf("get focused-page delivery: %w", err)
	}
	return delivery, nil
}

type rowScanner interface {
	Scan(...any) error
}

func scanDelivery(row rowScanner) (Delivery, error) {
	var delivery Delivery
	err := row.Scan(
		&delivery.ID, &delivery.TenantID, &delivery.TurnID, &delivery.Channel,
		&delivery.RecipientID, &delivery.FinalText, &delivery.FocusedPagePublicID,
		&delivery.Status, &delivery.AttemptCount, &delivery.NextAttemptAt,
		&delivery.LeaseToken, &delivery.LeaseExpiresAt, &delivery.DeliveredAt,
		&delivery.CreatedAt, &delivery.UpdatedAt,
	)
	return delivery, err
}
