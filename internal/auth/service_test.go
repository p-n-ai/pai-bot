package auth

import (
	"context"
	"errors"
	"testing"
)

func TestTenantRequiredErrorWrapsSentinel(t *testing.T) {
	err := NewTenantRequiredError([]TenantOption{{TenantID: "tenant-a", TenantSlug: "school-a", TenantName: "School A"}})

	if !errors.Is(err, ErrTenantRequired) {
		t.Fatalf("errors.Is(err, ErrTenantRequired) = false")
	}

	terr, ok := TenantRequiredOptions(err)
	if !ok {
		t.Fatal("TenantRequiredOptions() = false, want true")
	}
	if len(terr) != 1 || terr[0].TenantSlug != "school-a" {
		t.Fatalf("tenant options = %#v", terr)
	}
}

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

	_, err = svc.SwitchTenant(context.Background(), "", "", "")
	if !errors.Is(err, ErrNotImplemented) {
		t.Fatalf("SwitchTenant() error = %v, want ErrNotImplemented", err)
	}

	_, err = svc.IssueInvite(context.Background(), IssueInviteRequest{})
	if !errors.Is(err, ErrNotImplemented) {
		t.Fatalf("IssueInvite() error = %v, want ErrNotImplemented", err)
	}

	err = svc.Logout(context.Background(), "")
	if !errors.Is(err, ErrNotImplemented) {
		t.Fatalf("Logout() error = %v, want ErrNotImplemented", err)
	}

	_, err = svc.StartGoogleLogin(context.Background(), StartGoogleFlowRequest{})
	if !errors.Is(err, ErrNotImplemented) {
		t.Fatalf("StartGoogleLogin() error = %v, want ErrNotImplemented", err)
	}

	_, err = svc.StartGoogleLink(context.Background(), StartGoogleFlowRequest{})
	if !errors.Is(err, ErrNotImplemented) {
		t.Fatalf("StartGoogleLink() error = %v, want ErrNotImplemented", err)
	}

	_, err = svc.CompleteGoogleCallback(context.Background(), GoogleCallbackRequest{})
	if !errors.Is(err, ErrNotImplemented) {
		t.Fatalf("CompleteGoogleCallback() error = %v, want ErrNotImplemented", err)
	}

	_, err = svc.ListLinkedIdentities(context.Background(), "user-1")
	if !errors.Is(err, ErrNotImplemented) {
		t.Fatalf("ListLinkedIdentities() error = %v, want ErrNotImplemented", err)
	}
}
