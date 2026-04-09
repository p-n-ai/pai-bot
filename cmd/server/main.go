package main

import (
	"context"
	"encoding/csv"
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
	"github.com/p-n-ai/pai-bot/internal/apidocs"
	"github.com/p-n-ai/pai-bot/internal/auth"
	"github.com/p-n-ai/pai-bot/internal/chat"
	"github.com/p-n-ai/pai-bot/internal/curriculum"
	"github.com/p-n-ai/pai-bot/internal/platform/cache"
	"github.com/p-n-ai/pai-bot/internal/platform/config"
	"github.com/p-n-ai/pai-bot/internal/platform/database"
	"github.com/p-n-ai/pai-bot/internal/platform/mailer"
	"github.com/p-n-ai/pai-bot/internal/progress"
	"github.com/p-n-ai/pai-bot/internal/retrieval"

	"github.com/jackc/pgx/v5/pgxpool"
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
	retrievalService := newBootstrapRetrievalService(loader)

	// Create agent engine with streaks and XP tracking.
	eventLogger := agent.NewPostgresEventLogger(db.Pool)
	tracker := progress.NewPostgresTracker(db.Pool, store.TenantID())
	streakTracker := progress.NewMemoryStreakTracker()
	xpTracker := progress.NewMemoryXPTracker()
	goalStore := agent.NewPostgresGoalStore(db.Pool, store.TenantID())
	challengeStore := agent.NewPostgresChallengeStore(db.Pool, store.TenantID())
	groupStore := agent.NewPostgresGroupStore(db.Pool)
	engine := agent.NewEngine(agent.EngineConfig{
		AIRouter:             router,
		Store:                store,
		EventLogger:          eventLogger,
		CurriculumLoader:     loader,
		RetrievalService:     retrievalService,
		DisableMultiLanguage: cfg.Features.DisableMultiLanguage,
		RatingPromptEvery:    cfg.Features.RatingPromptEvery,
		Tracker:              tracker,
		Streaks:              streakTracker,
		XP:                   xpTracker,
		Goals:                goalStore,
		Challenges:           challengeStore,
		Groups:               groupStore,
		TenantID:             store.TenantID(),
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

	// WebSocket channel (always enabled — used by terminal-chat and future web clients).
	wsChannel := chat.NewWSChannel()
	gw.Register("websocket", wsChannel)

	// Wire challenge notifications through the gateway.
	engine.SetNotifier(gatewayNotifier{gw: gw, pool: db.Pool})

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
	scheduler.SetWeeklyParentReportSource(weeklyParentReportSource{admin: adminapi.New(db.Pool, store.TenantID())})

	scheduler.SetGroupStore(groupStore, store.TenantID())

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
			out.Text = chat.ConvertLaTeXToUnicode(resp)
			out.Text = chat.NormalizeTelegramMarkdown(out.Text)
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
	if strings.TrimSpace(cfg.Email.SMTPAddr) != "" && strings.TrimSpace(cfg.Email.FromAddress) != "" {
		inviteMailer, err := mailer.NewSMTPSender(mailer.SMTPConfig{
			Addr:        cfg.Email.SMTPAddr,
			Username:    cfg.Email.SMTPUsername,
			Password:    cfg.Email.SMTPPassword,
			FromAddress: cfg.Email.FromAddress,
			FromName:    cfg.Email.FromName,
		})
		if err != nil {
			slog.Error("failed to create invite mailer", "error", err)
			os.Exit(1)
		}
		authService.ConfigureInviteEmail(inviteMailer)
	}

	// HTTP endpoints.
	apiHandler := newHandlerWithAdminProvider(
		tenantAdminDataSourceProvider{
			newForTenant: func(tenantID string) adminDataSource {
				return adminapi.New(db.Pool, tenantID)
			},
			newForPlatform: func() adminDataSource {
				return adminapi.NewPlatform(db.Pool)
			},
		},
		gatewaySender{gw},
		retrievalService,
		authService,
		cfg.Auth.JWTSecret,
		defaultAccessTokenTTL,
		cfg.Email.BaseURL,
	)

	// Top-level mux adds the WebSocket upgrade route alongside the API handler.
	topMux := http.NewServeMux()
	topMux.Handle("GET /ws/chat", wsChannel.Handler())
	topMux.Handle("/", apiHandler)

	addr := fmt.Sprintf(":%d", cfg.Server.Port)
	srv := &http.Server{
		Addr:         addr,
		Handler:      topMux,
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
	GetUserManagement() (adminapi.UserManagementView, error)
	ExportStudents() ([]adminapi.StudentExportRow, error)
	ExportConversations() ([]adminapi.ConversationExportRecord, error)
	ExportProgress() ([]adminapi.ProgressExportRow, error)
	ListGroups(groupType string) ([]adminapi.AdminGroup, error)
	CreateGroup(input adminapi.CreateGroupInput, createdByUserID string) (adminapi.AdminGroup, error)
	GetGroupDetail(id string) (adminapi.AdminGroupDetail, error)
	UpdateGroup(id string, input adminapi.AdminUpdateGroupInput) (adminapi.AdminGroup, error)
	DeleteGroup(id string) error
	AddGroupMember(groupID, userID, role string) error
	RemoveGroupMember(groupID, userID string) error
	GetGroupLeaderboard(id string) ([]adminapi.AdminLeaderboardEntry, error)
}

type adminDataSourceProvider interface {
	ForRequest(r *http.Request) (adminDataSource, error)
}

type weeklyParentReportSource struct {
	admin *adminapi.Service
}

func (s weeklyParentReportSource) ListWeeklyParentReportSummaries(ctx context.Context) ([]agent.WeeklyParentReportSummary, error) {
	items, err := s.admin.ListWeeklyParentReportSummaries(ctx)
	if err != nil {
		return nil, err
	}

	out := make([]agent.WeeklyParentReportSummary, 0, len(items))
	for _, item := range items {
		out = append(out, agent.WeeklyParentReportSummary{
			ParentExternalID:   item.ParentExternalID,
			ParentChannel:      item.ParentChannel,
			ParentName:         item.ParentName,
			ChildName:          item.ChildName,
			ChildForm:          item.ChildForm,
			CurrentStreak:      item.CurrentStreak,
			TotalXP:            item.TotalXP,
			NeedsReviewCount:   item.NeedsReviewCount,
			WeakestTopicID:     item.WeakestTopicID,
			EncouragementTitle: item.EncouragementTitle,
			EncouragementText:  item.EncouragementText,
			WeeklyStats: agent.WeeklyParentWeeklyStats{
				DaysActive:        item.WeeklyStats.DaysActive,
				MessagesExchanged: item.WeeklyStats.MessagesExchanged,
				QuizzesCompleted:  item.WeeklyStats.QuizzesCompleted,
				NeedsReviewCount:  item.WeeklyStats.NeedsReviewCount,
			},
		})
	}

	return out, nil
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
	ReissueInvite(ctx context.Context, req auth.ReissueInviteRequest) (auth.InviteRecord, error)
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

// gatewayNotifier implements agent.Notifier by looking up the user's channel
// from the database and sending to the correct one.
type gatewayNotifier struct {
	gw   *chat.Gateway
	pool *pgxpool.Pool
}

func (g gatewayNotifier) Notify(ctx context.Context, _, userID, text string) {
	// Look up which channel this user is on.
	var channel string
	err := g.pool.QueryRow(ctx,
		`SELECT channel FROM users WHERE external_id = $1 LIMIT 1`,
		userID,
	).Scan(&channel)
	if err != nil {
		// User not found — try all channels as fallback.
		slog.Warn("notifier: user channel lookup failed, trying all channels", "user_id", userID, "error", err)
		for _, ch := range g.gw.ChannelNames() {
			_ = g.gw.Send(ctx, chat.OutboundMessage{Channel: ch, UserID: userID, Text: text})
		}
		return
	}

	if err := g.gw.Send(ctx, chat.OutboundMessage{Channel: channel, UserID: userID, Text: text}); err != nil {
		slog.Warn("notifier: failed to send", "channel", channel, "user_id", userID, "error", err)
	}
}

func telegramInlineKeyboardContext(store agent.ConversationStore, userID string) chat.TelegramInlineKeyboardContext {
	conv, found := store.GetActiveConversation(userID)
	if !found || conv == nil {
		return chat.TelegramInlineKeyboardContext{}
	}

	ctx := chat.TelegramInlineKeyboardContext{
		QuizIntensityPending: conv.State == "quiz_intensity",
		QuizActive:           conv.State == "quiz_active",
		ChallengeActive:      conv.State == "challenge_active",
		ChallengeReview:      conv.State == "challenge_review",
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
	return newHandlerWithRetrievalService(admin, sender, retrieval.NewMemoryService(), authSvc, jwtSecret, accessTokenTTL)
}

func newHandlerWithRetrievalService(admin adminDataSource, sender messageSender, retrievalService *retrieval.Service, authSvc authService, jwtSecret string, accessTokenTTL time.Duration) http.Handler {
	return newHandlerWithAdminProvider(fixedAdminDataSourceProvider{source: admin}, sender, retrievalService, authSvc, jwtSecret, accessTokenTTL, "")
}

func newHandlerWithAdminProvider(adminProvider adminDataSourceProvider, sender messageSender, retrievalService *retrieval.Service, authSvc authService, jwtSecret string, accessTokenTTL time.Duration, inviteBaseURL string) http.Handler {
	mux := newMux(nil, sender)
	manager := auth.NewTokenManager(jwtSecret, accessTokenTTL)
	authenticated := authenticateRequests(authSvc, manager, time.Now)
	retrievalService = ensureRetrievalService(retrievalService)

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
	mux.Handle("POST /api/admin/invites", adminOrAbove(handleAdminInvite(authSvc, inviteBaseURL)))
	mux.Handle("POST /api/admin/invites/{id}/reissue", adminOrAbove(handleAdminInviteReissue(authSvc, inviteBaseURL)))
	mux.Handle("GET /api/admin/users", adminOrAbove(handleAdminUsers(adminProvider)))
	mux.Handle("GET /api/admin/classes/{id}/progress", teacherOrAbove(handleAdminClassProgress(adminProvider)))
	mux.Handle("GET /api/admin/students/{id}", teacherOrAbove(handleAdminStudentDetail(adminProvider)))
	mux.Handle("GET /api/admin/students/{id}/conversations", teacherOrAbove(handleAdminStudentConversations(adminProvider)))
	mux.Handle("POST /api/admin/students/{id}/nudge", teacherOrAbove(handleAdminStudentNudge(adminProvider, sender)))
	mux.Handle("GET /api/admin/metrics", teacherOrAbove(handleAdminMetrics(adminProvider)))
	mux.Handle("GET /api/admin/ai/usage", teacherOrAbove(handleAdminAIUsage(adminProvider)))
	mux.Handle("POST /api/admin/ai/budget-window", adminOnly(handleAdminUpsertTokenBudgetWindow(adminProvider)))
	mux.Handle("GET /api/admin/export/students", adminOrAbove(handleAdminExportStudents(adminProvider)))
	mux.Handle("GET /api/admin/export/conversations", adminOrAbove(handleAdminExportConversations(adminProvider)))
	mux.Handle("GET /api/admin/export/progress", adminOrAbove(handleAdminExportProgress(adminProvider)))
	mux.Handle("GET /api/admin/parents/{id}", parentOrAbove(handleAdminParentSummary(adminProvider)))
	// Group CRUD
	mux.Handle("GET /api/admin/groups", teacherOrAbove(handleAdminListGroups(adminProvider)))
	mux.Handle("POST /api/admin/groups", teacherOrAbove(handleAdminCreateGroup(adminProvider)))
	mux.Handle("GET /api/admin/groups/{id}", teacherOrAbove(handleAdminGetGroup(adminProvider)))
	mux.Handle("PATCH /api/admin/groups/{id}", adminOrAbove(handleAdminUpdateGroup(adminProvider)))
	mux.Handle("DELETE /api/admin/groups/{id}", adminOrAbove(handleAdminDeleteGroup(adminProvider)))
	mux.Handle("POST /api/admin/groups/{id}/members", adminOrAbove(handleAdminAddGroupMember(adminProvider)))
	mux.Handle("DELETE /api/admin/groups/{id}/members/{uid}", adminOrAbove(handleAdminRemoveGroupMember(adminProvider)))
	mux.Handle("GET /api/admin/groups/{id}/leaderboard", teacherOrAbove(handleAdminGroupLeaderboard(adminProvider)))
	registerRetrievalRoutes(mux, retrievalService, teacherOrAbove, adminOrAbove)

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

func newBootstrapRetrievalService(loader *curriculum.Loader) *retrieval.Service {
	retrievalService := retrieval.NewMemoryService()
	// Boot flow:
	//  1. create one shared in-process retrieval service
	//  2. seed curriculum into it as a normal retrieval source
	//  3. hand the same instance to both chat-time retrieval and admin APIs
	if err := retrieval.SeedCurriculum(retrievalService, loader); err != nil {
		slog.Warn("retrieval seed failed", "error", err)
	}
	return retrievalService
}

func ensureRetrievalService(retrievalService *retrieval.Service) *retrieval.Service {
	if retrievalService != nil {
		return retrievalService
	}
	return retrieval.NewMemoryService()
}

func registerRetrievalRoutes(
	mux *http.ServeMux,
	retrievalService *retrieval.Service,
	teacherOrAbove func(http.Handler) http.Handler,
	adminOrAbove func(http.Handler) http.Handler,
) {
	// Entity model:
	//  1. Source is the real origin of knowledge: curriculum, website, PDF, book, YouTube.
	//  2. Collection is the grouping/scope unit.
	//  3. Document is the searchable knowledge unit.
	// Retrieval now uses only Source / Collection / Document names end to end.
	mux.Handle("GET /api/admin/retrieval/collections", teacherOrAbove(handleRetrievalListCollections(retrievalService)))
	mux.Handle("POST /api/admin/retrieval/collections", adminOrAbove(handleRetrievalCreateCollection(retrievalService)))
	mux.Handle("GET /api/admin/retrieval/collections/{id}", teacherOrAbove(handleRetrievalGetCollection(retrievalService)))
	mux.Handle("PUT /api/admin/retrieval/collections/{id}", adminOrAbove(handleRetrievalUpdateCollection(retrievalService)))
	mux.Handle("DELETE /api/admin/retrieval/collections/{id}", adminOrAbove(handleRetrievalDeleteCollection(retrievalService)))
	mux.Handle("POST /api/admin/retrieval/collections/{id}/activate", adminOrAbove(handleRetrievalActivateCollection(retrievalService)))
	mux.Handle("GET /api/admin/retrieval/documents", teacherOrAbove(handleRetrievalListDocuments(retrievalService)))
	mux.Handle("POST /api/admin/retrieval/documents", adminOrAbove(handleRetrievalCreateDocument(retrievalService)))
	mux.Handle("GET /api/admin/retrieval/documents/{id}", teacherOrAbove(handleRetrievalGetDocument(retrievalService)))
	mux.Handle("PUT /api/admin/retrieval/documents/{id}", adminOrAbove(handleRetrievalUpdateDocument(retrievalService)))
	mux.Handle("DELETE /api/admin/retrieval/documents/{id}", adminOrAbove(handleRetrievalDeleteDocument(retrievalService)))
	mux.Handle("POST /api/admin/retrieval/documents/{id}/activate", adminOrAbove(handleRetrievalActivateDocument(retrievalService)))
	mux.Handle("GET /api/admin/retrieval/sources", teacherOrAbove(handleRetrievalListSources(retrievalService)))
	mux.Handle("POST /api/admin/retrieval/sources", adminOrAbove(handleRetrievalCreateSource(retrievalService)))
	mux.Handle("GET /api/admin/retrieval/sources/{id}", teacherOrAbove(handleRetrievalGetSource(retrievalService)))
	mux.Handle("PUT /api/admin/retrieval/sources/{id}", adminOrAbove(handleRetrievalUpdateSource(retrievalService)))
	mux.Handle("DELETE /api/admin/retrieval/sources/{id}", adminOrAbove(handleRetrievalDeleteSource(retrievalService)))
	mux.Handle("POST /api/admin/retrieval/sources/{id}/activate", adminOrAbove(handleRetrievalActivateSource(retrievalService)))
	mux.Handle("POST /api/admin/retrieval/search", teacherOrAbove(handleRetrievalSearch(retrievalService)))
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

func handleAdminUsers(adminProvider adminDataSourceProvider) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		admin, ok := resolveAdminDataSource(w, r, adminProvider)
		if !ok {
			return
		}

		payload, err := admin.GetUserManagement()
		if err != nil {
			writeAdminError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, payload)
	}
}

func handleAdminExportStudents(adminProvider adminDataSourceProvider) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		admin, ok := resolveAdminDataSource(w, r, adminProvider)
		if !ok {
			return
		}

		rows, err := admin.ExportStudents()
		if err != nil {
			writeAdminError(w, err)
			return
		}

		writeCSV(w, "students-export.csv", []string{
			"student_id",
			"name",
			"external_id",
			"channel",
			"form",
			"average_mastery",
			"tracked_topics",
			"created_at",
		}, func(writeRow func([]string) error) error {
			for _, row := range rows {
				if err := writeRow([]string{
					row.StudentID,
					row.Name,
					row.ExternalID,
					row.Channel,
					row.Form,
					formatOptionalFloat(row.AverageMastery),
					fmt.Sprintf("%d", row.TrackedTopics),
					row.CreatedAt.UTC().Format(time.RFC3339),
				}); err != nil {
					return err
				}
			}
			return nil
		})
	}
}

func handleAdminExportConversations(adminProvider adminDataSourceProvider) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		admin, ok := resolveAdminDataSource(w, r, adminProvider)
		if !ok {
			return
		}

		payload, err := admin.ExportConversations()
		if err != nil {
			writeAdminError(w, err)
			return
		}
		w.Header().Set("Content-Disposition", `attachment; filename="conversations-export.json"`)
		writeJSON(w, http.StatusOK, payload)
	}
}

func handleAdminExportProgress(adminProvider adminDataSourceProvider) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		admin, ok := resolveAdminDataSource(w, r, adminProvider)
		if !ok {
			return
		}

		rows, err := admin.ExportProgress()
		if err != nil {
			writeAdminError(w, err)
			return
		}

		writeCSV(w, "progress-export.csv", []string{
			"student_id",
			"student_name",
			"topic_id",
			"mastery_score",
			"ease_factor",
			"interval_days",
			"next_review_at",
			"last_studied_at",
		}, func(writeRow func([]string) error) error {
			for _, row := range rows {
				if err := writeRow([]string{
					row.StudentID,
					row.StudentName,
					row.TopicID,
					fmt.Sprintf("%.4f", row.MasteryScore),
					fmt.Sprintf("%.4f", row.EaseFactor),
					fmt.Sprintf("%d", row.IntervalDays),
					formatOptionalTime(row.NextReviewAt),
					formatOptionalTime(row.LastStudiedAt),
				}); err != nil {
					return err
				}
			}
			return nil
		})
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

func writeCSV(w http.ResponseWriter, filename string, header []string, writeRows func(func([]string) error) error) {
	w.Header().Set("Content-Type", "text/csv; charset=utf-8")
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, filename))
	w.WriteHeader(http.StatusOK)

	writer := csv.NewWriter(w)
	if err := writer.Write(header); err != nil {
		http.Error(w, "failed to write export header", http.StatusInternalServerError)
		return
	}
	if err := writeRows(writer.Write); err != nil {
		http.Error(w, "failed to write export rows", http.StatusInternalServerError)
		return
	}
	writer.Flush()
}

