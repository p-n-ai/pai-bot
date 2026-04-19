// Copyright 2026 the P&AI authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package chat

import (
	"context"
	"testing"
)

// MockEmbedConfigStore is a test double for EmbedConfigStore.
type MockEmbedConfigStore struct {
	Configs map[string]EmbedConfig // keyed by tenant_id
	Tenants map[string]string      // slug -> tenant_id
}

func (m *MockEmbedConfigStore) GetByTenantID(_ context.Context, tenantID string) (EmbedConfig, error) {
	cfg, ok := m.Configs[tenantID]
	if !ok {
		return EmbedConfig{TenantID: tenantID}, nil
	}
	return cfg, nil
}

func (m *MockEmbedConfigStore) GetByTenantSlug(_ context.Context, slug string) (EmbedConfig, error) {
	tenantID, ok := m.Tenants[slug]
	if !ok {
		return EmbedConfig{}, nil
	}
	cfg, ok := m.Configs[tenantID]
	if !ok {
		return EmbedConfig{}, nil
	}
	return cfg, nil
}

func (m *MockEmbedConfigStore) Upsert(_ context.Context, cfg EmbedConfig) (EmbedConfig, error) {
	m.Configs[cfg.TenantID] = cfg
	return cfg, nil
}

func (m *MockEmbedConfigStore) AddOrigin(_ context.Context, tenantID, origin string) error {
	cfg := m.Configs[tenantID]
	cfg.TenantID = tenantID
	for _, o := range cfg.AllowedOrigins {
		if o == origin {
			return nil
		}
	}
	cfg.AllowedOrigins = append(cfg.AllowedOrigins, origin)
	m.Configs[tenantID] = cfg
	return nil
}

func (m *MockEmbedConfigStore) RemoveOrigin(_ context.Context, tenantID, origin string) error {
	cfg := m.Configs[tenantID]
	filtered := cfg.AllowedOrigins[:0]
	for _, o := range cfg.AllowedOrigins {
		if o != origin {
			filtered = append(filtered, o)
		}
	}
	cfg.AllowedOrigins = filtered
	m.Configs[tenantID] = cfg
	return nil
}

func (m *MockEmbedConfigStore) IsOriginAllowed(_ context.Context, tenantID, origin string) (bool, error) {
	cfg, ok := m.Configs[tenantID]
	if !ok || !cfg.Enabled {
		return false, nil
	}
	for _, o := range cfg.AllowedOrigins {
		if o == origin {
			return true, nil
		}
	}
	return false, nil
}

func (m *MockEmbedConfigStore) FindTenantBySlugAndOrigin(_ context.Context, slug, origin string) (string, error) {
	tenantID, ok := m.Tenants[slug]
	if !ok {
		return "", ErrEmbedNotConfigured
	}
	cfg, ok := m.Configs[tenantID]
	if !ok || !cfg.Enabled {
		return "", ErrEmbedNotConfigured
	}
	for _, o := range cfg.AllowedOrigins {
		if o == origin {
			return tenantID, nil
		}
	}
	return "", ErrEmbedNotConfigured
}

func newMockStore() *MockEmbedConfigStore {
	return &MockEmbedConfigStore{
		Configs: make(map[string]EmbedConfig),
		Tenants: make(map[string]string),
	}
}

func TestEmbedConfigStore_MockGetByTenantID(t *testing.T) {
	tests := []struct {
		name       string
		setup      func(s *MockEmbedConfigStore)
		tenantID   string
		wantFound  bool
		wantOrigin string
	}{
		{
			name: "returns config when set",
			setup: func(s *MockEmbedConfigStore) {
				s.Configs["tenant-1"] = EmbedConfig{
					ID:             "cfg-1",
					TenantID:       "tenant-1",
					Enabled:        true,
					AllowedOrigins: []string{"https://example.com"},
				}
			},
			tenantID:   "tenant-1",
			wantFound:  true,
			wantOrigin: "https://example.com",
		},
		{
			name:      "returns zero config when not set",
			setup:     func(_ *MockEmbedConfigStore) {},
			tenantID:  "tenant-missing",
			wantFound: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := newMockStore()
			tt.setup(s)

			cfg, err := s.GetByTenantID(context.Background(), tt.tenantID)
			if err != nil {
				t.Fatalf("GetByTenantID() error = %v", err)
			}
			if cfg.TenantID != tt.tenantID {
				t.Errorf("TenantID = %q, want %q", cfg.TenantID, tt.tenantID)
			}
			if tt.wantFound {
				if cfg.ID == "" {
					t.Error("expected non-empty ID for existing config")
				}
				if len(cfg.AllowedOrigins) == 0 || cfg.AllowedOrigins[0] != tt.wantOrigin {
					t.Errorf("AllowedOrigins[0] = %q, want %q", cfg.AllowedOrigins, tt.wantOrigin)
				}
			} else {
				if cfg.ID != "" {
					t.Errorf("expected empty ID for missing config, got %q", cfg.ID)
				}
			}
		})
	}
}

