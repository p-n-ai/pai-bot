// Copyright 2026 the P&AI authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package tenant

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

const (
	defaultTenantSlug = "default"
	defaultTenantName = "Default"
)

type queryRower interface {
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}

// EnsureDefaultTenantForMode enforces tenant bootstrap behavior by mode.
// In single mode, it ensures the default tenant exists and returns its ID.
// In multi mode, it performs no mutation and returns an empty tenant ID.
func EnsureDefaultTenantForMode(ctx context.Context, mode string, q queryRower) (string, error) {
	mode = strings.TrimSpace(mode)
	switch mode {
	case "single":
		return ensureDefaultTenant(ctx, q)
	case "multi":
		return "", nil
	default:
		return "", fmt.Errorf("invalid tenant mode: %q", mode)
	}
}

// EnsureDefaultTenantForPool is a pool-backed wrapper used by app startup.
func EnsureDefaultTenantForPool(ctx context.Context, mode string, pool *pgxpool.Pool) (string, error) {
	if pool == nil {
		return "", errors.New("pool is nil")
	}
	return EnsureDefaultTenantForMode(ctx, mode, pool)
}

func ensureDefaultTenant(ctx context.Context, q queryRower) (string, error) {
	var tenantID string
	err := q.QueryRow(ctx,
		`SELECT id::text FROM tenants WHERE slug = $1 LIMIT 1`,
		defaultTenantSlug,
	).Scan(&tenantID)
	if err == nil {
		return tenantID, nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return "", fmt.Errorf("query default tenant: %w", err)
	}

	err = q.QueryRow(ctx,
		`INSERT INTO tenants (name, slug)
		 VALUES ($1, $2)
		 ON CONFLICT (slug) DO UPDATE
		 SET name = EXCLUDED.name
		 RETURNING id::text`,
		defaultTenantName,
		defaultTenantSlug,
	).Scan(&tenantID)
	if err != nil {
		return "", fmt.Errorf("upsert default tenant: %w", err)
	}

	return tenantID, nil
}