func formatOptionalFloat(value *float64) string {
	if value == nil {
		return ""
	}
	return fmt.Sprintf("%.4f", *value)
}

func formatOptionalTime(value *time.Time) string {
	if value == nil {
		return ""
	}
	return value.UTC().Format(time.RFC3339)
}

func handleRetrievalListSources(service *retrieval.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		includeInactive := r.URL.Query().Get("include_inactive") == "true"
		var types []string
		if sourceType := strings.TrimSpace(r.URL.Query().Get("type")); sourceType != "" {
			types = []string{sourceType}
		}
		payload, err := service.ListSources(retrieval.ListSourcesRequest{
			Types:           types,
			IncludeInactive: includeInactive,
		})
		if err != nil {
			writeRetrievalError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, payload)
	}
}

func handleRetrievalGetSource(service *retrieval.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		source, err := service.GetSource(r.PathValue("id"))
		if err != nil {
			writeRetrievalError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, source)
	}
}

func handleRetrievalCreateSource(service *retrieval.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			Type        string            `json:"type"`
			URI         string            `json:"uri,omitempty"`
			Title       string            `json:"title"`
			Description string            `json:"description,omitempty"`
			Metadata    map[string]string `json:"metadata,omitempty"`
			Active      *bool             `json:"active,omitempty"`
		}
		if err := decodeJSONBody(r, &body); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		source, err := service.UpsertSource(retrieval.UpsertSourceInput{
			Type:        body.Type,
			URI:         body.URI,
			Title:       body.Title,
			Description: body.Description,
			Metadata:    body.Metadata,
			Active:      body.Active,
		})
		if err != nil {
			writeRetrievalError(w, err)
			return
		}
		writeJSON(w, http.StatusCreated, source)
	}
}

