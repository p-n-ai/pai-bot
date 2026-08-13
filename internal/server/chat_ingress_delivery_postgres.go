// Copyright 2026 the P&AI authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package server

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/p-n-ai/pai-bot/internal/agent"
	"github.com/p-n-ai/pai-bot/internal/chat"
)

type postgresInboundDeliveryStore struct {
	pool *pgxpool.Pool
}

func NewPostgresChatIngress(
	defaultTenantID string,
	pool *pgxpool.Pool,
	processor InboundTurnProcessor,
) (*ChatIngress, error) {
	return newChatIngressWithDefaults(
		defaultTenantID,
		newPostgresInboundDeliveryStore(pool),
		processor,
	)
}

func newPostgresInboundDeliveryStore(pool *pgxpool.Pool) *postgresInboundDeliveryStore {
	return &postgresInboundDeliveryStore{pool: pool}
}

func (s *postgresInboundDeliveryStore) Accept(
	ctx context.Context,
	input inboundDeliveryAcceptInput,
	now time.Time,
) (inboundDelivery, bool, error) {
	if s.pool == nil {
		return inboundDelivery{}, false, errors.New("inbound delivery pool is nil")
	}
	if err := validateInboundDeliveryAcceptInput(input); err != nil {
		return inboundDelivery{}, false, err
	}
	payload, err := json.Marshal(newInboundMessagePayload(input.Message))
	if err != nil {
		return inboundDelivery{}, false, fmt.Errorf("encode inbound delivery: %w", err)
	}
	payloadHash := sha256.Sum256(payload)
	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return inboundDelivery{}, false, fmt.Errorf("begin inbound delivery acceptance: %w", err)
	}
	defer func() { _ = tx.Rollback(context.Background()) }()

	lockKeys := []string{
		input.TenantID + ":learner:" + input.LearnerKey,
		input.TenantID + ":destination:" + input.DestinationKey,
	}
	sort.Strings(lockKeys)
	for _, lockKey := range lockKeys {
		if _, err := tx.Exec(ctx, `SELECT pg_advisory_xact_lock(hashtextextended($1, 0))`, lockKey); err != nil {
			return inboundDelivery{}, false, fmt.Errorf("lock inbound delivery ordering: %w", err)
		}
	}

	delivery, err := scanInboundDelivery(tx.QueryRow(ctx, `
		INSERT INTO chat_inbound_deliveries (
			tenant_id, channel, delivery_id, learner_key, destination_key,
			inbound_payload, inbound_payload_hash, next_attempt_at, created_at, updated_at
		) VALUES ($1::uuid, $2, $3, $4, $5, $6::jsonb, $7, $8, $8, $8)
		ON CONFLICT (tenant_id, channel, delivery_id) DO NOTHING
		RETURNING `+inboundDeliveryColumns,
		input.TenantID, input.Channel, input.DeliveryID, input.LearnerKey,
		input.DestinationKey, payload, payloadHash[:], now))
	inserted := true
	if errors.Is(err, pgx.ErrNoRows) {
		inserted = false
		delivery, err = scanInboundDelivery(tx.QueryRow(ctx, `
			SELECT `+inboundDeliveryColumns+`
			FROM chat_inbound_deliveries
			WHERE tenant_id = $1::uuid AND channel = $2 AND delivery_id = $3`,
			input.TenantID, input.Channel, input.DeliveryID))
	}
	if err != nil {
		return inboundDelivery{}, false, fmt.Errorf("accept inbound delivery: %w", err)
	}
	var storedHash []byte
	if err := tx.QueryRow(ctx, `
		SELECT inbound_payload_hash
		FROM chat_inbound_deliveries
		WHERE id = $1::uuid`, delivery.ID).Scan(&storedHash); err != nil {
		return inboundDelivery{}, false, fmt.Errorf("read inbound delivery identity: %w", err)
	}
	if !bytes.Equal(storedHash, payloadHash[:]) {
		return inboundDelivery{}, false, ErrInboundDeliveryConflict
	}
	if err := tx.Commit(ctx); err != nil {
		return inboundDelivery{}, false, fmt.Errorf("commit inbound delivery acceptance: %w", err)
	}
	return delivery, inserted, nil
}

