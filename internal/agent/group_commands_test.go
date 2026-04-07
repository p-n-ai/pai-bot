package agent_test

import (
	"context"
	"strings"
	"testing"

	"github.com/p-n-ai/pai-bot/internal/agent"
	"github.com/p-n-ai/pai-bot/internal/chat"
)

func TestEngine_CreateGroupCommand_NoArgs(t *testing.T) {
	engine := agent.NewEngine(agent.EngineConfig{
		Groups: agent.NewMemoryGroupStore(),
	})

	resp, err := engine.ProcessMessage(context.Background(), chat.InboundMessage{
		Channel: "telegram",
		UserID:  "user-cg-1",
		Text:    "/create_group",
	})
	if err != nil {
		t.Fatalf("error = %v", err)
	}
	if !strings.Contains(resp, "/create_group") {
		t.Fatalf("response = %q, want usage hint", resp)
	}
}

func TestEngine_CreateGroupCommand_CreatesStudyGroup(t *testing.T) {
	groupStore := agent.NewMemoryGroupStore()
	store := agent.NewMemoryStore()
	store.SetUserName("user-cg-2", "Ali")
	store.SetUserForm("user-cg-2", "Form 1")

	engine := agent.NewEngine(agent.EngineConfig{
		Store:    store,
		Groups:   groupStore,
		TenantID: "test-tenant",
	})

	resp, err := engine.ProcessMessage(context.Background(), chat.InboundMessage{
		Channel: "telegram",
		UserID:  "user-cg-2",
		Text:    "/create_group My Study Squad",
	})
	if err != nil {
		t.Fatalf("error = %v", err)
	}
	if !strings.Contains(resp, "My Study Squad") {
		t.Fatalf("response = %q, want group name in confirmation", resp)
	}

	// Verify the group was created as study_group, not class
	groups, _ := groupStore.ListGroups("test-tenant", "")
	if len(groups) != 1 {
		t.Fatalf("groups = %d, want 1", len(groups))
	}
	if groups[0].Type != "study_group" {
		t.Fatalf("Type = %q, want %q", groups[0].Type, "study_group")
	}

	// Verify creator is a member with leader role
	members, _ := groupStore.GetGroupMembers(groups[0].ID)
	if len(members) != 1 {
		t.Fatalf("members = %d, want 1 (creator)", len(members))
	}
	if members[0].Role != "leader" {
		t.Fatalf("creator role = %q, want %q", members[0].Role, "leader")
	}
}

func TestEngine_JoinCommand_NoArgs(t *testing.T) {
	engine := agent.NewEngine(agent.EngineConfig{
		Groups: agent.NewMemoryGroupStore(),
	})

	resp, err := engine.ProcessMessage(context.Background(), chat.InboundMessage{
		Channel: "telegram",
		UserID:  "user-join-1",
		Text:    "/join",
	})
	if err != nil {
		t.Fatalf("error = %v", err)
	}
	if !strings.Contains(resp, "/join") {
		t.Fatalf("response = %q, want usage hint", resp)
	}
}

func TestEngine_JoinCommand_InvalidCode(t *testing.T) {
	store := agent.NewMemoryStore()
	store.SetUserName("user-join-2", "Bob")
	store.SetUserForm("user-join-2", "Form 1")

	engine := agent.NewEngine(agent.EngineConfig{
		Store:  store,
		Groups: agent.NewMemoryGroupStore(),
	})

	resp, err := engine.ProcessMessage(context.Background(), chat.InboundMessage{
		Channel: "telegram",
		UserID:  "user-join-2",
		Text:    "/join BADCODE",
	})
	if err != nil {
		t.Fatalf("error = %v", err)
	}
	if !strings.Contains(resp, "BADCODE") {
		t.Fatalf("response = %q, want code in error message", resp)
	}
}

