// Copyright 2026 the P&AI authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/p-n-ai/pai-bot/internal/adminapi"
	"github.com/p-n-ai/pai-bot/internal/agent"
	"github.com/p-n-ai/pai-bot/internal/ai"
	"github.com/p-n-ai/pai-bot/internal/auth"
	"github.com/p-n-ai/pai-bot/internal/chat"
	"github.com/p-n-ai/pai-bot/internal/curriculum"
	"github.com/p-n-ai/pai-bot/internal/focusedpage"
	"github.com/p-n-ai/pai-bot/internal/focusedpagedelivery"
	"github.com/p-n-ai/pai-bot/internal/platform/airouter"
	"github.com/p-n-ai/pai-bot/internal/platform/cache"
	"github.com/p-n-ai/pai-bot/internal/platform/codexauth"
	"github.com/p-n-ai/pai-bot/internal/platform/config"
	"github.com/p-n-ai/pai-bot/internal/platform/database"
	"github.com/p-n-ai/pai-bot/internal/platform/featureflags"
	"github.com/p-n-ai/pai-bot/internal/platform/mailer"
	"github.com/p-n-ai/pai-bot/internal/platform/settings"
	platformtenant "github.com/p-n-ai/pai-bot/internal/platform/tenant"
	"github.com/p-n-ai/pai-bot/internal/progress"
	"github.com/p-n-ai/pai-bot/internal/retrieval"
	"github.com/p-n-ai/pai-bot/internal/server"
	"github.com/p-n-ai/pai-bot/internal/tenant"
)

func focusedPageChannelEnabled(devMode bool, msg chat.InboundMessage) bool {
	return msg.Channel == "telegram" || (devMode && msg.Channel == "websocket")
}

type runtimeSettingsUpdater interface {
	Update(
		context.Context,
		func(settings.Settings) (settings.Settings, error),
		settings.PrepareApply,
	) (settings.Settings, error)
}

func makeCodexDefault(
	ctx context.Context,
	store runtimeSettingsUpdater,
	prepareSettings settings.PrepareApply,
) error {
	_, err := store.Update(ctx, func(current settings.Settings) (settings.Settings, error) {
		current.AI.DefaultProvider = "codex"
		return current, nil
	}, prepareSettings)
	return err
}

func canAwaitCodexDeviceAuth(cfg *config.Config, manager *codexauth.Manager) bool {
	return cfg != nil &&
		manager != nil &&
		cfg.CodexDeviceAuthAvailable() &&
		manager.Available()
}

func main() {
	cfg, err := config.Load()
	if err != nil {
		slog.Error("failed to load config", "error", err)
		os.Exit(1)
	}

	slog.SetDefault(slog.New(newLogHandler(cfg.Log)))

	if err := cfg.Validate(); err != nil {
		slog.Error("invalid config", "error", err)
		os.Exit(1)
	}

	// Graceful shutdown on SIGTERM/SIGINT.
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGTERM, syscall.SIGINT)
	defer stop()

	if err := run(ctx, cfg); err != nil {
		slog.Error("server stopped", "error", err)
		os.Exit(1)
	}
}

