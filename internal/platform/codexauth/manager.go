// Copyright 2026 the P&AI authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

// Package codexauth owns the server-side Codex app-server login session.
package codexauth

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/p-n-ai/pai-bot/internal/ai"
	"github.com/p-n-ai/pai-bot/internal/jsonobject"
)

const requestTimeout = 15 * time.Second

var userCodePattern = regexp.MustCompile(`^[A-Z0-9]{4,8}-[A-Z0-9]{4,8}$`)

type State string

const (
	StateDisconnected State = "disconnected"
	StateStarting     State = "starting"
	StateAwaiting     State = "awaiting_authorization"
	StateConnected    State = "connected"
	StateFailed       State = "failed"
)

type Status struct {
	State           State  `json:"state"`
	VerificationURL string `json:"verificationUrl,omitempty"`
	UserCode        string `json:"userCode,omitempty"`
	Message         string `json:"message,omitempty"`
}

type commandFactory func(context.Context, string, ...string) *exec.Cmd

type rpcError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

type rpcRequestError struct {
	code int
}

func (e *rpcRequestError) Error() string {
	return fmt.Sprintf("codex app-server request failed (%d)", e.code)
}

type rpcMessage struct {
	ID     json.RawMessage `json:"id,omitempty"`
	Method string          `json:"method,omitempty"`
	Params json.RawMessage `json:"params,omitempty"`
	Result json.RawMessage `json:"result,omitempty"`
	Error  *rpcError       `json:"error,omitempty"`
}

type rpcRequest struct {
	ID     int64             `json:"id"`
	Method string            `json:"method"`
	Params jsonobject.Object `json:"params"`
}

type rpcNotification struct {
	Method string            `json:"method"`
	Params jsonobject.Object `json:"params"`
}

type rpcFailure struct {
	ID    json.RawMessage `json:"id"`
	Error rpcError        `json:"error"`
}

type pendingResponse struct {
	message rpcMessage
	err     error
}

type completionResult struct {
	response ai.CompletionResponse
	err      error
}

type completionState struct {
	waiter       chan completionResult
	inputTokens  int
	outputTokens int
	messages     []agentMessageItem
}

type Manager struct {
	mu          sync.RWMutex
	startMu     sync.Mutex
	writeMu     sync.Mutex
	parent      context.Context
	home        string
	executable  string
	command     commandFactory
	onConnected func(context.Context) error
	status      Status
	loginID     string
	nextID      int64
	running     bool
	stdin       io.WriteCloser
	cancel      context.CancelFunc
	pending     map[int64]chan pendingResponse
	completions map[string]*completionState
	workspace   string
}

func New(parent context.Context, home, executable string, onConnected func(context.Context) error) *Manager {
	if parent == nil {
		parent = context.Background()
	}
	return &Manager{
		parent:      parent,
		home:        strings.TrimSpace(home),
		executable:  strings.TrimSpace(executable),
		command:     exec.CommandContext,
		onConnected: onConnected,
		status:      Status{State: StateDisconnected},
		pending:     make(map[int64]chan pendingResponse),
		completions: make(map[string]*completionState),
	}
}

func (m *Manager) SetOnConnected(callback func(context.Context) error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.onConnected = callback
}

// Available reports whether this manager can start an isolated app-server.
func (m *Manager) Available() bool {
	return m.executable != "" && m.home != ""
}

// Initialize reconciles the displayed state with the isolated app-server account.
func (m *Manager) Initialize() {
	go func() {
		ctx, cancel := context.WithTimeout(m.parent, requestTimeout)
		defer cancel()
		if err := m.refresh(ctx, false); err != nil {
			if m.executable == "" {
				m.fail("Codex CLI is not installed on the server.")
			}
		}
	}()
}

func (m *Manager) Status() Status {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.status
}

