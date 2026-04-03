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
	"github.com/p-n-ai/pai-bot/internal/apidocs"
	"github.com/p-n-ai/pai-bot/internal/auth"
	"github.com/p-n-ai/pai-bot/internal/chat"
	"github.com/p-n-ai/pai-bot/internal/curriculum"
	"github.com/p-n-ai/pai-bot/internal/group"
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

	groupStore := group.NewPostgresStore(db.Pool)

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
	challengeStore := agent.NewPostgresChallengeStore(db.Pool, store.TenantID())
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
		Challenges:           challengeStore,
		Groups:               groupStore,
		DevMode:              cfg.Features.DevMode,
	})

	// Create Telegram channel + chat gateway.
	tg, err := chat.NewTelegramChannel(cfg.Telegram.BotToken)
	if err != nil {
		slog.Error("failed to create Telegram channel", "error", err)
		os.Exit(1)
	}
	tg.SetDevMode(cfg.Features.DevMode)

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
		goalStore,
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

	authService := auth.NewPostgresService(
		db.Pool,
		cfg.Auth.JWTSecret,
		time.Duration(cfg.Auth.AccessTokenTTL)*time.Minute,
		time.Duration(cfg.Auth.RefreshTokenTTL)*24*time.Hour,
	)

	// HTTP endpoints.
	mux := newHandlerWithServicesAndAdminProvider(
		tenantAdminDataSourceProvider{
			newForTenant: func(tenantID string) adminDataSource {
				return adminapi.New(db.Pool, tenantID, groupStore)
			},
			newForPlatform: func() adminDataSource {
				return adminapi.NewPlatform(db.Pool, groupStore)
			},
		},
		gatewaySender{gw},
		authService,
		cfg.Auth.JWTSecret,
		time.Duration(cfg.Auth.AccessTokenTTL)*time.Minute,
	)
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
	GetParentSummary(parentID string) (adminapi.ParentSummary, error)
	GetAIUsage() (adminapi.AIUsageSummary, error)
	UpsertTenantTokenBudgetWindow(req adminapi.UpsertTokenBudgetWindowRequest) (adminapi.AIUsageSummary, error)
	GetMetrics() (adminapi.MetricsSummary, error)
	ListClasses(ctx context.Context) ([]adminapi.ClassListItem, error)
	CreateClass(ctx context.Context, name, syllabusID, createdByUserID string) (*adminapi.ClassListItem, error)
	GetClassDetail(ctx context.Context, classID string) (*adminapi.ClassDetail, error)
	UpdateClass(ctx context.Context, classID string, name *string, status *string) error
}

type adminDataSourceProvider interface {
	ForRequest(r *http.Request) (adminDataSource, error)
}

type fixedAdminDataSourceProvider struct {
	source adminDataSource
}

func (p fixedAdminDataSourceProvider) ForRequest(_ *http.Request) (adminDataSource, error) {
	return p.source, nil
}

type tenantAdminDataSourceProvider struct {
	newForTenant   func(tenantID string) adminDataSource
	newForPlatform func() adminDataSource
}

func (p tenantAdminDataSourceProvider) ForRequest(r *http.Request) (adminDataSource, error) {
	claims, ok := auth.ClaimsFromContext(r.Context())
	if !ok {
		return nil, errors.New("missing auth claims")
	}

	if claims.Role == auth.RolePlatformAdmin && p.newForPlatform != nil {
		return p.newForPlatform(), nil
	}
	if strings.TrimSpace(claims.TenantID) == "" {
		return nil, errors.New("missing auth claims")
	}

	return p.newForTenant(claims.TenantID), nil
}

type authService interface {
	Login(ctx context.Context, req auth.LoginRequest) (auth.TokenPair, error)
	AcceptInvite(ctx context.Context, req auth.AcceptInviteRequest) (auth.TokenPair, error)
	IssueInvite(ctx context.Context, req auth.IssueInviteRequest) (auth.InviteRecord, error)
	Refresh(ctx context.Context, refreshToken string) (auth.TokenPair, error)
	SwitchTenant(ctx context.Context, refreshToken, tenantID, password string) (auth.TokenPair, error)
	Logout(ctx context.Context, refreshToken string) error
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
	mux.HandleFunc("GET /openapi.json", handleOpenAPI)
	mux.HandleFunc("GET /docs", handleScalarDocs)
	return mux
}

