// Copyright 2026 the P&AI authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package auth

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/p-n-ai/pai-bot/internal/platform/mailer"
)

const (
	authDBTimeout             = 5 * time.Second
	authRollbackTimeout       = 2 * time.Second
	authSessionRefreshWindow  = 24 * time.Hour
	authSessionRefreshDivisor = 4
)

type PostgresService struct {
	pool             *pgxpool.Pool
	sessionTTL       time.Duration
	httpClient       HTTPDoer
	google           *GoogleOAuthProvider
	inviteMailSender mailer.Sender
	now              func() time.Time
}

func NewPostgresService(pool *pgxpool.Pool, sessionTTL time.Duration) *PostgresService {
	return newPostgresService(pool, sessionTTL, time.Now)
}

func newPostgresService(pool *pgxpool.Pool, sessionTTL time.Duration, now func() time.Time) *PostgresService {
	if now == nil {
		now = time.Now
	}

	return &PostgresService{
		pool:       pool,
		sessionTTL: sessionTTL,
		httpClient: &httpClient{client: &defaultHTTPClient},
		now:        now,
	}
}

func (s *PostgresService) ConfigureGoogleOAuth(cfg GoogleOAuthProviderConfig) {
	s.google = NewGoogleOAuthProvider(cfg, s.httpClient, s.now)
}

func (s *PostgresService) ConfigureInviteEmail(sender mailer.Sender) {
	s.inviteMailSender = sender
}

func (s *PostgresService) EnsureBootstrapPlatformAdmin(ctx context.Context, email, password string) (bool, error) {
	ctx, cancel := context.WithTimeout(ctx, authDBTimeout)
	defer cancel()

	normalizedEmail := NormalizeIdentifier(email)
	if normalizedEmail == "" || strings.TrimSpace(password) == "" {
		return false, ErrInvalidCredentials
	}

	passwordHash, err := HashPassword(password)
	if err != nil {
		return false, fmt.Errorf("hash bootstrap password: %w", err)
	}

	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return false, fmt.Errorf("begin bootstrap transaction: %w", err)
	}
	defer rollbackAuthTx(tx)

	if _, err := tx.Exec(ctx, `SELECT pg_advisory_xact_lock($1)`, int64(8_420_001)); err != nil {
		return false, fmt.Errorf("lock bootstrap platform admin creation: %w", err)
	}

	var existingAdminCount int
	if err := tx.QueryRow(ctx, `
		SELECT COUNT(*)
		FROM users
		WHERE role IN ('admin', 'platform_admin')
	`).Scan(&existingAdminCount); err != nil {
		return false, fmt.Errorf("count admin-capable users: %w", err)
	}
	if existingAdminCount > 0 {
		return false, nil
	}

	var userID string
	if err := tx.QueryRow(ctx, `
		INSERT INTO users (tenant_id, role, name, channel, config)
		VALUES (NULL, 'platform_admin', 'Platform Admin', 'web', '{"bootstrap": true}'::jsonb)
		RETURNING id::text
	`).Scan(&userID); err != nil {
		return false, fmt.Errorf("insert bootstrap platform admin: %w", err)
	}

	now := s.now().UTC()
	if _, err := tx.Exec(ctx, `
		INSERT INTO auth_identities (
			user_id, tenant_id, provider, identifier, identifier_normalized, password_hash, email_verified_at, last_login_at, created_at, updated_at
		)
		VALUES ($1::uuid, NULL, 'password', $2, $3, $4, $5, $5, $5, $5)
		ON CONFLICT (provider, identifier_normalized) WHERE tenant_id IS NULL DO UPDATE
		SET user_id = EXCLUDED.user_id,
		    tenant_id = EXCLUDED.tenant_id,
		    identifier = EXCLUDED.identifier,
		    identifier_normalized = EXCLUDED.identifier_normalized,
		    password_hash = EXCLUDED.password_hash,
		    email_verified_at = EXCLUDED.email_verified_at,
		    last_login_at = EXCLUDED.last_login_at,
		    updated_at = EXCLUDED.updated_at
	`, userID, normalizedEmail, normalizedEmail, passwordHash, now); err != nil {
		return false, fmt.Errorf("insert bootstrap auth identity: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return false, fmt.Errorf("commit bootstrap platform admin: %w", err)
	}

	return true, nil
}

