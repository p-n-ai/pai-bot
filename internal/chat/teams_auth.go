// Copyright 2026 the P&AI authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package chat

import (
	"context"
	"crypto"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"math/big"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"
)

const (
	teamsOpenIDMetadataURL = "https://login.botframework.com/v1/.well-known/openidconfiguration"
	teamsConnectorScope    = "https://api.botframework.com/.default"
)

// TeamsAuthenticator validates Bot Framework activities and obtains Connector
// API tokens with the same application credentials.
type TeamsAuthenticator struct {
	appID          string
	appPassword    string
	metadataURL    string
	tokenURL       string
	client         *http.Client
	now            func() time.Time
	tokenMu        sync.Mutex
	keysMu         sync.Mutex
	outboundToken  string
	tokenExpiresAt time.Time
	keys           map[string]*rsa.PublicKey
	keysExpiresAt  time.Time
	issuer         string
}

func NewTeamsAuthenticator(appID, appPassword, tenantID string) (*TeamsAuthenticator, error) {
	appID = strings.TrimSpace(appID)
	appPassword = strings.TrimSpace(appPassword)
	if appID == "" || appPassword == "" {
		return nil, fmt.Errorf("teams app ID and password are required")
	}
	tenantID = strings.TrimSpace(tenantID)
	if tenantID == "" {
		tenantID = "botframework.com"
	}
	return &TeamsAuthenticator{
		appID:       appID,
		appPassword: appPassword,
		metadataURL: teamsOpenIDMetadataURL,
		tokenURL: "https://login.microsoftonline.com/" +
			url.PathEscape(tenantID) +
			"/oauth2/v2.0/token",
		client: newTeamsAuthenticationHTTPClient(),
		now:    time.Now,
	}, nil
}

func (a *TeamsAuthenticator) Token(ctx context.Context) (string, error) {
	a.tokenMu.Lock()
	defer a.tokenMu.Unlock()

	now := a.now()
	if a.outboundToken != "" && now.Add(time.Minute).Before(a.tokenExpiresAt) {
		return a.outboundToken, nil
	}
	form := url.Values{
		"client_id":     {a.appID},
		"client_secret": {a.appPassword},
		"grant_type":    {"client_credentials"},
		"scope":         {teamsConnectorScope},
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, a.tokenURL, strings.NewReader(form.Encode()))
	if err != nil {
		return "", fmt.Errorf("create Teams token request: %w", err)
	}
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	response, err := a.client.Do(request)
	if err != nil {
		return "", fmt.Errorf("request Teams token: %w", err)
	}
	defer func() { _ = response.Body.Close() }()
	if response.StatusCode != http.StatusOK {
		return "", fmt.Errorf("teams token endpoint returned status %d", response.StatusCode)
	}
	var payload struct {
		AccessToken string `json:"access_token"`
		ExpiresIn   int    `json:"expires_in"`
	}
	if err := json.NewDecoder(io.LimitReader(response.Body, 1<<20)).Decode(&payload); err != nil {
		return "", fmt.Errorf("decode Teams token response: %w", err)
	}
	if strings.TrimSpace(payload.AccessToken) == "" || payload.ExpiresIn <= 0 {
		return "", fmt.Errorf("teams token response is incomplete")
	}
	a.outboundToken = payload.AccessToken
	a.tokenExpiresAt = now.Add(time.Duration(payload.ExpiresIn) * time.Second)
	return a.outboundToken, nil
}

