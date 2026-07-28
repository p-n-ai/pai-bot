// Copyright 2026 the P&AI authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package codexauth

import (
	"bufio"
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"

	"github.com/p-n-ai/pai-bot/internal/ai"
)

func TestDeviceLoginUsesStructuredAppServerFlow(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	connected := make(chan struct{}, 1)
	manager := newTestManager(ctx, t.TempDir(), func(context.Context) error {
		connected <- struct{}{}
		return nil
	})

	status, err := manager.Start()
	if err != nil || status.State != StateStarting {
		t.Fatalf("Start() = %#v, %v; want starting", status, err)
	}
	awaitState(t, manager, StateAwaiting)
	awaitSignal(t, connected)
	awaitState(t, manager, StateConnected)
}

func TestInitializeAndRefreshUseManagedChatGPTAccount(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	manager := newTestManager(ctx, t.TempDir(), nil)

	manager.Initialize()
	awaitState(t, manager, StateConnected)
	if err := manager.Refresh(t.Context()); err != nil {
		t.Fatalf("Refresh() error = %v", err)
	}
}

func TestStartRenewsConnectedAccountWithDeviceLogin(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	manager := newTestManager(ctx, t.TempDir(), nil)
	manager.Initialize()
	awaitState(t, manager, StateConnected)

	status, err := manager.Start()
	if err != nil || status.State != StateStarting {
		t.Fatalf("Start() = %#v, %v; want starting", status, err)
	}
	awaitState(t, manager, StateAwaiting)
}

func TestRefreshDoesNotHideActiveDeviceLogin(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	manager := New(ctx, t.TempDir(), "codex", nil)
	manager.command = helperCommand("codex-app-server-awaits-login")

	if _, err := manager.Start(); err != nil {
		t.Fatalf("Start() error = %v", err)
	}
	awaitState(t, manager, StateAwaiting)
	if err := manager.Refresh(t.Context()); err != nil {
		t.Fatalf("Refresh() error = %v", err)
	}
	if got := manager.Status().State; got != StateAwaiting {
		t.Fatalf("Status().State = %q, want %q", got, StateAwaiting)
	}
}

func TestStartWithoutExecutableReturnsSafeFailure(t *testing.T) {
	manager := New(t.Context(), t.TempDir(), "", nil)

	status, err := manager.Start()

	if err == nil || status != (Status{
		State:   StateFailed,
		Message: "Codex CLI is not installed on the server.",
	}) {
		t.Fatalf("Start() = %#v, %v", status, err)
	}
}

func TestDeviceLoginFailsWhenAppServerStopsBeforeCompletion(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	manager := New(ctx, t.TempDir(), "codex", nil)
	manager.command = helperCommand("codex-app-server-stops-during-login")

	if _, err := manager.Start(); err != nil {
		t.Fatalf("Start() error = %v", err)
	}
	awaitState(t, manager, StateFailed)
	if got := manager.Status().Message; got != "Codex device login stopped before authorization completed." {
		t.Fatalf("Status().Message = %q", got)
	}
}

func TestConnectedStateFailsWhenAppServerStopsUnexpectedly(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	connected := make(chan struct{}, 1)
	manager := New(ctx, t.TempDir(), "codex", func(context.Context) error {
		connected <- struct{}{}
		return nil
	})
	manager.command = helperCommand("codex-app-server-stops-after-login")

	if _, err := manager.Start(); err != nil {
		t.Fatalf("Start() error = %v", err)
	}
	awaitSignal(t, connected)
	awaitState(t, manager, StateFailed)
	if got := manager.Status().Message; got != "Codex app-server stopped unexpectedly. Reconnect it in Admin." {
		t.Fatalf("Status().Message = %q", got)
	}
}

func TestWithCodexHomeReplacesInheritedValue(t *testing.T) {
	got := withCodexHome([]string{"PATH=/bin", "CODEX_HOME=/personal/codex"}, "/server/codex")
	want := []string{"PATH=/bin", "CODEX_HOME=/server/codex"}
	if len(got) != len(want) || got[0] != want[0] || got[1] != want[1] {
		t.Fatalf("withCodexHome() = %v, want %v", got, want)
	}
}

