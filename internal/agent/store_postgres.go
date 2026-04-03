package agent

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
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

func (s *PostgresStore) UserExists(externalID string) bool {
	ctx, cancel := context.WithTimeout(context.Background(), dbTimeout)
	defer cancel()

	var exists bool
	err := s.pool.QueryRow(ctx,
		`SELECT EXISTS(
			SELECT 1 FROM users
			WHERE tenant_id = $1::uuid
			  AND channel = $2
			  AND external_id = $3
		)`,
		s.tenantID,
		s.channel,
		externalID,
	).Scan(&exists)
	if err != nil {
		return false
	}
	return exists
}

func (s *PostgresStore) GetUserName(externalID string) (string, bool) {
	ctx, cancel := context.WithTimeout(context.Background(), dbTimeout)
	defer cancel()

	var name *string
	err := s.pool.QueryRow(ctx,
		`SELECT NULLIF(name, '')
		 FROM users
		 WHERE tenant_id = $1::uuid
		   AND channel = $2
		   AND external_id = $3
		 ORDER BY created_at ASC
		 LIMIT 1`,
		s.tenantID,
		s.channel,
		externalID,
	).Scan(&name)
	if err != nil || name == nil || *name == "" {
		return "", false
	}
	return *name, true
}

func (s *PostgresStore) SetUserName(externalID, name string) error {
	ctx, cancel := context.WithTimeout(context.Background(), dbTimeout)
	defer cancel()

	if externalID == "" {
		return fmt.Errorf("external_id is required")
	}
	name = strings.TrimSpace(name)
	if name == "" {
		return nil
	}

	_, err := s.resolveOrCreateUser(ctx, externalID)
	if err != nil {
		return err
	}

	cmd, err := s.pool.Exec(ctx,
		`UPDATE users
		 SET name = $4,
		     updated_at = NOW()
		 WHERE tenant_id = $1::uuid
		   AND channel = $2
		   AND external_id = $3`,
		s.tenantID,
		s.channel,
		externalID,
		name,
	)
	if err != nil {
		return fmt.Errorf("set user name: %w", err)
	}
	if cmd.RowsAffected() == 0 {
		return fmt.Errorf("user not found: %s", externalID)
	}
	return nil
}

func (s *PostgresStore) GetUserForm(externalID string) (string, bool) {
	ctx, cancel := context.WithTimeout(context.Background(), dbTimeout)
	defer cancel()

	var form *string
	err := s.pool.QueryRow(ctx,
		`SELECT NULLIF(form, '')
		 FROM users
		 WHERE tenant_id = $1::uuid
		   AND channel = $2
		   AND external_id = $3
		 ORDER BY created_at ASC
		 LIMIT 1`,
		s.tenantID,
		s.channel,
		externalID,
	).Scan(&form)
	if err != nil || form == nil || *form == "" {
		return "", false
	}
	return *form, true
}

func (s *PostgresStore) SetUserForm(externalID, form string) error {
	ctx, cancel := context.WithTimeout(context.Background(), dbTimeout)
	defer cancel()

	if externalID == "" {
		return fmt.Errorf("external_id is required")
	}
	form = strings.TrimSpace(form)

	_, err := s.resolveOrCreateUser(ctx, externalID)
	if err != nil {
		return err
	}

	var cmd pgconn.CommandTag
	if form == "" {
		cmd, err = s.pool.Exec(ctx,
			`UPDATE users
			 SET form = NULL,
			     updated_at = NOW()
			 WHERE tenant_id = $1::uuid
			   AND channel = $2
			   AND external_id = $3`,
			s.tenantID,
			s.channel,
			externalID,
		)
	} else {
		cmd, err = s.pool.Exec(ctx,
			`UPDATE users
			 SET form = $4,
			     updated_at = NOW()
			 WHERE tenant_id = $1::uuid
			   AND channel = $2
			   AND external_id = $3`,
			s.tenantID,
			s.channel,
			externalID,
			form,
		)
	}
	if err != nil {
		return fmt.Errorf("set user form: %w", err)
	}
	if cmd.RowsAffected() == 0 {
		return fmt.Errorf("user not found: %s", externalID)
	}
	return nil
}

