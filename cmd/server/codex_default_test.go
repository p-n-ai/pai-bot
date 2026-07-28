// Copyright 2026 the P&AI authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/p-n-ai/pai-bot/internal/platform/airouter"
	"github.com/p-n-ai/pai-bot/internal/platform/codexauth"
	"github.com/p-n-ai/pai-bot/internal/platform/config"
	"github.com/p-n-ai/pai-bot/internal/platform/settings"
)

type memoryRuntimeSettingsUpdater struct {
	current settings.Settings
	updates int
}

type availableCodexAuth struct{}

func (availableCodexAuth) Refresh(context.Context) error { return nil }
func (availableCodexAuth) Available() bool               { return true }

func (m *memoryRuntimeSettingsUpdater) Update(
	ctx context.Context,
	mutate func(settings.Settings) (settings.Settings, error),
	apply func(settings.Settings),
) (settings.Settings, error) {
	if err := ctx.Err(); err != nil {
		return settings.Settings{}, err
	}
	next, err := mutate(m.current)
	if err != nil {
		return settings.Settings{}, err
	}
	m.current = next
	m.updates++
	if apply != nil {
		apply(next)
	}
	return next, nil
}

func TestMakeCodexDefaultPreservesOtherSettingsAndApplies(t *testing.T) {
	store := &memoryRuntimeSettingsUpdater{current: settings.Settings{
		AI: settings.AISettings{
			DefaultProvider: "openrouter",
			OpenRouterModel: "existing-model",
		},
		Flags: map[string]bool{"turn_hooks": true},
	}}
	var applied settings.Settings

	if err := makeCodexDefault(t.Context(), store, func(next settings.Settings) {
		applied = next
	}); err != nil {
		t.Fatalf("makeCodexDefault() error = %v", err)
	}

	if store.updates != 1 || store.current.AI.DefaultProvider != "codex" {
		t.Fatalf("updates/settings = %d/%#v", store.updates, store.current)
	}
	if store.current.AI.OpenRouterModel != "existing-model" ||
		!store.current.Flags["turn_hooks"] {
		t.Fatalf("unrelated settings changed: %#v", store.current)
	}
	if applied.AI.DefaultProvider != "codex" {
		t.Fatalf("applied settings = %#v, want codex default", applied)
	}
}

func TestSuccessfulDeviceAuthRegistersCodexAsDefaultWithoutEnvToggle(t *testing.T) {
	home := t.TempDir()
	aiConfig := config.AIConfig{
		Codex: config.CodexConfig{
			Enabled:  true,
			Home:     home,
			AuthFile: filepath.Join(home, "auth.json"),
			Model:    "gpt-test",
		},
	}
	codexAuth := availableCodexAuth{}
	router := airouter.SetupWithCodexAuth(aiConfig, codexAuth)
	if router.HasProvider() {
		t.Fatal("Codex registered before device auth")
	}
	if err := os.WriteFile(aiConfig.Codex.AuthFile, []byte(`{"auth_mode":"chatgpt","tokens":{"access_token":"test-token","account_id":"test-account"}}`), 0o600); err != nil {
		t.Fatal(err)
	}
	store := &memoryRuntimeSettingsUpdater{}

	err := makeCodexDefault(t.Context(), store, func(next settings.Settings) {
		airouter.ApplyWithCodexAuth(
			router,
			settings.MergeAI(aiConfig, next),
			codexAuth,
		)
	})
	if err != nil {
		t.Fatalf("makeCodexDefault() error = %v", err)
	}

	order := router.ProviderOrder()
	if store.current.AI.DefaultProvider != "codex" ||
		len(order) != 1 ||
		order[0] != "codex" {
		t.Fatalf("default/order = %q/%v, want codex first", store.current.AI.DefaultProvider, order)
	}
}

func TestCanAwaitCodexDeviceAuthRequiresExecutable(t *testing.T) {
	home := t.TempDir()
	cfg := &config.Config{AI: config.AIConfig{Codex: config.CodexConfig{
		Enabled:  true,
		Home:     home,
		AuthFile: filepath.Join(home, "auth.json"),
	}}}

	unavailable := codexauth.New(t.Context(), cfg.AI.Codex.Home, "", nil)
	if canAwaitCodexDeviceAuth(cfg, unavailable) {
		t.Fatal("canAwaitCodexDeviceAuth() = true without a Codex executable")
	}

	available := codexauth.New(t.Context(), cfg.AI.Codex.Home, "/test/codex", nil)
	if !canAwaitCodexDeviceAuth(cfg, available) {
		t.Fatal("canAwaitCodexDeviceAuth() = false with configured storage and executable")
	}
}