func (a *TeamsAuthenticator) Validate(ctx context.Context, token, serviceURL string) error {
	header, claims, signed, signature, err := parseTeamsJWT(token)
	if err != nil {
		return err
	}
	if header.Alg != "RS256" || header.KeyID == "" {
		return fmt.Errorf("unsupported teams token signature")
	}
	keys, issuer, err := a.verificationKeys(ctx)
	if err != nil {
		return err
	}
	key := keys[header.KeyID]
	if key == nil {
		return fmt.Errorf("teams signing key is unavailable")
	}
	digest := sha256.Sum256(signed)
	if err := rsa.VerifyPKCS1v15(key, crypto.SHA256, digest[:], signature); err != nil {
		return fmt.Errorf("invalid teams token signature")
	}

	now := a.now().Unix()
	if claims.ExpiresAt <= now || (claims.NotBefore != 0 && claims.NotBefore > now+60) {
		return fmt.Errorf("teams token is expired or not active")
	}
	if claims.Issuer != issuer || !claims.Audience.Contains(a.appID) {
		return fmt.Errorf("teams token issuer or audience is invalid")
	}
	if claims.ServiceURL == "" || claims.ServiceURL != serviceURL {
		return fmt.Errorf("teams token service URL does not match activity")
	}
	return nil
}

func (a *TeamsAuthenticator) verificationKeys(ctx context.Context) (map[string]*rsa.PublicKey, string, error) {
	a.keysMu.Lock()
	defer a.keysMu.Unlock()

	now := a.now()
	if a.keys != nil && now.Before(a.keysExpiresAt) {
		return a.keys, a.issuer, nil
	}
	var metadata struct {
		Issuer  string `json:"issuer"`
		JWKSURL string `json:"jwks_uri"`
	}
	if err := a.getJSON(ctx, a.metadataURL, &metadata); err != nil {
		return nil, "", fmt.Errorf("load Teams OpenID metadata: %w", err)
	}
	if metadata.Issuer == "" || metadata.JWKSURL == "" {
		return nil, "", fmt.Errorf("teams OpenID metadata is incomplete")
	}
	if a.metadataURL == teamsOpenIDMetadataURL && !trustedTeamsJWKSURL(metadata.JWKSURL) {
		return nil, "", fmt.Errorf("teams OpenID metadata returned an untrusted JWKS URL")
	}
	var set struct {
		Keys []teamsJWK `json:"keys"`
	}
	if err := a.getJSON(ctx, metadata.JWKSURL, &set); err != nil {
		return nil, "", fmt.Errorf("load Teams signing keys: %w", err)
	}
	keys := make(map[string]*rsa.PublicKey, len(set.Keys))
	for _, jwk := range set.Keys {
		if key, err := jwk.rsaPublicKey(); err == nil && jwk.KeyID != "" {
			keys[jwk.KeyID] = key
		}
	}
	if len(keys) == 0 {
		return nil, "", fmt.Errorf("teams signing key set is empty")
	}
	a.keys = keys
	a.issuer = metadata.Issuer
	a.keysExpiresAt = now.Add(15 * time.Minute)
	return keys, metadata.Issuer, nil
}

func newTeamsAuthenticationHTTPClient() *http.Client {
	return &http.Client{
		Timeout: 15 * time.Second,
		CheckRedirect: func(*http.Request, []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}
}

func (a *TeamsAuthenticator) getJSON(ctx context.Context, endpoint string, target any) error {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return err
	}
	response, err := a.client.Do(request)
	if err != nil {
		return err
	}
	defer func() { _ = response.Body.Close() }()
	if response.StatusCode != http.StatusOK {
		return fmt.Errorf("status %d", response.StatusCode)
	}
	return json.NewDecoder(io.LimitReader(response.Body, 2<<20)).Decode(target)
}

func trustedTeamsJWKSURL(raw string) bool {
	parsed, err := url.Parse(raw)
	return err == nil &&
		parsed.Scheme == "https" &&
		parsed.Hostname() == "login.botframework.com" &&
		parsed.User == nil &&
		parsed.Port() == ""
}

type teamsJWTHeader struct {
	Alg   string `json:"alg"`
	KeyID string `json:"kid"`
}

type teamsAudience []string

