// Copyright 2026 the P&AI authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

// Package focusedpage owns private, temporary focused-message pages.
package focusedpage

import (
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"net/url"
	"strings"
	"time"
)

const (
	Lifetime         = time.Hour
	MaxMessageLength = 4000
	PageIndex        = 0
)

var (
	ErrNotFound  = errors.New("focused page not found")
	ErrForbidden = errors.New("focused page forbidden")
	ErrExpired   = errors.New("focused page expired")
	ErrRevoked   = errors.New("focused page revoked")
)

type Status string

const (
	StatusActive  Status = "active"
	StatusRevoked Status = "revoked"
	StatusExpired Status = "expired"
)

type Page struct {
	PublicID       string
	TenantID       string
	OwnerUserID    string
	ConversationID string
	TurnID         string
	RecipientName  string
	Message        string
	TokenHash      []byte
	Status         Status
	CreatedAt      time.Time
	ExpiresAt      time.Time
	RevokedAt      *time.Time
}

type CreateRecord struct {
	PublicID       string
	TenantID       string
	OwnerUserID    string
	ConversationID string
	TurnID         string
	PageIndex      int
	RecipientName  string
	Message        string
	TokenHash      []byte
	CreatedAt      time.Time
	ExpiresAt      time.Time
}

type Store interface {
	CreateOrGet(context.Context, CreateRecord) (Page, error)
	Redeem(context.Context, string, []byte, time.Time) (Page, error)
	Revoke(context.Context, string, string, string, time.Time) error
}

type CreateInput struct {
	TenantID       string
	OwnerUserID    string
	ConversationID string
	TurnID         string
	RecipientName  string
	Message        string
}

type Artifact struct {
	PublicID  string
	URL       string
	ExpiresAt time.Time
}

type Service struct {
	store   Store
	baseURL *url.URL
	secret  []byte
	now     func() time.Time
}

func NewService(store Store, baseURL string, secret []byte, now func() time.Time) (*Service, error) {
	if store == nil {
		return nil, fmt.Errorf("focused page store is required")
	}
	parsed, err := url.Parse(strings.TrimRight(strings.TrimSpace(baseURL), "/"))
	if err != nil || parsed.Scheme != "https" || parsed.Host == "" || parsed.User != nil || parsed.Path != "" || parsed.RawQuery != "" || parsed.Fragment != "" {
		return nil, fmt.Errorf("focused page base URL must be an HTTPS origin without credentials, query, or fragment")
	}
	if len(secret) < 16 {
		return nil, fmt.Errorf("focused page secret must be at least 16 bytes")
	}
	if now == nil {
		now = time.Now
	}
	return &Service{store: store, baseURL: parsed, secret: append([]byte(nil), secret...), now: now}, nil
}

func ParseMessage(raw string) (string, error) {
	message := strings.TrimSpace(raw)
	if message == "" {
		return "", fmt.Errorf("message is required")
	}
	if len([]rune(message)) > MaxMessageLength {
		return "", fmt.Errorf("message exceeds %d characters", MaxMessageLength)
	}
	return message, nil
}

func (s *Service) Create(ctx context.Context, input CreateInput) (Artifact, error) {
	if input.TenantID == "" || input.OwnerUserID == "" || input.ConversationID == "" || input.TurnID == "" {
		return Artifact{}, fmt.Errorf("trusted focused page identity is incomplete")
	}
	message, err := ParseMessage(input.Message)
	if err != nil {
		return Artifact{}, err
	}
	recipient := strings.TrimSpace(input.RecipientName)
	if recipient == "" {
		recipient = "you"
	}
	createdAt := s.now().UTC()
	token := s.capability(input.TenantID, input.TurnID, PageIndex)
	page, err := s.store.CreateOrGet(ctx, CreateRecord{
		PublicID:       randomPublicID(),
		TenantID:       input.TenantID,
		OwnerUserID:    input.OwnerUserID,
		ConversationID: input.ConversationID,
		TurnID:         input.TurnID,
		PageIndex:      PageIndex,
		RecipientName:  recipient,
		Message:        message,
		TokenHash:      hashToken(token),
		CreatedAt:      createdAt,
		ExpiresAt:      createdAt.Add(Lifetime),
	})
	if err != nil {
		return Artifact{}, err
	}
	link := *s.baseURL
	link.Path = strings.TrimRight(link.Path, "/") + "/a/" + url.PathEscape(page.PublicID)
	link.Fragment = token
	return Artifact{PublicID: page.PublicID, URL: link.String(), ExpiresAt: page.ExpiresAt}, nil
}

func (s *Service) Redeem(ctx context.Context, publicID, token string) (Page, error) {
	if strings.TrimSpace(token) == "" {
		return Page{}, ErrForbidden
	}
	return s.store.Redeem(ctx, publicID, hashToken(token), s.now().UTC())
}

func (s *Service) Revoke(ctx context.Context, publicID, tenantID, ownerUserID string) error {
	return s.store.Revoke(ctx, publicID, tenantID, ownerUserID, s.now().UTC())
}

func (s *Service) capability(tenantID, turnID string, pageIndex int) string {
	mac := hmac.New(sha256.New, s.secret)
	_, _ = fmt.Fprintf(mac, "focused-page:v1\x00%s\x00%s\x00%d", tenantID, turnID, pageIndex)
	return base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
}

func hashToken(token string) []byte {
	sum := sha256.Sum256([]byte(token))
	return sum[:]
}

func randomPublicID() string {
	raw := make([]byte, 18)
	if _, err := rand.Read(raw); err != nil {
		panic(fmt.Sprintf("generate focused page public id: %v", err))
	}
	return base64.RawURLEncoding.EncodeToString(raw)
}
