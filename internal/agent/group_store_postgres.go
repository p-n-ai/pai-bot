package agent

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"math/big"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

const joinCodeLen = 6

// PostgresGroupStore implements GroupStore backed by PostgreSQL.
type PostgresGroupStore struct {
	pool *pgxpool.Pool
}

// NewPostgresGroupStore creates a new PostgreSQL-backed group store.
func NewPostgresGroupStore(pool *pgxpool.Pool) *PostgresGroupStore {
	return &PostgresGroupStore{pool: pool}
}

func (s *PostgresGroupStore) CreateGroup(tenantID, name, groupType, description, syllabus, subject, cadence, createdByUserID string) (*Group, error) {
	ctx, cancel := context.WithTimeout(context.Background(), dbTimeout)
	defer cancel()

	code, err := generateJoinCode()
	if err != nil {
		return nil, fmt.Errorf("generate join code: %w", err)
	}

	var createdBy *string
	if createdByUserID != "" {
		createdBy = &createdByUserID
	}

	g := &Group{}
	err = s.pool.QueryRow(ctx,
		`INSERT INTO groups (tenant_id, name, type, description, syllabus, subject, cadence, join_code, created_by)
		 VALUES ($1::uuid, $2, $3, $4, $5, $6, $7, $8, $9)
		 RETURNING id::text, tenant_id::text, name, type, description, syllabus, subject, cadence, join_code,
		           COALESCE(created_by::text, ''), created_at, updated_at`,
		tenantID, name, groupType, description, syllabus, subject, cadence, code, createdBy,
	).Scan(&g.ID, &g.TenantID, &g.Name, &g.Type, &g.Description, &g.Syllabus, &g.Subject, &g.Cadence,
		&g.JoinCode, &g.CreatedBy, &g.CreatedAt, &g.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("create group: %w", err)
	}
	return g, nil
}

func (s *PostgresGroupStore) GetGroupByID(id string) (*Group, error) {
	ctx, cancel := context.WithTimeout(context.Background(), dbTimeout)
	defer cancel()
	return s.scanGroup(ctx, `
		SELECT g.id::text, g.tenant_id::text, g.name, g.type, g.description, g.syllabus, g.subject, g.cadence,
		       g.join_code, COALESCE(g.created_by::text, ''), g.created_at, g.updated_at,
		       (SELECT COUNT(*) FROM group_members gm WHERE gm.group_id = g.id)::int,
		       g.closed
		FROM groups g WHERE g.id = $1::uuid`, id)
}

func (s *PostgresGroupStore) GetGroupByJoinCode(code string) (*Group, error) {
	ctx, cancel := context.WithTimeout(context.Background(), dbTimeout)
	defer cancel()
	return s.scanGroup(ctx, `
		SELECT g.id::text, g.tenant_id::text, g.name, g.type, g.description, g.syllabus, g.subject, g.cadence,
		       g.join_code, COALESCE(g.created_by::text, ''), g.created_at, g.updated_at,
		       (SELECT COUNT(*) FROM group_members gm WHERE gm.group_id = g.id)::int,
		       g.closed
		FROM groups g WHERE g.join_code = $1`, strings.ToUpper(strings.TrimSpace(code)))
}

func (s *PostgresGroupStore) scanGroup(ctx context.Context, query string, args ...any) (*Group, error) {
	g := &Group{}
	err := s.pool.QueryRow(ctx, query, args...).Scan(
		&g.ID, &g.TenantID, &g.Name, &g.Type, &g.Description, &g.Syllabus, &g.Subject, &g.Cadence,
		&g.JoinCode, &g.CreatedBy, &g.CreatedAt, &g.UpdatedAt, &g.MemberCount, &g.Closed,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("scan group: %w", err)
	}
	return g, nil
}

