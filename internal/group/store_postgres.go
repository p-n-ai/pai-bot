package group

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

const (
	dbTimeout               = 5 * time.Second
	joinCodeInsertMaxAttempts = 8
)

// PostgresStore is a PostgreSQL-backed implementation of Store.
type PostgresStore struct {
	pool *pgxpool.Pool
}

// NewPostgresStore creates a new PostgresStore backed by the given connection pool.
func NewPostgresStore(pool *pgxpool.Pool) *PostgresStore {
	return &PostgresStore{pool: pool}
}

// Create inserts a new group and adds the creator as owner in a single transaction.
// It retries up to joinCodeInsertMaxAttempts times on join_code uniqueness collisions.
func (s *PostgresStore) Create(ctx context.Context, g Group) (*Group, error) {
	ctx, cancel := context.WithTimeout(ctx, dbTimeout)
	defer cancel()

	for attempt := 0; attempt < joinCodeInsertMaxAttempts; attempt++ {
		code, err := GenerateJoinCode()
		if err != nil {
			return nil, fmt.Errorf("generate join code: %w", err)
		}

		tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{})
		if err != nil {
			return nil, fmt.Errorf("begin create group tx: %w", err)
		}
		defer func() { _ = tx.Rollback(ctx) }()

		var created Group
		err = tx.QueryRow(ctx,
			`INSERT INTO groups (tenant_id, name, syllabus_id, join_code, status, created_by)
			 VALUES ($1::uuid, $2, $3, $4, 'active', $5::uuid)
			 RETURNING id::text, tenant_id::text, name, COALESCE(syllabus_id, ''), join_code, status, COALESCE(created_by::text, ''), created_at`,
			g.TenantID,
			g.Name,
			nilIfEmpty(g.SyllabusID),
			code,
			nilIfEmpty(g.CreatedBy),
		).Scan(
			&created.ID,
			&created.TenantID,
			&created.Name,
			&created.SyllabusID,
			&created.JoinCode,
			&created.Status,
			&created.CreatedBy,
			&created.CreatedAt,
		)
		if err != nil {
			_ = tx.Rollback(ctx)
			if isGroupUniqueViolation(err) {
				continue
			}
			return nil, fmt.Errorf("insert group: %w", err)
		}

		// Add creator as owner.
		if created.CreatedBy != "" {
			_, err = tx.Exec(ctx,
				`INSERT INTO group_members (group_id, user_id, membership_role)
				 VALUES ($1::uuid, $2::uuid, 'owner')`,
				created.ID,
				created.CreatedBy,
			)
			if err != nil {
				_ = tx.Rollback(ctx)
				return nil, fmt.Errorf("insert owner member: %w", err)
			}
		}

		if err := tx.Commit(ctx); err != nil {
			return nil, fmt.Errorf("commit create group: %w", err)
		}
		return &created, nil
	}

	return nil, fmt.Errorf("create group: exhausted join code retries")
}

// GetByID returns the group identified by groupID within tenantID.
func (s *PostgresStore) GetByID(ctx context.Context, tenantID, groupID string) (*Group, error) {
	ctx, cancel := context.WithTimeout(ctx, dbTimeout)
	defer cancel()

	var g Group
	err := s.pool.QueryRow(ctx,
		`SELECT id::text, tenant_id::text, name, COALESCE(syllabus_id, ''), join_code, status,
		        COALESCE(created_by::text, ''), created_at
		 FROM groups
		 WHERE tenant_id = $1::uuid AND id = $2::uuid`,
		tenantID, groupID,
	).Scan(&g.ID, &g.TenantID, &g.Name, &g.SyllabusID, &g.JoinCode, &g.Status, &g.CreatedBy, &g.CreatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrGroupNotFound
		}
		return nil, fmt.Errorf("get group by id: %w", err)
	}
	return &g, nil
}

