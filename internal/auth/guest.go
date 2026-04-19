// Copyright 2026 the P&AI authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package auth

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// ErrNotGuest is returned when UpgradeGuest is called on a user that is not a guest.
var ErrNotGuest = errors.New("user is not a guest")

// ErrEmailAlreadyUsed is returned when the email is already registered in the tenant.
var ErrEmailAlreadyUsed = errors.New("email already in use")

// GuestService handles guest token issuance for the embed widget.
type GuestService struct {
	pool         *pgxpool.Pool
	tokenManager *TokenManager
}

// NewGuestService constructs a GuestService backed by the given pool and token manager.
func NewGuestService(pool *pgxpool.Pool, tokenManager *TokenManager) *GuestService {
	return &GuestService{pool: pool, tokenManager: tokenManager}
}

// findGuestByFingerprint looks up an existing guest user by fingerprint for the given tenant.
// Returns empty string if fingerprint is empty or no matching guest is found.
func (gs *GuestService) findGuestByFingerprint(ctx context.Context, tenantID, fingerprint string) (string, error) {
	if fingerprint == "" {
		return "", nil
	}
	var userID string
	err := gs.pool.QueryRow(ctx,
		`SELECT id::text FROM users
         WHERE tenant_id = $1::uuid AND role = 'guest' AND config->>'fingerprint' = $2
         LIMIT 1`,
		tenantID, fingerprint,
	).Scan(&userID)
	if err != nil {
		if err == pgx.ErrNoRows {
			return "", nil
		}
		return "", fmt.Errorf("query guest by fingerprint: %w", err)
	}
	return userID, nil
}

// IssueGuestToken creates or reuses a guest user and issues a JWT.
// If fingerprint is non-empty, it looks up an existing guest with that fingerprint
// for the same tenant before creating a new one.
func (gs *GuestService) IssueGuestToken(ctx context.Context, tenantID, origin, fingerprint string) (token string, userID string, err error) {
	// Try to reuse an existing guest if fingerprint is provided.
	if fingerprint != "" {
		userID, err = gs.findGuestByFingerprint(ctx, tenantID, fingerprint)
		if err != nil {
			return "", "", fmt.Errorf("find guest by fingerprint: %w", err)
		}
	}

	if userID == "" {
		// Generate a random external_id for the new guest.
		buf := make([]byte, 16)
		if _, err = rand.Read(buf); err != nil {
			return "", "", fmt.Errorf("generate guest id: %w", err)
		}
		externalID := hex.EncodeToString(buf)

		configData := map[string]string{"embed_origin": origin}
		if fingerprint != "" {
			configData["fingerprint"] = fingerprint
		}
		config, err := json.Marshal(configData)
		if err != nil {
			return "", "", fmt.Errorf("marshal guest config: %w", err)
		}

		const query = `
			INSERT INTO users (tenant_id, role, name, external_id, channel, config)
			VALUES ($1, 'guest', 'Guest', $2, 'embed', $3)
			RETURNING id`

		if err = gs.pool.QueryRow(ctx, query, tenantID, externalID, config).Scan(&userID); err != nil {
			return "", "", fmt.Errorf("insert guest user: %w", err)
		}
	}

	claims := TokenClaims{
		Subject:  userID,
		TenantID: tenantID,
		Role:     RoleGuest,
	}
	token, err = gs.tokenManager.Issue(claims, time.Now().UTC())
	if err != nil {
		return "", "", fmt.Errorf("issue guest token: %w", err)
	}

	return token, userID, nil
}

// UpgradeGuest converts a guest user to a student, setting their name, email, and password.
// The userID must be an existing guest in the given tenant.
// Returns a new JWT with role=student.
func (gs *GuestService) UpgradeGuest(ctx context.Context, userID, tenantID, name, email, password string) (token string, err error) {
	ctx, cancel := context.WithTimeout(ctx, authDBTimeout)
	defer cancel()

	name = strings.TrimSpace(name)
	email = NormalizeIdentifier(email)
	password = strings.TrimSpace(password)

	// Verify the user exists and is a guest.
	var role string
	if err := gs.pool.QueryRow(ctx,
		`SELECT role FROM users WHERE id = $1::uuid AND tenant_id = $2::uuid`,
		userID, tenantID,
	).Scan(&role); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return "", ErrNotGuest
		}
		return "", fmt.Errorf("lookup guest user: %w", err)
	}
	if role != string(RoleGuest) {
		return "", ErrNotGuest
	}

	// Check if email is already taken in this tenant.
	var emailExists bool
	if err := gs.pool.QueryRow(ctx,
		`SELECT EXISTS(SELECT 1 FROM auth_identities WHERE tenant_id = $1::uuid AND provider = 'password' AND identifier_normalized = $2)`,
		tenantID, email,
	).Scan(&emailExists); err != nil {
		return "", fmt.Errorf("check email exists: %w", err)
	}
	if emailExists {
		return "", ErrEmailAlreadyUsed
	}

	passwordHash, err := HashPassword(password)
	if err != nil {
		return "", fmt.Errorf("hash password: %w", err)
	}

	tx, err := gs.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return "", fmt.Errorf("begin upgrade transaction: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	if _, err := tx.Exec(ctx,
		`UPDATE users SET role = 'student', name = $3, updated_at = now() WHERE id = $1::uuid AND tenant_id = $2::uuid`,
		userID, tenantID, name,
	); err != nil {
		return "", fmt.Errorf("upgrade guest user: %w", err)
	}

	now := time.Now().UTC()
	if _, err := tx.Exec(ctx, `
		INSERT INTO auth_identities (
			user_id, tenant_id, provider, identifier, identifier_normalized, password_hash, last_login_at, created_at, updated_at
		)
		VALUES ($1::uuid, $2::uuid, 'password', $3, $4, $5, $6, $6, $6)
	`, userID, tenantID, email, email, passwordHash, now); err != nil {
		return "", fmt.Errorf("insert auth identity: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return "", fmt.Errorf("commit upgrade transaction: %w", err)
	}

	claims := TokenClaims{
		Subject:  userID,
		TenantID: tenantID,
		Role:     RoleStudent,
	}
	token, err = gs.tokenManager.Issue(claims, time.Now().UTC())
	if err != nil {
		return "", fmt.Errorf("issue student token: %w", err)
	}

	return token, nil
}
