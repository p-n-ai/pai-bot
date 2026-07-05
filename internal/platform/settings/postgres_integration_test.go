// Copyright 2026 the P&AI authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

//go:build integration
// +build integration

package settings

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/p-n-ai/pai-bot/internal/platform/config"
	"github.com/p-n-ai/pai-bot/internal/platform/featureflags"
)

func TestStore_SaveLoadRoundtrip(t *testing.T) {
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

	store := New(pool, "test-auth-secret", config.AIConfig{}, featureflags.Features{})

	empty, err := store.Load(ctx)
	if err != nil {
		t.Fatalf("Load(missing row) error = %v", err)
	}
	if empty.AI != (AISettings{}) || len(empty.Flags) != 0 {
		t.Fatalf("Load(missing row) = %+v, want zero Settings", empty)
	}

	want := Settings{
		AI: AISettings{
			DefaultProvider:  "openrouter",
			OpenRouterModel:  "openrouter/auto",
			OpenRouterAPIKey: "sk-or-v1-roundtrip",
		},
		Flags: map[string]bool{"turn_hooks": true},
	}
	if _, err := store.Update(ctx, func(Settings) (Settings, error) { return want, nil }, nil); err != nil {
		t.Fatalf("Update() error = %v", err)
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
	if got.AI != want.AI {
		t.Fatalf("Load().AI = %+v, want %+v", got.AI, want.AI)
	}
	if !got.Flags["turn_hooks"] {
		t.Fatalf("Load().Flags = %v, want turn_hooks=true", got.Flags)
	}

	staleStore := New(pool, "test-auth-secret", config.AIConfig{}, featureflags.Features{})
	merged, err := staleStore.Update(ctx, func(cur Settings) (Settings, error) {
		cur.AI.OpenRouterModel = "deepseek/deepseek-chat"
		return cur, nil
	}, nil)
	if err != nil {
		t.Fatalf("Update(stale instance) error = %v", err)
	}
	if merged.AI.OpenRouterAPIKey != want.AI.OpenRouterAPIKey || merged.AI.DefaultProvider != want.AI.DefaultProvider {
		t.Fatalf("Update(stale instance) = %+v, want key and provider merged from DB row", merged.AI)
	}

	if _, err := store.Update(ctx, func(cur Settings) (Settings, error) {
		cur.AI.OpenRouterAPIKey = ""
		return cur, nil
	}, nil); err != nil {
		t.Fatalf("Update(cleared key) error = %v", err)
	}
	got, err = store.Load(ctx)
	if err != nil {
		t.Fatalf("Load(cleared key) error = %v", err)
	}
	if got.AI.OpenRouterAPIKey != "" {
		t.Fatal("cleared API key should not survive a save/load roundtrip")
	}
	if got.AI.OpenRouterModel != "deepseek/deepseek-chat" {
		t.Fatalf("Load().AI.OpenRouterModel = %q, want stale instance's write preserved", got.AI.OpenRouterModel)
	}
}

func applyRuntimeSettingsMigration(t *testing.T, ctx context.Context, pool *pgxpool.Pool) {
	t.Helper()

	path := filepath.Join("..", "..", "..", "migrations", "20260705090000_runtime_settings.sql")
	sqlBytes, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read migration %s: %v", path, err)
	}
	content := string(sqlBytes)
	up := content
	if i := strings.Index(content, "-- +goose Up"); i >= 0 {
		up = content[i+len("-- +goose Up"):]
	}
	if i := strings.Index(up, "-- +goose Down"); i >= 0 {
		up = up[:i]
	}
	if _, err := pool.Exec(ctx, `DROP TABLE IF EXISTS runtime_settings`); err != nil {
		t.Fatalf("drop runtime_settings: %v", err)
	}
	if _, err := pool.Exec(ctx, up); err != nil {
		t.Fatalf("apply migration %s: %v", path, err)
	}
}