func (s *PostgresService) Login(ctx context.Context, req LoginRequest) (Session, error) {
	ctx, cancel := context.WithTimeout(ctx, authDBTimeout)
	defer cancel()

	tenantID := strings.TrimSpace(req.TenantID)
	email := NormalizeIdentifier(req.Email)
	if email == "" || strings.TrimSpace(req.Password) == "" {
		return Session{}, ErrInvalidCredentials
	}

	candidate, err := s.resolvePasswordLoginCandidate(ctx, tenantID, email)
	if err != nil {
		return Session{}, err
	}
	if err := ComparePassword(candidate.passwordHash, req.Password); err != nil {
		return Session{}, ErrInvalidCredentials
	}

	now := s.now().UTC()
	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return Session{}, fmt.Errorf("begin login transaction: %w", err)
	}
	defer rollbackAuthTx(tx)

	if err := touchPasswordIdentityLoginTx(ctx, tx, candidate.tenantID, email, now, "update last_login_at"); err != nil {
		return Session{}, err
	}

	pair, err := s.issueSession(ctx, tx, candidate.sessionUser(email), now)
	if err != nil {
		return Session{}, err
	}
	pair.TenantChoices, err = s.tenantOptionsByEmail(ctx, tx, email)
	if err != nil {
		return Session{}, err
	}

	if err := tx.Commit(ctx); err != nil {
		return Session{}, fmt.Errorf("commit login transaction: %w", err)
	}

	return pair, nil
}

type passwordLoginCandidate struct {
	userID       string
	tenantID     string
	tenantSlug   string
	tenantName   string
	role         string
	name         string
	passwordHash string
}

func (c passwordLoginCandidate) sessionUser(email string) sessionUser {
	return sessionUser{
		UserID:     c.userID,
		TenantID:   c.tenantID,
		TenantSlug: c.tenantSlug,
		TenantName: c.tenantName,
		Role:       Role(c.role),
		Name:       c.name,
		Email:      email,
	}
}

func (s *PostgresService) resolvePasswordLoginCandidate(ctx context.Context, tenantID, email string) (passwordLoginCandidate, error) {
	if tenantID != "" {
		return passwordLoginCandidateByTenant(ctx, s.pool, tenantID, email, "query login identity")
	}

	candidates, err := passwordLoginCandidatesByEmail(ctx, s.pool, email)
	if err != nil {
		return passwordLoginCandidate{}, err
	}
	if len(candidates) == 0 {
		return passwordLoginCandidate{}, ErrInvalidCredentials
	}
	return candidates[0], nil
}

func passwordLoginCandidateByTenant(ctx context.Context, q authRowQueryer, tenantID, email, errPrefix string) (passwordLoginCandidate, error) {
	var c passwordLoginCandidate
	err := q.QueryRow(ctx, `
		SELECT u.id::text, COALESCE(u.tenant_id::text, ''), COALESCE(t.slug, ''), COALESCE(t.name, ''), u.role, u.name, ai.password_hash
		FROM auth_identities ai
		JOIN users u ON u.id = ai.user_id
		LEFT JOIN tenants t ON t.id = u.tenant_id
		WHERE ai.tenant_id = $1::uuid
		  AND ai.provider = 'password'
		  AND ai.identifier_normalized = $2
		LIMIT 1
	`, tenantID, email).Scan(&c.userID, &c.tenantID, &c.tenantSlug, &c.tenantName, &c.role, &c.name, &c.passwordHash)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return passwordLoginCandidate{}, ErrInvalidCredentials
		}
		return passwordLoginCandidate{}, fmt.Errorf("%s: %w", errPrefix, err)
	}
	return c, nil
}

func passwordLoginCandidatesByEmail(ctx context.Context, q authQueryer, email string) ([]passwordLoginCandidate, error) {
	rows, err := q.Query(ctx, `
		SELECT u.id::text, COALESCE(u.tenant_id::text, ''), COALESCE(t.slug, ''), COALESCE(t.name, ''), u.role, u.name, ai.password_hash
		FROM auth_identities ai
		JOIN users u ON u.id = ai.user_id
		LEFT JOIN tenants t ON t.id = u.tenant_id
		WHERE ai.provider = 'password'
		  AND ai.identifier_normalized = $1
		ORDER BY u.created_at ASC
		LIMIT 10
	`, email)
	if err != nil {
		return nil, fmt.Errorf("query login identities: %w", err)
	}
	defer rows.Close()

	var candidates []passwordLoginCandidate
	for rows.Next() {
		var c passwordLoginCandidate
		if err := rows.Scan(&c.userID, &c.tenantID, &c.tenantSlug, &c.tenantName, &c.role, &c.name, &c.passwordHash); err != nil {
			return nil, fmt.Errorf("scan login identity: %w", err)
		}
		candidates = append(candidates, c)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate login identities: %w", err)
	}
	return candidates, nil
}

