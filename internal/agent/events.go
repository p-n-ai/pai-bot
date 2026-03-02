package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Event represents an analytics event persisted to the events table.
type Event struct {
	ConversationID string
	UserID         string
	EventType      string
	Data           map[string]any
	CreatedAt      time.Time
}

// EventLogger defines event logging behavior.
type EventLogger interface {
	LogEvent(event Event) error
}

// NopEventLogger ignores all events.
type NopEventLogger struct{}

func (NopEventLogger) LogEvent(Event) error {
	return nil
}

// MemoryEventLogger stores events in memory for tests.
type MemoryEventLogger struct {
	mu     sync.Mutex
	events []Event
}

func NewMemoryEventLogger() *MemoryEventLogger {
	return &MemoryEventLogger{
		events: []Event{},
	}
}

func (l *MemoryEventLogger) LogEvent(event Event) error {
	if event.EventType == "" {
		return fmt.Errorf("event_type is required")
	}
	if event.CreatedAt.IsZero() {
		event.CreatedAt = time.Now()
	}

	l.mu.Lock()
	l.events = append(l.events, event)
	l.mu.Unlock()

	return nil
}

func (l *MemoryEventLogger) Events() []Event {
	l.mu.Lock()
	defer l.mu.Unlock()
	return append([]Event{}, l.events...)
}

// PostgresEventLogger inserts events into the events table.
type PostgresEventLogger struct {
	pool *pgxpool.Pool
}

func NewPostgresEventLogger(pool *pgxpool.Pool) *PostgresEventLogger {
	return &PostgresEventLogger{pool: pool}
}

func (l *PostgresEventLogger) LogEvent(event Event) error {
	if l == nil || l.pool == nil {
		return fmt.Errorf("event logger pool is nil")
	}
	if event.EventType == "" {
		return fmt.Errorf("event_type is required")
	}
	if event.ConversationID == "" {
		return fmt.Errorf("conversation_id is required")
	}

	payload := event.Data
	if payload == nil {
		payload = map[string]any{}
	}
	data, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal event data: %w", err)
	}

	createdAt := event.CreatedAt
	if createdAt.IsZero() {
		createdAt = time.Now()
	}

	ctx, cancel := context.WithTimeout(context.Background(), dbTimeout)
	defer cancel()

	cmd, err := l.pool.Exec(ctx,
		`INSERT INTO events (tenant_id, user_id, event_type, data, created_at)
		 SELECT c.tenant_id, c.user_id, $2, $3::jsonb, $4
		 FROM conversations c
		 WHERE c.id = $1::uuid`,
		event.ConversationID,
		event.EventType,
		string(data),
		createdAt,
	)
	if err != nil {
		return fmt.Errorf("insert event: %w", err)
	}
	if cmd.RowsAffected() == 0 {
		return fmt.Errorf("conversation not found: %s", event.ConversationID)
	}

	slog.Debug("event logged",
		"type", event.EventType,
		"conversation_id", event.ConversationID,
		"user_id", event.UserID,
	)
	return nil
}
