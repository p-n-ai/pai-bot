// Copyright 2026 the P&AI authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package settings

import (
	"bytes"
	"encoding/json"
	"errors"
	"log/slog"
	"strings"
	"testing"

	"github.com/p-n-ai/pai-bot/internal/platform/config"
	"github.com/p-n-ai/pai-bot/internal/platform/featureflags"
)

func envAIConfig() config.AIConfig {
	return config.AIConfig{
		DefaultProvider: "openai",
		OpenRouter: config.OpenRouterConfig{
			APIKey: "env-key",
			Model:  "env-model",
		},
	}
}

func TestMergeAI(t *testing.T) {
	tests := []struct {
		name         string
		st           Settings
		wantProvider string
		wantModel    string
		wantKey      string
	}{
		{
			name:         "env only",
			st:           Settings{},
			wantProvider: "openai",
			wantModel:    "env-model",
			wantKey:      "env-key",
		},
		{
			name: "db overrides all",
			st: Settings{AI: AISettings{
				DefaultProvider:  "openrouter",
				OpenRouterModel:  "db-model",
				OpenRouterAPIKey: "db-key",
			}},
			wantProvider: "openrouter",
			wantModel:    "db-model",
			wantKey:      "db-key",
		},
		{
			name: "db empty fields keep env",
			st: Settings{AI: AISettings{
				OpenRouterModel: "db-model",
			}},
			wantProvider: "openai",
			wantModel:    "db-model",
			wantKey:      "env-key",
		},
		{
			name: "cleared db key falls back to env key",
			st: Settings{AI: AISettings{
				DefaultProvider:  "openrouter",
				OpenRouterAPIKey: "",
			}},
			wantProvider: "openrouter",
			wantModel:    "env-model",
			wantKey:      "env-key",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			merged := MergeAI(envAIConfig(), tt.st)
			if merged.DefaultProvider != tt.wantProvider {
				t.Fatalf("DefaultProvider = %q, want %q", merged.DefaultProvider, tt.wantProvider)
			}
			if merged.OpenRouter.Model != tt.wantModel {
				t.Fatalf("OpenRouter.Model = %q, want %q", merged.OpenRouter.Model, tt.wantModel)
			}
			if merged.OpenRouter.APIKey != tt.wantKey {
				t.Fatalf("OpenRouter.APIKey = %q, want %q", merged.OpenRouter.APIKey, tt.wantKey)
			}
		})
	}
}