func touchPasswordIdentityLoginTx(ctx context.Context, tx pgx.Tx, tenantID, email string, now time.Time, errPrefix string) error {
	if tenantID != "" {
		if _, err := tx.Exec(ctx, `
			UPDATE auth_identities
			SET last_login_at = $3,
			    updated_at = $3
			WHERE tenant_id = $1::uuid
			  AND provider = 'password'
			  AND identifier_normalized = $2
		`, tenantID, email, now); err != nil {
			return fmt.Errorf("%s: %w", errPrefix, err)
		}
		return nil
	}

	if _, err := tx.Exec(ctx, `
		UPDATE auth_identities
		SET last_login_at = $2,
		    updated_at = $2
		WHERE tenant_id IS NULL
	  AND provider = 'password'
	  AND identifier_normalized = $1
	`, email, now); err != nil {
		return fmt.Errorf("%s: %w", errPrefix, err)
	}
	return nil
}

func (s *PostgresService) AcceptInvite(ctx context.Context, req AcceptInviteRequest) (Session, error) {
	ctx, cancel := context.WithTimeout(ctx, authDBTimeout)
	defer cancel()

	if strings.TrimSpace(req.Token) == "" || strings.TrimSpace(req.Name) == "" || strings.TrimSpace(req.Password) == "" {
		return Session{}, ErrInvalidInvite
	}

	passwordHash, err := HashPassword(req.Password)
	if err != nil {
		if errors.Is(err, ErrEmptyPassword) {
			return Session{}, ErrInvalidInvite
		}
		return Session{}, fmt.Errorf("hash password: %w", err)
	}

	now := s.now().UTC()
	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return Session{}, fmt.Errorf("begin accept invite transaction: %w", err)
	}
	defer rollbackAuthTx(tx)

	invite, err := loadInviteForAcceptTx(ctx, tx, HashOpaqueToken(req.Token))
	if err != nil {
		return Session{}, err
	}
	if err := validateInviteForAccept(invite, now); err != nil {
		return Session{}, err
	}

	user, err := createInvitedPasswordUserTx(ctx, tx, invite, strings.TrimSpace(req.Name), passwordHash, now)
	if err != nil {
		return Session{}, err
	}

	pair, err := s.issueSession(ctx, tx, user, now)
	if err != nil {
		return Session{}, err
	}

	if err := tx.Commit(ctx); err != nil {
		return Session{}, fmt.Errorf("commit accept invite transaction: %w", err)
	}

	return pair, nil
}

type acceptInviteRecord struct {
	id         string
	tenantID   string
	tenantSlug string
	tenantName string
	email      string
	role       string
	expiresAt  time.Time
	acceptedAt *time.Time
}

func loadInviteForAcceptTx(ctx context.Context, tx pgx.Tx, tokenHash string) (acceptInviteRecord, error) {
	var invite acceptInviteRecord
	err := tx.QueryRow(ctx, `
		SELECT i.id::text, i.tenant_id::text, t.slug, t.name, i.email_normalized, i.role, i.expires_at, i.accepted_at
		FROM auth_invites i
		JOIN tenants t ON t.id = i.tenant_id
		WHERE token_hash = $1
		LIMIT 1
		FOR UPDATE
	`, tokenHash).Scan(&invite.id, &invite.tenantID, &invite.tenantSlug, &invite.tenantName, &invite.email, &invite.role, &invite.expiresAt, &invite.acceptedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return acceptInviteRecord{}, ErrInvalidInvite
		}
		return acceptInviteRecord{}, fmt.Errorf("query invite: %w", err)
	}
	return invite, nil
}

func validateInviteForAccept(invite acceptInviteRecord, now time.Time) error {
	if invite.acceptedAt != nil {
		return ErrInvalidInvite
	}
	if !invite.expiresAt.After(now) {
		return ErrInviteExpired
	}
	return nil
}