func (s *PostgresStore) GetUserPreferredLanguage(externalID string) (string, bool) {
	ctx, cancel := context.WithTimeout(context.Background(), dbTimeout)
	defer cancel()

	var lang *string
	err := s.pool.QueryRow(ctx,
		`SELECT config->>'preferred_language'
		 FROM users
		 WHERE tenant_id = $1::uuid
		   AND channel = $2
		   AND external_id = $3
		 ORDER BY created_at ASC
		 LIMIT 1`,
		s.tenantID,
		s.channel,
		externalID,
	).Scan(&lang)
	if err != nil || lang == nil || *lang == "" {
		return "", false
	}
	return *lang, true
}

func (s *PostgresStore) SetUserPreferredLanguage(externalID, lang string) error {
	ctx, cancel := context.WithTimeout(context.Background(), dbTimeout)
	defer cancel()

	if externalID == "" {
		return fmt.Errorf("external_id is required")
	}

	_, err := s.resolveOrCreateUser(ctx, externalID)
	if err != nil {
		return err
	}

	var cmd pgconn.CommandTag
	if lang == "" {
		cmd, err = s.pool.Exec(ctx,
			`UPDATE users
			 SET config = COALESCE(config, '{}'::jsonb) - 'preferred_language',
			     updated_at = NOW()
			 WHERE tenant_id = $1::uuid
			   AND channel = $2
			   AND external_id = $3`,
			s.tenantID,
			s.channel,
			externalID,
		)
	} else {
		cmd, err = s.pool.Exec(ctx,
			`UPDATE users
			 SET config = jsonb_set(COALESCE(config, '{}'::jsonb), '{preferred_language}', to_jsonb($4::text), true),
			     updated_at = NOW()
			 WHERE tenant_id = $1::uuid
			   AND channel = $2
			   AND external_id = $3`,
			s.tenantID,
			s.channel,
			externalID,
			lang,
		)
	}
	if err != nil {
		return fmt.Errorf("set preferred language: %w", err)
	}
	if cmd.RowsAffected() == 0 {
		return fmt.Errorf("user not found: %s", externalID)
	}
	return nil
}

func (s *PostgresStore) GetUserPreferredQuizIntensity(externalID string) (string, bool) {
	ctx, cancel := context.WithTimeout(context.Background(), dbTimeout)
	defer cancel()

	var intensity *string
	err := s.pool.QueryRow(ctx,
		`SELECT config->>'preferred_quiz_intensity'
		 FROM users
		 WHERE tenant_id = $1::uuid
		   AND channel = $2
		   AND external_id = $3
		 ORDER BY created_at ASC
		 LIMIT 1`,
		s.tenantID,
		s.channel,
		externalID,
	).Scan(&intensity)
	if err != nil || intensity == nil || *intensity == "" {
		return "", false
	}
	return *intensity, true
}

func (s *PostgresStore) SetUserPreferredQuizIntensity(externalID, intensity string) error {
	ctx, cancel := context.WithTimeout(context.Background(), dbTimeout)
	defer cancel()

	if externalID == "" {
		return fmt.Errorf("external_id is required")
	}

	_, err := s.resolveOrCreateUser(ctx, externalID)
	if err != nil {
		return err
	}

	var cmd pgconn.CommandTag
	if intensity == "" {
		cmd, err = s.pool.Exec(ctx,
			`UPDATE users
			 SET config = COALESCE(config, '{}'::jsonb) - 'preferred_quiz_intensity',
			     updated_at = NOW()
			 WHERE tenant_id = $1::uuid
			   AND channel = $2
			   AND external_id = $3`,
			s.tenantID,
			s.channel,
			externalID,
		)
	} else {
		cmd, err = s.pool.Exec(ctx,
			`UPDATE users
			 SET config = jsonb_set(COALESCE(config, '{}'::jsonb), '{preferred_quiz_intensity}', to_jsonb($4::text), true),
			     updated_at = NOW()
			 WHERE tenant_id = $1::uuid
			   AND channel = $2
			   AND external_id = $3`,
			s.tenantID,
			s.channel,
			externalID,
			intensity,
		)
	}
	if err != nil {
		return fmt.Errorf("set preferred quiz intensity: %w", err)
	}
	if cmd.RowsAffected() == 0 {
		return fmt.Errorf("user not found: %s", externalID)
	}
	return nil
}

