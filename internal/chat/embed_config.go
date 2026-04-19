// Copyright 2026 the P&AI authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package chat

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// ErrEmbedNotConfigured is returned when no embed config exists for a given tenant/origin combination.
var ErrEmbedNotConfigured = errors.New("embed not configured for tenant")

// EmbedConfig holds the web-embed configuration for a tenant.
type EmbedConfig struct {
	ID             string
	TenantID       string
	Enabled        bool
	AllowedOrigins []string
	ThemeConfig    map[string]any
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

// EmbedConfigStore defines persistence operations for embed configurations.
type EmbedConfigStore interface {
	GetByTenantID(ctx context.Context, tenantID string) (EmbedConfig, error)
	GetByTenantSlug(ctx context.Context, slug string) (EmbedConfig, error)
	Upsert(ctx context.Context, cfg EmbedConfig) (EmbedConfig, error)
	AddOrigin(ctx context.Context, tenantID, origin string) error
	RemoveOrigin(ctx context.Context, tenantID, origin string) error
	IsOriginAllowed(ctx context.Context, tenantID, origin string) (bool, error)
	FindTenantBySlugAndOrigin(ctx context.Context, slug, origin string) (string, error)
}

// PostgresEmbedConfigStore is a PostgreSQL-backed EmbedConfigStore.
type PostgresEmbedConfigStore struct {
	pool *pgxpool.Pool
}

// NewPostgresEmbedConfigStore creates a new PostgresEmbedConfigStore.
func NewPostgresEmbedConfigStore(pool *pgxpool.Pool) *PostgresEmbedConfigStore {
	return &PostgresEmbedConfigStore{pool: pool}
}

// GetByTenantID returns the embed config for the given tenant.
// If no row exists, a zero EmbedConfig with the tenant_id set is returned (not an error).
func (s *PostgresEmbedConfigStore) GetByTenantID(ctx context.Context, tenantID string) (EmbedConfig, error) {
	var cfg EmbedConfig
	var themeBytes []byte
	var origins []string

	err := s.pool.QueryRow(ctx,
		`SELECT id::text, tenant_id::text, enabled, allowed_origins, theme_config, created_at, updated_at
		 FROM embed_configs
		 WHERE tenant_id = $1::uuid`,
		tenantID,
	).Scan(
		&cfg.ID,
		&cfg.TenantID,
		&cfg.Enabled,
		&origins,
		&themeBytes,
		&cfg.CreatedAt,
		&cfg.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return EmbedConfig{TenantID: tenantID}, nil
	}
	if err != nil {
		return EmbedConfig{}, fmt.Errorf("get embed config: %w", err)
	}

	cfg.AllowedOrigins = origins
	if len(themeBytes) > 0 {
		if err := json.Unmarshal(themeBytes, &cfg.ThemeConfig); err != nil {
			return EmbedConfig{}, fmt.Errorf("unmarshal theme_config: %w", err)
		}
	}

	return cfg, nil
}

// GetByTenantSlug returns the embed config for the given tenant slug.
// If no row exists, a zero EmbedConfig is returned (not an error).
func (s *PostgresEmbedConfigStore) GetByTenantSlug(ctx context.Context, slug string) (EmbedConfig, error) {
	var cfg EmbedConfig
	var themeBytes []byte
	var origins []string

	err := s.pool.QueryRow(ctx,
		`SELECT ec.id::text, ec.tenant_id::text, ec.enabled, ec.allowed_origins, ec.theme_config, ec.created_at, ec.updated_at
		 FROM embed_configs ec
		 JOIN tenants t ON t.id = ec.tenant_id
		 WHERE t.slug = $1`,
		slug,
	).Scan(
		&cfg.ID,
		&cfg.TenantID,
		&cfg.Enabled,
		&origins,
		&themeBytes,
		&cfg.CreatedAt,
		&cfg.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return EmbedConfig{}, nil
	}
	if err != nil {
		return EmbedConfig{}, fmt.Errorf("get embed config by slug: %w", err)
	}

	cfg.AllowedOrigins = origins
	if len(themeBytes) > 0 {
		if err := json.Unmarshal(themeBytes, &cfg.ThemeConfig); err != nil {
			return EmbedConfig{}, fmt.Errorf("unmarshal theme_config: %w", err)
		}
	}

	return cfg, nil
}

// Upsert inserts or updates the embed config for a tenant.
func (s *PostgresEmbedConfigStore) Upsert(ctx context.Context, cfg EmbedConfig) (EmbedConfig, error) {
	themeBytes, err := json.Marshal(cfg.ThemeConfig)
	if err != nil {
		return EmbedConfig{}, fmt.Errorf("marshal theme_config: %w", err)
	}

	origins := cfg.AllowedOrigins
	if origins == nil {
		origins = []string{}
	}

	var out EmbedConfig
	var outThemeBytes []byte
	var outOrigins []string

	err = s.pool.QueryRow(ctx,
		`INSERT INTO embed_configs (tenant_id, enabled, allowed_origins, theme_config)
		 VALUES ($1::uuid, $2, $3, $4::jsonb)
		 ON CONFLICT (tenant_id) DO UPDATE
		   SET enabled = $2,
		       allowed_origins = $3,
		       theme_config = $4::jsonb,
		       updated_at = now()
		 RETURNING id::text, tenant_id::text, enabled, allowed_origins, theme_config, created_at, updated_at`,
		cfg.TenantID,
		cfg.Enabled,
		origins,
		themeBytes,
	).Scan(
		&out.ID,
		&out.TenantID,
		&out.Enabled,
		&outOrigins,
		&outThemeBytes,
		&out.CreatedAt,
		&out.UpdatedAt,
	)
	if err != nil {
		return EmbedConfig{}, fmt.Errorf("upsert embed config: %w", err)
	}

	out.AllowedOrigins = outOrigins
	if len(outThemeBytes) > 0 {
		if err := json.Unmarshal(outThemeBytes, &out.ThemeConfig); err != nil {
			return EmbedConfig{}, fmt.Errorf("unmarshal theme_config after upsert: %w", err)
		}
	}

	return out, nil
}

// AddOrigin appends an origin to the tenant's allowed_origins if not already present.
// If no row exists, one is inserted with just that origin.
func (s *PostgresEmbedConfigStore) AddOrigin(ctx context.Context, tenantID, origin string) error {
	cmd, err := s.pool.Exec(ctx,
		`UPDATE embed_configs
		 SET allowed_origins = array_append(allowed_origins, $2),
		     updated_at = now()
		 WHERE tenant_id = $1::uuid
		   AND NOT ($2 = ANY(allowed_origins))`,
		tenantID,
		origin,
	)
	if err != nil {
		return fmt.Errorf("add origin: %w", err)
	}

	if cmd.RowsAffected() == 0 {
		// Either no row exists, or origin already present — try insert.
		_, err = s.pool.Exec(ctx,
			`INSERT INTO embed_configs (tenant_id, allowed_origins)
			 VALUES ($1::uuid, ARRAY[$2::text])
			 ON CONFLICT (tenant_id) DO NOTHING`,
			tenantID,
			origin,
		)
		if err != nil {
			return fmt.Errorf("insert embed config for origin: %w", err)
		}
	}

	return nil
}

// RemoveOrigin removes an origin from the tenant's allowed_origins.
func (s *PostgresEmbedConfigStore) RemoveOrigin(ctx context.Context, tenantID, origin string) error {
	_, err := s.pool.Exec(ctx,
		`UPDATE embed_configs
		 SET allowed_origins = array_remove(allowed_origins, $2),
		     updated_at = now()
		 WHERE tenant_id = $1::uuid`,
		tenantID,
		origin,
	)
	if err != nil {
		return fmt.Errorf("remove origin: %w", err)
	}
	return nil
}

// IsOriginAllowed returns true if the tenant has embed enabled and the origin is in allowed_origins.
func (s *PostgresEmbedConfigStore) IsOriginAllowed(ctx context.Context, tenantID, origin string) (bool, error) {
	var allowed bool
	err := s.pool.QueryRow(ctx,
		`SELECT EXISTS(
		   SELECT 1 FROM embed_configs
		   WHERE tenant_id = $1::uuid
		     AND enabled = true
		     AND $2 = ANY(allowed_origins)
		 )`,
		tenantID,
		origin,
	).Scan(&allowed)
	if err != nil {
		return false, fmt.Errorf("is origin allowed: %w", err)
	}
	return allowed, nil
}

// FindTenantBySlugAndOrigin returns the tenant ID for a given tenant slug and origin,
// provided embed is enabled for that tenant. Returns ErrEmbedNotConfigured if not found.
func (s *PostgresEmbedConfigStore) FindTenantBySlugAndOrigin(ctx context.Context, slug, origin string) (string, error) {
	var tenantID string
	err := s.pool.QueryRow(ctx,
		`SELECT t.id::text
		 FROM tenants t
		 JOIN embed_configs ec ON ec.tenant_id = t.id
		 WHERE t.slug = $1
		   AND ec.enabled = true
		   AND $2 = ANY(ec.allowed_origins)`,
		slug,
		origin,
	).Scan(&tenantID)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", ErrEmbedNotConfigured
	}
	if err != nil {
		return "", fmt.Errorf("find tenant by slug and origin: %w", err)
	}
	return tenantID, nil
}
