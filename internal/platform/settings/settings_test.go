// Copyright 2026 the P&AI authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package settings

import (
	"encoding/json"
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

func TestMergeFlags(t *testing.T) {
	base, err := featureflags.Parse("")
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}

	on, err := MergeFlags(base, map[string]bool{"turn_hooks": true})
	if err != nil {
		t.Fatalf("MergeFlags(on) error = %v", err)
	}
	if !on.Enabled(featureflags.TurnHooks) {
		t.Fatal("turn_hooks should be enabled by override")
	}
	if base.Enabled(featureflags.TurnHooks) {
		t.Fatal("MergeFlags must not mutate the base feature set")
	}

	off, err := MergeFlags(on, map[string]bool{"turn_hooks": false})
	if err != nil {
		t.Fatalf("MergeFlags(off) error = %v", err)
	}
	if off.Enabled(featureflags.TurnHooks) {
		t.Fatal("turn_hooks should be disabled by override")
	}

	if _, err := MergeFlags(base, map[string]bool{"unknown_flag": true}); err == nil {
		t.Fatal("MergeFlags should reject unknown flag names")
	}
}

func TestKeyLast4(t *testing.T) {
	tests := []struct {
		key  string
		want string
	}{
		{key: "", want: ""},
		{key: "abc", want: "abc"},
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

	st := decodeSettingsRow("rotated-auth-secret",
		[]byte(`{"default_provider":"openrouter","openrouter_model":"m"}`),
		[]byte(`{"turn_hooks":true}`),
		[]byte(`{"openrouter_api_key":"`+blob+`"}`))

	if st.AI.OpenRouterAPIKey != "" {
		t.Fatal("undecryptable key must be dropped, not returned")
	}
	if st.AI.DefaultProvider != "openrouter" || st.AI.OpenRouterModel != "m" || !st.Flags["turn_hooks"] {
		t.Fatalf("decodeSettingsRow() = %+v, want other settings kept", st)
	}
}

func TestDecodeSettingsRowDegradesCorruptedJSON(t *testing.T) {
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
			st := decodeSettingsRow("secret", tt.aiJSON, tt.flagsJSON, tt.secrets)
			if st.AI != (AISettings{}) || len(st.Flags) != 0 {
				t.Fatalf("decodeSettingsRow() = %+v, want zero Settings", st)
			}
		})
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
