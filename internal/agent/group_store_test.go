package agent_test

import (
	"testing"

	"github.com/p-n-ai/pai-bot/internal/agent"
)

func TestGroupStore_CreateGroup(t *testing.T) {
	store := agent.NewMemoryGroupStore()

	g, err := store.CreateGroup("tenant-1", "Form 1 Algebra", "class", "Algebra basics", "KSSM Form 1", "Mathematics", "Mon,Wed,Fri", "")
	if err != nil {
		t.Fatalf("CreateGroup() error = %v", err)
	}
	if g.ID == "" {
		t.Fatal("CreateGroup() returned empty ID")
	}
	if g.Name != "Form 1 Algebra" {
		t.Fatalf("Name = %q, want %q", g.Name, "Form 1 Algebra")
	}
	if g.Type != "class" {
		t.Fatalf("Type = %q, want %q", g.Type, "class")
	}
	if len(g.JoinCode) != 6 {
		t.Fatalf("JoinCode length = %d, want 6", len(g.JoinCode))
	}
}

func TestGroupStore_GetGroupByJoinCode(t *testing.T) {
	store := agent.NewMemoryGroupStore()

	g, _ := store.CreateGroup("tenant-1", "Test Group", "study_group", "", "", "", "", "")

	found, err := store.GetGroupByJoinCode(g.JoinCode)
	if err != nil {
		t.Fatalf("GetGroupByJoinCode() error = %v", err)
	}
	if found == nil {
		t.Fatal("GetGroupByJoinCode() returned nil")
	}
	if found.ID != g.ID {
		t.Fatalf("ID = %q, want %q", found.ID, g.ID)
	}

	notFound, err := store.GetGroupByJoinCode("ZZZZZZ")
	if err != nil {
		t.Fatalf("GetGroupByJoinCode(bad) error = %v", err)
	}
	if notFound != nil {
		t.Fatal("GetGroupByJoinCode(bad) should return nil")
	}
}

func TestGroupStore_JoinAndGetMembers(t *testing.T) {
	store := agent.NewMemoryGroupStore()

	g, _ := store.CreateGroup("tenant-1", "My Group", "study_group", "", "", "", "", "")

	if err := store.JoinGroup(g.ID, "user-1", "tenant-1", "leader"); err != nil {
		t.Fatalf("JoinGroup() error = %v", err)
	}
	if err := store.JoinGroup(g.ID, "user-2", "tenant-1", "member"); err != nil {
		t.Fatalf("JoinGroup() error = %v", err)
	}

	// Joining again should be idempotent
	if err := store.JoinGroup(g.ID, "user-1", "tenant-1", "leader"); err != nil {
		t.Fatalf("JoinGroup(duplicate) error = %v", err)
	}

	members, err := store.GetGroupMembers(g.ID)
	if err != nil {
		t.Fatalf("GetGroupMembers() error = %v", err)
	}
	if len(members) != 2 {
		t.Fatalf("members = %d, want 2", len(members))
	}
	if members[0].Role != "leader" {
		t.Fatalf("first member role = %q, want %q", members[0].Role, "leader")
	}
}

func TestGroupStore_LeaveGroup(t *testing.T) {
	store := agent.NewMemoryGroupStore()

	g, _ := store.CreateGroup("tenant-1", "Leave Test", "study_group", "", "", "", "", "")
	_ = store.JoinGroup(g.ID, "user-1", "tenant-1", "member")
	_ = store.JoinGroup(g.ID, "user-2", "tenant-1", "member")

	if err := store.LeaveGroup(g.ID, "user-1"); err != nil {
		t.Fatalf("LeaveGroup() error = %v", err)
	}

	members, _ := store.GetGroupMembers(g.ID)
	if len(members) != 1 {
		t.Fatalf("members after leave = %d, want 1", len(members))
	}
	if members[0].UserID != "user-2" {
		t.Fatalf("remaining member = %q, want %q", members[0].UserID, "user-2")
	}
}

func TestGroupStore_GetUserGroups(t *testing.T) {
	store := agent.NewMemoryGroupStore()

	g1, _ := store.CreateGroup("tenant-1", "Group A", "study_group", "", "", "", "", "")
	g2, _ := store.CreateGroup("tenant-1", "Group B", "class", "", "", "", "", "")
	_, _ = store.CreateGroup("tenant-1", "Group C", "study_group", "", "", "", "", "")

	_ = store.JoinGroup(g1.ID, "user-1", "tenant-1", "member")
	_ = store.JoinGroup(g2.ID, "user-1", "tenant-1", "member")

	groups, err := store.GetUserGroups("user-1")
	if err != nil {
		t.Fatalf("GetUserGroups() error = %v", err)
	}
	if len(groups) != 2 {
		t.Fatalf("groups = %d, want 2", len(groups))
	}
}