func newHandler(admin adminDataSource, sender messageSender) http.Handler {
	return newHandlerWithServices(admin, sender, auth.NewNoopService(), "change-me-in-production", time.Hour)
}

func newHandlerWithServices(admin adminDataSource, sender messageSender, authSvc authService, jwtSecret string, accessTokenTTL time.Duration) http.Handler {
	return newHandlerWithServicesAndAdminProvider(fixedAdminDataSourceProvider{source: admin}, sender, authSvc, jwtSecret, accessTokenTTL)
}

func newHandlerWithServicesAndAdminProvider(adminProvider adminDataSourceProvider, sender messageSender, authSvc authService, jwtSecret string, accessTokenTTL time.Duration) http.Handler {
	mux := newMux(nil, sender)
	manager := auth.NewTokenManager(jwtSecret, accessTokenTTL)

	teacherOrAbove := chain(
		auth.Authenticate(manager, time.Now),
		auth.RequireRoles(auth.RoleTeacher, auth.RoleAdmin, auth.RolePlatformAdmin),
	)
	parentOrAbove := chain(
		auth.Authenticate(manager, time.Now),
		auth.RequireRoles(auth.RoleParent, auth.RoleAdmin, auth.RolePlatformAdmin),
	)
	adminOrAbove := chain(
		auth.Authenticate(manager, time.Now),
		auth.RequireRoles(auth.RoleAdmin, auth.RolePlatformAdmin),
	)
	adminOnly := chain(
		auth.Authenticate(manager, time.Now),
		auth.RequireRoles(auth.RoleAdmin),
	)

	mux.Handle("POST /api/auth/login", handleAuthLogin(authSvc))
	mux.Handle("POST /api/auth/invitations/accept", handleAuthAcceptInvite(authSvc))
	mux.Handle("POST /api/auth/refresh", handleAuthRefresh(authSvc))
	mux.Handle("POST /api/auth/switch-tenant", handleAuthSwitchTenant(authSvc))
	mux.Handle("POST /api/auth/logout", handleAuthLogout(authSvc))
	mux.Handle("POST /api/admin/invites", adminOrAbove(handleAdminInvite(authSvc)))
	mux.Handle("GET /api/admin/classes", teacherOrAbove(handleAdminListClasses(adminProvider)))
	mux.Handle("POST /api/admin/classes", teacherOrAbove(handleAdminCreateClass(adminProvider)))
	mux.Handle("GET /api/admin/classes/{id}", teacherOrAbove(handleAdminGetClassDetail(adminProvider)))
	mux.Handle("PATCH /api/admin/classes/{id}", teacherOrAbove(handleAdminUpdateClass(adminProvider)))
	mux.Handle("GET /api/admin/classes/{id}/progress", teacherOrAbove(handleAdminClassProgress(adminProvider)))
	mux.Handle("GET /api/admin/students/{id}", teacherOrAbove(handleAdminStudentDetail(adminProvider)))
	mux.Handle("GET /api/admin/students/{id}/conversations", teacherOrAbove(handleAdminStudentConversations(adminProvider)))
	mux.Handle("POST /api/admin/students/{id}/nudge", teacherOrAbove(handleAdminStudentNudge(adminProvider, sender)))
	mux.Handle("GET /api/admin/metrics", teacherOrAbove(handleAdminMetrics(adminProvider)))
	mux.Handle("GET /api/admin/ai/usage", teacherOrAbove(handleAdminAIUsage(adminProvider)))
	mux.Handle("POST /api/admin/ai/budget-window", adminOnly(handleAdminUpsertTokenBudgetWindow(adminProvider)))
	mux.Handle("GET /api/admin/parents/{id}", parentOrAbove(handleAdminParentSummary(adminProvider)))

	return withCORS(mux)
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

func handleOpenAPI(w http.ResponseWriter, r *http.Request) {
	data, err := apidocs.JSON()
	if err != nil {
		http.Error(w, "failed to build OpenAPI document", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(data)
}

func handleScalarDocs(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(apidocs.ScalarHTML("/openapi.json")))
}

func resolveAdminDataSource(w http.ResponseWriter, r *http.Request, provider adminDataSourceProvider) (adminDataSource, bool) {
	admin, err := provider.ForRequest(r)
	if err != nil {
		http.Error(w, "missing auth claims", http.StatusUnauthorized)
		return nil, false
	}

	return admin, true
}

func handleAdminClassProgress(adminProvider adminDataSourceProvider) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		admin, ok := resolveAdminDataSource(w, r, adminProvider)
		if !ok {
			return
		}

		payload, err := admin.GetClassProgress(r.PathValue("id"))
		if err != nil {
			writeAdminError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, payload)
	}
}

