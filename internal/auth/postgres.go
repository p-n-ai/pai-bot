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
	pool       *pgxpool.Pool
	sessionTTL time.Duration
	httpClient HTTPDoer
	google     *GoogleOAuthProvider
	now        func() time.Time
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

func (s *PostgresService) Login(ctx context.Context, req LoginRequest) (Session, error) {
	ctx, cancel := context.WithTimeout(ctx, authDBTimeout)
	defer cancel()

	tenantID := strings.TrimSpace(req.TenantID)
	email := NormalizeIdentifier(req.Email)
	if email == "" || strings.TrimSpace(req.Password) == "" {
		return Session{}, ErrInvalidCredentials
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
			SELECT u.id::text, COALESCE(u.tenant_id::text, ''), COALESCE(t.slug, ''), COALESCE(t.name, ''), u.role, u.name, ai.password_hash
			FROM auth_identities ai
			JOIN users u ON u.id = ai.user_id
			LEFT JOIN tenants t ON t.id = u.tenant_id
			WHERE ai.tenant_id = $1::uuid
			  AND ai.provider = 'password'
			  AND ai.identifier_normalized = $2
			LIMIT 1
		`, tenantID, email).Scan(&userID, &resolvedTID, &tenantSlug, &tenantName, &role, &name, &passwordHash)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return Session{}, ErrInvalidCredentials
			}
			return Session{}, fmt.Errorf("query login identity: %w", err)
		}
	} else {
		rows, err := s.pool.Query(ctx, `
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
			return Session{}, fmt.Errorf("query login identities: %w", err)
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
				return Session{}, fmt.Errorf("scan login identity: %w", err)
			}
			candidates = append(candidates, c)
		}
		if err := rows.Err(); err != nil {
			return Session{}, fmt.Errorf("iterate login identities: %w", err)
		}
		switch len(candidates) {
		case 0:
			return Session{}, ErrInvalidCredentials
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
			chosen := candidates[0]
			userID = chosen.userID
			resolvedTID = chosen.tenantID
			tenantSlug = chosen.tenantSlug
			tenantName = chosen.tenantName
			role = chosen.role
			name = chosen.name
			passwordHash = chosen.passwordHash
			tenantID = resolvedTID
		}
	}
	if err := ComparePassword(passwordHash, req.Password); err != nil {
		return Session{}, ErrInvalidCredentials
	}

	if tenantID == "" {
		tenantID = resolvedTID
	}

	now := s.now().UTC()
	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return Session{}, fmt.Errorf("begin login transaction: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	if tenantID != "" {
		if _, err := tx.Exec(ctx, `
			UPDATE auth_identities
			SET last_login_at = $3,
			    updated_at = $3
			WHERE tenant_id = $1::uuid
			  AND provider = 'password'
			  AND identifier_normalized = $2
		`, tenantID, email, now); err != nil {
			return Session{}, fmt.Errorf("update last_login_at: %w", err)
		}
	} else {
		if _, err := tx.Exec(ctx, `
			UPDATE auth_identities
			SET last_login_at = $2,
			    updated_at = $2
			WHERE tenant_id IS NULL
			  AND provider = 'password'
			  AND identifier_normalized = $1
		`, email, now); err != nil {
			return Session{}, fmt.Errorf("update last_login_at: %w", err)
		}
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
			return Session{}, ErrInvalidInvite
		}
		return Session{}, fmt.Errorf("query invite: %w", err)
	}
	if accepted != nil {
		return Session{}, ErrInvalidInvite
	}
	if !expires.After(now) {
		return Session{}, ErrInviteExpired
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
		return Session{}, fmt.Errorf("check existing identity: %w", err)
	}
	if existing {
		return Session{}, ErrInvalidInvite
	}

	var userID string
	if err := tx.QueryRow(ctx, `
		INSERT INTO users (tenant_id, role, name, channel, config, created_at, updated_at)
		VALUES ($1::uuid, $2, $3, 'web', '{}'::jsonb, $4, $4)
		RETURNING id::text
	`, tenantID, role, strings.TrimSpace(req.Name), now).Scan(&userID); err != nil {
		return Session{}, fmt.Errorf("insert user: %w", err)
	}

	if _, err := tx.Exec(ctx, `
		INSERT INTO auth_identities (
			user_id, tenant_id, provider, identifier, identifier_normalized, password_hash, email_verified_at, last_login_at, created_at, updated_at
		)
		VALUES ($1::uuid, $2::uuid, 'password', $3, $4, $5, $6, $6, $6, $6)
	`, userID, tenantID, email, email, passwordHash, now); err != nil {
		return Session{}, fmt.Errorf("insert auth identity: %w", err)
	}

	if _, err := tx.Exec(ctx, `
		UPDATE auth_invites
		SET accepted_at = $2
		WHERE id = $1::uuid
	`, inviteID, now); err != nil {
		return Session{}, fmt.Errorf("mark invite accepted: %w", err)
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
		return Session{}, err
	}

	if err := tx.Commit(ctx); err != nil {
		return Session{}, fmt.Errorf("commit accept invite transaction: %w", err)
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
	defer func() { _ = tx.Rollback(ctx) }()

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

	return InviteRecord{
		Email:       email,
		Role:        Role(role),
		Token:       token,
		ExpiresAt:   expiresAt,
		InvitedByID: invitedByUserID,
	}, nil
}

