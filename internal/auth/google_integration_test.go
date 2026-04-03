//go:build integration
// +build integration

package auth

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

func TestPostgresService_GoogleLoginAutoLinksSingleVerifiedEmail(t *testing.T) {
	ctx := context.Background()
	pool := startAuthPostgres(t, ctx)
	now := time.Date(2026, 4, 3, 10, 0, 0, 0, time.UTC)
	svc := newPostgresService(pool, "test-secret", 15*time.Minute, 7*24*time.Hour, func() time.Time { return now })

	tenantID := loadDefaultTenantID(t, ctx, pool)
	userID := seedPasswordUser(t, ctx, pool, tenantID, "teacher@gmail.com", RoleTeacher, "secret-123")
	google := newGoogleOIDCTestServer(t)
	defer google.Close()

	svc.ConfigureGoogleOAuth(GoogleOAuthProviderConfig{
		ClientID:              "google-client",
		ClientSecret:          "google-secret",
		RedirectURL:           "http://localhost:8080/api/auth/google/callback",
		DiscoveryURL:          google.URL + "/.well-known/openid-configuration",
		AdminBaseURL:          "http://localhost:3000",
		EmulatorSigningSecret: google.signingSecret,
	})

	authURL, err := svc.StartGoogleLogin(ctx, StartGoogleFlowRequest{NextPath: "/dashboard"})
	if err != nil {
		t.Fatalf("StartGoogleLogin() error = %v", err)
	}

	state := mustQueryParam(t, authURL, "state")
	nonce := loadOIDCFlowNonce(t, ctx, pool, state)
	google.identity = googleIdentity{
		Sub:           "google-sub-1",
		Email:         "teacher@gmail.com",
		EmailVerified: true,
		Name:          "Teacher Example",
	}
	google.nonce = nonce

	result, err := svc.CompleteGoogleCallback(ctx, GoogleCallbackRequest{
		State: state,
		Code:  "google-code-1",
	})
	if err != nil {
		t.Fatalf("CompleteGoogleCallback() error = %v", err)
	}
	if result.Pair == nil {
		t.Fatal("CompleteGoogleCallback() pair = nil, want session")
	}
	if result.Pair.User.UserID != userID {
		t.Fatalf("user_id = %q, want %q", result.Pair.User.UserID, userID)
	}
	if !result.Linked {
		t.Fatal("result.Linked = false, want true for auto-link")
	}

	identities, err := svc.ListLinkedIdentities(ctx, userID)
	if err != nil {
		t.Fatalf("ListLinkedIdentities() error = %v", err)
	}
	if len(identities) != 1 || identities[0].Provider != "google" {
		t.Fatalf("linked identities = %#v, want one Google identity", identities)
	}
}

func TestPostgresService_GoogleLinkAllowsDifferentEmail(t *testing.T) {
	ctx := context.Background()
	pool := startAuthPostgres(t, ctx)
	now := time.Date(2026, 4, 3, 10, 0, 0, 0, time.UTC)
	svc := newPostgresService(pool, "test-secret", 15*time.Minute, 7*24*time.Hour, func() time.Time { return now })

	tenantID := loadDefaultTenantID(t, ctx, pool)
	userID := seedPasswordUser(t, ctx, pool, tenantID, "teacher@yahoo.com", RoleTeacher, "secret-123")
	google := newGoogleOIDCTestServer(t)
	defer google.Close()

	svc.ConfigureGoogleOAuth(GoogleOAuthProviderConfig{
		ClientID:              "google-client",
		ClientSecret:          "google-secret",
		RedirectURL:           "http://localhost:8080/api/auth/google/callback",
		DiscoveryURL:          google.URL + "/.well-known/openid-configuration",
		AdminBaseURL:          "http://localhost:3000",
		EmulatorSigningSecret: google.signingSecret,
	})

	linkURL, err := svc.StartGoogleLink(ctx, StartGoogleFlowRequest{
		UserID:   userID,
		NextPath: "/dashboard",
	})
	if err != nil {
		t.Fatalf("StartGoogleLink() error = %v", err)
	}

	state := mustQueryParam(t, linkURL, "state")
	nonce := loadOIDCFlowNonce(t, ctx, pool, state)
	google.identity = googleIdentity{
		Sub:           "google-sub-2",
		Email:         "teacher@gmail.com",
		EmailVerified: true,
		Name:          "Teacher Gmail",
	}
	google.nonce = nonce

	result, err := svc.CompleteGoogleCallback(ctx, GoogleCallbackRequest{
		State: state,
		Code:  "google-code-2",
	})
	if err != nil {
		t.Fatalf("CompleteGoogleCallback(link) error = %v", err)
	}
	if result.Pair == nil || result.Pair.User.UserID != userID {
		t.Fatalf("link result pair = %#v, want linked user", result.Pair)
	}

	loginURL, err := svc.StartGoogleLogin(ctx, StartGoogleFlowRequest{NextPath: "/dashboard"})
	if err != nil {
		t.Fatalf("StartGoogleLogin() error = %v", err)
	}
	loginState := mustQueryParam(t, loginURL, "state")
	google.nonce = loadOIDCFlowNonce(t, ctx, pool, loginState)

	loginResult, err := svc.CompleteGoogleCallback(ctx, GoogleCallbackRequest{
		State: loginState,
		Code:  "google-code-3",
	})
	if err != nil {
		t.Fatalf("CompleteGoogleCallback(login) error = %v", err)
	}
	if loginResult.Pair == nil || loginResult.Pair.User.UserID != userID {
		t.Fatalf("login result pair = %#v, want linked user", loginResult.Pair)
	}
	if loginResult.Pair.User.Email != "teacher@yahoo.com" {
		t.Fatalf("session email = %q, want primary local email", loginResult.Pair.User.Email)
	}
}