func (s *PostgresStore) GetUserABGroup(externalID string) (string, bool) {
	ctx, cancel := context.WithTimeout(context.Background(), dbTimeout)
	defer cancel()

	var group *string
	err := s.pool.QueryRow(ctx,
		`SELECT config->>'ab_group'
		 FROM users
		 WHERE tenant_id = $1::uuid
		   AND channel = $2
		   AND external_id = $3
		 ORDER BY created_at ASC
		 LIMIT 1`,
		s.tenantID,
		s.channel,
		externalID,
	).Scan(&group)
	if err != nil || group == nil || *group == "" {
		return "", false
	}
	return *group, true
}

func (s *PostgresStore) SetUserABGroup(externalID, group string) error {
	ctx, cancel := context.WithTimeout(context.Background(), dbTimeout)
	defer cancel()

	if externalID == "" {
		return fmt.Errorf("external_id is required")
	}

	_, err := s.resolveOrCreateUser(ctx, externalID)
	if err != nil {
		return err
	}

	var cmd pgconn.CommandTag
	if group == "" {
		cmd, err = s.pool.Exec(ctx,
			`UPDATE users
			 SET config = COALESCE(config, '{}'::jsonb) - 'ab_group',
			     updated_at = NOW()
			 WHERE tenant_id = $1::uuid
			   AND channel = $2
			   AND external_id = $3`,
			s.tenantID,
			s.channel,
			externalID,
		)
	} else {
		cmd, err = s.pool.Exec(ctx,
			`UPDATE users
			 SET config = jsonb_set(COALESCE(config, '{}'::jsonb), '{ab_group}', to_jsonb($4::text), true),
			     updated_at = NOW()
			 WHERE tenant_id = $1::uuid
			   AND channel = $2
			   AND external_id = $3`,
			s.tenantID,
			s.channel,
			externalID,
			group,
		)
	}
	if err != nil {
		return fmt.Errorf("set ab group: %w", err)
	}
	if cmd.RowsAffected() == 0 {
		return fmt.Errorf("user not found: %s", externalID)
	}
	return nil
}

func (s *PostgresStore) GetUserTenantID(externalID string) (string, bool) {
	ctx, cancel := context.WithTimeout(context.Background(), dbTimeout)
	defer cancel()
	var tenantID *string
	err := s.pool.QueryRow(ctx,
		`SELECT tenant_id::text FROM users
		 WHERE tenant_id = $1::uuid AND channel = $2 AND external_id = $3
		 ORDER BY created_at ASC LIMIT 1`,
		s.tenantID, s.channel, externalID,
	).Scan(&tenantID)
	if err != nil || tenantID == nil || *tenantID == "" {
		return "", false
	}
	return *tenantID, true
}

func (s *PostgresStore) GetUserRole(externalID string) (string, bool) {
	ctx, cancel := context.WithTimeout(context.Background(), dbTimeout)
	defer cancel()
	var role *string
	err := s.pool.QueryRow(ctx,
		`SELECT NULLIF(role, '') FROM users
		 WHERE tenant_id = $1::uuid AND channel = $2 AND external_id = $3
		 ORDER BY created_at ASC LIMIT 1`,
		s.tenantID, s.channel, externalID,
	).Scan(&role)
	if err != nil || role == nil || *role == "" {
		return "", false
	}
	return *role, true
}

func (s *PostgresStore) GetUserInternalID(externalID string) (string, bool) {
	ctx, cancel := context.WithTimeout(context.Background(), dbTimeout)
	defer cancel()
	var id *string
	err := s.pool.QueryRow(ctx,
		`SELECT id::text FROM users
		 WHERE tenant_id = $1::uuid AND channel = $2 AND external_id = $3
		 ORDER BY created_at ASC LIMIT 1`,
		s.tenantID, s.channel, externalID,
	).Scan(&id)
	if err != nil || id == nil || *id == "" {
		return "", false
	}
	return *id, true
}

// NewPostgresStore creates a PostgreSQL-backed conversation store for the default channel.
func NewPostgresStore(ctx context.Context, pool *pgxpool.Pool) (*PostgresStore, error) {
	return NewPostgresStoreForChannel(ctx, pool, defaultChannel)
}

