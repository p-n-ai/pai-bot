// Copyright 2026 the P&AI authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package settings

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"maps"
	"slices"
	"strings"
	"sync"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/p-n-ai/pai-bot/internal/platform/config"
	"github.com/p-n-ai/pai-bot/internal/platform/featureflags"
)

const openRouterAPIKeySecret = "openrouter_api_key"

// ErrConfigEncryptionKey refuses secret writes until a dedicated,
// sufficiently long encryption key is configured.
var ErrConfigEncryptionKey = errors.New("set PAI_CONFIG_ENCRYPTION_KEY to at least 32 characters before storing API keys")

// Store persists the single runtime_settings row layered over the env baseline captured at boot.
type Store struct {
	pool                *pgxpool.Pool
	encryptionKey       string
	legacyDecryptionKey string
	envAI               config.AIConfig
	envFlags            featureflags.Features

	updateMu sync.Mutex // orders Update commit+apply pairs within this process
	mu       sync.RWMutex
	current  Settings // in-process snapshot; single-process app, no cross-instance invalidation
}

// New builds a Store. encryptionKey is the only key used for new writes;
// legacyDecryptionKey may read ciphertext created before the keys were split.
func New(pool *pgxpool.Pool, encryptionKey, legacyDecryptionKey string, envAI config.AIConfig, envFlags featureflags.Features) *Store {
	return &Store{
		pool:                pool,
		encryptionKey:       encryptionKey,
		legacyDecryptionKey: legacyDecryptionKey,
		envAI:               envAI,
		envFlags:            envFlags,
	}
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

// MergedAI returns the env AI baseline with st layered on top.
func (s *Store) MergedAI(st Settings) config.AIConfig { return MergeAI(s.envAI, st) }

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

	// Strict decode: never rebuild the row from a degraded read, that would
	// persist the data loss.
	cur, prevSecrets, err := decodeSettingsRow(s.decryptionKeys(), aiJSON, flagsJSON, secretsJSON)
	if err != nil {
		return Settings{}, fmt.Errorf("decode runtime settings for update: %w", err)
	}
	decodedKey := cur.AI.OpenRouterAPIKey
	reencryptKey := s.secretNeedsReencryption(prevSecrets[openRouterAPIKeySecret], decodedKey)
	st, err := mutate(cur)
	if err != nil {
		return Settings{}, err
	}
	if err := saveSettingsRow(ctx, tx, s.writeEncryptionKey(), st, prevSecrets, decodedKey, reencryptKey); err != nil {
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
	return degradeSettingsRow(s.decryptionKeys(), aiJSON, flagsJSON, secretsJSON), nil
}

func (s *Store) decryptionKeys() []string {
	keys := make([]string, 0, 2)
	if s.encryptionKey != "" {
		keys = append(keys, s.encryptionKey)
	}
	if s.legacyDecryptionKey != "" &&
		s.legacyDecryptionKey != config.DefaultAuthSecret &&
		s.legacyDecryptionKey != s.encryptionKey {
		keys = append(keys, s.legacyDecryptionKey)
	}
	return keys
}

func (s *Store) writeEncryptionKey() string {
	if s.encryptionKey == s.legacyDecryptionKey {
		return ""
	}
	return s.encryptionKey
}

func (s *Store) secretNeedsReencryption(blob, plaintext string) bool {
	key := s.writeEncryptionKey()
	if blob == "" || plaintext == "" || key == "" {
		return false
	}
	_, err := decryptString(key, blob)
	return err != nil
}

// decodeSettingsRow strictly decodes the row, also returning the raw secrets
// map for save paths. Corrupt jsonb is an error; an undecryptable key blob
// is not — the key is dropped with a warning so an admin can re-enter it.
func decodeSettingsRow(decryptionKeys []string, aiJSON, flagsJSON, secretsJSON []byte) (Settings, map[string]string, error) {
	var st Settings
	if err := json.Unmarshal(aiJSON, &st.AI); err != nil {
		return Settings{}, nil, fmt.Errorf("decode ai column: %w", err)
	}
	if err := json.Unmarshal(flagsJSON, &st.Flags); err != nil {
		return Settings{}, nil, fmt.Errorf("decode flags column: %w", err)
	}
	var secrets map[string]string
	if err := json.Unmarshal(secretsJSON, &secrets); err != nil {
		return Settings{}, nil, fmt.Errorf("decode secrets column: %w", err)
	}
	pruneUnknownFlags(st.Flags)
	if blob := secrets[openRouterAPIKeySecret]; blob != "" {
		key, err := decryptStringWithKeys(decryptionKeys, blob)
		if err != nil {
			slog.Warn("runtime settings: dropping undecryptable openrouter api key", "error", err)
		} else {
			st.AI.OpenRouterAPIKey = key
		}
	}
	return st, secrets, nil
}

// degradeSettingsRow never fails: a corrupted row degrades to zero Settings so
// the server boots on env config. Updates refuse to overwrite corrupted rows,
// so the stored settings need manual repair.
func degradeSettingsRow(decryptionKeys []string, aiJSON, flagsJSON, secretsJSON []byte) Settings {
	st, _, err := decodeSettingsRow(decryptionKeys, aiJSON, flagsJSON, secretsJSON)
	if err != nil {
		slog.Warn("runtime settings: corrupted row; using env config", "error", err)
		return Settings{}
	}
	return st
}

// pruneUnknownFlags drops stale flag names so decode, Effective, and
// WithOverrides agree; the next save rewrites the row without them.
func pruneUnknownFlags(flags map[string]bool) {
	known := featureflags.Defaults()
	var dropped []string
	for name := range flags {
		if _, ok := known[name]; !ok {
			dropped = append(dropped, name)
			delete(flags, name)
		}
	}
	if len(dropped) > 0 {
		slices.Sort(dropped)
		slog.Warn("runtime settings: dropping unknown feature flags", "flags", dropped)
	}
}

// mergeSecrets returns the secrets map to persist: prev with only the
// openrouter key entry changed when the mutated key differs from decodedKey.
func mergeSecrets(secret string, prev map[string]string, decodedKey, key string, forceReencrypt bool) (map[string]string, error) {
	secrets := make(map[string]string, len(prev))
	maps.Copy(secrets, prev)
	switch key {
	case decodedKey:
		if forceReencrypt {
			break
		}
		// Unchanged (including "" after an undecryptable blob was dropped at
		// decode): keep the stored blob byte-for-byte so reverting
		// PAI_AUTH_SECRET can still recover the key.
		return secrets, nil
	case "":
		delete(secrets, openRouterAPIKeySecret)
		return secrets, nil
	}
	if len(strings.TrimSpace(secret)) < 32 {
		return nil, ErrConfigEncryptionKey
	}
	blob, err := encryptString(secret, key)
	if err != nil {
		return nil, fmt.Errorf("encrypt openrouter api key: %w", err)
	}
	secrets[openRouterAPIKeySecret] = blob
	return secrets, nil
}

func decryptStringWithKeys(keys []string, blob string) (string, error) {
	for _, key := range keys {
		plaintext, err := decryptString(key, blob)
		if err == nil {
			return plaintext, nil
		}
	}
	return "", errors.New("secret cannot be decrypted with configured keys")
}

// saveSettingsRow upserts the settings row; prevSecrets and decodedKey come
// from the strict decode of the locked row (see mergeSecrets).
func saveSettingsRow(ctx context.Context, tx pgx.Tx, secret string, st Settings, prevSecrets map[string]string, decodedKey string, forceReencrypt bool) error {
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
	secrets, err := mergeSecrets(secret, prevSecrets, decodedKey, st.AI.OpenRouterAPIKey, forceReencrypt)
	if err != nil {
		return err
	}
	secretsJSON, err := json.Marshal(secrets)
	if err != nil {
		return fmt.Errorf("marshal secrets: %w", err)
	}

	_, err = tx.Exec(ctx, `
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
