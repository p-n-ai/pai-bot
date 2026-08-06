// Copyright 2026 the P&AI authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package pai_test

import (
	"os/exec"
	"strings"
	"testing"
)

func TestProductionSecretValidation(t *testing.T) {
	helm, err := exec.LookPath("helm")
	if err != nil {
		t.Skip("helm is not installed")
	}

	valid := []string{
		"template", "test", ".",
		"--set-string", "secrets.authSecret=auth-secret-value-with-enough-variety",
		"--set-string", "secrets.configEncryptionKey=active-settings-encryption-key-1234",
		"--set-string", "secrets.bootstrapAdminPassword=private-bootstrap-password",
	}
	if output, err := exec.Command(helm, valid...).CombinedOutput(); err != nil {
		t.Fatalf("helm template valid production secrets: %v\n%s", err, output)
	}

	tests := []struct {
		name    string
		setArgs []string
		want    string
	}{
		{
			name:    "default auth",
			setArgs: []string{"--set-string", "secrets.authSecret=change-me-in-production"},
			want:    "authSecret must be a private value",
		},
		{
			name:    "whitespace active key",
			setArgs: []string{"--set-string", "secrets.configEncryptionKey=                                "},
			want:    "32 non-whitespace characters",
		},
		{
			name:    "weak active key",
			setArgs: []string{"--set-string", "secrets.configEncryptionKey=aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"},
			want:    "high-entropy secret",
		},
		{
			name: "duplicate previous keys",
			setArgs: []string{
				"--set-string", "secrets.previousConfigEncryptionKeys[0]=previous-settings-encryption-key-123",
				"--set-string", "secrets.previousConfigEncryptionKeys[1]=previous-settings-encryption-key-123",
			},
			want: "must not contain duplicates",
		},
		{
			name: "previous key reuses auth",
			setArgs: []string{
				"--set-string", "secrets.previousConfigEncryptionKeys[0]=auth-secret-value-with-enough-variety",
			},
			want: "must differ from active encryption and auth keys",
		},
		{
			name:    "default bootstrap password",
			setArgs: []string{"--set-string", "secrets.bootstrapAdminPassword=demo-password"},
			want:    "bootstrapAdminPassword must be a private value",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			args := append(append([]string(nil), valid...), tt.setArgs...)
			output, err := exec.Command(helm, args...).CombinedOutput()
			if err == nil {
				t.Fatal("helm template succeeded with unsafe production secrets")
			}
			if !strings.Contains(string(output), tt.want) {
				t.Fatalf("helm template output = %q, want %q", output, tt.want)
			}
		})
	}
}

func TestWhatsAppCloudAPIRendering(t *testing.T) {
	helm, err := exec.LookPath("helm")
	if err != nil {
		t.Skip("helm is not installed")
	}

	args := []string{
		"template", "test", ".",
		"--set-string", "secrets.authSecret=auth-secret-value-with-enough-variety",
		"--set-string", "secrets.configEncryptionKey=active-settings-encryption-key-1234",
		"--set-string", "secrets.bootstrapAdminPassword=private-bootstrap-password",
		"--set", "secrets.whatsapp.enabled=true",
		"--set-string", "secrets.whatsapp.accessToken=access-token",
		"--set-string", "secrets.whatsapp.phoneId=phone-id",
		"--set-string", "secrets.whatsapp.verifyToken=verify-token",
		"--set-string", "secrets.whatsapp.appSecret=app-secret",
	}
	output, err := exec.Command(helm, args...).CombinedOutput()
	if err != nil {
		t.Fatalf("helm template WhatsApp Cloud API: %v\n%s", err, output)
	}
	for _, want := range []string{
		`LEARN_WHATSAPP_ENABLED: "true"`,
		`LEARN_WHATSAPP_ACCESS_TOKEN: "access-token"`,
		`LEARN_WHATSAPP_APP_SECRET: "app-secret"`,
	} {
		if !strings.Contains(string(output), want) {
			t.Fatalf("helm template output does not contain %q", want)
		}
	}
}

func TestWhatsAppCloudAPIValidation(t *testing.T) {
	helm, err := exec.LookPath("helm")
	if err != nil {
		t.Skip("helm is not installed")
	}

	base := []string{
		"template", "test", ".",
		"--set-string", "secrets.authSecret=auth-secret-value-with-enough-variety",
		"--set-string", "secrets.configEncryptionKey=active-settings-encryption-key-1234",
		"--set-string", "secrets.bootstrapAdminPassword=private-bootstrap-password",
		"--set", "secrets.whatsapp.enabled=true",
	}
	tests := []struct {
		name string
		args []string
		want string
	}{
		{name: "missing cloud credentials", want: "are required when WhatsApp is enabled"},
		{name: "whitespace app secret", args: []string{
			"--set-string", "secrets.whatsapp.accessToken=access-token",
			"--set-string", "secrets.whatsapp.phoneId=phone-id",
			"--set-string", "secrets.whatsapp.verifyToken=verify-token",
			"--set-string", "secrets.whatsapp.appSecret=   ",
		}, want: "are required when WhatsApp is enabled"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output, err := exec.Command(helm, append(append([]string(nil), base...), tt.args...)...).CombinedOutput()
			if err == nil {
				t.Fatal("helm template succeeded with invalid WhatsApp configuration")
			}
			if !strings.Contains(string(output), tt.want) {
				t.Fatalf("helm template output = %q, want %q", output, tt.want)
			}
		})
	}
}