func TestEmbedConfigStore_MockIsOriginAllowed(t *testing.T) {
	tests := []struct {
		name      string
		setup     func(s *MockEmbedConfigStore)
		tenantID  string
		origin    string
		wantAllow bool
	}{
		{
			name: "allowed when enabled and origin present",
			setup: func(s *MockEmbedConfigStore) {
				s.Configs["t1"] = EmbedConfig{
					TenantID:       "t1",
					Enabled:        true,
					AllowedOrigins: []string{"https://app.example.com"},
				}
			},
			tenantID:  "t1",
			origin:    "https://app.example.com",
			wantAllow: true,
		},
		{
			name: "denied when enabled but origin absent",
			setup: func(s *MockEmbedConfigStore) {
				s.Configs["t2"] = EmbedConfig{
					TenantID:       "t2",
					Enabled:        true,
					AllowedOrigins: []string{"https://app.example.com"},
				}
			},
			tenantID:  "t2",
			origin:    "https://evil.com",
			wantAllow: false,
		},
		{
			name: "denied when disabled",
			setup: func(s *MockEmbedConfigStore) {
				s.Configs["t3"] = EmbedConfig{
					TenantID:       "t3",
					Enabled:        false,
					AllowedOrigins: []string{"https://app.example.com"},
				}
			},
			tenantID:  "t3",
			origin:    "https://app.example.com",
			wantAllow: false,
		},
		{
			name:      "denied when no config exists",
			setup:     func(_ *MockEmbedConfigStore) {},
			tenantID:  "t4",
			origin:    "https://app.example.com",
			wantAllow: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := newMockStore()
			tt.setup(s)

			got, err := s.IsOriginAllowed(context.Background(), tt.tenantID, tt.origin)
			if err != nil {
				t.Fatalf("IsOriginAllowed() error = %v", err)
			}
			if got != tt.wantAllow {
				t.Errorf("IsOriginAllowed() = %v, want %v", got, tt.wantAllow)
			}
		})
	}
}

func TestEmbedConfigStore_MockFindTenantBySlugAndOrigin(t *testing.T) {
	tests := []struct {
		name       string
		setup      func(s *MockEmbedConfigStore)
		slug       string
		origin     string
		wantID     string
		wantErr    bool
		wantErrVal error
	}{
		{
			name: "returns tenant ID when slug and origin match",
			setup: func(s *MockEmbedConfigStore) {
				s.Tenants["school-a"] = "tenant-abc"
				s.Configs["tenant-abc"] = EmbedConfig{
					TenantID:       "tenant-abc",
					Enabled:        true,
					AllowedOrigins: []string{"https://school-a.edu"},
				}
			},
			slug:    "school-a",
			origin:  "https://school-a.edu",
			wantID:  "tenant-abc",
			wantErr: false,
		},
		{
			name:       "returns ErrEmbedNotConfigured when slug not found",
			setup:      func(_ *MockEmbedConfigStore) {},
			slug:       "unknown-school",
			origin:     "https://example.com",
			wantErr:    true,
			wantErrVal: ErrEmbedNotConfigured,
		},
		{
			name: "returns ErrEmbedNotConfigured when embed disabled",
			setup: func(s *MockEmbedConfigStore) {
				s.Tenants["school-b"] = "tenant-xyz"
				s.Configs["tenant-xyz"] = EmbedConfig{
					TenantID:       "tenant-xyz",
					Enabled:        false,
					AllowedOrigins: []string{"https://school-b.edu"},
				}
			},
			slug:       "school-b",
			origin:     "https://school-b.edu",
			wantErr:    true,
			wantErrVal: ErrEmbedNotConfigured,
		},
		{
			name: "returns ErrEmbedNotConfigured when origin not in list",
			setup: func(s *MockEmbedConfigStore) {
				s.Tenants["school-c"] = "tenant-def"
				s.Configs["tenant-def"] = EmbedConfig{
					TenantID:       "tenant-def",
					Enabled:        true,
					AllowedOrigins: []string{"https://school-c.edu"},
				}
			},
			slug:       "school-c",
			origin:     "https://attacker.com",
			wantErr:    true,
			wantErrVal: ErrEmbedNotConfigured,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := newMockStore()
			tt.setup(s)

			got, err := s.FindTenantBySlugAndOrigin(context.Background(), tt.slug, tt.origin)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("FindTenantBySlugAndOrigin() expected error, got nil")
				}
				if tt.wantErrVal != nil && err != tt.wantErrVal {
					t.Errorf("FindTenantBySlugAndOrigin() error = %v, want %v", err, tt.wantErrVal)
				}
				return
			}
			if err != nil {
				t.Fatalf("FindTenantBySlugAndOrigin() error = %v", err)
			}
			if got != tt.wantID {
				t.Errorf("FindTenantBySlugAndOrigin() = %q, want %q", got, tt.wantID)
			}
		})
	}
}