func handleRetrievalUpdateSource(service *retrieval.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			Type        string            `json:"type"`
			URI         string            `json:"uri,omitempty"`
			Title       string            `json:"title"`
			Description string            `json:"description,omitempty"`
			Metadata    map[string]string `json:"metadata,omitempty"`
			Active      *bool             `json:"active,omitempty"`
		}
		if err := decodeJSONBody(r, &body); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		source, err := service.UpsertSource(retrieval.UpsertSourceInput{
			ID:          r.PathValue("id"),
			Type:        body.Type,
			URI:         body.URI,
			Title:       body.Title,
			Description: body.Description,
			Metadata:    body.Metadata,
			Active:      body.Active,
		})
		if err != nil {
			writeRetrievalError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, source)
	}
}

func handleRetrievalActivateSource(service *retrieval.Service) http.HandlerFunc {
	type request struct {
		Active bool `json:"active"`
	}

	return func(w http.ResponseWriter, r *http.Request) {
		var body request
		if err := decodeJSONBody(r, &body); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		source, err := service.SetSourceActive(r.PathValue("id"), body.Active)
		if err != nil {
			writeRetrievalError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, source)
	}
}

func handleRetrievalDeleteSource(service *retrieval.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := service.DeleteSource(r.PathValue("id")); err != nil {
			writeRetrievalError(w, err)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}
}

