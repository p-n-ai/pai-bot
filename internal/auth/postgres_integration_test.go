//go:build integration
// +build integration

package auth

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	tcpostgres "github.com/testcontainers/testcontainers-go/modules/postgres"
)

func TestPostgresService_AcceptInviteLoginRefreshAndLogout(t *testing.T) {
	ctx := context.Background()
	pool := startAuthPostgres(t, ctx)
	now := time.Date(2026, 3, 16, 10, 0, 0, 0, time.UTC)
	svc := newPostgresService(pool, "test-secret", 15*time.Minute, 7*24*time.Hour, func() time.Time { return now })

	tenantID := loadDefaultTenantID(t, ctx, pool)
	seedInvite(t, ctx, pool, tenantID, "teacher@example.com", RoleTeacher, "invite-token", now.Add(24*time.Hour))

	pair, err := svc.AcceptInvite(ctx, AcceptInviteRequest{
		Token:    "invite-token",
		Name:     "Teacher One",
		Password: "secret-123",
	})
	if err != nil {
		t.Fatalf("AcceptInvite() error = %v", err)
	}
	if pair.User.Email != "teacher@example.com" {
		t.Fatalf("accept email = %q, want teacher@example.com", pair.User.Email)
	}
	if pair.User.Role != RoleTeacher {
		t.Fatalf("accept role = %q, want %q", pair.User.Role, RoleTeacher)
	}
	assertRefreshTokenStored(t, ctx, pool, pair.RefreshToken, false)
	assertInviteAccepted(t, ctx, pool, "teacher@example.com")

	loginPair, err := svc.Login(ctx, LoginRequest{
		TenantID: tenantID,
		Email:    "teacher@example.com",
		Password: "secret-123",
	})
	if err != nil {
		t.Fatalf("Login() error = %v", err)
	}
	if loginPair.User.UserID != pair.User.UserID {
		t.Fatalf("login user_id = %q, want %q", loginPair.User.UserID, pair.User.UserID)
	}

	refreshPair, err := svc.Refresh(ctx, loginPair.RefreshToken)
	if err != nil {
		t.Fatalf("Refresh() error = %v", err)
	}
	if refreshPair.RefreshToken == loginPair.RefreshToken {
		t.Fatal("Refresh() should rotate the refresh token")
	}
	assertRefreshTokenStored(t, ctx, pool, loginPair.RefreshToken, true)
	assertRefreshTokenStored(t, ctx, pool, refreshPair.RefreshToken, false)

	if err := svc.Logout(ctx, refreshPair.RefreshToken); err != nil {
		t.Fatalf("Logout() error = %v", err)
	}
	assertRefreshTokenStored(t, ctx, pool, refreshPair.RefreshToken, true)
}

func TestPostgresService_LoginRejectsInvalidPassword(t *testing.T) {
	ctx := context.Background()
	pool := startAuthPostgres(t, ctx)
	now := time.Date(2026, 3, 16, 10, 0, 0, 0, time.UTC)
	svc := newPostgresService(pool, "test-secret", 15*time.Minute, 7*24*time.Hour, func() time.Time { return now })

	tenantID := loadDefaultTenantID(t, ctx, pool)
	userID := seedPasswordUser(t, ctx, pool, tenantID, "parent@example.com", RoleParent, "secret-123")

	_, err := svc.Login(ctx, LoginRequest{
		TenantID: tenantID,
		Email:    "parent@example.com",
		Password: "bad-pass",
	})
	if err != ErrInvalidCredentials {
		t.Fatalf("Login() error = %v, want ErrInvalidCredentials", err)
	}

	assertUserStillExists(t, ctx, pool, userID)
}

func TestPostgresService_IssueInvitePersistsHashedToken(t *testing.T) {
	ctx := context.Background()
	pool := startAuthPostgres(t, ctx)
	now := time.Date(2026, 3, 16, 10, 0, 0, 0, time.UTC)
	svc := newPostgresService(pool, "test-secret", 15*time.Minute, 7*24*time.Hour, func() time.Time { return now })

	tenantID := loadDefaultTenantID(t, ctx, pool)
	adminUserID := seedWebUser(t, ctx, pool, tenantID, RoleAdmin, "Admin Inviter")

	invite, err := svc.IssueInvite(ctx, IssueInviteRequest{
		InvitedByUserID: adminUserID,
		TenantID:        tenantID,
		Email:           "newteacher@example.com",
		Role:            RoleTeacher,
	})
	if err != nil {
		t.Fatalf("IssueInvite() error = %v", err)
	}
	if invite.Token == "" {
		t.Fatal("invite token should not be empty")
	}
	assertInviteTokenStored(t, ctx, pool, "newteacher@example.com", invite.Token)
}