func (s *PostgresGroupStore) UpdateGroup(id string, input UpdateGroupInput) (*Group, error) {
	ctx, cancel := context.WithTimeout(context.Background(), dbTimeout)
	defer cancel()

	setClauses := []string{"updated_at = NOW()"}
	args := []any{id}
	argIdx := 2

	if input.Name != nil {
		setClauses = append(setClauses, fmt.Sprintf("name = $%d", argIdx))
		args = append(args, *input.Name)
		argIdx++
	}
	if input.Description != nil {
		setClauses = append(setClauses, fmt.Sprintf("description = $%d", argIdx))
		args = append(args, *input.Description)
		argIdx++
	}
	if input.Syllabus != nil {
		setClauses = append(setClauses, fmt.Sprintf("syllabus = $%d", argIdx))
		args = append(args, *input.Syllabus)
		argIdx++
	}
	if input.Subject != nil {
		setClauses = append(setClauses, fmt.Sprintf("subject = $%d", argIdx))
		args = append(args, *input.Subject)
		argIdx++
	}
	if input.Cadence != nil {
		setClauses = append(setClauses, fmt.Sprintf("cadence = $%d", argIdx))
		args = append(args, *input.Cadence)
		argIdx++
	}
	if input.Closed != nil {
		setClauses = append(setClauses, fmt.Sprintf("closed = $%d", argIdx))
		args = append(args, *input.Closed)
		argIdx++
	}

	query := fmt.Sprintf(
		`UPDATE groups SET %s WHERE id = $1::uuid
		 RETURNING id::text, tenant_id::text, name, type, description, syllabus, subject, cadence,
		           join_code, COALESCE(created_by::text, ''), created_at, updated_at, closed`,
		strings.Join(setClauses, ", "),
	)

	g := &Group{}
	err := s.pool.QueryRow(ctx, query, args...).Scan(
		&g.ID, &g.TenantID, &g.Name, &g.Type, &g.Description, &g.Syllabus, &g.Subject, &g.Cadence,
		&g.JoinCode, &g.CreatedBy, &g.CreatedAt, &g.UpdatedAt, &g.Closed,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("update group: %w", err)
	}
	return g, nil
}

func (s *PostgresGroupStore) DeleteGroup(id string) error {
	ctx, cancel := context.WithTimeout(context.Background(), dbTimeout)
	defer cancel()

	cmd, err := s.pool.Exec(ctx, `DELETE FROM groups WHERE id = $1::uuid`, id)
	if err != nil {
		return fmt.Errorf("delete group: %w", err)
	}
	if cmd.RowsAffected() == 0 {
		return fmt.Errorf("group not found: %s", id)
	}
	return nil
}

func (s *PostgresGroupStore) JoinGroup(groupID, userID, tenantID, role string) error {
	ctx, cancel := context.WithTimeout(context.Background(), dbTimeout)
	defer cancel()

	// Check if group is closed.
	var closed bool
	err := s.pool.QueryRow(ctx, `SELECT closed FROM groups WHERE id = $1::uuid`, groupID).Scan(&closed)
	if errors.Is(err, pgx.ErrNoRows) {
		return fmt.Errorf("group not found: %s", groupID)
	}
	if err != nil {
		return fmt.Errorf("check group closed: %w", err)
	}
	if closed {
		return ErrGroupClosed
	}

	_, err = s.pool.Exec(ctx,
		`INSERT INTO group_members (group_id, user_id, tenant_id, role)
		 VALUES ($1::uuid, $2::uuid, $3::uuid, $4)
		 ON CONFLICT (group_id, user_id) DO NOTHING`,
		groupID, userID, tenantID, role,
	)
	if err != nil {
		return fmt.Errorf("join group: %w", err)
	}
	return nil
}

func (s *PostgresGroupStore) LeaveGroup(groupID, userID string) error {
	ctx, cancel := context.WithTimeout(context.Background(), dbTimeout)
	defer cancel()

	_, err := s.pool.Exec(ctx,
		`DELETE FROM group_members WHERE group_id = $1::uuid AND user_id = $2::uuid`,
		groupID, userID,
	)
	if err != nil {
		return fmt.Errorf("leave group: %w", err)
	}
	return nil
}

