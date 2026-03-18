package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"strings"

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
	var memory bool

	flag.StringVar(&userID, "user-id", "terminal-user", "stable user id for the terminal session")
	flag.StringVar(&language, "lang", "", "preferred language override (en, ms, zh)")
	flag.BoolVar(&memory, "memory", false, "use in-memory session state instead of PostgreSQL")
	flag.Parse()

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
		Channel: "terminal",
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
		goalStore = agent.NewPostgresGoalStoreForChannel(state.DB.Pool, state.TenantID, "terminal")
		challengeStore = agent.NewPostgresChallengeStoreForChannel(state.DB.Pool, state.TenantID, "terminal")
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

	if err := terminalchat.Run(context.Background(), os.Stdin, os.Stdout, engine, terminalchat.Config{
		UserID:  userID,
		Channel: "terminal",
	}); err != nil {
		fmt.Fprintf(os.Stderr, "terminal chat error: %v\n", err)
		os.Exit(1)
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
