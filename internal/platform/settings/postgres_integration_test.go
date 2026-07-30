// Copyright 2026 the P&AI authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

//go:build integration
// +build integration

package settings

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/p-n-ai/pai-bot/internal/platform/config"
	"github.com/p-n-ai/pai-bot/internal/platform/featureflags"
)

func TestStore_SaveLoadRoundtrip(t *testing.T) {
	ctx, pool := settingsTestPool(t)

	store := New(pool, "test-settings-encryption-key-12345", "test-auth-secret", config.AIConfig{}, featureflags.Features{})

	empty, err := store.Load(ctx)
	if err != nil {
		t.Fatalf("Load(missing row) error = %v", err)
	}
	if !reflect.DeepEqual(empty.AI, AISettings{}) || len(empty.Flags) != 0 {
		t.Fatalf("Load(missing row) = %+v, want zero Settings", empty)
	}

	want := Settings{
		AI: AISettings{
			DefaultProvider: stringPointer("openrouter"),
			Providers: ProviderOverrides{APIKey: map[APIKeyProvider]APIKeyProviderOverride{
				APIKeyProviderOpenRouter: {Model: stringPointer("openrouter/auto")},
			}},
			Credentials: map[APIKeyProvider]CredentialOverride{
				APIKeyProviderOpenRouter: {Value: "sk-or-v1-roundtrip", Operation: SecretReplace},
			},
		},
		Flags: map[string]bool{"turn_hooks": true},
	}
	saved, err := store.Update(
		ctx,
		func(Settings) (Settings, error) { return want, nil },
		func(Settings) (PreparedApply, error) { return func() {}, nil },
	)
	if err != nil {
		t.Fatalf("Update() error = %v", err)
	}
	if saved.Revision != 1 {
		t.Fatalf("Update().Revision = %d, want 1", saved.Revision)
	}
	effective := store.Effective()
	if effective.Revision != 1 || effective.AppliedRevision != 1 || effective.Drift {
		t.Fatalf("Effective revision = %d/%d drift=%v, want synced revision 1", effective.Revision, effective.AppliedRevision, effective.Drift)
	}

	var rawAI, rawSecrets string
	err = pool.QueryRow(ctx, `SELECT ai::text, secrets::text FROM runtime_settings WHERE id = 1`).Scan(&rawAI, &rawSecrets)
	if err != nil {
		t.Fatalf("select raw row: %v", err)
	}
	if strings.Contains(rawAI, "roundtrip") || strings.Contains(rawSecrets, "sk-or-v1-roundtrip") {
		t.Fatal("API key stored in plaintext")
	}

	got, err := store.Load(ctx)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if !reflect.DeepEqual(got.AI.DefaultProvider, want.AI.DefaultProvider) ||
		openRouterModel(got.AI) != openRouterModel(want.AI) ||
		openRouterCredential(got.AI).Value != openRouterCredential(want.AI).Value {
		t.Fatalf("Load().AI = %+v, want provider/model/key from %+v", got.AI, want.AI)
	}
	if envelope := openRouterCredential(got.AI).Envelope; !envelope.Stored || !envelope.Readable ||
		envelope.Version != "v1" || envelope.Algorithm != "a256gcm" ||
		envelope.KeyID == "" || envelope.MigrationNeeded {
		t.Fatalf("Load().OpenRouterEnvelope = %+v, want current readable envelope", envelope)
	}
	if got.Revision != 1 {
		t.Fatalf("Load().Revision = %d, want 1", got.Revision)
	}
	if !got.Flags["turn_hooks"] {
		t.Fatalf("Load().Flags = %v, want turn_hooks=true", got.Flags)
	}

	staleStore := New(pool, "test-settings-encryption-key-12345", "test-auth-secret", config.AIConfig{}, featureflags.Features{})
	merged, err := staleStore.Update(ctx, func(cur Settings) (Settings, error) {
		setOpenRouterModel(&cur.AI, stringPointer("deepseek/deepseek-chat"))
		return cur, nil
	}, nil)
	if err != nil {
		t.Fatalf("Update(stale instance) error = %v", err)
	}
	if openRouterCredential(merged.AI).Value != openRouterCredential(want.AI).Value ||
		!reflect.DeepEqual(merged.AI.DefaultProvider, want.AI.DefaultProvider) {
		t.Fatalf("Update(stale instance) = %+v, want key and provider merged from DB row", merged.AI)
	}
	if merged.Revision != 2 {
		t.Fatalf("Update(stale instance).Revision = %d, want 2", merged.Revision)
	}

	noOp, err := staleStore.Update(ctx, func(cur Settings) (Settings, error) {
		return cur, nil
	}, nil)
	if err != nil {
		t.Fatalf("Update(no-op) error = %v", err)
	}
	if noOp.Revision != 2 {
		t.Fatalf("Update(no-op).Revision = %d, want unchanged 2", noOp.Revision)
	}

	if _, err := store.Update(ctx, func(cur Settings) (Settings, error) {
		setOpenRouterCredential(&cur.AI, CredentialOverride{Operation: SecretClear})
		return cur, nil
	}, nil); err != nil {
		t.Fatalf("Update(cleared key) error = %v", err)
	}
	got, err = store.Load(ctx)
	if err != nil {
		t.Fatalf("Load(cleared key) error = %v", err)
	}
	if openRouterCredential(got.AI).Value != "" {
		t.Fatal("cleared API key should not survive a save/load roundtrip")
	}
	if openRouterModel(got.AI) != "deepseek/deepseek-chat" {
		t.Fatalf("Load().AI OpenRouter model = %q, want stale instance's write preserved", openRouterModel(got.AI))
	}
}

