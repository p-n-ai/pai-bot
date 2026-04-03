package auth

import (
	"context"
	"crypto"
	"crypto/hmac"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/jackc/pgx/v5"
)

const googleOIDCFlowTTL = 10 * time.Minute

var defaultHTTPClient = http.Client{Timeout: 10 * time.Second}

type HTTPDoer interface {
	Do(req *http.Request) (*http.Response, error)
}

type httpClient struct {
	client *http.Client
}

func (c *httpClient) Do(req *http.Request) (*http.Response, error) {
	return c.client.Do(req)
}

type GoogleOAuthProviderConfig struct {
	ClientID              string
	ClientSecret          string
	DiscoveryURL          string
	Policy                GoogleOAuthPolicy
	AdminBaseURL          string
	EmulatorSigningSecret string
}

type GoogleOAuthPolicy struct {
	AllowedDomains []string
}

func AllowGoogleHostedDomains(domains ...string) GoogleOAuthPolicy {
	normalized := make([]string, 0, len(domains))
	seen := make(map[string]struct{}, len(domains))
	for _, domain := range domains {
		domain = normalizeHostedDomain(domain)
		if domain == "" {
			continue
		}
		if _, ok := seen[domain]; ok {
			continue
		}
		seen[domain] = struct{}{}
		normalized = append(normalized, domain)
	}
	return GoogleOAuthPolicy{AllowedDomains: normalized}
}

type GoogleOAuthProvider struct {
	cfg  GoogleOAuthProviderConfig
	doer HTTPDoer
	now  func() time.Time

	mu              sync.RWMutex
	discovery       googleDiscoveryDocument
	discoveryLoaded time.Time
}

type googleDiscoveryDocument struct {
	Issuer                string   `json:"issuer"`
	AuthorizationEndpoint string   `json:"authorization_endpoint"`
	TokenEndpoint         string   `json:"token_endpoint"`
	UserInfoEndpoint      string   `json:"userinfo_endpoint"`
	JWKSURI               string   `json:"jwks_uri"`
	IDTokenSigningAlgs    []string `json:"id_token_signing_alg_values_supported"`
}

type googleTokenResponse struct {
	AccessToken string `json:"access_token"`
	IDToken     string `json:"id_token"`
	TokenType   string `json:"token_type"`
}

type googleUserInfo struct {
	Sub           string `json:"sub"`
	Email         string `json:"email"`
	EmailVerified bool   `json:"email_verified"`
	Name          string `json:"name"`
	Picture       string `json:"picture"`
}

type googleIdentity struct {
	Sub           string
	Email         string
	EmailVerified bool
	HostedDomain  string
	LocalIssuer   bool
	Name          string
	Picture       string
}

type googleFlowRecord struct {
	FlowType     string
	UserID       string
	NextPath     string
	Nonce        string
	PKCEVerifier string
}

type googleIDTokenClaims struct {
	Issuer        string      `json:"iss"`
	Subject       string      `json:"sub"`
	Audience      interface{} `json:"aud"`
	AuthorizedFor string      `json:"azp"`
	Email         string      `json:"email"`
	Name          string      `json:"name"`
	Picture       string      `json:"picture"`
	HostedDomain  string      `json:"hd"`
	Nonce         string      `json:"nonce"`
	ExpiresAt     int64       `json:"exp"`
	EmailVerified interface{} `json:"email_verified"`
}

func NewGoogleOAuthProvider(cfg GoogleOAuthProviderConfig, doer HTTPDoer, now func() time.Time) *GoogleOAuthProvider {
	if doer == nil {
		doer = &httpClient{client: &defaultHTTPClient}
	}
	if now == nil {
		now = time.Now
	}
	return &GoogleOAuthProvider{
		cfg:  cfg,
		doer: doer,
		now:  now,
	}
}

func (p *GoogleOAuthProvider) Configured() bool {
	return strings.TrimSpace(p.cfg.ClientID) != "" &&
		strings.TrimSpace(p.cfg.ClientSecret) != "" &&
		strings.TrimSpace(p.cfg.DiscoveryURL) != ""
}

func (p *GoogleOAuthProvider) AdminRedirect(path string) string {
	sanitized := sanitizeNextPath(path, "/login")
	if strings.TrimSpace(p.cfg.AdminBaseURL) == "" {
		return sanitized
	}
	base, err := url.Parse(strings.TrimRight(p.cfg.AdminBaseURL, "/"))
	if err != nil {
		return sanitized
	}
	rel, err := url.Parse(sanitized)
	if err != nil {
		return sanitized
	}
	return base.ResolveReference(rel).String()
}

func (p *GoogleOAuthProvider) AuthorizationURL(ctx context.Context, redirectURL, state, nonce, challenge, nextPath string) (string, error) {
	discovery, err := p.discoveryDocument(ctx)
	if err != nil {
		return "", err
	}
	redirectURL = strings.TrimSpace(redirectURL)
	if redirectURL == "" {
		return "", ErrAuthFlowInvalid
	}
	values := url.Values{}
	values.Set("client_id", p.cfg.ClientID)
	values.Set("redirect_uri", redirectURL)
	values.Set("response_type", "code")
	values.Set("scope", "openid email profile")
	values.Set("state", state)
	values.Set("nonce", nonce)
	values.Set("code_challenge", challenge)
	values.Set("code_challenge_method", "S256")
	if hostedDomain, ok := p.authorizationHostedDomainHint(); ok {
		values.Set("hd", hostedDomain)
	}
	return discovery.AuthorizationEndpoint + "?" + values.Encode(), nil
}