func TestEffective(t *testing.T) {
	envFlags, err := featureflags.Parse("turn_hooks=true")
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}

	t.Run("env only", func(t *testing.T) {
		eff := Effective(envAIConfig(), envFlags, Settings{})
		if eff.DefaultProvider != "openai" || eff.DefaultProviderSource != SourceEnv {
			t.Fatalf("DefaultProvider = %q (%s), want openai (env)", eff.DefaultProvider, eff.DefaultProviderSource)
		}
		if !eff.OpenRouterKeySet || eff.OpenRouterKeyLast4 != "" || eff.OpenRouterKeySource != SourceEnv {
			t.Fatalf("OpenRouterKey = set:%v last4:%q (%s), want set with no hint (env)", eff.OpenRouterKeySet, eff.OpenRouterKeyLast4, eff.OpenRouterKeySource)
		}
		if !eff.Flags["turn_hooks"] || eff.FlagSources["turn_hooks"] != SourceEnv {
			t.Fatalf("turn_hooks = %v (%s), want true (env)", eff.Flags["turn_hooks"], eff.FlagSources["turn_hooks"])
		}
		if !eff.Baseline.Flags["turn_hooks"] || !eff.Effective.Flags["turn_hooks"] {
			t.Fatalf("flag projections = baseline:%v effective:%v, want true/true",
				eff.Baseline.Flags, eff.Effective.Flags)
		}
		if len(eff.Override.Flags) != 0 {
			t.Fatalf("Override.Flags = %v, want no DB overrides", eff.Override.Flags)
		}
	})

	t.Run("db overrides env", func(t *testing.T) {
		eff := Effective(envAIConfig(), envFlags, Settings{
			AI:    AISettings{DefaultProvider: "openrouter", OpenRouterModel: "db-model", OpenRouterAPIKey: "db-secret-1234"},
			Flags: map[string]bool{"turn_hooks": false},
		})
		if eff.DefaultProvider != "openrouter" || eff.DefaultProviderSource != SourceDB {
			t.Fatalf("DefaultProvider = %q (%s), want openrouter (db)", eff.DefaultProvider, eff.DefaultProviderSource)
		}
		if !eff.OpenRouterKeySet || eff.OpenRouterKeyLast4 != "1234" || eff.OpenRouterKeySource != SourceDB {
			t.Fatalf("OpenRouterKey = set:%v last4:%q (%s), want set with last4 1234 (db)", eff.OpenRouterKeySet, eff.OpenRouterKeyLast4, eff.OpenRouterKeySource)
		}
		if eff.OpenRouterModel != "db-model" || eff.OpenRouterModelSource != SourceDB {
			t.Fatalf("OpenRouterModel = %q (%s), want db-model (db)", eff.OpenRouterModel, eff.OpenRouterModelSource)
		}
		if eff.Flags["turn_hooks"] || eff.FlagSources["turn_hooks"] != SourceDB {
			t.Fatalf("turn_hooks = %v (%s), want false (db)", eff.Flags["turn_hooks"], eff.FlagSources["turn_hooks"])
		}
		if !eff.Baseline.Flags["turn_hooks"] || eff.Override.Flags["turn_hooks"] ||
			eff.Effective.Flags["turn_hooks"] {
			t.Fatalf("flag projections = baseline:%v override:%v effective:%v, want true/false/false",
				eff.Baseline.Flags, eff.Override.Flags, eff.Effective.Flags)
		}
		if eff.Baseline.DefaultProvider != "openai" || eff.Baseline.OpenRouterModel != "env-model" {
			t.Fatalf("Baseline = %+v, want env provider/model", eff.Baseline)
		}
		if eff.Baseline.OpenRouterKey.Last4 != "" || !eff.Baseline.OpenRouterKey.Set {
			t.Fatalf("Baseline.OpenRouterKey = %+v, want redacted env key", eff.Baseline.OpenRouterKey)
		}
		if eff.Override.DefaultProvider == nil || *eff.Override.DefaultProvider != "openrouter" {
			t.Fatalf("Override.DefaultProvider = %#v, want openrouter", eff.Override.DefaultProvider)
		}
		if eff.Override.OpenRouterModel == nil || *eff.Override.OpenRouterModel != "db-model" {
			t.Fatalf("Override.OpenRouterModel = %#v, want db-model", eff.Override.OpenRouterModel)
		}
		if !eff.Override.OpenRouterKey.Set || eff.Override.OpenRouterKey.Last4 != "1234" {
			t.Fatalf("Override.OpenRouterKey = %+v, want redacted DB key", eff.Override.OpenRouterKey)
		}
		if eff.Effective.DefaultProvider != eff.DefaultProvider ||
			eff.Effective.OpenRouterModel != eff.OpenRouterModel ||
			eff.Effective.OpenRouterKey.Last4 != eff.OpenRouterKeyLast4 {
			t.Fatalf("Effective projection = %+v, aliases = provider:%q model:%q key:%q", eff.Effective, eff.DefaultProvider, eff.OpenRouterModel, eff.OpenRouterKeyLast4)
		}
	})

	t.Run("explicit false env override retains env source", func(t *testing.T) {
		explicitFalse, err := featureflags.Parse("turn_hooks=false")
		if err != nil {
			t.Fatalf("featureflags.Parse() error = %v", err)
		}
		eff := Effective(config.AIConfig{}, explicitFalse, Settings{})
		if eff.Flags["turn_hooks"] || eff.FlagSources["turn_hooks"] != SourceEnv {
			t.Fatalf("turn_hooks = %v (%s), want false from env",
				eff.Flags["turn_hooks"], eff.FlagSources["turn_hooks"])
		}
		if eff.Baseline.Flags["turn_hooks"] || eff.Effective.Flags["turn_hooks"] {
			t.Fatalf("flag projections = baseline:%v effective:%v, want false/false",
				eff.Baseline.Flags, eff.Effective.Flags)
		}
		if len(eff.Override.Flags) != 0 {
			t.Fatalf("Override.Flags = %v, want no DB overrides", eff.Override.Flags)
		}
	})

	t.Run("nothing set", func(t *testing.T) {
		eff := Effective(config.AIConfig{}, featureflags.Features{}, Settings{})
		if eff.DefaultProvider != "" || eff.DefaultProviderSource != SourceNone {
			t.Fatalf("DefaultProvider = %q (%s), want empty (none)", eff.DefaultProvider, eff.DefaultProviderSource)
		}
		if eff.Flags["turn_hooks"] || eff.FlagSources["turn_hooks"] != SourceNone {
			t.Fatalf("turn_hooks = %v (%s), want false (none)", eff.Flags["turn_hooks"], eff.FlagSources["turn_hooks"])
		}
	})
}