func TestStore_UpdatePreservesUndecryptableKeyBlob(t *testing.T) {
	ctx, pool := settingsTestPool(t)

	s1 := New(pool, "settings-encryption-key-one-12345", "secret-one", config.AIConfig{}, featureflags.Features{})
	if _, err := s1.Update(ctx, func(cur Settings) (Settings, error) {
		setOpenRouterCredential(&cur.AI, CredentialOverride{Value: "sk-or-v1-original", Operation: SecretReplace})
		return cur, nil
	}, nil); err != nil {
		t.Fatalf("Update(store key) error = %v", err)
	}
	var before string
	if err := pool.QueryRow(ctx, `SELECT secrets::text FROM runtime_settings WHERE id = 1`).Scan(&before); err != nil {
		t.Fatalf("select secrets: %v", err)
	}

	// Rotated auth secret: the blob no longer decrypts, but a flag-only
	// update must not destroy it.
	s2 := New(pool, "settings-encryption-key-two-12345", "secret-two", config.AIConfig{}, featureflags.Features{})
	if _, err := s2.Update(ctx, func(cur Settings) (Settings, error) {
		cur.Flags = map[string]bool{"turn_hooks": true}
		return cur, nil
	}, nil); err != nil {
		t.Fatalf("Update(flags only) error = %v", err)
	}

	var after string
	if err := pool.QueryRow(ctx, `SELECT secrets::text FROM runtime_settings WHERE id = 1`).Scan(&after); err != nil {
		t.Fatalf("select secrets: %v", err)
	}
	if after != before {
		t.Fatalf("secrets column changed:\nbefore %s\nafter  %s", before, after)
	}
	credentialHealth := openRouterCredential(s2.Current().AI).Envelope
	if !credentialHealth.Stored || credentialHealth.Readable {
		t.Fatalf("credential health = %+v, want stored and unreadable", credentialHealth)
	}

	got, err := s1.Load(ctx)
	if err != nil {
		t.Fatalf("Load(original secret) error = %v", err)
	}
	if openRouterCredential(got.AI).Value != "sk-or-v1-original" {
		t.Fatalf("Load(original secret) credential = %q, want key recoverable after reverting the auth secret", openRouterCredential(got.AI).Value)
	}

	var appliedRevision int64
	cleared, err := s2.Update(ctx, func(cur Settings) (Settings, error) {
		setOpenRouterCredential(&cur.AI, CredentialOverride{Operation: SecretClear})
		return cur, nil
	}, func(st Settings) (PreparedApply, error) {
		return func() { appliedRevision = st.Revision }, nil
	})
	if err != nil {
		t.Fatalf("Update(explicit clear) error = %v", err)
	}
	if cleared.Revision != 3 || appliedRevision != 3 {
		t.Fatalf("clear revision/applied = %d/%d, want 3/3", cleared.Revision, appliedRevision)
	}
	var clearedSecrets string
	if err := pool.QueryRow(ctx, `SELECT secrets::text FROM runtime_settings WHERE id = 1`).Scan(&clearedSecrets); err != nil {
		t.Fatalf("select cleared secrets: %v", err)
	}
	if clearedSecrets != "{}" {
		t.Fatalf("secrets after explicit clear = %s, want {}", clearedSecrets)
	}
}

