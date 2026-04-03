package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"slices"
	"strings"
	"syscall"
	"time"

	"github.com/p-n-ai/pai-bot/internal/adminapi"
	"github.com/p-n-ai/pai-bot/internal/agent"
	"github.com/p-n-ai/pai-bot/internal/ai"
	"github.com/p-n-ai/pai-bot/internal/auth"
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
		if cfg.Features.DevMode {
			slog.Warn("no AI providers configured; continuing in dev mode without AI-backed chat responses")
		} else {
			slog.Error("no AI providers configured")
			os.Exit(1)
		}
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
		DevMode:              cfg.Features.DevMode,
	})

	gw := chat.NewGateway()
	if strings.TrimSpace(cfg.Telegram.BotToken) != "" {
		tg, err := chat.NewTelegramChannel(cfg.Telegram.BotToken)
		if err != nil {
			slog.Error("failed to create Telegram channel", "error", err)
			os.Exit(1)
		}
		tg.SetDevMode(cfg.Features.DevMode)
		gw.Register("telegram", tg)
	} else {
		slog.Warn("telegram channel disabled; LEARN_TELEGRAM_BOT_TOKEN is not set")
	}

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
		defaultSessionTTL,
	)
	authService.ConfigureGoogleOAuth(auth.GoogleOAuthProviderConfig{
		ClientID:              cfg.Auth.Google.ClientID,
		ClientSecret:          cfg.Auth.Google.ClientSecret,
		DiscoveryURL:          cfg.Auth.Google.DiscoveryURL,
		Policy:                googleOAuthPolicy(cfg),
		EmulatorSigningSecret: cfg.Auth.Google.EmulatorSigningSecret,
	})

	// HTTP endpoints.
	mux := newHandlerWithServicesAndAdminProvider(
		tenantAdminDataSourceProvider{
			newForTenant: func(tenantID string) adminDataSource {
				return adminapi.New(db.Pool, tenantID)
			},
			newForPlatform: func() adminDataSource {
				return adminapi.NewPlatform(db.Pool)
			},
		},
		gatewaySender{gw},
		authService,
		cfg.Auth.JWTSecret,
		defaultAccessTokenTTL,
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

const (
	defaultAccessTokenTTL = 15 * time.Minute
	defaultSessionTTL     = 7 * 24 * time.Hour
)

type adminDataSource interface {
	GetClassProgress(classID string) (adminapi.ClassProgress, error)
	GetStudentDetail(studentID string) (adminapi.StudentDetail, error)
	GetStudentConversations(studentID string) ([]adminapi.StudentConversation, error)
	GetParentSummary(parentID string) (adminapi.ParentSummary, error)
	GetAIUsage() (adminapi.AIUsageSummary, error)
	UpsertTenantTokenBudgetWindow(req adminapi.UpsertTokenBudgetWindowRequest) (adminapi.AIUsageSummary, error)
	GetMetrics() (adminapi.MetricsSummary, error)
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
	Login(ctx context.Context, req auth.LoginRequest) (auth.Session, error)
	AcceptInvite(ctx context.Context, req auth.AcceptInviteRequest) (auth.Session, error)
	IssueInvite(ctx context.Context, req auth.IssueInviteRequest) (auth.InviteRecord, error)
	Session(ctx context.Context, sessionToken string) (auth.Session, error)
	SwitchTenant(ctx context.Context, sessionToken, tenantID, password string) (auth.Session, error)
	Logout(ctx context.Context, sessionToken string) error
	StartGoogleLogin(ctx context.Context, req auth.StartGoogleFlowRequest) (string, error)
	StartGoogleLink(ctx context.Context, req auth.StartGoogleFlowRequest) (string, error)
	CompleteGoogleCallback(ctx context.Context, req auth.GoogleCallbackRequest) (auth.GoogleCallbackResult, error)
	ListLinkedIdentities(ctx context.Context, userID string) ([]auth.LinkedIdentity, error)
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
	authenticated := authenticateRequests(authSvc, manager, time.Now)

	teacherOrAbove := chain(
		authenticated,
		auth.RequireRoles(auth.RoleTeacher, auth.RoleAdmin, auth.RolePlatformAdmin),
	)
	parentOrAbove := chain(
		authenticated,
		auth.RequireRoles(auth.RoleParent, auth.RoleAdmin, auth.RolePlatformAdmin),
	)
	adminOrAbove := chain(
		authenticated,
		auth.RequireRoles(auth.RoleAdmin, auth.RolePlatformAdmin),
	)
	adminOnly := chain(
		authenticated,
		auth.RequireRoles(auth.RoleAdmin),
	)

	mux.Handle("POST /api/auth/login", handleAuthLogin(authSvc))
	mux.Handle("GET /api/auth/google/start", handleAuthGoogleStart(authSvc))
	mux.Handle("GET /api/auth/google/callback", handleAuthGoogleCallback(authSvc))
	mux.Handle("POST /api/auth/google/link/start", authenticated(handleAuthGoogleLinkStart(authSvc)))
	mux.Handle("GET /api/auth/identities", authenticated(handleAuthIdentities(authSvc)))
	mux.Handle("POST /api/auth/invitations/accept", handleAuthAcceptInvite(authSvc))
	mux.Handle("GET /api/auth/session", handleAuthSession(authSvc))
	mux.Handle("POST /api/auth/switch-tenant", handleAuthSwitchTenant(authSvc))
	mux.Handle("POST /api/auth/logout", handleAuthLogout(authSvc))
	mux.Handle("POST /api/admin/invites", adminOrAbove(handleAdminInvite(authSvc)))
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

func authenticateRequests(authSvc authService, manager *auth.TokenManager, now func() time.Time) func(http.Handler) http.Handler {
	if now == nil {
		now = time.Now
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if token, err := bearerToken(r.Header.Get("Authorization")); err == nil {
				claims, err := manager.Parse(token, now().UTC())
				if err != nil {
					if errors.Is(err, auth.ErrExpiredToken) {
						http.Error(w, "expired token", http.StatusUnauthorized)
						return
					}
					http.Error(w, "invalid token", http.StatusUnauthorized)
					return
				}
				next.ServeHTTP(w, r.WithContext(auth.WithClaims(r.Context(), claims)))
				return
			}

			sessionToken := readCookieValue(r, auth.SessionCookieName)
			if sessionToken == "" {
				http.Error(w, "missing auth token", http.StatusUnauthorized)
				return
			}

			session, err := authSvc.Session(r.Context(), sessionToken)
			if err != nil {
				http.Error(w, "invalid session", http.StatusUnauthorized)
				return
			}

			next.ServeHTTP(w, r.WithContext(auth.WithClaims(r.Context(), auth.TokenClaims{
				Subject:  session.User.UserID,
				TenantID: session.User.TenantID,
				Role:     session.User.Role,
			})))
		})
	}
}