func TestPostgresService_LoginRequiresTenantWhenEmailExistsAcrossTenants(t *testing.T) {
	ctx := context.Background()
	pool := startAuthPostgres(t, ctx)
	now := time.Date(2026, 3, 16, 10, 0, 0, 0, time.UTC)
	svc := newPostgresService(pool, "test-secret", 15*time.Minute, 7*24*time.Hour, func() time.Time { return now })

	defaultTenantID := loadDefaultTenantID(t, ctx, pool)
	secondTenantID := seedTenant(t, ctx, pool, "school-b", "School B")

	seedPasswordUser(t, ctx, pool, defaultTenantID, "shared@example.com", RoleTeacher, "secret-123")
	seedPasswordUser(t, ctx, pool, secondTenantID, "shared@example.com", RoleTeacher, "secret-123")

	_, err := svc.Login(ctx, LoginRequest{
		Email:    "shared@example.com",
		Password: "secret-123",
	})
	if !errors.Is(err, ErrTenantRequired) {
		t.Fatalf("Login() error = %v, want ErrTenantRequired", err)
	}

	options, ok := TenantRequiredOptions(err)
	if !ok || len(options) != 2 {
		t.Fatalf("TenantRequiredOptions() = %#v, %v", options, ok)
	}

	pair, err := svc.Login(ctx, LoginRequest{
		TenantID: secondTenantID,
		Email:    "shared@example.com",
		Password: "secret-123",
	})
	if err != nil {
		t.Fatalf("Login(with tenant) error = %v", err)
	}
	if pair.User.TenantID != secondTenantID {
		t.Fatalf("tenant_id = %q, want %q", pair.User.TenantID, secondTenantID)
	}
}

func startAuthPostgres(t *testing.T, ctx context.Context) *pgxpool.Pool {
	t.Helper()

	container, err := tcpostgres.Run(
		ctx,
		"postgres:17-alpine",
		tcpostgres.WithDatabase("pai"),
		tcpostgres.WithUsername("pai"),
		tcpostgres.WithPassword("pai"),
	)
	if err != nil {
		t.Fatalf("start postgres container: %v", err)
	}
	t.Cleanup(func() {
		if err := container.Terminate(context.Background()); err != nil {
			t.Fatalf("terminate postgres container: %v", err)
		}
	})

	connString, err := container.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		t.Fatalf("container connection string: %v", err)
	}

	pool, err := pgxpool.New(ctx, connString)
	if err != nil {
		t.Fatalf("pgxpool.New() error = %v", err)
	}
	t.Cleanup(pool.Close)

	waitForAuthPostgresReady(t, ctx, pool)
	applyAuthMigrationFile(t, ctx, pool, filepath.Join("..", "..", "migrations", "001_initial.up.sql"))
	applyAuthMigrationFile(t, ctx, pool, filepath.Join("..", "..", "migrations", "009_auth_tables.up.sql"))

	return pool
}

func waitForAuthPostgresReady(t *testing.T, ctx context.Context, pool *pgxpool.Pool) {
	t.Helper()

	deadline := time.Now().Add(15 * time.Second)
	for {
		if err := pool.Ping(ctx); err == nil {
			return
		}
		if time.Now().After(deadline) {
			t.Fatal("postgres did not become ready before timeout")
		}
		time.Sleep(200 * time.Millisecond)
	}
}

func applyAuthMigrationFile(t *testing.T, ctx context.Context, pool *pgxpool.Pool, path string) {
	t.Helper()

	sqlBytes, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read migration %s: %v", path, err)
	}
	if _, err := pool.Exec(ctx, string(sqlBytes)); err != nil {
		t.Fatalf("apply migration %s: %v", path, err)
	}
}

func loadDefaultTenantID(t *testing.T, ctx context.Context, pool *pgxpool.Pool) string {
	t.Helper()

	var tenantID string
	if err := pool.QueryRow(ctx, `SELECT id::text FROM tenants WHERE slug = 'default'`).Scan(&tenantID); err != nil {
		t.Fatalf("load default tenant: %v", err)
	}
	return tenantID
}

func seedInvite(t *testing.T, ctx context.Context, pool *pgxpool.Pool, tenantID, email string, role Role, token string, expiresAt time.Time) {
	t.Helper()

	_, err := pool.Exec(ctx,
		`INSERT INTO auth_invites (tenant_id, email, email_normalized, role, token_hash, expires_at)
		 VALUES ($1::uuid, $2, $3, $4, $5, $6)`,
		tenantID,
		email,
		NormalizeIdentifier(email),
		string(role),
		HashOpaqueToken(token),
		expiresAt,
	)
	if err != nil {
		t.Fatalf("seed invite: %v", err)
	}
}

