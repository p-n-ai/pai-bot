// Copyright 2026 the P&AI authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"bufio"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/coder/websocket"

	"github.com/p-n-ai/pai-bot/internal/agent"
	"github.com/p-n-ai/pai-bot/internal/ai"
	"github.com/p-n-ai/pai-bot/internal/chat"
	"github.com/p-n-ai/pai-bot/internal/curriculum"
	"github.com/p-n-ai/pai-bot/internal/platform/airouter"
	"github.com/p-n-ai/pai-bot/internal/platform/config"
	"github.com/p-n-ai/pai-bot/internal/platform/featureflags"
	"github.com/p-n-ai/pai-bot/internal/progress"
	"github.com/p-n-ai/pai-bot/internal/terminalchat"
)

func main() {
	var userID string
	var language string
	var channel string
	var memory bool
	var multi bool
	var userCount int
	var wsURL string
	var oneShotMessage string
	var verbose bool
	var progressSideEffects bool
	var historyJSONPath string
	var dumpJSONPath string
	var dumpTurnLimit int

	flag.StringVar(&userID, "user-id", "terminal-user", "stable user id for the terminal session")
	flag.StringVar(&language, "lang", "", "preferred language override (en, ms, zh)")
	flag.StringVar(&channel, "channel", "terminal", "channel name for store scoping (use 'telegram' to share state with the live bot)")
	flag.BoolVar(&memory, "memory", false, "use in-memory session state instead of PostgreSQL")
	flag.BoolVar(&multi, "multi", false, "multi-user mode: prefix lines with N: to switch users (e.g., 1:hello, 2:/challenge ABC)")
	flag.IntVar(&userCount, "users", 2, "number of simulated users in multi-user mode")
	flag.StringVar(&wsURL, "ws", "", "WebSocket server URL (e.g. ws://localhost:8080/ws/chat); when set, runs as pure WS client")
	flag.StringVar(&oneShotMessage, "message", "", "send one WebSocket message and print one response; requires --ws")
	flag.BoolVar(&verbose, "verbose", false, "show diagnostic warnings from curriculum loading and background checks")
	flag.BoolVar(&progressSideEffects, "progress", false, "enable mastery, streak, and XP side effects in local terminal sessions")
	flag.StringVar(&historyJSONPath, "history-json", "", "write local terminal conversation history to a JSON file when the session ends")
	flag.StringVar(&dumpJSONPath, "dump-json", "", "write local terminal conversation history plus model-facing AI request/response traces to a JSON file when the session ends")
	flag.IntVar(&dumpTurnLimit, "turn-limit", 0, "limit exported conversation turns and model calls to the latest N items; 0 exports everything")
	flag.Parse()

	if wsURL != "" {
		if oneShotMessage != "" {
			if err := runWSClientOnce(wsURL, userID, oneShotMessage); err != nil {
				fmt.Fprintf(os.Stderr, "ws client error: %v\n", err)
				os.Exit(1)
			}
			return
		}
		if err := runWSClient(wsURL, userID); err != nil {
			fmt.Fprintf(os.Stderr, "ws client error: %v\n", err)
			os.Exit(1)
		}
		return
	}

	logLevel := slog.LevelError
	if verbose {
		logLevel = slog.LevelWarn
	}
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
		Level: logLevel,
	}))
	slog.SetDefault(logger)

	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "load config: %v\n", err)
		os.Exit(1)
	}
	if !cfg.HasAIProvider() {
		fmt.Fprintln(os.Stderr, "at least one AI provider must be configured")
		os.Exit(1)
	}

	router := setupAIRouter(cfg)
	if !router.HasProvider() {
		fmt.Fprintln(os.Stderr, "no AI providers configured")
		os.Exit(1)
	}

	var loader *curriculum.Loader
	loader, err = curriculum.NewLoader(cfg.CurriculumPath)
	if err != nil {
		slog.Warn("curriculum not loaded", "path", cfg.CurriculumPath, "error", err)
	}

	state, cleanup, err := terminalchat.BuildState(context.Background(), cfg.Database, terminalchat.StateOptions{
		Memory:  memory,
		Channel: channel,
	}, terminalchat.StateDeps{})
	if err != nil {
		fmt.Fprintf(os.Stderr, "build terminal chat state: %v\n", err)
		os.Exit(1)
	}
	defer cleanup()

	if lang := strings.TrimSpace(language); lang != "" {
		if err := state.Store.SetUserPreferredLanguage(userID, lang); err != nil {
			fmt.Fprintf(os.Stderr, "set preferred language: %v\n", err)
			os.Exit(1)
		}
	}
	var goalStore agent.GoalStore
	var challengeStore agent.ChallengeStore
	if memory {
		goalStore = agent.NewMemoryGoalStore()
		challengeStore = agent.NewMemoryChallengeStore()
	} else {
		goalStore = agent.NewPostgresGoalStoreForChannel(state.DB.Pool, state.TenantID, channel)
		challengeStore = agent.NewPostgresChallengeStoreForChannel(state.DB.Pool, state.TenantID, channel)
	}

	engineCfg := agent.EngineConfig{
		AIRouter:             router,
		Store:                state.Store,
		EventLogger:          state.EventLogger,
		CurriculumLoader:     loader,
		DisableMultiLanguage: cfg.Runtime.DisableMultiLanguage,
		RatingPromptEvery:    cfg.Runtime.RatingPromptEvery,
		Goals:                goalStore,
		Challenges:           challengeStore,
		DevMode:              cfg.Runtime.DevMode,
		FeatureFlags:         func() featureflags.Features { return cfg.FeatureFlags },
	}
	if cfg.Runtime.DevMode {
		engineCfg.TurnHookNotice = func(notice agent.TurnHookCallNotice) {
			_, _ = fmt.Fprintf(os.Stdout, "turn hook called: %s outcome=%s\n", notice.Name, notice.Outcome)
		}
	}
	if progressSideEffects {
		engineCfg.Tracker = state.Tracker
		engineCfg.Streaks = progress.NewMemoryStreakTracker()
		engineCfg.XP = progress.NewMemoryXPTracker()
	}
	engine := agent.NewEngine(engineCfg)

	processor := terminalchat.Processor(engine)
	var history *conversationHistory
	if strings.TrimSpace(historyJSONPath) != "" || strings.TrimSpace(dumpJSONPath) != "" {
		history = newConversationHistory(userID, channel)
		processor = &historyProcessor{inner: processor, history: history}
	}
	if history != nil && strings.TrimSpace(dumpJSONPath) != "" {
		router.SetTraceFunc(history.appendAITrace)
	}

	var runErr error
	if multi {
		runErr = terminalchat.RunMulti(context.Background(), os.Stdin, os.Stdout, processor, terminalchat.MultiConfig{
			UserCount:  userCount,
			UserPrefix: userID,
			Channel:    channel,
		})
	} else {
		runErr = terminalchat.Run(context.Background(), os.Stdin, os.Stdout, processor, terminalchat.Config{
			UserID:  userID,
			Channel: channel,
		})
	}
	if history != nil {
		for _, path := range []string{historyJSONPath, dumpJSONPath} {
			if strings.TrimSpace(path) == "" {
				continue
			}
			if err := writeConversationHistory(path, history, dumpTurnLimit); err != nil {
				fmt.Fprintf(os.Stderr, "write history: %v\n", err)
				os.Exit(1)
			}
		}
	}
	if runErr != nil {
		fmt.Fprintf(os.Stderr, "terminal chat error: %v\n", runErr)
		os.Exit(1)
	}
}

