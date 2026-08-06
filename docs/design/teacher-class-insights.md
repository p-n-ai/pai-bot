# Teacher Class Insights

## Summary

Add URL-scoped class selection and the existing weekly mastery-gain leaderboard to the teacher dashboard. Preserve the current all-learner view, keep progress usable when leaderboard loading fails, and align the leaderboard's learner identifier with the existing progress and student-detail contract. Make no database or authorization changes.

## Context / Current State

The dashboard route calls `fetchDashboardProgress()` without a scope. That module always requests `GET /api/admin/classes/all-students/progress`. The dashboard already supports learner search, progress filtering, nudges, and a student-detail sheet.

Existing backend capabilities are not connected to this surface:

- `GET /api/admin/groups?type=class` lists tenant-scoped classes.
- `GET /api/admin/classes/{id}/progress` accepts a real group UUID or the `all-students` sentinel.
- `GET /api/admin/groups/{id}/leaderboard` returns at most ten ranked seven-day average mastery gains.
- All three endpoints require `teacherOrAbove`; server-side tenant predicates remain authoritative.

The frontend has group and progress decoders but no leaderboard boundary contract. `GroupRecordSchema` also omits the backend `closed` field.

## Goals

- Select one active class or all learners from the existing dashboard.
- Store class and student selection in validated URL search state.
- Scope summary cards and learner progress to the selected class.
- Show the selected class's server-ranked weekly mastery gains.
- Preserve progress when the secondary leaderboard request fails.
- Prevent superseded requests from replacing current class data.

## Non-Goals

- Class create, edit, archive, delete, or roster management.
- Topic assignment or tutoring-schedule changes.
- Leaderboard scoring, ranking, window, or persistence changes.
- Aggregate leaderboard behavior for `all-students`.
- New state-management, query, or HTTP libraries.
- New backend endpoint, migration, RBAC, or tenant-policy changes.

## Invariants

1. URL search state is the single source of truth for selected class and student.
2. Missing `class` means all learners; a class ID never silently falls back to all learners.
3. Changing class clears `student` because membership in the new class is not established.
4. Only active classes appear as selectable concrete classes.
5. Leaderboard rank and ordering are server-owned; the client does not recalculate either.
6. Leaderboard `user_id` uses the same external-or-internal identifier projection as class progress and student detail.
7. `mastery_gain` is an average 0–1 mastery delta and is projected to signed percentage points for display.
8. Leaderboard failure cannot replace successful progress with a page-level error.
9. Client-selected IDs are routing inputs, not authorization. Server RBAC and tenant predicates remain authoritative.
10. A response for an older selection cannot mutate the current selection's state.
11. Student detail serializes an empty progress collection as `[]`, never `null`, so leaderboard drilldown crosses the existing frontend decoder safely.

## Design Constraints

- Route files guard, load, and wire. Display belongs in `components/dashboard`; decoding and calculations belong in `src/lib`.
- Unknown HTTP JSON must be decoded before it flows inward.
- Reuse Effect Schema, `fetchJSON`, `APIContractError`, `NativeSelect`, existing dashboard components, and current test architecture.
- Keep `routeTree.gen.ts` untouched because no route file is added or renamed.
- Local agent validation is limited to `./scripts/agent-check changed` once per coherent edit batch.

## Alternatives Considered

### Option 1: URL-scoped dashboard orchestration

The existing dashboard route parses `class` and `student`, loads the class catalog, and loads progress plus an optional leaderboard. Components receive refined state and callbacks.

```ts
type DashboardScope =
  | { readonly _tag: 'AllLearners' }
  | { readonly _tag: 'Class'; readonly classID: string }
```

Pros: preserves the current page, linkable state, local change, no new dependency.
Cons: route state becomes richer; independent requests need explicit race protection.

### Option 2: Dedicated class-insights route

Add `/dashboard/classes/$id` and keep `/dashboard` aggregate-only.

```ts
type ClassInsightsParams = { readonly id: string }
```

Pros: concrete class is guaranteed by the route; room for future class tools.
Cons: duplicates dashboard composition, adds navigation, separates closely related teacher tasks.

### Option 3: Backend-composed dashboard endpoint

Add one endpoint returning groups, selected progress, and leaderboard.