func TestKeyLast4(t *testing.T) {
	tests := []struct {
		key  string
		want string
	}{
		{key: "", want: ""},
		{key: "abc", want: ""},
		{key: "sk-1234", want: ""},
		{key: "sk-or-v1-abcd1234", want: "1234"},
	}
	for _, tt := range tests {
		if got := KeyLast4(tt.key); got != tt.want {
			t.Fatalf("KeyLast4(%q) = %q, want %q", tt.key, got, tt.want)
		}
	}
}

func TestDecodeSettingsRowDropsUndecryptableKey(t *testing.T) {
	blob, err := encryptString("old-auth-secret", "sk-or-v1-oldkey")
	if err != nil {
		t.Fatalf("encryptString() error = %v", err)
	}

	st, secrets, err := decodeSettingsRow("rotated-auth-secret", nil, nil,
		[]byte(`{"default_provider":"openrouter","openrouter_model":"m"}`),
		[]byte(`{"turn_hooks":true}`),
		[]byte(`{"openrouter_api_key":"`+blob+`"}`))
	if err != nil {
		t.Fatalf("decodeSettingsRow() error = %v", err)
	}

	if st.AI.OpenRouterAPIKey != "" {
		t.Fatal("undecryptable key must be dropped, not returned")
	}
	if secrets["openrouter_api_key"] != blob {
		t.Fatal("raw secrets map must keep the undecryptable blob")
	}
	if st.AI.DefaultProvider != "openrouter" || st.AI.OpenRouterModel != "m" || !st.Flags["turn_hooks"] {
		t.Fatalf("decodeSettingsRow() = %+v, want other settings kept", st)
	}
}

func TestDecodeSettingsRowPrunesUnknownFlags(t *testing.T) {
	st, _, err := decodeSettingsRow("secret", nil, nil,
		[]byte(`{}`),
		[]byte(`{"turn_hooks":true,"ghost_flag":true}`),
		[]byte(`{}`))
	if err != nil {
		t.Fatalf("decodeSettingsRow() error = %v", err)
	}
	if _, ok := st.Flags["ghost_flag"]; ok {
		t.Fatalf("Flags = %v, want ghost_flag pruned", st.Flags)
	}
	if !st.Flags["turn_hooks"] {
		t.Fatalf("Flags = %v, want turn_hooks kept", st.Flags)
	}
}

func TestDecodeSettingsRowCorruptedJSON(t *testing.T) {
	good := []byte(`{}`)
	bad := []byte(`{corrupt`)
	tests := []struct {
		name                       string
		aiJSON, flagsJSON, secrets []byte
	}{
		{"ai", bad, good, good},
		{"flags", good, bad, good},
		{"secrets", good, good, bad},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if _, _, err := decodeSettingsRow("secret", nil, nil, tt.aiJSON, tt.flagsJSON, tt.secrets); err == nil {
				t.Fatal("decodeSettingsRow() should reject corrupted jsonb")
			}
			st := degradeSettingsRow("secret", nil, nil, tt.aiJSON, tt.flagsJSON, tt.secrets)
			if st.AI != (AISettings{}) || len(st.Flags) != 0 {
				t.Fatalf("degradeSettingsRow() = %+v, want zero Settings", st)
			}
		})
	}
}

