package agent

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/p-n-ai/pai-bot/internal/chat"
	"github.com/p-n-ai/pai-bot/internal/i18n"
)

// handleCreateGroupCommand handles "/create_group <name>".
// Bot-created groups are always type "study_group" (never "class").
func (e *Engine) handleCreateGroupCommand(_ context.Context, msg chat.InboundMessage, args []string) (string, error) {
	locale := e.messageLocale(msg, nil)

	if len(args) == 0 {
		return i18n.S(locale, i18n.MsgGroupCreateUsage), nil
	}

	name := strings.Join(args, " ")
	if len(name) > 100 {
		name = name[:100]
	}

	userUUID, err := e.store.ResolveUserUUID(msg.UserID)
	if err != nil {
		return "", fmt.Errorf("resolve user for create_group: %w", err)
	}
	if userUUID == "" {
		return i18n.S(locale, i18n.MsgGroupUserNotFound), nil
	}

	g, err := e.groups.CreateGroup(e.tenantID, name, "study_group", "", "", "", "", userUUID)
	if err != nil {
		return "", fmt.Errorf("create group: %w", err)
	}

	// Creator joins as leader
	if err := e.groups.JoinGroup(g.ID, userUUID, e.tenantID, "leader"); err != nil {
		return "", fmt.Errorf("join own group: %w", err)
	}

	return i18n.S(locale, i18n.MsgGroupCreated, g.Name, g.JoinCode, g.JoinCode), nil
}

// handleJoinGroupCommand handles "/join <code>".
func (e *Engine) handleJoinGroupCommand(_ context.Context, msg chat.InboundMessage, args []string) (string, error) {
	locale := e.messageLocale(msg, nil)

	if len(args) == 0 {
		return i18n.S(locale, i18n.MsgGroupJoinUsage), nil
	}

	code := strings.ToUpper(strings.TrimSpace(args[0]))

	g, err := e.groups.GetGroupByJoinCode(code)
	if err != nil {
		return "", fmt.Errorf("lookup group by code: %w", err)
	}
	if g == nil {
		return i18n.S(locale, i18n.MsgGroupNotFound, code), nil
	}

	userUUID, err := e.store.ResolveUserUUID(msg.UserID)
	if err != nil {
		return "", fmt.Errorf("resolve user for join: %w", err)
	}
	if userUUID == "" {
		return i18n.S(locale, i18n.MsgGroupUserNotFound), nil
	}

	if err := e.groups.JoinGroup(g.ID, userUUID, g.TenantID, "member"); err != nil {
		if errors.Is(err, ErrGroupClosed) {
			return i18n.S(locale, i18n.MsgGroupClosed, g.Name), nil
		}
		return "", fmt.Errorf("join group: %w", err)
	}

	return i18n.S(locale, i18n.MsgGroupJoined, g.Name), nil
}

// handleLeaderboardCommand handles "/leaderboard [code]".
// Without args, shows leaderboard for the most recently joined group.
func (e *Engine) handleLeaderboardCommand(_ context.Context, msg chat.InboundMessage, args []string) (string, error) {
	locale := e.messageLocale(msg, nil)

	userUUID, err := e.store.ResolveUserUUID(msg.UserID)
	if err != nil {
		return "", fmt.Errorf("resolve user for leaderboard: %w", err)
	}
	if userUUID == "" {
		return i18n.S(locale, i18n.MsgGroupUserNotFound), nil
	}

	var g *Group

	// Always resolve from the user's own groups to prevent cross-tenant/non-member leaks.
	userGroups, err := e.groups.GetUserGroups(userUUID)
	if err != nil {
		return "", fmt.Errorf("get user groups for leaderboard: %w", err)
	}
	if len(userGroups) == 0 {
		return i18n.S(locale, i18n.MsgGroupNoGroups), nil
	}

	if len(args) > 0 {
		code := strings.ToUpper(strings.TrimSpace(args[0]))
		for i := range userGroups {
			if userGroups[i].JoinCode == code {
				g = &userGroups[i]
				break
			}
		}
		if g == nil {
			return i18n.S(locale, i18n.MsgGroupNotFound, code), nil
		}
	} else {
		g = &userGroups[0] // most recently joined
	}

	entries, err := e.groups.GetWeeklyLeaderboard(g.ID, 10)
	if err != nil {
		return "", fmt.Errorf("get leaderboard: %w", err)
	}
	if len(entries) == 0 {
		return i18n.S(locale, i18n.MsgLeaderboardEmpty, g.Name), nil
	}

	return formatLeaderboard(g.Name, entries, locale), nil
}

// handleDevCloseGroup toggles a group's closed state (dev-mode only).
func (e *Engine) handleDevCloseGroup(args []string) (string, error) {
	if len(args) == 0 {
		return "Usage: /dev-close-group CODE", nil
	}
	code := strings.ToUpper(strings.TrimSpace(args[0]))
	g, err := e.groups.GetGroupByJoinCode(code)
	if err != nil {
		return "", fmt.Errorf("lookup group: %w", err)
	}
	if g == nil {
		return fmt.Sprintf("No group found with code %s", code), nil
	}
	newClosed := !g.Closed
	_, err = e.groups.UpdateGroup(g.ID, UpdateGroupInput{Closed: &newClosed})
	if err != nil {
		return "", fmt.Errorf("toggle group closed: %w", err)
	}
	if newClosed {
		return fmt.Sprintf("Group *%s* is now CLOSED (no new joins).", g.Name), nil
	}
	return fmt.Sprintf("Group *%s* is now OPEN (joins allowed).", g.Name), nil
}

func formatLeaderboard(groupName string, entries []LeaderboardEntry, locale string) string {
	_ = locale // reserved for future i18n

	var b strings.Builder
	fmt.Fprintf(&b, "🏆 *%s — Weekly Leaderboard*\n\n", groupName)

	medals := []string{"🥇", "🥈", "🥉"}
	for _, e := range entries {
		prefix := fmt.Sprintf("%d.", e.Rank)
		if e.Rank <= 3 {
			prefix = medals[e.Rank-1]
		}
		fmt.Fprintf(&b, "%s %s — +%.0f%%\n", prefix, e.UserName, e.MasteryGain*100)
	}

	return b.String()
}