```ts
interface DashboardSnapshotDTO {
  readonly groups: ReadonlyArray<AdminGroupDTO>
  readonly progress: ClassProgressDTO
  readonly leaderboard: ReadonlyArray<AdminLeaderboardEntryDTO> | null
}
```

Pros: one request and one server snapshot contract.
Cons: unnecessary backend coupling, larger contract, duplicates working endpoints, makes partial failure harder to isolate.

## Recommendation

Choose Option 1. It completes existing contracts with the smallest coherent surface. The route already owns dashboard I/O and URL-backed student selection; class scope belongs beside it. Keep progress and leaderboard as separate state machines so the secondary capability cannot take down the primary dashboard.

Product decisions:

- Default: all learners.
- Selector: “All learners” plus active classes only.
- Persistence: URL only; do not remember a class in local storage.
- Leaderboard: concrete class only, both rank and signed gain visible.
- Drilldown: leaderboard row opens the existing student-detail sheet.
- Retry: inline leaderboard retry without reloading progress.

## Proposed Design

The route owns three independent concerns:

```ts
interface DashboardSelection {
  readonly scope: DashboardScope
  readonly studentID?: string
}

type ClassCatalogState =
  | { readonly _tag: 'Loading' }
  | { readonly _tag: 'Ready'; readonly classes: ReadonlyArray<GroupRecord> }
  | { readonly _tag: 'Error'; readonly message: string }

type ProgressState =
  | { readonly _tag: 'Loading' }
  | { readonly _tag: 'Ready'; readonly result: DashboardProgressResult }
  | { readonly _tag: 'Error'; readonly message: string }

type LeaderboardState =
  | { readonly _tag: 'UnavailableForAllLearners' }
  | { readonly _tag: 'Loading'; readonly classID: string }
  | {
      readonly _tag: 'Ready'
      readonly classID: string
      readonly entries: ReadonlyArray<LeaderboardEntry>
    }
  | { readonly _tag: 'Error'; readonly classID: string; readonly message: string }
```

`DashboardPageView` renders catalog controls and the primary progress state. `DashboardReady` renders stats, leaderboard, heatmap, and the existing student sheet. `ClassLeaderboard` owns only leaderboard projection and interaction; it performs no I/O.

## Domain Model and Types

### Search boundary

```ts
interface DashboardSearch {
  readonly class?: string
  readonly student?: string
}

function parseDashboardSearch(
  search: Readonly<Record<string, unknown>>,
): DashboardSearch

function getDashboardScope(search: DashboardSearch): DashboardScope
```

Parsing rules:

- Accept only non-empty string values after trimming.
- Absence or invalid `class` produces `{ _tag: 'AllLearners' }`.
- Do not validate UUID syntax in the browser; the server owns accepted resource identifiers.
- The sentinel is an internal projection, not written as `?class=all-students`.

### Group boundary

Change the existing schema to match the emitted backend contract:

```ts
const GroupRecordSchema = Schema.Struct({
  // existing fields
  closed: Schema.Boolean,
})
```

The selector derives active classes:

```ts
function getSelectableClasses(
  groups: ReadonlyArray<GroupRecord>,
): ReadonlyArray<GroupRecord> {
  return groups.filter((group) => !group.closed)
}
```

No second class representation is introduced.

### Leaderboard boundary

```ts
export const LeaderboardEntrySchema = Schema.Struct({
  user_id: Schema.String,
  user_name: Schema.String,
  mastery_gain: Schema.Number,
  rank: Schema.Number,
})

const LeaderboardEntriesSchema = Schema.mutable(
  Schema.Array(LeaderboardEntrySchema),
)

export interface LeaderboardEntry extends Schema.Schema.Type<
  typeof LeaderboardEntrySchema
> {}

export function readLeaderboardEntries(
  input: unknown,
): ReadonlyArray<LeaderboardEntry> | null
```

The decoder accepts an empty array. It rejects non-arrays, malformed entries, and missing fields. It does not repair rank or gain values; those are server contract defects.

### Display projection

```ts
interface LeaderboardRowView {
  readonly studentID: string
  readonly studentName: string
  readonly rankLabel: string
  readonly gainLabel: string
  readonly gainTone: 'positive' | 'neutral' | 'negative'
}

function getLeaderboardRowView(
  entry: LeaderboardEntry,
): LeaderboardRowView
```

