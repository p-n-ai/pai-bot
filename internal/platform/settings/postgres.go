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
	"sync"
	"unicode"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/p-n-ai/pai-bot/internal/platform/config"
	"github.com/p-n-ai/pai-bot/internal/platform/featureflags"
)

const openRouterAPIKeySecret = "openrouter_api_key"

var openRouterAPIKeyContext = credentialContext{
	Provider: "openrouter",
	Slot:     "api_key",
}

// ErrConfigEncryptionKey refuses secret writes until a dedicated,
// sufficiently long encryption key is configured.
var ErrConfigEncryptionKey = errors.New("set PAI_CONFIG_ENCRYPTION_KEY to at least 32 characters before storing API keys")

// ErrRevisionConflict rejects an update based on stale desired state.
var ErrRevisionConflict = errors.New("runtime settings revision conflict")

// Store persists the single runtime_settings row layered over the env baseline captured at boot.
type Store struct {
	pool                   *pgxpool.Pool
	encryptionKey          string
	previousEncryptionKeys []string
	legacyDecryptionKey    string
	envAI                  config.AIConfig
	envFlags               featureflags.Features

	updateMu        sync.Mutex // orders Update commit+apply pairs within this process
	mu              sync.RWMutex
	current         Settings // in-process snapshot; single-process app, no cross-instance invalidation
	appliedRevision int64
	hasApplied      bool
}

// New builds a Store. encryptionKey is the only key used for new writes;
// legacyDecryptionKey may read ciphertext created before the keys were split.
func New(pool *pgxpool.Pool, encryptionKey, legacyDecryptionKey string, envAI config.AIConfig, envFlags featureflags.Features) *Store {
	return NewWithPreviousKeys(pool, encryptionKey, nil, legacyDecryptionKey, envAI, envFlags)
}