func (m *Manager) Start() (Status, error) {
	m.mu.Lock()
	if m.status.State == StateStarting ||
		m.status.State == StateAwaiting {
		status := m.status
		m.mu.Unlock()
		return status, nil
	}
	if m.executable == "" {
		m.status = Status{State: StateFailed, Message: "Codex CLI is not installed on the server."}
		status := m.status
		m.mu.Unlock()
		return status, errors.New("codex executable unavailable")
	}
	if m.home == "" {
		m.status = Status{State: StateFailed, Message: "Codex server credential storage is not configured."}
		status := m.status
		m.mu.Unlock()
		return status, errors.New("codex home unavailable")
	}
	m.status = Status{State: StateStarting}
	status := m.status
	m.mu.Unlock()

	go m.startDeviceLogin()
	return status, nil
}

// Refresh asks Codex app-server to refresh its managed ChatGPT login.
func (m *Manager) Refresh(ctx context.Context) error {
	return m.refresh(ctx, true)
}

func (m *Manager) refresh(ctx context.Context, refreshToken bool) error {
	result, err := m.call(ctx, "account/read", jsonobject.New(jsonobject.Member("refreshToken", refreshToken)))
	if err != nil {
		if refreshToken {
			m.setStatusUnlessLoginActive(Status{
				State:   StateFailed,
				Message: "Codex login could not be refreshed. Reconnect it in Admin.",
			})
		}
		return errors.New("refresh Codex account")
	}
	var response struct {
		Account *struct {
			Type string `json:"type"`
		} `json:"account"`
	}
	if json.Unmarshal(result, &response) != nil {
		return errors.New("decode Codex account")
	}
	if response.Account == nil || response.Account.Type != "chatgpt" {
		m.setStatusUnlessLoginActive(Status{State: StateDisconnected})
		return errors.New("codex ChatGPT login is not connected")
	}
	m.setStatusUnlessLoginActive(Status{State: StateConnected})
	return nil
}

func (m *Manager) startDeviceLogin() {
	ctx, cancel := context.WithTimeout(m.parent, requestTimeout)
	defer cancel()
	result, err := m.call(ctx, "account/login/start", jsonobject.New(jsonobject.Member("type", "chatgptDeviceCode")))
	if err != nil {
		m.failStarting("Codex device login could not be started.")
		return
	}
	var response struct {
		Type            string `json:"type"`
		LoginID         string `json:"loginId"`
		VerificationURL string `json:"verificationUrl"`
		UserCode        string `json:"userCode"`
	}
	if json.Unmarshal(result, &response) != nil ||
		response.Type != "chatgptDeviceCode" ||
		strings.TrimSpace(response.LoginID) == "" ||
		response.VerificationURL != "https://auth.openai.com/codex/device" ||
		!userCodePattern.MatchString(response.UserCode) {
		m.fail("Codex device login returned invalid authorization instructions.")
		return
	}
	m.mu.Lock()
	m.loginID = response.LoginID
	m.status = Status{
		State:           StateAwaiting,
		VerificationURL: response.VerificationURL,
		UserCode:        response.UserCode,
	}
	m.mu.Unlock()
}