func TestCompleteUsesEphemeralNonInteractiveAppServerThread(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	manager := New(ctx, t.TempDir(), "codex", nil)
	manager.command = helperCommand("codex-app-server-completion")

	response, err := manager.Complete(t.Context(), ai.CompletionRequest{
		Model: "gpt-test",
		Messages: []ai.Message{
			{Role: "system", Content: "Teach clearly."},
			{Role: "user", Content: "Explain fractions."},
		},
		StructuredOutput: &ai.StructuredOutputSpec{
			Name:       "answer",
			JSONSchema: json.RawMessage(`{"type":"object","required":["answer"],"properties":{"answer":{"type":"string"}}}`),
			Strict:     true,
		},
	})

	if err != nil {
		t.Fatalf("Complete() error = %v", err)
	}
	if response.Content != `{"answer":"Use equal parts."}` ||
		string(response.StructuredOutput) != response.Content ||
		response.Model != "gpt-test" ||
		response.InputTokens != 17 ||
		response.OutputTokens != 6 {
		t.Fatalf("Complete() response = %#v", response)
	}
}

func newTestManager(
	ctx context.Context,
	home string,
	onConnected func(context.Context) error,
) *Manager {
	manager := New(ctx, home, "codex", onConnected)
	manager.command = helperCommand("codex-app-server-helper")
	return manager
}

func helperCommand(mode string) commandFactory {
	return func(ctx context.Context, _ string, _ ...string) *exec.Cmd {
		return exec.CommandContext(
			ctx,
			os.Args[0],
			"-test.run=TestCodexAppServerHelperProcess",
			"--",
			mode,
		)
	}
}

func awaitState(t *testing.T, manager *Manager, want State) {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if manager.Status().State == want {
			return
		}
		time.Sleep(time.Millisecond)
	}
	t.Fatalf("Status() = %#v, want %q", manager.Status(), want)
}

func awaitSignal(t *testing.T, signal <-chan struct{}) {
	t.Helper()
	select {
	case <-signal:
	case <-time.After(2 * time.Second):
		t.Fatal("connected callback was not called")
	}
}

