package agent

import (
	"context"
	"fmt"
	"strings"

	"github.com/p-n-ai/pai-bot/internal/chat"
	"github.com/p-n-ai/pai-bot/internal/group"
	"github.com/p-n-ai/pai-bot/internal/i18n"
)

func (e *Engine) handleGroupCommand(ctx context.Context, msg chat.InboundMessage, args []string) (string, error) {
	locale := e.messageLocale(msg, nil)
	if e.groups == nil {
		return i18n.S(locale, i18n.MsgGroupUsage), nil
	}
	if len(args) == 0 {
		return e.handleGroupList(ctx, msg)
	}
	switch strings.ToLower(args[0]) {
	case "create":
		return e.handleGroupCreate(ctx, msg, args[1:])
	case "join":
		return e.handleGroupJoin(ctx, msg, args[1:])
	case "leave":
		return e.handleGroupLeave(ctx, msg)
	case "list":
		return e.handleGroupList(ctx, msg)
	default:
		// If it looks like a join code, treat as join.
		if len(args[0]) >= 4 {
			return e.handleGroupJoin(ctx, msg, args)
		}
		return i18n.S(locale, i18n.MsgGroupUsage), nil
	}
}

func (e *Engine) handleGroupCreate(ctx context.Context, msg chat.InboundMessage, args []string) (string, error) {
	locale := e.messageLocale(msg, nil)

	role, _ := e.store.GetUserRole(msg.UserID)
	if role != "teacher" && role != "admin" && role != "platform_admin" {
		return i18n.S(locale, i18n.MsgGroupCreateDenied), nil
	}

	tenantID, _ := e.store.GetUserTenantID(msg.UserID)
	internalID, _ := e.store.GetUserInternalID(msg.UserID)

	name := strings.TrimSpace(strings.Join(args, " "))
	if name == "" {
		name = "New Group"
	}

	g, err := e.groups.Create(ctx, group.Group{
		TenantID:  tenantID,
		Name:      name,
		CreatedBy: internalID,
	})
	if err != nil {
		return i18n.S(locale, i18n.MsgTechnicalIssue), nil
	}

	return i18n.S(locale, i18n.MsgGroupCreated, g.Name, g.JoinCode), nil
}

func (e *Engine) handleGroupJoin(ctx context.Context, msg chat.InboundMessage, args []string) (string, error) {
	locale := e.messageLocale(msg, nil)

	if len(args) == 0 {
		return i18n.S(locale, i18n.MsgGroupUsage), nil
	}

	code := group.NormalizeJoinCode(args[0])
	tenantID, _ := e.store.GetUserTenantID(msg.UserID)
	internalID, _ := e.store.GetUserInternalID(msg.UserID)

	g, err := e.groups.GetByJoinCode(ctx, tenantID, code)
	if err != nil {
		return i18n.S(locale, i18n.MsgGroupNotFound), nil
	}

	if g.Status == "archived" {
		return i18n.S(locale, i18n.MsgGroupArchived), nil
	}

	// Syllabus validation.
	userForm, _ := e.store.GetUserForm(msg.UserID)
	if !validateSyllabusMatch(userForm, g.SyllabusID) {
		return i18n.S(locale, i18n.MsgGroupSyllabusMismatch, g.SyllabusID, userForm), nil
	}

	err = e.groups.AddMember(ctx, g.ID, internalID, "member")
	if err != nil {
		if err == group.ErrAlreadyMember {
			return i18n.S(locale, i18n.MsgGroupAlreadyMember), nil
		}
		if err == group.ErrGroupArchived {
			return i18n.S(locale, i18n.MsgGroupArchived), nil
		}
		return i18n.S(locale, i18n.MsgTechnicalIssue), nil
	}

	count, _ := e.groups.MemberCount(ctx, g.ID)
	// Subtract 1 for "classmates" (excluding self).
	classmates := count - 1
	if classmates < 0 {
		classmates = 0
	}

	return i18n.S(locale, i18n.MsgGroupJoined, g.Name, classmates), nil
}

func (e *Engine) handleGroupLeave(ctx context.Context, msg chat.InboundMessage) (string, error) {
	locale := e.messageLocale(msg, nil)
	internalID, _ := e.store.GetUserInternalID(msg.UserID)

	groups, err := e.groups.ListByUser(ctx, internalID)
	if err != nil || len(groups) == 0 {
		return i18n.S(locale, i18n.MsgGroupListEmpty), nil
	}

	// Leave the first group (v1 simplicity).
	g := groups[0]
	err = e.groups.RemoveMember(ctx, g.ID, internalID)
	if err != nil {
		if err == group.ErrOwnerCannotLeave {
			return i18n.S(locale, i18n.MsgGroupOwnerCannotLeave), nil
		}
		return i18n.S(locale, i18n.MsgTechnicalIssue), nil
	}

	return i18n.S(locale, i18n.MsgGroupLeft, g.Name), nil
}

func (e *Engine) handleGroupList(ctx context.Context, msg chat.InboundMessage) (string, error) {
	locale := e.messageLocale(msg, nil)
	internalID, _ := e.store.GetUserInternalID(msg.UserID)

	groups, err := e.groups.ListByUser(ctx, internalID)
	if err != nil || len(groups) == 0 {
		return i18n.S(locale, i18n.MsgGroupListEmpty), nil
	}

	var lines []string
	for _, g := range groups {
		count, _ := e.groups.MemberCount(ctx, g.ID)
		lines = append(lines, fmt.Sprintf("- %s (kod: %s, %d ahli)", g.Name, g.JoinCode, count))
	}

	return i18n.S(locale, i18n.MsgGroupList, strings.Join(lines, "\n")), nil
}

// validateSyllabusMatch checks if the user's form is compatible with the group's syllabus.
func validateSyllabusMatch(userForm, groupSyllabusID string) bool {
	if userForm == "" || groupSyllabusID == "" {
		return true
	}
	// Extract form number from user's form string.
	formNum := extractFormNumber(userForm)
	if formNum == "" {
		return true
	}
	// Check if syllabus_id contains a matching tingkatan reference.
	lower := strings.ToLower(groupSyllabusID)
	return strings.Contains(lower, "tingkatan-"+formNum) ||
		strings.Contains(lower, "form-"+formNum) ||
		strings.Contains(lower, "form"+formNum)
}

// extractFormNumber pulls the digit from strings like "Form 1", "Tingkatan 2", "3".
func extractFormNumber(form string) string {
	form = strings.TrimSpace(form)
	for _, c := range form {
		if c >= '1' && c <= '5' {
			return string(c)
		}
	}
	return ""
}