Formatting examples:

```txt
0.08  -> +8 pts
0     -> 0 pts
-0.03 -> −3 pts
```

Use the existing numeric formatting conventions. Do not clamp, sort, or rerank.

## Types, Interfaces, and APIs

### Changed progress API

```ts
export async function fetchDashboardProgress(
  classID: string,
  fetcher: typeof fetch = fetch,
): Promise<DashboardProgressResult>
```

The caller projects `AllLearners` to `all-students` and `Class` to `classID`. Removing the default parameter prevents accidental aggregate fetches.

### New leaderboard API

```ts
/** Fetches and decodes the server-ranked seven-day leaderboard for one class. */
export async function getGroupLeaderboard(
  classID: string,
  fetcher: typeof fetch = fetch,
): Promise<ReadonlyArray<LeaderboardEntry>>
```

Execution:

```ts
const payload: unknown = await fetchJSON(
  `/api/admin/groups/${encodeURIComponent(classID)}/leaderboard`,
  fetcher,
)
const entries = readLeaderboardEntries(payload)
if (!entries) throw new APIContractError('Invalid group leaderboard response')
return entries
```

Expected HTTP and contract failures continue through existing `AdminAPIError` and `APIContractError`. The route catches `unknown` at the recovery boundary and projects safe UI copy.

### View contracts

```ts
interface DashboardPageViewProps {
  readonly catalogState: ClassCatalogState
  readonly leaderboardState: LeaderboardState
  readonly progressState: ProgressState
  readonly selectedClassID?: string
  readonly selectedStudentID?: string
  readonly onSelectClass: (classID: string | undefined) => void
  readonly onRetryLeaderboard: () => void
  readonly onSelectStudent: (studentID: string) => void
  // existing nudge and close callbacks
}

interface ClassLeaderboardProps {
  readonly state: LeaderboardState
  readonly onRetry: () => void
  readonly onSelectStudent: (studentID: string) => void
}
```

## Seams, Boundaries, Adapters, and Implementations

| Seam | Owner | Input | Output | Must not own |
|---|---|---|---|---|
| TanStack search params | Dashboard route | `Record<string, unknown>` | `DashboardSearch` | API authorization |
| HTTP JSON decoding | `src/lib/*-types.ts` | `unknown` | Refined group, progress, leaderboard values | UI state or rendering |
| Admin HTTP adapter | `admin-api.ts` | Refined class ID | Decoded result or existing typed error class | Ranking or display formatting |
| Dashboard orchestration | Dashboard route | Scope, retry version | Independent catalog/progress/leaderboard state | SQL semantics |
| Display projection | `dashboard-leaderboard-view.ts` | `LeaderboardEntry` | `LeaderboardRowView` | Fetching or navigation |
| UI rendering | Dashboard components | Refined view props | Accessible DOM and callbacks | Parsing unknown input |
| Authorization and tenancy | Existing Go handlers/service | Authenticated request and class ID | Tenant-scoped response/error | Client filters |

The existing Go leaderboard projection changes one selected field from internal UUID to `COALESCE(NULLIF(u.external_id, ''), u.id::text)`, matching the established student-detail boundary. No new backend adapter or persistence seam is required.

## Call Stacks and Data Flow

### Current / Old Flow

```txt
browser /dashboard?student=...
  -> DashboardRoute
  -> parseDashboardSearch(unknown)
  -> fetchDashboardProgress()
  -> getClassProgress('all-students')
  -> fetchJSON('/api/admin/classes/all-students/progress')
  -> Go teacherOrAbove middleware
  -> handleAdminClassProgress
  -> tenant-scoped progress query
  -> JSON unknown
  -> readClassProgress
  -> DashboardProgressResult
  -> DashboardPageView
  -> DashboardReady(stats + heatmap + student sheet)
```

### Proposed / New Flow

#### Initial aggregate dashboard

```txt
browser /dashboard
  -> DashboardRoute
  -> parseDashboardSearch(unknown)
  -> DashboardScope.AllLearners
  -> parallel owned effects
       -> listGroups('class') -> decode GroupRecord[] -> active selector options
       -> fetchDashboardProgress('all-students') -> decode ClassProgress
  -> LeaderboardState.UnavailableForAllLearners
  -> DashboardPageView
```

