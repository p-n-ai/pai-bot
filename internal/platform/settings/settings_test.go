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

func stringPointer(value string) *string { return &value }
func boolPointer(value bool) *bool       { return &value }

func envAIConfig() config.AIConfig {
	return config.AIConfig{
		DefaultProvider: "openai",
		OpenAI:          config.OpenAIConfig{APIKey: "env-openai-key", Model: "env-openai-model"},
		OpenRouter:      config.OpenRouterConfig{APIKey: "env-key", Model: "env-model"},
		Ollama:          config.OllamaConfig{URL: "http://localhost:11434"},
		Codex:           config.CodexConfig{Enabled: true},
	}
}

func TestReconcileAIClosedProviderOverrides(t *testing.T) {
	st := AISettings{
		DefaultProvider: stringPointer("ollama"),
		Providers: ProviderOverrides{
			APIKey: map[APIKeyProvider]APIKeyProviderOverride{
				APIKeyProviderOpenRouter: {Model: stringPointer("db-model")},
			},
			Ollama:       &EnabledModelOverride{Enabled: boolPointer(true), Model: stringPointer("local-model")},
			ManagedCodex: &ModelOverride{Model: stringPointer("codex-model")},
		},
		Credentials: map[APIKeyProvider]CredentialOverride{
			APIKeyProviderOpenRouter: {Value: "db-secret-1234"},
		},
	}

	got := ReconcileAI(envAIConfig(), st)
	if got.Config.DefaultProvider != "ollama" || got.DefaultProviderSource != SourceDB {
		t.Fatalf("default provider = %q (%s), want ollama (db)", got.Config.DefaultProvider, got.DefaultProviderSource)
	}
	if got.Config.OpenRouter.Model != "db-model" || got.Config.OpenRouter.APIKey != "db-secret-1234" {
		t.Fatalf("OpenRouter = %+v, want DB model and credential", got.Config.OpenRouter)
	}
	if !got.Config.Ollama.Enabled || got.Config.Ollama.Model != "local-model" {
		t.Fatalf("Ollama = %+v, want enabled local-model", got.Config.Ollama)
	}
	if !got.Config.Codex.Enabled || got.Config.Codex.Model != "codex-model" {
		t.Fatalf("Codex = %+v, want managed Codex override", got.Config.Codex)
	}
	openRouter := got.Effective.Providers["openrouter"]
	if !openRouter.Credential.Set || openRouter.Credential.Last4 != "1234" {
		t.Fatalf("OpenRouter credential projection = %+v, want redacted DB credential", openRouter.Credential)
	}
	if got.ProviderSources["openrouter"].Credential != SourceDB ||
		got.ProviderSources["ollama"].Enabled != SourceDB {
		t.Fatalf("provider sources = %+v, want DB ownership", got.ProviderSources)
	}
}

func TestReconcileAIResetFallsBackToEnvironment(t *testing.T) {
	got := ReconcileAI(envAIConfig(), AISettings{})
	if got.Config.DefaultProvider != "openai" || got.DefaultProviderSource != SourceEnv {
		t.Fatalf("default provider = %q (%s), want openai (env)", got.Config.DefaultProvider, got.DefaultProviderSource)
	}
	if got.Config.OpenRouter.Model != "env-model" || got.Config.OpenRouter.APIKey != "env-key" {
		t.Fatalf("OpenRouter = %+v, want environment baseline", got.Config.OpenRouter)
	}
	if len(got.Override.Providers) != 0 || got.Override.DefaultProvider != nil {
		t.Fatalf("override = %+v, want empty sparse override", got.Override)
	}

	reset := ReconcileAI(envAIConfig(), AISettings{
		Credentials: map[APIKeyProvider]CredentialOverride{
			APIKeyProviderOpenRouter: {
				Operation: SecretClear,
				Envelope:  CredentialEnvelopeStatus{Stored: true},
			},
		},
	})
	if reset.Config.OpenRouter.APIKey != "env-key" ||
		reset.ProviderSources["openrouter"].Credential != SourceEnv {
		t.Fatalf("explicit clear = %+v, want environment credential", reset.Config.OpenRouter)
	}
}

