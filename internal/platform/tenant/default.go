// Copyright 2026 the P&AI authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package tenant

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
)

type defaultTenantQuerier interface {
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}

func DefaultTenantID(ctx context.Context, q defaultTenantQuerier) (string, error) {
	queryCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	var tenantID string
	if err := q.QueryRow(queryCtx, `
		SELECT id::text
		FROM tenants
		WHERE slug = 'default'
		ORDER BY created_at ASC, id ASC
		LIMIT 1
	`).Scan(&tenantID); err != nil {
		return "", fmt.Errorf("lookup default tenant: %w", err)
	}
	return tenantID, nil
}