func run(ctx context.Context, cfg *config.Config) (runErr error) {
	var cleanup []func() error
	defer func() {
		runErr = errors.Join(runErr, closeAll(cleanup))
	}()

	return server.Run(ctx, server.Options{
		Addr:            fmt.Sprintf(":%d", cfg.Server.Port),
		ReadTimeout:     10 * time.Second,
		WriteTimeout:    30 * time.Second,
		IdleTimeout:     60 * time.Second,
		ShutdownTimeout: 10 * time.Second,
		BuildHandler: func(ctx context.Context) (http.Handler, func(context.Context) error, error) {

			// Initialize PostgreSQL-backed conversation store.
			db, err := database.New(ctx, cfg.Database.URL, cfg.Database.MaxConns, cfg.Database.MinConns)
			if err != nil {
				return nil, nil, fmt.Errorf("connect to database: %w", err)
			}
			cleanup = append(cleanup, func() error {
				db.Close()
				return nil
			})

			// In single-tenant mode, ensure the default tenant exists for runtime dependencies.
			if _, err := tenant.EnsureDefaultTenantForPool(ctx, cfg.Tenant.Mode, db.Pool); err != nil {
				return nil, nil, fmt.Errorf("bootstrap tenant mode %q: %w", cfg.Tenant.Mode, err)
			}

			// Runtime settings overlay env config; admin saves re-apply live.
			settingsStore := settings.NewWithPreviousKeys(
				db.Pool,
				cfg.Security.RuntimeSettingsEncryptionKey,
				cfg.Security.PreviousSettingsEncryptionKeys,
				cfg.Auth.JWTSecret,
				cfg.AI,
				cfg.FeatureFlags,
			)
			if err := settingsStore.Start(ctx); err != nil {
				// Degrade to env-only config: a crash loop here would lock
				// admins out of the very UI that repairs the stored settings.
				slog.Warn("runtime settings unavailable; using env config", "error", err)
			}

			codexExecutable, _ := exec.LookPath("codex")
			codexDeviceAuth := codexauth.New(ctx, cfg.AI.Codex.Home, codexExecutable, nil)

			// Initialize AI router with configured providers.
			initialAI := settings.MergeAI(cfg.AI, settingsStore.Current())
			router := ai.NewRouter()
			initialPlan, initialPlanErr := airouter.PrepareWithCodexAuth(initialAI, codexDeviceAuth)
			initialPlan.Apply(router)
			if initialPlanErr == nil {
				settingsStore.MarkApplied(settingsStore.Current().Revision)
			} else {
				slog.Warn("runtime settings could not be fully applied", "error", initialPlanErr)
			}
			aiHealthCheck := server.NewCachedHealthCheck(
				server.NewAIHealthCheck(router),
				time.Minute,
				5*time.Second,
			)
			if !router.HasProvider() {
				if cfg.Runtime.DevMode {
					slog.Warn("no AI providers configured; continuing in dev mode without AI-backed chat responses")
				} else if canAwaitCodexDeviceAuth(cfg, codexDeviceAuth) {
					slog.Warn("no AI providers authenticated; continuing so an admin can connect Codex")
				} else {
					return nil, nil, errors.New("no AI providers configured")
				}
			}
			prepareSettings := func(st settings.Settings) (settings.PreparedApply, error) {
				merged := settings.MergeAI(cfg.AI, st)
				plan, err := airouter.PrepareWithCodexAuth(merged, codexDeviceAuth)
				if err != nil {
					return nil, err
				}
				return func() { plan.Apply(router) }, nil
			}
			var adminCodexDeviceAuth server.CodexDeviceAuth
			if cfg.AI.Codex.Enabled {
				codexDeviceAuth.SetOnConnected(func(callbackCtx context.Context) error {
					return makeCodexDefault(callbackCtx, settingsStore, prepareSettings)
				})
				codexDeviceAuth.Initialize()
				adminCodexDeviceAuth = codexDeviceAuth
			}

			var warnFlagOverrides sync.Once
			flagsProvider := func() featureflags.Features {
				merged, err := cfg.FeatureFlags.WithOverrides(settingsStore.Current().Flags)
				if err != nil {
					// Bad DB overrides must never crash a turn; fall back to env flags.
					warnFlagOverrides.Do(func() {
						slog.Warn("invalid runtime feature flag overrides; using env flags", "error", err)
					})
					return cfg.FeatureFlags
				}
				return merged
			}

			// Initialize cache (warn if unavailable, don't fail).
			if cfg.Cache.URL != "" {
				c, err := cache.New(ctx, cfg.Cache.URL)
				if err != nil {
					slog.Warn("cache not connected", "error", err)
				} else {
					cleanup = append(cleanup, func() error {
						return c.Close()
					})
					slog.Info("cache connected")
				}
			} else {
				slog.Warn("cache not configured, running without cache")
			}

			store, err := agent.NewPostgresStore(ctx, db.Pool)
			if err != nil {
				return nil, nil, fmt.Errorf("initialize conversation store: %w", err)
			}
			focusedPageStore := focusedpage.NewPostgresStore(db.Pool)
			focusedPageCleanup, err := server.NewFocusedPageCleanupWorker(focusedPageStore, nil)
			if err != nil {
				return nil, nil, fmt.Errorf("initialize focused page cleanup: %w", err)
			}
			var focusedPageService *focusedpage.Service
			var focusedPageHandler http.Handler
			if strings.TrimSpace(cfg.FocusedPage.BaseURL) != "" {
				focusedPageService, err = focusedpage.NewService(
					focusedPageStore, cfg.FocusedPage.BaseURL, []byte(cfg.Auth.JWTSecret), time.Now,
				)
				if err != nil {
					return nil, nil, fmt.Errorf("initialize focused pages: %w", err)
				}
				pageHandler, err := server.NewFocusedPageHandler(focusedPageService, cfg.FocusedPage.TelegramCTAURL)
				if err != nil {
					return nil, nil, fmt.Errorf("initialize focused page handler: %w", err)
				}
				focusedPageHandler = pageHandler
			}

			// Load curriculum (warn if unavailable, don't fail).
			loader, err := curriculum.NewLoader(cfg.CurriculumPath)
			if err != nil {
				slog.Warn("curriculum not loaded", "error", err, "path", cfg.CurriculumPath)
			} else {
				topics := loader.AllTopics()
				slog.Info("curriculum ready", "topics", len(topics))
			}
			retrievalService := server.NewBootstrapRetrievalService(loader)
			var retrievalEmbedder retrieval.Embedder
			if strings.TrimSpace(cfg.Retrieval.EmbeddingBaseURL) != "" {
				retrievalEmbedder, err = retrieval.NewOpenAICompatibleEmbedder(
					cfg.Retrieval.EmbeddingBaseURL,
					cfg.Retrieval.EmbeddingAPIKey,
					cfg.Retrieval.EmbeddingModel,
					cfg.Retrieval.EmbeddingDimensions,
					nil,
				)
				if err != nil {
					return nil, nil, fmt.Errorf("initialize retrieval embeddings: %w", err)
				}
			}
			teacherResources, err := retrieval.NewTeacherResourceService(db.Pool, retrieval.TeacherResourceOptions{
				Embedder:           retrievalEmbedder,
				EmbeddingModel:     cfg.Retrieval.EmbeddingModel,
				AllowGraphFallback: cfg.Runtime.DevMode,
				GraphDepth:         cfg.Retrieval.GraphDepth,
				GraphFrontier:      cfg.Retrieval.GraphFrontier,
			})
			if err != nil {
				return nil, nil, fmt.Errorf("initialize teacher retrieval: %w", err)
			}
			if err := teacherResources.VerifyGraph(ctx); err != nil {
				return nil, nil, fmt.Errorf("verify teacher retrieval graph: %w", err)
			}
			tutorEvidence, err := retrieval.NewTutorEvidenceService(db.Pool, retrievalService, teacherResources)
			if err != nil {
				return nil, nil, fmt.Errorf("initialize tutor evidence retrieval: %w", err)
			}

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
				EvidenceRetriever:    tutorEvidence,
				DisableMultiLanguage: cfg.Runtime.DisableMultiLanguage,
				Tracker:              tracker,
				Streaks:              streakTracker,
				XP:                   xpTracker,
				Goals:                goalStore,
				Challenges:           challengeStore,
				Groups:               groupStore,
				TenantID:             store.TenantID(),
				DevMode:              cfg.Runtime.DevMode,
				FeatureFlags:         flagsProvider,
				FocusedPages:         focusedPageService,
				FocusedPageEnabled: func(msg chat.InboundMessage) bool {
					return focusedPageChannelEnabled(cfg.Runtime.DevMode, msg)
				},
			})

			gw := chat.NewGateway()
			if strings.TrimSpace(cfg.Telegram.BotToken) != "" {
				tg, err := chat.NewTelegramChannel(cfg.Telegram.BotToken)
				if err != nil {
					return nil, nil, fmt.Errorf("create Telegram channel: %w", err)
				}
				tg.SetDevMode(cfg.Runtime.DevMode)
				gw.Register("telegram", tg)
			} else {
				slog.Warn("telegram channel disabled; LEARN_TELEGRAM_BOT_TOKEN is not set")
			}
			if cfg.Slack.Enabled {
				slack, err := chat.NewSlackChannel(cfg.Slack.BotToken, cfg.Slack.SigningSecret)
				if err != nil {
					return nil, nil, fmt.Errorf("initialize Slack channel: %w", err)
				}
				gw.Register("slack", slack)
			}
			if cfg.Discord.Enabled {
				discord, err := chat.NewDiscordChannel(chat.DiscordConfig{
					BotToken:      cfg.Discord.BotToken,
					PublicKey:     cfg.Discord.PublicKey,
					ApplicationID: cfg.Discord.ApplicationID,
				})
				if err != nil {
					return nil, nil, fmt.Errorf("initialize Discord channel: %w", err)
				}
				gw.Register("discord", discord)
			}
			if cfg.Teams.Enabled {
				authenticator, err := chat.NewTeamsAuthenticator(
					cfg.Teams.AppID,
					cfg.Teams.AppPassword,
					cfg.Teams.AppTenantID,
				)
				if err != nil {
					return nil, nil, fmt.Errorf("initialize Teams authentication: %w", err)
				}
				teams, err := chat.NewTeamsChannel(chat.TeamsConfig{
					TokenValidator: authenticator,
					TokenProvider:  authenticator,
				})
				if err != nil {
					return nil, nil, fmt.Errorf("initialize Teams channel: %w", err)
				}
				gw.Register("teams", teams)
			}

			// WhatsApp channel (behind feature flag).
			var waCloudChannel *chat.WhatsAppChannel
			var waMeowChannel *chat.WhatsAppMeowChannel
			if cfg.WhatsApp.Enabled {
				switch cfg.WhatsApp.Backend {
				case "cloudapi":
					var waErr error
					waCloudChannel, waErr = chat.NewWhatsAppChannel(cfg.WhatsApp.AccessToken, cfg.WhatsApp.PhoneID, cfg.WhatsApp.VerifyToken)
					if waErr != nil {
						return nil, nil, fmt.Errorf("create WhatsApp Cloud API channel: %w", waErr)
					}
					gw.Register("whatsapp", waCloudChannel)
					slog.Info("whatsapp backend: Cloud API")
				default: // "meow"
					var waErr error
					waMeowChannel, waErr = chat.NewWhatsAppMeowChannel(cfg.WhatsApp.MeowDBPath)
					if waErr != nil {
						return nil, nil, fmt.Errorf("create WhatsApp meow channel: %w", waErr)
					}
					gw.Register("whatsapp", waMeowChannel)
					slog.Info("whatsapp backend: whatsmeow")
				}
			} else {
				slog.Info("whatsapp channel disabled; set LEARN_WHATSAPP_ENABLED=true to enable")
			}

			// Embed config store (for embeddable web chat widget).
			embedConfigStore := chat.NewPostgresEmbedConfigStore(db.Pool)

			// Keep legacy terminal chat isolated from the JWT-authenticated embed socket.
			embedTokenManager := auth.NewTokenManager(cfg.Auth.JWTSecret, defaultEmbedTokenTTL)
			embedGuestService := auth.NewGuestService(db.Pool, embedTokenManager)
			embedMessageStore := server.NewPostgresEmbedMessageStore(db.Pool)
			var wsChannel *chat.WSChannel
			if cfg.Runtime.DevMode {
				wsChannel = chat.NewWSChannel()
				gw.Register("websocket", wsChannel)
			}
			embedWSChannel := chat.NewEmbedWSChannel(embedConfigStore, embedTokenManager)
			gw.Register("embed", embedWSChannel)

			// Wire challenge notifications through the gateway.
			engine.SetNotifier(server.NewGatewayNotifier(gw, store))
			var focusedPageDeliveries *focusedpagedelivery.Processor
			if focusedPageService != nil {
				focusedPageDeliveries, err = focusedpagedelivery.NewProcessor(
					focusedpagedelivery.NewPostgresStore(db.Pool),
					server.NewGatewayFocusedPageSender(gw, store, focusedPageService),
					focusedpagedelivery.DefaultConfig(),
				)
				if err != nil {
					return nil, nil, fmt.Errorf("initialize focused-page deliveries: %w", err)
				}
			}
			engine.SetTurnDeliverer(server.NewGatewayTurnDeliverer(gw, store, focusedPageDeliveries))
			turnRouter, err := server.NewTenantTurnRouter(engine, func(tenantID string) (server.TurnProcessor, error) {
				tenantStore, err := agent.NewPostgresStoreForTenantAndChannel(db.Pool, tenantID, "embed")
				if err != nil {
					return nil, fmt.Errorf("initialize tenant conversation store: %w", err)
				}
				tenantEngine := agent.NewEngine(agent.EngineConfig{
					AIRouter:             router,
					Store:                tenantStore,
					EventLogger:          eventLogger,
					CurriculumLoader:     loader,
					RetrievalService:     retrievalService,
					EvidenceRetriever:    tutorEvidence,
					DisableMultiLanguage: cfg.Runtime.DisableMultiLanguage,
					Tracker:              progress.NewPostgresTracker(db.Pool, tenantID),
					Streaks:              progress.NewMemoryStreakTracker(),
					XP:                   progress.NewMemoryXPTracker(),
					Goals:                agent.NewPostgresGoalStore(db.Pool, tenantID),
					Challenges:           agent.NewPostgresChallengeStoreForChannel(db.Pool, tenantID, "embed"),
					Groups:               groupStore,
					TenantID:             tenantID,
					DevMode:              cfg.Runtime.DevMode,
					FeatureFlags:         flagsProvider,
					FocusedPages:         focusedPageService,
					FocusedPageEnabled: func(msg chat.InboundMessage) bool {
						return focusedPageChannelEnabled(cfg.Runtime.DevMode, msg)
					},
				})
				tenantEngine.SetNotifier(server.NewGatewayNotifier(gw, tenantStore))
				tenantEngine.SetTurnDeliverer(server.NewGatewayTurnDeliverer(gw, tenantStore, focusedPageDeliveries))
				return tenantEngine, nil
			})
			if err != nil {
				return nil, nil, fmt.Errorf("initialize tenant turn router: %w", err)
			}

			// Start proactive scheduler (nudges for due reviews).
			nudgeTracker := agent.NewPostgresNudgeTracker(db.Pool, store.TenantID())
			scheduler := agent.NewScheduler(
				agent.SchedulerConfig{
					CheckInterval:               agent.DefaultSchedulerConfig().CheckInterval,
					MaxNudgesPerDay:             agent.DefaultSchedulerConfig().MaxNudgesPerDay,
					AIPersonalizedNudgesEnabled: cfg.Runtime.AIPersonalizedNudgesEnabled,
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
			scheduler.SetWeeklyParentReportSource(server.NewWeeklyParentReportSource(adminapi.New(db.Pool, store.TenantID())))

			scheduler.SetGroupStore(groupStore, store.TenantID())

			// Scheduler runs in background; user list is empty initially — will be populated
			// when we add user enumeration from the database.
			go scheduler.Start(ctx, []string{})

			// Start long-polling with message handler.
			// Shared inbound message handler for all channels.
			processInbound := func(processCtx context.Context, msg chat.InboundMessage) {
				// Show typing indicator while processing.
				if err := gw.SendTyping(processCtx, msg.Channel, msg.DestinationID()); err != nil {
					slog.Warn("failed to send typing indicator", "error", err)
				}

				_, err := turnRouter.ProcessAndDeliver(processCtx, msg)
				if err != nil {
					slog.Error("process or deliver turn failed", "error", err, "user_id", msg.UserID)
				}
			}
			chatIngress, err := server.NewChatIngress(256, processInbound)
			if err != nil {
				return nil, nil, fmt.Errorf("initialize chat ingress: %w", err)
			}
			handleInbound := func(msg chat.InboundMessage) {
				if err := chatIngress.Enqueue(ctx, msg); err != nil && ctx.Err() == nil {
					slog.Warn("failed to enqueue inbound chat message", "channel", msg.Channel, "error", err)
				}
			}

			chatWebhooks := gw.Webhooks(handleInbound)

			authService := auth.NewPostgresService(
				db.Pool,
				defaultSessionTTL,
			)
			authService.ConfigureGoogleOAuth(auth.GoogleOAuthProviderConfig{
				ClientID:              cfg.Auth.Google.ClientID,
				ClientSecret:          cfg.Auth.Google.ClientSecret,
				DiscoveryURL:          cfg.Auth.Google.DiscoveryURL,
				AdminBaseURL:          cfg.Auth.Google.AdminBaseURL,
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
					return nil, nil, fmt.Errorf("create invite mailer: %w", err)
				}
				authService.ConfigureInviteEmail(inviteMailer)
			}
			createdBootstrapAdmin, err := authService.EnsureBootstrapPlatformAdmin(
				ctx,
				cfg.Auth.BootstrapAdmin.Email,
				cfg.Auth.BootstrapAdmin.Password,
			)
			if err != nil {
				return nil, nil, fmt.Errorf("ensure bootstrap platform admin: %w", err)
			}
			if createdBootstrapAdmin {
				slog.Info("bootstrap platform admin created", "email", cfg.Auth.BootstrapAdmin.Email)
			}

			// HTTP endpoints.
			apiHandler := server.NewHandlerWithAdminProviderAndTeacherResourcesAndCodexAuth(
				server.NewTenantAdminDataSourceProvider(
					func(tenantID string) server.AdminDataSource {
						return adminapi.New(db.Pool, tenantID)
					},
					func() server.AdminDataSource {
						return adminapi.NewPlatform(db.Pool)
					},
					func(ctx context.Context) (string, error) {
						return platformtenant.DefaultTenantID(ctx, db.Pool)
					},
				),
				adminapi.NewPublic(db.Pool),
				server.NewGatewaySender(gw),
				retrievalService,
				teacherResources,
				authService,
				cfg.Auth.JWTSecret,
				defaultAccessTokenTTL,
				cfg.Email.BaseURL,
				settingsStore,
				prepareSettings,
				cfg.Tenant.Mode == "multi",
				adminCodexDeviceAuth,
			)

			topMux := server.NewTopMux(server.TopMuxOptions{
				APIHandler:            apiHandler,
				WSChannel:             wsChannel,
				EmbedWSChannel:        embedWSChannel,
				EmbedConfigStore:      embedConfigStore,
				EmbedGuestService:     embedGuestService,
				EmbedMessageStore:     embedMessageStore,
				EmbedIdentityResolver: embedMessageStore,
				EmbedAuthenticator:    authService,
				EmbedBaseURL:          cfg.Embed.BaseURL,
				EmbedTokenTTL:         defaultEmbedTokenTTL,
				WACloudChannel:        waCloudChannel,
				WAMeowChannel:         waMeowChannel,
				ChatWebhooks:          chatWebhooks,
				InboundHandler:        handleInbound,
				AuthService:           authService,
				JWTSecret:             cfg.Auth.JWTSecret,
				AccessTokenTTL:        defaultAccessTokenTTL,
				FocusedPageHandler:    focusedPageHandler,
				PublicHealthEnabled: func() bool {
					return flagsProvider().Enabled(featureflags.PublicHealth)
				},
				AIHealthEnabled: func() bool {
					return flagsProvider().Enabled(featureflags.AIHealth)
				},
				AIHealthToken: cfg.Runtime.AIHealthToken,
				AIHealthCheck: aiHealthCheck,
			})

			return http.Handler(topMux), func(ctx context.Context) error {
				ingressCtx, cancelIngress := context.WithCancel(ctx)
				ingressDone := make(chan struct{})
				go func() {
					defer close(ingressDone)
					chatIngress.Run(ingressCtx)
				}()
				if err := gw.StartAll(ctx, handleInbound); err != nil {
					cancelIngress()
					<-ingressDone
					return err
				}
				cleanup = append(cleanup, func() error {
					err := gw.StopAll()
					cancelIngress()
					<-ingressDone
					return err
				})
				if focusedPageDeliveries != nil {
					workerCtx, cancelWorker := context.WithCancel(ctx)
					workerDone := make(chan struct{})
					go func() {
						defer close(workerDone)
						focusedPageDeliveries.Run(workerCtx)
					}()
					cleanup = append(cleanup, func() error {
						cancelWorker()
						<-workerDone
						return nil
					})
				}
				focusedPageCleanupDone := make(chan struct{})
				go func() {
					defer close(focusedPageCleanupDone)
					focusedPageCleanup.Run(ctx)
				}()
				cleanup = append(cleanup, func() error {
					<-focusedPageCleanupDone
					return nil
				})
				slog.Info("P&AI Bot is running")
				return nil
			}, nil
		},
	})
}