func (s *postgresInboundDeliveryStore) ClaimDue(
	ctx context.Context,
	token string,
	now time.Time,
	leaseExpiresAt time.Time,
) (inboundDelivery, bool, error) {
	if s.pool == nil {
		return inboundDelivery{}, false, errors.New("inbound delivery pool is nil")
	}
	if err := validateLeaseInput(token, now, leaseExpiresAt); err != nil {
		return inboundDelivery{}, false, err
	}
	delivery, err := scanInboundDelivery(s.pool.QueryRow(ctx, `
		WITH candidate AS (
			SELECT delivery.id, delivery.status
			FROM chat_inbound_deliveries AS delivery
			WHERE (
				(delivery.status = 'received' AND delivery.next_attempt_at <= $2
				 AND NOT EXISTS (
					SELECT 1 FROM chat_inbound_deliveries AS earlier
					WHERE earlier.tenant_id = delivery.tenant_id
					  AND earlier.learner_key = delivery.learner_key
					  AND earlier.accepted_sequence < delivery.accepted_sequence
					  AND earlier.status NOT IN ('delivered', 'failed')
				 ))
				OR (delivery.status = 'delivery_pending' AND delivery.next_attempt_at <= $2
				 AND NOT EXISTS (
					SELECT 1 FROM chat_inbound_deliveries AS earlier
					WHERE earlier.tenant_id = delivery.tenant_id
					  AND earlier.destination_key = delivery.destination_key
					  AND earlier.accepted_sequence < delivery.accepted_sequence
					  AND earlier.status NOT IN ('delivered', 'failed')
				 ))
				OR (delivery.status = 'delivering' AND delivery.lease_expires_at <= $2)
			)
			ORDER BY delivery.next_attempt_at, delivery.accepted_sequence
			FOR UPDATE SKIP LOCKED
			LIMIT 1
		)
		UPDATE chat_inbound_deliveries AS delivery
		SET status = CASE candidate.status
				WHEN 'received' THEN 'processing'
				ELSE 'delivering'
			END,
			processing_attempt_count = delivery.processing_attempt_count
				+ CASE WHEN candidate.status = 'received' THEN 1 ELSE 0 END,
			delivery_attempt_count = delivery.delivery_attempt_count
				+ CASE WHEN candidate.status <> 'received' THEN 1 ELSE 0 END,
			lease_token = $1, lease_expires_at = $3, updated_at = $2
		FROM candidate
		WHERE delivery.id = candidate.id
		RETURNING `+prefixedInboundDeliveryColumns("delivery"), token, now, leaseExpiresAt))
	if errors.Is(err, pgx.ErrNoRows) {
		return inboundDelivery{}, false, nil
	}
	if err != nil {
		return inboundDelivery{}, false, fmt.Errorf("claim due inbound delivery: %w", err)
	}
	return delivery, true, nil
}

func (s *postgresInboundDeliveryStore) RenewLease(
	ctx context.Context,
	id string,
	token string,
	status inboundDeliveryStatus,
	now time.Time,
	leaseExpiresAt time.Time,
) error {
	if s.pool == nil {
		return errors.New("inbound delivery pool is nil")
	}
	if status != inboundDeliveryProcessing && status != inboundDeliveryDelivering {
		return errors.New("inbound delivery renewal status must be leased")
	}
	if err := validateLeaseInput(token, now, leaseExpiresAt); err != nil {
		return err
	}
	tag, err := s.pool.Exec(ctx, `
		UPDATE chat_inbound_deliveries
		SET lease_expires_at = $5, updated_at = $4
		WHERE id = $1::uuid AND lease_token = $2 AND status = $3
		  AND lease_expires_at > $4`, id, token, status, now, leaseExpiresAt)
	return fencedInboundDeliveryTransition("renew inbound delivery lease", tag, err)
}

func (s *postgresInboundDeliveryStore) CompleteProcessing(
	ctx context.Context,
	id string,
	token string,
	result agent.TurnResult,
	now time.Time,
) error {
	payload, err := json.Marshal(newInboundTurnResultPayload(result))
	if err != nil {
		return fmt.Errorf("encode inbound turn result: %w", err)
	}
	tag, err := s.pool.Exec(ctx, `
		UPDATE chat_inbound_deliveries
		SET status = 'delivery_pending', result_payload = $4::jsonb,
		    next_attempt_at = $3, lease_token = NULL, lease_expires_at = NULL, updated_at = $3
		WHERE id = $1::uuid AND lease_token = $2 AND status = 'processing'
		  AND lease_expires_at > $3`, id, token, now, payload)
	return fencedInboundDeliveryTransition("complete inbound delivery processing", tag, err)
}