func setupAIRouter(cfg *config.Config) *ai.Router {
	return airouter.Setup(cfg.AI)
}

type conversationHistory struct {
	mu         sync.Mutex             `json:"-"`
	UserID     string                 `json:"user_id"`
	Channel    string                 `json:"channel"`
	CreatedAt  time.Time              `json:"created_at"`
	TurnLimit  int                    `json:"turn_limit,omitempty"`
	Turns      []conversationTurnJSON `json:"turns"`
	ModelCalls []modelCallJSON        `json:"model_calls,omitempty"`
}

type conversationHistorySnapshot struct {
	UserID     string                 `json:"user_id"`
	Channel    string                 `json:"channel"`
	CreatedAt  time.Time              `json:"created_at"`
	TurnLimit  int                    `json:"turn_limit,omitempty"`
	Turns      []conversationTurnJSON `json:"turns"`
	ModelCalls []modelCallJSON        `json:"model_calls,omitempty"`
}

type conversationTurnJSON struct {
	UserID    string    `json:"user_id"`
	Channel   string    `json:"channel"`
	Role      string    `json:"role"`
	Text      string    `json:"text"`
	Timestamp time.Time `json:"timestamp"`
}

type modelCallJSON struct {
	Provider    string                 `json:"provider"`
	Task        string                 `json:"task"`
	Model       string                 `json:"model,omitempty"`
	MaxTokens   int                    `json:"max_tokens,omitempty"`
	Temperature float64                `json:"temperature,omitempty"`
	StartedAt   time.Time              `json:"started_at"`
	CompletedAt time.Time              `json:"completed_at"`
	Messages    []ai.Message           `json:"messages"`
	Response    *modelCallResponseJSON `json:"response,omitempty"`
	Error       string                 `json:"error,omitempty"`
}

