package main

import (
	"context"
	"flag"
	"log/slog"
	"os"

	"github.com/p-n-ai/pai-bot/internal/platform/config"
	"github.com/p-n-ai/pai-bot/internal/platform/database"
	"github.com/p-n-ai/pai-bot/internal/platform/seed"
)

type seedMode string

const demoSeedMode seedMode = "demo"

func (m seedMode) String() string {
	return string(m)
}

func main() {
	var mode string
	flag.StringVar(&mode, "mode", demoSeedMode.String(), "seed mode")
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
	default:
		slog.Error("unsupported seed mode", "mode", mode)
		os.Exit(1)
	}
}