func createInvitedPasswordUserTx(ctx context.Context, tx pgx.Tx, invite acceptInviteRecord, name, passwordHash string, now time.Time) (sessionUser, error) {
	var existing bool
	if err := tx.QueryRow(ctx, `
		SELECT EXISTS(
			SELECT 1
			FROM auth_identities
			WHERE tenant_id = $1::uuid
			  AND provider = 'password'
			  AND identifier_normalized = $2
		)
	`, invite.tenantID, invite.email).Scan(&existing); err != nil {
		return sessionUser{}, fmt.Errorf("check existing identity: %w", err)
	}
	if existing {
		return sessionUser{}, ErrInvalidInvite
	}

	var userID string
	if err := tx.QueryRow(ctx, `
		INSERT INTO users (tenant_id, role, name, channel, config, created_at, updated_at)
		VALUES ($1::uuid, $2, $3, 'web', '{}'::jsonb, $4, $4)
		RETURNING id::text
	`, invite.tenantID, invite.role, name, now).Scan(&userID); err != nil {
		return sessionUser{}, fmt.Errorf("insert user: %w", err)
	}

	if _, err := tx.Exec(ctx, `
		INSERT INTO auth_identities (
			user_id, tenant_id, provider, identifier, identifier_normalized, password_hash, email_verified_at, last_login_at, created_at, updated_at
		)
		VALUES ($1::uuid, $2::uuid, 'password', $3, $4, $5, $6, $6, $6, $6)
	`, userID, invite.tenantID, invite.email, invite.email, passwordHash, now); err != nil {
		return sessionUser{}, fmt.Errorf("insert auth identity: %w", err)
	}

	if _, err := tx.Exec(ctx, `
		UPDATE auth_invites
		SET accepted_at = $2
		WHERE id = $1::uuid
	`, invite.id, now); err != nil {
		return sessionUser{}, fmt.Errorf("mark invite accepted: %w", err)
	}

	return sessionUser{
		UserID:     userID,
		TenantID:   invite.tenantID,
		TenantSlug: invite.tenantSlug,
		TenantName: invite.tenantName,
		Role:       Role(invite.role),
		Name:       name,
		Email:      invite.email,
	}, nil
}

func (s *PostgresService) IssueInvite(ctx context.Context, req IssueInviteRequest) (InviteRecord, error) {
	ctx, cancel := context.WithTimeout(ctx, authDBTimeout)
	defer cancel()

	tenantID := strings.TrimSpace(req.TenantID)
	email := NormalizeIdentifier(req.Email)
	if tenantID == "" || email == "" || strings.TrimSpace(req.InvitedByUserID) == "" {
		return InviteRecord{}, ErrInvalidInvite
	}
	if !isInvitableRole(req.Role) {
		return InviteRecord{}, ErrInvalidInvite
	}

	var openInviteExists bool
	if err := s.pool.QueryRow(ctx, `
		SELECT EXISTS(
			SELECT 1
			FROM auth_invites
			WHERE tenant_id = $1::uuid
			  AND email_normalized = $2
			  AND role = $3
			  AND accepted_at IS NULL
			  AND expires_at > $4
		)
	`, tenantID, email, string(req.Role), s.now().UTC()).Scan(&openInviteExists); err != nil {
		return InviteRecord{}, fmt.Errorf("check open invite: %w", err)
	}
	if openInviteExists {
		return InviteRecord{}, ErrInviteConflict
	}

	token, err := generateOpaqueToken()
	if err != nil {
		return InviteRecord{}, fmt.Errorf("generate invite token: %w", err)
	}

	expiresAt := s.now().UTC().Add(7 * 24 * time.Hour)
	var inviteID string
	err = s.pool.QueryRow(ctx, `
		INSERT INTO auth_invites (tenant_id, email, email_normalized, role, token_hash, invited_by, expires_at)
		VALUES ($1::uuid, $2, $3, $4, $5, $6::uuid, $7)
		RETURNING id::text
	`, tenantID, email, email, string(req.Role), HashOpaqueToken(token), req.InvitedByUserID, expiresAt).Scan(&inviteID)
	if err != nil {
		return InviteRecord{}, fmt.Errorf("insert invite: %w", err)
	}

	record := InviteRecord{
		ID:             inviteID,
		Email:          email,
		Role:           req.Role,
		Token:          token,
		ExpiresAt:      expiresAt,
		InvitedByID:    req.InvitedByUserID,
		DeliveryStatus: "pending",
	}
	if err := s.deliverInviteEmail(ctx, inviteID, tenantID, req.InvitedByUserID, req.ActivationBaseURL, &record); err != nil {
		record.DeliveryStatus = "failed"
		record.DeliveryError = err.Error()
	}
	return record, nil
}