func TestStore_ConcurrentUpdatesApplyInCommittedRevisionOrder(t *testing.T) {
	ctx, pool := settingsTestPool(t)
	store := New(pool, "", "", config.AIConfig{}, featureflags.Features{})

	firstApplying := make(chan struct{})
	releaseFirst := make(chan struct{})
	errs := make(chan error, 2)
	var mu sync.Mutex
	var applied []int64
	prepare := func(st Settings) (PreparedApply, error) {
		return func() {
			if st.Revision == 1 {
				close(firstApplying)
				<-releaseFirst
			}
			mu.Lock()
			applied = append(applied, st.Revision)
			mu.Unlock()
		}, nil
	}

	go func() {
		_, err := store.Update(ctx, func(st Settings) (Settings, error) {
			setOpenRouterModel(&st.AI, stringPointer("first/model"))
			return st, nil
		}, prepare)
		errs <- err
	}()
	<-firstApplying
	go func() {
		_, err := store.Update(ctx, func(st Settings) (Settings, error) {
			setOpenRouterModel(&st.AI, stringPointer("second/model"))
			return st, nil
		}, prepare)
		errs <- err
	}()
	close(releaseFirst)

	for range 2 {
		if err := <-errs; err != nil {
			t.Fatalf("Update() error = %v", err)
		}
	}
	mu.Lock()
	gotApplied := append([]int64(nil), applied...)
	mu.Unlock()
	if len(gotApplied) != 2 || gotApplied[0] != 1 || gotApplied[1] != 2 {
		t.Fatalf("applied revisions = %v, want [1 2]", gotApplied)
	}
	loaded, err := store.Load(ctx)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if loaded.Revision != 2 || openRouterModel(loaded.AI) != "second/model" {
		t.Fatalf("persisted settings = %+v, want revision 2 second/model", loaded)
	}
	effective := store.Effective()
	if effective.AppliedRevision != 2 || effective.Drift {
		t.Fatalf("effective apply state = %d drift=%v, want applied revision 2", effective.AppliedRevision, effective.Drift)
	}
}

