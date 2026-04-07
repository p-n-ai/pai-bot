package agent

import (
	"fmt"
	"sync"
	"time"
)

// MemoryGroupStore is an in-memory implementation of GroupStore for testing.
type MemoryGroupStore struct {
	groups  map[string]*Group
	members map[string][]groupMemberEntry // groupID -> members
	mu      sync.RWMutex
	counter int
}

type groupMemberEntry struct {
	UserID   string
	TenantID string
	Role     string
	JoinedAt time.Time
}

// NewMemoryGroupStore creates a new in-memory group store.
func NewMemoryGroupStore() *MemoryGroupStore {
	return &MemoryGroupStore{
		groups:  make(map[string]*Group),
		members: make(map[string][]groupMemberEntry),
	}
}

func (s *MemoryGroupStore) CreateGroup(tenantID, name, groupType, description, syllabus, subject, cadence, createdByUserID string) (*Group, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.counter++
	code, err := generateJoinCode()
	if err != nil {
		return nil, err
	}
	g := &Group{
		ID:          fmt.Sprintf("group-%d", s.counter),
		TenantID:    tenantID,
		Name:        name,
		Type:        groupType,
		Description: description,
		Syllabus:    syllabus,
		Subject:     subject,
		Cadence:     cadence,
		JoinCode:    code,
		CreatedBy:   createdByUserID,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}
	s.groups[g.ID] = g
	return g, nil
}

func (s *MemoryGroupStore) GetGroupByID(id string) (*Group, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	g, ok := s.groups[id]
	if !ok {
		return nil, nil
	}
	cp := *g
	cp.MemberCount = len(s.members[id])
	return &cp, nil
}

func (s *MemoryGroupStore) GetGroupByJoinCode(code string) (*Group, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, g := range s.groups {
		if g.JoinCode == code {
			cp := *g
			cp.MemberCount = len(s.members[g.ID])
			return &cp, nil
		}
	}
	return nil, nil
}

func (s *MemoryGroupStore) UpdateGroup(id string, input UpdateGroupInput) (*Group, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	g, ok := s.groups[id]
	if !ok {
		return nil, nil
	}
	if input.Name != nil {
		g.Name = *input.Name
	}
	if input.Description != nil {
		g.Description = *input.Description
	}
	if input.Syllabus != nil {
		g.Syllabus = *input.Syllabus
	}
	if input.Subject != nil {
		g.Subject = *input.Subject
	}
	if input.Cadence != nil {
		g.Cadence = *input.Cadence
	}
	if input.Closed != nil {
		g.Closed = *input.Closed
	}
	g.UpdatedAt = time.Now()
	cp := *g
	return &cp, nil
}

func (s *MemoryGroupStore) DeleteGroup(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.groups[id]; !ok {
		return fmt.Errorf("group not found: %s", id)
	}
	delete(s.groups, id)
	delete(s.members, id)
	return nil
}

func (s *MemoryGroupStore) JoinGroup(groupID, userID, tenantID, role string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	g, ok := s.groups[groupID]
	if !ok {
		return fmt.Errorf("group not found: %s", groupID)
	}
	if g.Closed {
		return ErrGroupClosed
	}
	for _, m := range s.members[groupID] {
		if m.UserID == userID {
			return nil // already a member
		}
	}
	s.members[groupID] = append(s.members[groupID], groupMemberEntry{
		UserID:   userID,
		TenantID: tenantID,
		Role:     role,
		JoinedAt: time.Now(),
	})
	return nil
}

func (s *MemoryGroupStore) LeaveGroup(groupID, userID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	entries := s.members[groupID]
	for i, m := range entries {
		if m.UserID == userID {
			s.members[groupID] = append(entries[:i], entries[i+1:]...)
			return nil
		}
	}
	return nil
}

func (s *MemoryGroupStore) GetGroupMembers(groupID string) ([]GroupMember, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var result []GroupMember
	for _, m := range s.members[groupID] {
		result = append(result, GroupMember{
			UserID:   m.UserID,
			UserName: "User " + m.UserID,
			Role:     m.Role,
			Channel:  "telegram",
			JoinedAt: m.JoinedAt,
		})
	}
	return result, nil
}

func (s *MemoryGroupStore) GetUserGroups(userID string) ([]Group, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var result []Group
	for gID, entries := range s.members {
		for _, m := range entries {
			if m.UserID == userID {
				if g, ok := s.groups[gID]; ok {
					cp := *g
					cp.MemberCount = len(entries)
					result = append(result, cp)
				}
				break
			}
		}
	}
	return result, nil
}

func (s *MemoryGroupStore) ListGroups(tenantID, groupType string) ([]Group, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var result []Group
	for _, g := range s.groups {
		if g.TenantID != tenantID {
			continue
		}
		if groupType != "" && g.Type != groupType {
			continue
		}
		cp := *g
		cp.MemberCount = len(s.members[g.ID])
		result = append(result, cp)
	}
	return result, nil
}

func (s *MemoryGroupStore) GetGroupMembersWithChannel(groupID string) ([]GroupMemberDelivery, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var result []GroupMemberDelivery
	for _, m := range s.members[groupID] {
		result = append(result, GroupMemberDelivery{
			ExternalID: m.UserID, // in memory store, userID doubles as externalID
			Channel:    "telegram",
			UserName:   "User " + m.UserID,
		})
	}
	return result, nil
}

func (s *MemoryGroupStore) GetWeeklyLeaderboard(groupID string, limit int) ([]LeaderboardEntry, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	// In-memory: return empty leaderboard (no mastery snapshots to compare)
	return nil, nil
}
