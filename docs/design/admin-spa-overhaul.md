# Admin SPA overhaul

## Status

**Approved product direction.** Production will use the simplified, single-Tutor Build AI flow and the already approved Teach and Manage school flows. The scoped authoritative-grant model described as Option 1 is also approved, including explicit tenant routes. Implementation may begin once the contract gaps at the end of this document are resolved or deliberately sequenced.

## Decision

Use one authenticated, capability-adaptive product with three separated workspaces:

- **Build AI** for AI operators
- **Teach** for teachers
- **Manage school** for school administrators

Display names personalize greetings and copy only. Scoped capabilities authorize workspaces, actions, and data; roles remain persona labels and migration defaults. A person with multiple capabilities returns to the last-used still-authorized workspace and scope, then can switch explicitly.

Build AI follows a simple Intercom Fin-style guided setup around one **Tutor**, with a GPT Builder-style live preview:

`Overview → Curriculum → Teaching → Test tutor → Publish → Activity`

This is the complete public Build AI navigation. The operator edits a **Draft**, reads **Test results**, publishes a **Published version**, and checks **Version history** and **Used by classes**. Curriculum and Teaching provide a live preview beside the controls on wide layouts, with an equivalent explicit preview flow on narrow layouts. Restricted runtime settings and platform access remain contextual utilities for separately authorized operators; they are not extra build stages.

The page-by-page content and component contract is [`build-ai-page-content.md`](build-ai-page-content.md). It records the repeated Layers analysis, peer review, must-have content, exclusions, state ownership, and illustrative component acceptance checks.

The product does not expose technical build, test, publication, or class-link records as top-level objects. Immutable revisions, evidence, educator-review records, class-version links, audit records, and recovery pointers remain internal contracts that make the simpler experience safe.

## Why the current experience is confusing

The current application treats teaching, school administration, and AI operation as navigation groups inside one admin dashboard. Every elevated role defaults to `/dashboard`; teachers can reach AI activity; school-scoped budgets sit beside platform-global AI settings; and technical tools do not form a coherent build-and-publish workflow.

The surface compounds the structural problem. Shared semantic variables coexist with hard-coded page colors, shadows, and feature-specific patterns, so related objects and actions do not look or behave consistently.

The approved overhaul fixes both problems: it separates the jobs, then makes Build AI a short flow with one Tutor instead of asking operators to manage a technical object model.

## Jobs and outcomes

### AI operator

When the Tutor needs to change, the operator needs to update its curriculum or teaching behavior, check the result in a live preview, run required tests, complete one publish-readiness checklist including educator review, publish a version, and monitor the result without exposing secrets or changing a class accidentally.

Success means the complete path is legible:

`Overview → Curriculum → Teaching → Test tutor → Publish → Activity`

### Teacher

When preparing or reviewing a class, the teacher needs to know who needs attention, why, and what safe action to take. The teacher explicitly chooses when a Published version begins serving an assigned class and may adjust bounded controls expressed in teaching language rather than model or infrastructure language.

Success means the teacher can move through:

`Today → Class → Learner or topic → Action → Adapt tutor → Preview`

### School administrator

When operating a school workspace, the administrator needs to manage people, access, classes, budgets, channels, and records without entering teaching or platform-configuration flows.

## Public Tutor model

### Tutor

Build AI presents the Tutor for the active authorized scope. It is not a library, catalog, or canvas of agents. Scope is always visible so the single-Tutor presentation never hides whether the operator is working at platform or explicit school scope.

### Draft

The editable working state for Curriculum and Teaching. Saving, previewing, testing, and educator review do not affect learners or change the version available to teachers. An edit after relevant tests or review marks that readiness evidence out of date.

### Test results

The readable outcome of testing the exact Draft. Results group passed, failed, and needs-review checks, identify evidence freshness, and exclude private learner data. They are evidence for publishing, not a separate object operators must manage.

### Published version

An immutable version made available only by the final **Publish version** action. Publishing does not change any class automatically. Teachers review and apply the Published version to an assigned class explicitly.

### Version history and Used by classes

Version history provides provenance, comparison, and recovery without exposing an internal pipeline. Used by classes shows authorized class usage and whether a class is on the latest available version. It does not reveal schools, classes, or learners outside the current grant.

### Class adaptation

A bounded teacher-owned layer for one assigned class and one Published version. It may adjust approved teaching-level settings such as language support, pacing, or class focus. It cannot change providers, secrets, safety policy, or the underlying curriculum revision. Saving and previewing an adaptation do not activate it; the teacher applies it explicitly.

## Workspace structure

### Build AI

**Overview**

