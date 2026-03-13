package auth

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"
)

const defaultTokenTTL = time.Hour

var (
	ErrInvalidToken = errors.New("invalid token")
	ErrExpiredToken = errors.New("expired token")
)

type Role string

const (
	RoleStudent       Role = "student"
	RoleTeacher       Role = "teacher"
	RoleParent        Role = "parent"
	RoleAdmin         Role = "admin"
	RolePlatformAdmin Role = "platform_admin"
)

type TokenClaims struct {
	Subject   string    `json:"sub"`
	TenantID  string    `json:"tenant_id"`
	Role      Role      `json:"role"`
	IssuedAt  time.Time `json:"-"`
	ExpiresAt time.Time `json:"-"`
}

type tokenPayload struct {
	Subject   string `json:"sub"`
	TenantID  string `json:"tenant_id"`
	Role      Role   `json:"role"`
	IssuedAt  int64  `json:"iat"`
	ExpiresAt int64  `json:"exp"`
}

type TokenManager struct {
	secret []byte
	ttl    time.Duration
}

func NewTokenManager(secret string, ttl time.Duration) *TokenManager {
	if ttl <= 0 {
		ttl = defaultTokenTTL
	}
	return &TokenManager{
		secret: []byte(secret),
		ttl:    ttl,
	}
}

func (m *TokenManager) Issue(claims TokenClaims, now time.Time) (string, error) {
	if strings.TrimSpace(claims.Subject) == "" || strings.TrimSpace(claims.TenantID) == "" || claims.Role == "" {
		return "", fmt.Errorf("issue token: %w", ErrInvalidToken)
	}
	if len(m.secret) == 0 {
		return "", fmt.Errorf("issue token: %w", ErrInvalidToken)
	}

	headerJSON, err := json.Marshal(map[string]string{
		"alg": "HS256",
		"typ": "JWT",
	})
	if err != nil {
		return "", fmt.Errorf("marshal header: %w", err)
	}

	payloadJSON, err := json.Marshal(tokenPayload{
		Subject:   claims.Subject,
		TenantID:  claims.TenantID,
		Role:      claims.Role,
		IssuedAt:  now.Unix(),
		ExpiresAt: now.Add(m.ttl).Unix(),
	})
	if err != nil {
		return "", fmt.Errorf("marshal payload: %w", err)
	}

	unsigned := encodeSegment(headerJSON) + "." + encodeSegment(payloadJSON)
	signature := signHS256(unsigned, m.secret)
	return unsigned + "." + encodeSegment(signature), nil
}

func (m *TokenManager) Parse(token string, now time.Time) (TokenClaims, error) {
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return TokenClaims{}, ErrInvalidToken
	}

	headerJSON, err := decodeSegment(parts[0])
	if err != nil {
		return TokenClaims{}, ErrInvalidToken
	}

	var header struct {
		Algorithm string `json:"alg"`
		Type      string `json:"typ"`
	}
	if err := json.Unmarshal(headerJSON, &header); err != nil {
		return TokenClaims{}, ErrInvalidToken
	}
	if header.Algorithm != "HS256" || (header.Type != "" && header.Type != "JWT") {
		return TokenClaims{}, ErrInvalidToken
	}

	payloadJSON, err := decodeSegment(parts[1])
	if err != nil {
		return TokenClaims{}, ErrInvalidToken
	}

	signature, err := decodeSegment(parts[2])
	if err != nil {
		return TokenClaims{}, ErrInvalidToken
	}

	unsigned := parts[0] + "." + parts[1]
	if !hmac.Equal(signature, signHS256(unsigned, m.secret)) {
		return TokenClaims{}, ErrInvalidToken
	}

	var payload tokenPayload
	if err := json.Unmarshal(payloadJSON, &payload); err != nil {
		return TokenClaims{}, ErrInvalidToken
	}
	if payload.Subject == "" || payload.TenantID == "" || payload.Role == "" {
		return TokenClaims{}, ErrInvalidToken
	}
	if payload.ExpiresAt <= now.Unix() {
		return TokenClaims{}, ErrExpiredToken
	}

	return TokenClaims{
		Subject:   payload.Subject,
		TenantID:  payload.TenantID,
		Role:      payload.Role,
		IssuedAt:  time.Unix(payload.IssuedAt, 0).UTC(),
		ExpiresAt: time.Unix(payload.ExpiresAt, 0).UTC(),
	}, nil
}

func encodeSegment(src []byte) string {
	return base64.RawURLEncoding.EncodeToString(src)
}

func decodeSegment(src string) ([]byte, error) {
	return base64.RawURLEncoding.DecodeString(src)
}

func signHS256(unsigned string, secret []byte) []byte {
	mac := hmac.New(sha256.New, secret)
	_, _ = mac.Write([]byte(unsigned))
	return mac.Sum(nil)
}