func (s *PostgresGroupStore) GetGroupMembers(groupID string) ([]GroupMember, error) {
	ctx, cancel := context.WithTimeout(context.Background(), dbTimeout)
	defer cancel()

	rows, err := s.pool.Query(ctx, `
		SELECT gm.user_id::text, u.name, gm.role, u.channel, gm.joined_at
		FROM group_members gm
		JOIN users u ON u.id = gm.user_id
		WHERE gm.group_id = $1::uuid
		ORDER BY gm.joined_at ASC`,
		groupID,
	)
	if err != nil {
		return nil, fmt.Errorf("query group members: %w", err)
	}
	defer rows.Close()

	var members []GroupMember
	for rows.Next() {
		var m GroupMember
		if err := rows.Scan(&m.UserID, &m.UserName, &m.Role, &m.Channel, &m.JoinedAt); err != nil {
			return nil, fmt.Errorf("scan group member: %w", err)
		}
		members = append(members, m)
	}
	return members, rows.Err()
}

func (s *PostgresGroupStore) GetUserGroups(userID string) ([]Group, error) {
	ctx, cancel := context.WithTimeout(context.Background(), dbTimeout)
	defer cancel()

	rows, err := s.pool.Query(ctx, `
		SELECT g.id::text, g.tenant_id::text, g.name, g.type, g.description, g.syllabus, g.subject, g.cadence,
		       g.join_code, COALESCE(g.created_by::text, ''), g.created_at, g.updated_at,
		       (SELECT COUNT(*) FROM group_members gm2 WHERE gm2.group_id = g.id)::int,
		       g.closed
		FROM groups g
		JOIN group_members gm ON gm.group_id = g.id
		WHERE gm.user_id = $1::uuid
		ORDER BY gm.joined_at DESC`,
		userID,
	)
	if err != nil {
		return nil, fmt.Errorf("query user groups: %w", err)
	}
	defer rows.Close()

	var groups []Group
	for rows.Next() {
		var g Group
		if err := rows.Scan(&g.ID, &g.TenantID, &g.Name, &g.Type, &g.Description, &g.Syllabus, &g.Subject,
			&g.Cadence, &g.JoinCode, &g.CreatedBy, &g.CreatedAt, &g.UpdatedAt, &g.MemberCount, &g.Closed); err != nil {
			return nil, fmt.Errorf("scan user group: %w", err)
		}
		groups = append(groups, g)
	}
	return groups, rows.Err()
}

func (s *PostgresGroupStore) ListGroups(tenantID, groupType string) ([]Group, error) {
	ctx, cancel := context.WithTimeout(context.Background(), dbTimeout)
	defer cancel()

	query := `
		SELECT g.id::text, g.tenant_id::text, g.name, g.type, g.description, g.syllabus, g.subject, g.cadence,
		       g.join_code, COALESCE(g.created_by::text, ''), g.created_at, g.updated_at,
		       (SELECT COUNT(*) FROM group_members gm WHERE gm.group_id = g.id)::int,
		       g.closed
		FROM groups g
		WHERE g.tenant_id = $1::uuid`
	args := []any{tenantID}

	if groupType != "" {
		query += ` AND g.type = $2`
		args = append(args, groupType)
	}
	query += ` ORDER BY g.created_at DESC`

	rows, err := s.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list groups: %w", err)
	}
	defer rows.Close()

	var groups []Group
	for rows.Next() {
		var g Group
		if err := rows.Scan(&g.ID, &g.TenantID, &g.Name, &g.Type, &g.Description, &g.Syllabus, &g.Subject,
			&g.Cadence, &g.JoinCode, &g.CreatedBy, &g.CreatedAt, &g.UpdatedAt, &g.MemberCount, &g.Closed); err != nil {
			return nil, fmt.Errorf("scan group: %w", err)
		}
		groups = append(groups, g)
	}
	return groups, rows.Err()
}

