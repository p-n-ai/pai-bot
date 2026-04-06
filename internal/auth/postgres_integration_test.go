//go:build integration
// +build integration

package auth

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/p-n-ai/pai-bot/internal/platform/mailer"
	tcpostgres "github.com/testcontainers/testcontainers-go/modules/postgres"
)

func TestPostgresService_AcceptInviteLoginSessionAndLogout(t *testing.T) {
	ctx := context.Background()
	pool := startAuthPostgres(t, ctx)
	now := time.Date(2026, 3, 16, 10, 0, 0, 0, time.UTC)
	nowFn := func() time.Time { return now }
	svc := newPostgresService(pool, 7*24*time.Hour, nowFn)

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
	assertSessionStored(t, ctx, pool, pair.Token, false)
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

	now = now.Add(30 * time.Minute)
	session, err := svc.Session(ctx, loginPair.Token)
	if err != nil {
		t.Fatalf("Session() error = %v", err)
	}
	if session.Token != loginPair.Token {
		t.Fatalf("session token = %q, want %q", session.Token, loginPair.Token)
	}
	if session.User.UserID != loginPair.User.UserID {
		t.Fatalf("session user_id = %q, want %q", session.User.UserID, loginPair.User.UserID)
	}
	assertSessionStored(t, ctx, pool, loginPair.Token, false)

	if err := svc.Logout(ctx, loginPair.Token); err != nil {
		t.Fatalf("Logout() error = %v", err)
	}
	assertSessionStored(t, ctx, pool, loginPair.Token, true)
}