func handleAdminStudentDetail(adminProvider adminDataSourceProvider) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		admin, ok := resolveAdminDataSource(w, r, adminProvider)
		if !ok {
			return
		}

		payload, err := admin.GetStudentDetail(r.PathValue("id"))
		if err != nil {
			writeAdminError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, payload)
	}
}

func handleAdminStudentConversations(adminProvider adminDataSourceProvider) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		admin, ok := resolveAdminDataSource(w, r, adminProvider)
		if !ok {
			return
		}

		payload, err := admin.GetStudentConversations(r.PathValue("id"))
		if err != nil {
			writeAdminError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, payload)
	}
}

func handleAdminParentSummary(adminProvider adminDataSourceProvider) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		claims, ok := auth.ClaimsFromContext(r.Context())
		if !ok {
			http.Error(w, "missing auth claims", http.StatusUnauthorized)
			return
		}

		parentID := r.PathValue("id")
		if claims.Role == auth.RoleParent && claims.Subject != parentID {
			http.Error(w, "parents can only access their own summary", http.StatusForbidden)
			return
		}

		admin, ok := resolveAdminDataSource(w, r, adminProvider)
		if !ok {
			return
		}

		payload, err := admin.GetParentSummary(parentID)
		if err != nil {
			writeAdminError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, payload)
	}
}

func handleAdminAIUsage(adminProvider adminDataSourceProvider) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		admin, ok := resolveAdminDataSource(w, r, adminProvider)
		if !ok {
			return
		}

		payload, err := admin.GetAIUsage()
		if err != nil {
			writeAdminError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, payload)
	}
}

func handleAdminUpsertTokenBudgetWindow(adminProvider adminDataSourceProvider) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		admin, ok := resolveAdminDataSource(w, r, adminProvider)
		if !ok {
			return
		}

		var body adminapi.UpsertTokenBudgetWindowRequest
		if err := decodeJSONBody(r, &body); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		payload, err := admin.UpsertTenantTokenBudgetWindow(body)
		if err != nil {
			writeAdminError(w, err)
			return
		}

		writeJSON(w, http.StatusOK, payload)
	}
}

func handleAdminMetrics(adminProvider adminDataSourceProvider) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		admin, ok := resolveAdminDataSource(w, r, adminProvider)
		if !ok {
			return
		}

		payload, err := admin.GetMetrics()
		if err != nil {
			writeAdminError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, payload)
	}
}

func handleAdminStudentNudge(adminProvider adminDataSourceProvider, sender messageSender) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		admin, ok := resolveAdminDataSource(w, r, adminProvider)
		if !ok {
			return
		}

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

func handleAdminListClasses(adminProvider adminDataSourceProvider) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		admin, ok := resolveAdminDataSource(w, r, adminProvider)
		if !ok {
			return
		}

		payload, err := admin.ListClasses(r.Context())
		if err != nil {
			writeAdminError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, payload)
	}
}

func handleAdminCreateClass(adminProvider adminDataSourceProvider) http.HandlerFunc {
	type request struct {
		Name            string `json:"name"`
		SyllabusID      string `json:"syllabus_id"`
		CreatedByUserID string `json:"created_by_user_id"`
	}

	return func(w http.ResponseWriter, r *http.Request) {
		admin, ok := resolveAdminDataSource(w, r, adminProvider)
		if !ok {
			return
		}

		var body request
		if err := decodeJSONBody(r, &body); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		if strings.TrimSpace(body.Name) == "" {
			http.Error(w, "name is required", http.StatusBadRequest)
			return
		}

		// Default created_by_user_id to the authenticated user if not provided.
		if strings.TrimSpace(body.CreatedByUserID) == "" {
			if claims, ok := auth.ClaimsFromContext(r.Context()); ok {
				body.CreatedByUserID = claims.Subject
			}
		}

		payload, err := admin.CreateClass(r.Context(), body.Name, body.SyllabusID, body.CreatedByUserID)
		if err != nil {
			writeAdminError(w, err)
			return
		}
		writeJSON(w, http.StatusCreated, payload)
	}
}

