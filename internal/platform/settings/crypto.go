// Copyright 2026 the P&AI authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package settings

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/p-n-ai/pai-bot/internal/platform/config"
)

const (
	credentialEnvelopeAlgorithm = "a256gcm"
	credentialEnvelopePrefix    = "pai:v1:" + credentialEnvelopeAlgorithm + ":"
	maxCredentialPlaintextBytes = 4 * 1024
	maxCredentialEnvelopeBytes  = 8 * 1024
	maxCredentialPayloadBytes   = 6 * 1024
)

// ErrCredentialTooLarge rejects provider credentials that cannot remain
// readable within the bounded persisted envelope format.
var ErrCredentialTooLarge = errors.New("provider credential exceeds the 4 KiB limit")

var (
	errMalformedCredentialEnvelope    = errors.New("malformed credential envelope")
	errUnsupportedCredentialVersion   = errors.New("unsupported credential envelope version")
	errUnsupportedCredentialAlgorithm = errors.New("unsupported credential envelope algorithm")
	errUnknownCredentialKey           = errors.New("unknown credential encryption key")
	errCredentialAuthentication       = errors.New("credential authentication failed")
)

type credentialContext struct {
	Provider string
	Slot     string
}

type credentialDecryption struct {
	Plaintext    string
	KeyID        string
	Legacy       bool
	NeedsRewrite bool
}

func credentialEnvelopeMetadata(encoded string) CredentialEnvelopeStatus {
	if encoded == "" {
		return CredentialEnvelopeStatus{}
	}
	status := CredentialEnvelopeStatus{Stored: true, MigrationNeeded: true}
	if !strings.HasPrefix(encoded, "pai:") {
		status.Version = "legacy"
		return status
	}
	prefix, rest, ok := strings.Cut(encoded, ":")
	if !ok || prefix != "pai" {
		return status
	}
	status.Version, rest, ok = strings.Cut(rest, ":")
	if !ok {
		return status
	}
	status.Algorithm, rest, ok = strings.Cut(rest, ":")
	if !ok {
		return status
	}
	keyID, _, ok := strings.Cut(rest, ":")
	if ok && validCredentialKeyID(keyID) {
		status.KeyID = keyID
	}
	return status
}

// Key derivation: AES-256 key is sha256.Sum256 of the auth secret string.
func gcmFor(secret string) (cipher.AEAD, error) {
	key := sha256.Sum256([]byte(secret))
	block, err := aes.NewCipher(key[:])
	if err != nil {
		return nil, err
	}
	return cipher.NewGCM(block)
}

func credentialKeyID(secret string) string {
	return config.RuntimeSettingsKeyID(secret)
}

func credentialAAD(ctx credentialContext, keyID string) []byte {
	return []byte(
		credentialEnvelopePrefix + keyID + "\x00row=1\x00provider=" + ctx.Provider +
			"\x00slot=" + ctx.Slot,
	)
}

// encryptCredential seals plaintext in the current versioned envelope.
func encryptCredential(secret, plaintext string, ctx credentialContext) (string, error) {
	if len(plaintext) > maxCredentialPlaintextBytes {
		return "", ErrCredentialTooLarge
	}
	gcm, err := gcmFor(secret)
	if err != nil {
		return "", err
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", err
	}
	keyID := credentialKeyID(secret)
	sealed := gcm.Seal(nil, nonce, []byte(plaintext), credentialAAD(ctx, keyID))
	payload := append(nonce, sealed...)
	return credentialEnvelopePrefix + keyID + ":" + base64.RawURLEncoding.EncodeToString(payload), nil
}

// decryptCredential opens a versioned envelope by exact key ID. Values without
// the pai: prefix are treated as the only supported legacy format.
func decryptCredential(
	activeSecret string,
	previousSecrets []string,
	legacySecrets []string,
	encoded string,
	ctx credentialContext,
) (credentialDecryption, error) {
	if strings.HasPrefix(encoded, "pai:") {
		return decryptVersionedCredential(activeSecret, previousSecrets, encoded, ctx)
	}
	legacyKeys := appendUniqueSecrets([]string{activeSecret}, previousSecrets...)
	legacyKeys = appendUniqueSecrets(legacyKeys, legacySecrets...)
	for _, secret := range legacyKeys {
		if secret == "" {
			continue
		}
		plaintext, err := decryptString(secret, encoded)
		if err == nil {
			return credentialDecryption{
				Plaintext:    plaintext,
				KeyID:        credentialKeyID(secret),
				Legacy:       true,
				NeedsRewrite: activeSecret != "",
			}, nil
		}
	}
	return credentialDecryption{}, errCredentialAuthentication
}