func TestPostgresService_LoginRejectsInvalidPassword(t *testing.T) {
	ctx := context.Background()
	pool := startAuthPostgres(t, ctx)
	now := time.Date(2026, 3, 16, 10, 0, 0, 0, time.UTC)
	svc := newPostgresService(pool, 7*24*time.Hour, func() time.Time { return now })

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
	svc := newPostgresService(pool, 7*24*time.Hour, func() time.Time { return now })

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

func TestPostgresService_IssueInviteMarksDeliverySentWhenMailerSucceeds(t *testing.T) {
	ctx := context.Background()
	pool := startAuthPostgres(t, ctx)
	now := time.Date(2026, 4, 6, 10, 0, 0, 0, time.UTC)
	svc := newPostgresService(pool, 7*24*time.Hour, func() time.Time { return now })

	recorder := &recordingInviteMailer{}
	svc.ConfigureInviteEmail(recorder)

	tenantID := loadDefaultTenantID(t, ctx, pool)
	adminUserID := seedWebUser(t, ctx, pool, tenantID, RoleAdmin, "Admin Inviter")

	invite, err := svc.IssueInvite(ctx, IssueInviteRequest{
		InvitedByUserID:   adminUserID,
		TenantID:          tenantID,
		Email:             "newteacher@example.com",
		Role:              RoleTeacher,
		ActivationBaseURL: "https://admin.example.com",
	})
	if err != nil {
		t.Fatalf("IssueInvite() error = %v", err)
	}
	if invite.DeliveryStatus != "sent" {
		t.Fatalf("delivery_status = %q, want sent", invite.DeliveryStatus)
	}
	if invite.ActivationURL != "https://admin.example.com/activate?token="+invite.Token {
		t.Fatalf("activation_url = %q, want generated activation url", invite.ActivationURL)
	}
	if len(recorder.messages) != 1 {
		t.Fatalf("sent messages = %d, want 1", len(recorder.messages))
	}
	assertInviteDeliveryState(t, ctx, pool, "newteacher@example.com", "sent")
}

func TestPostgresService_IssueInviteMarksDeliveryFailedWhenMailerFails(t *testing.T) {
	ctx := context.Background()
	pool := startAuthPostgres(t, ctx)
	now := time.Date(2026, 4, 6, 10, 0, 0, 0, time.UTC)
	svc := newPostgresService(pool, 7*24*time.Hour, func() time.Time { return now })

	svc.ConfigureInviteEmail(failingInviteMailer{err: fmt.Errorf("smtp offline")})

	tenantID := loadDefaultTenantID(t, ctx, pool)
	adminUserID := seedWebUser(t, ctx, pool, tenantID, RoleAdmin, "Admin Inviter")

	invite, err := svc.IssueInvite(ctx, IssueInviteRequest{
		InvitedByUserID:   adminUserID,
		TenantID:          tenantID,
		Email:             "newteacher@example.com",
		Role:              RoleTeacher,
		ActivationBaseURL: "https://admin.example.com",
	})
	if err != nil {
		t.Fatalf("IssueInvite() error = %v", err)
	}
	if invite.DeliveryStatus != "failed" {
		t.Fatalf("delivery_status = %q, want failed", invite.DeliveryStatus)
	}
	if invite.DeliveryError != "smtp offline" {
		t.Fatalf("delivery_error = %q, want smtp offline", invite.DeliveryError)
	}
	assertInviteDeliveryState(t, ctx, pool, "newteacher@example.com", "failed")
}

func TestPostgresService_ReissueInviteRotatesToken(t *testing.T) {
	ctx := context.Background()
	pool := startAuthPostgres(t, ctx)
	now := time.Date(2026, 3, 16, 10, 0, 0, 0, time.UTC)
	svc := newPostgresService(pool, 7*24*time.Hour, func() time.Time { return now })

	tenantID := loadDefaultTenantID(t, ctx, pool)
	adminUserID := seedWebUser(t, ctx, pool, tenantID, RoleAdmin, "Admin Inviter")

	firstInvite, err := svc.IssueInvite(ctx, IssueInviteRequest{
		InvitedByUserID: adminUserID,
		TenantID:        tenantID,
		Email:           "newteacher@example.com",
		Role:            RoleTeacher,
	})
	if err != nil {
		t.Fatalf("IssueInvite() error = %v", err)
	}

	var inviteID string
	if err := pool.QueryRow(ctx, `
		SELECT id::text
		FROM auth_invites
		WHERE email_normalized = $1
	`, NormalizeIdentifier("newteacher@example.com")).Scan(&inviteID); err != nil {
		t.Fatalf("query invite id: %v", err)
	}

	reissuedInvite, err := svc.ReissueInvite(ctx, ReissueInviteRequest{
		InviteID:        inviteID,
		InvitedByUserID: adminUserID,
		TenantID:        tenantID,
	})
	if err != nil {
		t.Fatalf("ReissueInvite() error = %v", err)
	}
	if reissuedInvite.Token == "" {
		t.Fatal("reissued token should not be empty")
	}
	if reissuedInvite.Token == firstInvite.Token {
		t.Fatal("reissued token should rotate")
	}
	assertInviteTokenStored(t, ctx, pool, "newteacher@example.com", reissuedInvite.Token)

	if _, err := svc.AcceptInvite(ctx, AcceptInviteRequest{
		Token:    firstInvite.Token,
		Name:     "Teacher New",
		Password: "secret-123",
	}); err != ErrInvalidInvite {
		t.Fatalf("AcceptInvite(old token) error = %v, want ErrInvalidInvite", err)
	}
}

func TestPostgresService_LoginReturnsTenantChoicesWhenEmailExistsAcrossTenants(t *testing.T) {
	ctx := context.Background()
	pool := startAuthPostgres(t, ctx)
	now := time.Date(2026, 3, 16, 10, 0, 0, 0, time.UTC)
	svc := newPostgresService(pool, 7*24*time.Hour, func() time.Time { return now })

	defaultTenantID := loadDefaultTenantID(t, ctx, pool)
	secondTenantID := seedTenant(t, ctx, pool, "school-b", "School B")

	seedPasswordUser(t, ctx, pool, defaultTenantID, "shared@example.com", RoleTeacher, "secret-123")
	seedPasswordUser(t, ctx, pool, secondTenantID, "shared@example.com", RoleTeacher, "secret-123")

	pair, err := svc.Login(ctx, LoginRequest{
		Email:    "shared@example.com",
		Password: "secret-123",
	})
	if err != nil {
		t.Fatalf("Login() error = %v", err)
	}
	if pair.User.TenantID != defaultTenantID {
		t.Fatalf("tenant_id = %q, want %q", pair.User.TenantID, defaultTenantID)
	}
	if len(pair.TenantChoices) != 2 {
		t.Fatalf("tenant_choices = %#v, want 2 options", pair.TenantChoices)
	}

	pair, err = svc.Login(ctx, LoginRequest{
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

func TestPostgresService_SwitchTenantReissuesSessionWithoutLogout(t *testing.T) {
	ctx := context.Background()
	pool := startAuthPostgres(t, ctx)
	now := time.Date(2026, 3, 16, 10, 0, 0, 0, time.UTC)
	svc := newPostgresService(pool, 7*24*time.Hour, func() time.Time { return now })

	defaultTenantID := loadDefaultTenantID(t, ctx, pool)
	secondTenantID := seedTenant(t, ctx, pool, "school-b", "School B")

	seedPasswordUser(t, ctx, pool, defaultTenantID, "shared@example.com", RoleTeacher, "secret-123")
	secondUserID := seedPasswordUser(t, ctx, pool, secondTenantID, "shared@example.com", RoleTeacher, "secret-123")

	loginPair, err := svc.Login(ctx, LoginRequest{
		TenantID: defaultTenantID,
		Email:    "shared@example.com",
		Password: "secret-123",
	})
	if err != nil {
		t.Fatalf("Login() error = %v", err)
	}

	switchedPair, err := svc.SwitchTenant(ctx, loginPair.Token, secondTenantID, "secret-123")
	if err != nil {
		t.Fatalf("SwitchTenant() error = %v", err)
	}
	if switchedPair.User.UserID != secondUserID {
		t.Fatalf("user_id = %q, want %q", switchedPair.User.UserID, secondUserID)
	}
	if switchedPair.User.TenantID != secondTenantID {
		t.Fatalf("tenant_id = %q, want %q", switchedPair.User.TenantID, secondTenantID)
	}
	if switchedPair.Token == loginPair.Token {
		t.Fatal("SwitchTenant() should rotate the session token")
	}

	assertSessionStored(t, ctx, pool, loginPair.Token, true)
	assertSessionStored(t, ctx, pool, switchedPair.Token, false)
}

func TestPostgresService_SwitchTenantRequiresTargetTenantPassword(t *testing.T) {
	ctx := context.Background()
	pool := startAuthPostgres(t, ctx)
	now := time.Date(2026, 3, 16, 10, 0, 0, 0, time.UTC)
	svc := newPostgresService(pool, 7*24*time.Hour, func() time.Time { return now })

	defaultTenantID := loadDefaultTenantID(t, ctx, pool)
	secondTenantID := seedTenant(t, ctx, pool, "school-b", "School B")

	seedPasswordUser(t, ctx, pool, defaultTenantID, "shared@example.com", RoleTeacher, "secret-123")
	seedPasswordUser(t, ctx, pool, secondTenantID, "shared@example.com", RoleTeacher, "different-secret")

	loginPair, err := svc.Login(ctx, LoginRequest{
		TenantID: defaultTenantID,
		Email:    "shared@example.com",
		Password: "secret-123",
	})
	if err != nil {
		t.Fatalf("Login() error = %v", err)
	}

	_, err = svc.SwitchTenant(ctx, loginPair.Token, secondTenantID, "secret-123")
	if err != ErrInvalidCredentials {
		t.Fatalf("SwitchTenant() error = %v, want ErrInvalidCredentials", err)
	}

	assertSessionStored(t, ctx, pool, loginPair.Token, false)
}

func TestAuthIdentityTenantConstraintRejectsMismatchedTenant(t *testing.T) {
	ctx := context.Background()
	pool := startAuthPostgres(t, ctx)

	defaultTenantID := loadDefaultTenantID(t, ctx, pool)
	secondTenantID := seedTenant(t, ctx, pool, "school-b", "School B")
	userID := seedWebUser(t, ctx, pool, defaultTenantID, RoleTeacher, "Teacher One")

	_, err := pool.Exec(ctx,
		`INSERT INTO auth_identities (user_id, tenant_id, provider, identifier, identifier_normalized)
		 VALUES ($1::uuid, $2::uuid, 'password', $3, $4)`,
		userID,
		secondTenantID,
		"teacher@example.com",
		NormalizeIdentifier("teacher@example.com"),
	)
	if err == nil {
		t.Fatal("expected auth identity insert with mismatched tenant to fail")
	}
	if !strings.Contains(err.Error(), "auth_identities_user_id_tenant_id_fkey") {
		t.Fatalf("constraint error = %v, want auth_identities_user_id_tenant_id_fkey", err)
	}
}

func TestPostgresService_PlatformAdminLoginWithoutTenant(t *testing.T) {
	ctx := context.Background()
	pool := startAuthPostgres(t, ctx)
	now := time.Date(2026, 3, 18, 10, 0, 0, 0, time.UTC)
	svc := newPostgresService(pool, 7*24*time.Hour, func() time.Time { return now })

	userID := seedGlobalPasswordUser(t, ctx, pool, "platform-admin@example.com", RolePlatformAdmin, "secret-123")

	pair, err := svc.Login(ctx, LoginRequest{
		Email:    "platform-admin@example.com",
		Password: "secret-123",
	})
	if err != nil {
		t.Fatalf("Login() error = %v", err)
	}
	if pair.User.UserID != userID {
		t.Fatalf("user_id = %q, want %q", pair.User.UserID, userID)
	}
	if pair.User.Role != RolePlatformAdmin {
		t.Fatalf("role = %q, want %q", pair.User.Role, RolePlatformAdmin)
	}
	if pair.User.TenantID != "" {
		t.Fatalf("tenant_id = %q, want empty", pair.User.TenantID)
	}

	session, err := svc.Session(ctx, pair.Token)
	if err != nil {
		t.Fatalf("Session() error = %v", err)
	}
	if session.User.TenantID != "" {
		t.Fatalf("session tenant_id = %q, want empty", session.User.TenantID)
	}
	assertSessionStored(t, ctx, pool, pair.Token, false)
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
	applyAuthMigrationFile(t, ctx, pool, filepath.Join("..", "..", "migrations", "20260318100000_initial.sql"))
	applyAuthMigrationFile(t, ctx, pool, filepath.Join("..", "..", "migrations", "20260318100300_auth_tables.sql"))
	applyAuthMigrationFile(t, ctx, pool, filepath.Join("..", "..", "migrations", "20260318100400_auth_identity_tenant_consistency.sql"))
	applyAuthMigrationFile(t, ctx, pool, filepath.Join("..", "..", "migrations", "20260318100500_global_platform_admins.sql"))
	applyAuthMigrationFile(t, ctx, pool, filepath.Join("..", "..", "migrations", "20260403110000_google_auth_oidc.sql"))
	applyAuthMigrationFile(t, ctx, pool, filepath.Join("..", "..", "migrations", "20260406110000_invite_delivery.sql"))

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
	upSQL, err := authGooseUpSQL(string(sqlBytes))
	if err != nil {
		t.Fatalf("parse migration %s: %v", path, err)
	}
	if _, err := pool.Exec(ctx, upSQL); err != nil {
		t.Fatalf("apply migration %s: %v", path, err)
	}
}

func authGooseUpSQL(content string) (string, error) {
	upMarker := "-- +goose Up"
	downMarker := "-- +goose Down"

	upIdx := strings.Index(content, upMarker)
	if upIdx == -1 {
		return strings.TrimSpace(content), nil
	}

	upBody := content[upIdx+len(upMarker):]
	if downIdx := strings.Index(upBody, downMarker); downIdx >= 0 {
		upBody = upBody[:downIdx]
	}

	upBody = strings.TrimSpace(upBody)
	if upBody == "" {
		return "", fmt.Errorf("missing goose Up section")
	}

	return upBody, nil
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

func seedGlobalPasswordUser(t *testing.T, ctx context.Context, pool *pgxpool.Pool, email string, role Role, password string) string {
	t.Helper()

	passwordHash, err := HashPassword(password)
	if err != nil {
		t.Fatalf("HashPassword() error = %v", err)
	}

	var userID string
	err = pool.QueryRow(ctx,
		`INSERT INTO users (tenant_id, role, name, channel)
		 VALUES (NULL, $1, 'Platform Admin', 'web')
		 RETURNING id::text`,
		string(role),
	).Scan(&userID)
	if err != nil {
		t.Fatalf("insert global user: %v", err)
	}

	_, err = pool.Exec(ctx,
		`INSERT INTO auth_identities (user_id, tenant_id, provider, identifier, identifier_normalized, password_hash, email_verified_at)
		 VALUES ($1::uuid, NULL, 'password', $2, $3, $4, NOW())`,
		userID,
		email,
		NormalizeIdentifier(email),
		passwordHash,
	)
	if err != nil {
		t.Fatalf("insert global identity: %v", err)
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

func assertInviteDeliveryState(t *testing.T, ctx context.Context, pool *pgxpool.Pool, email, wantStatus string) {
	t.Helper()

	var (
		status    string
		sentAt    *time.Time
		lastError *string
	)
	if err := pool.QueryRow(ctx,
		`SELECT delivery_status, delivery_sent_at, delivery_error FROM auth_invites WHERE email_normalized = $1`,
		NormalizeIdentifier(email),
	).Scan(&status, &sentAt, &lastError); err != nil {
		t.Fatalf("query invite delivery status: %v", err)
	}
	if status != wantStatus {
		t.Fatalf("delivery_status = %q, want %q", status, wantStatus)
	}
	if wantStatus == "sent" && sentAt == nil {
		t.Fatal("delivery_sent_at should be set when invite email is sent")
	}
	if wantStatus == "failed" && (lastError == nil || strings.TrimSpace(*lastError) == "") {
		t.Fatal("delivery_error should be set when invite email fails")
	}
}

func assertSessionStored(t *testing.T, ctx context.Context, pool *pgxpool.Pool, rawToken string, wantRevoked bool) {
	t.Helper()

	var revokedAt *time.Time
	if err := pool.QueryRow(ctx,
		`SELECT revoked_at FROM auth_sessions WHERE token_hash = $1`,
		HashOpaqueToken(rawToken),
	).Scan(&revokedAt); err != nil {
		t.Fatalf("query session: %v", err)
	}
	if wantRevoked && revokedAt == nil {
		t.Fatal("session should be revoked")
	}
	if !wantRevoked && revokedAt != nil {
		t.Fatal("session should be active")
	}
}

type recordingInviteMailer struct {
	messages []mailer.InviteMessage
}

func (m *recordingInviteMailer) SendInvite(_ context.Context, msg mailer.InviteMessage) error {
	m.messages = append(m.messages, msg)
	return nil
}

type failingInviteMailer struct {
	err error
}

func (m failingInviteMailer) SendInvite(_ context.Context, _ mailer.InviteMessage) error {
	return m.err
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