func handleAdminGetClassDetail(adminProvider adminDataSourceProvider) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		admin, ok := resolveAdminDataSource(w, r, adminProvider)
		if !ok {
			return
		}

		payload, err := admin.GetClassDetail(r.Context(), r.PathValue("id"))
		if err != nil {
			writeAdminError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, payload)
	}
}

func handleAdminUpdateClass(adminProvider adminDataSourceProvider) http.HandlerFunc {
	type request struct {
		Name   *string `json:"name"`
		Status *string `json:"status"`
	}

	return func(w http.ResponseWriter, r *http.Request) {
		admin, ok := resolveAdminDataSource(w, r, adminProvider)
		if !ok {
			return
		}

		var body request
		if err := decodeJSONBody(r, &body); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		if body.Name == nil && body.Status == nil {
			http.Error(w, "at least one of name or status is required", http.StatusBadRequest)
			return
		}
		if body.Status != nil && *body.Status != "archived" {
			http.Error(w, "status must be 'archived'", http.StatusBadRequest)
			return
		}
		if body.Name != nil && strings.TrimSpace(*body.Name) == "" {
			http.Error(w, "name must not be empty", http.StatusBadRequest)
			return
		}

		if err := admin.UpdateClass(r.Context(), r.PathValue("id"), body.Name, body.Status); err != nil {
			writeAdminError(w, err)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}
}

func handleAdminInvite(authSvc authService) http.HandlerFunc {
	type request struct {
		Email string `json:"email"`
		Role  string `json:"role"`
	}

	return func(w http.ResponseWriter, r *http.Request) {
		var body request
		if err := decodeJSONBody(r, &body); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		if strings.TrimSpace(body.Email) == "" || strings.TrimSpace(body.Role) == "" {
			http.Error(w, "email and role are required", http.StatusBadRequest)
			return
		}

		claims, ok := auth.ClaimsFromContext(r.Context())
		if !ok {
			http.Error(w, "missing auth claims", http.StatusUnauthorized)
			return
		}

		resp, err := authSvc.IssueInvite(r.Context(), auth.IssueInviteRequest{
			InvitedByUserID: claims.Subject,
			TenantID:        claims.TenantID,
			Email:           body.Email,
			Role:            auth.Role(body.Role),
		})
		if err != nil {
			writeAuthError(w, err)
			return
		}

		writeJSON(w, http.StatusCreated, resp)
	}
}

func handleAuthLogin(authSvc authService) http.HandlerFunc {
	type request struct {
		TenantID string `json:"tenant_id"`
		Email    string `json:"email"`
		Password string `json:"password"`
	}

	return func(w http.ResponseWriter, r *http.Request) {
		var body request
		if err := decodeJSONBody(r, &body); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		if strings.TrimSpace(body.Email) == "" || strings.TrimSpace(body.Password) == "" {
			http.Error(w, "email and password are required", http.StatusBadRequest)
			return
		}

		resp, err := authSvc.Login(r.Context(), auth.LoginRequest{
			TenantID: body.TenantID,
			Email:    body.Email,
			Password: body.Password,
		})
		if err != nil {
			writeAuthError(w, err)
			return
		}

		writeJSON(w, http.StatusOK, resp)
	}
}

func handleAuthAcceptInvite(authSvc authService) http.HandlerFunc {
	type request struct {
		Token    string `json:"token"`
		Name     string `json:"name"`
		Password string `json:"password"`
	}

	return func(w http.ResponseWriter, r *http.Request) {
		var body request
		if err := decodeJSONBody(r, &body); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		if strings.TrimSpace(body.Token) == "" || strings.TrimSpace(body.Name) == "" || strings.TrimSpace(body.Password) == "" {
			http.Error(w, "token, name, and password are required", http.StatusBadRequest)
			return
		}

		resp, err := authSvc.AcceptInvite(r.Context(), auth.AcceptInviteRequest{
			Token:    body.Token,
			Name:     body.Name,
			Password: body.Password,
		})
		if err != nil {
			writeAuthError(w, err)
			return
		}

		writeJSON(w, http.StatusCreated, resp)
	}
}

func handleAuthRefresh(authSvc authService) http.HandlerFunc {
	type request struct {
		RefreshToken string `json:"refresh_token"`
	}

	return func(w http.ResponseWriter, r *http.Request) {
		var body request
		if err := decodeJSONBody(r, &body); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		if strings.TrimSpace(body.RefreshToken) == "" {
			http.Error(w, "refresh_token is required", http.StatusBadRequest)
			return
		}

		resp, err := authSvc.Refresh(r.Context(), body.RefreshToken)
		if err != nil {
			writeAuthError(w, err)
			return
		}

		writeJSON(w, http.StatusOK, resp)
	}
}

func handleAuthSwitchTenant(authSvc authService) http.HandlerFunc {
	type request struct {
		RefreshToken string `json:"refresh_token"`
		TenantID     string `json:"tenant_id"`
		Password     string `json:"password"`
	}

	return func(w http.ResponseWriter, r *http.Request) {
		var body request
		if err := decodeJSONBody(r, &body); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		if strings.TrimSpace(body.RefreshToken) == "" {
			http.Error(w, "refresh_token is required", http.StatusBadRequest)
			return
		}
		if strings.TrimSpace(body.TenantID) == "" {
			http.Error(w, "tenant_id is required", http.StatusBadRequest)
			return
		}
		if strings.TrimSpace(body.Password) == "" {
			http.Error(w, "password is required", http.StatusBadRequest)
			return
		}

		resp, err := authSvc.SwitchTenant(r.Context(), body.RefreshToken, body.TenantID, body.Password)
		if err != nil {
			writeAuthError(w, err)
			return
		}

		writeJSON(w, http.StatusOK, resp)
	}
}

func handleAuthLogout(authSvc authService) http.HandlerFunc {
	type request struct {
		RefreshToken string `json:"refresh_token"`
	}

	return func(w http.ResponseWriter, r *http.Request) {
		var body request
		if err := decodeJSONBody(r, &body); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		if strings.TrimSpace(body.RefreshToken) == "" {
			http.Error(w, "refresh_token is required", http.StatusBadRequest)
			return
		}

		if err := authSvc.Logout(r.Context(), body.RefreshToken); err != nil {
			writeAuthError(w, err)
			return
		}

		w.WriteHeader(http.StatusNoContent)
	}
}

func decodeJSONBody(r *http.Request, target any) (err error) {
	defer func() {
		closeErr := r.Body.Close()
		if err == nil && closeErr != nil {
			err = fmt.Errorf("close request body: %w", closeErr)
		}
	}()

	if err = json.NewDecoder(r.Body).Decode(target); err != nil {
		return fmt.Errorf("invalid json body")
	}
	return nil
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
	if errors.Is(err, adminapi.ErrInvalidArgument) {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
}

func writeAuthError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, auth.ErrInvalidCredentials), errors.Is(err, auth.ErrInvalidInvite), errors.Is(err, auth.ErrInviteExpired):
		http.Error(w, err.Error(), http.StatusUnauthorized)
	case errors.Is(err, auth.ErrInviteConflict):
		http.Error(w, err.Error(), http.StatusConflict)
	case errors.Is(err, auth.ErrTenantRequired):
		options, _ := auth.TenantRequiredOptions(err)
		writeJSON(w, http.StatusBadRequest, map[string]any{
			"error":   err.Error(),
			"tenants": options,
		})
	case errors.Is(err, auth.ErrNotImplemented):
		http.Error(w, err.Error(), http.StatusNotImplemented)
	default:
		http.Error(w, err.Error(), http.StatusBadRequest)
	}
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
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PATCH, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Authorization, Content-Type")
		}

		if r.Method == http.MethodOptions && strings.HasPrefix(r.URL.Path, "/api/") {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func chain(middlewares ...func(http.Handler) http.Handler) func(http.Handler) http.Handler {
	return func(final http.Handler) http.Handler {
		wrapped := final
		for i := len(middlewares) - 1; i >= 0; i-- {
			wrapped = middlewares[i](wrapped)
		}
		return wrapped
	}
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
