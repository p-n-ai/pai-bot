// Copyright 2026 the P&AI authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package adminapi

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"math/big"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
)

// AdminGroup represents a group in the admin API.
type AdminGroup struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Type        string `json:"type"`
	Description string `json:"description"`
	Syllabus    string `json:"syllabus"`
	Subject     string `json:"subject"`
	Cadence     string `json:"cadence"`
	JoinCode    string `json:"join_code"`
	MemberCount int    `json:"member_count"`
	Closed      bool   `json:"closed"`
	CreatedAt   string `json:"created_at"`
	UpdatedAt   string `json:"updated_at"`
}

// AdminGroupDetail is a group with its members.
type AdminGroupDetail struct {
	AdminGroup
	Members []AdminGroupMember `json:"members"`
}

// AdminGroupMember represents a member in a group for the admin API.
type AdminGroupMember struct {
	ID      string  `json:"id"`
	Name    string  `json:"name"`
	Role    string  `json:"role"`
	Channel string  `json:"channel"`
	Mastery float64 `json:"mastery"`
}

// CreateGroupInput is the request body for creating a group.
type CreateGroupInput struct {
	Name        string `json:"name"`
	Type        string `json:"type"`
	Description string `json:"description"`
	Syllabus    string `json:"syllabus"`
	Subject     string `json:"subject"`
	Cadence     string `json:"cadence"`
}

// UpdateGroupInput is the request body for updating a group.
type AdminUpdateGroupInput struct {
	Name        *string `json:"name,omitempty"`
	Description *string `json:"description,omitempty"`
	Syllabus    *string `json:"syllabus,omitempty"`
	Subject     *string `json:"subject,omitempty"`
	Cadence     *string `json:"cadence,omitempty"`
	Closed      *bool   `json:"closed,omitempty"`
}

// AddMemberInput is the request body for adding a member to a group.
type AddMemberInput struct {
	UserID string `json:"user_id"`
	Role   string `json:"role"`
}

// AdminLeaderboardEntry represents a single leaderboard row.
type AdminLeaderboardEntry struct {
	UserID      string  `json:"user_id"`
	UserName    string  `json:"user_name"`
	MasteryGain float64 `json:"mastery_gain"`
	Rank        int     `json:"rank"`
}

const adminJoinCodeLen = 6
const adminJoinCodeAlphabet = "ABCDEFGHJKLMNPQRSTUVWXYZ23456789"

func (s *Service) ListGroups(groupType string) ([]AdminGroup, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	query := fmt.Sprintf(`
		SELECT g.id::text, g.name, g.type, g.description, g.syllabus, g.subject, g.cadence,
		       g.join_code, g.created_at, g.updated_at,
		       (SELECT COUNT(*) FROM group_members gm WHERE gm.group_id = g.id)::int,
		       g.closed
		FROM groups g
		WHERE %s`, s.tenantPredicate("g.tenant_id", 1))

	args := []any{s.tenantArg()}
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

	var groups []AdminGroup
	for rows.Next() {
		var g AdminGroup
		var createdAt, updatedAt time.Time
		if err := rows.Scan(&g.ID, &g.Name, &g.Type, &g.Description, &g.Syllabus, &g.Subject,
			&g.Cadence, &g.JoinCode, &createdAt, &updatedAt, &g.MemberCount, &g.Closed); err != nil {
			return nil, fmt.Errorf("scan group: %w", err)
		}
		g.CreatedAt = createdAt.UTC().Format(time.RFC3339)
		g.UpdatedAt = updatedAt.UTC().Format(time.RFC3339)
		groups = append(groups, g)
	}
	return groups, rows.Err()
}

func (s *Service) CreateGroup(input CreateGroupInput, createdByUserID string) (AdminGroup, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if strings.TrimSpace(input.Name) == "" {
		return AdminGroup{}, fmt.Errorf("%w: name is required", ErrInvalidArgument)
	}
	groupType := input.Type
	if groupType == "" {
		groupType = "class"
	}
	if groupType != "class" && groupType != "study_group" {
		return AdminGroup{}, fmt.Errorf("%w: type must be 'class' or 'study_group'", ErrInvalidArgument)
	}

	code, err := adminGenerateJoinCode()
	if err != nil {
		return AdminGroup{}, fmt.Errorf("generate join code: %w", err)
	}

	tenantID := s.tenantID
	if s.allTenants {
		return AdminGroup{}, fmt.Errorf("%w: cannot create group without tenant scope", ErrInvalidArgument)
	}

	var createdBy *string
	if createdByUserID != "" {
		createdBy = &createdByUserID
	}

	var g AdminGroup
	var createdAt, updatedAt time.Time
	err = s.pool.QueryRow(ctx,
		`INSERT INTO groups (tenant_id, name, type, description, syllabus, subject, cadence, join_code, created_by)
		 VALUES ($1::uuid, $2, $3, $4, $5, $6, $7, $8, $9)
		 RETURNING id::text, name, type, description, syllabus, subject, cadence, join_code, created_at, updated_at`,
		tenantID, input.Name, groupType, input.Description, input.Syllabus, input.Subject,
		input.Cadence, code, createdBy,
	).Scan(&g.ID, &g.Name, &g.Type, &g.Description, &g.Syllabus, &g.Subject, &g.Cadence,
		&g.JoinCode, &createdAt, &updatedAt)
	if err != nil {
		return AdminGroup{}, fmt.Errorf("create group: %w", err)
	}
	g.CreatedAt = createdAt.UTC().Format(time.RFC3339)
	g.UpdatedAt = updatedAt.UTC().Format(time.RFC3339)
	return g, nil
}