func (s *postgresInboundDeliveryStore) ScheduleDeliveryRetry(
	ctx context.Context,
	id string,
	token string,
	nextAttempt time.Time,
	now time.Time,
) error {
	tag, err := s.pool.Exec(ctx, `
		UPDATE chat_inbound_deliveries
		SET status = 'delivery_pending', next_attempt_at = $4,
		    lease_token = NULL, lease_expires_at = NULL, updated_at = $3
		WHERE id = $1::uuid AND lease_token = $2 AND status = 'delivering'
		  AND lease_expires_at > $3`, id, token, now, nextAttempt)
	return fencedInboundDeliveryTransition("schedule inbound delivery retry", tag, err)
}

func (s *postgresInboundDeliveryStore) MarkDelivered(
	ctx context.Context,
	id string,
	token string,
	now time.Time,
) error {
	tag, err := s.pool.Exec(ctx, `
		UPDATE chat_inbound_deliveries
		SET status = 'delivered', delivered_at = $3,
		    lease_token = NULL, lease_expires_at = NULL, updated_at = $3
		WHERE id = $1::uuid AND lease_token = $2 AND status = 'delivering'
		  AND lease_expires_at > $3`, id, token, now)
	return fencedInboundDeliveryTransition("mark inbound delivery delivered", tag, err)
}

func (s *postgresInboundDeliveryStore) MarkDeliveryFailed(
	ctx context.Context,
	id string,
	token string,
	now time.Time,
) error {
	tag, err := s.pool.Exec(ctx, `
		UPDATE chat_inbound_deliveries
		SET status = 'failed', failed_at = $3,
		    lease_token = NULL, lease_expires_at = NULL, updated_at = $3
		WHERE id = $1::uuid AND lease_token = $2 AND status = 'delivering'
		  AND lease_expires_at > $3`, id, token, now)
	return fencedInboundDeliveryTransition("mark inbound delivery failed", tag, err)
}

func (s *postgresInboundDeliveryStore) RecoverExpiredProcessing(
	ctx context.Context,
	resultFor func(chat.InboundMessage) agent.TurnResult,
	now time.Time,
	limit int,
) (int64, error) {
	if limit <= 0 {
		return 0, errors.New("inbound delivery recovery limit must be positive")
	}
	if resultFor == nil {
		return 0, errors.New("inbound delivery recovery result renderer is required")
	}
	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return 0, fmt.Errorf("begin expired inbound processing recovery: %w", err)
	}
	defer func() { _ = tx.Rollback(context.Background()) }()
	rows, err := tx.Query(ctx, `
		SELECT `+inboundDeliveryColumns+`
		FROM chat_inbound_deliveries
		WHERE status = 'processing' AND lease_expires_at <= $1
		ORDER BY accepted_sequence
		FOR UPDATE SKIP LOCKED
		LIMIT $2`, now, limit)
	if err != nil {
		return 0, fmt.Errorf("select expired inbound processing: %w", err)
	}
	var expired []inboundDelivery
	for rows.Next() {
		delivery, scanErr := scanInboundDelivery(rows)
		if scanErr != nil {
			rows.Close()
			return 0, fmt.Errorf("scan expired inbound processing: %w", scanErr)
		}
		expired = append(expired, delivery)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return 0, fmt.Errorf("read expired inbound processing: %w", err)
	}
	rows.Close()
	for _, delivery := range expired {
		payload, marshalErr := json.Marshal(newInboundTurnResultPayload(resultFor(delivery.Message)))
		if marshalErr != nil {
			return 0, fmt.Errorf("encode recovered inbound result: %w", marshalErr)
		}
		if _, err := tx.Exec(ctx, `
			UPDATE chat_inbound_deliveries
			SET status = 'delivery_pending', result_payload = $2::jsonb,
			    next_attempt_at = $1, lease_token = NULL, lease_expires_at = NULL, updated_at = $1
			WHERE id = $3::uuid`, now, payload, delivery.ID); err != nil {
			return 0, fmt.Errorf("recover expired inbound processing: %w", err)
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return 0, fmt.Errorf("commit expired inbound processing recovery: %w", err)
	}
	return int64(len(expired)), nil
}

