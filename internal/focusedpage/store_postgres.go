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

type PostgresStore struct{ pool *pgxpool.Pool }

func NewPostgresStore(pool *pgxpool.Pool) *PostgresStore { return &PostgresStore{pool: pool} }

func (s *PostgresStore) CreateOrGet(ctx context.Context, record CreateRecord) (Page, error) {
	if s.pool == nil {
		return Page{}, fmt.Errorf("focused page pool is nil")
	}
	_, err := s.pool.Exec(ctx, `
		INSERT INTO focused_pages (
			public_id, tenant_id, owner_user_id, conversation_id, turn_id, page_index,
			recipient_name, message, token_hash, status, created_at, expires_at
		)
		SELECT $1, $2::uuid, $3::uuid, c.id, $5, $6, $7, $8, $9, 'active', $10, $11
		FROM conversations c
		WHERE c.id = $4::uuid AND c.tenant_id = $2::uuid AND c.user_id = $3::uuid
		ON CONFLICT (tenant_id, turn_id, page_index) DO NOTHING`,
		record.PublicID, record.TenantID, record.OwnerUserID, record.ConversationID, record.TurnID, record.PageIndex,
		record.RecipientName, record.Message, record.TokenHash, record.CreatedAt, record.ExpiresAt)
	if err != nil {
		return Page{}, fmt.Errorf("create focused page: %w", err)
	}
	page, err := s.getByTurn(ctx, record.TenantID, record.OwnerUserID, record.ConversationID, record.TurnID, record.PageIndex)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return Page{}, ErrForbidden
		}
		return Page{}, err
	}
	return page, nil
}

func (s *PostgresStore) Redeem(ctx context.Context, publicID string, tokenHash []byte, now time.Time) (Page, error) {
	var page Page
	var revokedAt *time.Time
	err := s.pool.QueryRow(ctx, `
		UPDATE focused_pages
		SET status = CASE WHEN status = 'active' AND expires_at <= $3 THEN 'expired' ELSE status END,
		    expired_at = CASE WHEN status = 'active' AND expires_at <= $3 THEN $3 ELSE expired_at END
		WHERE public_id = $1 AND token_hash = $2
		RETURNING public_id, tenant_id::text, owner_user_id::text, conversation_id::text, turn_id::text,
		          recipient_name, message, token_hash, status, created_at, expires_at, revoked_at`,
		publicID, tokenHash, now).Scan(&page.PublicID, &page.TenantID, &page.OwnerUserID, &page.ConversationID,
		&page.TurnID, &page.RecipientName, &page.Message, &page.TokenHash, &page.Status, &page.CreatedAt, &page.ExpiresAt, &revokedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return Page{}, ErrForbidden
		}
		return Page{}, fmt.Errorf("redeem focused page: %w", err)
	}
	page.RevokedAt = revokedAt
	if page.Status == StatusRevoked {
		return Page{}, ErrRevoked
	}
	if page.Status == StatusExpired {
		return Page{}, ErrExpired
	}
	return page, nil
}

func (s *PostgresStore) Revoke(ctx context.Context, publicID, tenantID, ownerUserID string, now time.Time) error {
	cmd, err := s.pool.Exec(ctx, `UPDATE focused_pages SET status = 'revoked', revoked_at = $4
		WHERE public_id = $1 AND tenant_id = $2::uuid AND owner_user_id = $3::uuid AND status = 'active'`,
		publicID, tenantID, ownerUserID, now)
	if err != nil {
		return fmt.Errorf("revoke focused page: %w", err)
	}
	if cmd.RowsAffected() == 0 {
		return ErrForbidden
	}
	return nil
}

func (s *PostgresStore) getByTurn(ctx context.Context, tenantID, ownerUserID, conversationID, turnID string, pageIndex int) (Page, error) {
	var page Page
	var revokedAt *time.Time
	err := s.pool.QueryRow(ctx, `SELECT public_id, tenant_id::text, owner_user_id::text, conversation_id::text, turn_id::text,
		recipient_name, message, token_hash, status, created_at, expires_at, revoked_at
		FROM focused_pages WHERE tenant_id = $1::uuid AND owner_user_id = $2::uuid AND conversation_id = $3::uuid
		AND turn_id = $4 AND page_index = $5`, tenantID, ownerUserID, conversationID, turnID, pageIndex).
		Scan(&page.PublicID, &page.TenantID, &page.OwnerUserID, &page.ConversationID, &page.TurnID,
			&page.RecipientName, &page.Message, &page.TokenHash, &page.Status, &page.CreatedAt, &page.ExpiresAt, &revokedAt)
	if err != nil {
		return Page{}, err
	}
	page.RevokedAt = revokedAt
	return page, nil
}
