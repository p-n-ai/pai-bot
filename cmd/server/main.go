package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"slices"
	"strings"
	"syscall"
	"time"

	"github.com/p-n-ai/pai-bot/internal/adminapi"
	"github.com/p-n-ai/pai-bot/internal/agent"
	"github.com/p-n-ai/pai-bot/internal/ai"
	"github.com/p-n-ai/pai-bot/internal/chat"
	"github.com/p-n-ai/pai-bot/internal/curriculum"
	"github.com/p-n-ai/pai-bot/internal/platform/cache"
	"github.com/p-n-ai/pai-bot/internal/platform/config"
	"github.com/p-n-ai/pai-bot/internal/platform/database"
	"github.com/p-n-ai/pai-bot/internal/progress"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	slog.SetDefault(logger)

	cfg, err := config.Load()
	if err != nil {
		slog.Error("failed to load config", "error", err)
		os.Exit(1)
	}

	if err := cfg.Validate(); err != nil {
		slog.Error("invalid config", "error", err)
		os.Exit(1)
	}

	// Initialize AI router with configured providers.
	router := setupAIRouter(cfg)
	if !router.HasProvider() {
		slog.Error("no AI providers configured")
		os.Exit(1)
	}

	// Initialize PostgreSQL-backed conversation store.
	db, err := database.New(context.Background(), cfg.Database.URL, cfg.Database.MaxConns, cfg.Database.MinConns)
	if err != nil {
		slog.Error("failed to connect to database", "error", err)
		os.Exit(1)
	}
	defer db.Close()

	// Initialize cache (warn if unavailable, don't fail).
	if cfg.Cache.URL != "" {
		c, err := cache.New(context.Background(), cfg.Cache.URL)
		if err != nil {
			slog.Warn("cache not connected", "error", err)
		} else {
			defer func() { _ = c.Close() }()
			slog.Info("cache connected")
		}
	} else {
		slog.Warn("cache not configured, running without cache")
	}

	store, err := agent.NewPostgresStore(context.Background(), db.Pool)
	if err != nil {
		slog.Error("failed to initialize conversation store", "error", err)
		os.Exit(1)
	}

	// Load curriculum (warn if unavailable, don't fail).
	loader, err := curriculum.NewLoader(cfg.CurriculumPath)
	if err != nil {
		slog.Warn("curriculum not loaded", "error", err, "path", cfg.CurriculumPath)
	} else {
		topics := loader.AllTopics()
		slog.Info("curriculum ready", "topics", len(topics))
	}

	// Create agent engine with streaks and XP tracking.
	eventLogger := agent.NewPostgresEventLogger(db.Pool)
	tracker := progress.NewPostgresTracker(db.Pool, store.TenantID())
	streakTracker := progress.NewMemoryStreakTracker()
	xpTracker := progress.NewMemoryXPTracker()
	goalStore := agent.NewPostgresGoalStore(db.Pool, store.TenantID())
	engine := agent.NewEngine(agent.EngineConfig{
		AIRouter:             router,
		Store:                store,
		EventLogger:          eventLogger,
		CurriculumLoader:     loader,
		DisableMultiLanguage: cfg.Features.DisableMultiLanguage,
		RatingPromptEvery:    cfg.Features.RatingPromptEvery,
		Tracker:              tracker,
		Streaks:              streakTracker,
		XP:                   xpTracker,
		Goals:                goalStore,
	})

	// Create Telegram channel + chat gateway.
	tg, err := chat.NewTelegramChannel(cfg.Telegram.BotToken)
	if err != nil {
		slog.Error("failed to create Telegram channel", "error", err)
		os.Exit(1)
	}

	gw := chat.NewGateway()
	gw.Register("telegram", tg)

	// Start proactive scheduler (nudges for due reviews).
	nudgeTracker := agent.NewPostgresNudgeTracker(db.Pool, store.TenantID())
	scheduler := agent.NewScheduler(
		agent.SchedulerConfig{
			CheckInterval:               agent.DefaultSchedulerConfig().CheckInterval,
			MaxNudgesPerDay:             agent.DefaultSchedulerConfig().MaxNudgesPerDay,
			AIPersonalizedNudgesEnabled: cfg.Features.AIPersonalizedNudgesEnabled,
		},
		tracker,
		streakTracker,
		xpTracker,
		nudgeTracker,
		gw,
		router,
		store,
	)

	// Graceful shutdown on SIGTERM/SIGINT.
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGTERM, syscall.SIGINT)
	defer stop()

	// Scheduler runs in background; user list is empty initially — will be populated
	// when we add user enumeration from the database.
	go scheduler.Start(ctx, []string{})

	// Start long-polling with message handler.
	err = gw.StartAll(ctx, func(msg chat.InboundMessage) {
		// Show typing indicator while processing.
		if err := gw.SendTyping(ctx, msg.Channel, msg.UserID); err != nil {
			slog.Warn("failed to send typing indicator", "error", err)
		}

		resp, err := engine.ProcessMessage(ctx, msg)
		if err != nil {
			slog.Error("ProcessMessage failed", "error", err, "user_id", msg.UserID)
			return
		}

		out := chat.OutboundMessage{
			Channel: msg.Channel,
			UserID:  msg.UserID,
			Text:    resp,
		}
		if msg.Channel == "telegram" {
			out.Text = chat.NormalizeTelegramMarkdown(resp)
			out.ParseMode = "Markdown"
			out.ReplyKeyboard = chat.BuildTelegramReplyKeyboard(resp)
			out.InlineKeyboard = chat.BuildTelegramInlineKeyboardWithContext(resp, telegramInlineKeyboardContext(store, msg.UserID))
			out.Text = chat.StripReviewActionCodes(out.Text)
		}
		if strings.TrimSpace(out.Text) == "" {
			return
		}

		if err := gw.Send(ctx, out); err != nil {
			slog.Error("failed to send response", "error", err, "user_id", msg.UserID)
		}
	})
	if err != nil {
		slog.Error("failed to start channels", "error", err)
		os.Exit(1)
	}

	adminService := adminapi.New(db.Pool, store.TenantID())

	// HTTP endpoints.
	mux := newHandler(adminService, gatewaySender{gw})
	addr := fmt.Sprintf(":%d", cfg.Server.Port)
	srv := &http.Server{
		Addr:         addr,
		Handler:      mux,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	go func() {
		slog.Info("server starting", "addr", srv.Addr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("server error", "error", err)
			os.Exit(1)
		}
	}()

	slog.Info("P&AI Bot is running")

	<-ctx.Done()
	slog.Info("shutting down")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		slog.Error("shutdown error", "error", err)
	}
}