func handleRetrievalListCollections(service *retrieval.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		includeInactive := r.URL.Query().Get("include_inactive") == "true"
		payload := service.ListCollections(retrieval.ListCollectionsRequest{
			IncludeInactive: includeInactive,
		})
		writeJSON(w, http.StatusOK, payload)
	}
}

func handleRetrievalGetCollection(service *retrieval.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		collection, err := service.GetCollection(r.PathValue("id"))
		if err != nil {
			writeRetrievalError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, collection)
	}
}

func handleRetrievalCreateCollection(service *retrieval.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			Name        string            `json:"name"`
			Description string            `json:"description"`
			Metadata    map[string]string `json:"metadata"`
			Active      *bool             `json:"active,omitempty"`
		}
		if err := decodeJSONBody(r, &body); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		collection, err := service.UpsertCollection(retrieval.UpsertCollectionInput{
			Name:        body.Name,
			Description: body.Description,
			Metadata:    body.Metadata,
			Active:      body.Active,
		})
		if err != nil {
			writeRetrievalError(w, err)
			return
		}
		writeJSON(w, http.StatusCreated, collection)
	}
}

func handleRetrievalUpdateCollection(service *retrieval.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			Name        string            `json:"name"`
			Description string            `json:"description"`
			Metadata    map[string]string `json:"metadata"`
			Active      *bool             `json:"active,omitempty"`
		}
		if err := decodeJSONBody(r, &body); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		collection, err := service.UpsertCollection(retrieval.UpsertCollectionInput{
			ID:          r.PathValue("id"),
			Name:        body.Name,
			Description: body.Description,
			Metadata:    body.Metadata,
			Active:      body.Active,
		})
		if err != nil {
			writeRetrievalError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, collection)
	}
}

