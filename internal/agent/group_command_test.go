package agent_test

import (
	"context"
	"strings"
	"testing"

	"github.com/p-n-ai/pai-bot/internal/agent"
	"github.com/p-n-ai/pai-bot/internal/chat"
	"github.com/p-n-ai/pai-bot/internal/group"
)

func setupGroupTestEngine(t *testing.T) (*agent.Engine, *agent.MemoryStore, *group.MemoryStore) {
	t.Helper()
	store := agent.NewMemoryStore()
	groupStore := group.NewMemoryStore()
	engine := agent.NewEngine(agent.EngineConfig{
		Store:                store,
		EventLogger:          agent.NewMemoryEventLogger(),
		Groups:               groupStore,
		DisableMultiLanguage: true,
	})
	return engine, store, groupStore
}

func setGroupTestUser(t *testing.T, store *agent.MemoryStore, userID, role, tenantID string) {
	t.Helper()
	if err := store.SetUserRole(userID, role); err != nil {
		t.Fatalf("SetUserRole(%s): %v", userID, err)
	}
	if err := store.SetUserTenantID(userID, tenantID); err != nil {
		t.Fatalf("SetUserTenantID(%s): %v", userID, err)
	}
}

func TestGroupCreate_TeacherSuccess(t *testing.T) {
	engine, store, _ := setupGroupTestEngine(t)
	setGroupTestUser(t, store, "teacher-1", "teacher", "tenant-1")

	resp, err := engine.ProcessMessage(context.Background(), chat.InboundMessage{
		Channel: "terminal",
		UserID:  "teacher-1",
		Text:    "/group create Math Class",
	})
	if err != nil {
		t.Fatalf("ProcessMessage: %v", err)
	}

	if !strings.Contains(resp, "Math Class") {
		t.Errorf("expected response to contain group name, got: %s", resp)
	}
	// The response should contain a join code (6 uppercase alphanum chars).
	lower := strings.ToLower(resp)
	if !strings.Contains(lower, "kod") && !strings.Contains(lower, "code") {
		t.Errorf("expected response to contain a join code reference, got: %s", resp)
	}
}

func TestGroupCreate_StudentDenied(t *testing.T) {
	engine, store, _ := setupGroupTestEngine(t)
	setGroupTestUser(t, store, "student-1", "student", "tenant-1")

	resp, err := engine.ProcessMessage(context.Background(), chat.InboundMessage{
		Channel: "terminal",
		UserID:  "student-1",
		Text:    "/group create Study Group",
	})
	if err != nil {
		t.Fatalf("ProcessMessage: %v", err)
	}

	// Should get denied message (ms locale default: "Hanya guru dan pentadbir").
	if !strings.Contains(resp, "guru") && !strings.Contains(resp, "teacher") && !strings.Contains(resp, "admin") {
		t.Errorf("expected denial message, got: %s", resp)
	}
}

func TestGroupCreate_AdminSuccess(t *testing.T) {
	engine, store, _ := setupGroupTestEngine(t)
	setGroupTestUser(t, store, "admin-1", "admin", "tenant-1")

	resp, err := engine.ProcessMessage(context.Background(), chat.InboundMessage{
		Channel: "terminal",
		UserID:  "admin-1",
		Text:    "/group create Admin Group",
	})
	if err != nil {
		t.Fatalf("ProcessMessage: %v", err)
	}

	if !strings.Contains(resp, "Admin Group") {
		t.Errorf("expected response to contain group name, got: %s", resp)
	}
}

func TestGroupJoin_Success(t *testing.T) {
	engine, store, groupStore := setupGroupTestEngine(t)

	// Teacher creates a group first.
	setGroupTestUser(t, store, "teacher-1", "teacher", "tenant-1")
	g, err := groupStore.Create(context.Background(), group.Group{
		TenantID:  "tenant-1",
		Name:      "Math Class",
		CreatedBy: "teacher-1",
	})
	if err != nil {
		t.Fatalf("Create group: %v", err)
	}

	// Student joins.
	setGroupTestUser(t, store, "student-1", "student", "tenant-1")

	resp, err := engine.ProcessMessage(context.Background(), chat.InboundMessage{
		Channel: "terminal",
		UserID:  "student-1",
		Text:    "/group join " + g.JoinCode,
	})
	if err != nil {
		t.Fatalf("ProcessMessage: %v", err)
	}

	if !strings.Contains(resp, "Math Class") {
		t.Errorf("expected response to contain group name, got: %s", resp)
	}
}