func (s *PostgresService) ReissueInvite(ctx context.Context, req ReissueInviteRequest) (InviteRecord, error) {
	ctx, cancel := context.WithTimeout(ctx, authDBTimeout)
	defer cancel()

	tenantID := strings.TrimSpace(req.TenantID)
	inviteID := strings.TrimSpace(req.InviteID)
	invitedByUserID := strings.TrimSpace(req.InvitedByUserID)
	if tenantID == "" || inviteID == "" || invitedByUserID == "" {
		return InviteRecord{}, ErrInvalidInvite
	}

	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return InviteRecord{}, fmt.Errorf("begin reissue invite transaction: %w", err)
	}
	defer rollbackAuthTx(tx)

	var (
		email      string
		role       string
		acceptedAt *time.Time
	)
	err = tx.QueryRow(ctx, `
		SELECT email_normalized, role, accepted_at
		FROM auth_invites
		WHERE id = $1::uuid
		  AND tenant_id = $2::uuid
		FOR UPDATE
	`, inviteID, tenantID).Scan(&email, &role, &acceptedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return InviteRecord{}, ErrInvalidInvite
		}
		return InviteRecord{}, fmt.Errorf("query invite for reissue: %w", err)
	}
	if acceptedAt != nil || !isInvitableRole(Role(role)) {
		return InviteRecord{}, ErrInvalidInvite
	}

	token, err := generateOpaqueToken()
	if err != nil {
		return InviteRecord{}, fmt.Errorf("generate invite token: %w", err)
	}

	expiresAt := s.now().UTC().Add(7 * 24 * time.Hour)
	if _, err := tx.Exec(ctx, `
		UPDATE auth_invites
		SET token_hash = $2,
		    invited_by = $3::uuid,
		    expires_at = $4
		WHERE id = $1::uuid
	`, inviteID, HashOpaqueToken(token), invitedByUserID, expiresAt); err != nil {
		return InviteRecord{}, fmt.Errorf("update invite for reissue: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return InviteRecord{}, fmt.Errorf("commit reissue invite transaction: %w", err)
	}

	record := InviteRecord{
		ID:             inviteID,
		Email:          email,
		Role:           Role(role),
		Token:          token,
		ExpiresAt:      expiresAt,
		InvitedByID:    invitedByUserID,
		DeliveryStatus: "pending",
	}
	if err := s.deliverInviteEmail(ctx, inviteID, tenantID, invitedByUserID, req.ActivationBaseURL, &record); err != nil {
		record.DeliveryStatus = "failed"
		record.DeliveryError = err.Error()
	}
	return record, nil
}

func (s *PostgresService) Session(ctx context.Context, sessionToken string) (Session, error) {
	ctx, cancel := context.WithTimeout(ctx, authDBTimeout)
	defer cancel()

	sessionToken = strings.TrimSpace(sessionToken)
	if sessionToken == "" {
		return Session{}, ErrInvalidCredentials
	}

	now := s.now().UTC()
	var (
		userID     string
		tenantID   string
		tenantSlug string
		tenantName string
		role       string
		name       string
		email      string
		expiresAt  time.Time
		revokedAt  *time.Time
	)
	err := s.pool.QueryRow(ctx, `
		SELECT u.id::text,
		       COALESCE(u.tenant_id::text, ''),
		       COALESCE(t.slug, ''),
		       COALESCE(t.name, ''),
		       u.role,
		       u.name,
		       COALESCE(
		           (SELECT identifier_normalized FROM auth_identities WHERE user_id = u.id AND provider = 'password' ORDER BY created_at ASC LIMIT 1),
		           (SELECT provider_email FROM auth_identities WHERE user_id = u.id AND provider = 'google' ORDER BY created_at ASC LIMIT 1),
		           ''
		       ),
		       rt.expires_at,
		       rt.revoked_at
		FROM auth_sessions rt
		JOIN users u ON u.id = rt.user_id
		LEFT JOIN tenants t ON t.id = u.tenant_id
		WHERE rt.token_hash = $1
		LIMIT 1
	`, HashOpaqueToken(sessionToken)).Scan(&userID, &tenantID, &tenantSlug, &tenantName, &role, &name, &email, &expiresAt, &revokedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return Session{}, ErrInvalidCredentials
		}
		return Session{}, fmt.Errorf("query session: %w", err)
	}
	if revokedAt != nil || !expiresAt.After(now) {
		return Session{}, ErrInvalidCredentials
	}

	if shouldRefreshAuthSession(now, expiresAt, s.sessionTTL) {
		expiresAt = now.Add(s.sessionTTL)
		tag, err := s.pool.Exec(ctx, `
			UPDATE auth_sessions
			SET expires_at = $2
			WHERE token_hash = $1
			  AND revoked_at IS NULL
			  AND expires_at > $3
		`, HashOpaqueToken(sessionToken), expiresAt, now)
		if err != nil {
			return Session{}, fmt.Errorf("extend session: %w", err)
		}
		if tag.RowsAffected() == 0 {
			return Session{}, ErrInvalidCredentials
		}
	}

	tenantChoices, err := s.tenantOptionsByEmail(ctx, s.pool, email)
	if err != nil {
		return Session{}, err
	}

	session := Session{
		Token:     sessionToken,
		ExpiresAt: expiresAt,
		User: UserSession{
			UserID:     userID,
			TenantID:   tenantID,
			TenantSlug: tenantSlug,
			TenantName: tenantName,
			Role:       Role(role),
			Name:       name,
			Email:      email,
		},
	}
	session.TenantChoices = tenantChoices
	return session, nil
}

