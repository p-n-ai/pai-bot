// Copyright 2026 the P&AI authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package tenant

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

func DefaultTenantID(ctx context.Context, pool *pgxpool.Pool) (string, error) {
	queryCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	return scanDefaultTenantID(pool.QueryRow(queryCtx, `
		SELECT id::text
		FROM tenants
		WHERE slug = 'default'
		ORDER BY created_at ASC, id ASC
		LIMIT 1
	`))
}

func scanDefaultTenantID(row pgx.Row) (string, error) {
	var tenantID string
	if err := row.Scan(&tenantID); err != nil {
		return "", fmt.Errorf("lookup default tenant: %w", err)
	}
	return tenantID, nil
}