func closeAll(cleanup []func() error) error {
	var cleanupErrs []error
	for i := len(cleanup) - 1; i >= 0; i-- {
		if err := cleanup[i](); err != nil {
			cleanupErrs = append(cleanupErrs, err)
		}
	}
	return errors.Join(cleanupErrs...)
}

const (
	defaultAccessTokenTTL = 15 * time.Minute
	defaultEmbedTokenTTL  = time.Hour
	defaultSessionTTL     = 7 * 24 * time.Hour
)

func googleOAuthPolicy(cfg *config.Config) auth.GoogleOAuthPolicy {
	if cfg == nil {
		return auth.GoogleOAuthPolicy{}
	}
	return auth.AllowGoogleHostedDomains(cfg.Auth.Google.AllowedDomain)
}

func newLogHandler(cfg config.LogConfig) slog.Handler {
	var level slog.Level
	switch strings.ToLower(cfg.Level) {
	case "debug":
		level = slog.LevelDebug
	case "warn", "warning":
		level = slog.LevelWarn
	case "error":
		level = slog.LevelError
	default:
		level = slog.LevelInfo
	}
	opts := &slog.HandlerOptions{Level: level}

	if strings.ToLower(cfg.Format) == "text" {
		return slog.NewTextHandler(os.Stdout, opts)
	}
	return slog.NewJSONHandler(os.Stdout, opts)
}