func handleRetrievalActivateCollection(service *retrieval.Service) http.HandlerFunc {
	type request struct {
		Active bool `json:"active"`
	}

	return func(w http.ResponseWriter, r *http.Request) {
		var body request
		if err := decodeJSONBody(r, &body); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		collection, err := service.SetCollectionActive(r.PathValue("id"), body.Active)
		if err != nil {
			writeRetrievalError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, collection)
	}
}

func handleRetrievalDeleteCollection(service *retrieval.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := service.DeleteCollection(r.PathValue("id")); err != nil {
			writeRetrievalError(w, err)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}
}

func handleRetrievalListDocuments(service *retrieval.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		includeInactive := r.URL.Query().Get("include_inactive") == "true"
		var collectionIDs []string
		collectionID := strings.TrimSpace(r.URL.Query().Get("collection_id"))
		if collectionID != "" {
			collectionIDs = []string{collectionID}
		}
		var sourceIDs []string
		if sourceID := strings.TrimSpace(r.URL.Query().Get("source_id")); sourceID != "" {
			sourceIDs = []string{sourceID}
		}
		var sourceTypes []string
		if sourceType := strings.TrimSpace(r.URL.Query().Get("source_type")); sourceType != "" {
			sourceTypes = []string{sourceType}
		}
		payload, err := service.ListDocument(retrieval.ListDocumentsRequest{
			CollectionIDs:   collectionIDs,
			SourceIDs:       sourceIDs,
			SourceTypes:     sourceTypes,
			IncludeInactive: includeInactive,
		})
		if err != nil {
			writeRetrievalError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, payload)
	}
}