func TestReconcileAIUnreadableStoredCredentialDoesNotFallBackToEnvironment(t *testing.T) {
	st := AISettings{
		Credentials: map[APIKeyProvider]CredentialOverride{
			APIKeyProviderOpenRouter: {
				Envelope: CredentialEnvelopeStatus{Stored: true},
			},
		},
	}

	got := ReconcileAI(envAIConfig(), st)
	if got.Config.OpenRouter.APIKey != "" {
		t.Fatal("unreadable stored credential fell back to the environment")
	}
	if got.ProviderSources["openrouter"].Credential != SourceDB {
		t.Fatalf("credential source = %q, want db", got.ProviderSources["openrouter"].Credential)
	}
	if got.Effective.Providers["openrouter"].Credential.Set {
		t.Fatal("unreadable stored credential reported as effective")
	}
	if !got.Override.Providers["openrouter"].Credential.Set {
		t.Fatal("unreadable stored credential did not retain database ownership")
	}
}

func TestEffectiveFlagsUseSameProjection(t *testing.T) {
	envFlags, err := featureflags.Parse("turn_hooks=true")
	if err != nil {
		t.Fatal(err)
	}
	eff := Effective(envAIConfig(), envFlags, Settings{
		Flags: map[string]bool{"turn_hooks": false},
	})
	if eff.Flags["turn_hooks"] || eff.FlagSources["turn_hooks"] != SourceDB {
		t.Fatalf("turn_hooks = %v (%s), want false (db)", eff.Flags["turn_hooks"], eff.FlagSources["turn_hooks"])
	}
	if !eff.Baseline.Flags["turn_hooks"] || eff.Override.Flags["turn_hooks"] ||
		eff.Effective.Flags["turn_hooks"] {
		t.Fatalf("flag projections = baseline:%v override:%v effective:%v", eff.Baseline.Flags, eff.Override.Flags, eff.Effective.Flags)
	}
}

func TestKeyLast4(t *testing.T) {
	for _, test := range []struct{ key, want string }{
		{"", ""}, {"abc", ""}, {"sk-1234", ""}, {"sk-or-v1-abcd1234", "1234"},
	} {
		if got := KeyLast4(test.key); got != test.want {
			t.Fatalf("KeyLast4(%q) = %q, want %q", test.key, got, test.want)
		}
	}
}

func TestDecodeSettingsRowMigratesLegacyOpenRouterShape(t *testing.T) {
	const legacyKey = "legacy-auth-secret"
	blob, err := encryptString(legacyKey, "sk-or-legacy")
	if err != nil {
		t.Fatal(err)
	}
	st, secrets, err := decodeSettingsRow(
		"new-settings-encryption-key-1234", nil, []string{legacyKey},
		[]byte(`{"default_provider":"openrouter","openrouter_model":"legacy-model"}`),
		[]byte(`{"turn_hooks":true}`),
		[]byte(`{"openrouter_api_key":"`+blob+`","unknown":"preserve-me"}`),
	)
	if err != nil {
		t.Fatalf("decodeSettingsRow() error = %v", err)
	}
	if st.AI.DefaultProvider == nil || *st.AI.DefaultProvider != "openrouter" {
		t.Fatalf("DefaultProvider = %#v, want openrouter", st.AI.DefaultProvider)
	}
	override := st.AI.Providers.APIKey[APIKeyProviderOpenRouter]
	if override.Model == nil || *override.Model != "legacy-model" {
		t.Fatalf("OpenRouter override = %+v, want legacy-model", override)
	}
	if got := st.AI.Credentials[APIKeyProviderOpenRouter].Value; got != "sk-or-legacy" {
		t.Fatalf("OpenRouter credential = %q, want legacy plaintext in private state", got)
	}
	if secrets["unknown"] != "preserve-me" {
		t.Fatal("unknown secret entry was not preserved")
	}
}