type adminDataSource interface {
	GetClassProgress(classID string) (adminapi.ClassProgress, error)
	GetStudentDetail(studentID string) (adminapi.StudentDetail, error)
	GetStudentConversations(studentID string) ([]adminapi.StudentConversation, error)
}

type outboundMessage struct {
	Channel string `json:"channel"`
	UserID  string `json:"user_id"`
	Text    string `json:"text"`
}

type messageSender interface {
	Send(ctx context.Context, msg outboundMessage) error
}

type gatewaySender struct {
	gw *chat.Gateway
}

func (g gatewaySender) Send(ctx context.Context, msg outboundMessage) error {
	return g.gw.Send(ctx, chat.OutboundMessage{
		Channel: msg.Channel,
		UserID:  msg.UserID,
		Text:    msg.Text,
	})
}

func telegramInlineKeyboardContext(store agent.ConversationStore, userID string) chat.TelegramInlineKeyboardContext {
	conv, found := store.GetActiveConversation(userID)
	if !found || conv == nil {
		return chat.TelegramInlineKeyboardContext{}
	}

	ctx := chat.TelegramInlineKeyboardContext{
		QuizIntensityPending: conv.State == "quiz_intensity",
		QuizActive:           conv.State == "quiz_active",
	}
	if conv.QuizState != nil && conv.QuizState.RunState == "paused" {
		ctx.QuizPaused = true
		ctx.QuizActive = false
	}
	return ctx
}

func setupAIRouter(cfg *config.Config) *ai.Router {
	router := ai.NewRouter()

	if cfg.AI.OpenAI.APIKey != "" {
		router.Register("openai", ai.NewOpenAIProvider(cfg.AI.OpenAI.APIKey))
		slog.Info("AI provider registered", "provider", "openai")
	}

	if cfg.AI.Anthropic.APIKey != "" {
		provider, err := ai.NewAnthropicProvider(cfg.AI.Anthropic.APIKey)
		if err != nil {
			slog.Warn("failed to create Anthropic provider", "error", err)
		} else {
			router.Register("anthropic", provider)
			slog.Info("AI provider registered", "provider", "anthropic")
		}
	}

	if cfg.AI.DeepSeek.APIKey != "" {
		router.Register("deepseek", ai.NewDeepSeekProvider(cfg.AI.DeepSeek.APIKey))
		slog.Info("AI provider registered", "provider", "deepseek")
	}

	if cfg.AI.Google.APIKey != "" {
		router.Register("google", ai.NewGoogleProvider(cfg.AI.Google.APIKey))
		slog.Info("AI provider registered", "provider", "google")
	}

	if cfg.AI.Ollama.Enabled {
		router.Register("ollama", ai.NewOllamaProvider(cfg.AI.Ollama.URL))
		slog.Info("AI provider registered", "provider", "ollama")
	}

	if cfg.AI.OpenRouter.APIKey != "" {
		router.Register("openrouter", ai.NewOpenRouterProvider(cfg.AI.OpenRouter.APIKey))
		slog.Info("AI provider registered", "provider", "openrouter")
	}

	return router
}

