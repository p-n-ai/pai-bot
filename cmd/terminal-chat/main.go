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

	"github.com/coder/websocket"

	"github.com/p-n-ai/pai-bot/internal/agent"
	"github.com/p-n-ai/pai-bot/internal/ai"
	"github.com/p-n-ai/pai-bot/internal/curriculum"
	"github.com/p-n-ai/pai-bot/internal/platform/config"
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

	flag.StringVar(&userID, "user-id", "terminal-user", "stable user id for the terminal session")
	flag.StringVar(&language, "lang", "", "preferred language override (en, ms, zh)")
	flag.StringVar(&channel, "channel", "terminal", "channel name for store scoping (use 'telegram' to share state with the live bot)")
	flag.BoolVar(&memory, "memory", false, "use in-memory session state instead of PostgreSQL")
	flag.BoolVar(&multi, "multi", false, "multi-user mode: prefix lines with N: to switch users (e.g., 1:hello, 2:/challenge ABC)")
	flag.IntVar(&userCount, "users", 2, "number of simulated users in multi-user mode")
	flag.StringVar(&wsURL, "ws", "", "WebSocket server URL (e.g. ws://localhost:8080/ws/chat); when set, runs as pure WS client")
	flag.Parse()

	if wsURL != "" {
		if err := runWSClient(wsURL, userID); err != nil {
			fmt.Fprintf(os.Stderr, "ws client error: %v\n", err)
			os.Exit(1)
		}
		return
	}

	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
		Level: slog.LevelWarn,
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

	state, cleanup, err := terminalchat.BuildState(context.Background(), cfg, terminalchat.StateOptions{
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

	engine := agent.NewEngine(agent.EngineConfig{
		AIRouter:             router,
		Store:                state.Store,
		EventLogger:          state.EventLogger,
		CurriculumLoader:     loader,
		DisableMultiLanguage: cfg.Features.DisableMultiLanguage,
		RatingPromptEvery:    cfg.Features.RatingPromptEvery,
		Tracker:              state.Tracker,
		Streaks:              progress.NewMemoryStreakTracker(),
		XP:                   progress.NewMemoryXPTracker(),
		Goals:                goalStore,
		Challenges:           challengeStore,
		DevMode:              cfg.Features.DevMode,
	})

	if multi {
		if err := terminalchat.RunMulti(context.Background(), os.Stdin, os.Stdout, engine, terminalchat.MultiConfig{
			UserCount:  userCount,
			UserPrefix: userID,
			Channel:    channel,
		}); err != nil {
			fmt.Fprintf(os.Stderr, "terminal chat error: %v\n", err)
			os.Exit(1)
		}
	} else {
		if err := terminalchat.Run(context.Background(), os.Stdin, os.Stdout, engine, terminalchat.Config{
			UserID:  userID,
			Channel: channel,
		}); err != nil {
			fmt.Fprintf(os.Stderr, "terminal chat error: %v\n", err)
			os.Exit(1)
		}
	}
}

func setupAIRouter(cfg *config.Config) *ai.Router {
	router := ai.NewRouter()

	if cfg.AI.OpenAI.APIKey != "" {
		router.Register("openai", ai.NewOpenAIProvider(cfg.AI.OpenAI.APIKey))
	}

	if cfg.AI.Anthropic.APIKey != "" {
		provider, err := ai.NewAnthropicProvider(cfg.AI.Anthropic.APIKey)
		if err != nil {
			slog.Warn("failed to create Anthropic provider", "error", err)
		} else {
			router.Register("anthropic", provider)
		}
	}

	if cfg.AI.DeepSeek.APIKey != "" {
		router.Register("deepseek", ai.NewDeepSeekProvider(cfg.AI.DeepSeek.APIKey))
	}

	if cfg.AI.Google.APIKey != "" {
		router.Register("google", ai.NewGoogleProvider(cfg.AI.Google.APIKey))
	}

	if cfg.AI.Ollama.Enabled {
		router.Register("ollama", ai.NewOllamaProvider(cfg.AI.Ollama.URL))
	}

	if cfg.AI.OpenRouter.APIKey != "" {
		router.Register("openrouter", ai.NewOpenRouterProvider(cfg.AI.OpenRouter.APIKey))
	}

	return router
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
