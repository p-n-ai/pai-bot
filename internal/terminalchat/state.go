// Copyright 2026 the P&AI authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package terminalchat

import (
	"context"
	"fmt"

	"github.com/p-n-ai/pai-bot/internal/agent"
	"github.com/p-n-ai/pai-bot/internal/platform/config"
	"github.com/p-n-ai/pai-bot/internal/platform/database"
	"github.com/p-n-ai/pai-bot/internal/progress"
)

// StateOptions controls whether the terminal chat uses persistent or in-memory state.
type StateOptions struct {
	Memory  bool
	Channel string
}

// State bundles the dependencies needed by the engine for session state.
type State struct {
	Store       agent.ConversationStore
	Tracker     progress.Tracker
	EventLogger agent.EventLogger
	DB          *database.DB
	TenantID    string
}

// StateDeps allows tests to substitute the persistence builders.
type StateDeps struct {
	OpenDB                 func(context.Context, string, int, int) (*database.DB, error)
	NewPostgresStore       func(context.Context, *database.DB) (agent.ConversationStore, string, error)
	NewPostgresTracker     func(*database.DB, string) progress.Tracker
	NewPostgresEventLogger func(*database.DB) agent.EventLogger
}

// BuildState constructs terminal chat state dependencies.
// Persistent PostgreSQL-backed state is the default; in-memory mode must be explicit.
func BuildState(ctx context.Context, cfg *config.Config, opts StateOptions, deps StateDeps) (State, func(), error) {
	if opts.Memory {
		return State{
			Store:       agent.NewMemoryStore(),
			Tracker:     progress.NewMemoryTracker(),
			EventLogger: agent.NewMemoryEventLogger(),
			DB:          nil,
			TenantID:    "",
		}, func() {}, nil
	}

	openDB := deps.OpenDB
	if openDB == nil {
		openDB = database.New
	}

	newPostgresStore := deps.NewPostgresStore
	if newPostgresStore == nil {
		newPostgresStore = func(ctx context.Context, db *database.DB) (agent.ConversationStore, string, error) {
			store, err := agent.NewPostgresStoreForChannel(ctx, db.Pool, opts.Channel)
			if err != nil {
				return nil, "", err
			}
			return store, store.TenantID(), nil
		}
	}

	newPostgresTracker := deps.NewPostgresTracker
	if newPostgresTracker == nil {
		newPostgresTracker = func(db *database.DB, tenantID string) progress.Tracker {
			return progress.NewPostgresTracker(db.Pool, tenantID)
		}
	}

	newPostgresEventLogger := deps.NewPostgresEventLogger
	if newPostgresEventLogger == nil {
		newPostgresEventLogger = func(db *database.DB) agent.EventLogger {
			return agent.NewPostgresEventLogger(db.Pool)
		}
	}

	db, err := openDB(ctx, cfg.Database.URL, cfg.Database.MaxConns, cfg.Database.MinConns)
	if err != nil {
		return State{}, nil, fmt.Errorf("connect database: %w", err)
	}

	store, tenantID, err := newPostgresStore(ctx, db)
	if err != nil {
		db.Close()
		return State{}, nil, fmt.Errorf("create postgres store: %w", err)
	}

	cleanup := func() {
		if db != nil && db.Pool != nil {
			db.Close()
		}
	}

	return State{
		Store:       store,
		Tracker:     newPostgresTracker(db, tenantID),
		EventLogger: newPostgresEventLogger(db),
		DB:          db,
		TenantID:    tenantID,
	}, cleanup, nil
}