// GetByJoinCode returns an active group matching the given join code within tenantID.
func (s *PostgresStore) GetByJoinCode(ctx context.Context, tenantID, code string) (*Group, error) {
	ctx, cancel := context.WithTimeout(ctx, dbTimeout)
	defer cancel()

	code = NormalizeJoinCode(code)

	var g Group
	err := s.pool.QueryRow(ctx,
		`SELECT id::text, tenant_id::text, name, COALESCE(syllabus_id, ''), join_code, status,
		        COALESCE(created_by::text, ''), created_at
		 FROM groups
		 WHERE tenant_id = $1::uuid AND join_code = $2 AND status = 'active'`,
		tenantID, code,
	).Scan(&g.ID, &g.TenantID, &g.Name, &g.SyllabusID, &g.JoinCode, &g.Status, &g.CreatedBy, &g.CreatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrGroupNotFound
		}
		return nil, fmt.Errorf("get group by join code: %w", err)
	}
	return &g, nil
}

// ListByTenant returns all active groups belonging to tenantID, ordered by creation time descending.
func (s *PostgresStore) ListByTenant(ctx context.Context, tenantID string) ([]Group, error) {
	ctx, cancel := context.WithTimeout(ctx, dbTimeout)
	defer cancel()

	rows, err := s.pool.Query(ctx,
		`SELECT id::text, tenant_id::text, name, COALESCE(syllabus_id, ''), join_code, status,
		        COALESCE(created_by::text, ''), created_at
		 FROM groups
		 WHERE tenant_id = $1::uuid AND status = 'active'
		 ORDER BY created_at DESC`,
		tenantID,
	)
	if err != nil {
		return nil, fmt.Errorf("list groups by tenant: %w", err)
	}
	defer rows.Close()

	return scanGroups(rows)
}

// ListByUser returns all active groups where userID is a member.
func (s *PostgresStore) ListByUser(ctx context.Context, userID string) ([]Group, error) {
	ctx, cancel := context.WithTimeout(ctx, dbTimeout)
	defer cancel()

	rows, err := s.pool.Query(ctx,
		`SELECT g.id::text, g.tenant_id::text, g.name, COALESCE(g.syllabus_id, ''), g.join_code,
		        g.status, COALESCE(g.created_by::text, ''), g.created_at
		 FROM groups g
		 JOIN group_members gm ON gm.group_id = g.id
		 WHERE gm.user_id = $1::uuid AND g.status = 'active'
		 ORDER BY g.created_at DESC`,
		userID,
	)
	if err != nil {
		return nil, fmt.Errorf("list groups by user: %w", err)
	}
	defer rows.Close()

	return scanGroups(rows)
}

// Archive sets the group's status to "archived".
func (s *PostgresStore) Archive(ctx context.Context, tenantID, groupID string) error {
	ctx, cancel := context.WithTimeout(ctx, dbTimeout)
	defer cancel()

	tag, err := s.pool.Exec(ctx,
		`UPDATE groups SET status = 'archived', updated_at = NOW()
		 WHERE tenant_id = $1::uuid AND id = $2::uuid`,
		tenantID, groupID,
	)
	if err != nil {
		return fmt.Errorf("archive group: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrGroupNotFound
	}
	return nil
}

// Rename updates the name of a group.
func (s *PostgresStore) Rename(ctx context.Context, tenantID, groupID, newName string) error {
	ctx, cancel := context.WithTimeout(ctx, dbTimeout)
	defer cancel()

	tag, err := s.pool.Exec(ctx,
		`UPDATE groups SET name = $3, updated_at = NOW()
		 WHERE tenant_id = $1::uuid AND id = $2::uuid`,
		tenantID, groupID, newName,
	)
	if err != nil {
		return fmt.Errorf("rename group: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrGroupNotFound
	}
	return nil
}

// AddMember adds userID to the group with the given membership role.
// Returns ErrGroupNotFound if the group does not exist, ErrGroupArchived if archived,
// and ErrAlreadyMember if the user is already in the group.
func (s *PostgresStore) AddMember(ctx context.Context, groupID, userID, membershipRole string) error {
	ctx, cancel := context.WithTimeout(ctx, dbTimeout)
	defer cancel()

	// Verify group exists and is active.
	var status string
	err := s.pool.QueryRow(ctx,
		`SELECT status FROM groups WHERE id = $1::uuid`,
		groupID,
	).Scan(&status)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrGroupNotFound
		}
		return fmt.Errorf("check group status: %w", err)
	}
	if status == "archived" {
		return ErrGroupArchived
	}

	_, err = s.pool.Exec(ctx,
		`INSERT INTO group_members (group_id, user_id, membership_role)
		 VALUES ($1::uuid, $2::uuid, $3)`,
		groupID, userID, membershipRole,
	)
	if err != nil {
		if isGroupUniqueViolation(err) {
			return ErrAlreadyMember
		}
		return fmt.Errorf("add member: %w", err)
	}
	return nil
}