func TestMergeSecrets(t *testing.T) {
	prev := map[string]string{openRouterAPIKeySecret: "stored-blob"}

	t.Run("unchanged key preserves blob", func(t *testing.T) {
		got, err := mergeSecrets("s", prev, "sk-old", "sk-old", SecretPreserve, false)
		if err != nil {
			t.Fatalf("mergeSecrets() error = %v", err)
		}
		if got[openRouterAPIKeySecret] != "stored-blob" {
			t.Fatalf("secrets = %v, want stored blob preserved", got)
		}
	})

	t.Run("undecryptable blob survives unrelated update", func(t *testing.T) {
		// Decoded key is "" because the blob did not decrypt; mutate left it "".
		got, err := mergeSecrets("s", prev, "", "", SecretPreserve, false)
		if err != nil {
			t.Fatalf("mergeSecrets() error = %v", err)
		}
		if got[openRouterAPIKeySecret] != "stored-blob" {
			t.Fatalf("secrets = %v, want undecryptable blob preserved", got)
		}
	})

	t.Run("explicit clear deletes entry", func(t *testing.T) {
		got, err := mergeSecrets("s", prev, "sk-old", "", SecretClear, false)
		if err != nil {
			t.Fatalf("mergeSecrets() error = %v", err)
		}
		if _, ok := got[openRouterAPIKeySecret]; ok {
			t.Fatalf("secrets = %v, want entry deleted", got)
		}
	})

	t.Run("explicit clear deletes undecryptable blob", func(t *testing.T) {
		got, err := mergeSecrets("s", prev, "", "", SecretClear, false)
		if err != nil {
			t.Fatalf("mergeSecrets() error = %v", err)
		}
		if _, ok := got[openRouterAPIKeySecret]; ok {
			t.Fatalf("secrets = %v, want undecryptable entry explicitly deleted", got)
		}
	})

	t.Run("new key replaces entry", func(t *testing.T) {
		const encryptionKey = "settings-test-encryption-key-123456"
		got, err := mergeSecrets(encryptionKey, prev, "sk-old", "sk-new", SecretReplace, false)
		if err != nil {
			t.Fatalf("mergeSecrets() error = %v", err)
		}
		decrypted, err := decryptCredential(encryptionKey, nil, nil, got[openRouterAPIKeySecret], openRouterAPIKeyContext)
		if err != nil || decrypted.Plaintext != "sk-new" {
			t.Fatalf("decrypt stored blob = %+v, %v; want sk-new", decrypted, err)
		}
		if prev[openRouterAPIKeySecret] != "stored-blob" {
			t.Fatal("mergeSecrets() must not mutate prev")
		}
	})

	t.Run("missing dedicated encryption key refused", func(t *testing.T) {
		_, err := mergeSecrets("", nil, "", "sk-new", SecretReplace, false)
		if !errors.Is(err, ErrConfigEncryptionKey) {
			t.Fatalf("mergeSecrets() error = %v, want ErrConfigEncryptionKey", err)
		}
	})

	t.Run("short dedicated encryption key refused", func(t *testing.T) {
		_, err := mergeSecrets("too-short", nil, "", "sk-new", SecretReplace, false)
		if !errors.Is(err, ErrConfigEncryptionKey) {
			t.Fatalf("mergeSecrets() error = %v, want ErrConfigEncryptionKey", err)
		}
	})

	t.Run("legacy ciphertext is re-encrypted even when plaintext is unchanged", func(t *testing.T) {
		const encryptionKey = "settings-test-encryption-key-123456"
		got, err := mergeSecrets(encryptionKey, prev, "sk-old", "sk-old", SecretPreserve, true)
		if err != nil {
			t.Fatalf("mergeSecrets() error = %v", err)
		}
		if got[openRouterAPIKeySecret] == "stored-blob" {
			t.Fatal("forced re-encryption must replace the stored blob")
		}
		decrypted, err := decryptCredential(encryptionKey, nil, nil, got[openRouterAPIKeySecret], openRouterAPIKeyContext)
		if err != nil || decrypted.Plaintext != "sk-old" || decrypted.NeedsRewrite {
			t.Fatalf("decrypt re-encrypted blob = %+v, %v; want current sk-old", decrypted, err)
		}
	})
}