func TestPostgresService_GoogleLinkReplacesExistingGoogleIdentity(t *testing.T) {
	ctx := context.Background()
	pool := startAuthPostgres(t, ctx)
	now := time.Date(2026, 4, 3, 10, 0, 0, 0, time.UTC)
	svc := newPostgresService(pool, "test-secret", 15*time.Minute, 7*24*time.Hour, func() time.Time { return now })

	tenantID := loadDefaultTenantID(t, ctx, pool)
	userID := seedPasswordUser(t, ctx, pool, tenantID, "teacher@yahoo.com", RoleTeacher, "secret-123")
	google := newGoogleOIDCTestServer(t)
	defer google.Close()

	svc.ConfigureGoogleOAuth(GoogleOAuthProviderConfig{
		ClientID:              "google-client",
		ClientSecret:          "google-secret",
		RedirectURL:           "http://localhost:8080/api/auth/google/callback",
		DiscoveryURL:          google.URL + "/.well-known/openid-configuration",
		AdminBaseURL:          "http://localhost:3000",
		EmulatorSigningSecret: google.signingSecret,
	})

	linkFirstURL, err := svc.StartGoogleLink(ctx, StartGoogleFlowRequest{
		UserID:   userID,
		NextPath: "/dashboard",
	})
	if err != nil {
		t.Fatalf("StartGoogleLink(first) error = %v", err)
	}
	firstState := mustQueryParam(t, linkFirstURL, "state")
	google.nonce = loadOIDCFlowNonce(t, ctx, pool, firstState)
	google.identity = googleIdentity{
		Sub:           "google-sub-old",
		Email:         "teacher.old@gmail.com",
		EmailVerified: true,
		Name:          "Teacher Old",
	}
	if _, err := svc.CompleteGoogleCallback(ctx, GoogleCallbackRequest{
		State: firstState,
		Code:  "google-code-old",
	}); err != nil {
		t.Fatalf("CompleteGoogleCallback(first link) error = %v", err)
	}

	linkSecondURL, err := svc.StartGoogleLink(ctx, StartGoogleFlowRequest{
		UserID:   userID,
		NextPath: "/dashboard",
	})
	if err != nil {
		t.Fatalf("StartGoogleLink(second) error = %v", err)
	}
	secondState := mustQueryParam(t, linkSecondURL, "state")
	google.nonce = loadOIDCFlowNonce(t, ctx, pool, secondState)
	google.identity = googleIdentity{
		Sub:           "google-sub-new",
		Email:         "teacher.new@gmail.com",
		EmailVerified: true,
		Name:          "Teacher New",
	}
	if _, err := svc.CompleteGoogleCallback(ctx, GoogleCallbackRequest{
		State: secondState,
		Code:  "google-code-new",
	}); err != nil {
		t.Fatalf("CompleteGoogleCallback(second link) error = %v", err)
	}

	identities, err := svc.ListLinkedIdentities(ctx, userID)
	if err != nil {
		t.Fatalf("ListLinkedIdentities() error = %v", err)
	}
	if len(identities) != 1 {
		t.Fatalf("linked identities = %#v, want exactly one Google identity", identities)
	}
	if identities[0].Email != "teacher.new@gmail.com" {
		t.Fatalf("linked email = %q, want teacher.new@gmail.com", identities[0].Email)
	}

	loginURL, err := svc.StartGoogleLogin(ctx, StartGoogleFlowRequest{NextPath: "/dashboard"})
	if err != nil {
		t.Fatalf("StartGoogleLogin() error = %v", err)
	}
	loginState := mustQueryParam(t, loginURL, "state")
	google.nonce = loadOIDCFlowNonce(t, ctx, pool, loginState)
	google.identity = googleIdentity{
		Sub:           "google-sub-new",
		Email:         "teacher.new@gmail.com",
		EmailVerified: true,
		Name:          "Teacher New",
	}
	loginResult, err := svc.CompleteGoogleCallback(ctx, GoogleCallbackRequest{
		State: loginState,
		Code:  "google-code-login",
	})
	if err != nil {
		t.Fatalf("CompleteGoogleCallback(login) error = %v", err)
	}
	if loginResult.Pair == nil || loginResult.Pair.User.UserID != userID {
		t.Fatalf("login result pair = %#v, want relinked user", loginResult.Pair)
	}
}

