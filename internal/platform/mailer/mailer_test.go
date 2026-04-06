package mailer

import (
	"context"
	"net/smtp"
	"strings"
	"testing"
	"time"
)

func TestNewSMTPSenderValidatesRequiredFields(t *testing.T) {
	t.Run("missing addr", func(t *testing.T) {
		_, err := NewSMTPSender(SMTPConfig{FromAddress: "bot@example.com"})
		if err == nil {
			t.Fatal("NewSMTPSender() error = nil, want validation error")
		}
	})

	t.Run("missing from address", func(t *testing.T) {
		_, err := NewSMTPSender(SMTPConfig{Addr: "smtp.example.com:587"})
		if err == nil {
			t.Fatal("NewSMTPSender() error = nil, want validation error")
		}
	})
}

func TestSMTPSenderSendInviteBuildsInviteEmail(t *testing.T) {
	sender, err := NewSMTPSender(SMTPConfig{
		Addr:        "smtp.example.com:587",
		Username:    "mailer",
		Password:    "secret",
		FromAddress: "bot@example.com",
		FromName:    "P&AI Bot",
	})
	if err != nil {
		t.Fatalf("NewSMTPSender() error = %v", err)
	}

	var got struct {
		addr string
		from string
		to   []string
		msg  string
	}
	sender.sendMail = func(addr string, _ smtp.Auth, from string, to []string, msg []byte) error {
		got.addr = addr
		got.from = from
		got.to = append([]string(nil), to...)
		got.msg = string(msg)
		return nil
	}

	err = sender.SendInvite(context.Background(), InviteMessage{
		ToEmail:       "teacher@example.com",
		TenantName:    "Pandai School",
		InviterName:   "Admin User",
		RoleLabel:     "teacher",
		ActivationURL: "https://admin.example.com/activate?token=abc123",
		ExpiresAt:     time.Date(2026, 4, 13, 10, 0, 0, 0, time.UTC),
	})
	if err != nil {
		t.Fatalf("SendInvite() error = %v", err)
	}

	if got.addr != "smtp.example.com:587" {
		t.Fatalf("smtp addr = %q, want smtp.example.com:587", got.addr)
	}
	if got.from != "bot@example.com" {
		t.Fatalf("from = %q, want bot@example.com", got.from)
	}
	if len(got.to) != 1 || got.to[0] != "teacher@example.com" {
		t.Fatalf("to = %#v, want teacher@example.com", got.to)
	}
	for _, want := range []string{
		"Subject: Your Pandai School invite",
		"Admin User invited you to join Pandai School as a teacher.",
		"https://admin.example.com/activate?token=abc123",
		"This invite expires on 2026-04-13 10:00 UTC.",
	} {
		if !strings.Contains(got.msg, want) {
			t.Fatalf("email body missing %q\n%s", want, got.msg)
		}
	}
}