// NewPostgresStoreForChannel creates a PostgreSQL-backed conversation store for a specific channel.
func NewPostgresStoreForChannel(ctx context.Context, pool *pgxpool.Pool, channel string) (*PostgresStore, error) {
	if pool == nil {
		return nil, fmt.Errorf("pool is nil")
	}
	channel = strings.TrimSpace(channel)
	if channel == "" {
		channel = defaultChannel
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
		channel:  channel,
	}, nil
}

// TenantID returns the resolved tenant UUID for this store.
func (s *PostgresStore) TenantID() string { return s.tenantID }

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
		if _, err := s.AddMessage(id, msg); err != nil {
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
		`SELECT id::text, role, content, model, input_tokens, output_tokens, created_at
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
			&msg.ID,
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

	full, err := s.GetConversation(conv.ID)
	if err != nil {
		return nil, false
	}
	return full, true
}

func (s *PostgresStore) AddMessage(conversationID string, msg StoredMessage) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), dbTimeout)
	defer cancel()

	createdAt := msg.CreatedAt
	if createdAt.IsZero() {
		createdAt = time.Now()
	}

	if msg.Role == "" {
		return "", fmt.Errorf("message role is required")
	}
	if msg.Content == "" {
		return "", fmt.Errorf("message content is required")
	}

	var id string
	err := s.pool.QueryRow(ctx,
		`INSERT INTO messages (conversation_id, tenant_id, role, content, model, input_tokens, output_tokens, created_at)
		 SELECT $1::uuid, c.tenant_id, $2, $3, $4, $5, $6, $7
		 FROM conversations c
		 WHERE c.id = $1::uuid
		 RETURNING id::text`,
		conversationID,
		msg.Role,
		msg.Content,
		nullIfEmpty(msg.Model),
		nullIfZero(msg.InputTokens),
		nullIfZero(msg.OutputTokens),
		createdAt,
	).Scan(&id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return "", fmt.Errorf("conversation not found: %s", conversationID)
		}
		return "", fmt.Errorf("insert message: %w", err)
	}

	return id, nil
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

func (s *PostgresStore) UpdateConversationState(conversationID string, state string) error {
	ctx, cancel := context.WithTimeout(context.Background(), dbTimeout)
	defer cancel()

	if state == "" {
		return fmt.Errorf("state is required")
	}

	cmd, err := s.pool.Exec(ctx,
		`UPDATE conversations
		 SET state = $2
		 WHERE id = $1::uuid`,
		conversationID,
		state,
	)
	if err != nil {
		return fmt.Errorf("update conversation state: %w", err)
	}
	if cmd.RowsAffected() == 0 {
		return fmt.Errorf("conversation not found: %s", conversationID)
	}

	return nil
}

func (s *PostgresStore) UpdateConversationTopicID(conversationID, topicID string) error {
	ctx, cancel := context.WithTimeout(context.Background(), dbTimeout)
	defer cancel()

	cmd, err := s.pool.Exec(ctx,
		`UPDATE conversations
		 SET topic_id = $2
		 WHERE id = $1::uuid`,
		conversationID,
		nullIfEmpty(topicID),
	)
	if err != nil {
		return fmt.Errorf("update conversation topic_id: %w", err)
	}
	if cmd.RowsAffected() == 0 {
		return fmt.Errorf("conversation not found: %s", conversationID)
	}

	return nil
}

func (s *PostgresStore) UpdateConversationPendingQuiz(conversationID, state, topicID string) error {
	ctx, cancel := context.WithTimeout(context.Background(), dbTimeout)
	defer cancel()

	if state == "" {
		return fmt.Errorf("state is required")
	}

	cmd, err := s.pool.Exec(ctx,
		`UPDATE conversations
		 SET state = $2,
		     metadata = ((jsonb_set(COALESCE(metadata, '{}'::jsonb), '{pending_quiz_topic_id}', to_jsonb($3::text), true) - 'quiz_state') - 'pending_goal')
		 WHERE id = $1::uuid`,
		conversationID,
		state,
		topicID,
	)
	if err != nil {
		return fmt.Errorf("update pending quiz state: %w", err)
	}
	if cmd.RowsAffected() == 0 {
		return fmt.Errorf("conversation not found: %s", conversationID)
	}

	return nil
}

func (s *PostgresStore) UpdateConversationQuizState(conversationID, state string, quizState ConversationQuizState) error {
	ctx, cancel := context.WithTimeout(context.Background(), dbTimeout)
	defer cancel()

	if state == "" {
		return fmt.Errorf("state is required")
	}
	payload, err := json.Marshal(quizState)
	if err != nil {
		return fmt.Errorf("marshal quiz state: %w", err)
	}

	cmd, err := s.pool.Exec(ctx,
		`UPDATE conversations
		 SET state = $2,
		     metadata = (((jsonb_set(COALESCE(metadata, '{}'::jsonb), '{quiz_state}', $3::jsonb, true) - 'pending_quiz_topic_id') - 'pending_goal'))
		 WHERE id = $1::uuid`,
		conversationID,
		state,
		payload,
	)
	if err != nil {
		return fmt.Errorf("update active quiz state: %w", err)
	}
	if cmd.RowsAffected() == 0 {
		return fmt.Errorf("conversation not found: %s", conversationID)
	}

	return nil
}

func (s *PostgresStore) ClearConversationQuizState(conversationID, state string) error {
	ctx, cancel := context.WithTimeout(context.Background(), dbTimeout)
	defer cancel()

	if state == "" {
		return fmt.Errorf("state is required")
	}

	cmd, err := s.pool.Exec(ctx,
		`UPDATE conversations
		 SET state = $2,
		     metadata = ((COALESCE(metadata, '{}'::jsonb) - 'pending_quiz_topic_id') - 'quiz_state')
		 WHERE id = $1::uuid`,
		conversationID,
		state,
	)
	if err != nil {
		return fmt.Errorf("clear quiz state: %w", err)
	}
	if cmd.RowsAffected() == 0 {
		return fmt.Errorf("conversation not found: %s", conversationID)
	}

	return nil
}

func (s *PostgresStore) SetConversationPendingGoal(conversationID string, goal PendingGoalDraft) error {
	ctx, cancel := context.WithTimeout(context.Background(), dbTimeout)
	defer cancel()

	payload, err := json.Marshal(goal)
	if err != nil {
		return fmt.Errorf("marshal pending goal: %w", err)
	}

	cmd, err := s.pool.Exec(ctx,
		`UPDATE conversations
		 SET metadata = jsonb_set(COALESCE(metadata, '{}'::jsonb), '{pending_goal}', $2::jsonb, true)
		 WHERE id = $1::uuid`,
		conversationID,
		payload,
	)
	if err != nil {
		return fmt.Errorf("set pending goal: %w", err)
	}
	if cmd.RowsAffected() == 0 {
		return fmt.Errorf("conversation not found: %s", conversationID)
	}
	return nil
}

func (s *PostgresStore) ClearConversationPendingGoal(conversationID string) error {
	ctx, cancel := context.WithTimeout(context.Background(), dbTimeout)
	defer cancel()

	cmd, err := s.pool.Exec(ctx,
		`UPDATE conversations
		 SET metadata = COALESCE(metadata, '{}'::jsonb) - 'pending_goal'
		 WHERE id = $1::uuid`,
		conversationID,
	)
	if err != nil {
		return fmt.Errorf("clear pending goal: %w", err)
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
	metadata := parseConversationMetadata(metadataBytes)
	conv.Summary = metadata.Summary
	conv.CompactedAt = metadata.CompactedAt
	conv.PendingQuizTopicID = metadata.PendingQuizTopicID
	conv.QuizState = metadata.QuizState
	conv.PendingGoal = metadata.PendingGoal

	return conv, nil
}

type conversationMetadata struct {
	Summary            string                 `json:"summary,omitempty"`
	CompactedAt        int                    `json:"compacted_at,omitempty"`
	PendingQuizTopicID string                 `json:"pending_quiz_topic_id,omitempty"`
	QuizState          *ConversationQuizState `json:"quiz_state,omitempty"`
	PendingGoal        *PendingGoalDraft      `json:"pending_goal,omitempty"`
}

func parseConversationMetadata(metadata []byte) conversationMetadata {
	if len(metadata) == 0 {
		return conversationMetadata{}
	}
	var parsed conversationMetadata
	if err := json.Unmarshal(metadata, &parsed); err != nil {
		return conversationMetadata{}
	}
	return parsed
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