func TestPostgresService_GoogleLoginRequiresExistingAccountWhenNoMatch(t *testing.T) {
	ctx := context.Background()
	pool := startAuthPostgres(t, ctx)
	now := time.Date(2026, 4, 3, 10, 0, 0, 0, time.UTC)
	svc := newPostgresService(pool, "test-secret", 15*time.Minute, 7*24*time.Hour, func() time.Time { return now })
	google := newGoogleOIDCTestServer(t)
	defer google.Close()

	svc.ConfigureGoogleOAuth(GoogleOAuthProviderConfig{
		ClientID:              "google-client",
		ClientSecret:          "google-secret",
		RedirectURL:           "http://localhost:8080/api/auth/google/callback",
		DiscoveryURL:          google.URL + "/.well-known/openid-configuration",
		AdminBaseURL:          "http://localhost:3000",
		EmulatorSigningSecret: google.signingSecret,
	})

	authURL, err := svc.StartGoogleLogin(ctx, StartGoogleFlowRequest{})
	if err != nil {
		t.Fatalf("StartGoogleLogin() error = %v", err)
	}
	state := mustQueryParam(t, authURL, "state")
	google.nonce = loadOIDCFlowNonce(t, ctx, pool, state)
	google.identity = googleIdentity{
		Sub:           "google-sub-3",
		Email:         "missing@example.com",
		EmailVerified: true,
		Name:          "Missing Account",
	}

	_, err = svc.CompleteGoogleCallback(ctx, GoogleCallbackRequest{
		State: state,
		Code:  "google-code-4",
	})
	if err != ErrIdentityLinkRequired {
		t.Fatalf("CompleteGoogleCallback() error = %v, want ErrIdentityLinkRequired", err)
	}
}

func TestPostgresService_GoogleLoginRequiresTenantWhenEmailMatchesMultipleSchools(t *testing.T) {
	ctx := context.Background()
	pool := startAuthPostgres(t, ctx)
	now := time.Date(2026, 4, 3, 10, 0, 0, 0, time.UTC)
	svc := newPostgresService(pool, "test-secret", 15*time.Minute, 7*24*time.Hour, func() time.Time { return now })
	google := newGoogleOIDCTestServer(t)
	defer google.Close()

	defaultTenantID := loadDefaultTenantID(t, ctx, pool)
	secondTenantID := seedTenant(t, ctx, pool, "school-b", "School B")
	seedPasswordUser(t, ctx, pool, defaultTenantID, "shared@gmail.com", RoleTeacher, "secret-123")
	seedPasswordUser(t, ctx, pool, secondTenantID, "shared@gmail.com", RoleTeacher, "secret-123")

	svc.ConfigureGoogleOAuth(GoogleOAuthProviderConfig{
		ClientID:              "google-client",
		ClientSecret:          "google-secret",
		RedirectURL:           "http://localhost:8080/api/auth/google/callback",
		DiscoveryURL:          google.URL + "/.well-known/openid-configuration",
		AdminBaseURL:          "http://localhost:3000",
		EmulatorSigningSecret: google.signingSecret,
	})

	authURL, err := svc.StartGoogleLogin(ctx, StartGoogleFlowRequest{})
	if err != nil {
		t.Fatalf("StartGoogleLogin() error = %v", err)
	}
	state := mustQueryParam(t, authURL, "state")
	google.nonce = loadOIDCFlowNonce(t, ctx, pool, state)
	google.identity = googleIdentity{
		Sub:           "google-sub-4",
		Email:         "shared@gmail.com",
		EmailVerified: true,
		Name:          "Shared Teacher",
	}

	_, err = svc.CompleteGoogleCallback(ctx, GoogleCallbackRequest{
		State: state,
		Code:  "google-code-5",
	})
	if !errors.Is(err, ErrTenantRequired) {
		t.Fatalf("CompleteGoogleCallback() error = %v, want ErrTenantRequired", err)
	}
}

