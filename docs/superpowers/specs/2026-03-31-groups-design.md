# Groups (Classes) Design

**Task:** `P-W3D12-1`
**Date:** 2026-03-31
**Status:** Approved

## Goal

Teachers create groups (displayed as "classes" in admin panel), students join via code. This enables leaderboards and Monday recaps.

## Terminology

- **Database:** `groups` table, `group_members` table
- **Go code:** `internal/group` package, `GroupStore` interface
- **Bot commands:** `/group create`, `/group join` (alias: `/join`)
- **Admin panel/API:** "classes" (`/api/admin/classes`) — maps to groups table

## Schema

```sql
CREATE TABLE groups (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id   UUID NOT NULL REFERENCES tenants(id),
    name        TEXT NOT NULL,
    syllabus_id TEXT,
    join_code   TEXT NOT NULL UNIQUE,
    status      TEXT NOT NULL DEFAULT 'active' CHECK (status IN ('active', 'archived')),
    created_by  UUID REFERENCES users(id) ON DELETE SET NULL,
    created_at  TIMESTAMPTZ DEFAULT NOW(),
    updated_at  TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_groups_tenant_status ON groups(tenant_id, status);
CREATE INDEX idx_groups_join_code ON groups(join_code);

CREATE TABLE group_members (
    group_id        UUID NOT NULL REFERENCES groups(id) ON DELETE CASCADE,
    user_id         UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    membership_role TEXT NOT NULL DEFAULT 'member'
                    CHECK (membership_role IN ('owner', 'admin', 'member')),
    joined_at       TIMESTAMPTZ DEFAULT NOW(),
    PRIMARY KEY (group_id, user_id)
);

CREATE INDEX idx_group_members_user ON group_members(user_id);
```

### Key decisions

- `membership_role` on `group_members` — `'owner'`, `'admin'`, `'member'`. Distinct name from `users.role` to avoid shadowing.
- `created_by` kept on `groups` as audit trail. `membership_role = 'owner'` is the access control mechanism.
- `status` column — supports archiving without hard deletes
- `ON DELETE SET NULL` for `created_by` — group survives teacher removal
- `join_code` is globally unique but join validation includes `tenant_id` check
- `syllabus_id` is nullable — not all groups need a syllabus (future study groups)

## Package Structure

New package: `internal/group/`

```go
// internal/group/store.go
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
    MembershipRole string // "owner", "admin", "member"
    JoinedAt       time.Time
}

type Store interface {
    Create(ctx context.Context, g Group) (*Group, error)
    GetByID(ctx context.Context, tenantID, groupID string) (*Group, error)
    GetByJoinCode(ctx context.Context, tenantID, code string) (*Group, error)
    ListByTenant(ctx context.Context, tenantID string) ([]Group, error)
    ListByUser(ctx context.Context, userID string) ([]Group, error)
    Archive(ctx context.Context, tenantID, groupID string) error
    AddMember(ctx context.Context, groupID, userID, membershipRole string) error
    RemoveMember(ctx context.Context, groupID, userID string) error
    GetMembers(ctx context.Context, groupID string) ([]Member, error)
    MemberCount(ctx context.Context, groupID string) (int, error)
    IsMember(ctx context.Context, groupID, userID string) (bool, error)
}
```

Memory and Postgres implementations. Follows the same pattern as `internal/progress` — imported by both agent engine and admin API.

## Bot Commands

Primary command `/group` with subcommands. `/join` is an alias for `/group join`.

### `/group create [name]`
- **Who:** Teachers and admins only (check `users.role`)
- **Flow:** If no name provided, ask interactively. Ask for Form (1/2/3) via inline buttons. Generate 6-char join code.
- **Response:** "Group created! Share this code with your students: **ABC123**"
- **Needs:** Engine must know user's role. Add `GetUserRole(userID string) (string, bool)` to ConversationStore.

### `/group join [code]` (alias: `/join [code]`)
- **Who:** Any user (primarily students)
- **Validation:**
  1. Look up group by join code
  2. Verify group.tenant_id matches user's tenant_id (critical — prevents cross-school joins)
  3. Verify group.status is 'active'
  4. Verify group.syllabus_id matches user.form (if both are set)
  5. Verify user is not already a member
- **Response:** "Joined **Form 1 Algebra A**! You now have X classmates."
- **Error messages:** code not found, already a member, wrong form level, group archived