func (p *GoogleOAuthProvider) ExchangeCode(ctx context.Context, code, verifier, redirectURL string) (googleTokenResponse, error) {
	discovery, err := p.discoveryDocument(ctx)
	if err != nil {
		return googleTokenResponse{}, err
	}
	redirectURL = strings.TrimSpace(redirectURL)
	if redirectURL == "" {
		return googleTokenResponse{}, ErrAuthFlowInvalid
	}

	form := url.Values{}
	form.Set("grant_type", "authorization_code")
	form.Set("client_id", p.cfg.ClientID)
	form.Set("redirect_uri", redirectURL)
	form.Set("code", code)
	form.Set("code_verifier", verifier)
	if strings.TrimSpace(p.cfg.ClientSecret) != "" {
		form.Set("client_secret", p.cfg.ClientSecret)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, discovery.TokenEndpoint, strings.NewReader(form.Encode()))
	if err != nil {
		return googleTokenResponse{}, fmt.Errorf("build token request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	res, err := p.doer.Do(req)
	if err != nil {
		return googleTokenResponse{}, fmt.Errorf("exchange Google auth code: %w", err)
	}
	defer func() { _ = res.Body.Close() }()

	if res.StatusCode < 200 || res.StatusCode >= 300 {
		return googleTokenResponse{}, fmt.Errorf("exchange Google auth code: unexpected status %d", res.StatusCode)
	}

	var payload googleTokenResponse
	if err := json.NewDecoder(res.Body).Decode(&payload); err != nil {
		return googleTokenResponse{}, fmt.Errorf("decode Google token response: %w", err)
	}
	if payload.AccessToken == "" || payload.IDToken == "" {
		return googleTokenResponse{}, errors.New("google token response missing tokens")
	}
	return payload, nil
}

func (p *GoogleOAuthProvider) ResolveIdentity(ctx context.Context, rawIDToken, accessToken, nonce string) (googleIdentity, error) {
	discovery, err := p.discoveryDocument(ctx)
	if err != nil {
		return googleIdentity{}, err
	}

	claims, err := p.verifyIDToken(ctx, discovery, rawIDToken, nonce)
	if err != nil {
		return googleIdentity{}, err
	}

	identity := googleIdentity{
		Sub:           claims.Subject,
		Email:         claims.Email,
		EmailVerified: googleEmailVerified(claims.EmailVerified),
		HostedDomain:  claims.HostedDomain,
		LocalIssuer:   isLocalOIDCIssuer(discovery.Issuer),
		Name:          claims.Name,
		Picture:       claims.Picture,
	}
	if discovery.UserInfoEndpoint == "" || accessToken == "" {
		return identity, nil
	}

	userInfo, err := p.fetchUserInfo(ctx, discovery.UserInfoEndpoint, accessToken)
	if err != nil {
		return googleIdentity{}, err
	}
	if userInfo.Sub != claims.Subject {
		return googleIdentity{}, errors.New("google userinfo subject mismatch")
	}
	if strings.TrimSpace(userInfo.Email) != "" {
		identity.Email = userInfo.Email
	}
	identity.EmailVerified = identity.EmailVerified || userInfo.EmailVerified
	if strings.TrimSpace(userInfo.Name) != "" {
		identity.Name = userInfo.Name
	}
	if strings.TrimSpace(userInfo.Picture) != "" {
		identity.Picture = userInfo.Picture
	}

	return identity, nil
}

func (p *GoogleOAuthProvider) ValidateIdentity(identity googleIdentity) error {
	if len(p.cfg.Policy.AllowedDomains) == 0 {
		return nil
	}
	if !identity.EmailVerified {
		return ErrGoogleDomainNotAllowed
	}
	emailDomain := googleEmailDomain(identity.Email)
	hostedDomain := normalizeHostedDomain(identity.HostedDomain)
	if containsGoogleAllowedDomain(p.cfg.Policy.AllowedDomains, hostedDomain) {
		return nil
	}
	if hostedDomain != "" {
		return ErrGoogleDomainNotAllowed
	}
	if containsGoogleAllowedDomain(p.cfg.Policy.AllowedDomains, emailDomain) {
		return nil
	}
	return ErrGoogleDomainNotAllowed
}

func (p *GoogleOAuthProvider) authorizationHostedDomainHint() (string, bool) {
	if len(p.cfg.Policy.AllowedDomains) != 1 {
		return "", false
	}
	return p.cfg.Policy.AllowedDomains[0], true
}

func containsGoogleAllowedDomain(allowed []string, domain string) bool {
	domain = normalizeHostedDomain(domain)
	if domain == "" {
		return false
	}
	for _, allowedDomain := range allowed {
		if normalizeHostedDomain(allowedDomain) == domain {
			return true
		}
	}
	return false
}

func googleEmailDomain(email string) string {
	email = NormalizeIdentifier(email)
	at := strings.LastIndex(email, "@")
	if at == -1 || at == len(email)-1 {
		return ""
	}
	return normalizeHostedDomain(email[at+1:])
}

func (p *GoogleOAuthProvider) discoveryDocument(ctx context.Context) (googleDiscoveryDocument, error) {
	p.mu.RLock()
	if time.Since(p.discoveryLoaded) < 15*time.Minute && p.discovery.AuthorizationEndpoint != "" {
		doc := p.discovery
		p.mu.RUnlock()
		return doc, nil
	}
	p.mu.RUnlock()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, p.cfg.DiscoveryURL, nil)
	if err != nil {
		return googleDiscoveryDocument{}, fmt.Errorf("build discovery request: %w", err)
	}

	res, err := p.doer.Do(req)
	if err != nil {
		return googleDiscoveryDocument{}, fmt.Errorf("fetch discovery document: %w", err)
	}
	defer func() { _ = res.Body.Close() }()

	if res.StatusCode < 200 || res.StatusCode >= 300 {
		return googleDiscoveryDocument{}, fmt.Errorf("fetch discovery document: unexpected status %d", res.StatusCode)
	}

	var doc googleDiscoveryDocument
	if err := json.NewDecoder(res.Body).Decode(&doc); err != nil {
		return googleDiscoveryDocument{}, fmt.Errorf("decode discovery document: %w", err)
	}
	if doc.AuthorizationEndpoint == "" || doc.TokenEndpoint == "" || doc.Issuer == "" {
		return googleDiscoveryDocument{}, errors.New("discovery document missing required fields")
	}
	doc = rewriteLocalOIDCTransportEndpoints(doc, p.cfg.DiscoveryURL)

	p.mu.Lock()
	p.discovery = doc
	p.discoveryLoaded = p.now().UTC()
	p.mu.Unlock()

	return doc, nil
}

func (p *GoogleOAuthProvider) fetchUserInfo(ctx context.Context, endpoint, accessToken string) (googleUserInfo, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return googleUserInfo{}, fmt.Errorf("build userinfo request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)

	res, err := p.doer.Do(req)
	if err != nil {
		return googleUserInfo{}, fmt.Errorf("fetch Google userinfo: %w", err)
	}
	defer func() { _ = res.Body.Close() }()

	if res.StatusCode < 200 || res.StatusCode >= 300 {
		return googleUserInfo{}, fmt.Errorf("fetch Google userinfo: unexpected status %d", res.StatusCode)
	}

	var payload googleUserInfo
	if err := json.NewDecoder(res.Body).Decode(&payload); err != nil {
		return googleUserInfo{}, fmt.Errorf("decode Google userinfo: %w", err)
	}
	return payload, nil
}

func (p *GoogleOAuthProvider) verifyIDToken(ctx context.Context, discovery googleDiscoveryDocument, rawToken, nonce string) (googleIDTokenClaims, error) {
	header, claims, signingInput, signature, err := parseJWT(rawToken)
	if err != nil {
		return googleIDTokenClaims{}, err
	}

	var tokenClaims googleIDTokenClaims
	if err := json.Unmarshal(claims, &tokenClaims); err != nil {
		return googleIDTokenClaims{}, fmt.Errorf("decode Google ID token claims: %w", err)
	}

	if tokenClaims.Subject == "" || tokenClaims.Issuer == "" || tokenClaims.ExpiresAt == 0 {
		return googleIDTokenClaims{}, errors.New("google ID token missing required claims")
	}
	if tokenClaims.Nonce != nonce {
		return googleIDTokenClaims{}, errors.New("google ID token nonce mismatch")
	}
	if tokenClaims.Issuer != discovery.Issuer {
		return googleIDTokenClaims{}, errors.New("google ID token issuer mismatch")
	}
	if tokenClaims.ExpiresAt <= p.now().UTC().Unix() {
		return googleIDTokenClaims{}, ErrExpiredToken
	}
	if !googleAudienceContains(tokenClaims.Audience, p.cfg.ClientID) {
		return googleIDTokenClaims{}, errors.New("google ID token audience mismatch")
	}
	if tokenClaims.AuthorizedFor != "" && tokenClaims.AuthorizedFor != p.cfg.ClientID {
		return googleIDTokenClaims{}, errors.New("google ID token authorized party mismatch")
	}

	switch header.Algorithm {
	case "RS256":
		if err := p.verifyRS256(ctx, discovery.JWKSURI, header.KeyID, signingInput, signature); err != nil {
			return googleIDTokenClaims{}, err
		}
	case "HS256":
		if !isLocalOIDCIssuer(discovery.Issuer) {
			return googleIDTokenClaims{}, errors.New("HS256 tokens are only allowed for local emulator issuers")
		}
		if err := p.verifyHS256(signingInput, signature); err != nil {
			return googleIDTokenClaims{}, err
		}
	default:
		return googleIDTokenClaims{}, fmt.Errorf("unsupported Google ID token algorithm %q", header.Algorithm)
	}

	return tokenClaims, nil
}

func (p *GoogleOAuthProvider) verifyHS256(signingInput string, signature []byte) error {
	secret := strings.TrimSpace(p.cfg.EmulatorSigningSecret)
	if secret == "" {
		return errors.New("HS256 emulator token received without emulator signing secret")
	}
	mac := hmac.New(sha256.New, []byte(secret))
	_, _ = mac.Write([]byte(signingInput))
	if !hmac.Equal(signature, mac.Sum(nil)) {
		return errors.New("invalid Google HS256 token signature")
	}
	return nil
}

func (p *GoogleOAuthProvider) verifyRS256(ctx context.Context, jwksURI, kid, signingInput string, signature []byte) error {
	if jwksURI == "" {
		return errors.New("google JWKS URL missing")
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, jwksURI, nil)
	if err != nil {
		return fmt.Errorf("build JWKS request: %w", err)
	}
	res, err := p.doer.Do(req)
	if err != nil {
		return fmt.Errorf("fetch Google JWKS: %w", err)
	}
	defer func() { _ = res.Body.Close() }()

	if res.StatusCode < 200 || res.StatusCode >= 300 {
		return fmt.Errorf("fetch Google JWKS: unexpected status %d", res.StatusCode)
	}

	var payload struct {
		Keys []struct {
			KeyID string `json:"kid"`
			N     string `json:"n"`
			E     string `json:"e"`
		} `json:"keys"`
	}
	if err := json.NewDecoder(res.Body).Decode(&payload); err != nil {
		return fmt.Errorf("decode Google JWKS: %w", err)
	}

	for _, key := range payload.Keys {
		if key.KeyID != kid {
			continue
		}
		publicKey, err := rsaPublicKeyFromJWK(key.N, key.E)
		if err != nil {
			return err
		}
		sum := sha256.Sum256([]byte(signingInput))
		if err := rsa.VerifyPKCS1v15(publicKey, crypto.SHA256, sum[:], signature); err != nil {
			return fmt.Errorf("verify Google RS256 token signature: %w", err)
		}
		return nil
	}

	return errors.New("matching Google JWKS key not found")
}
func rsaPublicKeyFromJWK(nValue, eValue string) (*rsa.PublicKey, error) {
	nBytes, err := base64.RawURLEncoding.DecodeString(nValue)
	if err != nil {
		return nil, fmt.Errorf("decode JWK modulus: %w", err)
	}
	eBytes, err := base64.RawURLEncoding.DecodeString(eValue)
	if err != nil {
		return nil, fmt.Errorf("decode JWK exponent: %w", err)
	}

	exponent := 0
	for _, part := range eBytes {
		exponent = exponent<<8 + int(part)
	}
	if exponent == 0 {
		return nil, errors.New("invalid JWK exponent")
	}

	return &rsa.PublicKey{
		N: new(big.Int).SetBytes(nBytes),
		E: exponent,
	}, nil
}

type jwtHeader struct {
	Algorithm string `json:"alg"`
	KeyID     string `json:"kid"`
}

func parseJWT(token string) (jwtHeader, []byte, string, []byte, error) {
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return jwtHeader{}, nil, "", nil, ErrInvalidToken
	}

	headerBytes, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil {
		return jwtHeader{}, nil, "", nil, ErrInvalidToken
	}
	var header jwtHeader
	if err := json.Unmarshal(headerBytes, &header); err != nil {
		return jwtHeader{}, nil, "", nil, ErrInvalidToken
	}

	claims, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return jwtHeader{}, nil, "", nil, ErrInvalidToken
	}
	signature, err := base64.RawURLEncoding.DecodeString(parts[2])
	if err != nil {
		return jwtHeader{}, nil, "", nil, ErrInvalidToken
	}

	return header, claims, parts[0] + "." + parts[1], signature, nil
}