type googleOIDCTestServer struct {
	*httptest.Server
	identity      googleIdentity
	nonce         string
	signingSecret string
}

func newGoogleOIDCTestServer(t *testing.T) *googleOIDCTestServer {
	t.Helper()

	server := &googleOIDCTestServer{
		signingSecret: "emulate-google-jwt-secret",
	}

	httpServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/.well-known/openid-configuration":
			writeTestJSON(t, w, map[string]any{
				"issuer":                                server.URL,
				"authorization_endpoint":                server.URL + "/o/oauth2/v2/auth",
				"token_endpoint":                        server.URL + "/oauth2/token",
				"userinfo_endpoint":                     server.URL + "/oauth2/v2/userinfo",
				"jwks_uri":                              "",
				"id_token_signing_alg_values_supported": []string{"HS256"},
			})
		case "/oauth2/token":
			writeTestJSON(t, w, map[string]any{
				"access_token": "access-token",
				"token_type":   "Bearer",
				"id_token":     server.idToken(),
			})
		case "/oauth2/v2/userinfo":
			writeTestJSON(t, w, map[string]any{
				"sub":            server.identity.Sub,
				"email":          server.identity.Email,
				"email_verified": server.identity.EmailVerified,
				"name":           server.identity.Name,
				"picture":        server.identity.Picture,
			})
		default:
			http.NotFound(w, r)
		}
	}))
	server.Server = httpServer

	return server
}

func (s *googleOIDCTestServer) idToken() string {
	header := map[string]string{
		"alg": "HS256",
		"typ": "JWT",
	}
	claims := map[string]any{
		"iss":            s.URL,
		"sub":            s.identity.Sub,
		"aud":            "google-client",
		"email":          s.identity.Email,
		"email_verified": s.identity.EmailVerified,
		"name":           s.identity.Name,
		"picture":        s.identity.Picture,
		"nonce":          s.nonce,
		"exp":            time.Date(2030, 1, 1, 0, 0, 0, 0, time.UTC).Unix(),
	}
	headerJSON, _ := json.Marshal(header)
	claimsJSON, _ := json.Marshal(claims)
	unsigned := base64.RawURLEncoding.EncodeToString(headerJSON) + "." + base64.RawURLEncoding.EncodeToString(claimsJSON)
	mac := hmac.New(sha256.New, []byte(s.signingSecret))
	_, _ = mac.Write([]byte(unsigned))
	signature := base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
	return unsigned + "." + signature
}

func writeTestJSON(t *testing.T, w http.ResponseWriter, payload any) {
	t.Helper()
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(payload); err != nil {
		t.Fatalf("json encode: %v", err)
	}
}

func mustQueryParam(t *testing.T, rawURL, key string) string {
	t.Helper()
	parsed, err := url.Parse(rawURL)
	if err != nil {
		t.Fatalf("url.Parse(%q) error = %v", rawURL, err)
	}
	value := parsed.Query().Get(key)
	if value == "" {
		t.Fatalf("query %q missing in %q", key, rawURL)
	}
	return value
}

func loadOIDCFlowNonce(t *testing.T, ctx context.Context, pool *pgxpool.Pool, state string) string {
	t.Helper()

	var nonce string
	if err := pool.QueryRow(ctx, `SELECT nonce FROM auth_oidc_flows WHERE state_hash = $1`, HashOpaqueToken(state)).Scan(&nonce); err != nil {
		t.Fatalf("load flow nonce: %v", err)
	}
	return nonce
}
