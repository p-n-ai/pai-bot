// Copyright 2026 the P&AI authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package agent

import (
	"errors"
	"time"
)

// ErrGroupClosed is returned when attempting to join a closed group.
var ErrGroupClosed = errors.New("group is closed for new members")

// Group represents a grouping entity. A "class" is a group with Type="class".
type Group struct {
	ID          string    `json:"id"`
	TenantID    string    `json:"tenant_id"`
	Name        string    `json:"name"`
	Type        string    `json:"type"` // "class", "study_group"
	Description string    `json:"description"`
	Syllabus    string    `json:"syllabus"`
	Subject     string    `json:"subject"`
	Cadence     string    `json:"cadence"`
	JoinCode    string    `json:"join_code"`
	CreatedBy   string    `json:"created_by"`
	MemberCount int       `json:"member_count"`
	Closed      bool      `json:"closed"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// GroupMember represents a user's membership in a group.
// UserID is the internal users.id UUID.
type GroupMember struct {
	UserID   string    `json:"user_id"`
	UserName string    `json:"user_name"`
	Role     string    `json:"role"` // "member", "leader", "teacher"
	Channel  string    `json:"channel"`
	JoinedAt time.Time `json:"joined_at"`
}

// GroupMemberDelivery contains the fields needed to send a chat message to a group member.
type GroupMemberDelivery struct {
	ExternalID string // external chat ID for gateway.Send
	Channel    string // "telegram", "whatsapp"
	UserName   string
}

// LeaderboardEntry represents a single row in the weekly leaderboard.
type LeaderboardEntry struct {
	UserID      string  `json:"user_id"`
	UserName    string  `json:"user_name"`
	MasteryGain float64 `json:"mastery_gain"` // average gain across topics over 7 days
	Rank        int     `json:"rank"`
}

// UpdateGroupInput contains optional fields for updating a group.
type UpdateGroupInput struct {
	Name        *string `json:"name,omitempty"`
	Description *string `json:"description,omitempty"`
	Syllabus    *string `json:"syllabus,omitempty"`
	Subject     *string `json:"subject,omitempty"`
	Cadence     *string `json:"cadence,omitempty"`
	Closed      *bool   `json:"closed,omitempty"`
}

// GroupStore persists group and membership data.
// All user IDs are internal users.id UUIDs unless otherwise noted.
type GroupStore interface {
	// CRUD
	CreateGroup(tenantID, name, groupType, description, syllabus, subject, cadence, createdByUserID string) (*Group, error)
	GetGroupByID(id string) (*Group, error)
	GetGroupByJoinCode(code string) (*Group, error)
	UpdateGroup(id string, input UpdateGroupInput) (*Group, error)
	DeleteGroup(id string) error

	// Membership — userID is internal UUID, tenantID enforced by DB trigger
	JoinGroup(groupID, userID, tenantID, role string) error
	LeaveGroup(groupID, userID string) error
	GetGroupMembers(groupID string) ([]GroupMember, error)
	GetUserGroups(userID string) ([]Group, error)

	// Enumeration — for scheduler and admin
	ListGroups(tenantID, groupType string) ([]Group, error)
	GetGroupMembersWithChannel(groupID string) ([]GroupMemberDelivery, error)

	// Leaderboard — uses mastery_snapshots for 7-day delta
	GetWeeklyLeaderboard(groupID string, limit int) ([]LeaderboardEntry, error)
}
