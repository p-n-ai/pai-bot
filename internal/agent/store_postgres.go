package agent

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

const (
	defaultTenantSlug = "default"
	defaultChannel    = "telegram"
	dbTimeout         = 5 * time.Second
)

// PostgresStore is a PostgreSQL-backed ConversationStore implementation.
type PostgresStore struct {
	pool     *pgxpool.Pool
	tenantID string
	channel  string
}

// NewPostgresStore creates a PostgreSQL-backed conversation store for the default tenant.
func NewPostgresStore(ctx context.Context, pool *pgxpool.Pool) (*PostgresStore, error) {
	if pool == nil {
		return nil, fmt.Errorf("pool is nil")
	}

	var tenantID string
	if err := pool.QueryRow(ctx,
		`SELECT id::text FROM tenants WHERE slug = $1 LIMIT 1`,
		defaultTenantSlug,
	).Scan(&tenantID); err != nil {
		return nil, fmt.Errorf("find default tenant: %w", err)
	}

	return &PostgresStore{
		pool:     pool,
		tenantID: tenantID,
		channel:  defaultChannel,
	}, nil
}

func (s *PostgresStore) CreateConversation(conv Conversation) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), dbTimeout)
	defer cancel()

	if conv.UserID == "" {
		return "", fmt.Errorf("user_id is required")
	}

	userID, err := s.resolveOrCreateUser(ctx, conv.UserID)
	if err != nil {
		return "", err
	}

	state := conv.State
	if state == "" {
		state = "teaching"
	}

	startedAt := conv.StartedAt
	if startedAt.IsZero() {
		startedAt = time.Now()
	}

	var id string
	var dbStartedAt time.Time
	err = s.pool.QueryRow(ctx,
		`INSERT INTO conversations (user_id, tenant_id, topic_id, state, started_at)
		 VALUES ($1::uuid, $2::uuid, $3, $4, $5)
		 RETURNING id::text, started_at`,
		userID,
		s.tenantID,
		nullIfEmpty(conv.TopicID),
		state,
		startedAt,
	).Scan(&id, &dbStartedAt)
	if err != nil {
		return "", fmt.Errorf("create conversation: %w", err)
	}

	for _, msg := range conv.Messages {
		if err := s.AddMessage(id, msg); err != nil {
			return "", fmt.Errorf("save initial messages: %w", err)
		}
	}

	_ = dbStartedAt
	return id, nil
}

func (s *PostgresStore) GetConversation(id string) (*Conversation, error) {
	ctx, cancel := context.WithTimeout(context.Background(), dbTimeout)
	defer cancel()

	conv, err := s.getConversationByQuery(ctx,
		`SELECT c.id::text, u.external_id, c.topic_id, c.state, c.started_at, c.ended_at, c.metadata
		 FROM conversations c
		 JOIN users u ON u.id = c.user_id
		 WHERE c.id = $1::uuid
		 LIMIT 1`,
		id,
	)
	if err != nil {
		return nil, err
	}

	rows, err := s.pool.Query(ctx,
		`SELECT role, content, model, input_tokens, output_tokens, created_at
		 FROM messages
		 WHERE conversation_id = $1::uuid
		 ORDER BY created_at ASC`,
		id,
	)
	if err != nil {
		return nil, fmt.Errorf("query messages: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var msg StoredMessage
		var model *string
		var inputTokens *int
		var outputTokens *int
		if err := rows.Scan(
			&msg.Role,
			&msg.Content,
			&model,
			&inputTokens,
			&outputTokens,
			&msg.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan message: %w", err)
		}
		if model != nil {
			msg.Model = *model
		}
		if inputTokens != nil {
			msg.InputTokens = *inputTokens
		}
		if outputTokens != nil {
			msg.OutputTokens = *outputTokens
		}
		conv.Messages = append(conv.Messages, msg)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate messages: %w", err)
	}

	return conv, nil
}

func (s *PostgresStore) GetActiveConversation(userID string) (*Conversation, bool) {
	ctx, cancel := context.WithTimeout(context.Background(), dbTimeout)
	defer cancel()

	conv, err := s.getConversationByQuery(ctx,
		`SELECT c.id::text, u.external_id, c.topic_id, c.state, c.started_at, c.ended_at, c.metadata
		 FROM conversations c
		 JOIN users u ON u.id = c.user_id
		 WHERE u.external_id = $1
		   AND u.channel = $2
		   AND c.tenant_id = $3::uuid
		   AND c.ended_at IS NULL
		 ORDER BY c.started_at DESC
		 LIMIT 1`,
		userID,
		s.channel,
		s.tenantID,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, false
		}
		return nil, false
	}

	return conv, true
}