func TestEngine_JoinCommand_Success(t *testing.T) {
	groupStore := agent.NewMemoryGroupStore()
	store := agent.NewMemoryStore()
	store.SetUserName("user-join-3", "Charlie")
	store.SetUserForm("user-join-3", "Form 1")

	engine := agent.NewEngine(agent.EngineConfig{
		Store:    store,
		Groups:   groupStore,
		TenantID: "test-tenant",
	})

	g, _ := groupStore.CreateGroup("test-tenant", "Join Test", "study_group", "", "", "", "", "")

	resp, err := engine.ProcessMessage(context.Background(), chat.InboundMessage{
		Channel: "telegram",
		UserID:  "user-join-3",
		Text:    "/join " + g.JoinCode,
	})
	if err != nil {
		t.Fatalf("error = %v", err)
	}
	if !strings.Contains(resp, "Join Test") {
		t.Fatalf("response = %q, want group name in confirmation", resp)
	}

	members, _ := groupStore.GetGroupMembers(g.ID)
	if len(members) != 1 {
		t.Fatalf("members = %d, want 1", len(members))
	}
}

func TestEngine_LeaderboardCommand_NoGroups(t *testing.T) {
	store := agent.NewMemoryStore()
	store.SetUserName("user-lb-1", "Dave")
	store.SetUserForm("user-lb-1", "Form 1")

	engine := agent.NewEngine(agent.EngineConfig{
		Store:  store,
		Groups: agent.NewMemoryGroupStore(),
	})

	resp, err := engine.ProcessMessage(context.Background(), chat.InboundMessage{
		Channel: "telegram",
		UserID:  "user-lb-1",
		Text:    "/leaderboard",
	})
	if err != nil {
		t.Fatalf("error = %v", err)
	}
	if !strings.Contains(resp, "join") || !strings.Contains(resp, "group") {
		t.Fatalf("response = %q, want no-groups message", resp)
	}
}

func TestEngine_LeaderboardCommand_EmptyLeaderboard(t *testing.T) {
	groupStore := agent.NewMemoryGroupStore()
	store := agent.NewMemoryStore()
	store.SetUserName("user-lb-2", "Eve")
	store.SetUserForm("user-lb-2", "Form 1")

	engine := agent.NewEngine(agent.EngineConfig{
		Store:    store,
		Groups:   groupStore,
		TenantID: "test-tenant",
	})

	g, _ := groupStore.CreateGroup("test-tenant", "LB Test", "study_group", "", "", "", "", "")
	_ = groupStore.JoinGroup(g.ID, "user-lb-2", "test-tenant", "member")

	resp, err := engine.ProcessMessage(context.Background(), chat.InboundMessage{
		Channel: "telegram",
		UserID:  "user-lb-2",
		Text:    "/leaderboard",
	})
	if err != nil {
		t.Fatalf("error = %v", err)
	}
	if !strings.Contains(resp, "LB Test") {
		t.Fatalf("response = %q, want group name in empty leaderboard message", resp)
	}
}

func TestEngine_LeaderboardCommand_NonMemberCodeDenied(t *testing.T) {
	groupStore := agent.NewMemoryGroupStore()
	store := agent.NewMemoryStore()
	store.SetUserName("user-lb-3", "Frank")
	store.SetUserForm("user-lb-3", "Form 1")

	engine := agent.NewEngine(agent.EngineConfig{
		Store:    store,
		Groups:   groupStore,
		TenantID: "test-tenant",
	})

	// Create a group but do NOT add user-lb-3 as a member
	g, _ := groupStore.CreateGroup("test-tenant", "Secret Group", "study_group", "", "", "", "", "")
	_ = groupStore.JoinGroup(g.ID, "other-user", "test-tenant", "leader")

	resp, err := engine.ProcessMessage(context.Background(), chat.InboundMessage{
		Channel: "telegram",
		UserID:  "user-lb-3",
		Text:    "/leaderboard " + g.JoinCode,
	})
	if err != nil {
		t.Fatalf("error = %v", err)
	}
	// Non-member should NOT see the group name — should get "no groups" or "not found"
	if strings.Contains(resp, "Secret Group") {
		t.Fatalf("response = %q, non-member should not see group leaderboard", resp)
	}
}
