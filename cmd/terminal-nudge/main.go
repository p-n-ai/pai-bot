package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"os"

	"github.com/p-n-ai/pai-bot/internal/agent"
	"github.com/p-n-ai/pai-bot/internal/ai"
	"github.com/p-n-ai/pai-bot/internal/chat"
	"github.com/p-n-ai/pai-bot/internal/curriculum"
	"github.com/p-n-ai/pai-bot/internal/platform/cache"
	"github.com/p-n-ai/pai-bot/internal/platform/config"
	"github.com/p-n-ai/pai-bot/internal/terminalchat"
	"github.com/p-n-ai/pai-bot/internal/terminalnudge"
)

func main() {
	var userID string

	flag.StringVar(&userID, "user-id", "", "user id to check for due-review nudges")
	flag.Parse()

	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
		Level: slog.LevelWarn,
	}))
	slog.SetDefault(logger)

	if userID == "" {
		fmt.Fprintln(os.Stderr, "--user-id is required")
		os.Exit(1)
	}

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

	// Match server behavior: cache connection is non-fatal.
	if cfg.Cache.URL != "" {
		c, err := cache.New(context.Background(), cfg.Cache.URL)
		if err != nil {
			slog.Warn("cache not connected", "error", err)
		} else {
			defer func() { _ = c.Close() }()
		}
	}

	state, cleanup, err := terminalchat.BuildState(context.Background(), cfg, terminalchat.StateOptions{}, terminalchat.StateDeps{})
	if err != nil {
		fmt.Fprintf(os.Stderr, "build nudge state: %v\n", err)
		os.Exit(1)
	}
	defer cleanup()

	var loader *curriculum.Loader
	loader, err = curriculum.NewLoader(cfg.CurriculumPath)
	if err != nil {
		slog.Warn("curriculum not loaded", "path", cfg.CurriculumPath, "error", err)
	}
	_ = loader

	capture := &terminalnudge.CaptureChannel{}
	gateway := chat.NewGateway()
	gateway.Register("telegram", capture)

	if state.DB == nil || state.TenantID == "" {
		fmt.Fprintln(os.Stderr, "persistent postgres state is required for terminal nudge")
		os.Exit(1)
	}

	scheduler := agent.NewScheduler(
		agent.SchedulerConfig{
			CheckInterval:               agent.DefaultSchedulerConfig().CheckInterval,
			MaxNudgesPerDay:             agent.DefaultSchedulerConfig().MaxNudgesPerDay,
			AIPersonalizedNudgesEnabled: cfg.Features.AIPersonalizedNudgesEnabled,
		},
		state.Tracker,
		nil,
		nil,
		nil,
		agent.NewPostgresNudgeTracker(state.DB.Pool, state.TenantID),
		gateway,
		router,
		state.Store,
	)

	if err := terminalnudge.Run(context.Background(), os.Stdout, terminalnudge.Config{
		UserID: userID,
	}, scheduler, capture); err != nil {
		fmt.Fprintf(os.Stderr, "terminal nudge error: %v\n", err)
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