func TestCodexAppServerHelperProcess(t *testing.T) {
	if len(os.Args) == 0 {
		return
	}
	mode := os.Args[len(os.Args)-1]
	if mode != "codex-app-server-helper" &&
		mode != "codex-app-server-awaits-login" &&
		mode != "codex-app-server-completion" &&
		mode != "codex-app-server-stops-after-login" &&
		mode != "codex-app-server-stops-during-login" {
		return
	}
	scanner := bufio.NewScanner(os.Stdin)
	encoder := json.NewEncoder(os.Stdout)
	for scanner.Scan() {
		var request struct {
			ID     *int64          `json:"id"`
			Method string          `json:"method"`
			Params json.RawMessage `json:"params"`
		}
		if json.Unmarshal(scanner.Bytes(), &request) != nil || request.ID == nil {
			continue
		}
		switch request.Method {
		case "initialize":
			_ = encoder.Encode(map[string]any{"id": *request.ID, "result": map[string]any{}})
		case "account/read":
			var params struct {
				RefreshToken *bool `json:"refreshToken"`
			}
			if json.Unmarshal(request.Params, &params) != nil || params.RefreshToken == nil {
				_ = encoder.Encode(map[string]any{
					"id":    *request.ID,
					"error": map[string]any{"code": -1, "message": "missing refreshToken"},
				})
				continue
			}
			_ = encoder.Encode(map[string]any{
				"id": *request.ID,
				"result": map[string]any{
					"account": map[string]any{"type": "chatgpt"},
				},
			})
		case "account/login/start":
			if mode == "codex-app-server-stops-during-login" {
				os.Exit(0)
			}
			var params struct {
				Type string `json:"type"`
			}
			if json.Unmarshal(request.Params, &params) != nil || params.Type != "chatgptDeviceCode" {
				_ = encoder.Encode(map[string]any{
					"id":    *request.ID,
					"error": map[string]any{"code": -1, "message": "wrong login type"},
				})
				continue
			}
			_ = encoder.Encode(map[string]any{
				"id": *request.ID,
				"result": map[string]any{
					"type":            "chatgptDeviceCode",
					"loginId":         "login-1",
					"verificationUrl": "https://auth.openai.com/codex/device",
					"userCode":        "ABCD-1234",
				},
			})
			if mode == "codex-app-server-awaits-login" {
				continue
			}
			time.Sleep(25 * time.Millisecond)
			_ = encoder.Encode(map[string]any{
				"method": "account/login/completed",
				"params": map[string]any{
					"loginId": "login-1",
					"success": true,
				},
			})
			if mode == "codex-app-server-stops-after-login" {
				os.Exit(0)
			}
		case "thread/start":
			var params struct {
				BaseInstructions      string `json:"baseInstructions"`
				CWD                   string `json:"cwd"`
				DeveloperInstructions string `json:"developerInstructions"`
				Ephemeral             bool   `json:"ephemeral"`
				Model                 string `json:"model"`
				Sandbox               string `json:"sandbox"`
			}
			if json.Unmarshal(request.Params, &params) != nil ||
				params.BaseInstructions != "Teach clearly." ||
				params.CWD == "" ||
				!strings.Contains(params.DeveloperInstructions, "Do not inspect files") ||
				!params.Ephemeral ||
				params.Model != "gpt-test" ||
				params.Sandbox != "read-only" {
				_ = encoder.Encode(map[string]any{
					"id":    *request.ID,
					"error": map[string]any{"code": -1, "message": "unsafe thread parameters"},
				})
				continue
			}
			_ = encoder.Encode(map[string]any{
				"id": *request.ID,
				"result": map[string]any{
					"thread": map[string]any{"id": "thread-1"},
					"model":  "gpt-test",
				},
			})
		case "turn/start":
			var params struct {
				ApprovalPolicy string           `json:"approvalPolicy"`
				Input          []map[string]any `json:"input"`
				OutputSchema   map[string]any   `json:"outputSchema"`
				ThreadID       string           `json:"threadId"`
			}
			var text string
			if json.Unmarshal(request.Params, &params) != nil {
				text = ""
			} else if len(params.Input) > 0 {
				text, _ = params.Input[0]["text"].(string)
			}
			if params.ApprovalPolicy != "never" ||
				!strings.Contains(text, "USER:\nExplain fractions.") ||
				params.OutputSchema["type"] != "object" ||
				params.ThreadID != "thread-1" {
				_ = encoder.Encode(map[string]any{
					"id":    *request.ID,
					"error": map[string]any{"code": -1, "message": "unsafe turn parameters"},
				})
				continue
			}
			_ = encoder.Encode(map[string]any{
				"id":     *request.ID,
				"result": map[string]any{"turn": map[string]any{"id": "turn-1"}},
			})
			_ = encoder.Encode(map[string]any{
				"method": "thread/tokenUsage/updated",
				"params": map[string]any{
					"threadId": "thread-1",
					"tokenUsage": map[string]any{
						"last": map[string]any{"inputTokens": 17, "outputTokens": 6},
					},
				},
			})
			_ = encoder.Encode(map[string]any{
				"method": "item/completed",
				"params": map[string]any{
					"threadId": "thread-1",
					"turnId":   "turn-1",
					"item": map[string]any{
						"id":    "message-1",
						"type":  "agentMessage",
						"text":  `{"answer":"Use equal parts."}`,
						"phase": "final_answer",
					},
				},
			})
			_ = encoder.Encode(map[string]any{
				"method": "turn/completed",
				"params": map[string]any{
					"threadId": "thread-1",
					"turn": map[string]any{
						"status": "completed",
						"items":  []map[string]any{},
					},
				},
			})
		}
	}
	os.Exit(0)
}