func (s *PostgresStore) AddMessage(conversationID string, msg StoredMessage) error {
	ctx, cancel := context.WithTimeout(context.Background(), dbTimeout)
	defer cancel()

	createdAt := msg.CreatedAt
	if createdAt.IsZero() {
		createdAt = time.Now()
	}

	if msg.Role == "" {
		return fmt.Errorf("message role is required")
	}
	if msg.Content == "" {
		return fmt.Errorf("message content is required")
	}

	cmd, err := s.pool.Exec(ctx,
		`INSERT INTO messages (conversation_id, tenant_id, role, content, model, input_tokens, output_tokens, created_at)
		 SELECT $1::uuid, c.tenant_id, $2, $3, $4, $5, $6, $7
		 FROM conversations c
		 WHERE c.id = $1::uuid`,
		conversationID,
		msg.Role,
		msg.Content,
		nullIfEmpty(msg.Model),
		nullIfZero(msg.InputTokens),
		nullIfZero(msg.OutputTokens),
		createdAt,
	)
	if err != nil {
		return fmt.Errorf("insert message: %w", err)
	}
	if cmd.RowsAffected() == 0 {
		return fmt.Errorf("conversation not found: %s", conversationID)
	}

	return nil
}

func (s *PostgresStore) SetSummary(conversationID string, summary string, compactedAt int) error {
	ctx, cancel := context.WithTimeout(context.Background(), dbTimeout)
	defer cancel()

	cmd, err := s.pool.Exec(ctx,
		`UPDATE conversations
		 SET metadata = jsonb_set(
		   jsonb_set(COALESCE(metadata, '{}'::jsonb), '{summary}', to_jsonb($2::text), true),
		   '{compacted_at}',
		   to_jsonb($3::int),
		   true
		 )
		 WHERE id = $1::uuid`,
		conversationID,
		summary,
		compactedAt,
	)
	if err != nil {
		return fmt.Errorf("set summary: %w", err)
	}
	if cmd.RowsAffected() == 0 {
		return fmt.Errorf("conversation not found: %s", conversationID)
	}

	return nil
}

func (s *PostgresStore) EndConversation(id string) error {
	ctx, cancel := context.WithTimeout(context.Background(), dbTimeout)
	defer cancel()

	cmd, err := s.pool.Exec(ctx,
		`UPDATE conversations
		 SET ended_at = NOW()
		 WHERE id = $1::uuid`,
		id,
	)
	if err != nil {
		return fmt.Errorf("end conversation: %w", err)
	}
	if cmd.RowsAffected() == 0 {
		return fmt.Errorf("conversation not found: %s", id)
	}

	return nil
}

func (s *PostgresStore) resolveOrCreateUser(ctx context.Context, externalID string) (string, error) {
	var userID string
	err := s.pool.QueryRow(ctx,
		`SELECT id::text
		 FROM users
		 WHERE tenant_id = $1::uuid
		   AND channel = $2
		   AND external_id = $3
		 ORDER BY created_at ASC
		 LIMIT 1`,
		s.tenantID,
		s.channel,
		externalID,
	).Scan(&userID)
	if err == nil {
		return userID, nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return "", fmt.Errorf("lookup user: %w", err)
	}

	name := fmt.Sprintf("Student %s", externalID)
	err = s.pool.QueryRow(ctx,
		`INSERT INTO users (tenant_id, role, name, external_id, channel)
		 VALUES ($1::uuid, 'student', $2, $3, $4)
		 RETURNING id::text`,
		s.tenantID,
		name,
		externalID,
		s.channel,
	).Scan(&userID)
	if err != nil {
		return "", fmt.Errorf("create user: %w", err)
	}

	return userID, nil
}

func (s *PostgresStore) getConversationByQuery(ctx context.Context, query string, args ...any) (*Conversation, error) {
	conv := &Conversation{}
	var topicID *string
	var endedAt *time.Time
	var metadataBytes []byte

	err := s.pool.QueryRow(ctx, query, args...).Scan(
		&conv.ID,
		&conv.UserID,
		&topicID,
		&conv.State,
		&conv.StartedAt,
		&endedAt,
		&metadataBytes,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, pgx.ErrNoRows
		}
		return nil, fmt.Errorf("get conversation: %w", err)
	}

	if topicID != nil {
		conv.TopicID = *topicID
	}
	conv.EndedAt = endedAt
	conv.Messages = []StoredMessage{}
	conv.Summary, conv.CompactedAt = parseConversationMetadata(metadataBytes)

	return conv, nil
}

func parseConversationMetadata(metadata []byte) (string, int) {
	if len(metadata) == 0 {
		return "", 0
	}
	var raw map[string]any
	if err := json.Unmarshal(metadata, &raw); err != nil {
		return "", 0
	}

	summary, _ := raw["summary"].(string)
	compactedAt := 0
	if v, ok := raw["compacted_at"]; ok {
		switch n := v.(type) {
		case float64:
			compactedAt = int(n)
		case int:
			compactedAt = n
		}
	}

	return summary, compactedAt
}

func nullIfZero(v int) any {
	if v == 0 {
		return nil
	}
	return v
}

func nullIfEmpty(v string) any {
	if v == "" {
		return nil
	}
	return v
}