func googleAudienceContains(raw interface{}, want string) bool {
	switch value := raw.(type) {
	case string:
		return value == want
	case []interface{}:
		for _, item := range value {
			if text, ok := item.(string); ok && text == want {
				return true
			}
		}
	}
	return false
}

func googleEmailVerified(raw interface{}) bool {
	switch value := raw.(type) {
	case bool:
		return value
	case string:
		return strings.EqualFold(value, "true")
	default:
		return false
	}
}

func googleAuthoritativeEmail(identity googleIdentity) bool {
	if !identity.EmailVerified {
		return false
	}
	if identity.LocalIssuer {
		return true
	}
	email := NormalizeIdentifier(identity.Email)
	if strings.HasSuffix(email, "@gmail.com") {
		return true
	}
	return strings.TrimSpace(identity.HostedDomain) != ""
}

func normalizeHostedDomain(raw string) string {
	value := strings.ToLower(strings.TrimSpace(raw))
	value = strings.TrimPrefix(value, "@")
	return value
}

func isLocalOIDCIssuer(issuer string) bool {
	parsed, err := url.Parse(strings.TrimSpace(issuer))
	if err != nil {
		return false
	}
	host := strings.ToLower(parsed.Hostname())
	return host == "localhost" || host == "127.0.0.1"
}