func (m *Manager) ensureProcess(ctx context.Context) error {
	m.startMu.Lock()
	defer m.startMu.Unlock()

	m.mu.RLock()
	running := m.running
	m.mu.RUnlock()
	if running {
		return nil
	}
	if m.executable == "" || m.home == "" {
		return errors.New("codex app-server is unavailable")
	}
	if err := os.MkdirAll(m.home, 0o700); err != nil {
		return errors.New("prepare Codex home")
	}
	homeInfo, err := os.Lstat(m.home)
	if err != nil || !homeInfo.IsDir() {
		return errors.New("codex home must be a real directory")
	}
	workspace, err := os.MkdirTemp("", "pai-bot-codex-runtime-")
	if err != nil {
		return errors.New("prepare Codex runtime workspace")
	}

	processCtx, cancel := context.WithCancel(m.parent)
	cmd := m.command(processCtx, m.executable, "app-server", "--listen", "stdio://")
	cmd.Env = withCodexHome(os.Environ(), m.home)
	stdin, err := cmd.StdinPipe()
	if err != nil {
		cancel()
		_ = os.RemoveAll(workspace)
		return errors.New("open Codex app-server input")
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		cancel()
		_ = os.RemoveAll(workspace)
		return errors.New("open Codex app-server output")
	}
	cmd.Stderr = io.Discard
	if err := cmd.Start(); err != nil {
		cancel()
		_ = os.RemoveAll(workspace)
		return errors.New("start Codex app-server")
	}

	m.mu.Lock()
	m.running = true
	m.stdin = stdin
	m.cancel = cancel
	m.workspace = workspace
	m.mu.Unlock()
	go m.readLoop(cmd, stdout)

	initCtx, initCancel := context.WithTimeout(ctx, requestTimeout)
	defer initCancel()
	if _, err := m.callRunning(initCtx, "initialize", jsonobject.New(
		jsonobject.Member("clientInfo", struct {
			Name    string `json:"name"`
			Title   string `json:"title"`
			Version string `json:"version"`
		}{Name: "pai_bot", Title: "PaiBot", Version: "1"}),
	)); err != nil {
		cancel()
		return errors.New("initialize Codex app-server")
	}
	if err := m.notify(rpcNotification{Method: "initialized", Params: jsonobject.New()}); err != nil {
		cancel()
		return errors.New("acknowledge Codex app-server")
	}
	return nil
}

func (m *Manager) call(ctx context.Context, method string, params jsonobject.Object) (json.RawMessage, error) {
	if err := m.ensureProcess(ctx); err != nil {
		return nil, err
	}
	return m.callRunning(ctx, method, params)
}

func (m *Manager) callRunning(ctx context.Context, method string, params jsonobject.Object) (json.RawMessage, error) {
	m.mu.Lock()
	m.nextID++
	id := m.nextID
	response := make(chan pendingResponse, 1)
	m.pending[id] = response
	m.mu.Unlock()

	if err := writeCodexMessage(m, rpcRequest{ID: id, Method: method, Params: params}); err != nil {
		m.removePending(id)
		return nil, err
	}
	select {
	case result := <-response:
		if result.err != nil {
			return nil, result.err
		}
		if result.message.Error != nil {
			return nil, &rpcRequestError{
				code: result.message.Error.Code,
			}
		}
		return result.message.Result, nil
	case <-ctx.Done():
		m.removePending(id)
		return nil, ctx.Err()
	}
}

func (m *Manager) notify(message rpcNotification) error {
	return writeCodexMessage(m, message)
}

func writeCodexMessage[T any](m *Manager, message T) error {
	m.writeMu.Lock()
	defer m.writeMu.Unlock()
	m.mu.RLock()
	stdin := m.stdin
	running := m.running
	m.mu.RUnlock()
	if !running || stdin == nil {
		return errors.New("codex app-server is not running")
	}
	if err := json.NewEncoder(stdin).Encode(message); err != nil {
		return errors.New("write Codex app-server request")
	}
	return nil
}

func (m *Manager) readLoop(cmd *exec.Cmd, output io.Reader) {
	scanner := bufio.NewScanner(output)
	scanner.Buffer(make([]byte, 64*1024), 1<<20)
	for scanner.Scan() {
		var message rpcMessage
		if json.Unmarshal(scanner.Bytes(), &message) != nil {
			continue
		}
		if len(message.ID) > 0 && string(message.ID) != "null" {
			if message.Method != "" {
				m.rejectServerRequest(message.ID)
				continue
			}
			var id int64
			if json.Unmarshal(message.ID, &id) == nil {
				m.resolve(id, pendingResponse{message: message})
			}
			continue
		}
		switch message.Method {
		case "account/login/completed":
			m.handleLoginCompleted(message.Params)
		case "thread/tokenUsage/updated":
			m.handleTokenUsage(message.Params)
		case "item/completed":
			m.handleItemCompleted(message.Params)
		case "turn/completed":
			m.handleTurnCompleted(message.Params)
		}
	}
	_ = cmd.Wait()
	m.processStopped()
}