func handleRetrievalGetDocument(service *retrieval.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		document, err := service.GetDocument(r.PathValue("id"))
		if err != nil {
			writeRetrievalError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, document)
	}
}

func handleRetrievalCreateDocument(service *retrieval.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			CollectionID string            `json:"collection_id,omitempty"`
			Kind         string            `json:"kind"`
			Title        string            `json:"title"`
			Body         string            `json:"body"`
			Tags         []string          `json:"tags,omitempty"`
			SourceID     string            `json:"source_id,omitempty"`
			SourceType   string            `json:"source_type,omitempty"`
			Metadata     map[string]string `json:"metadata,omitempty"`
			Source       string            `json:"source,omitempty"`
			Active       *bool             `json:"active,omitempty"`
		}
		if err := decodeJSONBody(r, &body); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		document, err := service.UpsertDocument(retrieval.UpsertDocumentInput{
			CollectionID: body.CollectionID,
			Kind:         body.Kind,
			Title:        body.Title,
			Body:         body.Body,
			Tags:         body.Tags,
			SourceID:     body.SourceID,
			SourceType:   body.SourceType,
			Metadata:     body.Metadata,
			Source:       body.Source,
			Active:       body.Active,
		})
		if err != nil {
			writeRetrievalError(w, err)
			return
		}
		writeJSON(w, http.StatusCreated, document)
	}
}

