package group

import (
	"context"
	"testing"
)

func newMemoryStoreForTest() *MemoryStore {
	return NewMemoryStore()
}

func TestMemoryStore_Create(t *testing.T) {
	ctx := context.Background()
	s := newMemoryStoreForTest()

	g := Group{
		TenantID:   "tenant1",
		Name:       "Algebra Class",
		SyllabusID: "math-f1",
		CreatedBy:  "user1",
	}

	got, err := s.Create(ctx, g)
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if got.ID == "" {
		t.Error("Create() expected non-empty ID")
	}
	if got.JoinCode == "" {
		t.Error("Create() expected non-empty JoinCode")
	}
	if got.Status != "active" {
		t.Errorf("Create() status = %q, want %q", got.Status, "active")
	}
	if got.TenantID != "tenant1" {
		t.Errorf("Create() tenantID = %q, want %q", got.TenantID, "tenant1")
	}
	if got.Name != "Algebra Class" {
		t.Errorf("Create() name = %q, want %q", got.Name, "Algebra Class")
	}
	if got.CreatedAt.IsZero() {
		t.Error("Create() expected non-zero CreatedAt")
	}
}

func TestMemoryStore_Create_AddsCreatorAsOwner(t *testing.T) {
	ctx := context.Background()
	s := newMemoryStoreForTest()

	g := Group{
		TenantID:  "tenant1",
		Name:      "Test Group",
		CreatedBy: "creator1",
	}

	got, err := s.Create(ctx, g)
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	members, err := s.GetMembers(ctx, got.ID)
	if err != nil {
		t.Fatalf("GetMembers() error = %v", err)
	}
	if len(members) != 1 {
		t.Fatalf("expected 1 member after create, got %d", len(members))
	}
	if members[0].UserID != "creator1" {
		t.Errorf("member UserID = %q, want %q", members[0].UserID, "creator1")
	}
	if members[0].MembershipRole != "owner" {
		t.Errorf("member MembershipRole = %q, want %q", members[0].MembershipRole, "owner")
	}
}

func TestMemoryStore_GetByID(t *testing.T) {
	ctx := context.Background()
	s := newMemoryStoreForTest()

	created, _ := s.Create(ctx, Group{TenantID: "t1", Name: "G1", CreatedBy: "u1"})

	tests := []struct {
		name     string
		tenantID string
		groupID  string
		wantErr  error
	}{
		{"found", "t1", created.ID, nil},
		{"not found by id", "t1", "nonexistent", ErrGroupNotFound},
		{"wrong tenant", "t2", created.ID, ErrGroupNotFound},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := s.GetByID(ctx, tt.tenantID, tt.groupID)
			if err != tt.wantErr {
				t.Errorf("GetByID() error = %v, want %v", err, tt.wantErr)
			}
			if tt.wantErr == nil && got.ID != created.ID {
				t.Errorf("GetByID() ID = %q, want %q", got.ID, created.ID)
			}
		})
	}
}

func TestMemoryStore_GetByJoinCode(t *testing.T) {
	ctx := context.Background()
	s := newMemoryStoreForTest()

	created, _ := s.Create(ctx, Group{TenantID: "t1", Name: "G1", CreatedBy: "u1"})

	tests := []struct {
		name     string
		tenantID string
		code     string
		wantErr  error
	}{
		{"found", "t1", created.JoinCode, nil},
		{"not found code", "t1", "XXXXXX", ErrGroupNotFound},
		{"wrong tenant isolation", "t2", created.JoinCode, ErrGroupNotFound},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := s.GetByJoinCode(ctx, tt.tenantID, tt.code)
			if err != tt.wantErr {
				t.Errorf("GetByJoinCode() error = %v, want %v", err, tt.wantErr)
			}
			if tt.wantErr == nil && got.JoinCode != created.JoinCode {
				t.Errorf("GetByJoinCode() JoinCode = %q, want %q", got.JoinCode, created.JoinCode)
			}
		})
	}
}