func (s *PostgresService) SwitchTenant(ctx context.Context, sessionToken, tenantID, password string) (Session, error) {
	ctx, cancel := context.WithTimeout(ctx, authDBTimeout)
	defer cancel()

	if strings.TrimSpace(sessionToken) == "" || strings.TrimSpace(tenantID) == "" || strings.TrimSpace(password) == "" {
		return Session{}, ErrInvalidCredentials
	}

	now := s.now().UTC()
	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return Session{}, fmt.Errorf("begin switch tenant transaction: %w", err)
	}
	defer rollbackAuthTx(tx)

	current, err := loadSwitchTenantSessionTx(ctx, tx, HashOpaqueToken(sessionToken))
	if err != nil {
		return Session{}, err
	}
	if current.revokedAt != nil || !current.expiresAt.After(now) || strings.TrimSpace(current.email) == "" {
		return Session{}, ErrInvalidCredentials
	}

	candidate, err := passwordLoginCandidateByTenant(ctx, tx, tenantID, current.email, "query target tenant identity")
	if err != nil {
		return Session{}, err
	}
	if err := ComparePassword(candidate.passwordHash, password); err != nil {
		return Session{}, ErrInvalidCredentials
	}

	if err := touchPasswordIdentityLoginTx(ctx, tx, candidate.tenantID, current.email, now, "update switch tenant last_login_at"); err != nil {
		return Session{}, err
	}

	pair, newTokenID, err := s.issueSessionWithID(ctx, tx, candidate.sessionUser(current.email), now)
	if err != nil {
		return Session{}, err
	}

	if _, err := tx.Exec(ctx, `
		UPDATE auth_sessions
		SET revoked_at = $2,
		    replaced_by = $3::uuid
		WHERE id = $1::uuid
	`, current.tokenID, now, newTokenID); err != nil {
		return Session{}, fmt.Errorf("revoke previous session: %w", err)
	}

	pair.TenantChoices, err = s.tenantOptionsByEmail(ctx, tx, current.email)
	if err != nil {
		return Session{}, err
	}

	if err := tx.Commit(ctx); err != nil {
		return Session{}, fmt.Errorf("commit switch tenant transaction: %w", err)
	}
	return pair, nil
}

type switchTenantSession struct {
	tokenID   string
	email     string
	expiresAt time.Time
	revokedAt *time.Time
}

func loadSwitchTenantSessionTx(ctx context.Context, tx pgx.Tx, tokenHash string) (switchTenantSession, error) {
	var current switchTenantSession
	err := tx.QueryRow(ctx, `
		SELECT rt.id::text, COALESCE(ai.identifier_normalized, ''), rt.expires_at, rt.revoked_at
		FROM auth_sessions rt
		JOIN users u ON u.id = rt.user_id
		LEFT JOIN auth_identities ai
		  ON ai.user_id = u.id
		 AND ai.provider = 'password'
		WHERE rt.token_hash = $1
		LIMIT 1
		FOR UPDATE OF rt, u
	`, tokenHash).Scan(&current.tokenID, &current.email, &current.expiresAt, &current.revokedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return switchTenantSession{}, ErrInvalidCredentials
		}
		return switchTenantSession{}, fmt.Errorf("query switch tenant token: %w", err)
	}
	return current, nil
}