func TestStore_ReconcilesEnvironmentChangesAndPersistedOverridesAcrossRestart(t *testing.T) {
	ctx, pool := settingsTestPool(t)
	env := config.AIConfig{
		DefaultProvider: "openrouter",
		OpenRouter:      config.OpenRouterConfig{Model: "environment/one"},
	}
	store := New(pool, "", "", env, featureflags.Features{})
	if _, err := store.Update(ctx, func(st Settings) (Settings, error) {
		st.AI.DefaultProvider = stringPointer(env.DefaultProvider)
		setOpenRouterModel(&st.AI, stringPointer(env.OpenRouter.Model))
		return st, nil
	}, nil); err != nil {
		t.Fatalf("save redundant overrides: %v", err)
	}
	var rawAI string
	if err := pool.QueryRow(ctx, `SELECT ai::text FROM runtime_settings WHERE id = 1`).Scan(&rawAI); err != nil {
		t.Fatalf("read canonicalized AI settings: %v", err)
	}
	if strings.Contains(rawAI, "environment/one") || strings.Contains(rawAI, "openrouter") {
		t.Fatalf("redundant environment values persisted as overrides: %s", rawAI)
	}

	changedEnv := env
	changedEnv.OpenRouter.Model = "environment/two"
	restarted := New(pool, "", "", changedEnv, featureflags.Features{})
	if err := restarted.Start(ctx); err != nil {
		t.Fatalf("restart with changed environment: %v", err)
	}
	if effective := restarted.Effective(); effective.Providers["openrouter"].Model != "environment/two" ||
		effective.ProviderSources["openrouter"].Model != SourceEnv {
		t.Fatalf("effective after environment change = %q/%q, want environment/two from env",
			effective.Providers["openrouter"].Model, effective.ProviderSources["openrouter"].Model)
	}

	if _, err := restarted.Update(ctx, func(st Settings) (Settings, error) {
		setOpenRouterModel(&st.AI, stringPointer("database/model"))
		return st, nil
	}, nil); err != nil {
		t.Fatalf("save nonredundant override: %v", err)
	}
	thirdEnv := env
	thirdEnv.OpenRouter.Model = "environment/three"
	withOverride := New(pool, "", "", thirdEnv, featureflags.Features{})
	if err := withOverride.Start(ctx); err != nil {
		t.Fatalf("restart with preserved override: %v", err)
	}
	if effective := withOverride.Effective(); effective.Providers["openrouter"].Model != "database/model" ||
		effective.ProviderSources["openrouter"].Model != SourceDB {
		t.Fatalf("effective preserved override = %q/%q, want database/model from db",
			effective.Providers["openrouter"].Model, effective.ProviderSources["openrouter"].Model)
	}

	if _, err := withOverride.Update(ctx, func(st Settings) (Settings, error) {
		setOpenRouterModel(&st.AI, nil)
		return st, nil
	}, nil); err != nil {
		t.Fatalf("reset model override: %v", err)
	}
	finalEnv := env
	finalEnv.OpenRouter.Model = "environment/four"
	afterReset := New(pool, "", "", finalEnv, featureflags.Features{})
	if err := afterReset.Start(ctx); err != nil {
		t.Fatalf("restart after reset: %v", err)
	}
	if effective := afterReset.Effective(); effective.Providers["openrouter"].Model != "environment/four" ||
		effective.ProviderSources["openrouter"].Model != SourceEnv {
		t.Fatalf("effective after reset = %q/%q, want environment/four from env",
			effective.Providers["openrouter"].Model, effective.ProviderSources["openrouter"].Model)
	}
}