func TestDecodeSettingsRowUsesLegacyDecryptionKey(t *testing.T) {
	const legacyKey = "legacy-auth-secret"
	blob, err := encryptString(legacyKey, "sk-or-legacy")
	if err != nil {
		t.Fatalf("encryptString() error = %v", err)
	}

	st, _, err := decodeSettingsRow(
		"new-settings-encryption-key-1234",
		nil,
		[]string{legacyKey},
		[]byte(`{}`),
		[]byte(`{}`),
		[]byte(`{"openrouter_api_key":"`+blob+`"}`),
	)
	if err != nil {
		t.Fatalf("decodeSettingsRow() error = %v", err)
	}
	if st.AI.OpenRouterAPIKey != "sk-or-legacy" {
		t.Fatalf("OpenRouterAPIKey = %q, want legacy key decrypted", st.AI.OpenRouterAPIKey)
	}
}

func TestDecodeSettingsRowNeverUsesAuthKeyForVersionedEnvelope(t *testing.T) {
	const (
		active  = "new-settings-encryption-key-1234"
		authKey = "legacy-auth-secret-that-is-long-enough"
	)
	blob, err := encryptCredential(authKey, "sk-or-must-not-decrypt", openRouterAPIKeyContext)
	if err != nil {
		t.Fatalf("encryptCredential() error = %v", err)
	}

	st, secrets, err := decodeSettingsRow(
		active,
		nil,
		[]string{authKey},
		[]byte(`{}`),
		[]byte(`{}`),
		[]byte(`{"openrouter_api_key":"`+blob+`"}`),
	)
	if err != nil {
		t.Fatalf("decodeSettingsRow() error = %v", err)
	}
	if st.AI.OpenRouterAPIKey != "" {
		t.Fatal("versioned envelope encrypted with the auth key must not decrypt")
	}
	if secrets[openRouterAPIKeySecret] != blob {
		t.Fatal("rejected versioned envelope must remain byte-for-byte")
	}
}

func TestDecodeSettingsRowLogsOnlySafeCredentialFailure(t *testing.T) {
	const (
		active = "active-secret-that-must-never-be-logged"
		blob   = "pai:v1:a256gcm:AAAAAAAAAAAAAAAAAAAAAA:bWFya2VyLWNpcGhlcnRleHQ"
	)
	var logs bytes.Buffer
	previous := slog.Default()
	slog.SetDefault(slog.New(slog.NewTextHandler(&logs, nil)))
	t.Cleanup(func() { slog.SetDefault(previous) })

	if _, _, err := decodeSettingsRow(
		active,
		nil,
		nil,
		[]byte(`{}`),
		[]byte(`{}`),
		[]byte(`{"openrouter_api_key":"`+blob+`"}`),
	); err != nil {
		t.Fatalf("decodeSettingsRow() error = %v", err)
	}
	got := logs.String()
	for _, forbidden := range []string{active, blob, "marker-ciphertext", "bWFya2VyLWNpcGhlcnRleHQ"} {
		if strings.Contains(got, forbidden) {
			t.Fatalf("credential failure log leaked %q: %s", forbidden, got)
		}
	}
	if !strings.Contains(got, "unknown credential encryption key") {
		t.Fatalf("credential failure log = %q, want safe error classification", got)
	}
}

func TestStoreSeparatesEncryptionAndLegacyAuthKeys(t *testing.T) {
	const sharedKey = "shared-auth-and-encryption-key-123"
	store := New(nil, sharedKey, sharedKey, config.AIConfig{}, featureflags.Features{})
	if store.writeEncryptionKey() != "" {
		t.Fatal("shared auth and encryption key must be refused for writes")
	}

	legacyDefault := New(nil, "", config.DefaultAuthSecret, config.AIConfig{}, featureflags.Features{})
	if keys := legacyDefault.decryptionKeys(); len(keys) != 0 {
		t.Fatalf("default auth secret must not be accepted as a legacy decryption key: %v", keys)
	}
}