func (s *PostgresService) Session(ctx context.Context, sessionToken string) (Session, error) {
	ctx, cancel := context.WithTimeout(ctx, authDBTimeout)
	defer cancel()

	sessionToken = strings.TrimSpace(sessionToken)
	if sessionToken == "" {
		return Session{}, ErrInvalidCredentials
	}

	now := s.now().UTC()
	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return Session{}, fmt.Errorf("begin session transaction: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

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
	err = tx.QueryRow(ctx, `
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
		FOR UPDATE OF rt, u
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

	expiresAt = now.Add(s.sessionTTL)
	if _, err := tx.Exec(ctx, `
		UPDATE auth_sessions
		SET expires_at = $2
		WHERE token_hash = $1
	`, HashOpaqueToken(sessionToken), expiresAt); err != nil {
		return Session{}, fmt.Errorf("extend session: %w", err)
	}

	tenantChoices, err := s.tenantOptionsByEmail(ctx, tx, email)
	if err != nil {
		return Session{}, err
	}

	if err := tx.Commit(ctx); err != nil {
		return Session{}, fmt.Errorf("commit session transaction: %w", err)
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
	defer func() { _ = tx.Rollback(ctx) }()

	var (
		tokenID      string
		currentEmail string
		expiresAt    time.Time
		revokedAt    *time.Time
	)
	err = tx.QueryRow(ctx, `
		SELECT rt.id::text, COALESCE(ai.identifier_normalized, ''), rt.expires_at, rt.revoked_at
		FROM auth_sessions rt
		JOIN users u ON u.id = rt.user_id
		LEFT JOIN auth_identities ai
		  ON ai.user_id = u.id
		 AND ai.provider = 'password'
		WHERE rt.token_hash = $1
		LIMIT 1
		FOR UPDATE OF rt, u
	`, HashOpaqueToken(sessionToken)).Scan(&tokenID, &currentEmail, &expiresAt, &revokedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return Session{}, ErrInvalidCredentials
		}
		return Session{}, fmt.Errorf("query switch tenant token: %w", err)
	}
	if revokedAt != nil || !expiresAt.After(now) || strings.TrimSpace(currentEmail) == "" {
		return Session{}, ErrInvalidCredentials
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
	err = tx.QueryRow(ctx, `
		SELECT u.id::text, COALESCE(u.tenant_id::text, ''), COALESCE(t.slug, ''), COALESCE(t.name, ''), u.role, u.name, ai.password_hash
		FROM auth_identities ai
		JOIN users u ON u.id = ai.user_id
		LEFT JOIN tenants t ON t.id = u.tenant_id
		WHERE ai.tenant_id = $1::uuid
		  AND ai.provider = 'password'
		  AND ai.identifier_normalized = $2
		LIMIT 1
	`, tenantID, currentEmail).Scan(&userID, &resolvedTID, &tenantSlug, &tenantName, &role, &name, &passwordHash)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return Session{}, ErrInvalidCredentials
		}
		return Session{}, fmt.Errorf("query target tenant identity: %w", err)
	}
	if err := ComparePassword(passwordHash, password); err != nil {
		return Session{}, ErrInvalidCredentials
	}

	if _, err := tx.Exec(ctx, `
		UPDATE auth_identities
		SET last_login_at = $3,
		    updated_at = $3
		WHERE tenant_id = $1::uuid
		  AND provider = 'password'
		  AND identifier_normalized = $2
	`, resolvedTID, currentEmail, now); err != nil {
		return Session{}, fmt.Errorf("update switch tenant last_login_at: %w", err)
	}

	pair, newTokenID, err := s.issueSessionWithID(ctx, tx, sessionUser{
		UserID:     userID,
		TenantID:   resolvedTID,
		TenantSlug: tenantSlug,
		TenantName: tenantName,
		Role:       Role(role),
		Name:       name,
		Email:      currentEmail,
	}, now)
	if err != nil {
		return Session{}, err
	}

	if _, err := tx.Exec(ctx, `
		UPDATE auth_sessions
		SET revoked_at = $2,
		    replaced_by = $3::uuid
		WHERE id = $1::uuid
	`, tokenID, now, newTokenID); err != nil {
		return Session{}, fmt.Errorf("revoke previous session: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return Session{}, fmt.Errorf("commit switch tenant transaction: %w", err)
	}

	pair.TenantChoices, err = s.tenantOptionsByEmail(ctx, tx, currentEmail)
	if err != nil {
		return Session{}, err
	}
	return pair, nil
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

func (s *PostgresService) tenantOptionsByEmail(ctx context.Context, q pgx.Tx, email string) ([]TenantOption, error) {
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