func TestDecodeSettingsRowRejectsDuplicateLegacyAndCanonicalState(t *testing.T) {
	for _, test := range []struct {
		name, ai, secrets string
	}{
		{
			name:    "model",
			ai:      `{"openrouter_model":"old","providers":{"api_key":{"openrouter":{"model":"new"}}}}`,
			secrets: `{}`,
		},
		{
			name:    "credential",
			ai:      `{}`,
			secrets: `{"openrouter_api_key":"old","provider/openrouter/api_key":"new"}`,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			if _, _, err := decodeSettingsRow("secret", nil, nil, []byte(test.ai), []byte(`{}`), []byte(test.secrets)); err == nil {
				t.Fatal("decodeSettingsRow() should reject duplicate legacy and canonical state")
			}
		})
	}
}

func TestDecodeSettingsRowPreservesUndecryptableCredential(t *testing.T) {
	blob, err := encryptString("old-auth-secret", "sk-or-v1-oldkey")
	if err != nil {
		t.Fatal(err)
	}
	st, secrets, err := decodeSettingsRow(
		"rotated-auth-secret", nil, nil, []byte(`{}`), []byte(`{}`),
		[]byte(`{"openrouter_api_key":"`+blob+`"}`),
	)
	if err != nil {
		t.Fatal(err)
	}
	if st.AI.Credentials[APIKeyProviderOpenRouter].Value != "" {
		t.Fatal("undecryptable credential must not enter private plaintext state")
	}
	if secrets[legacyOpenRouterAPIKeySecret] != blob {
		t.Fatal("undecryptable ciphertext must remain byte-for-byte")
	}
}

func TestDecodeSettingsRowCorruptedJSON(t *testing.T) {
	good, bad := []byte(`{}`), []byte(`{corrupt`)
	for _, columns := range [][3][]byte{{bad, good, good}, {good, bad, good}, {good, good, bad}} {
		if _, _, err := decodeSettingsRow("secret", nil, nil, columns[0], columns[1], columns[2]); err == nil {
			t.Fatal("decodeSettingsRow() should reject corrupted jsonb")
		}
	}
}

func TestMergeSecretsGenericSlotsAndUnknownPreservation(t *testing.T) {
	const encryptionKey = "settings-test-encryption-key-123456"
	prev := map[string]string{
		legacyOpenRouterAPIKeySecret: "stored-legacy-blob",
		"unknown":                    "preserve-byte-for-byte",
	}
	credentials := map[APIKeyProvider]CredentialOverride{
		APIKeyProviderOpenRouter: {Value: "sk-new", Operation: SecretReplace},
		APIKeyProviderGoogle:     {Value: "google-new", Operation: SecretReplace},
	}
	got, err := mergeSecrets(
		encryptionKey, prev,
		map[APIKeyProvider]string{APIKeyProviderOpenRouter: "sk-old"},
		credentials, map[APIKeyProvider]bool{},
	)
	if err != nil {
		t.Fatalf("mergeSecrets() error = %v", err)
	}
	if got["unknown"] != "preserve-byte-for-byte" || prev[legacyOpenRouterAPIKeySecret] != "stored-legacy-blob" {
		t.Fatal("mergeSecrets mutated unknown or input entries")
	}
	if _, exists := got[legacyOpenRouterAPIKeySecret]; exists {
		t.Fatal("successful rewrite must remove the known legacy slot")
	}
	for provider, want := range map[APIKeyProvider]string{
		APIKeyProviderOpenRouter: "sk-new",
		APIKeyProviderGoogle:     "google-new",
	} {
		decrypted, decryptErr := decryptCredential(
			encryptionKey, nil, nil, got[credentialSecretName(provider)], apiKeyCredentialContext(provider),
		)
		if decryptErr != nil || decrypted.Plaintext != want {
			t.Fatalf("decrypt %s = %+v, %v; want %q", provider, decrypted, decryptErr, want)
		}
	}
}

