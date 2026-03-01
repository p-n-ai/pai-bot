// Package database provides PostgreSQL connection management via pgx.
package database

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// DB wraps a pgx connection pool.
type DB struct {
	Pool *pgxpool.Pool
}

// ParseURL validates a PostgreSQL connection URL.
func ParseURL(url string) (*pgxpool.Config, error) {
	if url == "" {
		return nil, fmt.Errorf("database URL is empty")
	}
	cfg, err := pgxpool.ParseConfig(url)
	if err != nil {
		return nil, fmt.Errorf("invalid database URL: %w", err)
	}
	return cfg, nil
}

// New creates a new database connection pool.
func New(ctx context.Context, url string, maxConns, minConns int) (*DB, error) {
	cfg, err := ParseURL(url)
	if err != nil {
		return nil, err
	}

	cfg.MaxConns = int32(maxConns)
	cfg.MinConns = int32(minConns)
	cfg.MaxConnLifetime = 30 * time.Minute
	cfg.MaxConnIdleTime = 5 * time.Minute

	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		return nil, fmt.Errorf("creating connection pool: %w", err)
	}

	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("pinging database: %w", err)
	}

	return &DB{Pool: pool}, nil
}

// Close shuts down the connection pool.
func (db *DB) Close() {
	db.Pool.Close()
}

// HealthCheck verifies the database connection is alive.
func (db *DB) HealthCheck(ctx context.Context) error {
	return db.Pool.Ping(ctx)
}
