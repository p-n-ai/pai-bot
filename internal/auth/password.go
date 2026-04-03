package auth

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"strings"

	"golang.org/x/crypto/bcrypt"
)

var ErrEmptyPassword = errors.New("password is required")

// NormalizeIdentifier trims whitespace and lowercases login identifiers such as email.
func NormalizeIdentifier(identifier string) string {
	return strings.ToLower(strings.TrimSpace(identifier))
}

// HashPassword hashes a plaintext password with bcrypt.
func HashPassword(password string) (string, error) {
	if strings.TrimSpace(password) == "" {
		return "", ErrEmptyPassword
	}

	hashed, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(hashed), nil
}

// ComparePassword verifies a plaintext password against a bcrypt hash.
func ComparePassword(hash, password string) error {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
}

// HashOpaqueToken hashes invite and session tokens before they are persisted.
func HashOpaqueToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}