func rewriteLocalOIDCTransportEndpoints(doc googleDiscoveryDocument, discoveryURL string) googleDiscoveryDocument {
	if !isLocalOIDCIssuer(doc.Issuer) {
		return doc
	}

	discoveryBase, err := url.Parse(strings.TrimSpace(discoveryURL))
	if err != nil || discoveryBase.Scheme == "" || discoveryBase.Host == "" {
		return doc
	}
	issuerURL, err := url.Parse(strings.TrimSpace(doc.Issuer))
	if err != nil || issuerURL.Host == "" {
		return doc
	}
	if strings.EqualFold(discoveryBase.Host, issuerURL.Host) && strings.EqualFold(discoveryBase.Scheme, issuerURL.Scheme) {
		return doc
	}

	doc.TokenEndpoint = rewriteLocalOIDCEndpoint(doc.TokenEndpoint, issuerURL, discoveryBase)
	doc.UserInfoEndpoint = rewriteLocalOIDCEndpoint(doc.UserInfoEndpoint, issuerURL, discoveryBase)
	doc.JWKSURI = rewriteLocalOIDCEndpoint(doc.JWKSURI, issuerURL, discoveryBase)
	return doc
}

func rewriteLocalOIDCEndpoint(raw string, issuerBase, discoveryBase *url.URL) string {
	if strings.TrimSpace(raw) == "" {
		return raw
	}
	parsed, err := url.Parse(raw)
	if err != nil {
		return raw
	}
	if !strings.EqualFold(parsed.Host, issuerBase.Host) || !strings.EqualFold(parsed.Scheme, issuerBase.Scheme) {
		return raw
	}
	parsed.Scheme = discoveryBase.Scheme
	parsed.Host = discoveryBase.Host
	return parsed.String()
}

