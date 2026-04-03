# A/B Test Infrastructure Design

**Task:** `P-W3D13-1`
**Date:** 2026-03-27
**Status:** Approved

## Goal

Randomly assign new students to group A (all motivation features) or group B (no motivation features) to measure whether gamification improves retention and mastery gains.

## Flag Storage

Use the existing `users.config` JSONB column. Add `"ab_group": "A"` or `"B"`.

No new migration for the column — it already exists. One migration to backfill existing users to group A.

## Assignment Logic

- **New students:** 50/50 random assignment during `/start` and auto-start flows. Determined by `rand.Intn(2)` — 0 = A, 1 = B.
- **Existing students:** Backfill all to group A via migration (`UPDATE users SET config = config || '{"ab_group":"A"}' WHERE config->>'ab_group' IS NULL`).

Assignment happens in `resolveOrCreateUser` or immediately after user creation in `handleStart`/auto-start. Once assigned, the group never changes (unless manually overridden via `/dev-ab`).

## Feature Gating

Group B disables:
- **Milestone celebrations** — `milestones.add()` calls in `assessMasteryAsync`, `recordActivityAsync`, `recordQuizOutcomeAsync` are skipped
- **Streak celebration messages** — `FormatStreakRecordCelebration` is skipped, but `RecordActivity` still runs (streaks are tracked, just not celebrated)
- **Proactive nudges** — scheduler's `checkUser` skips group B users entirely

Group B still gets:
- All tutoring and AI responses
- Quizzes (entry, grading, hints, summary)
- Progress tracking (`/progress` command works normally)
- XP and streaks are recorded internally (for fair post-experiment comparison)
- Daily summary at 22:00 MYT (this is informational, not motivational)
- All events logged (essential for analysis)

## Implementation Approach

### Reading the flag

Add a helper to the `ConversationStore` interface or a standalone function:

```go
func (e *Engine) userABGroup(userID string) string
```

Returns `"A"`, `"B"`, or `"A"` as default if unset. Reads from `users.config->>'ab_group'` via the store.

### Store interface addition

Add to `ConversationStore`:

```go
GetUserABGroup(userID string) (string, bool)
SetUserABGroup(userID, group string) error
```

Memory and Postgres implementations.

### Gating milestones

In `assessMasteryAsync`, `recordActivityAsync`, and `recordQuizOutcomeAsync`, wrap milestone adds:

```go
if e.milestones != nil && e.userABGroup(userID) == "A" {
    e.milestones.add(userID, FormatTopicMasteredCelebration(...))
}
```

### Gating nudges

In `Scheduler.checkUser`, skip if user is in group B. The scheduler needs access to the AB group — pass it via the store (already available as `nudgeLanguageStore`). Extend the store interface or add a new `abGroupStore` interface.

### Event tagging

Modify `logEventAsync` to automatically inject `ab_group` into every event's Data map:

```go
func (e *Engine) logEventAsync(event Event) {
    if group := e.userABGroup(event.UserID); group != "" {
        if event.Data == nil {
            event.Data = map[string]any{}
        }
        event.Data["ab_group"] = group
    }
    go func() { ... }()
}
```

This ensures ALL events (30+ types) get tagged without modifying each call site.

### Migration

```sql
-- +goose Up
UPDATE users SET config = config || '{"ab_group":"A"}'
WHERE config->>'ab_group' IS NULL;

-- +goose Down
UPDATE users SET config = config - 'ab_group';
```

### Dev command

`/dev-ab A` or `/dev-ab B` — manually override a user's group. Requires `LEARN_DEV_MODE=true`.

## Analytics Queries

```sql
-- Retention by group (7-day)
SELECT data->>'ab_group' AS grp, COUNT(DISTINCT user_id)
FROM events
WHERE event_type = 'session_started'
  AND created_at > NOW() - INTERVAL '7 days'
GROUP BY grp;

-- Average mastery gain by group
SELECT data->>'ab_group' AS grp, AVG(mastery_score)
FROM learning_progress lp
JOIN users u ON lp.user_id = u.id
GROUP BY u.config->>'ab_group';

-- Messages per session by group
SELECT data->>'ab_group' AS grp, COUNT(*)::float / COUNT(DISTINCT data->>'conversation_id')
FROM events
WHERE event_type = 'message_sent'
GROUP BY grp;
```

## Files Affected

| File | Change |
|------|--------|
| `internal/agent/store.go` | Add `GetUserABGroup`, `SetUserABGroup` to interface + MemoryStore |
| `internal/agent/store_postgres.go` | Postgres implementation |
| `internal/agent/engine.go` | `userABGroup` helper, gate milestones, AB assignment in handleStart/auto-start, inject AB group in logEventAsync |
| `internal/agent/quiz_progress.go` | Gate quiz milestone celebration |
| `internal/agent/scheduler.go` | Skip nudges for group B |
| `internal/agent/dev_commands.go` | `/dev-ab` command |
| `migrations/YYYYMMDD_ab_group_backfill.sql` | Backfill existing users to group A |
| `docs/development-timeline.md` | Mark P-W3D13-1 complete |

## Testing

- Unit: AB group assignment is 50/50 over N=1000 trials
- Unit: Group A gets milestones, group B does not
- Unit: Group B events still include `ab_group: "B"`
- Unit: Scheduler skips group B for nudges
- Unit: `/dev-ab` overrides group
- Integration: terminal-chat session with group B user shows no celebrations
