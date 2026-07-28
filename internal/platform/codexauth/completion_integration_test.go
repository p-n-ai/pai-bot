// Copyright 2026 the P&AI authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

//go:build integration

package codexauth

import (
	"context"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"

	"github.com/p-n-ai/pai-bot/internal/ai"
)

func TestCodexAppServerLiveCompletion(t *testing.T) {
	home := strings.TrimSpace(os.Getenv("LEARN_AI_CODEX_HOME"))
	if home == "" {
		t.Skip("LEARN_AI_CODEX_HOME is not configured")
	}
	executable, err := exec.LookPath("codex")
	if err != nil {
		t.Skip("Codex CLI is not available")
	}

	ctx, cancel := context.WithTimeout(t.Context(), 2*time.Minute)
	defer cancel()
	manager := New(ctx, home, executable, nil)

	response, err := manager.Complete(ctx, ai.CompletionRequest{
		Model: strings.TrimSpace(os.Getenv("LEARN_AI_CODEX_MODEL")),
		Messages: []ai.Message{{
			Role:    "user",
			Content: "Reply with exactly: codex app-server connected",
		}},
	})
	if err != nil {
		t.Fatalf("Complete() error = %v", err)
	}
	if strings.TrimSpace(response.Content) != "codex app-server connected" {
		t.Fatalf("Complete() content = %q", response.Content)
	}
	if strings.TrimSpace(response.Model) == "" {
		t.Fatal("Complete() returned no model")
	}
}
