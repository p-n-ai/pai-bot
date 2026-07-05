// Copyright 2026 the P&AI authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package settings

import (
	"encoding/base64"
	"testing"
)

func TestCryptoRoundtrip(t *testing.T) {
	const secret = "auth-secret"
	const plaintext = "sk-or-v1-abcdef123456"

	encoded, err := encryptString(secret, plaintext)
	if err != nil {
		t.Fatalf("encryptString() error = %v", err)
	}
	if encoded == plaintext {
		t.Fatal("ciphertext should not equal plaintext")
	}

	got, err := decryptString(secret, encoded)
	if err != nil {
		t.Fatalf("decryptString() error = %v", err)
	}
	if got != plaintext {
		t.Fatalf("decryptString() = %q, want %q", got, plaintext)
	}
}

func TestCryptoTamperDetected(t *testing.T) {
	const secret = "auth-secret"

	encoded, err := encryptString(secret, "payload")
	if err != nil {
		t.Fatalf("encryptString() error = %v", err)
	}
	raw, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		t.Fatalf("decode blob: %v", err)
	}
	raw[len(raw)-1] ^= 0xff
	tampered := base64.StdEncoding.EncodeToString(raw)

	if _, err := decryptString(secret, tampered); err == nil {
		t.Fatal("decryptString() should reject tampered ciphertext")
	}
}

func TestCryptoWrongSecret(t *testing.T) {
	encoded, err := encryptString("right-secret", "payload")
	if err != nil {
		t.Fatalf("encryptString() error = %v", err)
	}
	if _, err := decryptString("wrong-secret", encoded); err == nil {
		t.Fatal("decryptString() should fail with the wrong secret")
	}
}

func TestCryptoBadBlob(t *testing.T) {
	if _, err := decryptString("secret", "not-base64!!"); err == nil {
		t.Fatal("decryptString() should reject invalid base64")
	}
	if _, err := decryptString("secret", base64.StdEncoding.EncodeToString([]byte("tiny"))); err == nil {
		t.Fatal("decryptString() should reject blobs shorter than the nonce")
	}
}
