// Copyright 2026 the P&AI authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/p-n-ai/pai-bot/internal/ai"
	"github.com/p-n-ai/pai-bot/internal/platform/codexauth"
	"github.com/p-n-ai/pai-bot/internal/platform/config"
)

type fakeManagedCodexLogin struct {
	mu              sync.Mutex
	available       bool
	refreshErr      error
	startErr        error
	statuses        []codexauth.Status
	statusIndex     int
	startCalls      int
	completionCalls int
	lastRequest     ai.CompletionRequest
}

func (f *fakeManagedCodexLogin) Available() bool {
	return f.available
}

func (f *fakeManagedCodexLogin) Refresh(context.Context) error {
	return f.refreshErr
}

func (f *fakeManagedCodexLogin) Complete(
	_ context.Context,
	request ai.CompletionRequest,
) (ai.CompletionResponse, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.completionCalls++
	f.lastRequest = request
	return ai.CompletionResponse{Content: "Codex reply", Model: request.Model}, nil
}

func (f *fakeManagedCodexLogin) Start() (codexauth.Status, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.startCalls++
	if f.startErr != nil {
		return f.currentStatus(), f.startErr
	}
	return f.currentStatus(), nil
}

func (f *fakeManagedCodexLogin) Status() codexauth.Status {
	f.mu.Lock()
	defer f.mu.Unlock()
	status := f.currentStatus()
	if f.statusIndex < len(f.statuses)-1 {
		f.statusIndex++
	}
	return status
}

func (f *fakeManagedCodexLogin) currentStatus() codexauth.Status {
	if len(f.statuses) == 0 {
		return codexauth.Status{State: codexauth.StateDisconnected}
	}
	return f.statuses[f.statusIndex]
}

func TestBuildTerminalAIRouterPinsCodexWithoutFallback(t *testing.T) {
	login := &fakeManagedCodexLogin{available: true}
	cfg := config.AIConfig{
		OpenAI: config.OpenAIConfig{APIKey: "configured-but-forbidden-fallback"},
		Codex: config.CodexConfig{
			Enabled: true,
			Home:    t.TempDir(),
			Model:   "gpt-test",
		},
	}
	var statusOutput strings.Builder

	router, err := buildTerminalAIRouterWithDependencies(
		t.Context(),
		cfg,
		"codex",
		&statusOutput,
		terminalProviderDependencies{
			findExecutable: func(string) (string, error) { return "/test/codex", nil },
			newCodexLogin: func(context.Context, string, string) managedCodexLogin {
				return login
			},
			pollInterval: time.Millisecond,
		},
	)
	if err != nil {
		t.Fatalf("buildTerminalAIRouterWithDependencies() error = %v", err)
	}
	if got := router.ProviderOrder(); !reflect.DeepEqual(got, []string{"codex"}) {
		t.Fatalf("ProviderOrder() = %v, want Codex only", got)
	}
	response, err := router.Complete(t.Context(), ai.CompletionRequest{
		Messages: []ai.Message{{Role: "user", Content: "hello"}},
	})
	if err != nil {
		t.Fatalf("Complete() error = %v", err)
	}
	if response.Content != "Codex reply" || login.completionCalls != 1 {
		t.Fatalf("Complete() = %#v, calls = %d", response, login.completionCalls)
	}
	if login.lastRequest.Model != "gpt-test" {
		t.Fatalf("Codex request model = %q, want gpt-test", login.lastRequest.Model)
	}
	if !strings.Contains(statusOutput.String(), "Codex provider ready.") {
		t.Fatalf("status output = %q, want ready confirmation", statusOutput.String())
	}
}

func TestEnsureManagedCodexLoginPrintsDeviceInstructionsAndWaits(t *testing.T) {
	login := &fakeManagedCodexLogin{
		available:  true,
		refreshErr: errors.New("not connected"),
		statuses: []codexauth.Status{
			{
				State:           codexauth.StateAwaiting,
				VerificationURL: "https://auth.openai.com/codex/device",
				UserCode:        "ABCD-EFGH",
			},
			{State: codexauth.StateConnected},
		},
	}
	var output strings.Builder

	if err := ensureManagedCodexLogin(t.Context(), login, &output, time.Millisecond); err != nil {
		t.Fatalf("ensureManagedCodexLogin() error = %v", err)
	}
	if login.startCalls != 1 {
		t.Fatalf("Start() calls = %d, want 1", login.startCalls)
	}
	rendered := output.String()
	for _, want := range []string{
		"https://auth.openai.com/codex/device",
		"ABCD-EFGH",
		"Waiting for authorization",
		"Codex connected.",
	} {
		if !strings.Contains(rendered, want) {
			t.Fatalf("output = %q, want %q", rendered, want)
		}
	}
}

func TestBuildTerminalAIRouterRejectsMissingCodexCLIWithoutFallback(t *testing.T) {
	cfg := config.AIConfig{
		OpenAI: config.OpenAIConfig{APIKey: "configured-but-forbidden-fallback"},
		Codex: config.CodexConfig{
			Enabled: true,
			Home:    t.TempDir(),
		},
	}

	_, err := buildTerminalAIRouterWithDependencies(
		t.Context(),
		cfg,
		"codex",
		&strings.Builder{},
		terminalProviderDependencies{
			findExecutable: func(string) (string, error) {
				return "", errors.New("not found")
			},
		},
	)
	if err == nil || !strings.Contains(err.Error(), "codex CLI") {
		t.Fatalf("error = %v, want actionable missing CLI error", err)
	}
}

func TestEnsureManagedCodexLoginReturnsSafeFailureMessage(t *testing.T) {
	login := &fakeManagedCodexLogin{
		available:  true,
		refreshErr: errors.New("not connected"),
		startErr:   errors.New("unsafe lower-level detail"),
		statuses: []codexauth.Status{{
			State:   codexauth.StateFailed,
			Message: "Codex CLI is not installed on the server.",
		}},
	}

	err := ensureManagedCodexLogin(t.Context(), login, &strings.Builder{}, time.Millisecond)
	if err == nil || err.Error() != "Codex CLI is not installed on the server." {
		t.Fatalf("error = %v, want safe status message", err)
	}
}

func TestEnsureManagedCodexLoginReturnsGenericFailureWithoutStatusMessage(t *testing.T) {
	login := &fakeManagedCodexLogin{
		available:  true,
		refreshErr: errors.New("not connected"),
		statuses: []codexauth.Status{{
			State: codexauth.StateFailed,
		}},
	}

	err := ensureManagedCodexLogin(t.Context(), login, &strings.Builder{}, time.Millisecond)
	if err == nil || err.Error() != "codex device login failed" {
		t.Fatalf("error = %v, want generic safe failure", err)
	}
}
