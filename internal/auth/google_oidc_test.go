package auth

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/url"
	"strings"
	"testing"
)

func TestRewriteLocalOIDCTransportEndpoints(t *testing.T) {
	doc := googleDiscoveryDocument{
		Issuer:                "http://localhost:4003",
		AuthorizationEndpoint: "http://localhost:4003/o/oauth2/v2/auth",
		TokenEndpoint:         "http://localhost:4003/oauth2/token",
		UserInfoEndpoint:      "http://localhost:4003/oauth2/v2/userinfo",
		JWKSURI:               "http://localhost:4003/oauth2/v3/certs",
	}

	rewritten := rewriteLocalOIDCTransportEndpoints(doc, "http://host.docker.internal:4003/.well-known/openid-configuration")

	if rewritten.Issuer != doc.Issuer {
		t.Fatalf("issuer = %q, want %q", rewritten.Issuer, doc.Issuer)
	}
	if rewritten.AuthorizationEndpoint != doc.AuthorizationEndpoint {
		t.Fatalf("authorization endpoint = %q, want %q", rewritten.AuthorizationEndpoint, doc.AuthorizationEndpoint)
	}
	if rewritten.TokenEndpoint != "http://host.docker.internal:4003/oauth2/token" {
		t.Fatalf("token endpoint = %q", rewritten.TokenEndpoint)
	}
	if rewritten.UserInfoEndpoint != "http://host.docker.internal:4003/oauth2/v2/userinfo" {
		t.Fatalf("userinfo endpoint = %q", rewritten.UserInfoEndpoint)
	}
	if rewritten.JWKSURI != "http://host.docker.internal:4003/oauth2/v3/certs" {
		t.Fatalf("jwks uri = %q", rewritten.JWKSURI)
	}
}

func TestRewriteLocalOIDCTransportEndpointsSkipsNonLocalIssuers(t *testing.T) {
	doc := googleDiscoveryDocument{
		Issuer:                "https://accounts.google.com",
		AuthorizationEndpoint: "https://accounts.google.com/o/oauth2/v2/auth",
		TokenEndpoint:         "https://oauth2.googleapis.com/token",
		UserInfoEndpoint:      "https://openidconnect.googleapis.com/v1/userinfo",
		JWKSURI:               "https://www.googleapis.com/oauth2/v3/certs",
	}

	rewritten := rewriteLocalOIDCTransportEndpoints(doc, "http://host.docker.internal:4003/.well-known/openid-configuration")

	if rewritten.Issuer != doc.Issuer ||
		rewritten.AuthorizationEndpoint != doc.AuthorizationEndpoint ||
		rewritten.TokenEndpoint != doc.TokenEndpoint ||
		rewritten.UserInfoEndpoint != doc.UserInfoEndpoint ||
		rewritten.JWKSURI != doc.JWKSURI {
		t.Fatalf("non-local discovery doc was rewritten: %#v", rewritten)
	}
}

type staticGoogleTransport struct {
	discovery googleDiscoveryDocument
}

func (s staticGoogleTransport) Do(req *http.Request) (*http.Response, error) {
	return jsonResponse(tinyResponseWriter{
		status: http.StatusOK,
		body:   mustJSON(req.Context(), s.discovery),
	}), nil
}

type tinyResponseWriter struct {
	status int
	body   string
}

func jsonResponse(w tinyResponseWriter) *http.Response {
	return &http.Response{
		StatusCode: w.status,
		Body:       io.NopCloser(strings.NewReader(w.body)),
		Header:     make(http.Header),
	}
}

func mustJSON(_ context.Context, v any) string {
	bytes, err := json.Marshal(v)
	if err != nil {
		panic(err)
	}
	return string(bytes)
}

