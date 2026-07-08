// Copyright 2026 the P&AI authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package settings

import (
	"encoding/json"
	"errors"
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
	})

	t.Run("db overrides env", func(t *testing.T) {
		eff := Effective(envAIConfig(), envFlags, Settings{
			AI:    AISettings{DefaultProvider: "openrouter", OpenRouterAPIKey: "db-secret-1234"},
			Flags: map[string]bool{"turn_hooks": false},
		})
		if eff.DefaultProvider != "openrouter" || eff.DefaultProviderSource != SourceDB {
			t.Fatalf("DefaultProvider = %q (%s), want openrouter (db)", eff.DefaultProvider, eff.DefaultProviderSource)
		}
		if !eff.OpenRouterKeySet || eff.OpenRouterKeyLast4 != "1234" || eff.OpenRouterKeySource != SourceDB {
			t.Fatalf("OpenRouterKey = set:%v last4:%q (%s), want set with last4 1234 (db)", eff.OpenRouterKeySet, eff.OpenRouterKeyLast4, eff.OpenRouterKeySource)
		}
		if eff.OpenRouterModel != "env-model" || eff.OpenRouterModelSource != SourceEnv {
			t.Fatalf("OpenRouterModel = %q (%s), want env-model (env)", eff.OpenRouterModel, eff.OpenRouterModelSource)
		}
		if eff.Flags["turn_hooks"] || eff.FlagSources["turn_hooks"] != SourceDB {
			t.Fatalf("turn_hooks = %v (%s), want false (db)", eff.Flags["turn_hooks"], eff.FlagSources["turn_hooks"])
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

	st, secrets, err := decodeSettingsRow("rotated-auth-secret",
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
	st, _, err := decodeSettingsRow("secret",
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
			if _, _, err := decodeSettingsRow("secret", tt.aiJSON, tt.flagsJSON, tt.secrets); err == nil {
				t.Fatal("decodeSettingsRow() should reject corrupted jsonb")
			}
			st := degradeSettingsRow("secret", tt.aiJSON, tt.flagsJSON, tt.secrets)
			if st.AI != (AISettings{}) || len(st.Flags) != 0 {
				t.Fatalf("degradeSettingsRow() = %+v, want zero Settings", st)
			}
		})
	}
}

func TestMergeSecrets(t *testing.T) {
	prev := map[string]string{openRouterAPIKeySecret: "stored-blob"}

	t.Run("unchanged key preserves blob", func(t *testing.T) {
		got, err := mergeSecrets("s", prev, "sk-old", "sk-old")
		if err != nil {
			t.Fatalf("mergeSecrets() error = %v", err)
		}
		if got[openRouterAPIKeySecret] != "stored-blob" {
			t.Fatalf("secrets = %v, want stored blob preserved", got)
		}
	})

	t.Run("undecryptable blob survives unrelated update", func(t *testing.T) {
		// Decoded key is "" because the blob did not decrypt; mutate left it "".
		got, err := mergeSecrets("s", prev, "", "")
		if err != nil {
			t.Fatalf("mergeSecrets() error = %v", err)
		}
		if got[openRouterAPIKeySecret] != "stored-blob" {
			t.Fatalf("secrets = %v, want undecryptable blob preserved", got)
		}
	})

	t.Run("explicit clear deletes entry", func(t *testing.T) {
		got, err := mergeSecrets("s", prev, "sk-old", "")
		if err != nil {
			t.Fatalf("mergeSecrets() error = %v", err)
		}
		if _, ok := got[openRouterAPIKeySecret]; ok {
			t.Fatalf("secrets = %v, want entry deleted", got)
		}
	})

	t.Run("new key replaces entry", func(t *testing.T) {
		got, err := mergeSecrets("s", prev, "sk-old", "sk-new")
		if err != nil {
			t.Fatalf("mergeSecrets() error = %v", err)
		}
		key, err := decryptString("s", got[openRouterAPIKeySecret])
		if err != nil || key != "sk-new" {
			t.Fatalf("decrypt stored blob = %q, %v; want sk-new", key, err)
		}
		if prev[openRouterAPIKeySecret] != "stored-blob" {
			t.Fatal("mergeSecrets() must not mutate prev")
		}
	})

	t.Run("default auth secret refused", func(t *testing.T) {
		_, err := mergeSecrets(config.DefaultAuthSecret, nil, "", "sk-new")
		if !errors.Is(err, ErrDefaultAuthSecret) {
			t.Fatalf("mergeSecrets() error = %v, want ErrDefaultAuthSecret", err)
		}
	})
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