### `/group leave`
- **Who:** Any member
- **Response:** "You've left **Form 1 Algebra A**."
- **Owner cannot leave** — checked via `membership_role = 'owner'`, must archive instead

### `/group list`
- **Who:** Any user
- **Response:** Lists user's active groups with names and member counts

## Admin API

All scoped to the authenticated user's tenant.

### `GET /api/admin/classes`
Returns all active groups for the tenant with member counts.
```json
[
  {
    "id": "uuid",
    "name": "Form 1 Algebra A",
    "syllabus_id": "malaysia-kssm-matematik-tingkatan-1",
    "join_code": "ABC123",
    "status": "active",
    "member_count": 24,
    "created_at": "2026-03-31T..."
  }
]
```

### `POST /api/admin/classes`
Create a new group. Requires teacher/admin role.
```json
{
  "name": "Form 1 Algebra A",
  "syllabus_id": "malaysia-kssm-matematik-tingkatan-1"
}
```

### `GET /api/admin/classes/:id`
Group detail with member list + mastery data.

### `PATCH /api/admin/classes/:id`
Update name or archive.

## Tenant Isolation

- Join code is globally unique but `GetByJoinCode(tenantID, code)` filters by tenant
- If no match in the user's tenant, return "code not found" — don't reveal other tenants' codes exist
- All list queries scoped by `tenant_id`

## Syllabus Validation on Join

- If group has `syllabus_id` AND user has `form` set: validate compatibility
- Form "Form 1" → syllabus contains "tingkatan-1", etc.
- If mismatch: "This class is for Form 2 students. Your profile is set to Form 1."
- If either is null: allow join (flexible for cross-form study groups)

## Join Code Generation

6 characters, uppercase alphanumeric, excluding ambiguous characters (0/O, 1/I/L):
```
charset: ABCDEFGHJKMNPQRSTUVWXYZ23456789
```
Generated with `crypto/rand`. Retry on collision (unlikely with 30^6 ≈ 729M combinations).

## Engine Wiring

Add to `EngineConfig`:
```go
Groups group.Store
```

Add `/group` and `/join` to `handleCommand` switch — `/group` delegates to `handleGroupCommand(ctx, msg, args)`, `/join` aliases to `handleGroupCommand(ctx, msg, ["join"] + args)`.

Engine needs access to user's tenant_id and role for validation. Add to `ConversationStore`:
```go
GetUserTenantID(userID string) (string, bool)
GetUserRole(userID string) (string, bool)
```

## What This Does NOT Include

- Leaderboard (`P-W3D12-2`) — separate task, depends on this
- Monday recap (`P-W3D12-3`) — separate task, depends on leaderboard
- Assigned topics per class (`P-W4D18-2`) — separate feature
- Class management admin UI wiring — frontend already has mock scaffold, just needs real API

## Files

| File | Change |
|------|--------|
| `internal/group/store.go` | New: Store interface, Group/Member types |
| `internal/group/store_memory.go` | New: in-memory implementation for tests |
| `internal/group/store_postgres.go` | New: Postgres implementation |
| `internal/group/store_test.go` | New: unit tests |
| `internal/group/joincode.go` | New: join code generation |
| `internal/agent/engine.go` | Add Groups field, `/class` command routing |
| `internal/agent/group_command.go` | New: `/group create/join/leave/list` handlers |
| `internal/agent/group_command_test.go` | New: command handler tests |
| `internal/agent/store.go` | Add GetUserTenantID, GetUserRole to interface |
| `internal/agent/store_postgres.go` | Postgres implementation of new methods |
| `internal/adminapi/service.go` | Add class list/detail/create endpoints |
| `cmd/server/main.go` | Wire GroupStore into engine + admin API |
| `internal/i18n/messages.go` | Add class-related messages (ms/en/zh) |
| `migrations/YYYYMMDD_groups.sql` | New: groups + group_members tables |
| `internal/platform/seed/seed.go` | Add demo classes to seed data |
| `docs/development-timeline.md` | Mark P-W3D12-1 complete |

## Testing

- Unit: join code generation (uniqueness, charset, length)
- Unit: MemoryStore CRUD (create, join, leave, list, archive)
- Unit: tenant isolation (can't join cross-tenant)
- Unit: syllabus validation on join
- Unit: role check (only teacher/admin can create)
- Unit: `/class` command handlers
- Integration: Postgres store with testcontainers
- Live: terminal-chat create + join flow
