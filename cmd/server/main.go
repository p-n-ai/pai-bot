// Copyright 2026 the P&AI authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/p-n-ai/pai-bot/internal/adminapi"
	"github.com/p-n-ai/pai-bot/internal/agent"
	"github.com/p-n-ai/pai-bot/internal/auth"
	"github.com/p-n-ai/pai-bot/internal/chat"
	"github.com/p-n-ai/pai-bot/internal/curriculum"
	"github.com/p-n-ai/pai-bot/internal/focusedpage"
	"github.com/p-n-ai/pai-bot/internal/platform/airouter"
	"github.com/p-n-ai/pai-bot/internal/platform/cache"
	"github.com/p-n-ai/pai-bot/internal/platform/config"
	"github.com/p-n-ai/pai-bot/internal/platform/database"
	"github.com/p-n-ai/pai-bot/internal/platform/featureflags"
	"github.com/p-n-ai/pai-bot/internal/platform/mailer"
	"github.com/p-n-ai/pai-bot/internal/platform/settings"
	platformtenant "github.com/p-n-ai/pai-bot/internal/platform/tenant"
	"github.com/p-n-ai/pai-bot/internal/progress"
	"github.com/p-n-ai/pai-bot/internal/server"
	"github.com/p-n-ai/pai-bot/internal/tenant"
)

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
	var cleanup []func()
	defer func() {
		for i := len(cleanup) - 1; i >= 0; i-- {
			cleanup[i]()
		}
	}()

	if err := server.Run(ctx, server.Options{
		Addr:            fmt.Sprintf(":%d", cfg.Server.Port),
		ReadTimeout:     10 * time.Second,
		WriteTimeout:    30 * time.Second,
		IdleTimeout:     60 * time.Second,
		ShutdownTimeout: 10 * time.Second,
		BuildHandler: func(ctx context.Context) (http.Handler, func(context.Context) error, error) {

			// Initialize PostgreSQL-backed conversation store.
			db, err := database.New(context.Background(), cfg.Database.URL, cfg.Database.MaxConns, cfg.Database.MinConns)
			if err != nil {
				slog.Error("failed to connect to database", "error", err)
				os.Exit(1)
			}
			cleanup = append(cleanup, db.Close)

			// In single-tenant mode, ensure the default tenant exists for runtime dependencies.
			if _, err := tenant.EnsureDefaultTenantForPool(context.Background(), cfg.Tenant.Mode, db.Pool); err != nil {
				slog.Error("failed to bootstrap tenant mode", "mode", cfg.Tenant.Mode, "error", err)
				os.Exit(1)
			}

			// Runtime settings overlay env config; admin saves re-apply live.
			settingsStore := settings.New(db.Pool, cfg.Auth.JWTSecret, cfg.AI, cfg.FeatureFlags)
			if err := settingsStore.Start(context.Background()); err != nil {
				// Degrade to env-only config: a crash loop here would lock
				// admins out of the very UI that repairs the stored settings.
				slog.Warn("runtime settings unavailable; using env config", "error", err)
			}

			// Initialize AI router with configured providers.
			lastApplied := settings.MergeAI(cfg.AI, settingsStore.Current())
			router := airouter.Setup(lastApplied)
			if !router.HasProvider() {
				if cfg.Runtime.DevMode {
					slog.Warn("no AI providers configured; continuing in dev mode without AI-backed chat responses")
				} else {
					slog.Error("no AI providers configured")
					os.Exit(1)
				}
			}
			applySettings := func(st settings.Settings) {
				// Applies run in commit order under the store's update lock, so a plain lastApplied variable is safe.
				merged := settings.MergeAI(cfg.AI, st)
				if merged == lastApplied {
					return
				}
				lastApplied = merged
				airouter.Apply(router, merged)
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
				c, err := cache.New(context.Background(), cfg.Cache.URL)
				if err != nil {
					slog.Warn("cache not connected", "error", err)
				} else {
					cleanup = append(cleanup, func() { _ = c.Close() })
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
			var focusedPageService *focusedpage.Service
			var focusedPageHandler http.Handler
			if strings.TrimSpace(cfg.FocusedPage.BaseURL) != "" {
				focusedPageService, err = focusedpage.NewService(
					focusedpage.NewPostgresStore(db.Pool), cfg.FocusedPage.BaseURL, []byte(cfg.Auth.JWTSecret), time.Now,
				)
				if err != nil {
					return nil, nil, fmt.Errorf("initialize focused pages: %w", err)
				}
				pageHandler, err := focusedpage.NewHandler(focusedPageService, cfg.FocusedPage.TelegramCTAURL)
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
				DisableMultiLanguage: cfg.Runtime.DisableMultiLanguage,
				RatingPromptEvery:    cfg.Runtime.RatingPromptEvery,
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
			})

			gw := chat.NewGateway()
			if strings.TrimSpace(cfg.Telegram.BotToken) != "" {
				tg, err := chat.NewTelegramChannel(cfg.Telegram.BotToken)
				if err != nil {
					slog.Error("failed to create Telegram channel", "error", err)
					os.Exit(1)
				}
				tg.SetDevMode(cfg.Runtime.DevMode)
				gw.Register("telegram", tg)
			} else {
				slog.Warn("telegram channel disabled; LEARN_TELEGRAM_BOT_TOKEN is not set")
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
						slog.Error("failed to create WhatsApp Cloud API channel", "error", waErr)
						os.Exit(1)
					}
					gw.Register("whatsapp", waCloudChannel)
					slog.Info("whatsapp backend: Cloud API")
				default: // "meow"
					var waErr error
					waMeowChannel, waErr = chat.NewWhatsAppMeowChannel(cfg.WhatsApp.MeowDBPath)
					if waErr != nil {
						slog.Error("failed to create WhatsApp meow channel", "error", waErr)
						os.Exit(1)
					}
					gw.Register("whatsapp", waMeowChannel)
					slog.Info("whatsapp backend: whatsmeow")
				}
			} else {
				slog.Info("whatsapp channel disabled; set LEARN_WHATSAPP_ENABLED=true to enable")
			}

			// Embed config store (for embeddable web chat widget).
			embedConfigStore := chat.NewPostgresEmbedConfigStore(db.Pool)

			// WebSocket channel (always enabled — used by terminal-chat and embed web clients).
			// Dev mode keeps first-message auth for terminal-chat; production embed mode
			// requires origin checking and subprotocol JWT auth.
			embedTokenManager := auth.NewTokenManager(cfg.Auth.JWTSecret, time.Hour)
			var wsChannel *chat.WSChannel
			if cfg.Runtime.DevMode {
				wsChannel = chat.NewWSChannel()
			} else {
				wsChannel = chat.NewEmbedWSChannel(embedConfigStore, embedTokenManager)
			}
			gw.Register("websocket", wsChannel)

			// Wire challenge notifications through the gateway.
			engine.SetNotifier(server.NewGatewayNotifier(gw, store))
			engine.SetTurnDeliverer(server.NewGatewayTurnDeliverer(gw, store))

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
			handleInbound := func(msg chat.InboundMessage) {
				// Show typing indicator while processing.
				if err := gw.SendTyping(ctx, msg.Channel, msg.UserID); err != nil {
					slog.Warn("failed to send typing indicator", "error", err)
				}

				_, err := engine.ProcessAndDeliver(ctx, msg)
				if err != nil {
					slog.Error("process or deliver turn failed", "error", err, "user_id", msg.UserID)
				}
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
			createdBootstrapAdmin, err := authService.EnsureBootstrapPlatformAdmin(
				context.Background(),
				cfg.Auth.BootstrapAdmin.Email,
				cfg.Auth.BootstrapAdmin.Password,
			)
			if err != nil {
				slog.Error("failed to ensure bootstrap platform admin", "error", err)
				os.Exit(1)
			}
			if createdBootstrapAdmin {
				slog.Info("bootstrap platform admin created", "email", cfg.Auth.BootstrapAdmin.Email)
			}

			// HTTP endpoints.
			apiHandler := server.NewHandlerWithAdminProvider(
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
				authService,
				cfg.Auth.JWTSecret,
				defaultAccessTokenTTL,
				cfg.Email.BaseURL,
				settingsStore,
				applySettings,
				cfg.Tenant.Mode == "multi",
			)

			topMux := server.NewTopMux(server.TopMuxOptions{
				APIHandler:         apiHandler,
				WSChannel:          wsChannel,
				EmbedConfigStore:   embedConfigStore,
				WACloudChannel:     waCloudChannel,
				WAMeowChannel:      waMeowChannel,
				InboundHandler:     handleInbound,
				AuthService:        authService,
				JWTSecret:          cfg.Auth.JWTSecret,
				AccessTokenTTL:     defaultAccessTokenTTL,
				FocusedPageHandler: focusedPageHandler,
			})

			return http.Handler(topMux), func(ctx context.Context) error {
				if err := gw.StartAll(ctx, handleInbound); err != nil {
					return err
				}
				slog.Info("P&AI Bot is running")
				return nil
			}, nil
		},
	}); err != nil {
		slog.Error("server stopped", "error", err)
		os.Exit(1)
	}
}

const (
	defaultAccessTokenTTL = 15 * time.Minute
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
