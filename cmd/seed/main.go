// Copyright 2026 the P&AI authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/p-n-ai/pai-bot/internal/platform/config"
	"github.com/p-n-ai/pai-bot/internal/platform/database"
	"github.com/p-n-ai/pai-bot/internal/platform/seed"
)

type seedMode string

const demoSeedMode seedMode = "demo"
const tokenBudgetSeedMode seedMode = "token-budget"

func (m seedMode) String() string {
	return string(m)
}

func main() {
	var mode string
	var tenantSlug string
	var budgetTokens int64
	var periodStart string
	var periodEnd string
	flag.StringVar(&mode, "mode", demoSeedMode.String(), "seed mode")
	flag.StringVar(&tenantSlug, "tenant-slug", "default", "tenant slug for token-budget mode")
	flag.Int64Var(&budgetTokens, "budget-tokens", 5000, "budget tokens for token-budget mode")
	flag.StringVar(&periodStart, "period-start", "", "RFC3339 timestamp for token-budget mode; defaults to the current UTC month start")
	flag.StringVar(&periodEnd, "period-end", "", "RFC3339 timestamp for token-budget mode; defaults to the next UTC month start")
	flag.Parse()

	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	slog.SetDefault(logger)

	cfg, err := config.Load()
	if err != nil {
		slog.Error("failed to load config", "error", err)
		os.Exit(1)
	}

	db, err := database.New(context.Background(), cfg.Database.URL, cfg.Database.MaxConns, cfg.Database.MinConns)
	if err != nil {
		slog.Error("failed to connect to database", "error", err)
		os.Exit(1)
	}
	defer db.Close()

	switch seedMode(mode) {
	case demoSeedMode:
		if err := seed.SeedDemo(context.Background(), db.Pool); err != nil {
			slog.Error("failed to seed demo data", "error", err)
			os.Exit(1)
		}
		slog.Info("demo data seeded")
	case tokenBudgetSeedMode:
		start, end, err := parseBudgetWindow(periodStart, periodEnd)
		if err != nil {
			slog.Error("failed to parse token budget window", "error", err)
			os.Exit(1)
		}
		if err := seed.SeedTokenBudget(context.Background(), db.Pool, seed.TokenBudgetSeedParams{
			TenantSlug:   tenantSlug,
			BudgetTokens: budgetTokens,
			PeriodStart:  start,
			PeriodEnd:    end,
		}); err != nil {
			slog.Error("failed to seed token budget", "error", err, "tenant_slug", tenantSlug)
			os.Exit(1)
		}
		slog.Info("token budget seeded", "tenant_slug", tenantSlug, "budget_tokens", budgetTokens, "period_start", start, "period_end", end)
	default:
		slog.Error("unsupported seed mode", "mode", mode)
		os.Exit(1)
	}
}

func parseBudgetWindow(startRaw, endRaw string) (time.Time, time.Time, error) {
	now := time.Now().UTC()
	defaultStart := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)
	defaultEnd := defaultStart.AddDate(0, 1, 0)

	start := defaultStart
	if startRaw != "" {
		parsed, err := time.Parse(time.RFC3339, startRaw)
		if err != nil {
			return time.Time{}, time.Time{}, err
		}
		start = parsed.UTC()
	}

	end := defaultEnd
	if endRaw != "" {
		parsed, err := time.Parse(time.RFC3339, endRaw)
		if err != nil {
			return time.Time{}, time.Time{}, err
		}
		end = parsed.UTC()
	}

	if !end.After(start) {
		return time.Time{}, time.Time{}, fmt.Errorf("period end must be after period start")
	}

	return start, end, nil
}