func bearerToken(header string) (string, error) {
	parts := strings.Fields(header)
	if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") || strings.TrimSpace(parts[1]) == "" {
		return "", errors.New("missing auth token")
	}
	return parts[1], nil
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

		writeAuthSessionResponse(w, r, http.StatusOK, resp)
	}
}

func handleAuthGoogleStart(authSvc authService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		target, err := authSvc.StartGoogleLogin(r.Context(), auth.StartGoogleFlowRequest{
			NextPath:    r.URL.Query().Get("next"),
			RedirectURL: googleCallbackURL(r),
		})
		if err != nil {
			writeAuthError(w, err)
			return
		}
		http.Redirect(w, r, target, http.StatusTemporaryRedirect)
	}
}

func handleAuthGoogleLinkStart(authSvc authService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !isAllowedBrowserOrigin(r.Header.Get("Origin")) {
			http.Error(w, "origin not allowed", http.StatusForbidden)
			return
		}

		claims, ok := auth.ClaimsFromContext(r.Context())
		if !ok {
			http.Error(w, "missing auth claims", http.StatusUnauthorized)
			return
		}

		target, err := authSvc.StartGoogleLink(r.Context(), auth.StartGoogleFlowRequest{
			UserID:      claims.Subject,
			NextPath:    r.URL.Query().Get("next"),
			RedirectURL: googleCallbackURL(r),
		})
		if err != nil {
			writeAuthError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, map[string]string{"url": target})
	}
}

func handleAuthGoogleCallback(authSvc authService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		result, err := authSvc.CompleteGoogleCallback(r.Context(), auth.GoogleCallbackRequest{
			State:       r.URL.Query().Get("state"),
			Code:        r.URL.Query().Get("code"),
			RedirectURL: googleCallbackURL(r),
		})
		if err != nil {
			target := result.RedirectPath
			if strings.TrimSpace(target) == "" {
				http.Error(w, "auth redirect target missing", http.StatusBadGateway)
				return
			}
			http.Redirect(w, r, addQueryValue(target, "auth_error", auth.GoogleCallbackErrorCode(err)), http.StatusSeeOther)
			return
		}
		if result.Session != nil {
			setSessionCookies(w, r, *result.Session)
		}
		target := addQueryValue(resolvePostAuthRedirect(result.RedirectPath, result.Session), "auth_provider", "google")
		if result.Linked {
			target = addQueryValue(target, "identity_linked", "google")
		}
		http.Redirect(w, r, target, http.StatusSeeOther)
	}
}