// NewWithPreviousKeys builds a Store that writes with encryptionKey and reads
// versioned envelopes produced by the bounded previous key set.
func NewWithPreviousKeys(
	pool *pgxpool.Pool,
	encryptionKey string,
	previousEncryptionKeys []string,
	legacyDecryptionKey string,
	envAI config.AIConfig,
	envFlags featureflags.Features,
) *Store {
	return &Store{
		pool:                   pool,
		encryptionKey:          encryptionKey,
		previousEncryptionKeys: append([]string(nil), previousEncryptionKeys...),
		legacyDecryptionKey:    legacyDecryptionKey,
		envAI:                  envAI,
		envFlags:               envFlags,
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

func (s *Store) setAppliedRevision(revision int64) {
	s.mu.Lock()
	s.appliedRevision = revision
	s.hasApplied = true
	s.mu.Unlock()
}

// MarkApplied records that the live runtime accepted revision.
func (s *Store) MarkApplied(revision int64) {
	s.setAppliedRevision(revision)
}

// Effective returns the merged env+DB view of the current snapshot.
func (s *Store) Effective() EffectiveSettings {
	s.mu.RLock()
	current := s.current
	appliedRevision := s.appliedRevision
	hasApplied := s.hasApplied
	s.mu.RUnlock()
	effective := Effective(s.envAI, s.envFlags, current)
	effective.AppliedRevision = appliedRevision
	effective.Drift = !hasApplied || effective.Revision != effective.AppliedRevision
	return effective
}

// EffectiveFor returns the reconciled view for a committed snapshot whose
// synchronous apply completed before Update returned.
func (s *Store) EffectiveFor(st Settings) EffectiveSettings {
	effective := Effective(s.envAI, s.envFlags, st)
	effective.AppliedRevision = st.Revision
	effective.Drift = false
	return effective
}

// MergedAI returns the env AI baseline with st layered on top.
func (s *Store) MergedAI(st Settings) config.AIConfig { return MergeAI(s.envAI, st) }

// Update prepares runtime state before persistence, commits desired state, then
// atomically applies the prepared plan while updateMu preserves commit order.
func (s *Store) Update(ctx context.Context, mutate func(Settings) (Settings, error), prepare PrepareApply) (Settings, error) {
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
	var revision int64
	if err := tx.QueryRow(ctx,
		`SELECT ai, flags, secrets, revision FROM runtime_settings WHERE id = 1 FOR UPDATE`,
	).Scan(&aiJSON, &flagsJSON, &secretsJSON, &revision); err != nil {
		return Settings{}, fmt.Errorf("load runtime settings for update: %w", err)
	}

	// Strict decode: never rebuild the row from a degraded read, that would
	// persist the data loss.
	cur, prevSecrets, err := decodeSettingsRow(
		s.encryptionKey,
		s.previousEncryptionKeys,
		s.legacyKeys(),
		aiJSON,
		flagsJSON,
		secretsJSON,
	)
	if err != nil {
		return Settings{}, fmt.Errorf("decode runtime settings for update: %w", err)
	}
	cur.Revision = revision
	decodedKey := cur.AI.OpenRouterAPIKey
	reencryptKey := s.secretNeedsReencryption(prevSecrets[openRouterAPIKeySecret], decodedKey)
	st, err := mutate(cur)
	if err != nil {
		return Settings{}, err
	}
	s.canonicalizeRedundantOverrides(&st)
	_, hadStoredKey := prevSecrets[openRouterAPIKeySecret]
	explicitStoredKeyClear := st.AI.OpenRouterAPIKeyOperation == SecretClear && hadStoredKey
	if sameDesiredSettings(cur, st) && !explicitStoredKeyClear {
		st.Revision = cur.Revision
	} else {
		st.Revision = cur.Revision + 1
	}
	var apply PreparedApply
	if prepare != nil {
		apply, err = prepare(st)
		if err != nil {
			return Settings{}, err
		}
	}
	if err := saveSettingsRow(ctx, tx, s.writeEncryptionKey(), st, prevSecrets, decodedKey, reencryptKey); err != nil {
		return Settings{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return Settings{}, fmt.Errorf("commit runtime settings update: %w", err)
	}
	switch {
	case st.AI.OpenRouterAPIKeyOperation == SecretClear:
		st.AI.OpenRouterEnvelope = CredentialEnvelopeStatus{}
	case st.AI.OpenRouterAPIKeyOperation == SecretReplace || reencryptKey:
		st.AI.OpenRouterEnvelope = CredentialEnvelopeStatus{
			Stored:    true,
			Readable:  true,
			Version:   "v1",
			Algorithm: credentialEnvelopeAlgorithm,
			KeyID:     credentialKeyID(s.writeEncryptionKey()),
		}
	}
	st.AI.OpenRouterAPIKeyOperation = SecretPreserve
	s.setCurrent(st)
	if apply != nil {
		apply()
		s.setAppliedRevision(st.Revision)
	}
	return st, nil
}

// canonicalizeRedundantOverrides keeps the database as an override layer
// rather than a copy of the boot environment. A later deployment can then
// change the baseline without an equal historical override masking it.
func (s *Store) canonicalizeRedundantOverrides(st *Settings) {
	if st.AI.DefaultProvider == s.envAI.DefaultProvider {
		st.AI.DefaultProvider = ""
	}
	if st.AI.OpenRouterModel == s.envAI.OpenRouter.Model {
		st.AI.OpenRouterModel = ""
	}
	if st.AI.OpenRouterAPIKey != "" &&
		st.AI.OpenRouterAPIKey == s.envAI.OpenRouter.APIKey {
		st.AI.OpenRouterAPIKey = ""
		st.AI.OpenRouterAPIKeyOperation = SecretClear
	}
	for name, value := range st.Flags {
		if value == s.envFlags.Enabled(featureflags.Feature(name)) {
			delete(st.Flags, name)
		}
	}
}

// Load reads the settings row; a missing row yields zero Settings and a
// corrupted row degrades (see decodeSettingsRow) instead of failing boot.
func (s *Store) Load(ctx context.Context) (Settings, error) {
	var aiJSON, flagsJSON, secretsJSON []byte
	var revision int64
	err := s.pool.QueryRow(ctx,
		`SELECT ai, flags, secrets, revision FROM runtime_settings WHERE id = 1`,
	).Scan(&aiJSON, &flagsJSON, &secretsJSON, &revision)
	if errors.Is(err, pgx.ErrNoRows) {
		return Settings{}, nil
	}
	if err != nil {
		return Settings{}, fmt.Errorf("load runtime settings: %w", err)
	}
	st := degradeSettingsRow(
		s.encryptionKey,
		s.previousEncryptionKeys,
		s.legacyKeys(),
		aiJSON,
		flagsJSON,
		secretsJSON,
	)
	st.Revision = revision
	return st, nil
}

func (s *Store) decryptionKeys() []string {
	keys := make([]string, 0, 2+len(s.previousEncryptionKeys))
	if s.encryptionKey != "" {
		keys = append(keys, s.encryptionKey)
	}
	keys = appendUniqueSecrets(keys, s.previousEncryptionKeys...)
	if s.legacyDecryptionKey != "" &&
		s.legacyDecryptionKey != config.DefaultAuthSecret &&
		s.legacyDecryptionKey != s.encryptionKey {
		keys = appendUniqueSecrets(keys, s.legacyDecryptionKey)
	}
	return keys
}

func (s *Store) legacyKeys() []string {
	if s.legacyDecryptionKey == "" ||
		s.legacyDecryptionKey == config.DefaultAuthSecret ||
		s.legacyDecryptionKey == s.encryptionKey {
		return nil
	}
	return []string{s.legacyDecryptionKey}
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
	decrypted, err := decryptCredential(
		key,
		s.previousEncryptionKeys,
		s.legacyKeys(),
		blob,
		openRouterAPIKeyContext,
	)
	return err != nil || decrypted.NeedsRewrite
}

// decodeSettingsRow strictly decodes the row, also returning the raw secrets
// map for save paths. Corrupt jsonb is an error; an undecryptable key blob
// is not — the key is dropped with a warning so an admin can re-enter it.
func decodeSettingsRow(
	activeKey string,
	previousKeys []string,
	legacyKeys []string,
	aiJSON, flagsJSON, secretsJSON []byte,
) (Settings, map[string]string, error) {
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
		st.AI.OpenRouterEnvelope = credentialEnvelopeMetadata(blob)
		decrypted, err := decryptCredential(
			activeKey,
			previousKeys,
			legacyKeys,
			blob,
			openRouterAPIKeyContext,
		)
		if err != nil {
			slog.Warn("runtime settings: dropping undecryptable openrouter api key", "error", err)
		} else {
			st.AI.OpenRouterAPIKey = decrypted.Plaintext
			st.AI.OpenRouterEnvelope.Readable = true
			st.AI.OpenRouterEnvelope.KeyID = decrypted.KeyID
			st.AI.OpenRouterEnvelope.MigrationNeeded = decrypted.NeedsRewrite
			if decrypted.Legacy {
				st.AI.OpenRouterEnvelope.Version = "legacy"
				st.AI.OpenRouterEnvelope.Algorithm = ""
			}
		}
	}
	return st, secrets, nil
}

// degradeSettingsRow never fails: a corrupted row degrades to zero Settings so
// the server boots on env config. Updates refuse to overwrite corrupted rows,
// so the stored settings need manual repair.
func degradeSettingsRow(
	activeKey string,
	previousKeys []string,
	legacyKeys []string,
	aiJSON, flagsJSON, secretsJSON []byte,
) Settings {
	st, _, err := decodeSettingsRow(activeKey, previousKeys, legacyKeys, aiJSON, flagsJSON, secretsJSON)
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
func mergeSecrets(
	secret string,
	prev map[string]string,
	decodedKey, key string,
	operation SecretUpdateOperation,
	forceReencrypt bool,
) (map[string]string, error) {
	secrets := make(map[string]string, len(prev))
	maps.Copy(secrets, prev)
	if operation == SecretClear {
		delete(secrets, openRouterAPIKeySecret)
		return secrets, nil
	}
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
	if countNonWhitespace(secret) < 32 {
		return nil, ErrConfigEncryptionKey
	}
	blob, err := encryptCredential(secret, key, openRouterAPIKeyContext)
	if err != nil {
		return nil, fmt.Errorf("encrypt openrouter api key: %w", err)
	}
	secrets[openRouterAPIKeySecret] = blob
	return secrets, nil
}

func countNonWhitespace(value string) int {
	count := 0
	for _, r := range value {
		if !unicode.IsSpace(r) {
			count++
		}
	}
	return count
}

func sameDesiredSettings(left, right Settings) bool {
	return left.AI.DefaultProvider == right.AI.DefaultProvider &&
		left.AI.OpenRouterModel == right.AI.OpenRouterModel &&
		left.AI.OpenRouterAPIKey == right.AI.OpenRouterAPIKey &&
		maps.Equal(left.Flags, right.Flags)
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
	secrets, err := mergeSecrets(
		secret,
		prevSecrets,
		decodedKey,
		st.AI.OpenRouterAPIKey,
		st.AI.OpenRouterAPIKeyOperation,
		forceReencrypt,
	)
	if err != nil {
		return err
	}
	secretsJSON, err := json.Marshal(secrets)
	if err != nil {
		return fmt.Errorf("marshal secrets: %w", err)
	}

	_, err = tx.Exec(ctx, `
		INSERT INTO runtime_settings (id, ai, flags, secrets, revision, updated_at)
		VALUES (1, $1, $2, $3, $4, now())
		ON CONFLICT (id) DO UPDATE
		SET ai = EXCLUDED.ai, flags = EXCLUDED.flags, secrets = EXCLUDED.secrets,
		    revision = EXCLUDED.revision, updated_at = now()`,
		aiJSON, flagsJSON, secretsJSON, st.Revision)
	if err != nil {
		return fmt.Errorf("save runtime settings: %w", err)
	}
	return nil
}