// RemoveMember removes userID from the group.
// Returns ErrNotMember if not present, ErrOwnerCannotLeave if the user is the owner.
func (s *PostgresStore) RemoveMember(ctx context.Context, groupID, userID string) error {
	ctx, cancel := context.WithTimeout(ctx, dbTimeout)
	defer cancel()

	var role string
	err := s.pool.QueryRow(ctx,
		`SELECT membership_role FROM group_members WHERE group_id = $1::uuid AND user_id = $2::uuid`,
		groupID, userID,
	).Scan(&role)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrNotMember
		}
		return fmt.Errorf("check member role: %w", err)
	}
	if role == "owner" {
		return ErrOwnerCannotLeave
	}

	_, err = s.pool.Exec(ctx,
		`DELETE FROM group_members WHERE group_id = $1::uuid AND user_id = $2::uuid`,
		groupID, userID,
	)
	if err != nil {
		return fmt.Errorf("remove member: %w", err)
	}
	return nil
}

// GetMembers returns all members of the given group.
func (s *PostgresStore) GetMembers(ctx context.Context, groupID string) ([]Member, error) {
	ctx, cancel := context.WithTimeout(ctx, dbTimeout)
	defer cancel()

	rows, err := s.pool.Query(ctx,
		`SELECT user_id::text, membership_role, joined_at
		 FROM group_members
		 WHERE group_id = $1::uuid
		 ORDER BY joined_at ASC`,
		groupID,
	)
	if err != nil {
		return nil, fmt.Errorf("get members: %w", err)
	}
	defer rows.Close()

	var members []Member
	for rows.Next() {
		var m Member
		if err := rows.Scan(&m.UserID, &m.MembershipRole, &m.JoinedAt); err != nil {
			return nil, fmt.Errorf("scan member: %w", err)
		}
		members = append(members, m)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate members: %w", err)
	}
	return members, nil
}

// MemberCount returns the number of members in a group.
func (s *PostgresStore) MemberCount(ctx context.Context, groupID string) (int, error) {
	ctx, cancel := context.WithTimeout(ctx, dbTimeout)
	defer cancel()

	var count int
	err := s.pool.QueryRow(ctx,
		`SELECT COUNT(*) FROM group_members WHERE group_id = $1::uuid`,
		groupID,
	).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("member count: %w", err)
	}
	return count, nil
}

// IsMember reports whether userID is a member of groupID.
func (s *PostgresStore) IsMember(ctx context.Context, groupID, userID string) (bool, error) {
	ctx, cancel := context.WithTimeout(ctx, dbTimeout)
	defer cancel()

	var exists bool
	err := s.pool.QueryRow(ctx,
		`SELECT EXISTS(
		   SELECT 1 FROM group_members WHERE group_id = $1::uuid AND user_id = $2::uuid
		 )`,
		groupID, userID,
	).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("is member: %w", err)
	}
	return exists, nil
}

// ── helpers ──────────────────────────────────────────────────────────────────

func isGroupUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505"
}

// nilIfEmpty returns nil for an empty string so PostgreSQL receives NULL instead of ''.
func nilIfEmpty(s string) interface{} {
	if s == "" {
		return nil
	}
	return s
}

func scanGroups(rows pgx.Rows) ([]Group, error) {
	var groups []Group
	for rows.Next() {
		var g Group
		if err := rows.Scan(&g.ID, &g.TenantID, &g.Name, &g.SyllabusID, &g.JoinCode, &g.Status, &g.CreatedBy, &g.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan group: %w", err)
		}
		groups = append(groups, g)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate groups: %w", err)
	}
	return groups, nil
}