- Shows the Tutor's Draft state, next incomplete step, current Published version, latest Test results, publish readiness, recent activity, and number of authorized classes using each relevant version.
- Links each incomplete item directly to Curriculum, Teaching, Test tutor, or Publish.
- Leads with the next safe action rather than a metric wall or work queue.

**Curriculum**

- Selects and pins the approved curriculum source and immutable revision.
- Shows coverage, validation, language scope, and educator-facing provenance.
- Updates only the Draft.
- Keeps the live preview available so an operator can inspect grounding and citations with synthetic scenarios.

**Teaching**

- Groups explanation style, questioning, hints, assessment, language behavior, safety boundaries, channel behavior, and teacher-adaptable bounds.
- Distinguishes operator-only controls from class-level controls in plain teaching language.
- Uses a GPT Builder-style live preview beside the structured controls without becoming a free-form prompt canvas.
- Updates only the Draft.

**Test tutor**

- Runs required suites against the exact Draft and presents Test results in one place.
- Supports an interactive synthetic preview and reproducible required checks.
- Labels failures, inconclusive checks, and stale results accurately.
- Never publishes or changes a class.

**Publish**

- Presents one readiness checklist for the exact Draft: saved Draft, approved curriculum revision, current required Test results, required safety and privacy checks, completed educator review, and recovery metadata.
- Provides contextual actions to fix an incomplete item or request educator review; it does not split readiness into technical stages.
- Enables **Publish version** only when every required item is current and satisfied.
- On success, creates the Published version, adds it to Version history, and makes it available to teachers. It does not apply it to a class.
- Shows the current Published version, Version history, and Used by classes below readiness.

**Activity**

- Shows tutor health, safeguarding and answer-quality signals, curriculum grounding, channel delivery, privacy-safe aggregate usage, budgets, recent published versions, and incidents.
- Never requires reading learner conversations to detect operational failure.
- Links an incident to the relevant Published version and recovery action without exposing internal record types as navigation.

### Teach

**Today**

- Prioritizes learners and topics needing attention over aggregate metrics.
- Shows why an item is prioritized and the next available action.
- Shows when a newer Published version is available for an assigned class; it never applies the update automatically.

**Class**

- Brings roster, topic progress, Published version in use, bounded adaptation, and recent interventions together.
- Lets an authorized teacher compare, preview, and explicitly apply an available Published version to this class.
- Uses progressive disclosure for detailed mastery and conversation evidence.

**Learner**

- Shows progress, recent learning context, and educator actions without presenting raw private conversation history by default.

**Adapt tutor**

- Uses teaching vocabulary and bounded choices.
- Shows the Published version in use, effective settings, and which controls are locked by operator or school policy.
- Requires preview before application and explains that the change affects one class.

### Manage school

- Overview
- People and access
- Classes and rosters
- School setup
- Budgets and limits
- Channels owned by the school
- Records and exports
- Audit log

Platform-global providers, credentials, policies, and publishing controls never appear here. A class administration page may show its Published version as read-only context, but teachers apply versions and adaptations from Teach.

## Role-based entry and navigation

| Scoped capability | Default workspace | Home |
| --- | --- | --- |
| `can_build_ai` | Build AI | Overview |
| `can_teach` | Teach | Today |
| `can_manage_school` | Manage school | School overview |
| Multiple capabilities | Last-used authorized workspace | Matching workspace home |
| Parent | Existing restricted parent view | Own learner summary |

If a remembered workspace or school is no longer authorized, redirect to the current authorized default and explain the change. Never infer access from a name, email pattern, tenant name, role label, or client-only navigation state.

## Approved authorization contract: scoped authoritative grants (Option 1)

- Persist capability plus platform or explicit school scope.
- Use platform-only Build AI routes for platform scope and `/schools/:schoolSlug/...` for every school-scoped Build AI, Teach, and Manage school route.
- Store the last successful target per workspace server-side and revalidate it on every resolution; never silently choose a first or default school.
- Derive Teach object access from assigned classes and designated educator-review assignments.
- Existing teacher → Teach for the current school.
- Existing administrator → Teach plus Manage school for the current school.
- Existing platform administrator → Build AI at platform scope only, with no implicit access to a school's people, classes, learners, budget, or records.
- Grant school-scoped Build AI only through an explicit reviewed grant. A separate platform authority grants platform Build AI.
- School managers may grant Teach and Manage school inside their school. No one can self-grant.
- `can_manage_ai_settings` remains a narrower platform-only grant and never follows from `can_build_ai`.

## Required states

Every workspace and first-class page must cover:

