// Copyright 2026 the P&AI authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package chat

import (
	"context"
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"math/big"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"
)

func TestTeamsAuthenticatorCachesConnectorToken(t *testing.T) {
	var requests atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		requests.Add(1)
		if err := request.ParseForm(); err != nil {
			t.Fatalf("ParseForm() error = %v", err)
		}
		if request.Form.Get("grant_type") != "client_credentials" ||
			request.Form.Get("client_id") != "app-123" ||
			request.Form.Get("client_secret") != "app-secret" ||
			request.Form.Get("scope") != teamsConnectorScope {
			t.Fatalf("token form = %v", request.Form)
		}
		_ = json.NewEncoder(response).Encode(map[string]any{
			"access_token": "connector-token",
			"expires_in":   3600,
		})
	}))
	defer server.Close()

	authenticator, err := NewTeamsAuthenticator("app-123", "app-secret", "")
	if err != nil {
		t.Fatalf("NewTeamsAuthenticator() error = %v", err)
	}
	authenticator.tokenURL = server.URL
	for range 2 {
		token, err := authenticator.Token(t.Context())
		if err != nil || token != "connector-token" {
			t.Fatalf("Token() = %q, %v", token, err)
		}
	}
	if requests.Load() != 1 {
		t.Fatalf("token requests = %d, want 1 cached request", requests.Load())
	}
}

func TestTeamsAuthenticatorValidatesAudienceIssuerAndServiceURL(t *testing.T) {
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("GenerateKey() error = %v", err)
	}
	var serverURL string
	var metadataRequests atomic.Int32
	var keyRequests atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		switch request.URL.Path {
		case "/openid":
			metadataRequests.Add(1)
			_ = json.NewEncoder(response).Encode(map[string]string{
				"issuer":   "https://api.botframework.test",
				"jwks_uri": serverURL + "/keys",
			})
		case "/keys":
			keyRequests.Add(1)
			_ = json.NewEncoder(response).Encode(map[string]any{
				"keys": []map[string]string{teamsTestJWK("key-1", &privateKey.PublicKey)},
			})
		default:
			http.NotFound(response, request)
		}
	}))
	serverURL = server.URL
	defer server.Close()

	authenticator, err := NewTeamsAuthenticator("app-123", "app-secret", "")
	if err != nil {
		t.Fatalf("NewTeamsAuthenticator() error = %v", err)
	}
	authenticator.metadataURL = server.URL + "/openid"
	now := time.Unix(1_800_000_000, 0)
	authenticator.now = func() time.Time { return now }
	serviceURL := "https://smba.trafficmanager.net/teams/"
	token := teamsTestJWT(t, privateKey, map[string]any{
		"iss":        "https://api.botframework.test",
		"aud":        "app-123",
		"exp":        now.Add(time.Hour).Unix(),
		"nbf":        now.Add(-time.Minute).Unix(),
		"serviceurl": serviceURL,
	})

	if err := authenticator.Validate(context.Background(), token, serviceURL); err != nil {
		t.Fatalf("Validate() error = %v", err)
	}
	if err := authenticator.Validate(context.Background(), token, "https://smba.trafficmanager.net/other/"); err == nil {
		t.Fatal("Validate() accepted an activity service URL absent from the token")
	}
	unknownKeyToken := teamsTestJWTWithKeyID(t, privateKey, "unknown-key", map[string]any{
		"iss":        "https://api.botframework.test",
		"aud":        "app-123",
		"exp":        now.Add(time.Hour).Unix(),
		"serviceurl": serviceURL,
	})
	for range 2 {
		if err := authenticator.Validate(context.Background(), unknownKeyToken, serviceURL); err == nil {
			t.Fatal("Validate() accepted an unknown signing key")
		}
	}
	if metadataRequests.Load() != 1 || keyRequests.Load() != 1 {
		t.Fatalf(
			"OpenID requests = metadata:%d keys:%d, want one cached fetch despite unknown kid",
			metadataRequests.Load(),
			keyRequests.Load(),
		)
	}
}

func teamsTestJWK(keyID string, key *rsa.PublicKey) map[string]string {
	exponent := big.NewInt(int64(key.E)).Bytes()
	return map[string]string{
		"kid": keyID,
		"kty": "RSA",
		"n":   base64.RawURLEncoding.EncodeToString(key.N.Bytes()),
		"e":   base64.RawURLEncoding.EncodeToString(exponent),
	}
}

func teamsTestJWT(t *testing.T, key *rsa.PrivateKey, claims map[string]any) string {
	return teamsTestJWTWithKeyID(t, key, "key-1", claims)
}

func teamsTestJWTWithKeyID(
	t *testing.T,
	key *rsa.PrivateKey,
	keyID string,
	claims map[string]any,
) string {
	t.Helper()
	header, _ := json.Marshal(map[string]string{"alg": "RS256", "kid": keyID})
	payload, _ := json.Marshal(claims)
	unsigned := base64.RawURLEncoding.EncodeToString(header) + "." +
		base64.RawURLEncoding.EncodeToString(payload)
	digest := sha256.Sum256([]byte(unsigned))
	signature, err := rsa.SignPKCS1v15(rand.Reader, key, crypto.SHA256, digest[:])
	if err != nil {
		t.Fatalf("SignPKCS1v15() error = %v", err)
	}
	return unsigned + "." + base64.RawURLEncoding.EncodeToString(signature)
}