func handleRetrievalUpdateDocument(service *retrieval.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			CollectionID string            `json:"collection_id,omitempty"`
			Kind         string            `json:"kind"`
			Title        string            `json:"title"`
			Body         string            `json:"body"`
			Tags         []string          `json:"tags,omitempty"`
			SourceID     string            `json:"source_id,omitempty"`
			SourceType   string            `json:"source_type,omitempty"`
			Metadata     map[string]string `json:"metadata,omitempty"`
			Source       string            `json:"source,omitempty"`
			Active       *bool             `json:"active,omitempty"`
		}
		if err := decodeJSONBody(r, &body); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		document, err := service.UpsertDocument(retrieval.UpsertDocumentInput{
			ID:           r.PathValue("id"),
			CollectionID: body.CollectionID,
			Kind:         body.Kind,
			Title:        body.Title,
			Body:         body.Body,
			Tags:         body.Tags,
			SourceID:     body.SourceID,
			SourceType:   body.SourceType,
			Metadata:     body.Metadata,
			Source:       body.Source,
			Active:       body.Active,
		})
		if err != nil {
			writeRetrievalError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, document)
	}
}

func handleRetrievalActivateDocument(service *retrieval.Service) http.HandlerFunc {
	type request struct {
		Active bool `json:"active"`
	}

	return func(w http.ResponseWriter, r *http.Request) {
		var body request
		if err := decodeJSONBody(r, &body); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		document, err := service.SetDocumentActive(r.PathValue("id"), body.Active)
		if err != nil {
			writeRetrievalError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, document)
	}
}

func handleRetrievalDeleteDocument(service *retrieval.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := service.DeleteDocument(r.PathValue("id")); err != nil {
			writeRetrievalError(w, err)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}
}

func handleRetrievalSearch(service *retrieval.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var body retrieval.SearchRequest
		if err := decodeJSONBody(r, &body); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		results, err := service.Search(body)
		if err != nil {
			writeRetrievalError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, results)
	}
}

func handleAdminListGroups(adminProvider adminDataSourceProvider) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		admin, ok := resolveAdminDataSource(w, r, adminProvider)
		if !ok {
			return
		}
		groupType := r.URL.Query().Get("type")
		payload, err := admin.ListGroups(groupType)
		if err != nil {
			writeAdminError(w, err)
			return
		}
		if payload == nil {
			payload = []adminapi.AdminGroup{}
		}
		writeJSON(w, http.StatusOK, payload)
	}
}

func handleAdminCreateGroup(adminProvider adminDataSourceProvider) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		admin, ok := resolveAdminDataSource(w, r, adminProvider)
		if !ok {
			return
		}
		var input adminapi.CreateGroupInput
		if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
			http.Error(w, "invalid request body", http.StatusBadRequest)
			return
		}
		createdBy := ""
		if claims, ok := auth.ClaimsFromContext(r.Context()); ok {
			createdBy = claims.Subject
		}
		payload, err := admin.CreateGroup(input, createdBy)
		if err != nil {
			writeAdminError(w, err)
			return
		}
		writeJSON(w, http.StatusCreated, payload)
	}
}

func handleAdminGetGroup(adminProvider adminDataSourceProvider) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		admin, ok := resolveAdminDataSource(w, r, adminProvider)
		if !ok {
			return
		}
		payload, err := admin.GetGroupDetail(r.PathValue("id"))
		if err != nil {
			writeAdminError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, payload)
	}
}