type modelCallResponseJSON struct {
	Content      string `json:"content"`
	Model        string `json:"model"`
	InputTokens  int    `json:"input_tokens"`
	OutputTokens int    `json:"output_tokens"`
}

type historyProcessor struct {
	inner   terminalchat.Processor
	history *conversationHistory
}

func newConversationHistory(userID, channel string) *conversationHistory {
	return &conversationHistory{
		UserID:     strings.TrimSpace(userID),
		Channel:    strings.TrimSpace(channel),
		CreatedAt:  time.Now(),
		Turns:      []conversationTurnJSON{},
		ModelCalls: []modelCallJSON{},
	}
}

func (p *historyProcessor) ProcessMessage(ctx context.Context, msg chat.InboundMessage) (string, error) {
	p.append(msg.UserID, msg.Channel, "student", msg.Text)
	resp, err := p.inner.ProcessMessage(ctx, msg)
	if err != nil {
		p.append(msg.UserID, msg.Channel, "error", err.Error())
		return resp, err
	}
	p.append(msg.UserID, msg.Channel, "assistant", strings.TrimSpace(resp))
	return resp, nil
}

func (p *historyProcessor) append(userID, channel, role, text string) {
	if p == nil || p.history == nil {
		return
	}
	p.history.mu.Lock()
	defer p.history.mu.Unlock()
	p.history.Turns = append(p.history.Turns, conversationTurnJSON{
		UserID:    userID,
		Channel:   channel,
		Role:      role,
		Text:      text,
		Timestamp: time.Now(),
	})
}

func (h *conversationHistory) appendAITrace(trace ai.CompletionTrace) {
	if h == nil {
		return
	}
	call := modelCallJSON{
		Provider:    trace.Provider,
		Task:        trace.Request.Task.String(),
		Model:       trace.Request.Model,
		MaxTokens:   trace.Request.MaxTokens,
		Temperature: trace.Request.Temperature,
		StartedAt:   trace.StartedAt,
		CompletedAt: trace.CompletedAt,
		Messages:    append([]ai.Message(nil), trace.Request.Messages...),
		Error:       trace.Error,
	}
	if trace.Response != nil {
		call.Response = &modelCallResponseJSON{
			Content:      trace.Response.Content,
			Model:        trace.Response.Model,
			InputTokens:  trace.Response.InputTokens,
			OutputTokens: trace.Response.OutputTokens,
		}
		if call.Model == "" {
			call.Model = trace.Response.Model
		}
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	h.ModelCalls = append(h.ModelCalls, call)
}

func writeConversationHistory(path string, history *conversationHistory, turnLimit int) error {
	if history == nil {
		return nil
	}
	snapshot := history.snapshot(turnLimit)
	b, err := json.MarshalIndent(snapshot, "", "  ")
	if err != nil {
		return err
	}
	if err := os.WriteFile(path, append(b, '\n'), 0o600); err != nil {
		return err
	}
	return os.Chmod(path, 0o600)
}

func (h *conversationHistory) snapshot(turnLimit int) conversationHistorySnapshot {
	h.mu.Lock()
	defer h.mu.Unlock()
	turns := latestItems(h.Turns, turnLimit)
	modelCalls := latestItems(h.ModelCalls, turnLimit)
	return conversationHistorySnapshot{
		UserID:     h.UserID,
		Channel:    h.Channel,
		CreatedAt:  h.CreatedAt,
		TurnLimit:  normalizedTurnLimit(turnLimit),
		Turns:      turns,
		ModelCalls: modelCalls,
	}
}

func latestItems[T any](items []T, limit int) []T {
	if limit <= 0 || len(items) <= limit {
		return append([]T(nil), items...)
	}
	return append([]T(nil), items[len(items)-limit:]...)
}

func normalizedTurnLimit(limit int) int {
	if limit <= 0 {
		return 0
	}
	return limit
}

// wsInboundMsg mirrors the WebSocket protocol envelope for outgoing client messages.
type wsInboundMsg struct {
	Type   string `json:"type"`
	UserID string `json:"user_id,omitempty"`
	Text   string `json:"text,omitempty"`
}

// wsOutboundMsg mirrors the WebSocket protocol envelope for incoming server messages.
type wsOutboundMsg struct {
	Type string `json:"type"`
	Text string `json:"text,omitempty"`
}

// runWSClient connects to a pai-bot WebSocket server and runs an interactive
// chat session. No local engine or database — pure remote client.
func runWSClient(serverURL, userID string) error {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	conn, _, err := websocket.Dial(ctx, serverURL, nil)
	if err != nil {
		return fmt.Errorf("connecting to %s: %w", serverURL, err)
	}
	defer func() { _ = conn.Close(websocket.StatusNormalClosure, "bye") }()

	// Authenticate.
	authMsg, _ := json.Marshal(wsInboundMsg{Type: "auth", UserID: userID})
	if err := conn.Write(ctx, websocket.MessageText, authMsg); err != nil {
		return fmt.Errorf("sending auth: %w", err)
	}

	// Read auth_ok.
	_, data, err := conn.Read(ctx)
	if err != nil {
		return fmt.Errorf("reading auth response: %w", err)
	}
	var authResp wsOutboundMsg
	if err := json.Unmarshal(data, &authResp); err != nil {
		return fmt.Errorf("parsing auth response: %w", err)
	}
	if authResp.Type != "auth_ok" {
		return fmt.Errorf("expected auth_ok, got %q", authResp.Type)
	}

	fmt.Printf("Connected to %s as %s\n", serverURL, userID)
	fmt.Println("Type a message and press Enter. Ctrl+C to quit.")
	fmt.Println()

	// Read server messages in background.
	go func() {
		for {
			_, data, err := conn.Read(ctx)
			if err != nil {
				if ctx.Err() == nil {
					fmt.Fprintf(os.Stderr, "\nconnection closed: %v\n", err)
				}
				cancel()
				return
			}

			var msg wsOutboundMsg
			if err := json.Unmarshal(data, &msg); err != nil {
				fmt.Fprintf(os.Stderr, "\ninvalid message: %v\n", err)
				continue
			}

			switch msg.Type {
			case "response":
				fmt.Printf("\nBot: %s\n\nYou: ", msg.Text)
			case "notification":
				fmt.Printf("\n[notification] %s\n\nYou: ", msg.Text)
			case "typing":
				// Could show a typing indicator; skip for simplicity.
			default:
				fmt.Printf("\n[%s] %s\n\nYou: ", msg.Type, msg.Text)
			}
		}
	}()

	// Read stdin and send messages.
	fmt.Print("You: ")
	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		text := strings.TrimSpace(scanner.Text())
		if text == "" {
			fmt.Print("You: ")
			continue
		}

		msg, _ := json.Marshal(wsInboundMsg{Type: "message", Text: text})
		if err := conn.Write(ctx, websocket.MessageText, msg); err != nil {
			return fmt.Errorf("sending message: %w", err)
		}
	}

	return scanner.Err()
}