func TestGroupJoin_InvalidCode(t *testing.T) {
	engine, store, _ := setupGroupTestEngine(t)
	setGroupTestUser(t, store, "student-1", "student", "tenant-1")

	resp, err := engine.ProcessMessage(context.Background(), chat.InboundMessage{
		Channel: "terminal",
		UserID:  "student-1",
		Text:    "/group join ZZZZZZ",
	})
	if err != nil {
		t.Fatalf("ProcessMessage: %v", err)
	}

	// Should contain "not found" or Malay equivalent.
	if !strings.Contains(resp, "tidak dijumpai") && !strings.Contains(resp, "not found") && !strings.Contains(resp, "tidak") {
		t.Errorf("expected not-found message, got: %s", resp)
	}
}

func TestGroupJoin_AlreadyMember(t *testing.T) {
	engine, store, groupStore := setupGroupTestEngine(t)

	setGroupTestUser(t, store, "teacher-1", "teacher", "tenant-1")
	g, err := groupStore.Create(context.Background(), group.Group{
		TenantID:  "tenant-1",
		Name:      "Math Class",
		CreatedBy: "teacher-1",
	})
	if err != nil {
		t.Fatalf("Create group: %v", err)
	}

	setGroupTestUser(t, store, "student-1", "student", "tenant-1")

	// Join once.
	_, err = engine.ProcessMessage(context.Background(), chat.InboundMessage{
		Channel: "terminal",
		UserID:  "student-1",
		Text:    "/group join " + g.JoinCode,
	})
	if err != nil {
		t.Fatalf("first join: %v", err)
	}

	// Join again.
	resp, err := engine.ProcessMessage(context.Background(), chat.InboundMessage{
		Channel: "terminal",
		UserID:  "student-1",
		Text:    "/group join " + g.JoinCode,
	})
	if err != nil {
		t.Fatalf("second join: %v", err)
	}

	if !strings.Contains(resp, "sudah") && !strings.Contains(resp, "already") {
		t.Errorf("expected already-member message, got: %s", resp)
	}
}

func TestGroupLeave_Success(t *testing.T) {
	engine, store, groupStore := setupGroupTestEngine(t)

	setGroupTestUser(t, store, "teacher-1", "teacher", "tenant-1")
	g, err := groupStore.Create(context.Background(), group.Group{
		TenantID:  "tenant-1",
		Name:      "Math Class",
		CreatedBy: "teacher-1",
	})
	if err != nil {
		t.Fatalf("Create group: %v", err)
	}

	setGroupTestUser(t, store, "student-1", "student", "tenant-1")

	// Join first.
	if err := groupStore.AddMember(context.Background(), g.ID, "student-1", "member"); err != nil {
		t.Fatalf("AddMember: %v", err)
	}

	resp, err := engine.ProcessMessage(context.Background(), chat.InboundMessage{
		Channel: "terminal",
		UserID:  "student-1",
		Text:    "/group leave",
	})
	if err != nil {
		t.Fatalf("ProcessMessage: %v", err)
	}

	if !strings.Contains(resp, "Math Class") {
		t.Errorf("expected response to mention group name, got: %s", resp)
	}
	if !strings.Contains(resp, "keluar") && !strings.Contains(resp, "left") {
		t.Errorf("expected leave confirmation, got: %s", resp)
	}
}

func TestGroupLeave_OwnerDenied(t *testing.T) {
	engine, store, groupStore := setupGroupTestEngine(t)

	setGroupTestUser(t, store, "teacher-1", "teacher", "tenant-1")
	_, err := groupStore.Create(context.Background(), group.Group{
		TenantID:  "tenant-1",
		Name:      "Math Class",
		CreatedBy: "teacher-1",
	})
	if err != nil {
		t.Fatalf("Create group: %v", err)
	}

	resp, err := engine.ProcessMessage(context.Background(), chat.InboundMessage{
		Channel: "terminal",
		UserID:  "teacher-1",
		Text:    "/group leave",
	})
	if err != nil {
		t.Fatalf("ProcessMessage: %v", err)
	}

	if !strings.Contains(resp, "Pemilik") && !strings.Contains(resp, "Owner") && !strings.Contains(resp, "owner") {
		t.Errorf("expected owner-cannot-leave message, got: %s", resp)
	}
}

