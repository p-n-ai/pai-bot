package auth

import (
	"context"
	"errors"
	"testing"
)

func TestNoopServiceReturnsNotImplemented(t *testing.T) {
	svc := NewNoopService()

	_, err := svc.Login(context.Background(), LoginRequest{})
	if !errors.Is(err, ErrNotImplemented) {
		t.Fatalf("Login() error = %v, want ErrNotImplemented", err)
	}

	_, err = svc.AcceptInvite(context.Background(), AcceptInviteRequest{})
	if !errors.Is(err, ErrNotImplemented) {
		t.Fatalf("AcceptInvite() error = %v, want ErrNotImplemented", err)
	}

	_, err = svc.Refresh(context.Background(), "")
	if !errors.Is(err, ErrNotImplemented) {
		t.Fatalf("Refresh() error = %v, want ErrNotImplemented", err)
	}

	err = svc.Logout(context.Background(), "")
	if !errors.Is(err, ErrNotImplemented) {
		t.Fatalf("Logout() error = %v, want ErrNotImplemented", err)
	}
}
