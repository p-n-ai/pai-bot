package auth

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

const authDBTimeout = 5 * time.Second

type PostgresService struct {
	pool            *pgxpool.Pool
	tokenManager    *TokenManager
	refreshTokenTTL time.Duration
	now             func() time.Time
}

func NewPostgresService(pool *pgxpool.Pool, jwtSecret string, accessTokenTTL, refreshTokenTTL time.Duration) *PostgresService {
	return newPostgresService(pool, jwtSecret, accessTokenTTL, refreshTokenTTL, time.Now)
}

func newPostgresService(pool *pgxpool.Pool, jwtSecret string, accessTokenTTL, refreshTokenTTL time.Duration, now func() time.Time) *PostgresService {
	if now == nil {
		now = time.Now
	}

	return &PostgresService{
		pool:            pool,
		tokenManager:    NewTokenManager(jwtSecret, accessTokenTTL),
		refreshTokenTTL: refreshTokenTTL,
		now:             now,
	}
}

func (s *PostgresService) Login(ctx context.Context, req LoginRequest) (TokenPair, error) {
	ctx, cancel := context.WithTimeout(ctx, authDBTimeout)
	defer cancel()

	tenantID := strings.TrimSpace(req.TenantID)
	email := NormalizeIdentifier(req.Email)
	if email == "" || strings.TrimSpace(req.Password) == "" {
		return TokenPair{}, ErrInvalidCredentials
	}

	var (
		userID       string
		resolvedTID  string
		tenantSlug   string
		tenantName   string
		role         string
		name         string
		passwordHash string
	)
	if tenantID != "" {
		err := s.pool.QueryRow(ctx, `
			SELECT u.id::text, u.tenant_id::text, t.slug, t.name, u.role, u.name, ai.password_hash
			FROM auth_identities ai
			JOIN users u ON u.id = ai.user_id
			JOIN tenants t ON t.id = u.tenant_id
			WHERE ai.tenant_id = $1::uuid
			  AND ai.provider = 'password'
			  AND ai.identifier_normalized = $2
			LIMIT 1
		`, tenantID, email).Scan(&userID, &resolvedTID, &tenantSlug, &tenantName, &role, &name, &passwordHash)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return TokenPair{}, ErrInvalidCredentials
			}
			return TokenPair{}, fmt.Errorf("query login identity: %w", err)
		}
	} else {
		rows, err := s.pool.Query(ctx, `
			SELECT u.id::text, u.tenant_id::text, t.slug, t.name, u.role, u.name, ai.password_hash
			FROM auth_identities ai
			JOIN users u ON u.id = ai.user_id
			JOIN tenants t ON t.id = u.tenant_id
			WHERE ai.provider = 'password'
			  AND ai.identifier_normalized = $1
			ORDER BY u.created_at ASC
			LIMIT 10
		`, email)
		if err != nil {
			return TokenPair{}, fmt.Errorf("query login identities: %w", err)
		}
		defer rows.Close()

		type candidate struct {
			userID       string
			tenantID     string
			tenantSlug   string
			tenantName   string
			role         string
			name         string
			passwordHash string
		}

		var candidates []candidate
		for rows.Next() {
			var c candidate
			if err := rows.Scan(&c.userID, &c.tenantID, &c.tenantSlug, &c.tenantName, &c.role, &c.name, &c.passwordHash); err != nil {
				return TokenPair{}, fmt.Errorf("scan login identity: %w", err)
			}
			candidates = append(candidates, c)
		}
		if err := rows.Err(); err != nil {
			return TokenPair{}, fmt.Errorf("iterate login identities: %w", err)
		}
		switch len(candidates) {
		case 0:
			return TokenPair{}, ErrInvalidCredentials
		case 1:
			chosen := candidates[0]
			userID = chosen.userID
			resolvedTID = chosen.tenantID
			tenantSlug = chosen.tenantSlug
			tenantName = chosen.tenantName
			role = chosen.role
			name = chosen.name
			passwordHash = chosen.passwordHash
			tenantID = resolvedTID
		default:
			options := make([]TenantOption, 0, len(candidates))
			for _, candidate := range candidates {
				options = append(options, TenantOption{
					TenantID:   candidate.tenantID,
					TenantSlug: candidate.tenantSlug,
					TenantName: candidate.tenantName,
				})
			}
			return TokenPair{}, NewTenantRequiredError(options)
		}
	}
	if err := ComparePassword(passwordHash, req.Password); err != nil {
		return TokenPair{}, ErrInvalidCredentials
	}

	if tenantID == "" {
		tenantID = resolvedTID
	}

	now := s.now().UTC()
	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return TokenPair{}, fmt.Errorf("begin login transaction: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	if _, err := tx.Exec(ctx, `
		UPDATE auth_identities
		SET last_login_at = $3,
		    updated_at = $3
		WHERE tenant_id = $1::uuid
		  AND provider = 'password'
		  AND identifier_normalized = $2
	`, tenantID, email, now); err != nil {
		return TokenPair{}, fmt.Errorf("update last_login_at: %w", err)
	}

	pair, err := s.issueSession(ctx, tx, sessionUser{
		UserID:     userID,
		TenantID:   tenantID,
		TenantSlug: tenantSlug,
		TenantName: tenantName,
		Role:       Role(role),
		Name:       name,
		Email:      email,
	}, now)
	if err != nil {
		return TokenPair{}, err
	}

	if err := tx.Commit(ctx); err != nil {
		return TokenPair{}, fmt.Errorf("commit login transaction: %w", err)
	}

	return pair, nil
}

func (s *PostgresService) AcceptInvite(ctx context.Context, req AcceptInviteRequest) (TokenPair, error) {
	ctx, cancel := context.WithTimeout(ctx, authDBTimeout)
	defer cancel()

	if strings.TrimSpace(req.Token) == "" || strings.TrimSpace(req.Name) == "" || strings.TrimSpace(req.Password) == "" {
		return TokenPair{}, ErrInvalidInvite
	}

	passwordHash, err := HashPassword(req.Password)
	if err != nil {
		if errors.Is(err, ErrEmptyPassword) {
			return TokenPair{}, ErrInvalidInvite
		}
		return TokenPair{}, fmt.Errorf("hash password: %w", err)
	}

	now := s.now().UTC()
	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return TokenPair{}, fmt.Errorf("begin accept invite transaction: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var (
		inviteID   string
		tenantID   string
		tenantSlug string
		tenantName string
		email      string
		role       string
		expires    time.Time
		accepted   *time.Time
	)
	err = tx.QueryRow(ctx, `
		SELECT i.id::text, i.tenant_id::text, t.slug, t.name, i.email_normalized, i.role, i.expires_at, i.accepted_at
		FROM auth_invites i
		JOIN tenants t ON t.id = i.tenant_id
		WHERE token_hash = $1
		LIMIT 1
		FOR UPDATE
	`, HashOpaqueToken(req.Token)).Scan(&inviteID, &tenantID, &tenantSlug, &tenantName, &email, &role, &expires, &accepted)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return TokenPair{}, ErrInvalidInvite
		}
		return TokenPair{}, fmt.Errorf("query invite: %w", err)
	}
	if accepted != nil {
		return TokenPair{}, ErrInvalidInvite
	}
	if !expires.After(now) {
		return TokenPair{}, ErrInviteExpired
	}

	var existing bool
	if err := tx.QueryRow(ctx, `
		SELECT EXISTS(
			SELECT 1
			FROM auth_identities
			WHERE tenant_id = $1::uuid
			  AND provider = 'password'
			  AND identifier_normalized = $2
		)
	`, tenantID, email).Scan(&existing); err != nil {
		return TokenPair{}, fmt.Errorf("check existing identity: %w", err)
	}
	if existing {
		return TokenPair{}, ErrInvalidInvite
	}

	var userID string
	if err := tx.QueryRow(ctx, `
		INSERT INTO users (tenant_id, role, name, channel, config, created_at, updated_at)
		VALUES ($1::uuid, $2, $3, 'web', '{}'::jsonb, $4, $4)
		RETURNING id::text
	`, tenantID, role, strings.TrimSpace(req.Name), now).Scan(&userID); err != nil {
		return TokenPair{}, fmt.Errorf("insert user: %w", err)
	}

	if _, err := tx.Exec(ctx, `
		INSERT INTO auth_identities (
			user_id, tenant_id, provider, identifier, identifier_normalized, password_hash, email_verified_at, last_login_at, created_at, updated_at
		)
		VALUES ($1::uuid, $2::uuid, 'password', $3, $4, $5, $6, $6, $6, $6)
	`, userID, tenantID, email, email, passwordHash, now); err != nil {
		return TokenPair{}, fmt.Errorf("insert auth identity: %w", err)
	}

	if _, err := tx.Exec(ctx, `
		UPDATE auth_invites
		SET accepted_at = $2
		WHERE id = $1::uuid
	`, inviteID, now); err != nil {
		return TokenPair{}, fmt.Errorf("mark invite accepted: %w", err)
	}

	pair, err := s.issueSession(ctx, tx, sessionUser{
		UserID:     userID,
		TenantID:   tenantID,
		TenantSlug: tenantSlug,
		TenantName: tenantName,
		Role:       Role(role),
		Name:       strings.TrimSpace(req.Name),
		Email:      email,
	}, now)
	if err != nil {
		return TokenPair{}, err
	}

	if err := tx.Commit(ctx); err != nil {
		return TokenPair{}, fmt.Errorf("commit accept invite transaction: %w", err)
	}

	return pair, nil
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
	_, err = s.pool.Exec(ctx, `
		INSERT INTO auth_invites (tenant_id, email, email_normalized, role, token_hash, invited_by, expires_at)
		VALUES ($1::uuid, $2, $3, $4, $5, $6::uuid, $7)
	`, tenantID, email, email, string(req.Role), HashOpaqueToken(token), req.InvitedByUserID, expiresAt)
	if err != nil {
		return InviteRecord{}, fmt.Errorf("insert invite: %w", err)
	}

	return InviteRecord{
		Email:       email,
		Role:        req.Role,
		Token:       token,
		ExpiresAt:   expiresAt,
		InvitedByID: req.InvitedByUserID,
	}, nil
}

func (s *PostgresService) Refresh(ctx context.Context, refreshToken string) (TokenPair, error) {
	ctx, cancel := context.WithTimeout(ctx, authDBTimeout)
	defer cancel()

	if strings.TrimSpace(refreshToken) == "" {
		return TokenPair{}, ErrInvalidCredentials
	}

	now := s.now().UTC()
	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return TokenPair{}, fmt.Errorf("begin refresh transaction: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var (
		tokenID    string
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
	err = tx.QueryRow(ctx, `
		SELECT rt.id::text, u.id::text, u.tenant_id::text, t.slug, t.name, u.role, u.name, ai.identifier_normalized, rt.expires_at, rt.revoked_at
		FROM auth_refresh_tokens rt
		JOIN users u ON u.id = rt.user_id
		JOIN tenants t ON t.id = u.tenant_id
		LEFT JOIN auth_identities ai
		  ON ai.user_id = u.id
		 AND ai.provider = 'password'
		WHERE rt.token_hash = $1
		LIMIT 1
		FOR UPDATE
	`, HashOpaqueToken(refreshToken)).Scan(&tokenID, &userID, &tenantID, &tenantSlug, &tenantName, &role, &name, &email, &expiresAt, &revokedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return TokenPair{}, ErrInvalidCredentials
		}
		return TokenPair{}, fmt.Errorf("query refresh token: %w", err)
	}
	if revokedAt != nil || !expiresAt.After(now) {
		return TokenPair{}, ErrInvalidCredentials
	}

	pair, newTokenID, err := s.issueSessionWithID(ctx, tx, sessionUser{
		UserID:     userID,
		TenantID:   tenantID,
		TenantSlug: tenantSlug,
		TenantName: tenantName,
		Role:       Role(role),
		Name:       name,
		Email:      email,
	}, now)
	if err != nil {
		return TokenPair{}, err
	}

	if _, err := tx.Exec(ctx, `
		UPDATE auth_refresh_tokens
		SET revoked_at = $2,
		    replaced_by = $3::uuid
		WHERE id = $1::uuid
	`, tokenID, now, newTokenID); err != nil {
		return TokenPair{}, fmt.Errorf("revoke previous refresh token: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return TokenPair{}, fmt.Errorf("commit refresh transaction: %w", err)
	}

	return pair, nil
}

func (s *PostgresService) Logout(ctx context.Context, refreshToken string) error {
	ctx, cancel := context.WithTimeout(ctx, authDBTimeout)
	defer cancel()

	if strings.TrimSpace(refreshToken) == "" {
		return ErrInvalidCredentials
	}

	now := s.now().UTC()
	_, err := s.pool.Exec(ctx, `
		UPDATE auth_refresh_tokens
		SET revoked_at = COALESCE(revoked_at, $2)
		WHERE token_hash = $1
	`, HashOpaqueToken(refreshToken), now)
	if err != nil {
		return fmt.Errorf("logout refresh token: %w", err)
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

func (s *PostgresService) issueSession(ctx context.Context, tx pgx.Tx, user sessionUser, now time.Time) (TokenPair, error) {
	pair, _, err := s.issueSessionWithID(ctx, tx, user, now)
	return pair, err
}

func (s *PostgresService) issueSessionWithID(ctx context.Context, tx pgx.Tx, user sessionUser, now time.Time) (TokenPair, string, error) {
	accessToken, err := s.tokenManager.Issue(TokenClaims{
		Subject:  user.UserID,
		TenantID: user.TenantID,
		Role:     user.Role,
	}, now)
	if err != nil {
		return TokenPair{}, "", fmt.Errorf("issue access token: %w", err)
	}

	refreshToken, err := generateOpaqueToken()
	if err != nil {
		return TokenPair{}, "", fmt.Errorf("generate refresh token: %w", err)
	}

	refreshExpiresAt := now.Add(s.refreshTokenTTL)
	var tokenID string
	if err := tx.QueryRow(ctx, `
		INSERT INTO auth_refresh_tokens (user_id, tenant_id, token_hash, expires_at, created_at)
		VALUES ($1::uuid, $2::uuid, $3, $4, $5)
		RETURNING id::text
	`, user.UserID, user.TenantID, HashOpaqueToken(refreshToken), refreshExpiresAt, now).Scan(&tokenID); err != nil {
		return TokenPair{}, "", fmt.Errorf("insert refresh token: %w", err)
	}

	return TokenPair{
		AccessToken:      accessToken,
		RefreshToken:     refreshToken,
		AccessExpiresAt:  now.Add(s.tokenManager.ttl),
		RefreshExpiresAt: refreshExpiresAt,
		User: UserSession{
			UserID:     user.UserID,
			TenantID:   user.TenantID,
			TenantSlug: user.TenantSlug,
			TenantName: user.TenantName,
			Role:       user.Role,
			Name:       user.Name,
			Email:      user.Email,
		},
	}, tokenID, nil
}

func generateOpaqueToken() (string, error) {
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(buf), nil
}

func isInvitableRole(role Role) bool {
	switch role {
	case RoleTeacher, RoleParent, RoleAdmin:
		return true
	default:
		return false
	}
}