- First run and empty state
- Loading and partial data
- Recoverable API or validation failure
- Permission denied and capability removed
- Unsaved edit and cancel
- Concurrent or stale Draft
- Preview loading, ready, failed, and out of date
- Tests running, passed, needs review, inconclusive, failed, and stale
- Educator review pending, complete, changes requested, expired, and invalidated by an edit
- Publication in progress, published, failed, and recovery available
- Published version in use by no classes, one class, and many authorized classes
- Class update available, previewed, applied, conflicted, and failed without changing the current version
- Long Bahasa Melayu, English, and mixed-language labels
- Narrow viewport, keyboard-only operation, reduced motion, and increased contrast

## Design system contract

Pandai Design System 1.5 is the visual foundation. The implementation uses its semantic token names and explicit interaction states; it does not derive state colors or invent replacement brand tokens.

- Flat semantic surfaces; no decorative gradients
- Borders rather than card or container shadows
- Role-aware product tokens only where the design system defines them
- Subject colors only for subject meaning
- Accessible focus, contrast, labels, and non-color status cues
- Existing shadcn/Radix Nova primitives provide accessible behavior and composition
- shadcn visual defaults are mapped to approved Pandai tokens rather than treated as the brand
- Pandai fonts, icons, images, and other assets are not copied until reuse permissions are confirmed

Likely shadcn foundations include Sidebar, Tabs, Button, Badge, Table, Progress, Sheet, Dialog, Select, Field, Empty, Skeleton, and Sonner. Product-specific wrappers own workspace and Tutor vocabulary; low-level primitives remain generic.

The full accessibility, layout, writing, typography, color, and UI-polish requirements are normative in [`admin-spa-page-flows.md`](admin-spa-page-flows.md).

## Scope

### Included

- Capability- and scope-based entry, explicit tenant routing, navigation, and route compatibility
- Build AI Overview, Curriculum, Teaching with live preview, Test tutor, Publish, and Activity
- One Tutor with Draft, Test results, Published version, Version history, and Used by classes
- One publish-readiness checklist including educator review
- Teacher-controlled application of a Published version and bounded adaptation to an assigned class
- Teach Today, class, learner, topic, action, preview, and educator-review journeys
- Manage school reorganization without mixing in teaching or platform settings
- Shared Pandai token layer and product-specific shadcn compositions
- Empty, loading, error, permission, responsive, localization, and accessibility states
- Migration of existing routes and links without losing working capabilities

### Not included without a separate product decision

- A general-purpose free-form agent builder or a catalog of Tutors
- Teacher access to provider credentials or platform-global model policy
- Automatic curriculum publication or production following OSS main
- Automatic application of a Published version to a class
- Whole-school, multi-class, or learner-specific adaptation
- Raw learner transcript surveillance
- New engagement mechanics
- Copying unlicensed assets from the Pandai repository

## Delivery shape

1. Establish the Pandai token bridge and capability-aware workspace shell.
2. Implement scoped entry, explicit tenant routes, navigation, and route compatibility.
3. Build the approved Teach Today and class journeys on existing progress APIs.
4. Build the single-Tutor Overview, Curriculum, and Teaching pages with live preview against explicit typed contracts.
5. Add Test tutor, the Publish readiness checklist, educator review, Published version, Version history, and Used by classes.
6. Add explicit teacher application of a Published version and bounded class adaptation.
7. Build Activity and privacy-safe incident recovery.
8. Reorganize Manage school and remove duplicated legacy navigation.
9. Run accessibility, responsive, localization, failure-state, browser, type, test, and build verification before replacing current routes.

## Implementation contract gaps

The product direction is approved; these are implementation dependencies, not unresolved navigation decisions:

- Scoped grant persistence, authorization versioning, live revocation, designated-reviewer predicates, and grant-administration APIs
- Explicit school-route resolution and server-validated last-workspace/last-school preferences
- Single-Tutor Draft read/write, optimistic concurrency, immutable revision, and Version history APIs
- Approved curriculum revision, validation, coverage, and provenance contracts
- Live preview isolation, synthetic scenario, grounding citation, and sensitive-data boundaries
- Required test-suite execution, immutable Test results, cancellation, freshness, and evidence APIs
- Educator review assignment, separation of duties, expiry, requested changes, and exact-Draft binding
- Atomic and idempotent publication, immutable Published version identity, version comparison, and recovery metadata
- Used by classes queries that enforce school and assigned-class privacy boundaries
- Teacher-controlled class-version application, conflict handling, audit history, and bounded adaptation schema
- Activity, incident, budget, and privacy-safe aggregate monitoring APIs
- Export generation, expiry, sensitivity, and audit APIs
- One authoritative `groups` and membership model for onboarding, classes, and rosters
- Confirmation of which Pandai font and icon assets are cleared for use
- A real support destination for an authenticated account with no workspace access
