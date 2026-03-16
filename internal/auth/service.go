package auth

import (
	"context"
	"errors"
	"time"
)

var (
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrInvalidInvite      = errors.New("invalid invite")
	ErrInviteExpired      = errors.New("invite expired")
	ErrNotImplemented     = errors.New("not implemented")
)

// UserSession contains the authenticated user payload returned by auth flows.
type UserSession struct {
	UserID   string `json:"user_id"`
	TenantID string `json:"tenant_id"`
	Role     Role   `json:"role"`
	Name     string `json:"name"`
	Email    string `json:"email"`
}

// TokenPair contains the access and refresh tokens returned after successful auth.
type TokenPair struct {
	AccessToken      string      `json:"access_token"`
	RefreshToken     string      `json:"refresh_token"`
	AccessExpiresAt  time.Time   `json:"access_expires_at"`
	RefreshExpiresAt time.Time   `json:"refresh_expires_at"`
	User             UserSession `json:"user"`
}

// LoginRequest is the email/password login payload for web users.
type LoginRequest struct {
	TenantID string `json:"tenant_id"`
	Email    string `json:"email"`
	Password string `json:"password"`
}

// AcceptInviteRequest activates an invited account and sets the initial password.
type AcceptInviteRequest struct {
	Token    string `json:"token"`
	Name     string `json:"name"`
	Password string `json:"password"`
}

// Service defines the auth flows needed by the HTTP layer.
type Service interface {
	Login(ctx context.Context, req LoginRequest) (TokenPair, error)
	AcceptInvite(ctx context.Context, req AcceptInviteRequest) (TokenPair, error)
	Refresh(ctx context.Context, refreshToken string) (TokenPair, error)
	Logout(ctx context.Context, refreshToken string) error
}

type noopService struct{}

// NewNoopService returns a placeholder auth service until the DB-backed auth flow lands.
func NewNoopService() Service {
	return noopService{}
}

func (noopService) Login(_ context.Context, _ LoginRequest) (TokenPair, error) {
	return TokenPair{}, ErrNotImplemented
}

func (noopService) AcceptInvite(_ context.Context, _ AcceptInviteRequest) (TokenPair, error) {
	return TokenPair{}, ErrNotImplemented
}

func (noopService) Refresh(_ context.Context, _ string) (TokenPair, error) {
	return TokenPair{}, ErrNotImplemented
}

func (noopService) Logout(_ context.Context, _ string) error {
	return ErrNotImplemented
}
