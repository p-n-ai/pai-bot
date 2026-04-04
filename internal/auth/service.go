package auth

import (
	"context"
	"errors"
	"time"
)

var (
	ErrInvalidCredentials     = errors.New("invalid credentials")
	ErrInvalidInvite          = errors.New("invalid invite")
	ErrInviteExpired          = errors.New("invite expired")
	ErrNotImplemented         = errors.New("not implemented")
	ErrInviteConflict         = errors.New("invite already exists")
	ErrTenantRequired         = errors.New("tenant is required for this account")
	ErrProviderNotConfigured  = errors.New("auth provider is not configured")
	ErrIdentityAlreadyLinked  = errors.New("identity is already linked to another account")
	ErrIdentityLinkRequired   = errors.New("sign in with email once, then link Google")
	ErrAuthFlowInvalid        = errors.New("auth flow is invalid or expired")
	ErrGoogleDomainNotAllowed = errors.New("google account domain is not allowed")
)

type TenantOption struct {
	TenantID   string `json:"tenant_id"`
	TenantSlug string `json:"tenant_slug"`
	TenantName string `json:"tenant_name"`
}

type tenantRequiredError struct {
	Options []TenantOption
}

func (e tenantRequiredError) Error() string {
	return ErrTenantRequired.Error()
}

func (e tenantRequiredError) Unwrap() error {
	return ErrTenantRequired
}

func NewTenantRequiredError(options []TenantOption) error {
	return tenantRequiredError{Options: options}
}

func TenantRequiredOptions(err error) ([]TenantOption, bool) {
	var target tenantRequiredError
	if errors.As(err, &target) {
		return target.Options, true
	}
	return nil, false
}

// UserSession contains the authenticated user payload returned by auth flows.
type UserSession struct {
	UserID     string `json:"user_id"`
	TenantID   string `json:"tenant_id"`
	TenantSlug string `json:"tenant_slug"`
	TenantName string `json:"tenant_name"`
	Role       Role   `json:"role"`
	Name       string `json:"name"`
	Email      string `json:"email"`
}

// Session is the Better-Auth-style browser session contract used by the admin UI.
type Session struct {
	Token         string         `json:"-"`
	ExpiresAt     time.Time      `json:"expires_at"`
	User          UserSession    `json:"user"`
	TenantChoices []TenantOption `json:"tenant_choices,omitempty"`
}

// LoginRequest is the email/password login payload for web users.
// TenantID is optional and is only needed when the same email exists in multiple tenants.
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

// IssueInviteRequest creates a new invite for a teacher, parent, or admin user.
type IssueInviteRequest struct {
	InvitedByUserID string `json:"invited_by_user_id"`
	TenantID        string `json:"tenant_id"`
	Email           string `json:"email"`
	Role            Role   `json:"role"`
}

// InviteRecord is returned when an invite is created.
type InviteRecord struct {
	Email       string    `json:"email"`
	Role        Role      `json:"role"`
	Token       string    `json:"invite_token"`
	ExpiresAt   time.Time `json:"expires_at"`
	InvitedByID string    `json:"invited_by_user_id"`
}

type LinkedIdentity struct {
	Provider   string     `json:"provider"`
	Email      string     `json:"email"`
	LinkedAt   *time.Time `json:"linked_at,omitempty"`
	LastUsedAt *time.Time `json:"last_used_at,omitempty"`
}

type StartGoogleFlowRequest struct {
	UserID      string
	NextPath    string
	RedirectURL string
}

type GoogleCallbackRequest struct {
	State       string
	Code        string
	RedirectURL string
}

type GoogleCallbackResult struct {
	RedirectPath string
	Linked       bool
	Session      *Session
}

// Service defines the auth flows needed by the HTTP layer.
type Service interface {
	Login(ctx context.Context, req LoginRequest) (Session, error)
	AcceptInvite(ctx context.Context, req AcceptInviteRequest) (Session, error)
	IssueInvite(ctx context.Context, req IssueInviteRequest) (InviteRecord, error)
	Session(ctx context.Context, sessionToken string) (Session, error)
	SwitchTenant(ctx context.Context, sessionToken, tenantID, password string) (Session, error)
	Logout(ctx context.Context, sessionToken string) error
	StartGoogleLogin(ctx context.Context, req StartGoogleFlowRequest) (string, error)
	StartGoogleLink(ctx context.Context, req StartGoogleFlowRequest) (string, error)
	CompleteGoogleCallback(ctx context.Context, req GoogleCallbackRequest) (GoogleCallbackResult, error)
	ListLinkedIdentities(ctx context.Context, userID string) ([]LinkedIdentity, error)
}

type noopService struct{}

// NewNoopService returns a placeholder auth service until the DB-backed auth flow lands.
func NewNoopService() Service {
	return noopService{}
}

func (noopService) Login(_ context.Context, _ LoginRequest) (Session, error) {
	return Session{}, ErrNotImplemented
}

func (noopService) AcceptInvite(_ context.Context, _ AcceptInviteRequest) (Session, error) {
	return Session{}, ErrNotImplemented
}

func (noopService) IssueInvite(_ context.Context, _ IssueInviteRequest) (InviteRecord, error) {
	return InviteRecord{}, ErrNotImplemented
}

func (noopService) Session(_ context.Context, _ string) (Session, error) {
	return Session{}, ErrNotImplemented
}

func (noopService) SwitchTenant(_ context.Context, _, _, _ string) (Session, error) {
	return Session{}, ErrNotImplemented
}

func (noopService) Logout(_ context.Context, _ string) error {
	return ErrNotImplemented
}

func (noopService) StartGoogleLogin(_ context.Context, _ StartGoogleFlowRequest) (string, error) {
	return "", ErrNotImplemented
}

func (noopService) StartGoogleLink(_ context.Context, _ StartGoogleFlowRequest) (string, error) {
	return "", ErrNotImplemented
}

func (noopService) CompleteGoogleCallback(_ context.Context, _ GoogleCallbackRequest) (GoogleCallbackResult, error) {
	return GoogleCallbackResult{}, ErrNotImplemented
}

func (noopService) ListLinkedIdentities(_ context.Context, _ string) ([]LinkedIdentity, error) {
	return nil, ErrNotImplemented
}