func (s *postgresInboundDeliveryStore) DeleteTerminalBefore(
	ctx context.Context,
	before time.Time,
	limit int,
) (int64, error) {
	if limit <= 0 {
		return 0, errors.New("inbound delivery cleanup limit must be positive")
	}
	tag, err := s.pool.Exec(ctx, `
		WITH expired AS (
			SELECT id FROM chat_inbound_deliveries
			WHERE (status = 'delivered' AND delivered_at <= $1)
			   OR (status = 'failed' AND failed_at <= $1)
			ORDER BY COALESCE(delivered_at, failed_at), accepted_sequence
			FOR UPDATE SKIP LOCKED
			LIMIT $2
		)
		DELETE FROM chat_inbound_deliveries AS delivery
		USING expired
		WHERE delivery.id = expired.id`, before, limit)
	if err != nil {
		return 0, fmt.Errorf("delete terminal inbound records: %w", err)
	}
	return tag.RowsAffected(), nil
}

const inboundDeliveryColumns = `
	id::text, tenant_id::text, channel, delivery_id, learner_key, destination_key,
	accepted_sequence, inbound_payload, result_payload, status,
	processing_attempt_count, delivery_attempt_count, next_attempt_at,
	COALESCE(lease_token, ''), lease_expires_at, delivered_at, failed_at, created_at, updated_at`

func prefixedInboundDeliveryColumns(alias string) string {
	return `
	` + alias + `.id::text, ` + alias + `.tenant_id::text, ` + alias + `.channel,
	` + alias + `.delivery_id, ` + alias + `.learner_key, ` + alias + `.destination_key,
	` + alias + `.accepted_sequence, ` + alias + `.inbound_payload, ` + alias + `.result_payload,
	` + alias + `.status, ` + alias + `.processing_attempt_count, ` + alias + `.delivery_attempt_count,
	` + alias + `.next_attempt_at, COALESCE(` + alias + `.lease_token, ''),
	` + alias + `.lease_expires_at, ` + alias + `.delivered_at, ` + alias + `.failed_at,
	` + alias + `.created_at, ` + alias + `.updated_at`
}

func scanInboundDelivery(row pgx.Row) (inboundDelivery, error) {
	var delivery inboundDelivery
	var inboundJSON, resultJSON []byte
	err := row.Scan(
		&delivery.ID, &delivery.TenantID, &delivery.Channel, &delivery.DeliveryID,
		&delivery.LearnerKey, &delivery.DestinationKey, &delivery.AcceptedSequence,
		&inboundJSON, &resultJSON, &delivery.Status, &delivery.ProcessingAttemptCount,
		&delivery.DeliveryAttemptCount, &delivery.NextAttemptAt, &delivery.LeaseToken,
		&delivery.LeaseExpiresAt, &delivery.DeliveredAt, &delivery.FailedAt,
		&delivery.CreatedAt, &delivery.UpdatedAt,
	)
	if err != nil {
		return inboundDelivery{}, err
	}
	var inboundPayload inboundMessagePayload
	if err := json.Unmarshal(inboundJSON, &inboundPayload); err != nil {
		return inboundDelivery{}, fmt.Errorf("decode inbound message payload: %w", err)
	}
	delivery.Message, err = inboundPayload.message()
	if err != nil {
		return inboundDelivery{}, err
	}
	if len(resultJSON) > 0 {
		var resultPayload inboundTurnResultPayload
		if err := json.Unmarshal(resultJSON, &resultPayload); err != nil {
			return inboundDelivery{}, fmt.Errorf("decode inbound result payload: %w", err)
		}
		delivery.Result, err = resultPayload.result()
		if err != nil {
			return inboundDelivery{}, err
		}
	}
	return delivery, nil
}

func validateLeaseInput(token string, now, leaseExpiresAt time.Time) error {
	if token == "" {
		return errors.New("inbound delivery lease token is required")
	}
	if !leaseExpiresAt.After(now) {
		return errors.New("inbound delivery lease expiry must be after current time")
	}
	return nil
}

func fencedInboundDeliveryTransition(operation string, tag pgconn.CommandTag, err error) error {
	if err != nil {
		return fmt.Errorf("%s: %w", operation, err)
	}
	if tag.RowsAffected() != 1 {
		return ErrInboundDeliveryLeaseLost
	}
	return nil
}