func (s *Service) GetGroupDetail(id string) (AdminGroupDetail, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var g AdminGroup
	var createdAt, updatedAt time.Time
	err := s.pool.QueryRow(ctx, fmt.Sprintf(`
		SELECT g.id::text, g.name, g.type, g.description, g.syllabus, g.subject, g.cadence,
		       g.join_code, g.created_at, g.updated_at,
		       (SELECT COUNT(*) FROM group_members gm WHERE gm.group_id = g.id)::int,
		       g.closed
		FROM groups g
		WHERE g.id = $1::uuid AND %s`, s.tenantPredicate("g.tenant_id", 2)),
		id, s.tenantArg(),
	).Scan(&g.ID, &g.Name, &g.Type, &g.Description, &g.Syllabus, &g.Subject, &g.Cadence,
		&g.JoinCode, &createdAt, &updatedAt, &g.MemberCount, &g.Closed)
	if errors.Is(err, pgx.ErrNoRows) {
		return AdminGroupDetail{}, fmt.Errorf("%w: group %s", ErrNotFound, id)
	}
	if err != nil {
		return AdminGroupDetail{}, fmt.Errorf("get group detail: %w", err)
	}
	g.CreatedAt = createdAt.UTC().Format(time.RFC3339)
	g.UpdatedAt = updatedAt.UTC().Format(time.RFC3339)

	// Fetch members with average mastery
	rows, err := s.pool.Query(ctx, `
		SELECT gm.user_id::text, u.name, gm.role, u.channel,
		       COALESCE(AVG(lp.mastery_score), 0) AS avg_mastery
		FROM group_members gm
		JOIN users u ON u.id = gm.user_id
		LEFT JOIN learning_progress lp ON lp.user_id = gm.user_id AND lp.tenant_id = gm.tenant_id
		WHERE gm.group_id = $1::uuid
		GROUP BY gm.user_id, u.name, gm.role, u.channel, gm.joined_at
		ORDER BY gm.joined_at ASC`,
		id,
	)
	if err != nil {
		return AdminGroupDetail{}, fmt.Errorf("query group members: %w", err)
	}
	defer rows.Close()

	var members []AdminGroupMember
	for rows.Next() {
		var m AdminGroupMember
		if err := rows.Scan(&m.ID, &m.Name, &m.Role, &m.Channel, &m.Mastery); err != nil {
			return AdminGroupDetail{}, fmt.Errorf("scan group member: %w", err)
		}
		members = append(members, m)
	}
	if err := rows.Err(); err != nil {
		return AdminGroupDetail{}, fmt.Errorf("iterate group members: %w", err)
	}

	return AdminGroupDetail{AdminGroup: g, Members: members}, nil
}

func (s *Service) UpdateGroup(id string, input AdminUpdateGroupInput) (AdminGroup, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	setClauses := []string{"updated_at = NOW()"}
	args := []any{id, s.tenantArg()}
	argIdx := 3

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
	}

	var g AdminGroup
	var createdAt, updatedAt time.Time
	err := s.pool.QueryRow(ctx, fmt.Sprintf(
		`UPDATE groups SET %s WHERE id = $1::uuid AND %s
		 RETURNING id::text, name, type, description, syllabus, subject, cadence, join_code, created_at, updated_at, closed`,
		strings.Join(setClauses, ", "), s.tenantPredicate("tenant_id", 2)),
		args...,
	).Scan(&g.ID, &g.Name, &g.Type, &g.Description, &g.Syllabus, &g.Subject, &g.Cadence,
		&g.JoinCode, &createdAt, &updatedAt, &g.Closed)
	if errors.Is(err, pgx.ErrNoRows) {
		return AdminGroup{}, fmt.Errorf("%w: group %s", ErrNotFound, id)
	}
	if err != nil {
		return AdminGroup{}, fmt.Errorf("update group: %w", err)
	}
	g.CreatedAt = createdAt.UTC().Format(time.RFC3339)
	g.UpdatedAt = updatedAt.UTC().Format(time.RFC3339)
	return g, nil
}