func (s *PostgresService) Logout(ctx context.Context, sessionToken string) error {
	ctx, cancel := context.WithTimeout(ctx, authDBTimeout)
	defer cancel()

	if strings.TrimSpace(sessionToken) == "" {
		return ErrInvalidCredentials
	}

	now := s.now().UTC()
	_, err := s.pool.Exec(ctx, `
		UPDATE auth_sessions
		SET revoked_at = COALESCE(revoked_at, $2)
		WHERE token_hash = $1
	`, HashOpaqueToken(sessionToken), now)
	if err != nil {
		return fmt.Errorf("logout session: %w", err)
	}
	return nil
}

type sessionUser struct {
	UserID     string
	TenantID   string
	TenantSlug string
	TenantName string
	Role       Role
	Name       string
	Email      string
}

func (s *PostgresService) issueSession(ctx context.Context, tx pgx.Tx, user sessionUser, now time.Time) (Session, error) {
	session, _, err := s.issueSessionWithID(ctx, tx, user, now)
	return session, err
}

func (s *PostgresService) issueSessionWithID(ctx context.Context, tx pgx.Tx, user sessionUser, now time.Time) (Session, string, error) {
	sessionToken, err := generateOpaqueToken()
	if err != nil {
		return Session{}, "", fmt.Errorf("generate session token: %w", err)
	}

	sessionExpiresAt := now.Add(s.sessionTTL)
	var tokenID string
	if user.TenantID != "" {
		if err := tx.QueryRow(ctx, `
			INSERT INTO auth_sessions (user_id, tenant_id, token_hash, expires_at, created_at)
			VALUES ($1::uuid, $2::uuid, $3, $4, $5)
			RETURNING id::text
		`, user.UserID, user.TenantID, HashOpaqueToken(sessionToken), sessionExpiresAt, now).Scan(&tokenID); err != nil {
			return Session{}, "", fmt.Errorf("insert session: %w", err)
		}
	} else {
		if err := tx.QueryRow(ctx, `
			INSERT INTO auth_sessions (user_id, tenant_id, token_hash, expires_at, created_at)
			VALUES ($1::uuid, NULL, $2, $3, $4)
			RETURNING id::text
		`, user.UserID, HashOpaqueToken(sessionToken), sessionExpiresAt, now).Scan(&tokenID); err != nil {
			return Session{}, "", fmt.Errorf("insert session: %w", err)
		}
	}

	return Session{
		Token:     sessionToken,
		ExpiresAt: sessionExpiresAt,
		User:      UserSession(user),
	}, tokenID, nil
}

func generateOpaqueToken() (string, error) {
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(buf), nil
}

type authQueryer interface {
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
}

type authRowQueryer interface {
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}

func rollbackAuthTx(tx pgx.Tx) {
	ctx, cancel := context.WithTimeout(context.Background(), authRollbackTimeout)
	defer cancel()
	_ = tx.Rollback(ctx)
}

func shouldRefreshAuthSession(now, expiresAt time.Time, ttl time.Duration) bool {
	if ttl <= 0 {
		return true
	}
	window := ttl / authSessionRefreshDivisor
	if window <= 0 || window > authSessionRefreshWindow {
		window = authSessionRefreshWindow
	}
	return !expiresAt.After(now.Add(window))
}

func (s *PostgresService) tenantOptionsByEmail(ctx context.Context, q authQueryer, email string) ([]TenantOption, error) {
	email = NormalizeIdentifier(email)
	if email == "" {
		return nil, nil
	}
	rows, err := q.Query(ctx, `
		SELECT COALESCE(u.tenant_id::text, ''),
		       COALESCE(t.slug, ''),
		       COALESCE(t.name, '')
		FROM auth_identities ai
		JOIN users u ON u.id = ai.user_id
		LEFT JOIN tenants t ON t.id = u.tenant_id
		WHERE ai.provider = 'password'
		  AND ai.identifier_normalized = $1
		  AND u.tenant_id IS NOT NULL
		ORDER BY u.created_at ASC
		LIMIT 10
	`, email)
	if err != nil {
		return nil, fmt.Errorf("query tenant options: %w", err)
	}
	defer rows.Close()

	options := make([]TenantOption, 0)
	seen := map[string]struct{}{}
	for rows.Next() {
		var option TenantOption
		if err := rows.Scan(&option.TenantID, &option.TenantSlug, &option.TenantName); err != nil {
			return nil, fmt.Errorf("scan tenant option: %w", err)
		}
		if option.TenantID == "" {
			continue
		}
		if _, ok := seen[option.TenantID]; ok {
			continue
		}
		seen[option.TenantID] = struct{}{}
		options = append(options, option)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate tenant options: %w", err)
	}
	if len(options) <= 1 {
		return nil, nil
	}
	return options, nil
}