func (s *PostgresService) StartGoogleLogin(ctx context.Context, req StartGoogleFlowRequest) (string, error) {
	return s.startGoogleFlow(ctx, "login", req)
}

func (s *PostgresService) StartGoogleLink(ctx context.Context, req StartGoogleFlowRequest) (string, error) {
	return s.startGoogleFlow(ctx, "link", req)
}

func (s *PostgresService) startGoogleFlow(ctx context.Context, flowType string, req StartGoogleFlowRequest) (string, error) {
	if s.google == nil || !s.google.Configured() {
		return "", ErrProviderNotConfigured
	}
	if strings.TrimSpace(req.RedirectURL) == "" {
		return "", ErrAuthFlowInvalid
	}

	ctx, cancel := context.WithTimeout(ctx, authDBTimeout)
	defer cancel()

	state, err := generateOpaqueToken()
	if err != nil {
		return "", fmt.Errorf("generate Google auth state: %w", err)
	}
	nonce, err := generateOpaqueToken()
	if err != nil {
		return "", fmt.Errorf("generate Google auth nonce: %w", err)
	}
	verifier, err := generateOpaqueToken()
	if err != nil {
		return "", fmt.Errorf("generate Google PKCE verifier: %w", err)
	}
	challenge := pkceChallenge(verifier)
	now := s.now().UTC()
	nextPath := defaultGoogleNextPath(flowType, req.NextPath)

	if _, err := s.pool.Exec(ctx, `
		INSERT INTO auth_oidc_flows (provider, flow_type, state_hash, nonce, pkce_verifier, user_id, next_path, expires_at, created_at)
		VALUES ('google', $1, $2, $3, $4, NULLIF($5, '')::uuid, $6, $7, $8)
	`, flowType, HashOpaqueToken(state), nonce, verifier, strings.TrimSpace(req.UserID), nextPath, now.Add(googleOIDCFlowTTL), now); err != nil {
		return "", fmt.Errorf("persist Google OIDC flow: %w", err)
	}

	return s.google.AuthorizationURL(ctx, req.RedirectURL, state, nonce, challenge, nextPath)
}

func (s *PostgresService) CompleteGoogleCallback(ctx context.Context, req GoogleCallbackRequest) (GoogleCallbackResult, error) {
	if s.google == nil || !s.google.Configured() {
		return GoogleCallbackResult{
			RedirectPath: s.googleLoginRedirectPath(),
		}, ErrProviderNotConfigured
	}
	if strings.TrimSpace(req.State) == "" || strings.TrimSpace(req.Code) == "" {
		return GoogleCallbackResult{
			RedirectPath: s.googleLoginRedirectPath(),
		}, ErrAuthFlowInvalid
	}
	if strings.TrimSpace(req.RedirectURL) == "" {
		return GoogleCallbackResult{
			RedirectPath: s.googleLoginRedirectPath(),
		}, ErrAuthFlowInvalid
	}

	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	flow, err := s.consumeGoogleFlow(ctx, req.State)
	if err != nil {
		return GoogleCallbackResult{
			RedirectPath: s.google.AdminRedirect("/login"),
		}, err
	}
	result := GoogleCallbackResult{
		RedirectPath: s.google.AdminRedirect(flow.NextPath),
	}

	tokenResponse, err := s.google.ExchangeCode(ctx, req.Code, flow.PKCEVerifier, req.RedirectURL)
	if err != nil {
		return result, err
	}

	identity, err := s.google.ResolveIdentity(ctx, tokenResponse.IDToken, tokenResponse.AccessToken, flow.Nonce)
	if err != nil {
		return result, err
	}
	if err := s.google.ValidateIdentity(identity); err != nil {
		return result, err
	}

	switch flow.FlowType {
	case "link":
		session, err := s.completeGoogleLink(ctx, flow.UserID, identity)
		if err != nil {
			return result, err
		}
		result.Linked = true
		result.Session = &session
		return result, nil
	default:
		session, linked, err := s.completeGoogleLogin(ctx, identity)
		if err != nil {
			return result, err
		}
		result.Linked = linked
		result.Session = &session
		return result, nil
	}
}

func (s *PostgresService) ListLinkedIdentities(ctx context.Context, userID string) ([]LinkedIdentity, error) {
	ctx, cancel := context.WithTimeout(ctx, authDBTimeout)
	defer cancel()

	rows, err := s.pool.Query(ctx, `
		SELECT provider, COALESCE(provider_email, identifier), linked_at, last_used_at
		FROM auth_identities
		WHERE user_id = $1::uuid
		  AND provider <> 'password'
		ORDER BY linked_at ASC NULLS LAST, created_at ASC
	`, userID)
	if err != nil {
		return nil, fmt.Errorf("list linked identities: %w", err)
	}
	defer rows.Close()

	identities := make([]LinkedIdentity, 0)
	for rows.Next() {
		var identity LinkedIdentity
		if err := rows.Scan(&identity.Provider, &identity.Email, &identity.LinkedAt, &identity.LastUsedAt); err != nil {
			return nil, fmt.Errorf("scan linked identity: %w", err)
		}
		identities = append(identities, identity)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate linked identities: %w", err)
	}
	return identities, nil
}