func TestMemoryStore_ListByTenant(t *testing.T) {
	ctx := context.Background()
	s := newMemoryStoreForTest()

	if _, err := s.Create(ctx, Group{TenantID: "t1", Name: "G1", CreatedBy: "u1"}); err != nil {
		t.Fatalf("Create G1 error = %v", err)
	}
	if _, err := s.Create(ctx, Group{TenantID: "t1", Name: "G2", CreatedBy: "u1"}); err != nil {
		t.Fatalf("Create G2 error = %v", err)
	}
	if _, err := s.Create(ctx, Group{TenantID: "t2", Name: "G3", CreatedBy: "u2"}); err != nil {
		t.Fatalf("Create G3 error = %v", err)
	}

	list, err := s.ListByTenant(ctx, "t1")
	if err != nil {
		t.Fatalf("ListByTenant() error = %v", err)
	}
	if len(list) != 2 {
		t.Errorf("ListByTenant() len = %d, want 2", len(list))
	}

	list2, _ := s.ListByTenant(ctx, "t2")
	if len(list2) != 1 {
		t.Errorf("ListByTenant(t2) len = %d, want 1", len(list2))
	}

	list3, _ := s.ListByTenant(ctx, "t9")
	if len(list3) != 0 {
		t.Errorf("ListByTenant(unknown) len = %d, want 0", len(list3))
	}
}

func TestMemoryStore_ListByUser(t *testing.T) {
	ctx := context.Background()
	s := newMemoryStoreForTest()

	g1, _ := s.Create(ctx, Group{TenantID: "t1", Name: "G1", CreatedBy: "alice"})
	g2, _ := s.Create(ctx, Group{TenantID: "t1", Name: "G2", CreatedBy: "bob"})
	// Add alice to g2 as a regular member
	if err := s.AddMember(ctx, g2.ID, "alice", "member"); err != nil {
		t.Fatalf("AddMember alice error = %v", err)
	}
	// bob is owner of g2 only; alice is owner of g1 and member of g2

	list, err := s.ListByUser(ctx, "alice")
	if err != nil {
		t.Fatalf("ListByUser() error = %v", err)
	}
	if len(list) != 2 {
		t.Errorf("ListByUser(alice) len = %d, want 2", len(list))
	}

	listBob, _ := s.ListByUser(ctx, "bob")
	if len(listBob) != 1 {
		t.Errorf("ListByUser(bob) len = %d, want 1", len(listBob))
	}

	_ = g1
}

func TestMemoryStore_Archive(t *testing.T) {
	ctx := context.Background()
	s := newMemoryStoreForTest()

	g, _ := s.Create(ctx, Group{TenantID: "t1", Name: "G1", CreatedBy: "u1"})

	err := s.Archive(ctx, "t1", g.ID)
	if err != nil {
		t.Fatalf("Archive() error = %v", err)
	}

	got, _ := s.GetByID(ctx, "t1", g.ID)
	if got.Status != "archived" {
		t.Errorf("after Archive() status = %q, want %q", got.Status, "archived")
	}

	// Archive wrong tenant should fail
	g2, _ := s.Create(ctx, Group{TenantID: "t1", Name: "G2", CreatedBy: "u1"})
	err = s.Archive(ctx, "t2", g2.ID)
	if err != ErrGroupNotFound {
		t.Errorf("Archive wrong tenant error = %v, want %v", err, ErrGroupNotFound)
	}
}

func TestMemoryStore_AddMember(t *testing.T) {
	ctx := context.Background()
	s := newMemoryStoreForTest()

	g, _ := s.Create(ctx, Group{TenantID: "t1", Name: "G1", CreatedBy: "owner1"})

	// Add a new member
	err := s.AddMember(ctx, g.ID, "user2", "member")
	if err != nil {
		t.Fatalf("AddMember() error = %v", err)
	}

	count, _ := s.MemberCount(ctx, g.ID)
	if count != 2 {
		t.Errorf("MemberCount after add = %d, want 2", count)
	}

	// Duplicate member
	err = s.AddMember(ctx, g.ID, "user2", "member")
	if err != ErrAlreadyMember {
		t.Errorf("duplicate AddMember error = %v, want %v", err, ErrAlreadyMember)
	}

	// Add to archived group
	if err := s.Archive(ctx, "t1", g.ID); err != nil {
		t.Fatalf("Archive error = %v", err)
	}
	err = s.AddMember(ctx, g.ID, "user3", "member")
	if err != ErrGroupArchived {
		t.Errorf("AddMember on archived error = %v, want %v", err, ErrGroupArchived)
	}
}

