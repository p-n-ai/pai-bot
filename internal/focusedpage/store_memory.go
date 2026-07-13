// Copyright 2026 the P&AI authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package focusedpage

import (
	"context"
	"crypto/subtle"
	"strconv"
	"sync"
	"time"
)

type MemoryStore struct {
	mu    sync.Mutex
	pages map[string]Page
	keys  map[string]string
}

func NewMemoryStore() *MemoryStore {
	return &MemoryStore{pages: make(map[string]Page), keys: make(map[string]string)}
}

func (s *MemoryStore) CreateOrGet(_ context.Context, record CreateRecord) (Page, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	key := record.TenantID + "\x00" + record.TurnID + "\x00" + strconv.Itoa(record.PageIndex)
	if publicID, ok := s.keys[key]; ok {
		return clonePage(s.pages[publicID]), nil
	}
	page := Page{
		PublicID: record.PublicID, TenantID: record.TenantID, OwnerUserID: record.OwnerUserID,
		ConversationID: record.ConversationID, TurnID: record.TurnID, RecipientName: record.RecipientName,
		Message: record.Message, TokenHash: append([]byte(nil), record.TokenHash...), Status: StatusActive,
		CreatedAt: record.CreatedAt, ExpiresAt: record.ExpiresAt,
	}
	s.pages[page.PublicID] = page
	s.keys[key] = page.PublicID
	return clonePage(page), nil
}

func (s *MemoryStore) Redeem(_ context.Context, publicID string, tokenHash []byte, now time.Time) (Page, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	page, ok := s.pages[publicID]
	if !ok {
		return Page{}, ErrForbidden
	}
	if page.Status == StatusRevoked {
		return Page{}, ErrRevoked
	}
	if !now.Before(page.ExpiresAt) {
		page.Status = StatusExpired
		s.pages[publicID] = page
		return Page{}, ErrExpired
	}
	if subtle.ConstantTimeCompare(page.TokenHash, tokenHash) != 1 {
		return Page{}, ErrForbidden
	}
	return clonePage(page), nil
}

func (s *MemoryStore) Revoke(_ context.Context, publicID, tenantID, ownerUserID string, now time.Time) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	page, ok := s.pages[publicID]
	if !ok {
		return ErrNotFound
	}
	if page.TenantID != tenantID || page.OwnerUserID != ownerUserID {
		return ErrForbidden
	}
	page.Status = StatusRevoked
	page.RevokedAt = &now
	s.pages[publicID] = page
	return nil
}

func clonePage(page Page) Page {
	page.TokenHash = append([]byte(nil), page.TokenHash...)
	return page
}