#### Select one class

```txt
NativeSelect change(classID)
  -> onSelectClass(classID)
  -> navigate({ search: { class: classID, student: undefined } })
  -> route observes new parsed search
  -> request generation increments
  -> parallel owned effects
       -> fetchDashboardProgress(classID)
          -> GET /api/admin/classes/{classID}/progress
          -> teacherOrAbove + tenant-scoped group progress
          -> unknown -> readClassProgress -> DashboardProgressResult
       -> getGroupLeaderboard(classID)
          -> GET /api/admin/groups/{classID}/leaderboard
          -> teacherOrAbove + tenant-scoped seven-day query
          -> unknown -> readLeaderboardEntries -> LeaderboardEntry[]
  -> current-generation results update independent states
  -> DashboardReady
       -> DashboardStats
       -> ClassLeaderboard
       -> DashboardHeatmap
```

#### Open learner from leaderboard

```txt
leaderboard row action(studentID)
  -> existing onSelectStudent(studentID)
  -> navigate({ search: { class: currentClassID, student: studentID } })
  -> progress request does not restart solely for student change
  -> StudentDetailSheet opens
  -> existing StudentDetailPanel I/O
```

### Failure Flow

```txt
class catalog HTTP/contract failure
  -> ClassCatalogState.Error
  -> selector-local error
  -> aggregate progress remains usable

progress HTTP/contract failure
  -> ProgressState.Error
  -> existing page-level “Class data unavailable” state

leaderboard HTTP/contract failure
  -> LeaderboardState.Error(classID, safe message)
  -> inline alert + retry button
  -> successful progress remains rendered

unknown/inaccessible class ID
  -> server 4xx
  -> no aggregate fallback
  -> selected scope remains visible with failure state
```

### Retry / Cancellation / Idempotency Flow

These requests are read-only, so retries do not create mutation or idempotency concerns.

```ts
type RequestGeneration = number

selection changes
  -> generation += 1
  -> capture generation for progress and leaderboard effects
  -> await request
  -> if capturedGeneration !== currentGeneration: discard result
  -> otherwise commit state

leaderboard retry
  -> leaderboardRetryVersion += 1
  -> retain current progress
  -> set LeaderboardState.Loading(currentClassID)
  -> issue one new request under current generation
```

Every promise is awaited by its owning effect chain and has rejection handling. Cleanup marks the generation inactive. No automatic retry or backoff is introduced.

### Observability Flow

No new telemetry pipeline is introduced. User-visible state is the diagnostic surface:

```txt
HTTP/contract error
  -> existing safe Error.message projection
  -> page-level progress alert OR leaderboard-local alert
```

Do not log response bodies, learner data, class names, or raw errors. Existing server request handling remains unchanged.

## Files to Add / Change / Delete

### Add

| File | Responsibility |
|---|---|
| `admin-spa/src/lib/leaderboard-types.ts` | Effect schemas, refined leaderboard type, unknown-input decoder |
| `admin-spa/src/lib/leaderboard-types.test.ts` | Accepted and rejected wire shapes |
| `admin-spa/src/lib/dashboard-leaderboard-view.ts` | Pure signed-gain and row projection |
| `admin-spa/src/lib/dashboard-leaderboard-view.test.ts` | Positive, zero, and negative display behavior |
| `admin-spa/src/components/dashboard/class-leaderboard.tsx` | Accessible leaderboard states and learner action |

### Change