func TestMemoryStore_RemoveMember(t *testing.T) {
	ctx := context.Background()
	s := newMemoryStoreForTest()

	g, _ := s.Create(ctx, Group{TenantID: "t1", Name: "G1", CreatedBy: "owner1"})
	if err := s.AddMember(ctx, g.ID, "user2", "member"); err != nil {
		t.Fatalf("AddMember user2 error = %v", err)
	}

	// Remove regular member
	err := s.RemoveMember(ctx, g.ID, "user2")
	if err != nil {
		t.Fatalf("RemoveMember() error = %v", err)
	}

	isMember, _ := s.IsMember(ctx, g.ID, "user2")
	if isMember {
		t.Error("user2 should not be a member after remove")
	}

	// Not a member
	err = s.RemoveMember(ctx, g.ID, "nonexistent")
	if err != ErrNotMember {
		t.Errorf("RemoveMember non-member error = %v, want %v", err, ErrNotMember)
	}

	// Owner cannot leave
	err = s.RemoveMember(ctx, g.ID, "owner1")
	if err != ErrOwnerCannotLeave {
		t.Errorf("RemoveMember owner error = %v, want %v", err, ErrOwnerCannotLeave)
	}
}

func TestMemoryStore_GetMembers(t *testing.T) {
	ctx := context.Background()
	s := newMemoryStoreForTest()

	g, _ := s.Create(ctx, Group{TenantID: "t1", Name: "G1", CreatedBy: "owner1"})
	if err := s.AddMember(ctx, g.ID, "user2", "member"); err != nil {
		t.Fatalf("AddMember user2 error = %v", err)
	}
	if err := s.AddMember(ctx, g.ID, "user3", "member"); err != nil {
		t.Fatalf("AddMember user3 error = %v", err)
	}

	members, err := s.GetMembers(ctx, g.ID)
	if err != nil {
		t.Fatalf("GetMembers() error = %v", err)
	}
	if len(members) != 3 {
		t.Errorf("GetMembers() len = %d, want 3", len(members))
	}
}

func TestMemoryStore_MemberCount(t *testing.T) {
	ctx := context.Background()
	s := newMemoryStoreForTest()

	g, _ := s.Create(ctx, Group{TenantID: "t1", Name: "G1", CreatedBy: "owner1"})

	count, err := s.MemberCount(ctx, g.ID)
	if err != nil {
		t.Fatalf("MemberCount() error = %v", err)
	}
	if count != 1 {
		t.Errorf("MemberCount() = %d, want 1", count)
	}

	if err := s.AddMember(ctx, g.ID, "user2", "member"); err != nil {
		t.Fatalf("AddMember user2 error = %v", err)
	}
	count, _ = s.MemberCount(ctx, g.ID)
	if count != 2 {
		t.Errorf("MemberCount after add = %d, want 2", count)
	}
}

func TestMemoryStore_IsMember(t *testing.T) {
	ctx := context.Background()
	s := newMemoryStoreForTest()

	g, _ := s.Create(ctx, Group{TenantID: "t1", Name: "G1", CreatedBy: "owner1"})
	if err := s.AddMember(ctx, g.ID, "user2", "member"); err != nil {
		t.Fatalf("AddMember user2 error = %v", err)
	}

	tests := []struct {
		name   string
		userID string
		want   bool
	}{
		{"owner is member", "owner1", true},
		{"added member is member", "user2", true},
		{"non-member is not", "user3", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := s.IsMember(ctx, g.ID, tt.userID)
			if err != nil {
				t.Fatalf("IsMember() error = %v", err)
			}
			if got != tt.want {
				t.Errorf("IsMember(%q) = %v, want %v", tt.userID, got, tt.want)
			}
		})
	}
}
