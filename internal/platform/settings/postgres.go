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
	"github.com/jackc/pgx/v5/pgxpool"
)

const openRouterAPIKeySecret = "openrouter_api_key"

// Store persists the single runtime_settings row.
type Store struct {
	pool   *pgxpool.Pool
	secret string

	mu      sync.RWMutex
	current Settings // in-process snapshot; single-process app, no cross-instance invalidation
}

// New builds a Store; secret is the auth secret used to encrypt stored keys.
func New(pool *pgxpool.Pool, secret string) *Store {
	return &Store{pool: pool, secret: secret}
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

// Save upserts the settings row, encrypting the OpenRouter API key.
// An empty key stores no secret, removing any previous one.
func (s *Store) Save(ctx context.Context, st Settings) error {
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
		blob, err := encryptString(s.secret, st.AI.OpenRouterAPIKey)
		if err != nil {
			return fmt.Errorf("encrypt openrouter api key: %w", err)
		}
		secrets[openRouterAPIKeySecret] = blob
	}
	secretsJSON, err := json.Marshal(secrets)
	if err != nil {
		return fmt.Errorf("marshal secrets: %w", err)
	}

	_, err = s.pool.Exec(ctx, `
		INSERT INTO runtime_settings (id, ai, flags, secrets, updated_at)
		VALUES (1, $1, $2, $3, now())
		ON CONFLICT (id) DO UPDATE
		SET ai = EXCLUDED.ai, flags = EXCLUDED.flags, secrets = EXCLUDED.secrets, updated_at = now()`,
		aiJSON, flagsJSON, secretsJSON)
	if err != nil {
		return fmt.Errorf("save runtime settings: %w", err)
	}
	s.setCurrent(st)
	return nil
}
