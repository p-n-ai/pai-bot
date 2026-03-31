package group

import (
	"context"
	"errors"
	"time"
)

var (
	ErrGroupNotFound    = errors.New("group not found")
	ErrAlreadyMember    = errors.New("already a member")
	ErrNotMember        = errors.New("not a member")
	ErrOwnerCannotLeave = errors.New("owner cannot leave group")
	ErrGroupArchived    = errors.New("group is archived")
)

type Group struct {
	ID         string
	TenantID   string
	Name       string
	SyllabusID string
	JoinCode   string
	Status     string
	CreatedBy  string
	CreatedAt  time.Time
}

type Member struct {
	UserID         string
	MembershipRole string
	JoinedAt       time.Time
}

type Store interface {
	Create(ctx context.Context, g Group) (*Group, error)
	GetByID(ctx context.Context, tenantID, groupID string) (*Group, error)
	GetByJoinCode(ctx context.Context, tenantID, code string) (*Group, error)
	ListByTenant(ctx context.Context, tenantID string) ([]Group, error)
	ListByUser(ctx context.Context, userID string) ([]Group, error)
	Archive(ctx context.Context, tenantID, groupID string) error
	Rename(ctx context.Context, tenantID, groupID, newName string) error
	AddMember(ctx context.Context, groupID, userID, membershipRole string) error
	RemoveMember(ctx context.Context, groupID, userID string) error
	GetMembers(ctx context.Context, groupID string) ([]Member, error)
	MemberCount(ctx context.Context, groupID string) (int, error)
	IsMember(ctx context.Context, groupID, userID string) (bool, error)
}