func decryptVersionedCredential(
	activeSecret string,
	previousSecrets []string,
	encoded string,
	ctx credentialContext,
) (credentialDecryption, error) {
	if len(encoded) > maxCredentialEnvelopeBytes {
		return credentialDecryption{}, errMalformedCredentialEnvelope
	}
	prefix, rest, ok := strings.Cut(encoded, ":")
	if !ok || prefix != "pai" {
		return credentialDecryption{}, errMalformedCredentialEnvelope
	}
	version, rest, ok := strings.Cut(rest, ":")
	if !ok {
		return credentialDecryption{}, errMalformedCredentialEnvelope
	}
	if version != "v1" {
		return credentialDecryption{}, errUnsupportedCredentialVersion
	}
	algorithm, rest, ok := strings.Cut(rest, ":")
	if !ok {
		return credentialDecryption{}, errMalformedCredentialEnvelope
	}
	if algorithm != credentialEnvelopeAlgorithm {
		return credentialDecryption{}, errUnsupportedCredentialAlgorithm
	}
	keyID, payload, ok := strings.Cut(rest, ":")
	if !ok || strings.Contains(payload, ":") || !validCredentialKeyID(keyID) {
		return credentialDecryption{}, errMalformedCredentialEnvelope
	}
	secrets := appendUniqueSecrets([]string{activeSecret}, previousSecrets...)
	var secret string
	for _, candidate := range secrets {
		if candidate != "" && credentialKeyID(candidate) == keyID {
			secret = candidate
			break
		}
	}
	if secret == "" {
		return credentialDecryption{}, errUnknownCredentialKey
	}
	raw, err := base64.RawURLEncoding.DecodeString(payload)
	if err != nil {
		return credentialDecryption{}, errMalformedCredentialEnvelope
	}
	if len(raw) > maxCredentialPayloadBytes {
		return credentialDecryption{}, errMalformedCredentialEnvelope
	}
	gcm, err := gcmFor(secret)
	if err != nil {
		return credentialDecryption{}, errCredentialAuthentication
	}
	if len(raw) < gcm.NonceSize()+gcm.Overhead() {
		return credentialDecryption{}, errMalformedCredentialEnvelope
	}
	plaintext, err := gcm.Open(
		nil,
		raw[:gcm.NonceSize()],
		raw[gcm.NonceSize():],
		credentialAAD(ctx, keyID),
	)
	if err != nil {
		return credentialDecryption{}, errCredentialAuthentication
	}
	activeKeyID := ""
	if activeSecret != "" {
		activeKeyID = credentialKeyID(activeSecret)
	}
	return credentialDecryption{
		Plaintext:    string(plaintext),
		KeyID:        keyID,
		NeedsRewrite: activeKeyID != "" && keyID != activeKeyID,
	}, nil
}

func validCredentialKeyID(keyID string) bool {
	if len(keyID) != 22 {
		return false
	}
	for _, r := range keyID {
		if (r >= 'a' && r <= 'z') ||
			(r >= 'A' && r <= 'Z') ||
			(r >= '0' && r <= '9') ||
			r == '.' || r == '_' || r == '-' {
			continue
		}
		return false
	}
	return true
}

func appendUniqueSecrets(dst []string, values ...string) []string {
	for _, value := range values {
		if value == "" || slicesContains(dst, value) {
			continue
		}
		dst = append(dst, value)
	}
	return dst
}

func slicesContains(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}

// encryptString seals plaintext with AES-256-GCM and a random nonce,
// returning base64(nonce||ciphertext).
func encryptString(secret, plaintext string) (string, error) {
	gcm, err := gcmFor(secret)
	if err != nil {
		return "", err
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", err
	}
	sealed := gcm.Seal(nonce, nonce, []byte(plaintext), nil)
	return base64.StdEncoding.EncodeToString(sealed), nil
}

func decryptString(secret, encoded string) (string, error) {
	gcm, err := gcmFor(secret)
	if err != nil {
		return "", err
	}
	raw, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return "", fmt.Errorf("decode secret blob: %w", err)
	}
	if len(raw) < gcm.NonceSize() {
		return "", fmt.Errorf("secret blob too short")
	}
	plaintext, err := gcm.Open(nil, raw[:gcm.NonceSize()], raw[gcm.NonceSize():], nil)
	if err != nil {
		return "", fmt.Errorf("decrypt secret: %w", err)
	}
	return string(plaintext), nil
}