| File | Responsibility |
|---|---|
| `admin-spa/src/lib/group-types.ts` | Decode backend `closed` field |
| `admin-spa/src/lib/group-types.test.ts` | Prove closed-state decoding |
| `admin-spa/src/lib/admin-api.ts` | Add `getGroupLeaderboard` |
| `admin-spa/src/lib/admin-api.test.ts` | Prove URL, empty success, HTTP failure, malformed response |
| `admin-spa/src/lib/dashboard-progress.ts` | Require explicit class ID |
| `admin-spa/src/lib/dashboard-progress.test.ts` | Prove selected ID propagation and aggregate projection |
| `admin-spa/src/routes/_authenticated/dashboard.index.tsx` | Parse URL scope; own catalog, progress, leaderboard, retry, and race protection |
| `admin-spa/src/components/dashboard/dashboard-page-view.tsx` | Render selector and independent top-level states |
| `admin-spa/src/components/dashboard/dashboard-ready.tsx` | Compose leaderboard before heatmap and preserve student sheet |
| `admin-spa/src/routes/_authenticated/-dashboard.test.tsx` | Prove user-visible selection, leaderboard, isolation, and drilldown behavior |
| `internal/adminapi/groups.go` | Project canonical learner IDs from the existing leaderboard query |
| `internal/adminapi/service_test.go` | Prove leaderboard query uses the canonical learner ID projection |
| `internal/server/handler.go` | Normalize empty student-detail progress at the HTTP boundary |
| `internal/server/handler_test.go` | Prove empty progress serializes as a non-null array |

### Delete

None.

### Config, migration, runtime

None.

## RGR TDD Test Plan

Use vertical red-green-refactor slices. Extend established Vitest files and Testing Library style; do not introduce module mocks beyond existing seams, snapshots, fixtures systems, or E2E infrastructure.

### Slice 1: Parse leaderboard responses

**Red:** `leaderboard-types.test.ts` expects a valid array to decode, an empty array to remain empty, and malformed entries to return `null`.
**Green:** add the Effect schemas and decoder.
**Refactor:** keep wire decoding and refined type co-located.

### Slice 2: Fetch one class leaderboard

**Red:** `admin-api.test.ts` proves the encoded class endpoint, decoded response, empty response, non-OK propagation, and contract failure.
**Green:** add `getGroupLeaderboard`.
**Refactor:** reuse `fetchJSON` and existing error classes; no wrapper abstraction.

### Slice 3: Make progress scope explicit

**Red:** update `dashboard-progress.test.ts` to require and propagate a concrete ID; retain an explicit `all-students` case.
**Green:** remove the default and pass the ID to `getClassProgress`.
**Refactor:** update only known callers.

### Slice 4: Select a class through the URL

**Red:** dashboard behavior proves active class options, default aggregate selection, navigation to `?class=<id>`, and clearing `student`.
**Green:** load groups, parse `class`, wire `NativeSelect`, and navigate.
**Refactor:** extract pure search/scope projection only if the route becomes difficult to read.

### Slice 5: Render class leaderboard

**Red:** component behavior proves unavailable, loading, ranked ready, empty, and failure states; clicking a learner invokes the existing student-selection callback.
**Green:** add row projection and `ClassLeaderboard`.
**Refactor:** separate pure numeric presentation from JSX.

### Slice 6: Isolate failure and stale responses

**Red:** prove leaderboard failure leaves progress visible; prove an older class response cannot replace the latest selection; prove inline retry reloads only leaderboard.
**Green:** add independent state and request generation ownership.
**Refactor:** keep effects explicit; do not introduce a generic request-state framework.

### Slice 7: Bounded validation

Run once after the coherent implementation batch:

```bash
./scripts/agent-check changed
```

CI remains responsible for validation outside the changed-file guardrail. Browser E2E and production API behavior remain unproven unless explicitly requested.

## Risks and Open Questions

### Resolved decisions

- Closed classes are excluded from selection.
- All learners remains the default.
- Selection persists only in the URL.
- Leaderboard opens the existing student sheet.
- Rank and signed percentage-point gain are both displayed.
- Server rank/order is preserved exactly.
- Secondary failure is isolated from progress.

### Risks

1. **Contract alignment:** making `closed` required may reveal stale frontend fixtures. Fix fixtures to match the real backend response; do not make the field optional merely to preserve invalid test data.
2. **Race conditions:** class changes can resolve out of order. Request-generation tests are required before completion.
3. **Misleading gain copy:** `mastery_gain` is an average across topics, not total improvement. UI copy must say mastery gain without implying summed points.
4. **Stale linked class:** bookmarked class IDs may later be closed or removed. Preserve the requested scope and show the server failure; never substitute aggregate data silently.
5. **Test-policy budget:** use the smallest existing test surfaces. If the legitimate vertical slices exceed the focused profile, stop and request the appropriate expanded profile instead of weakening coverage.

No unresolved product or architecture questions remain.
