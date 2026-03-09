package terminalchat

import (
	"context"
	"testing"

	"github.com/p-n-ai/pai-bot/internal/agent"
	"github.com/p-n-ai/pai-bot/internal/platform/config"
	"github.com/p-n-ai/pai-bot/internal/platform/database"
	"github.com/p-n-ai/pai-bot/internal/progress"
)

func TestBuildState_DefaultsToPersistentMode(t *testing.T) {
	var openDBCalled bool
	var postgresStoreCalled bool
	var postgresTrackerCalled bool
	var postgresLoggerCalled bool

	state, cleanup, err := BuildState(context.Background(), &config.Config{
		Database: config.DatabaseConfig{
			URL:      "postgres://example",
			MaxConns: 5,
			MinConns: 1,
		},
	}, StateOptions{}, StateDeps{
		OpenDB: func(context.Context, string, int, int) (*database.DB, error) {
			openDBCalled = true
			return &database.DB{}, nil
		},
		NewPostgresStore: func(context.Context, *database.DB) (agent.ConversationStore, string, error) {
			postgresStoreCalled = true
			return agent.NewMemoryStore(), "tenant-1", nil
		},
		NewPostgresTracker: func(*database.DB, string) progress.Tracker {
			postgresTrackerCalled = true
			return progress.NewMemoryTracker()
		},
		NewPostgresEventLogger: func(*database.DB) agent.EventLogger {
			postgresLoggerCalled = true
			return agent.NewMemoryEventLogger()
		},
	})
	if err != nil {
		t.Fatalf("BuildState() error = %v", err)
	}
	defer cleanup()

	if !openDBCalled || !postgresStoreCalled || !postgresTrackerCalled || !postgresLoggerCalled {
		t.Fatalf("expected persistent builders to be used, got openDB=%v store=%v tracker=%v logger=%v",
			openDBCalled, postgresStoreCalled, postgresTrackerCalled, postgresLoggerCalled)
	}
	if state.Store == nil || state.Tracker == nil || state.EventLogger == nil {
		t.Fatal("BuildState() returned nil persistent dependencies")
	}
	if state.DB == nil || state.TenantID != "tenant-1" {
		t.Fatalf("state DB/TenantID = %#v / %q, want non-nil DB and tenant-1", state.DB, state.TenantID)
	}
}

func TestBuildState_MemoryModeSkipsDatabase(t *testing.T) {
	var openDBCalled bool

	state, cleanup, err := BuildState(context.Background(), &config.Config{}, StateOptions{
		Memory: true,
	}, StateDeps{
		OpenDB: func(context.Context, string, int, int) (*database.DB, error) {
			openDBCalled = true
			return &database.DB{}, nil
		},
	})
	if err != nil {
		t.Fatalf("BuildState() error = %v", err)
	}
	defer cleanup()

	if openDBCalled {
		t.Fatal("OpenDB should not be called in memory mode")
	}
	if state.Store == nil || state.Tracker == nil || state.EventLogger == nil {
		t.Fatal("BuildState() returned nil memory dependencies")
	}
	if state.DB != nil || state.TenantID != "" {
		t.Fatalf("memory mode DB/TenantID = %#v / %q, want nil and empty", state.DB, state.TenantID)
	}
}
