// Copyright 2026 the P&AI authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package settings

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"sync"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/p-n-ai/pai-bot/internal/platform/config"
	"github.com/p-n-ai/pai-bot/internal/platform/featureflags"
)

const openRouterAPIKeySecret = "openrouter_api_key"

// Store persists the single runtime_settings row layered over the env baseline captured at boot.
type Store struct {
	pool     *pgxpool.Pool
	secret   string
	envAI    config.AIConfig
	envFlags featureflags.Features

	updateMu sync.Mutex // orders Update commit+apply pairs within this process
	mu       sync.RWMutex
	current  Settings // in-process snapshot; single-process app, no cross-instance invalidation
}

// New builds a Store; secret is the auth secret used to encrypt stored keys.
func New(pool *pgxpool.Pool, secret string, envAI config.AIConfig, envFlags featureflags.Features) *Store {
	return &Store{pool: pool, secret: secret, envAI: envAI, envFlags: envFlags}
}

// Start loads the initial snapshot served by Current.
func (s *Store) Start(ctx context.Context) error {
	st, err := s.Load(ctx)
	if err != nil {
		return err
	}
	s.setCurrent(st)
	return nil
}

// Current returns the last loaded or saved settings snapshot.
func (s *Store) Current() Settings {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.current
}

func (s *Store) setCurrent(st Settings) {
	s.mu.Lock()
	s.current = st
	s.mu.Unlock()
}

// Effective returns the merged env+DB view of the current snapshot.
func (s *Store) Effective() EffectiveSettings {
	return Effective(s.envAI, s.envFlags, s.Current())
}

// Update mutates the row re-read under a Postgres row lock and saves in the same tx; apply (nil ok) runs before updateMu releases, in commit order.
func (s *Store) Update(ctx context.Context, mutate func(Settings) (Settings, error), apply func(Settings)) (Settings, error) {
	s.updateMu.Lock()
	defer s.updateMu.Unlock()

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return Settings{}, fmt.Errorf("begin runtime settings update: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	// Insert-if-missing first so FOR UPDATE always has a row to lock.
	if _, err := tx.Exec(ctx, `INSERT INTO runtime_settings (id) VALUES (1) ON CONFLICT (id) DO NOTHING`); err != nil {
		return Settings{}, fmt.Errorf("init runtime settings row: %w", err)
	}
	var aiJSON, flagsJSON, secretsJSON []byte
	if err := tx.QueryRow(ctx,
		`SELECT ai, flags, secrets FROM runtime_settings WHERE id = 1 FOR UPDATE`,
	).Scan(&aiJSON, &flagsJSON, &secretsJSON); err != nil {
		return Settings{}, fmt.Errorf("load runtime settings for update: %w", err)
	}

	st, err := mutate(decodeSettingsRow(s.secret, aiJSON, flagsJSON, secretsJSON))
	if err != nil {
		return Settings{}, err
	}
	if err := saveSettingsRow(ctx, tx, s.secret, st); err != nil {
		return Settings{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return Settings{}, fmt.Errorf("commit runtime settings update: %w", err)
	}
	s.setCurrent(st)
	if apply != nil {
		apply(st)
	}
	return st, nil
}

// Load reads the settings row; a missing row yields zero Settings and a
// corrupted row degrades (see decodeSettingsRow) instead of failing boot.
func (s *Store) Load(ctx context.Context) (Settings, error) {
	var aiJSON, flagsJSON, secretsJSON []byte
	err := s.pool.QueryRow(ctx,
		`SELECT ai, flags, secrets FROM runtime_settings WHERE id = 1`,
	).Scan(&aiJSON, &flagsJSON, &secretsJSON)
	if errors.Is(err, pgx.ErrNoRows) {
		return Settings{}, nil
	}
	if err != nil {
		return Settings{}, fmt.Errorf("load runtime settings: %w", err)
	}
	return decodeSettingsRow(s.secret, aiJSON, flagsJSON, secretsJSON), nil
}

// decodeSettingsRow never fails: corrupted jsonb degrades to zero Settings,
// and an undecryptable key (e.g. PAI_AUTH_SECRET rotated after the key was
// stored) is dropped so the server boots and an admin can re-enter it.
func decodeSettingsRow(secret string, aiJSON, flagsJSON, secretsJSON []byte) Settings {
	var st Settings
	if err := json.Unmarshal(aiJSON, &st.AI); err != nil {
		slog.Warn("runtime settings: corrupted ai column; using env config", "error", err)
		return Settings{}
	}
	if err := json.Unmarshal(flagsJSON, &st.Flags); err != nil {
		slog.Warn("runtime settings: corrupted flags column; using env config", "error", err)
		return Settings{}
	}
	var secrets map[string]string
	if err := json.Unmarshal(secretsJSON, &secrets); err != nil {
		slog.Warn("runtime settings: corrupted secrets column; using env config", "error", err)
		return Settings{}
	}
	if blob := secrets[openRouterAPIKeySecret]; blob != "" {
		key, err := decryptString(secret, blob)
		if err != nil {
			slog.Warn("runtime settings: dropping undecryptable openrouter api key", "error", err)
		} else {
			st.AI.OpenRouterAPIKey = key
		}
	}
	return st
}

type settingsExecer interface {
	Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)
}

// saveSettingsRow upserts the settings row, encrypting the OpenRouter API key.
// An empty key stores no secret, removing any previous one.
func saveSettingsRow(ctx context.Context, db settingsExecer, secret string, st Settings) error {
	aiJSON, err := json.Marshal(st.AI)
	if err != nil {
		return fmt.Errorf("marshal ai settings: %w", err)
	}
	flags := st.Flags
	if flags == nil {
		flags = map[string]bool{}
	}
	flagsJSON, err := json.Marshal(flags)
	if err != nil {
		return fmt.Errorf("marshal flags: %w", err)
	}
	secrets := map[string]string{}
	if st.AI.OpenRouterAPIKey != "" {
		blob, err := encryptString(secret, st.AI.OpenRouterAPIKey)
		if err != nil {
			return fmt.Errorf("encrypt openrouter api key: %w", err)
		}
		secrets[openRouterAPIKeySecret] = blob
	}
	secretsJSON, err := json.Marshal(secrets)
	if err != nil {
		return fmt.Errorf("marshal secrets: %w", err)
	}

	_, err = db.Exec(ctx, `
		INSERT INTO runtime_settings (id, ai, flags, secrets, updated_at)
		VALUES (1, $1, $2, $3, now())
		ON CONFLICT (id) DO UPDATE
		SET ai = EXCLUDED.ai, flags = EXCLUDED.flags, secrets = EXCLUDED.secrets, updated_at = now()`,
		aiJSON, flagsJSON, secretsJSON)
	if err != nil {
		return fmt.Errorf("save runtime settings: %w", err)
	}
	return nil
}