// newMux creates the HTTP router with health check and admin endpoints.
func newMux(admin adminDataSource, sender messageSender) *http.ServeMux {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", handleHealthz)
	mux.HandleFunc("GET /readyz", handleReadyz)
	mux.HandleFunc("GET /api/admin/classes/{id}/progress", handleAdminClassProgress(admin))
	mux.HandleFunc("GET /api/admin/students/{id}", handleAdminStudentDetail(admin))
	mux.HandleFunc("GET /api/admin/students/{id}/conversations", handleAdminStudentConversations(admin))
	mux.HandleFunc("POST /api/admin/students/{id}/nudge", handleAdminStudentNudge(admin, sender))
	return mux
}

func newHandler(admin adminDataSource, sender messageSender) http.Handler {
	return withCORS(newMux(admin, sender))
}

func handleHealthz(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(`{"status":"ok"}`))
}

func handleReadyz(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(`{"status":"ready"}`))
}

func handleAdminClassProgress(admin adminDataSource) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		payload, err := admin.GetClassProgress(r.PathValue("id"))
		if err != nil {
			writeAdminError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, payload)
	}
}

func handleAdminStudentDetail(admin adminDataSource) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		payload, err := admin.GetStudentDetail(r.PathValue("id"))
		if err != nil {
			writeAdminError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, payload)
	}
}

func handleAdminStudentConversations(admin adminDataSource) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		payload, err := admin.GetStudentConversations(r.PathValue("id"))
		if err != nil {
			writeAdminError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, payload)
	}
}

func handleAdminStudentNudge(admin adminDataSource, sender messageSender) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		detail, err := admin.GetStudentDetail(r.PathValue("id"))
		if err != nil {
			writeAdminError(w, err)
			return
		}

		if detail.Student.Channel != "telegram" {
			http.Error(w, "manual nudge is only supported for telegram students", http.StatusBadRequest)
			return
		}
		if !isTelegramChatID(detail.Student.ExternalID) {
			http.Error(w, "student does not have a real Telegram chat ID yet", http.StatusBadRequest)
			return
		}

		msg := outboundMessage{
			Channel: "telegram",
			UserID:  detail.Student.ExternalID,
			Text:    buildManualNudgeMessage(detail),
		}
		if err := sender.Send(r.Context(), msg); err != nil {
			http.Error(w, http.StatusText(http.StatusBadGateway), http.StatusBadGateway)
			return
		}

		writeJSON(w, http.StatusAccepted, map[string]any{
			"status":  "queued",
			"student": detail.Student.ID,
			"channel": msg.Channel,
		})
	}
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

func writeAdminError(w http.ResponseWriter, err error) {
	if errors.Is(err, adminapi.ErrNotFound) {
		http.Error(w, http.StatusText(http.StatusNotFound), http.StatusNotFound)
		return
	}
	http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
}

func withCORS(next http.Handler) http.Handler {
	allowedOrigins := []string{
		"http://localhost:3000",
		"http://127.0.0.1:3000",
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")
		if slices.Contains(allowedOrigins, origin) {
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Set("Vary", "Origin")
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Authorization, Content-Type")
		}

		if r.Method == http.MethodOptions && strings.HasPrefix(r.URL.Path, "/api/") {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func isTelegramChatID(v string) bool {
	if v == "" {
		return false
	}
	for i, r := range v {
		if i == 0 && r == '-' {
			continue
		}
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}

func buildManualNudgeMessage(detail adminapi.StudentDetail) string {
	if len(detail.Progress) == 0 {
		return fmt.Sprintf("Hi %s, it is a good time to do a short math check-in today. Open the bot and reply with a question to continue.", detail.Student.Name)
	}

	weakest := detail.Progress[0]
	for _, item := range detail.Progress[1:] {
		if item.MasteryScore < weakest.MasteryScore {
			weakest = item
		}
	}

	return fmt.Sprintf(
		"Hi %s, let's revisit %s today. Your current mastery is %d%%. Reply to the bot and we can work through one quick practice step together.",
		detail.Student.Name,
		strings.ReplaceAll(weakest.TopicID, "-", " "),
		int(weakest.MasteryScore*100),
	)
}