func seedPasswordUser(t *testing.T, ctx context.Context, pool *pgxpool.Pool, tenantID, email string, role Role, password string) string {
	t.Helper()

	passwordHash, err := HashPassword(password)
	if err != nil {
		t.Fatalf("HashPassword() error = %v", err)
	}

	var userID string
	err = pool.QueryRow(ctx,
		`INSERT INTO users (tenant_id, role, name, channel)
		 VALUES ($1::uuid, $2, 'Existing User', 'web')
		 RETURNING id::text`,
		tenantID,
		string(role),
	).Scan(&userID)
	if err != nil {
		t.Fatalf("insert user: %v", err)
	}

	_, err = pool.Exec(ctx,
		`INSERT INTO auth_identities (user_id, tenant_id, provider, identifier, identifier_normalized, password_hash, email_verified_at)
		 VALUES ($1::uuid, $2::uuid, 'password', $3, $4, $5, NOW())`,
		userID,
		tenantID,
		email,
		NormalizeIdentifier(email),
		passwordHash,
	)
	if err != nil {
		t.Fatalf("insert identity: %v", err)
	}

	return userID
}

func seedWebUser(t *testing.T, ctx context.Context, pool *pgxpool.Pool, tenantID string, role Role, name string) string {
	t.Helper()

	var userID string
	err := pool.QueryRow(ctx,
		`INSERT INTO users (tenant_id, role, name, channel)
		 VALUES ($1::uuid, $2, $3, 'web')
		 RETURNING id::text`,
		tenantID,
		string(role),
		name,
	).Scan(&userID)
	if err != nil {
		t.Fatalf("insert web user: %v", err)
	}

	return userID
}

func seedTenant(t *testing.T, ctx context.Context, pool *pgxpool.Pool, slug, name string) string {
	t.Helper()

	var tenantID string
	err := pool.QueryRow(ctx,
		`INSERT INTO tenants (name, slug)
		 VALUES ($1, $2)
		 RETURNING id::text`,
		name,
		slug,
	).Scan(&tenantID)
	if err != nil {
		t.Fatalf("insert tenant: %v", err)
	}

	return tenantID
}

func assertInviteAccepted(t *testing.T, ctx context.Context, pool *pgxpool.Pool, email string) {
	t.Helper()

	var acceptedAt *time.Time
	if err := pool.QueryRow(ctx,
		`SELECT accepted_at FROM auth_invites WHERE email_normalized = $1`,
		NormalizeIdentifier(email),
	).Scan(&acceptedAt); err != nil {
		t.Fatalf("query accepted_at: %v", err)
	}
	if acceptedAt == nil {
		t.Fatal("accepted_at should be set")
	}
}

func assertInviteTokenStored(t *testing.T, ctx context.Context, pool *pgxpool.Pool, email, rawToken string) {
	t.Helper()

	var tokenHash string
	if err := pool.QueryRow(ctx,
		`SELECT token_hash FROM auth_invites WHERE email_normalized = $1`,
		NormalizeIdentifier(email),
	).Scan(&tokenHash); err != nil {
		t.Fatalf("query invite token hash: %v", err)
	}
	if tokenHash != HashOpaqueToken(rawToken) {
		t.Fatalf("token_hash = %q, want hash of raw token", tokenHash)
	}
}

func assertRefreshTokenStored(t *testing.T, ctx context.Context, pool *pgxpool.Pool, rawToken string, wantRevoked bool) {
	t.Helper()

	var revokedAt *time.Time
	if err := pool.QueryRow(ctx,
		`SELECT revoked_at FROM auth_refresh_tokens WHERE token_hash = $1`,
		HashOpaqueToken(rawToken),
	).Scan(&revokedAt); err != nil {
		t.Fatalf("query refresh token: %v", err)
	}
	if wantRevoked && revokedAt == nil {
		t.Fatal("refresh token should be revoked")
	}
	if !wantRevoked && revokedAt != nil {
		t.Fatal("refresh token should be active")
	}
}

func assertUserStillExists(t *testing.T, ctx context.Context, pool *pgxpool.Pool, userID string) {
	t.Helper()

	var exists bool
	if err := pool.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM users WHERE id = $1::uuid)`, userID).Scan(&exists); err != nil {
		t.Fatalf("query user exists: %v", err)
	}
	if !exists {
		t.Fatalf("expected user %s to still exist", userID)
	}
}