func TestMergeSecretsExplicitClearDeletesKnownSlotsOnly(t *testing.T) {
	prev := map[string]string{
		legacyOpenRouterAPIKeySecret:                   "old",
		credentialSecretName(APIKeyProviderOpenRouter): "new",
		"unknown": "keep",
	}
	got, err := mergeSecrets("", prev, nil, map[APIKeyProvider]CredentialOverride{
		APIKeyProviderOpenRouter: {Operation: SecretClear},
	}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := got[legacyOpenRouterAPIKeySecret]; ok {
		t.Fatal("legacy slot survived explicit clear")
	}
	if _, ok := got[credentialSecretName(APIKeyProviderOpenRouter)]; ok {
		t.Fatal("canonical slot survived explicit clear")
	}
	if got["unknown"] != "keep" {
		t.Fatal("explicit clear removed unknown secret")
	}
}

func TestMergeSecretsRequiresDedicatedEncryptionKey(t *testing.T) {
	_, err := mergeSecrets("", nil, nil, map[APIKeyProvider]CredentialOverride{
		APIKeyProviderOpenRouter: {Value: "new", Operation: SecretReplace},
	}, nil)
	if !errors.Is(err, ErrConfigEncryptionKey) {
		t.Fatalf("mergeSecrets() error = %v, want ErrConfigEncryptionKey", err)
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

	_, _, err := decodeSettingsRow(active, nil, nil, []byte(`{}`), []byte(`{}`),
		[]byte(`{"provider/openrouter/api_key":"`+blob+`"}`))
	if err != nil {
		t.Fatal(err)
	}
	got := logs.String()
	for _, forbidden := range []string{active, blob, "marker-ciphertext", "bWFya2VyLWNpcGhlcnRleHQ"} {
		if strings.Contains(got, forbidden) {
			t.Fatalf("credential failure log leaked %q: %s", forbidden, got)
		}
	}
}

func TestStoreDetectsProviderCiphertextForReencryption(t *testing.T) {
	const (
		encryptionKey = "dedicated-settings-encryption-key-123"
		legacyKey     = "legacy-auth-key"
		plaintext     = "sk-or-legacy"
	)
	legacyBlob, err := encryptString(legacyKey, plaintext)
	if err != nil {
		t.Fatal(err)
	}
	store := New(nil, encryptionKey, legacyKey, config.AIConfig{}, featureflags.Features{})
	if !store.secretNeedsReencryption(APIKeyProviderOpenRouter, legacyBlob, plaintext) {
		t.Fatal("legacy ciphertext should require re-encryption")
	}
	currentBlob, err := encryptCredential(encryptionKey, plaintext, apiKeyCredentialContext(APIKeyProviderOpenRouter))
	if err != nil {
		t.Fatal(err)
	}
	if store.secretNeedsReencryption(APIKeyProviderOpenRouter, currentBlob, plaintext) {
		t.Fatal("current ciphertext should not require re-encryption")
	}
}

func TestAISettingsCredentialsNeverSerialize(t *testing.T) {
	raw, err := json.Marshal(AISettings{
		DefaultProvider: stringPointer("openrouter"),
		Credentials: map[APIKeyProvider]CredentialOverride{
			APIKeyProviderOpenRouter: {Value: "sk-or-super-secret"},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(raw), "secret") {
		t.Fatalf("AISettings JSON leaked the API key: %s", raw)
	}
}

func TestCloneSettingsForMutationDoesNotAliasLockedSnapshot(t *testing.T) {
	source := Settings{
		AI: AISettings{
			DefaultProvider: stringPointer("openrouter"),
			Providers: ProviderOverrides{
				APIKey: map[APIKeyProvider]APIKeyProviderOverride{
					APIKeyProviderOpenRouter: {Model: stringPointer("original-api-model")},
				},
				Ollama:       &EnabledModelOverride{Enabled: boolPointer(true), Model: stringPointer("original-ollama-model")},
				ManagedCodex: &ModelOverride{Model: stringPointer("original-codex-model")},
			},
			Credentials: map[APIKeyProvider]CredentialOverride{
				APIKeyProviderOpenRouter: {Value: "original-key"},
			},
		},
		Flags:    map[string]bool{"turn_hooks": true},
		Revision: 7,
	}

	mutable := cloneSettingsForMutation(source)
	*mutable.AI.DefaultProvider = "ollama"
	apiOverride := mutable.AI.Providers.APIKey[APIKeyProviderOpenRouter]
	*apiOverride.Model = "changed-api-model"
	mutable.AI.Providers.APIKey[APIKeyProviderOpenRouter] = apiOverride
	*mutable.AI.Providers.Ollama.Enabled = false
	*mutable.AI.Providers.Ollama.Model = "changed-ollama-model"
	*mutable.AI.Providers.ManagedCodex.Model = "changed-codex-model"
	mutable.AI.Credentials[APIKeyProviderOpenRouter] = CredentialOverride{Value: "changed-key"}
	mutable.Flags["turn_hooks"] = false

	if *source.AI.DefaultProvider != "openrouter" ||
		*source.AI.Providers.APIKey[APIKeyProviderOpenRouter].Model != "original-api-model" ||
		!*source.AI.Providers.Ollama.Enabled ||
		*source.AI.Providers.Ollama.Model != "original-ollama-model" ||
		*source.AI.Providers.ManagedCodex.Model != "original-codex-model" ||
		source.AI.Credentials[APIKeyProviderOpenRouter].Value != "original-key" ||
		!source.Flags["turn_hooks"] {
		t.Fatalf("mutation clone changed locked snapshot: %+v", source)
	}
}

func TestStoreCanonicalizesOverridesEqualToEnvironment(t *testing.T) {
	envFlags, err := featureflags.Parse("turn_hooks")
	if err != nil {
		t.Fatal(err)
	}
	env := envAIConfig()
	store := New(nil, "", "", env, envFlags)
	st := Settings{
		AI: AISettings{
			DefaultProvider: stringPointer(env.DefaultProvider),
			Providers: ProviderOverrides{
				APIKey: map[APIKeyProvider]APIKeyProviderOverride{
					APIKeyProviderOpenRouter: {Model: stringPointer(env.OpenRouter.Model)},
				},
			},
			Credentials: map[APIKeyProvider]CredentialOverride{
				APIKeyProviderOpenRouter: {Value: env.OpenRouter.APIKey, Operation: SecretReplace},
			},
		},
		Flags: map[string]bool{"turn_hooks": false, "agent_core": false},
	}
	store.canonicalizeRedundantOverrides(&st)
	if st.AI.DefaultProvider != nil || len(st.AI.Providers.APIKey) != 0 {
		t.Fatalf("AI scalar overrides = %+v, want canonicalized away", st.AI)
	}
	credential := st.AI.Credentials[APIKeyProviderOpenRouter]
	if credential.Value != "" || credential.Operation != SecretClear {
		t.Fatalf("credential = %+v, want explicit stored override clear", credential)
	}
	if _, ok := st.Flags["agent_core"]; ok {
		t.Fatal("flag equal to baseline was not canonicalized")
	}
}

func TestStoreReportsDriftUntilRuntimeMarksRevisionApplied(t *testing.T) {
	store := New(nil, "", "", config.AIConfig{}, featureflags.Features{})
	store.setCurrent(Settings{Revision: 7})
	if before := store.Effective(); !before.Drift || before.Revision != 7 {
		t.Fatalf("before apply = %+v, want drift", before)
	}
	store.MarkApplied(7)
	if after := store.Effective(); after.Drift || after.AppliedRevision != 7 {
		t.Fatalf("after apply = %+v, want synchronized", after)
	}
}