func TestGroupStore_ListGroups(t *testing.T) {
	store := agent.NewMemoryGroupStore()

	_, _ = store.CreateGroup("tenant-1", "Class A", "class", "", "", "", "", "")
	_, _ = store.CreateGroup("tenant-1", "Study B", "study_group", "", "", "", "", "")
	_, _ = store.CreateGroup("tenant-2", "Class C", "class", "", "", "", "", "")

	all, _ := store.ListGroups("tenant-1", "")
	if len(all) != 2 {
		t.Fatalf("ListGroups(tenant-1, all) = %d, want 2", len(all))
	}

	classes, _ := store.ListGroups("tenant-1", "class")
	if len(classes) != 1 {
		t.Fatalf("ListGroups(tenant-1, class) = %d, want 1", len(classes))
	}

	other, _ := store.ListGroups("tenant-2", "")
	if len(other) != 1 {
		t.Fatalf("ListGroups(tenant-2) = %d, want 1", len(other))
	}
}

func TestGroupStore_UpdateGroup(t *testing.T) {
	store := agent.NewMemoryGroupStore()

	g, _ := store.CreateGroup("tenant-1", "Old Name", "class", "", "", "", "", "")

	newName := "New Name"
	newDesc := "Updated description"
	updated, err := store.UpdateGroup(g.ID, agent.UpdateGroupInput{
		Name:        &newName,
		Description: &newDesc,
	})
	if err != nil {
		t.Fatalf("UpdateGroup() error = %v", err)
	}
	if updated.Name != "New Name" {
		t.Fatalf("Name = %q, want %q", updated.Name, "New Name")
	}
	if updated.Description != "Updated description" {
		t.Fatalf("Description = %q, want %q", updated.Description, "Updated description")
	}
}

func TestGroupStore_DeleteGroup(t *testing.T) {
	store := agent.NewMemoryGroupStore()

	g, _ := store.CreateGroup("tenant-1", "Delete Me", "study_group", "", "", "", "", "")
	_ = store.JoinGroup(g.ID, "user-1", "tenant-1", "member")

	if err := store.DeleteGroup(g.ID); err != nil {
		t.Fatalf("DeleteGroup() error = %v", err)
	}

	found, _ := store.GetGroupByID(g.ID)
	if found != nil {
		t.Fatal("GetGroupByID() should return nil after delete")
	}

	if err := store.DeleteGroup("nonexistent"); err == nil {
		t.Fatal("DeleteGroup(nonexistent) should error")
	}
}

func TestGroupStore_MemberCount(t *testing.T) {
	store := agent.NewMemoryGroupStore()

	g, _ := store.CreateGroup("tenant-1", "Count Test", "study_group", "", "", "", "", "")
	_ = store.JoinGroup(g.ID, "user-1", "tenant-1", "member")
	_ = store.JoinGroup(g.ID, "user-2", "tenant-1", "member")
	_ = store.JoinGroup(g.ID, "user-3", "tenant-1", "member")

	found, _ := store.GetGroupByID(g.ID)
	if found.MemberCount != 3 {
		t.Fatalf("MemberCount = %d, want 3", found.MemberCount)
	}
}

func TestGroupStore_JoinClosedGroup(t *testing.T) {
	store := agent.NewMemoryGroupStore()

	g, _ := store.CreateGroup("tenant-1", "Closed Class", "class", "", "", "", "", "")

	// Join before closing — should work
	if err := store.JoinGroup(g.ID, "user-1", "tenant-1", "member"); err != nil {
		t.Fatalf("JoinGroup(open) error = %v", err)
	}

	// Close the group
	closed := true
	_, _ = store.UpdateGroup(g.ID, agent.UpdateGroupInput{Closed: &closed})

	// Join after closing — should fail with ErrGroupClosed
	err := store.JoinGroup(g.ID, "user-2", "tenant-1", "member")
	if err == nil {
		t.Fatal("JoinGroup(closed) should error")
	}
	if err != agent.ErrGroupClosed {
		t.Fatalf("JoinGroup(closed) error = %v, want ErrGroupClosed", err)
	}

	// Verify the first member is still there
	members, _ := store.GetGroupMembers(g.ID)
	if len(members) != 1 {
		t.Fatalf("members = %d, want 1", len(members))
	}
}

func TestGroupStore_GetGroupMembersWithChannel(t *testing.T) {
	store := agent.NewMemoryGroupStore()

	g, _ := store.CreateGroup("tenant-1", "Delivery Test", "study_group", "", "", "", "", "")
	_ = store.JoinGroup(g.ID, "user-1", "tenant-1", "member")
	_ = store.JoinGroup(g.ID, "user-2", "tenant-1", "member")

	deliveries, err := store.GetGroupMembersWithChannel(g.ID)
	if err != nil {
		t.Fatalf("GetGroupMembersWithChannel() error = %v", err)
	}
	if len(deliveries) != 2 {
		t.Fatalf("deliveries = %d, want 2", len(deliveries))
	}
	if deliveries[0].Channel != "telegram" {
		t.Fatalf("Channel = %q, want %q", deliveries[0].Channel, "telegram")
	}
}