func (s *PostgresService) consumeGoogleFlow(ctx context.Context, state string) (googleFlowRecord, error) {
	now := s.now().UTC()
	var flow googleFlowRecord
	err := s.pool.QueryRow(ctx, `
		UPDATE auth_oidc_flows
		SET used_at = $2
		WHERE provider = 'google'
		  AND state_hash = $1
		  AND used_at IS NULL
		  AND expires_at > $2
		RETURNING flow_type, COALESCE(user_id::text, ''), COALESCE(next_path, ''), nonce, pkce_verifier
	`, HashOpaqueToken(state), now).Scan(&flow.FlowType, &flow.UserID, &flow.NextPath, &flow.Nonce, &flow.PKCEVerifier)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return googleFlowRecord{}, ErrAuthFlowInvalid
		}
		return googleFlowRecord{}, fmt.Errorf("consume Google auth flow: %w", err)
	}
	return flow, nil
}

func (s *PostgresService) completeGoogleLink(ctx context.Context, userID string, identity googleIdentity) (Session, error) {
	if strings.TrimSpace(userID) == "" {
		return Session{}, ErrAuthFlowInvalid
	}

	now := s.now().UTC()
	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return Session{}, fmt.Errorf("begin Google link transaction: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	user, err := loadSessionUserByUserID(ctx, tx, userID)
	if err != nil {
		return Session{}, err
	}
	if err := s.linkGoogleIdentityTx(ctx, tx, user, identity, now); err != nil {
		return Session{}, err
	}
	session, err := s.issueSession(ctx, tx, user, now)
	if err != nil {
		return Session{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return Session{}, fmt.Errorf("commit Google link transaction: %w", err)
	}
	return session, nil
}

func (s *PostgresService) completeGoogleLogin(ctx context.Context, identity googleIdentity) (Session, bool, error) {
	now := s.now().UTC()

	if user, ok, err := s.googleUserBySubject(ctx, identity.Sub); err != nil {
		return Session{}, false, err
	} else if ok {
		tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{})
		if err != nil {
			return Session{}, false, fmt.Errorf("begin Google login transaction: %w", err)
		}
		defer func() { _ = tx.Rollback(ctx) }()
		if err := s.touchGoogleIdentityTx(ctx, tx, user.UserID, identity, now); err != nil {
			return Session{}, false, err
		}
		session, err := s.issueSession(ctx, tx, user, now)
		if err != nil {
			return Session{}, false, err
		}
		if err := tx.Commit(ctx); err != nil {
			return Session{}, false, fmt.Errorf("commit Google login transaction: %w", err)
		}
		return session, false, nil
	}

	if !identity.EmailVerified || strings.TrimSpace(identity.Email) == "" {
		return Session{}, false, ErrIdentityLinkRequired
	}
	if !googleAuthoritativeEmail(identity) {
		return Session{}, false, ErrIdentityLinkRequired
	}

	candidates, err := s.passwordUsersByEmail(ctx, identity.Email)
	if err != nil {
		return Session{}, false, err
	}
	if len(candidates) == 0 {
		return Session{}, false, ErrIdentityLinkRequired
	}
	chosen := candidates[0]

	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return Session{}, false, fmt.Errorf("begin Google auto-link transaction: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	if err := s.linkGoogleIdentityTx(ctx, tx, chosen, identity, now); err != nil {
		return Session{}, false, err
	}
	session, err := s.issueSession(ctx, tx, chosen, now)
	if err != nil {
		return Session{}, false, err
	}
	session.TenantChoices, err = s.tenantOptionsByEmail(ctx, tx, chosen.Email)
	if err != nil {
		return Session{}, false, err
	}
	if err := tx.Commit(ctx); err != nil {
		return Session{}, false, fmt.Errorf("commit Google auto-link transaction: %w", err)
	}
	return session, true, nil
}

func (s *PostgresService) googleUserBySubject(ctx context.Context, subject string) (sessionUser, bool, error) {
	var user sessionUser
	err := s.pool.QueryRow(ctx, `
		SELECT u.id::text,
		       COALESCE(u.tenant_id::text, ''),
		       COALESCE(t.slug, ''),
		       COALESCE(t.name, ''),
		       u.role,
		       u.name,
		       COALESCE(
		           (SELECT identifier_normalized FROM auth_identities WHERE user_id = u.id AND provider = 'password' ORDER BY created_at ASC LIMIT 1),
		           (SELECT COALESCE(provider_email_normalized, identifier_normalized) FROM auth_identities WHERE user_id = u.id AND provider = 'google' ORDER BY created_at ASC LIMIT 1),
		           ''
		       ) AS email
		FROM auth_identities ai
		JOIN users u ON u.id = ai.user_id
		LEFT JOIN tenants t ON t.id = u.tenant_id
		WHERE ai.provider = 'google'
		  AND ai.provider_account_id = $1
		LIMIT 1
	`, subject).Scan(&user.UserID, &user.TenantID, &user.TenantSlug, &user.TenantName, &user.Role, &user.Name, &user.Email)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return sessionUser{}, false, nil
		}
		return sessionUser{}, false, fmt.Errorf("lookup Google identity: %w", err)
	}
	return user, true, nil
}