func (m *Manager) rejectServerRequest(id json.RawMessage) {
	_ = writeCodexMessage(m, rpcFailure{
		ID: id,
		Error: rpcError{
			Code:    -32601,
			Message: "PaiBot does not expose interactive Codex tools",
		},
	})
}

func (m *Manager) handleLoginCompleted(params json.RawMessage) {
	var completed struct {
		LoginID string `json:"loginId"`
		Success bool   `json:"success"`
	}
	if json.Unmarshal(params, &completed) != nil {
		return
	}
	m.mu.RLock()
	activeLoginID := m.loginID
	m.mu.RUnlock()
	if activeLoginID == "" || completed.LoginID != activeLoginID {
		return
	}
	if !completed.Success {
		m.fail("Codex device login failed. Start a new login.")
		return
	}

	m.mu.RLock()
	callback := m.onConnected
	m.mu.RUnlock()
	if callback != nil {
		ctx, cancel := context.WithTimeout(m.parent, requestTimeout)
		err := callback(ctx)
		cancel()
		if err != nil {
			m.fail("Codex connected, but PaiBot could not select it as the default provider.")
			return
		}
	}
	m.mu.Lock()
	m.loginID = ""
	m.status = Status{State: StateConnected}
	m.mu.Unlock()
}

func (m *Manager) resolve(id int64, response pendingResponse) {
	m.mu.Lock()
	waiter := m.pending[id]
	delete(m.pending, id)
	m.mu.Unlock()
	if waiter != nil {
		waiter <- response
	}
}

func (m *Manager) removePending(id int64) {
	m.mu.Lock()
	delete(m.pending, id)
	m.mu.Unlock()
}

func (m *Manager) processStopped() {
	m.mu.Lock()
	m.running = false
	m.stdin = nil
	m.cancel = nil
	workspace := m.workspace
	m.workspace = ""
	if m.status.State == StateStarting || m.status.State == StateAwaiting {
		m.status = Status{
			State:   StateFailed,
			Message: "Codex device login stopped before authorization completed.",
		}
		m.loginID = ""
	} else if m.status.State == StateConnected && m.parent.Err() == nil {
		m.status = Status{
			State:   StateFailed,
			Message: "Codex app-server stopped unexpectedly. Reconnect it in Admin.",
		}
	}
	pending := m.pending
	m.pending = make(map[int64]chan pendingResponse)
	completions := m.completions
	m.completions = make(map[string]*completionState)
	m.mu.Unlock()
	if workspace != "" {
		_ = os.RemoveAll(workspace)
	}
	for _, waiter := range pending {
		waiter <- pendingResponse{err: errors.New("codex app-server stopped")}
	}
	for _, completion := range completions {
		completion.waiter <- completionResult{err: errors.New("codex app-server stopped")}
	}
}

func (m *Manager) fail(message string) {
	m.setStatus(Status{State: StateFailed, Message: message})
}

func (m *Manager) failStarting(message string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.status.State == StateStarting {
		m.status = Status{State: StateFailed, Message: message}
	}
}

func (m *Manager) setStatus(status Status) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.status = status
}

func (m *Manager) setStatusUnlessLoginActive(status Status) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.status.State == StateStarting || m.status.State == StateAwaiting {
		return
	}
	m.status = status
}

func withCodexHome(environment []string, home string) []string {
	result := make([]string, 0, len(environment)+1)
	for _, variable := range environment {
		if strings.HasPrefix(variable, "CODEX_HOME=") {
			continue
		}
		result = append(result, variable)
	}
	return append(result, "CODEX_HOME="+filepath.Clean(home))
}