func TestStore_CommitFailureLeavesDesiredAndRuntimeUnchanged(t *testing.T) {
	ctx, pool := settingsTestPool(t)
	if _, err := pool.Exec(ctx, `
		DROP FUNCTION IF EXISTS pai_test_reject_runtime_setting() CASCADE;
		CREATE FUNCTION pai_test_reject_runtime_setting() RETURNS trigger
		LANGUAGE plpgsql AS $$
		BEGIN
				IF NEW.ai #>> '{providers,api_key,openrouter,model}' = 'reject-at-commit' THEN
				RAISE EXCEPTION 'sentinel commit rejection';
			END IF;
			RETURN NEW;
		END
		$$;
		CREATE CONSTRAINT TRIGGER pai_test_reject_runtime_setting
		AFTER INSERT OR UPDATE ON runtime_settings
		DEFERRABLE INITIALLY DEFERRED
		FOR EACH ROW EXECUTE FUNCTION pai_test_reject_runtime_setting();
	`); err != nil {
		t.Fatalf("install deferred rejection trigger: %v", err)
	}
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), `DROP FUNCTION IF EXISTS pai_test_reject_runtime_setting() CASCADE`)
	})

	store := New(pool, "", "", config.AIConfig{}, featureflags.Features{})
	var applies atomic.Int32
	_, err := store.Update(ctx, func(st Settings) (Settings, error) {
		setOpenRouterModel(&st.AI, stringPointer("reject-at-commit"))
		return st, nil
	}, func(Settings) (PreparedApply, error) {
		return func() { applies.Add(1) }, nil
	})
	if err == nil || !strings.Contains(err.Error(), "commit runtime settings update") {
		t.Fatalf("Update() error = %v, want safe commit failure", err)
	}
	if applies.Load() != 0 {
		t.Fatalf("apply calls = %d, want zero after commit failure", applies.Load())
	}
	var count int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM runtime_settings`).Scan(&count); err != nil {
		t.Fatalf("count runtime settings: %v", err)
	}
	if count != 0 {
		t.Fatalf("runtime settings rows = %d, want transaction rolled back", count)
	}
	if got := store.Current(); !reflect.DeepEqual(got.AI, AISettings{}) || len(got.Flags) != 0 || got.Revision != 0 {
		t.Fatalf("Current() = %+v, want unchanged zero settings", got)
	}
}

func TestStore_UpdateMigratesLegacyCiphertextAndRestartsWithActiveKeyOnly(t *testing.T) {
	const (
		legacyAuthKey = "legacy-auth-key"
		encryptionKey = "dedicated-settings-encryption-key-123"
		apiKey        = "sk-or-v1-legacy"
	)
	for _, tt := range []struct {
		name      string
		legacyKey string
		authKey   string
	}{
		{name: "PR 224 dedicated key", legacyKey: encryptionKey},
		{name: "pre PR 224 auth key", legacyKey: legacyAuthKey, authKey: legacyAuthKey},
	} {
		t.Run(tt.name, func(t *testing.T) {
			ctx, pool := settingsTestPool(t)
			legacyBlob, err := encryptString(tt.legacyKey, apiKey)
			if err != nil {
				t.Fatalf("encrypt legacy key: %v", err)
			}
			if _, err := pool.Exec(ctx, `
				INSERT INTO runtime_settings (id, ai, flags, secrets)
				VALUES (1, '{}', '{}', jsonb_build_object('openrouter_api_key', $1::text))
			`, legacyBlob); err != nil {
				t.Fatalf("insert legacy settings: %v", err)
			}

			store := New(pool, encryptionKey, tt.authKey, config.AIConfig{}, featureflags.Features{})
			if _, err := store.Update(ctx, func(cur Settings) (Settings, error) {
				cur.Flags = map[string]bool{"turn_hooks": true}
				return cur, nil
			}, nil); err != nil {
				t.Fatalf("Update() error = %v", err)
			}

			var secretsJSON []byte
			if err := pool.QueryRow(ctx, `SELECT secrets FROM runtime_settings WHERE id = 1`).Scan(&secretsJSON); err != nil {
				t.Fatalf("select migrated secrets: %v", err)
			}
			var secrets map[string]string
			if err := json.Unmarshal(secretsJSON, &secrets); err != nil {
				t.Fatalf("decode migrated secrets: %v", err)
			}
			migratedBlob := secrets[credentialSecretName(APIKeyProviderOpenRouter)]
			if legacyBlob == migratedBlob || !strings.HasPrefix(migratedBlob, credentialEnvelopePrefix) {
				t.Fatalf("migrated ciphertext = %q, want new versioned envelope", migratedBlob)
			}
			if _, exists := secrets[legacyOpenRouterAPIKeySecret]; exists {
				t.Fatal("legacy secret slot survived canonical migration")
			}

			restarted := New(pool, encryptionKey, "", config.AIConfig{}, featureflags.Features{})
			if err := restarted.Start(ctx); err != nil {
				t.Fatalf("restart with active key only: %v", err)
			}
			if got := openRouterCredential(restarted.Current().AI).Value; got != apiKey {
				t.Fatalf("restarted key = %q, want %q", got, apiKey)
			}
		})
	}
}

func TestStore_UpdateRotatesRetiredVersionedCiphertext(t *testing.T) {
	ctx, pool := settingsTestPool(t)

	const (
		retiredKey = "retired-settings-encryption-key-12345"
		activeKey  = "active-settings-encryption-key-123456"
		apiKey     = "sk-or-v1-rotated"
	)
	retiredStore := New(pool, retiredKey, "", config.AIConfig{}, featureflags.Features{})
	if _, err := retiredStore.Update(ctx, func(cur Settings) (Settings, error) {
		setOpenRouterCredential(&cur.AI, CredentialOverride{Value: apiKey, Operation: SecretReplace})
		return cur, nil
	}, nil); err != nil {
		t.Fatalf("write retired-key envelope: %v", err)
	}

	var retiredSecretsJSON []byte
	if err := pool.QueryRow(ctx, `SELECT secrets FROM runtime_settings WHERE id = 1`).Scan(&retiredSecretsJSON); err != nil {
		t.Fatalf("select retired-key envelope: %v", err)
	}
	var retiredSecrets map[string]string
	if err := json.Unmarshal(retiredSecretsJSON, &retiredSecrets); err != nil {
		t.Fatalf("decode retired-key secrets: %v", err)
	}
	retiredBlob := retiredSecrets[credentialSecretName(APIKeyProviderOpenRouter)]
	if !strings.Contains(retiredBlob, credentialKeyID(retiredKey)) {
		t.Fatalf("retired envelope does not contain its derived key ID")
	}

	activeStore := NewWithPreviousKeys(
		pool,
		activeKey,
		[]string{retiredKey},
		"",
		config.AIConfig{},
		featureflags.Features{},
	)
	if _, err := activeStore.Update(ctx, func(cur Settings) (Settings, error) {
		cur.Flags = map[string]bool{"turn_hooks": true}
		return cur, nil
	}, nil); err != nil {
		t.Fatalf("rotate retired-key envelope: %v", err)
	}

	var activeSecretsJSON []byte
	if err := pool.QueryRow(ctx, `SELECT secrets FROM runtime_settings WHERE id = 1`).Scan(&activeSecretsJSON); err != nil {
		t.Fatalf("select active envelope: %v", err)
	}
	var activeSecrets map[string]string
	if err := json.Unmarshal(activeSecretsJSON, &activeSecrets); err != nil {
		t.Fatalf("decode active secrets: %v", err)
	}
	activeBlob := activeSecrets[credentialSecretName(APIKeyProviderOpenRouter)]
	if activeBlob == retiredBlob || !strings.Contains(activeBlob, credentialKeyID(activeKey)) {
		t.Fatal("retired envelope was not rewritten with the active key")
	}

	restarted := New(pool, activeKey, "", config.AIConfig{}, featureflags.Features{})
	if err := restarted.Start(ctx); err != nil {
		t.Fatalf("restart without retired key: %v", err)
	}
	if got := openRouterCredential(restarted.Current().AI).Value; got != apiKey {
		t.Fatalf("restarted key = %q, want %q", got, apiKey)
	}
}

func TestStore_UpdateRejectsCorruptRow(t *testing.T) {
	ctx, pool := settingsTestPool(t)

	store := New(pool, "test-settings-encryption-key-12345", "test-auth-secret", config.AIConfig{}, featureflags.Features{})
	if _, err := store.Update(ctx, func(cur Settings) (Settings, error) {
		setOpenRouterModel(&cur.AI, stringPointer("openrouter/auto"))
		return cur, nil
	}, nil); err != nil {
		t.Fatalf("Update(seed) error = %v", err)
	}
	// jsonb rejects invalid JSON, so corrupt the shape instead: a string
	// where a flag object is expected.
	if _, err := pool.Exec(ctx, `UPDATE runtime_settings SET flags = '"corrupt"'::jsonb WHERE id = 1`); err != nil {
		t.Fatalf("corrupt flags column: %v", err)
	}
	var before string
	if err := pool.QueryRow(ctx, `SELECT ai::text || flags::text || secrets::text FROM runtime_settings WHERE id = 1`).Scan(&before); err != nil {
		t.Fatalf("select row: %v", err)
	}

	if _, err := store.Update(ctx, func(cur Settings) (Settings, error) {
		cur.Flags = map[string]bool{"turn_hooks": true}
		return cur, nil
	}, nil); err == nil {
		t.Fatal("Update() should refuse to rebuild a corrupt row")
	}

	var after string
	if err := pool.QueryRow(ctx, `SELECT ai::text || flags::text || secrets::text FROM runtime_settings WHERE id = 1`).Scan(&after); err != nil {
		t.Fatalf("select row: %v", err)
	}
	if after != before {
		t.Fatalf("row changed after failed Update:\nbefore %s\nafter  %s", before, after)
	}

	got, err := store.Load(ctx)
	if err != nil {
		t.Fatalf("Load(corrupt row) error = %v", err)
	}
	if !reflect.DeepEqual(got.AI, AISettings{}) || len(got.Flags) != 0 {
		t.Fatalf("Load(corrupt row) = %+v, want degraded zero Settings", got)
	}
}

func setOpenRouterModel(ai *AISettings, model *string) {
	if ai.Providers.APIKey == nil {
		ai.Providers.APIKey = make(map[APIKeyProvider]APIKeyProviderOverride)
	}
	if model == nil {
		delete(ai.Providers.APIKey, APIKeyProviderOpenRouter)
		return
	}
	ai.Providers.APIKey[APIKeyProviderOpenRouter] = APIKeyProviderOverride{Model: model}
}

func openRouterModel(ai AISettings) string {
	model := ai.Providers.APIKey[APIKeyProviderOpenRouter].Model
	if model == nil {
		return ""
	}
	return *model
}

func setOpenRouterCredential(ai *AISettings, credential CredentialOverride) {
	if ai.Credentials == nil {
		ai.Credentials = make(map[APIKeyProvider]CredentialOverride)
	}
	ai.Credentials[APIKeyProviderOpenRouter] = credential
}

func openRouterCredential(ai AISettings) CredentialOverride {
	return ai.Credentials[APIKeyProviderOpenRouter]
}

func settingsTestPool(t *testing.T) (context.Context, *pgxpool.Pool) {
	t.Helper()

	dbURL := strings.TrimSpace(os.Getenv("LEARN_TEST_DATABASE_URL"))
	if dbURL == "" {
		t.Skip("LEARN_TEST_DATABASE_URL is not set; skipping runtime settings postgres test")
	}
	ctx := context.Background()
	pool, err := pgxpool.New(ctx, dbURL)
	if err != nil {
		t.Fatalf("pgxpool.New() error = %v", err)
	}
	t.Cleanup(pool.Close)
	applyRuntimeSettingsMigration(t, ctx, pool)
	return ctx, pool
}

func applyRuntimeSettingsMigration(t *testing.T, ctx context.Context, pool *pgxpool.Pool) {
	t.Helper()

	if _, err := pool.Exec(ctx, `DROP TABLE IF EXISTS runtime_settings`); err != nil {
		t.Fatalf("drop runtime_settings: %v", err)
	}
	for _, name := range []string{
		"20260705090000_runtime_settings.sql",
		"20260729100000_runtime_settings_revision.sql",
	} {
		path := filepath.Join("..", "..", "..", "migrations", name)
		if _, err := pool.Exec(ctx, migrationUpSQL(t, path)); err != nil {
			t.Fatalf("apply migration %s: %v", path, err)
		}
	}
}

func migrationUpSQL(t *testing.T, path string) string {
	t.Helper()
	sqlBytes, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read migration %s: %v", path, err)
	}
	up := string(sqlBytes)
	if i := strings.Index(up, "-- +goose Up"); i >= 0 {
		up = up[i+len("-- +goose Up"):]
	}
	if i := strings.Index(up, "-- +goose Down"); i >= 0 {
		up = up[:i]
	}
	return up
}
