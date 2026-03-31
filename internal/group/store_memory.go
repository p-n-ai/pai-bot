package group

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"sync"
	"time"
)

// MemoryStore is an in-memory implementation of Store, suitable for testing.
type MemoryStore struct {
	mu      sync.RWMutex
	groups  map[string]*Group   // id → Group
	members map[string][]Member // groupID → members
	codes   map[string]string   // joinCode → groupID
}

// NewMemoryStore creates a new empty MemoryStore.
func NewMemoryStore() *MemoryStore {
	return &MemoryStore{
		groups:  make(map[string]*Group),
		members: make(map[string][]Member),
		codes:   make(map[string]string),
	}
}

func generateID() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

// Create stores a new group and adds the creator as owner.
func (s *MemoryStore) Create(ctx context.Context, g Group) (*Group, error) {
	id, err := generateID()
	if err != nil {
		return nil, err
	}
	code, err := GenerateJoinCode()
	if err != nil {
		return nil, err
	}

	g.ID = id
	g.JoinCode = code
	g.Status = "active"
	if g.CreatedAt.IsZero() {
		g.CreatedAt = time.Now().UTC()
	}

	s.mu.Lock()
	s.groups[id] = &g
	s.codes[code] = id
	s.mu.Unlock()

	// Add creator as owner (uses AddMember which locks internally).
	if err := s.AddMember(ctx, id, g.CreatedBy, "owner"); err != nil {
		return nil, err
	}

	// Return a copy.
	cp := g
	return &cp, nil
}

// GetByID returns the group if it belongs to tenantID.
func (s *MemoryStore) GetByID(ctx context.Context, tenantID, groupID string) (*Group, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	g, ok := s.groups[groupID]
	if !ok || g.TenantID != tenantID {
		return nil, ErrGroupNotFound
	}
	cp := *g
	return &cp, nil
}

// GetByJoinCode looks up a group by join code and enforces tenant isolation.
func (s *MemoryStore) GetByJoinCode(ctx context.Context, tenantID, code string) (*Group, error) {
	code = NormalizeJoinCode(code)

	s.mu.RLock()
	defer s.mu.RUnlock()

	groupID, ok := s.codes[code]
	if !ok {
		return nil, ErrGroupNotFound
	}
	g, ok := s.groups[groupID]
	if !ok || g.TenantID != tenantID {
		return nil, ErrGroupNotFound
	}
	cp := *g
	return &cp, nil
}

// ListByTenant returns all groups belonging to tenantID.
func (s *MemoryStore) ListByTenant(ctx context.Context, tenantID string) ([]Group, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var result []Group
	for _, g := range s.groups {
		if g.TenantID == tenantID {
			result = append(result, *g)
		}
	}
	return result, nil
}

// ListByUser returns all groups where userID is a member.
func (s *MemoryStore) ListByUser(ctx context.Context, userID string) ([]Group, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var result []Group
	for groupID, members := range s.members {
		for _, m := range members {
			if m.UserID == userID {
				if g, ok := s.groups[groupID]; ok {
					result = append(result, *g)
				}
				break
			}
		}
	}
	return result, nil
}

// Archive sets the group's status to "archived".
func (s *MemoryStore) Archive(ctx context.Context, tenantID, groupID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	g, ok := s.groups[groupID]
	if !ok || g.TenantID != tenantID {
		return ErrGroupNotFound
	}
	g.Status = "archived"
	return nil
}

// Rename updates the name of a group.
func (s *MemoryStore) Rename(ctx context.Context, tenantID, groupID, newName string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	g, ok := s.groups[groupID]
	if !ok || g.TenantID != tenantID {
		return ErrGroupNotFound
	}
	g.Name = newName
	return nil
}

// AddMember adds userID to the group with the given role.
// Returns ErrAlreadyMember if already present, ErrGroupArchived if group is archived.
func (s *MemoryStore) AddMember(ctx context.Context, groupID, userID, membershipRole string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	g, ok := s.groups[groupID]
	if !ok {
		return ErrGroupNotFound
	}
	if g.Status == "archived" {
		return ErrGroupArchived
	}

	for _, m := range s.members[groupID] {
		if m.UserID == userID {
			return ErrAlreadyMember
		}
	}

	s.members[groupID] = append(s.members[groupID], Member{
		UserID:         userID,
		MembershipRole: membershipRole,
		JoinedAt:       time.Now().UTC(),
	})
	return nil
}

// RemoveMember removes userID from the group.
// Returns ErrNotMember if not present, ErrOwnerCannotLeave if they are the owner.
func (s *MemoryStore) RemoveMember(ctx context.Context, groupID, userID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	members := s.members[groupID]
	idx := -1
	for i, m := range members {
		if m.UserID == userID {
			if m.MembershipRole == "owner" {
				return ErrOwnerCannotLeave
			}
			idx = i
			break
		}
	}
	if idx == -1 {
		return ErrNotMember
	}

	s.members[groupID] = append(members[:idx], members[idx+1:]...)
	return nil
}

// GetMembers returns all members of a group.
func (s *MemoryStore) GetMembers(ctx context.Context, groupID string) ([]Member, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	members := s.members[groupID]
	result := make([]Member, len(members))
	copy(result, members)
	return result, nil
}

// MemberCount returns the number of members in a group.
func (s *MemoryStore) MemberCount(ctx context.Context, groupID string) (int, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return len(s.members[groupID]), nil
}

// IsMember reports whether userID is a member of groupID.
func (s *MemoryStore) IsMember(ctx context.Context, groupID, userID string) (bool, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	for _, m := range s.members[groupID] {
		if m.UserID == userID {
			return true, nil
		}
	}
	return false, nil
}
