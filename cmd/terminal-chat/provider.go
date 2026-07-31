// Copyright 2026 the P&AI authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os/exec"
	"strings"
	"time"

	"github.com/p-n-ai/pai-bot/internal/ai"
	"github.com/p-n-ai/pai-bot/internal/platform/airouter"
	"github.com/p-n-ai/pai-bot/internal/platform/codexauth"
	"github.com/p-n-ai/pai-bot/internal/platform/config"
)

const managedCodexPollInterval = 100 * time.Millisecond

type managedCodexLogin interface {
	ai.CodexAppServerClient
	Start() (codexauth.Status, error)
	Status() codexauth.Status
}

type terminalProviderDependencies struct {
	findExecutable func(string) (string, error)
	newCodexLogin  func(context.Context, string, string) managedCodexLogin
	pollInterval   time.Duration
}

func defaultTerminalProviderDependencies() terminalProviderDependencies {
	return terminalProviderDependencies{
		findExecutable: exec.LookPath,
		newCodexLogin: func(ctx context.Context, home, executable string) managedCodexLogin {
			return codexauth.New(ctx, home, executable, nil)
		},
		pollInterval: managedCodexPollInterval,
	}
}

func buildTerminalAIRouter(
	ctx context.Context,
	cfg config.AIConfig,
	selectedProvider string,
	statusOutput io.Writer,
) (*ai.Router, error) {
	return buildTerminalAIRouterWithDependencies(
		ctx,
		cfg,
		selectedProvider,
		statusOutput,
		defaultTerminalProviderDependencies(),
	)
}

func buildTerminalAIRouterWithDependencies(
	ctx context.Context,
	cfg config.AIConfig,
	selectedProvider string,
	statusOutput io.Writer,
	deps terminalProviderDependencies,
) (*ai.Router, error) {
	if ctx == nil {
		return nil, errors.New("context is required")
	}
	if statusOutput == nil {
		statusOutput = io.Discard
	}
	selectedProvider = strings.ToLower(strings.TrimSpace(selectedProvider))

	var codexLogin managedCodexLogin
	if cfg.Codex.Enabled && (selectedProvider == "" || selectedProvider == "codex") {
		executable, err := deps.findExecutable("codex")
		if err != nil || strings.TrimSpace(executable) == "" {
			if selectedProvider == "codex" || codexMustBeReady(cfg) {
				return nil, errors.New("codex CLI is not installed or is not on PATH")
			}
		} else {
			codexLogin = deps.newCodexLogin(ctx, cfg.Codex.Home, executable)
			if codexLogin == nil || !codexLogin.Available() {
				if selectedProvider == "codex" || codexMustBeReady(cfg) {
					return nil, errors.New("managed Codex requires an isolated LEARN_AI_CODEX_HOME")
				}
				codexLogin = nil
			}
		}
	}

	if codexLogin != nil && (selectedProvider == "codex" || codexMustBeReady(cfg)) {
		if err := ensureManagedCodexLogin(ctx, codexLogin, statusOutput, deps.pollInterval); err != nil {
			return nil, err
		}
	}

	router := ai.NewRouter()
	if selectedProvider != "" {
		plan, err := airouter.PrepareProviderWithCodexAuth(selectedProvider, cfg, codexLogin)
		if err != nil {
			return nil, err
		}
		plan.Apply(router)
	} else {
		router = airouter.SetupWithCodexAuth(cfg, codexLogin)
	}
	if !router.HasProvider() {
		return nil, errors.New("no AI providers configured")
	}
	return router, nil
}

func codexMustBeReady(cfg config.AIConfig) bool {
	if !cfg.Codex.Enabled {
		return false
	}
	preferred := strings.ToLower(strings.TrimSpace(cfg.DefaultProvider))
	if preferred != "" {
		return preferred == "codex"
	}
	for _, name := range airouter.ProviderNames() {
		if name == "codex" || name == "mock" {
			continue
		}
		if airouter.HasProviderConfiguration(name, cfg) {
			return false
		}
	}
	return true
}

func ensureManagedCodexLogin(
	ctx context.Context,
	login managedCodexLogin,
	statusOutput io.Writer,
	pollInterval time.Duration,
) error {
	if login == nil {
		return errors.New("managed Codex login is required")
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := login.Refresh(ctx); err == nil {
		_, _ = fmt.Fprintln(statusOutput, "Codex provider ready.")
		return nil
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	if _, err := login.Start(); err != nil {
		status := login.Status()
		if strings.TrimSpace(status.Message) != "" {
			return errors.New(status.Message)
		}
		return fmt.Errorf("start Codex device login: %w", err)
	}
	if pollInterval <= 0 {
		pollInterval = managedCodexPollInterval
	}

	ticker := time.NewTicker(pollInterval)
	defer ticker.Stop()
	instructionsShown := false
	for {
		status := login.Status()
		switch status.State {
		case codexauth.StateConnected:
			_, _ = fmt.Fprintln(statusOutput, "Codex connected.")
			return nil
		case codexauth.StateAwaiting:
			if !instructionsShown {
				_, _ = fmt.Fprintf(
					statusOutput,
					"Codex login required. Open %s and enter code %s.\nWaiting for authorization…\n",
					status.VerificationURL,
					status.UserCode,
				)
				instructionsShown = true
			}
		case codexauth.StateFailed:
			if strings.TrimSpace(status.Message) != "" {
				return errors.New(status.Message)
			}
			return errors.New("codex device login failed")
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
		}
	}
}