func (s *PostgresGroupStore) GetGroupMembersWithChannel(groupID string) ([]GroupMemberDelivery, error) {
	ctx, cancel := context.WithTimeout(context.Background(), dbTimeout)
	defer cancel()

	rows, err := s.pool.Query(ctx, `
		SELECT u.external_id, u.channel, u.name
		FROM group_members gm
		JOIN users u ON u.id = gm.user_id
		WHERE gm.group_id = $1::uuid
		  AND u.external_id IS NOT NULL AND u.external_id <> ''
		ORDER BY u.name ASC`,
		groupID,
	)
	if err != nil {
		return nil, fmt.Errorf("query group members with channel: %w", err)
	}
	defer rows.Close()

	var members []GroupMemberDelivery
	for rows.Next() {
		var m GroupMemberDelivery
		if err := rows.Scan(&m.ExternalID, &m.Channel, &m.UserName); err != nil {
			return nil, fmt.Errorf("scan group member delivery: %w", err)
		}
		members = append(members, m)
	}
	return members, rows.Err()
}

func (s *PostgresGroupStore) GetWeeklyLeaderboard(groupID string, limit int) ([]LeaderboardEntry, error) {
	ctx, cancel := context.WithTimeout(context.Background(), dbTimeout)
	defer cancel()

	if limit <= 0 {
		limit = 10
	}

	// Compare current mastery_score with the oldest snapshot within the past 7 days.
	// Uses DISTINCT ON to pick the earliest snapshot per user/topic in the window.
	// Only includes students (not teachers/leaders).
	rows, err := s.pool.Query(ctx, `
		WITH members AS (
			SELECT gm.user_id
			FROM group_members gm
			JOIN users u ON u.id = gm.user_id AND u.role = 'student'
			WHERE gm.group_id = $1::uuid
		),
		current_scores AS (
			SELECT lp.user_id, lp.topic_id, lp.mastery_score
			FROM learning_progress lp
			WHERE lp.user_id IN (SELECT user_id FROM members)
		),
		baseline_scores AS (
			SELECT DISTINCT ON (ms.user_id, ms.topic_id)
			       ms.user_id, ms.topic_id, ms.mastery_score
			FROM mastery_snapshots ms
			WHERE ms.user_id IN (SELECT user_id FROM members)
			  AND ms.snapshot_date >= (CURRENT_DATE - INTERVAL '7 days')::date
			  AND ms.snapshot_date < CURRENT_DATE
			ORDER BY ms.user_id, ms.topic_id, ms.snapshot_date ASC
		),
		gains AS (
			SELECT cs.user_id,
			       AVG(cs.mastery_score - COALESCE(bs.mastery_score, 0)) AS avg_gain
			FROM current_scores cs
			LEFT JOIN baseline_scores bs ON bs.user_id = cs.user_id AND bs.topic_id = cs.topic_id
			GROUP BY cs.user_id
		)
		SELECT g.user_id::text, u.name, g.avg_gain,
		       ROW_NUMBER() OVER (ORDER BY g.avg_gain DESC) AS rank
		FROM gains g
		JOIN users u ON u.id = g.user_id
		ORDER BY g.avg_gain DESC
		LIMIT $2`,
		groupID, limit,
	)
	if err != nil {
		return nil, fmt.Errorf("query weekly leaderboard: %w", err)
	}
	defer rows.Close()

	var entries []LeaderboardEntry
	for rows.Next() {
		var e LeaderboardEntry
		if err := rows.Scan(&e.UserID, &e.UserName, &e.MasteryGain, &e.Rank); err != nil {
			return nil, fmt.Errorf("scan leaderboard entry: %w", err)
		}
		entries = append(entries, e)
	}
	return entries, rows.Err()
}

const joinCodeAlphabet = "ABCDEFGHJKLMNPQRSTUVWXYZ23456789" // no I/O/0/1 to avoid confusion

func generateJoinCode() (string, error) {
	b := make([]byte, joinCodeLen)
	for i := range b {
		n, err := rand.Int(rand.Reader, big.NewInt(int64(len(joinCodeAlphabet))))
		if err != nil {
			return "", err
		}
		b[i] = joinCodeAlphabet[n.Int64()]
	}
	return string(b), nil
}