func runWSClientOnce(serverURL, userID, text string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	conn, _, err := websocket.Dial(ctx, serverURL, nil)
	if err != nil {
		return fmt.Errorf("connecting to %s: %w", serverURL, err)
	}
	defer func() { _ = conn.Close(websocket.StatusNormalClosure, "bye") }()

	authMsg, _ := json.Marshal(wsInboundMsg{Type: "auth", UserID: userID})
	if err := conn.Write(ctx, websocket.MessageText, authMsg); err != nil {
		return fmt.Errorf("sending auth: %w", err)
	}
	if err := readExpectedWSMessage(ctx, conn, "auth_ok"); err != nil {
		return err
	}

	msg, _ := json.Marshal(wsInboundMsg{Type: "message", Text: strings.TrimSpace(text)})
	if err := conn.Write(ctx, websocket.MessageText, msg); err != nil {
		return fmt.Errorf("sending message: %w", err)
	}

	for {
		_, data, err := conn.Read(ctx)
		if err != nil {
			return fmt.Errorf("reading response: %w", err)
		}
		var resp wsOutboundMsg
		if err := json.Unmarshal(data, &resp); err != nil {
			return fmt.Errorf("parsing response: %w", err)
		}
		if resp.Type == "typing" {
			continue
		}
		fmt.Printf("%s\n", resp.Text)
		return nil
	}
}

func readExpectedWSMessage(ctx context.Context, conn *websocket.Conn, want string) error {
	_, data, err := conn.Read(ctx)
	if err != nil {
		return fmt.Errorf("reading %s: %w", want, err)
	}
	var resp wsOutboundMsg
	if err := json.Unmarshal(data, &resp); err != nil {
		return fmt.Errorf("parsing %s: %w", want, err)
	}
	if resp.Type != want {
		return fmt.Errorf("expected %s, got %q", want, resp.Type)
	}
	return nil
}
