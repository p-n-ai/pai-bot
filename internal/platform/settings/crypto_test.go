// Copyright 2026 the P&AI authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package settings

import (
	"encoding/base64"
	"errors"
	"strings"
	"testing"
)

func TestCredentialEnvelopeRoundTrip(t *testing.T) {
	const secret = "credential-envelope-test-secret-12345"
	ctx := credentialContext{Provider: "openrouter", Slot: "api_key"}

	first, err := encryptCredential(secret, "sk-or-secret", ctx)
	if err != nil {
		t.Fatalf("encryptCredential() error = %v", err)
	}
	second, err := encryptCredential(secret, "sk-or-secret", ctx)
	if err != nil {
		t.Fatalf("encryptCredential(second) error = %v", err)
	}
	if first == second {
		t.Fatal("envelopes must use independent random nonces")
	}
	if !strings.HasPrefix(first, credentialEnvelopePrefix+credentialKeyID(secret)+":") {
		t.Fatalf("envelope = %q, want version and derived key ID", first)
	}
	if strings.Contains(first, "sk-or-secret") {
		t.Fatal("envelope contains plaintext")
	}

	got, err := decryptCredential(secret, nil, nil, first, ctx)
	if err != nil {
		t.Fatalf("decryptCredential() error = %v", err)
	}
	if got.Plaintext != "sk-or-secret" || got.Legacy || got.NeedsRewrite {
		t.Fatalf("decryptCredential() = %+v, want current v1 plaintext", got)
	}
}

func TestCredentialEnvelopeBindsContextAndKeyID(t *testing.T) {
	const secret = "credential-envelope-test-secret-12345"
	ctx := credentialContext{Provider: "openrouter", Slot: "api_key"}
	encoded, err := encryptCredential(secret, "sk-or-secret", ctx)
	if err != nil {
		t.Fatalf("encryptCredential() error = %v", err)
	}

	for _, wrong := range []credentialContext{
		{Provider: "openai", Slot: "api_key"},
		{Provider: "openrouter", Slot: "refresh_token"},
	} {
		if _, err := decryptCredential(secret, nil, nil, encoded, wrong); !errors.Is(err, errCredentialAuthentication) {
			t.Fatalf("decryptCredential(wrong context %+v) error = %v, want authentication failure", wrong, err)
		}
	}

	parts := strings.Split(encoded, ":")
	parts[3] = credentialKeyID("different-credential-key-secret-123")
	tamperedKeyID := strings.Join(parts, ":")
	if _, err := decryptCredential(secret, nil, nil, tamperedKeyID, ctx); !errors.Is(err, errUnknownCredentialKey) {
		t.Fatalf("decryptCredential(tampered key ID) error = %v, want unknown key", err)
	}
}

func TestCredentialEnvelopeUsesRetiredKeyAndRequestsRewrite(t *testing.T) {
	const (
		active  = "active-credential-key-secret-123456"
		retired = "retired-credential-key-secret-1234"
	)
	ctx := credentialContext{Provider: "openrouter", Slot: "api_key"}
	encoded, err := encryptCredential(retired, "sk-or-retired", ctx)
	if err != nil {
		t.Fatalf("encryptCredential() error = %v", err)
	}

	got, err := decryptCredential(active, []string{retired}, nil, encoded, ctx)
	if err != nil {
		t.Fatalf("decryptCredential() error = %v", err)
	}
	if got.Plaintext != "sk-or-retired" || got.Legacy || !got.NeedsRewrite {
		t.Fatalf("decryptCredential() = %+v, want retired-key rewrite", got)
	}
}

func TestCredentialEnvelopeLegacyMigration(t *testing.T) {
	const (
		active = "active-credential-key-secret-123456"
		legacy = "legacy-auth-secret"
		// PR #224 format: base64(nonce || AES-GCM ciphertext || tag), with
		// SHA-256(legacy secret) as the AES key and no AAD.
		golden = "AAECAwQFBgcICQoLFTeOW4MVf/El53pxeKG9hlD6CmpEzj9uASAWLg=="
	)
	ctx := credentialContext{Provider: "openrouter", Slot: "api_key"}

	got, err := decryptCredential(active, nil, []string{legacy}, golden, ctx)
	if err != nil {
		t.Fatalf("decryptCredential() error = %v", err)
	}
	if got.Plaintext != "sk-or-legacy" || !got.Legacy || !got.NeedsRewrite {
		t.Fatalf("decryptCredential() = %+v, want legacy rewrite", got)
	}
}

func TestCredentialEnvelopeRejectsMalformedAndUnknownWithoutLegacyFallback(t *testing.T) {
	const secret = "credential-envelope-test-secret-12345"
	ctx := credentialContext{Provider: "openrouter", Slot: "api_key"}
	tests := []struct {
		name string
		blob string
		err  error
	}{
		{name: "unsupported version", blob: "pai:v2:a256gcm:kid:payload", err: errUnsupportedCredentialVersion},
		{name: "unsupported algorithm", blob: "pai:v1:a128gcm:AAAAAAAAAAAAAAAAAAAAAA:payload", err: errUnsupportedCredentialAlgorithm},
		{name: "unreleased four field shape", blob: "pai:v1:AAAAAAAAAAAAAAAAAAAAAA:payload", err: errUnsupportedCredentialAlgorithm},
		{name: "missing fields", blob: "pai:v1:a256gcm:kid", err: errMalformedCredentialEnvelope},
		{name: "invalid key id", blob: "pai:v1:a256gcm:bad kid:payload", err: errMalformedCredentialEnvelope},
		{name: "unknown key", blob: "pai:v1:a256gcm:AAAAAAAAAAAAAAAAAAAAAA:payload", err: errUnknownCredentialKey},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := decryptCredential(secret, nil, []string{secret}, tt.blob, ctx)
			if !errors.Is(err, tt.err) {
				t.Fatalf("decryptCredential() error = %v, want %v", err, tt.err)
			}
		})
	}
}

func TestCredentialEnvelopeRejectsCiphertextBitFlip(t *testing.T) {
	const secret = "credential-envelope-test-secret-12345"
	ctx := credentialContext{Provider: "openrouter", Slot: "api_key"}
	encoded, err := encryptCredential(secret, "sk-or-secret", ctx)
	if err != nil {
		t.Fatalf("encryptCredential() error = %v", err)
	}
	header, payload, ok := strings.Cut(encoded, credentialEnvelopePrefix+credentialKeyID(secret)+":")
	if !ok || header != "" {
		t.Fatal("encrypted credential has unexpected envelope header")
	}
	raw, err := base64.RawURLEncoding.DecodeString(payload)
	if err != nil {
		t.Fatalf("decode payload: %v", err)
	}
	raw[len(raw)-1] ^= 1
	tampered := credentialEnvelopePrefix + credentialKeyID(secret) + ":" +
		base64.RawURLEncoding.EncodeToString(raw)

	if _, err := decryptCredential(secret, nil, nil, tampered, ctx); !errors.Is(err, errCredentialAuthentication) {
		t.Fatalf("decryptCredential(bit flip) error = %v, want authentication failure", err)
	}
}