func handleAdminUpdateGroup(adminProvider adminDataSourceProvider) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		admin, ok := resolveAdminDataSource(w, r, adminProvider)
		if !ok {
			return
		}
		var input adminapi.AdminUpdateGroupInput
		if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
			http.Error(w, "invalid request body", http.StatusBadRequest)
			return
		}
		payload, err := admin.UpdateGroup(r.PathValue("id"), input)
		if err != nil {
			writeAdminError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, payload)
	}
}

func handleAdminDeleteGroup(adminProvider adminDataSourceProvider) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		admin, ok := resolveAdminDataSource(w, r, adminProvider)
		if !ok {
			return
		}
		if err := admin.DeleteGroup(r.PathValue("id")); err != nil {
			writeAdminError(w, err)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}
}

func handleAdminAddGroupMember(adminProvider adminDataSourceProvider) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		admin, ok := resolveAdminDataSource(w, r, adminProvider)
		if !ok {
			return
		}
		var input adminapi.AddMemberInput
		if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
			http.Error(w, "invalid request body", http.StatusBadRequest)
			return
		}
		if err := admin.AddGroupMember(r.PathValue("id"), input.UserID, input.Role); err != nil {
			writeAdminError(w, err)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}
}

func handleAdminRemoveGroupMember(adminProvider adminDataSourceProvider) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		admin, ok := resolveAdminDataSource(w, r, adminProvider)
		if !ok {
			return
		}
		if err := admin.RemoveGroupMember(r.PathValue("id"), r.PathValue("uid")); err != nil {
			writeAdminError(w, err)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}
}

func handleAdminGroupLeaderboard(adminProvider adminDataSourceProvider) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		admin, ok := resolveAdminDataSource(w, r, adminProvider)
		if !ok {
			return
		}
		payload, err := admin.GetGroupLeaderboard(r.PathValue("id"))
		if err != nil {
			writeAdminError(w, err)
			return
		}
		if payload == nil {
			payload = []adminapi.AdminLeaderboardEntry{}
		}
		writeJSON(w, http.StatusOK, payload)
	}
}

func handleAdminInvite(authSvc authService, defaultBaseURL string) http.HandlerFunc {
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
			InvitedByUserID:   claims.Subject,
			TenantID:          claims.TenantID,
			Email:             body.Email,
			Role:              auth.Role(body.Role),
			ActivationBaseURL: inviteActivationBaseURL(r, defaultBaseURL),
		})
		if err != nil {
			writeAuthError(w, err)
			return
		}

		writeJSON(w, http.StatusCreated, resp)
	}
}

func handleAdminInviteReissue(authSvc authService, defaultBaseURL string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		claims, ok := auth.ClaimsFromContext(r.Context())
		if !ok {
			http.Error(w, "missing auth claims", http.StatusUnauthorized)
			return
		}

		resp, err := authSvc.ReissueInvite(r.Context(), auth.ReissueInviteRequest{
			InviteID:          r.PathValue("id"),
			InvitedByUserID:   claims.Subject,
			TenantID:          claims.TenantID,
			ActivationBaseURL: inviteActivationBaseURL(r, defaultBaseURL),
		})
		if err != nil {
			writeAuthError(w, err)
			return
		}

		writeJSON(w, http.StatusOK, resp)
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

		writeAuthSessionResponse(w, r, http.StatusOK, session)
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
	ExpiresAt     time.Time           `json:"expires_at"`
	User          auth.UserSession    `json:"user"`
	TenantChoices []auth.TenantOption `json:"tenant_choices,omitempty"`
}

func writeAuthSessionResponse(w http.ResponseWriter, r *http.Request, status int, session auth.Session) {
	setSessionCookies(w, r, session)
	writeJSON(w, status, authSessionResponse{
		ExpiresAt:     session.ExpiresAt,
		User:          session.User,
		TenantChoices: session.TenantChoices,
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

func inviteActivationBaseURL(r *http.Request, defaultBaseURL string) string {
	baseURL := requestBaseURL(r)
	if strings.TrimSpace(baseURL) != "" {
		return baseURL
	}
	return strings.TrimSpace(defaultBaseURL)
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

func writeRetrievalError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, retrieval.ErrNotFound):
		http.Error(w, http.StatusText(http.StatusNotFound), http.StatusNotFound)
	case errors.Is(err, retrieval.ErrInvalidArgument):
		http.Error(w, err.Error(), http.StatusBadRequest)
	default:
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
	}
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