func TestStoreDetectsLegacyCiphertextForReencryption(t *testing.T) {
	const (
		encryptionKey = "dedicated-settings-encryption-key-123"
		legacyKey     = "legacy-auth-key"
		plaintext     = "sk-or-legacy"
	)
	legacyBlob, err := encryptString(legacyKey, plaintext)
	if err != nil {
		t.Fatalf("encrypt legacy blob: %v", err)
	}
	store := New(nil, encryptionKey, legacyKey, config.AIConfig{}, featureflags.Features{})
	if !store.secretNeedsReencryption(legacyBlob, plaintext) {
		t.Fatal("legacy ciphertext should require re-encryption")
	}

	currentBlob, err := encryptCredential(encryptionKey, plaintext, openRouterAPIKeyContext)
	if err != nil {
		t.Fatalf("encrypt current blob: %v", err)
	}
	if store.secretNeedsReencryption(currentBlob, plaintext) {
		t.Fatal("current ciphertext should not require re-encryption")
	}
}

func TestAISettingsAPIKeyNeverSerializes(t *testing.T) {
	raw, err := json.Marshal(AISettings{
		DefaultProvider:  "openrouter",
		OpenRouterAPIKey: "sk-or-super-secret",
	})
	if err != nil {
		t.Fatalf("Marshal() error = %v", err)
	}
	if strings.Contains(string(raw), "secret") {
		t.Fatalf("AISettings JSON leaked the API key: %s", raw)
	}
}

func TestStoreCanonicalizesOverridesEqualToEnvironment(t *testing.T) {
	envFlags, err := featureflags.Parse("turn_hooks")
	if err != nil {
		t.Fatalf("featureflags.Parse() error = %v", err)
	}
	store := New(nil, "", "", config.AIConfig{
		DefaultProvider: "openrouter",
		OpenRouter: config.OpenRouterConfig{
			APIKey: "sk-or-environment",
			Model:  "environment/model",
		},
	}, envFlags)
	st := Settings{
		AI: AISettings{
			DefaultProvider:           "openrouter",
			OpenRouterModel:           "environment/model",
			OpenRouterAPIKey:          "sk-or-environment",
			OpenRouterAPIKeyOperation: SecretReplace,
		},
		Flags: map[string]bool{
			"turn_hooks": false,
			"ai_health":  true,
			"agent_core": false,
		},
	}

	store.canonicalizeRedundantOverrides(&st)

	if st.AI.DefaultProvider != "" || st.AI.OpenRouterModel != "" ||
		st.AI.OpenRouterAPIKey != "" || st.AI.OpenRouterAPIKeyOperation != SecretClear {
		t.Fatalf("AI overrides = %+v, want redundant values cleared", st.AI)
	}
	if got, ok := st.Flags["turn_hooks"]; !ok || got {
		t.Fatalf("turn_hooks override = %v, %v; want non-redundant false preserved", got, ok)
	}
	if _, ok := st.Flags["ai_health"]; !ok {
		t.Fatal("ai_health override must remain because the env baseline is false")
	}
	if _, ok := st.Flags["agent_core"]; ok {
		t.Fatal("agent_core override must be removed because it equals the env baseline")
	}
}

func TestStoreReportsDriftUntilRuntimeMarksRevisionApplied(t *testing.T) {
	store := New(nil, "", "", config.AIConfig{}, featureflags.Features{})
	store.setCurrent(Settings{Revision: 7})

	before := store.Effective()
	if !before.Drift || before.Revision != 7 || before.AppliedRevision != 0 {
		t.Fatalf("before apply = desired %d applied %d drift %v, want unapplied revision",
			before.Revision, before.AppliedRevision, before.Drift)
	}

	store.MarkApplied(7)
	after := store.Effective()
	if after.Drift || after.AppliedRevision != 7 {
		t.Fatalf("after apply = desired %d applied %d drift %v, want synchronized",
			after.Revision, after.AppliedRevision, after.Drift)
	}
}