func TestGroupList_WithGroups(t *testing.T) {
	engine, store, groupStore := setupGroupTestEngine(t)

	setGroupTestUser(t, store, "student-1", "student", "tenant-1")

	// Create a group and add student.
	setGroupTestUser(t, store, "teacher-1", "teacher", "tenant-1")
	g, err := groupStore.Create(context.Background(), group.Group{
		TenantID:  "tenant-1",
		Name:      "Math Class",
		CreatedBy: "teacher-1",
	})
	if err != nil {
		t.Fatalf("Create group: %v", err)
	}
	if err := groupStore.AddMember(context.Background(), g.ID, "student-1", "member"); err != nil {
		t.Fatalf("AddMember: %v", err)
	}

	resp, err := engine.ProcessMessage(context.Background(), chat.InboundMessage{
		Channel: "terminal",
		UserID:  "student-1",
		Text:    "/group list",
	})
	if err != nil {
		t.Fatalf("ProcessMessage: %v", err)
	}

	if !strings.Contains(resp, "Math Class") {
		t.Errorf("expected group name in list, got: %s", resp)
	}
}

func TestGroupList_Empty(t *testing.T) {
	engine, store, _ := setupGroupTestEngine(t)
	setGroupTestUser(t, store, "student-1", "student", "tenant-1")

	resp, err := engine.ProcessMessage(context.Background(), chat.InboundMessage{
		Channel: "terminal",
		UserID:  "student-1",
		Text:    "/group list",
	})
	if err != nil {
		t.Fatalf("ProcessMessage: %v", err)
	}

	if !strings.Contains(resp, "belum") && !strings.Contains(resp, "haven't") && !strings.Contains(resp, "join") {
		t.Errorf("expected empty-list message, got: %s", resp)
	}
}

func TestJoinAlias(t *testing.T) {
	engine, store, groupStore := setupGroupTestEngine(t)

	setGroupTestUser(t, store, "teacher-1", "teacher", "tenant-1")
	g, err := groupStore.Create(context.Background(), group.Group{
		TenantID:  "tenant-1",
		Name:      "Science Group",
		CreatedBy: "teacher-1",
	})
	if err != nil {
		t.Fatalf("Create group: %v", err)
	}

	setGroupTestUser(t, store, "student-1", "student", "tenant-1")

	// Use /join directly instead of /group join.
	resp, err := engine.ProcessMessage(context.Background(), chat.InboundMessage{
		Channel: "terminal",
		UserID:  "student-1",
		Text:    "/join " + g.JoinCode,
	})
	if err != nil {
		t.Fatalf("ProcessMessage: %v", err)
	}

	if !strings.Contains(resp, "Science Group") {
		t.Errorf("expected response to contain group name, got: %s", resp)
	}
}

func TestGroupUsage_NoSubcommand(t *testing.T) {
	engine, store, _ := setupGroupTestEngine(t)
	setGroupTestUser(t, store, "student-1", "student", "tenant-1")

	// /group with no args should show list (not usage)
	resp, err := engine.ProcessMessage(context.Background(), chat.InboundMessage{
		Channel: "terminal",
		UserID:  "student-1",
		Text:    "/group",
	})
	if err != nil {
		t.Fatalf("ProcessMessage: %v", err)
	}

	// With no groups, should show empty list message.
	if !strings.Contains(resp, "belum") && !strings.Contains(resp, "haven't") {
		t.Errorf("expected empty list message for bare /group, got: %s", resp)
	}
}

func TestGroupJoin_ArchivedGroup(t *testing.T) {
	engine, store, groupStore := setupGroupTestEngine(t)

	setGroupTestUser(t, store, "teacher-1", "teacher", "tenant-1")
	g, err := groupStore.Create(context.Background(), group.Group{
		TenantID:  "tenant-1",
		Name:      "Old Group",
		CreatedBy: "teacher-1",
	})
	if err != nil {
		t.Fatalf("Create group: %v", err)
	}

	// Archive it.
	if err := groupStore.Archive(context.Background(), "tenant-1", g.ID); err != nil {
		t.Fatalf("Archive: %v", err)
	}

	setGroupTestUser(t, store, "student-1", "student", "tenant-1")

	resp, err := engine.ProcessMessage(context.Background(), chat.InboundMessage{
		Channel: "terminal",
		UserID:  "student-1",
		Text:    "/join " + g.JoinCode,
	})
	if err != nil {
		t.Fatalf("ProcessMessage: %v", err)
	}

	if !strings.Contains(resp, "diarkibkan") && !strings.Contains(resp, "archived") {
		t.Errorf("expected archived message, got: %s", resp)
	}
}