func isInvitableRole(role Role) bool {
	switch role {
	case RoleTeacher, RoleParent, RoleAdmin:
		return true
	default:
		return false
	}
}

type inviteEmailContext struct {
	tenantName  string
	inviterName string
}

func (s *PostgresService) deliverInviteEmail(ctx context.Context, inviteID, tenantID, invitedByUserID, baseURL string, record *InviteRecord) error {
	if record == nil {
		return errors.New("invite record is required")
	}

	record.ActivationURL = buildInviteActivationURL(baseURL, record.Token)
	if s.inviteMailSender == nil {
		return nil
	}

	if strings.TrimSpace(record.ActivationURL) == "" {
		return s.markInviteDeliveryFailure(ctx, inviteID, "activation base url is required")
	}

	deliveryCtx, cancel := context.WithTimeout(ctx, authDBTimeout)
	defer cancel()

	emailCtx, err := s.loadInviteEmailContext(deliveryCtx, tenantID, invitedByUserID)
	if err != nil {
		return s.markInviteDeliveryFailure(deliveryCtx, inviteID, err.Error())
	}

	if err := s.inviteMailSender.SendInvite(deliveryCtx, mailer.InviteMessage{
		ToEmail:       record.Email,
		TenantName:    emailCtx.tenantName,
		InviterName:   emailCtx.inviterName,
		RoleLabel:     string(record.Role),
		ActivationURL: record.ActivationURL,
		ExpiresAt:     record.ExpiresAt,
	}); err != nil {
		return s.markInviteDeliveryFailure(deliveryCtx, inviteID, err.Error())
	}

	if _, err := s.pool.Exec(deliveryCtx, `
		UPDATE auth_invites
		SET delivery_status = 'sent',
		    delivery_attempted_at = $2,
		    delivery_sent_at = $2,
		    delivery_error = NULL
		WHERE id = $1::uuid
	`, inviteID, s.now().UTC()); err != nil {
		return fmt.Errorf("update invite delivery sent status: %w", err)
	}

	record.DeliveryStatus = "sent"
	record.DeliveryError = ""
	return nil
}

func (s *PostgresService) markInviteDeliveryFailure(ctx context.Context, inviteID, reason string) error {
	if _, err := s.pool.Exec(ctx, `
		UPDATE auth_invites
		SET delivery_status = 'failed',
		    delivery_attempted_at = $2,
		    delivery_error = $3
		WHERE id = $1::uuid
	`, inviteID, s.now().UTC(), strings.TrimSpace(reason)); err != nil {
		return fmt.Errorf("update invite delivery failure: %w", err)
	}
	return fmt.Errorf("%s", strings.TrimSpace(reason))
}

func (s *PostgresService) loadInviteEmailContext(ctx context.Context, tenantID, invitedByUserID string) (inviteEmailContext, error) {
	var result inviteEmailContext
	if err := s.pool.QueryRow(ctx, `
		SELECT COALESCE(t.name, ''), COALESCE(u.name, '')
		FROM tenants t
		LEFT JOIN users u ON u.id = $2::uuid
		WHERE t.id = $1::uuid
		LIMIT 1
	`, tenantID, invitedByUserID).Scan(&result.tenantName, &result.inviterName); err != nil {
		return inviteEmailContext{}, fmt.Errorf("load invite email context: %w", err)
	}
	return result, nil
}

func buildInviteActivationURL(baseURL, token string) string {
	baseURL = strings.TrimSpace(baseURL)
	token = strings.TrimSpace(token)
	if baseURL == "" || token == "" {
		return ""
	}
	return strings.TrimRight(baseURL, "/") + "/activate?token=" + url.QueryEscape(token)
}
