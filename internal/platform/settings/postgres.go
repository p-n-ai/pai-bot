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
	"reflect"
	"slices"
	"sync"
	"unicode"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/p-n-ai/pai-bot/internal/platform/config"
	"github.com/p-n-ai/pai-bot/internal/platform/featureflags"
)

const legacyOpenRouterAPIKeySecret = "openrouter_api_key"

func credentialSecretName(provider APIKeyProvider) string {
	return "provider/" + string(provider) + "/api_key"
}

func apiKeyCredentialContext(provider APIKeyProvider) credentialContext {
	return credentialContext{Provider: string(provider), Slot: "api_key"}
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
	decodedKeys := make(map[APIKeyProvider]string, len(apiKeyProviders))
	reencryptKeys := make(map[APIKeyProvider]bool, len(apiKeyProviders))
	for _, provider := range apiKeyProviders {
		credential := cur.AI.Credentials[provider]
		decodedKeys[provider] = credential.Value
		blob, legacy, blobErr := storedCredentialBlob(prevSecrets, provider)
		if blobErr != nil {
			return Settings{}, fmt.Errorf("decode runtime settings for update: %w", blobErr)
		}
		reencryptKeys[provider] = legacy || s.secretNeedsReencryption(provider, blob, credential.Value)
	}
	st, err := mutate(cloneSettingsForMutation(cur))
	if err != nil {
		return Settings{}, err
	}
	s.canonicalizeRedundantOverrides(&st)
	explicitStoredKeyClear := false
	for _, provider := range apiKeyProviders {
		credential := st.AI.Credentials[provider]
		blob, _, blobErr := storedCredentialBlob(prevSecrets, provider)
		if blobErr != nil {
			return Settings{}, fmt.Errorf("decode runtime settings for update: %w", blobErr)
		}
		explicitStoredKeyClear = explicitStoredKeyClear ||
			(credential.Operation == SecretClear && blob != "")
	}
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
	if err := saveSettingsRow(ctx, tx, s.writeEncryptionKey(), st, prevSecrets, decodedKeys, reencryptKeys); err != nil {
		return Settings{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return Settings{}, fmt.Errorf("commit runtime settings update: %w", err)
	}
	for _, provider := range apiKeyProviders {
		credential, ok := st.AI.Credentials[provider]
		if !ok {
			continue
		}
		switch {
		case credential.Operation == SecretClear:
			credential.Envelope = CredentialEnvelopeStatus{}
		case credential.Operation == SecretReplace ||
			(reencryptKeys[provider] && credential.Value != ""):
			credential.Envelope = CredentialEnvelopeStatus{
				Stored:    true,
				Readable:  true,
				Version:   "v1",
				Algorithm: credentialEnvelopeAlgorithm,
				KeyID:     credentialKeyID(s.writeEncryptionKey()),
			}
		}
		credential.Operation = SecretPreserve
		st.AI.Credentials[provider] = credential
	}
	s.setCurrent(st)
	if apply != nil {
		apply()
		s.setAppliedRevision(st.Revision)
	}
	return st, nil
}

func cloneSettingsForMutation(st Settings) Settings {
	cloned := Settings{
		AI: AISettings{
			DefaultProvider: cloneStringPointer(st.AI.DefaultProvider),
			Providers: ProviderOverrides{
				APIKey: make(map[APIKeyProvider]APIKeyProviderOverride, len(st.AI.Providers.APIKey)),
			},
			Credentials: make(map[APIKeyProvider]CredentialOverride, len(st.AI.Credentials)),
		},
		Flags:    maps.Clone(st.Flags),
		Revision: st.Revision,
	}
	for provider, override := range st.AI.Providers.APIKey {
		cloned.AI.Providers.APIKey[provider] = APIKeyProviderOverride{
			Model: cloneStringPointer(override.Model),
		}
	}
	if st.AI.Providers.Ollama != nil {
		cloned.AI.Providers.Ollama = &EnabledModelOverride{
			Enabled: cloneBoolPointer(st.AI.Providers.Ollama.Enabled),
			Model:   cloneStringPointer(st.AI.Providers.Ollama.Model),
		}
	}
	if st.AI.Providers.ManagedCodex != nil {
		cloned.AI.Providers.ManagedCodex = &ModelOverride{
			Model: cloneStringPointer(st.AI.Providers.ManagedCodex.Model),
		}
	}
	maps.Copy(cloned.AI.Credentials, st.AI.Credentials)
	return cloned
}

// canonicalizeRedundantOverrides keeps the database as an override layer
// rather than a copy of the boot environment. A later deployment can then
// change the baseline without an equal historical override masking it.
func (s *Store) canonicalizeRedundantOverrides(st *Settings) {
	if st.AI.DefaultProvider != nil && *st.AI.DefaultProvider == s.envAI.DefaultProvider {
		st.AI.DefaultProvider = nil
	}
	for provider, override := range st.AI.Providers.APIKey {
		envModel, envKey := apiKeyConfig(s.envAI, provider)
		if override.Model != nil && *override.Model == envModel {
			override.Model = nil
		}
		if override.Model == nil {
			delete(st.AI.Providers.APIKey, provider)
		} else {
			st.AI.Providers.APIKey[provider] = override
		}
		if credential, ok := st.AI.Credentials[provider]; ok &&
			credential.Value != "" && credential.Value == envKey {
			credential.Value = ""
			credential.Operation = SecretClear
			st.AI.Credentials[provider] = credential
		}
	}
	canonicalizeEnabledModelOverride(&st.AI.Providers.Ollama, s.envAI.Ollama.Enabled, s.envAI.Ollama.Model)
	canonicalizeModelOverride(&st.AI.Providers.ManagedCodex, s.envAI.Codex.Model)
	for provider, credential := range st.AI.Credentials {
		if _, known := ParseAPIKeyProvider(string(provider)); !known {
			delete(st.AI.Credentials, provider)
			continue
		}
		_, envKey := apiKeyConfig(s.envAI, provider)
		if credential.Value != "" && credential.Value == envKey {
			credential.Value = ""
			credential.Operation = SecretClear
			st.AI.Credentials[provider] = credential
		}
	}
	for name, value := range st.Flags {
		if value == s.envFlags.Enabled(featureflags.Feature(name)) {
			delete(st.Flags, name)
		}
	}
}

func canonicalizeModelOverride(override **ModelOverride, envModel string) {
	if *override == nil {
		return
	}
	value := **override
	if value.Model != nil && *value.Model == envModel {
		value.Model = nil
	}
	if value.Model == nil {
		*override = nil
		return
	}
	*override = &value
}

func canonicalizeEnabledModelOverride(override **EnabledModelOverride, envEnabled bool, envModel string) {
	if *override == nil {
		return
	}
	value := **override
	if value.Enabled != nil && *value.Enabled == envEnabled {
		value.Enabled = nil
	}
	if value.Model != nil && *value.Model == envModel {
		value.Model = nil
	}
	if value.Enabled == nil && value.Model == nil {
		*override = nil
		return
	}
	*override = &value
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

func (s *Store) secretNeedsReencryption(provider APIKeyProvider, blob, plaintext string) bool {
	key := s.writeEncryptionKey()
	if blob == "" || plaintext == "" || key == "" {
		return false
	}
	decrypted, err := decryptCredential(
		key,
		s.previousEncryptionKeys,
		s.legacyKeys(),
		blob,
		apiKeyCredentialContext(provider),
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
	if err := decodeAISettings(aiJSON, &st.AI); err != nil {
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
	for _, provider := range apiKeyProviders {
		blob, _, err := storedCredentialBlob(secrets, provider)
		if err != nil {
			return Settings{}, nil, err
		}
		if blob == "" {
			continue
		}
		credential := CredentialOverride{Envelope: credentialEnvelopeMetadata(blob)}
		decrypted, err := decryptCredential(
			activeKey,
			previousKeys,
			legacyKeys,
			blob,
			apiKeyCredentialContext(provider),
		)
		if err != nil {
			slog.Warn("runtime settings: dropping undecryptable provider credential", "provider", provider, "error", err)
		} else {
			credential.Value = decrypted.Plaintext
			credential.Envelope.Readable = true
			credential.Envelope.KeyID = decrypted.KeyID
			credential.Envelope.MigrationNeeded = decrypted.NeedsRewrite
			if decrypted.Legacy {
				credential.Envelope.Version = "legacy"
				credential.Envelope.Algorithm = ""
			}
		}
		if st.AI.Credentials == nil {
			st.AI.Credentials = make(map[APIKeyProvider]CredentialOverride)
		}
		st.AI.Credentials[provider] = credential
	}
	return st, secrets, nil
}

func decodeAISettings(data []byte, target *AISettings) error {
	var wire struct {
		DefaultProvider *string            `json:"default_provider"`
		Providers       *ProviderOverrides `json:"providers"`
		OpenRouterModel *string            `json:"openrouter_model"`
	}
	if err := json.Unmarshal(data, &wire); err != nil {
		return err
	}
	if wire.DefaultProvider != nil && *wire.DefaultProvider != "" {
		target.DefaultProvider = cloneStringPointer(wire.DefaultProvider)
	}
	if wire.Providers != nil {
		target.Providers = *wire.Providers
	}
	for provider, override := range target.Providers.APIKey {
		if _, ok := ParseAPIKeyProvider(string(provider)); !ok {
			return fmt.Errorf("unknown API-key provider %q", provider)
		}
		if override.Model != nil && *override.Model == "" {
			return fmt.Errorf("empty model override for provider %q", provider)
		}
	}
	if wire.OpenRouterModel != nil && *wire.OpenRouterModel != "" {
		current, exists := target.Providers.APIKey[APIKeyProviderOpenRouter]
		if exists && current.Model != nil {
			return errors.New("ambiguous canonical and legacy OpenRouter model overrides")
		}
		if target.Providers.APIKey == nil {
			target.Providers.APIKey = make(map[APIKeyProvider]APIKeyProviderOverride)
		}
		current.Model = cloneStringPointer(wire.OpenRouterModel)
		target.Providers.APIKey[APIKeyProviderOpenRouter] = current
	}
	return nil
}

func storedCredentialBlob(secrets map[string]string, provider APIKeyProvider) (blob string, legacy bool, err error) {
	canonical := secrets[credentialSecretName(provider)]
	if provider != APIKeyProviderOpenRouter {
		return canonical, false, nil
	}
	old := secrets[legacyOpenRouterAPIKeySecret]
	if canonical != "" && old != "" {
		return "", false, errors.New("ambiguous canonical and legacy OpenRouter credentials")
	}
	if canonical != "" {
		return canonical, false, nil
	}
	return old, old != "", nil
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

// mergeSecrets returns prev with only known credential slots changed.
func mergeSecrets(
	secret string,
	prev map[string]string,
	decodedKeys map[APIKeyProvider]string,
	credentials map[APIKeyProvider]CredentialOverride,
	forceReencrypt map[APIKeyProvider]bool,
) (map[string]string, error) {
	secrets := make(map[string]string, len(prev))
	maps.Copy(secrets, prev)
	for _, provider := range apiKeyProviders {
		credential := credentials[provider]
		canonicalName := credentialSecretName(provider)
		if credential.Operation == SecretClear {
			delete(secrets, canonicalName)
			if provider == APIKeyProviderOpenRouter {
				delete(secrets, legacyOpenRouterAPIKeySecret)
			}
			continue
		}
		key := credential.Value
		if key == decodedKeys[provider] && !forceReencrypt[provider] {
			continue
		}
		if key == "" {
			// An unreadable stored blob decodes to an empty value. Preserve it
			// unless the caller explicitly clears the slot.
			continue
		}
		if countNonWhitespace(secret) < 32 {
			return nil, ErrConfigEncryptionKey
		}
		blob, err := encryptCredential(secret, key, apiKeyCredentialContext(provider))
		if err != nil {
			return nil, fmt.Errorf("encrypt %s api key: %w", provider, err)
		}
		secrets[canonicalName] = blob
		if provider == APIKeyProviderOpenRouter {
			delete(secrets, legacyOpenRouterAPIKeySecret)
		}
	}
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
	return reflect.DeepEqual(left.AI.DefaultProvider, right.AI.DefaultProvider) &&
		reflect.DeepEqual(left.AI.Providers, right.AI.Providers) &&
		equalCredentialValues(left.AI.Credentials, right.AI.Credentials) &&
		maps.Equal(left.Flags, right.Flags)
}

func equalCredentialValues(left, right map[APIKeyProvider]CredentialOverride) bool {
	for _, provider := range apiKeyProviders {
		if left[provider].Value != right[provider].Value {
			return false
		}
	}
	return true
}

// saveSettingsRow upserts the settings row; prevSecrets and decoded keys come
// from the strict decode of the locked row (see mergeSecrets).
func saveSettingsRow(
	ctx context.Context,
	tx pgx.Tx,
	secret string,
	st Settings,
	prevSecrets map[string]string,
	decodedKeys map[APIKeyProvider]string,
	forceReencrypt map[APIKeyProvider]bool,
) error {
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
		decodedKeys,
		st.AI.Credentials,
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