func mustGoogleQueryParam(t *testing.T, rawURL, key string) string {
	t.Helper()
	parsed, err := url.Parse(rawURL)
	if err != nil {
		t.Fatalf("url.Parse() error = %v", err)
	}
	value := parsed.Query().Get(key)
	if value == "" {
		t.Fatalf("query param %q missing from %q", key, rawURL)
	}
	return value
}

func TestGoogleOAuthProviderAuthorizationURLAddsHostedDomainHint(t *testing.T) {
	provider := NewGoogleOAuthProvider(GoogleOAuthProviderConfig{
		ClientID:     "google-client",
		ClientSecret: "google-secret",
		DiscoveryURL: "http://127.0.0.1:4002/.well-known/openid-configuration",
		Policy:       AllowGoogleHostedDomains("pandai.org"),
		AdminBaseURL: "http://localhost:3000",
	}, staticGoogleTransport{discovery: googleDiscoveryDocument{
		Issuer:                "https://accounts.google.com",
		AuthorizationEndpoint: "https://accounts.google.com/o/oauth2/v2/auth",
		TokenEndpoint:         "https://oauth2.googleapis.com/token",
	}}, nil)

	authURL, err := provider.AuthorizationURL(context.Background(), "http://localhost:8080/api/auth/google/callback", "state-1", "nonce-1", "challenge-1", "/dashboard")
	if err != nil {
		t.Fatalf("AuthorizationURL() error = %v", err)
	}

	if got := mustGoogleQueryParam(t, authURL, "hd"); got != "pandai.org" {
		t.Fatalf("hd = %q, want pandai.org", got)
	}
	if got := mustGoogleQueryParam(t, authURL, "redirect_uri"); got != "http://localhost:8080/api/auth/google/callback" {
		t.Fatalf("redirect_uri = %q, want callback URL", got)
	}
}

func TestGoogleOAuthProviderValidateIdentity_AllowedDomain(t *testing.T) {
	provider := NewGoogleOAuthProvider(GoogleOAuthProviderConfig{
		ClientID:     "google-client",
		ClientSecret: "google-secret",
		DiscoveryURL: "https://accounts.google.com/.well-known/openid-configuration",
		Policy:       AllowGoogleHostedDomains("pandai.org"),
		AdminBaseURL: "http://localhost:3000",
	}, nil, nil)

	tests := []struct {
		name     string
		identity googleIdentity
		wantErr  error
	}{
		{
			name: "hosted domain matches",
			identity: googleIdentity{
				Email:         "teacher@pandai.org",
				EmailVerified: true,
				HostedDomain:  "pandai.org",
			},
		},
		{
			name: "email domain matches without hosted domain",
			identity: googleIdentity{
				Email:         "teacher@pandai.org",
				EmailVerified: true,
			},
		},
		{
			name: "wrong hosted domain rejected",
			identity: googleIdentity{
				Email:         "teacher@pandai.org",
				EmailVerified: true,
				HostedDomain:  "gmail.com",
			},
			wantErr: ErrGoogleDomainNotAllowed,
		},
		{
			name: "wrong email domain rejected",
			identity: googleIdentity{
				Email:         "teacher@yahoo.com",
				EmailVerified: true,
			},
			wantErr: ErrGoogleDomainNotAllowed,
		},
		{
			name: "unverified rejected",
			identity: googleIdentity{
				Email:         "teacher@pandai.org",
				EmailVerified: false,
				HostedDomain:  "pandai.org",
			},
			wantErr: ErrGoogleDomainNotAllowed,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := provider.ValidateIdentity(tt.identity)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("ValidateIdentity() error = %v, want %v", err, tt.wantErr)
			}
		})
	}
}

func TestGoogleAuthoritativeEmail_AllowsVerifiedLocalOIDCIdentity(t *testing.T) {
	if !googleAuthoritativeEmail(googleIdentity{
		Email:         "platform-admin@example.com",
		EmailVerified: true,
		LocalIssuer:   true,
	}) {
		t.Fatal("googleAuthoritativeEmail() = false, want true for verified local OIDC identity")
	}
}