func (a *teamsAudience) UnmarshalJSON(data []byte) error {
	var single string
	if err := json.Unmarshal(data, &single); err == nil {
		*a = teamsAudience{single}
		return nil
	}
	var multiple []string
	if err := json.Unmarshal(data, &multiple); err != nil {
		return err
	}
	*a = multiple
	return nil
}

func (a teamsAudience) Contains(want string) bool {
	for _, value := range a {
		if value == want {
			return true
		}
	}
	return false
}

type teamsJWTClaims struct {
	Issuer     string        `json:"iss"`
	Audience   teamsAudience `json:"aud"`
	ExpiresAt  int64         `json:"exp"`
	NotBefore  int64         `json:"nbf"`
	ServiceURL string        `json:"serviceurl"`
}

func parseTeamsJWT(token string) (teamsJWTHeader, teamsJWTClaims, []byte, []byte, error) {
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return teamsJWTHeader{}, teamsJWTClaims{}, nil, nil, fmt.Errorf("invalid Teams token")
	}
	decode := base64.RawURLEncoding.DecodeString
	headerBytes, err := decode(parts[0])
	if err != nil {
		return teamsJWTHeader{}, teamsJWTClaims{}, nil, nil, fmt.Errorf("invalid Teams token header")
	}
	claimsBytes, err := decode(parts[1])
	if err != nil {
		return teamsJWTHeader{}, teamsJWTClaims{}, nil, nil, fmt.Errorf("invalid Teams token claims")
	}
	signature, err := decode(parts[2])
	if err != nil {
		return teamsJWTHeader{}, teamsJWTClaims{}, nil, nil, fmt.Errorf("invalid Teams token signature")
	}
	var header teamsJWTHeader
	var claims teamsJWTClaims
	if json.Unmarshal(headerBytes, &header) != nil || json.Unmarshal(claimsBytes, &claims) != nil {
		return teamsJWTHeader{}, teamsJWTClaims{}, nil, nil, fmt.Errorf("invalid Teams token payload")
	}
	return header, claims, []byte(parts[0] + "." + parts[1]), signature, nil
}

type teamsJWK struct {
	KeyID       string   `json:"kid"`
	KeyType     string   `json:"kty"`
	Modulus     string   `json:"n"`
	Exponent    string   `json:"e"`
	Certificate []string `json:"x5c"`
}

func (j teamsJWK) rsaPublicKey() (*rsa.PublicKey, error) {
	if j.KeyType != "" && j.KeyType != "RSA" {
		return nil, fmt.Errorf("unsupported key type")
	}
	if j.Modulus != "" && j.Exponent != "" {
		modulusBytes, err := base64.RawURLEncoding.DecodeString(j.Modulus)
		if err != nil {
			return nil, err
		}
		exponentBytes, err := base64.RawURLEncoding.DecodeString(j.Exponent)
		if err != nil {
			return nil, err
		}
		exponentValue := new(big.Int).SetBytes(exponentBytes)
		if !exponentValue.IsInt64() {
			return nil, fmt.Errorf("invalid RSA exponent")
		}
		exponent, err := strconv.Atoi(exponentValue.String())
		if err != nil || exponent < 3 {
			return nil, fmt.Errorf("invalid RSA exponent")
		}
		return &rsa.PublicKey{N: new(big.Int).SetBytes(modulusBytes), E: exponent}, nil
	}
	if len(j.Certificate) == 0 {
		return nil, fmt.Errorf("RSA key material is missing")
	}
	certificateDER, err := base64.StdEncoding.DecodeString(j.Certificate[0])
	if err != nil {
		return nil, err
	}
	certificate, err := x509.ParseCertificate(certificateDER)
	if err != nil {
		return nil, err
	}
	key, ok := certificate.PublicKey.(*rsa.PublicKey)
	if !ok {
		return nil, fmt.Errorf("certificate does not contain an RSA key")
	}
	return key, nil
}

var _ TeamsTokenValidator = (*TeamsAuthenticator)(nil)
var _ TeamsTokenProvider = (*TeamsAuthenticator)(nil)
