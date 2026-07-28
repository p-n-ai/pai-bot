// Copyright 2026 the P&AI authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package config

import (
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"os"
)

// WriteNewSecretEnvFile creates a private env fragment containing independent
// auth and runtime-settings encryption roots. It refuses every existing path.
func WriteNewSecretEnvFile(path string) (err error) {
	authSecret, err := randomSecretRoot()
	if err != nil {
		return fmt.Errorf("generate auth secret: %w", err)
	}
	configSecret, err := randomSecretRoot()
	if err != nil {
		return fmt.Errorf("generate config encryption secret: %w", err)
	}

	file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		return fmt.Errorf("create secret env file without overwrite: %w", err)
	}
	defer func() {
		closeErr := file.Close()
		if err != nil {
			_ = os.Remove(path)
			return
		}
		if closeErr != nil {
			err = fmt.Errorf("close secret env file: %w", closeErr)
		}
	}()
	if err := file.Chmod(0o600); err != nil {
		return fmt.Errorf("set secret env file permissions: %w", err)
	}
	content := "PAI_AUTH_SECRET=" + authSecret + "\n" +
		"PAI_CONFIG_ENCRYPTION_KEY=" + configSecret + "\n"
	if _, err := io.WriteString(file, content); err != nil {
		return fmt.Errorf("write secret env file: %w", err)
	}
	if err := file.Sync(); err != nil {
		return fmt.Errorf("sync secret env file: %w", err)
	}
	return nil
}

func randomSecretRoot() (string, error) {
	var raw [32]byte
	if _, err := io.ReadFull(rand.Reader, raw[:]); err != nil {
		return "", err
	}
	encoded := base64.RawURLEncoding.EncodeToString(raw[:])
	if encoded == "" {
		return "", errors.New("empty random secret")
	}
	return encoded, nil
}