func handleAuthIdentities(authSvc authService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		claims, ok := auth.ClaimsFromContext(r.Context())
		if !ok {
			http.Error(w, "missing auth claims", http.StatusUnauthorized)
			return
		}

		identities, err := authSvc.ListLinkedIdentities(r.Context(), claims.Subject)
		if err != nil {
			writeAuthError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"identities": identities})
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

		writeAuthSessionResponse(w, r, http.StatusCreated, resp)
	}
}

func handleAuthSession(authSvc authService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		sessionToken := readCookieValue(r, auth.SessionCookieName)
		if sessionToken == "" {
			http.Error(w, "missing session", http.StatusUnauthorized)
			return
		}

		session, err := authSvc.Session(r.Context(), sessionToken)
		if err != nil {
			writeAuthError(w, err)
			return
		}

		http.SetCookie(w, buildSessionCookie(auth.SessionCookieName, session.Token, session.ExpiresAt, time.Now().UTC(), requestUsesHTTPS(r)))
		writeJSON(w, http.StatusOK, authSessionResponse{
			ExpiresAt: session.ExpiresAt,
			User:      session.User,
		})
	}
}

func handleAuthSwitchTenant(authSvc authService) http.HandlerFunc {
	type request struct {
		SessionToken string `json:"session_token"`
		TenantID     string `json:"tenant_id"`
		Password     string `json:"password"`
	}

	return func(w http.ResponseWriter, r *http.Request) {
		var body request
		if err := decodeOptionalJSONBody(r, &body); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		sessionToken := strings.TrimSpace(body.SessionToken)
		if sessionToken == "" {
			sessionToken = readCookieValue(r, auth.SessionCookieName)
		}
		if sessionToken == "" {
			http.Error(w, "session_token is required", http.StatusBadRequest)
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

		resp, err := authSvc.SwitchTenant(r.Context(), sessionToken, body.TenantID, body.Password)
		if err != nil {
			writeAuthError(w, err)
			return
		}

		writeAuthSessionResponse(w, r, http.StatusOK, resp)
	}
}

func handleAuthLogout(authSvc authService) http.HandlerFunc {
	type request struct {
		SessionToken string `json:"session_token"`
	}

	return func(w http.ResponseWriter, r *http.Request) {
		var body request
		if err := decodeOptionalJSONBody(r, &body); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		sessionToken := strings.TrimSpace(body.SessionToken)
		if sessionToken == "" {
			sessionToken = readCookieValue(r, auth.SessionCookieName)
		}
		if sessionToken == "" {
			clearSessionCookies(w, r)
			w.WriteHeader(http.StatusNoContent)
			return
		}

		if err := authSvc.Logout(r.Context(), sessionToken); err != nil {
			if !errors.Is(err, auth.ErrInvalidCredentials) {
				writeAuthError(w, err)
				return
			}
			clearSessionCookies(w, r)
			w.WriteHeader(http.StatusNoContent)
			return
		}

		clearSessionCookies(w, r)
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

func decodeOptionalJSONBody(r *http.Request, target any) (err error) {
	if r.Body == nil || r.ContentLength == 0 {
		return nil
	}

	defer func() {
		closeErr := r.Body.Close()
		if err == nil && closeErr != nil {
			err = fmt.Errorf("close request body: %w", closeErr)
		}
	}()

	if err = json.NewDecoder(r.Body).Decode(target); err != nil {
		if errors.Is(err, io.EOF) {
			return nil
		}
		return fmt.Errorf("invalid json body")
	}
	return nil
}

type authSessionResponse struct {
	ExpiresAt time.Time        `json:"expires_at"`
	User      auth.UserSession `json:"user"`
}

func writeAuthSessionResponse(w http.ResponseWriter, r *http.Request, status int, session auth.Session) {
	setSessionCookies(w, r, session)
	writeJSON(w, status, authSessionResponse{
		ExpiresAt: session.ExpiresAt,
		User:      session.User,
	})
}

func setSessionCookies(w http.ResponseWriter, r *http.Request, session auth.Session) {
	now := time.Now().UTC()
	secure := requestUsesHTTPS(r)
	http.SetCookie(w, buildSessionCookie(auth.SessionCookieName, session.Token, session.ExpiresAt, now, secure))
}

func clearSessionCookies(w http.ResponseWriter, r *http.Request) {
	now := time.Now().UTC()
	secure := requestUsesHTTPS(r)

	for _, name := range []string{auth.SessionCookieName} {
		http.SetCookie(w, buildExpiredCookie(name, now, secure))
	}
}

func buildSessionCookie(name, value string, expiresAt, now time.Time, secure bool) *http.Cookie {
	maxAge := int(expiresAt.Sub(now).Seconds())
	if maxAge < 0 {
		maxAge = 0
	}

	return &http.Cookie{
		Name:     name,
		Value:    value,
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Secure:   secure,
		Expires:  expiresAt,
		MaxAge:   maxAge,
	}
}

func buildExpiredCookie(name string, now time.Time, secure bool) *http.Cookie {
	return &http.Cookie{
		Name:     name,
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Secure:   secure,
		Expires:  now.Add(-time.Hour),
		MaxAge:   -1,
	}
}

func requestUsesHTTPS(r *http.Request) bool {
	if r == nil {
		return false
	}
	if r.TLS != nil {
		return true
	}
	return strings.EqualFold(r.Header.Get("X-Forwarded-Proto"), "https")
}

func requestBaseURL(r *http.Request) string {
	if r == nil {
		return ""
	}
	scheme := "http"
	if requestUsesHTTPS(r) {
		scheme = "https"
	}
	host := strings.TrimSpace(r.Header.Get("X-Forwarded-Host"))
	if host == "" {
		host = strings.TrimSpace(r.Host)
	}
	if host == "" {
		return ""
	}
	return scheme + "://" + host
}

func googleCallbackURL(r *http.Request) string {
	base := requestBaseURL(r)
	if base == "" {
		return ""
	}
	return base + "/api/auth/google/callback"
}

func googleOAuthPolicy(cfg *config.Config) auth.GoogleOAuthPolicy {
	if cfg == nil {
		return auth.GoogleOAuthPolicy{}
	}
	return auth.AllowGoogleHostedDomains(cfg.Auth.Google.AllowedDomain)
}

func readCookieValue(r *http.Request, name string) string {
	cookie, err := r.Cookie(name)
	if err != nil {
		return ""
	}
	return strings.TrimSpace(cookie.Value)
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

func addQueryValue(rawURL, key, value string) string {
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return rawURL
	}
	query := parsed.Query()
	query.Set(key, value)
	parsed.RawQuery = query.Encode()
	return parsed.String()
}

func resolvePostAuthRedirect(rawURL string, session *auth.Session) string {
	if session == nil {
		return rawURL
	}

	parsed, err := url.Parse(strings.TrimSpace(rawURL))
	if err != nil {
		return rawURL
	}
	if parsed.Path != "" && parsed.Path != "/" && parsed.Path != "/login" {
		return rawURL
	}

	defaultPath := defaultPostAuthPath(session.User)
	if parsed.IsAbs() {
		parsed.Path = defaultPath
		parsed.RawPath = ""
		parsed.RawQuery = ""
		parsed.Fragment = ""
		return parsed.String()
	}
	return defaultPath
}

func defaultPostAuthPath(user auth.UserSession) string {
	if user.Role == auth.RoleParent && strings.TrimSpace(user.UserID) != "" {
		return "/parents/" + user.UserID
	}
	return "/dashboard"
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
	case errors.Is(err, auth.ErrProviderNotConfigured):
		http.Error(w, err.Error(), http.StatusNotImplemented)
	case errors.Is(err, auth.ErrIdentityAlreadyLinked):
		http.Error(w, err.Error(), http.StatusConflict)
	case errors.Is(err, auth.ErrGoogleDomainNotAllowed):
		http.Error(w, err.Error(), http.StatusForbidden)
	case errors.Is(err, auth.ErrIdentityLinkRequired), errors.Is(err, auth.ErrAuthFlowInvalid):
		http.Error(w, err.Error(), http.StatusBadRequest)
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
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, "/api/") {
			setPrivateNoStoreHeaders(w)
		}

		origin := r.Header.Get("Origin")
		if isAllowedBrowserOrigin(origin) {
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Add("Vary", "Origin")
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Authorization, Content-Type")
			w.Header().Set("Access-Control-Allow-Credentials", "true")
		}

		if r.Method == http.MethodOptions && strings.HasPrefix(r.URL.Path, "/api/") {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func isAllowedBrowserOrigin(origin string) bool {
	return slices.Contains([]string{
		"http://localhost:3000",
		"http://127.0.0.1:3000",
	}, origin)
}

func setPrivateNoStoreHeaders(w http.ResponseWriter) {
	w.Header().Set("Cache-Control", "private, no-store, max-age=0")
	w.Header().Set("Pragma", "no-cache")
	w.Header().Set("Expires", "0")
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
