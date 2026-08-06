// Copyright 2026 the P&AI authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package auth

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"strings"

	"golang.org/x/crypto/bcrypt"
)

const MinimumPasswordLength = 12

var (
	ErrEmptyPassword = errors.New("password is required")
	ErrWeakPassword  = errors.New("password must be at least 12 characters")
)

// NormalizeIdentifier trims whitespace and lowercases login identifiers such as email.
func NormalizeIdentifier(identifier string) string {
	return strings.ToLower(strings.TrimSpace(identifier))
}

// HashPassword hashes a plaintext password with bcrypt.
func HashPassword(password string) (string, error) {
	if err := ValidatePassword(password); err != nil {
		return "", err
	}

	hashed, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(hashed), nil
}

// ValidatePassword enforces the durable password-account policy.
func ValidatePassword(password string) error {
	if strings.TrimSpace(password) == "" {
		return ErrEmptyPassword
	}
	if len([]rune(password)) < MinimumPasswordLength {
		return ErrWeakPassword
	}
	return nil
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