func (s *Service) DeleteGroup(id string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	cmd, err := s.pool.Exec(ctx, fmt.Sprintf(
		`DELETE FROM groups WHERE id = $1::uuid AND %s`,
		s.tenantPredicate("tenant_id", 2)),
		id, s.tenantArg(),
	)
	if err != nil {
		return fmt.Errorf("delete group: %w", err)
	}
	if cmd.RowsAffected() == 0 {
		return fmt.Errorf("%w: group %s", ErrNotFound, id)
	}
	return nil
}

func (s *Service) AddGroupMember(groupID, userID, role string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if role == "" {
		role = "member"
	}

	tenantID := s.tenantID
	if s.allTenants {
		// Resolve tenant from the group
		err := s.pool.QueryRow(ctx, `SELECT tenant_id::text FROM groups WHERE id = $1::uuid`, groupID).Scan(&tenantID)
		if errors.Is(err, pgx.ErrNoRows) {
			return fmt.Errorf("%w: group %s", ErrNotFound, groupID)
		}
		if err != nil {
			return fmt.Errorf("resolve group tenant: %w", err)
		}
	}

	_, err := s.pool.Exec(ctx,
		`INSERT INTO group_members (group_id, user_id, tenant_id, role)
		 VALUES ($1::uuid, $2::uuid, $3::uuid, $4)
		 ON CONFLICT (group_id, user_id) DO UPDATE SET role = $4`,
		groupID, userID, tenantID, role,
	)
	if err != nil {
		return fmt.Errorf("add group member: %w", err)
	}
	return nil
}

func (s *Service) RemoveGroupMember(groupID, userID string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	cmd, err := s.pool.Exec(ctx, fmt.Sprintf(
		`DELETE FROM group_members gm
		 USING groups g
		 WHERE gm.group_id = $1::uuid AND gm.user_id = $2::uuid
		   AND g.id = gm.group_id AND %s`,
		s.tenantPredicate("g.tenant_id", 3)),
		groupID, userID, s.tenantArg(),
	)
	if err != nil {
		return fmt.Errorf("remove group member: %w", err)
	}
	if cmd.RowsAffected() == 0 {
		return fmt.Errorf("%w: member %s in group %s", ErrNotFound, userID, groupID)
	}
	return nil
}

func (s *Service) GetGroupLeaderboard(id string) ([]AdminLeaderboardEntry, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	rows, err := s.pool.Query(ctx, fmt.Sprintf(`
		WITH members AS (
			SELECT gm.user_id
			FROM group_members gm
			JOIN groups g ON g.id = gm.group_id
			JOIN users u ON u.id = gm.user_id AND u.role = 'student'
			WHERE gm.group_id = $1::uuid AND %s
		),`, s.tenantPredicate("g.tenant_id", 2))+`
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
		LIMIT 10`,
		id, s.tenantArg(),
	)
	if err != nil {
		return nil, fmt.Errorf("query group leaderboard: %w", err)
	}
	defer rows.Close()

	var entries []AdminLeaderboardEntry
	for rows.Next() {
		var e AdminLeaderboardEntry
		if err := rows.Scan(&e.UserID, &e.UserName, &e.MasteryGain, &e.Rank); err != nil {
			return nil, fmt.Errorf("scan leaderboard entry: %w", err)
		}
		entries = append(entries, e)
	}
	return entries, rows.Err()
}

// GetGroupClassProgress returns class progress for a real group (by UUID).
// This is the group-backed replacement for form-based class progress.
func (s *Service) GetGroupClassProgress(groupID string) (ClassProgress, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	rows, err := s.pool.Query(ctx, fmt.Sprintf(`
		SELECT
			COALESCE(NULLIF(u.external_id, ''), u.id::text) AS student_id,
			u.name,
			lp.topic_id,
			lp.mastery_score
		FROM group_members gm
		JOIN users u ON u.id = gm.user_id AND u.role = 'student'
		LEFT JOIN learning_progress lp ON lp.user_id = gm.user_id AND lp.tenant_id = gm.tenant_id
		WHERE gm.group_id = $1::uuid AND %s
		ORDER BY gm.joined_at ASC, u.name ASC, lp.topic_id ASC
	`, s.tenantPredicate("gm.tenant_id", 2)), groupID, s.tenantArg())
	if err != nil {
		return ClassProgress{}, fmt.Errorf("query group class progress: %w", err)
	}
	defer rows.Close()

	return scanClassProgressRows(rows)
}

func adminGenerateJoinCode() (string, error) {
	b := make([]byte, adminJoinCodeLen)
	for i := range b {
		n, err := rand.Int(rand.Reader, big.NewInt(int64(len(adminJoinCodeAlphabet))))
		if err != nil {
			return "", err
		}
		b[i] = adminJoinCodeAlphabet[n.Int64()]
	}
	return string(b), nil
}