func (s *PostgresService) passwordUsersByEmail(ctx context.Context, email string) ([]sessionUser, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT u.id::text,
		       COALESCE(u.tenant_id::text, ''),
		       COALESCE(t.slug, ''),
		       COALESCE(t.name, ''),
		       u.role,
		       u.name,
		       ai.identifier_normalized
		FROM auth_identities ai
		JOIN users u ON u.id = ai.user_id
		LEFT JOIN tenants t ON t.id = u.tenant_id
		WHERE ai.provider = 'password'
		  AND ai.identifier_normalized = $1
		ORDER BY u.created_at ASC
		LIMIT 10
	`, NormalizeIdentifier(email))
	if err != nil {
		return nil, fmt.Errorf("lookup password identities by email: %w", err)
	}
	defer rows.Close()

	var users []sessionUser
	for rows.Next() {
		var user sessionUser
		if err := rows.Scan(&user.UserID, &user.TenantID, &user.TenantSlug, &user.TenantName, &user.Role, &user.Name, &user.Email); err != nil {
			return nil, fmt.Errorf("scan password identity candidate: %w", err)
		}
		users = append(users, user)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate password identity candidates: %w", err)
	}
	return users, nil
}

func (s *PostgresService) linkGoogleIdentityTx(ctx context.Context, tx pgx.Tx, user sessionUser, identity googleIdentity, now time.Time) error {
	var existingUserID string
	err := tx.QueryRow(ctx, `
		SELECT user_id::text
		FROM auth_identities
		WHERE provider = 'google'
		  AND provider_account_id = $1
		LIMIT 1
	`, identity.Sub).Scan(&existingUserID)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return fmt.Errorf("lookup existing Google link: %w", err)
	}
	if existingUserID != "" && existingUserID != user.UserID {
		return ErrIdentityAlreadyLinked
	}

	var existingIdentityID string
	var existingProviderAccountID string
	err = tx.QueryRow(ctx, `
		SELECT id::text, COALESCE(provider_account_id, '')
		FROM auth_identities
		WHERE user_id = $1::uuid
		  AND provider = 'google'
		ORDER BY linked_at DESC NULLS LAST, created_at DESC
		LIMIT 1
	`, user.UserID).Scan(&existingIdentityID, &existingProviderAccountID)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return fmt.Errorf("lookup current Google link: %w", err)
	}

	profileJSON, err := json.Marshal(map[string]string{
		"name":    identity.Name,
		"picture": identity.Picture,
	})
	if err != nil {
		return fmt.Errorf("marshal Google profile: %w", err)
	}

	emailNormalized := NormalizeIdentifier(identity.Email)
	if existingIdentityID != "" {
		if _, err := tx.Exec(ctx, `
			DELETE FROM auth_identities
			WHERE user_id = $1::uuid
			  AND provider = 'google'
			  AND id <> $2::uuid
		`, user.UserID, existingIdentityID); err != nil {
			return fmt.Errorf("prune extra Google identities: %w", err)
		}

		linkedAt := now
		if existingProviderAccountID == identity.Sub {
			_, err := tx.Exec(ctx, `
				UPDATE auth_identities
				SET provider_email = $2,
				    provider_email_normalized = NULLIF($3, ''),
				    provider_profile = $4::jsonb,
				    email_verified_at = CASE WHEN $5::boolean THEN COALESCE(email_verified_at, $6::timestamptz) ELSE email_verified_at END,
				    last_used_at = $6::timestamptz,
				    last_login_at = $6::timestamptz,
				    linked_at = COALESCE(linked_at, $6::timestamptz),
				    updated_at = $6::timestamptz
				WHERE id = $1::uuid
			`, existingIdentityID, identity.Email, emailNormalized, string(profileJSON), identity.EmailVerified, now)
			if err != nil {
				return fmt.Errorf("update Google identity: %w", err)
			}
			return nil
		}

		_, err := tx.Exec(ctx, `
			UPDATE auth_identities
			SET identifier = $2,
			    identifier_normalized = $2,
			    provider_account_id = $2,
			    provider_email = $3,
			    provider_email_normalized = NULLIF($4, ''),
			    provider_profile = $5::jsonb,
			    email_verified_at = CASE WHEN $6::boolean THEN COALESCE(email_verified_at, $7::timestamptz) ELSE email_verified_at END,
			    linked_at = $7::timestamptz,
			    last_used_at = $7::timestamptz,
			    last_login_at = $7::timestamptz,
			    updated_at = $7::timestamptz
			WHERE id = $1::uuid
		`, existingIdentityID, identity.Sub, identity.Email, emailNormalized, string(profileJSON), identity.EmailVerified, linkedAt)
		if err != nil {
			return fmt.Errorf("replace Google identity: %w", err)
		}
		return nil
	}

	if user.TenantID != "" {
		_, err = tx.Exec(ctx, `
			INSERT INTO auth_identities (
				user_id, tenant_id, provider, identifier, identifier_normalized, provider_account_id, provider_email, provider_email_normalized,
				provider_profile, email_verified_at, linked_at, last_used_at, last_login_at, created_at, updated_at
			)
			VALUES ($1::uuid, $2::uuid, 'google', $3, $3, $3, $4, NULLIF($5, ''), $6::jsonb, CASE WHEN $7::boolean THEN $8::timestamptz ELSE NULL END, $8::timestamptz, $8::timestamptz, $8::timestamptz, $8::timestamptz, $8::timestamptz)
		`, user.UserID, user.TenantID, identity.Sub, identity.Email, emailNormalized, string(profileJSON), identity.EmailVerified, now)
	} else {
		_, err = tx.Exec(ctx, `
			INSERT INTO auth_identities (
				user_id, tenant_id, provider, identifier, identifier_normalized, provider_account_id, provider_email, provider_email_normalized,
				provider_profile, email_verified_at, linked_at, last_used_at, last_login_at, created_at, updated_at
			)
			VALUES ($1::uuid, NULL, 'google', $2, $2, $2, $3, NULLIF($4, ''), $5::jsonb, CASE WHEN $6::boolean THEN $7::timestamptz ELSE NULL END, $7::timestamptz, $7::timestamptz, $7::timestamptz, $7::timestamptz, $7::timestamptz)
		`, user.UserID, identity.Sub, identity.Email, emailNormalized, string(profileJSON), identity.EmailVerified, now)
	}
	if err != nil {
		return fmt.Errorf("insert Google identity: %w", err)
	}
	return nil
}

func (s *PostgresService) touchGoogleIdentityTx(ctx context.Context, tx pgx.Tx, userID string, identity googleIdentity, now time.Time) error {
	profileJSON, err := json.Marshal(map[string]string{
		"name":    identity.Name,
		"picture": identity.Picture,
	})
	if err != nil {
		return fmt.Errorf("marshal Google profile: %w", err)
	}
	_, err = tx.Exec(ctx, `
		UPDATE auth_identities
		SET provider_email = $2,
		    provider_email_normalized = NULLIF($3, ''),
		    provider_profile = $4::jsonb,
		    email_verified_at = CASE WHEN $5::boolean THEN COALESCE(email_verified_at, $6::timestamptz) ELSE email_verified_at END,
		    last_used_at = $6::timestamptz,
		    last_login_at = $6::timestamptz,
		    updated_at = $6::timestamptz
		WHERE user_id = $1::uuid
		  AND provider = 'google'
		  AND provider_account_id = $7
	`, userID, identity.Email, NormalizeIdentifier(identity.Email), string(profileJSON), identity.EmailVerified, now, identity.Sub)
	if err != nil {
		return fmt.Errorf("touch Google identity: %w", err)
	}
	return nil
}

func loadSessionUserByUserID(ctx context.Context, tx pgx.Tx, userID string) (sessionUser, error) {
	var user sessionUser
	err := tx.QueryRow(ctx, `
		SELECT u.id::text,
		       COALESCE(u.tenant_id::text, ''),
		       COALESCE(t.slug, ''),
		       COALESCE(t.name, ''),
		       u.role,
		       u.name,
		       COALESCE(
		           (SELECT identifier_normalized FROM auth_identities WHERE user_id = u.id AND provider = 'password' ORDER BY created_at ASC LIMIT 1),
		           (SELECT COALESCE(provider_email_normalized, identifier_normalized) FROM auth_identities WHERE user_id = u.id AND provider = 'google' ORDER BY created_at ASC LIMIT 1),
		           ''
		       ) AS email
		FROM users u
		LEFT JOIN tenants t ON t.id = u.tenant_id
		WHERE u.id = $1::uuid
		LIMIT 1
	`, userID).Scan(&user.UserID, &user.TenantID, &user.TenantSlug, &user.TenantName, &user.Role, &user.Name, &user.Email)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return sessionUser{}, ErrInvalidCredentials
		}
		return sessionUser{}, fmt.Errorf("load session user: %w", err)
	}
	return user, nil
}

func sanitizeNextPath(raw, fallback string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return fallback
	}
	parsed, err := url.Parse(raw)
	if err != nil || parsed.IsAbs() || strings.TrimSpace(parsed.Host) != "" || !strings.HasPrefix(parsed.Path, "/") {
		return fallback
	}
	if parsed.Path == "" {
		parsed.Path = "/"
	}
	return parsed.Path + func() string {
		if parsed.RawQuery == "" {
			return ""
		}
		return "?" + parsed.RawQuery
	}()
}

func defaultGoogleNextPath(flowType, requested string) string {
	fallback := "/login"
	if flowType == "link" {
		fallback = "/dashboard"
	}
	return sanitizeNextPath(requested, fallback)
}

func (s *PostgresService) googleLoginRedirectPath() string {
	if s.google == nil {
		return ""
	}
	return s.google.AdminRedirect("/login")
}

func pkceChallenge(verifier string) string {
	sum := sha256.Sum256([]byte(verifier))
	return base64.RawURLEncoding.EncodeToString(sum[:])
}

func GoogleCallbackErrorCode(err error) string {
	switch {
	case errors.Is(err, ErrTenantRequired):
		return "tenant_required"
	case errors.Is(err, ErrIdentityLinkRequired):
		return "link_required"
	case errors.Is(err, ErrIdentityAlreadyLinked):
		return "already_linked"
	case errors.Is(err, ErrAuthFlowInvalid):
		return "flow_invalid"
	case errors.Is(err, ErrGoogleDomainNotAllowed):
		return "domain_not_allowed"
	case errors.Is(err, ErrProviderNotConfigured):
		return "provider_unavailable"
	default:
		return "google_auth_failed"
	}
}
