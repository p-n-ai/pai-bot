# Admin SPA page flows

**Status:** Approved interaction specification. The simplified single-Tutor product direction, the Teach and Manage school flows, and scoped authoritative grants Option 1 are approved. The implementation contract gaps near the end remain to be resolved or sequenced; they do not reopen the approved navigation.

## Scope and method

This document specifies the production page flow for the Pai Bot admin SPA. It covers the complete authenticated shell and three workspaces:

- **Build AI** for AI operators
- **Teach** for educators
- **Manage school** for school administration

The design follows the Layers of Product Design from strategic intent through interaction flow:

1. **Need:** move a supervised school pilot forward safely.
2. **Strategy:** separate operator, educator, and school-administration jobs, then make the safe next action obvious.
3. **Concept:** one Tutor with a Draft, Test results, a Published version, Version history, and Used by classes.
4. **Interaction:** Build AI follows **Overview → Curriculum → Teaching → Test tutor → Publish → Activity**; teachers explicitly apply a Published version to a class.
5. **Interface:** Pandai Design System 1.5 supplies visual authority; shadcn/Radix supplies accessible behavior.

The interface quality rules come from the better-accessibility, better-layout, better-writing, better-typography, better-colors, and better-ui disciplines. The current routes are mapped for migration, but they do not constrain the final information architecture.

## Product boundary

Pai Bot is an AI learning companion. It does not present itself as a human, friend, sibling, or replacement educator. Operators configure and test the Tutor, designated educators review curriculum-facing behavior, teachers explicitly apply a Published version and bounded adaptation to assigned classes, and school administrators manage school operations. No workspace inherits another workspace's authority.

Build AI presents one Tutor for the active authorized scope, not a catalog of agents or technical records. Editing, previewing, testing, and educator review affect only the Draft. **Publish version** is the only action that makes a version available. Publishing never changes a class automatically.

## Shared public language

Use these terms consistently in navigation, headings, controls, status labels, and help copy.

| Term | Meaning |
| --- | --- |
| Tutor | The single curriculum-grounded teaching AI shown in the active Build AI scope. |
| Draft | The editable Curriculum and Teaching configuration. It is not available to classes. |
| Test results | Evidence from required checks against one exact Draft. |
| Educator review | Curriculum and teaching-quality review by a designated educator who did not create the Draft. |
| Published version | An immutable Tutor version made available only through Publish. |
| Version history | Prior Published versions and their provenance, comparison, and recovery context. |
| Used by classes | Authorized classes using a Published version; never a cross-school or out-of-scope directory. |
| Class adaptation | Bounded class-specific teaching choices layered on one Published version. |
| Incident | Safety, quality, privacy, reliability, or availability event requiring response. |
| Workspace | Build AI, Teach, or Manage school. It is a job boundary, not a role hierarchy. |
| Scope | Platform, school, or assigned-class boundary in which a grant applies. |
| Grant | Revocable capability plus scope. A name or role label is never a grant. |

Use sentence case everywhere. Buttons begin with a specific verb: **Save Draft**, **Run tests**, **Request educator review**, **Publish version**, **Apply to 5 Amanah**, **Apply adaptation**, **Invite staff member**, **Download progress records**. Do not use “OK”, “Yes”, “Continue” for a consequential final action, “AI magic”, “user”, or “teacher/admin or above”.

Internal services may use immutable revision, evidence, approval, class-version binding, audit, and recovery records. Those contracts must not become primary navigation, page titles, or object-management work for operators and teachers.

## Authorization model required by the flows

The approved model is **Option 1: scoped authoritative grants with safe cutover**.

| Capability | Allowed scope | Additional predicate |
| --- | --- | --- |
| `can_build_ai` | Platform or an explicit school | The requested Tutor Draft, Test results, Published version, setting, channel, or aggregate belongs to that scope. |
| `can_teach` | Explicit school | The educator is assigned to the requested class or is the designated reviewer for the requested Educator review. |
| `can_manage_school` | Explicit school | The requested staff member, class, budget, setup record, channel, or export belongs to that school. |
| `can_manage_ai_settings` | Platform only | The deployment exposes platform settings and the principal has this narrower grant. |

Every allow decision also requires an active principal, active scope, current authorization version, and an object-level predicate. `can_build_ai` never grants learner or school-administration access. Platform scope never means “all schools”. A person acting in more than one school must select a target school before a scoped request is made.

The UI may derive `can_build_ai`, `can_teach`, and `can_manage_school` booleans for the active context, but persistence must retain scope. Local navigation state is a preference only; the backend is authoritative.

## Global shell flow

### Sign in and entry resolution

```text
Sign in
  → authenticate
  → reload current grants from the backend
  → if a safe requested URL is still authorized, open it
  → otherwise if the remembered workspace and scope are authorized, open its home
  → otherwise if exactly one workspace is available, open its home
  → otherwise use the persona migration default
       educator → Teach
       school administrator → Manage school
       AI operator → Build AI
  → if no workspace is available, show No workspace access
```

A safe requested URL includes a normalized pathname plus its original search parameters. The resolver never trusts a stored workspace, role, display name, or hidden navigation item as authority.

### Desktop shell

- First focusable element: **Skip to content**.
- Sidebar top: product mark, active workspace switcher, and explicit scope label.
- Sidebar body: only the active workspace's navigation.
- Sidebar bottom: signed-in identity, school/platform context, account menu, and sign out.
- Top bar: page breadcrumb, context switcher when needed, help, and urgent incident state.
- Main content: exactly one `main`, one page `h1`, then page-specific hierarchy.
- Workspace switching is a button or menu action. Page navigation uses links.

Only people with more than one authorized workspace see the workspace switcher. Only people with more than one authorized school see the school switcher. “Platform” and a school name are visually distinct contexts.

### Breadcrumbs and context switching

Breadcrumbs begin with the active workspace home and then show navigable ancestry: **Teach / 5 Amanah / Alya / Progress**. The current school stays in the persistent scope control rather than repeating as a crumb. Narrow screens show an immediate-parent back link while retaining the complete ordered breadcrumb for assistive technology.

Switching workspace opens that workspace home, not a remembered learner or Tutor subpage. Switching school first validates the target, then opens the current workspace home when granted. It never carries object IDs, caches, breadcrumbs, preview conversations, or form state into another school. A dirty form asks **Leave [school]?** with **Leave and discard changes** and **Keep editing**. A failed switch leaves the old context active and states that nothing changed.

### Mobile shell

- A 44px menu button opens a Radix Sheet containing the same workspace and page navigation.
- Selecting a link closes the sheet, moves focus to the page heading, and announces the new page title.
- The active workspace and scope remain visible in the compact top bar.
- Primary actions sit inside 16px inline margins and safe-area padding; they never press edge to edge.
- No persistent bottom navigation is introduced: the three workspaces contain different jobs and cannot be reduced to equal tabs.

### Access changes and session freshness

```text
Protected request returns 401
  → refresh the session once
  → if unauthenticated, sign in with a safe return URL

Protected request returns 403
  → refresh grants once
  → if the current page is no longer authorized, show Access changed
  → offer authorized workspace homes
  → never bounce through the sign-in root

Grant or scope is removed while editing
  → preserve an unsent local Draft in memory
  → disable mutation controls after the authoritative response
  → explain that access changed and where to go next
```

**No workspace access** says what happened and offers a verified school or helpdesk contact when one exists, plus **Sign out**; it never ships a dead generic contact action. **Access changed** names the unavailable workspace without exposing hidden data and offers the remaining authorized destinations. A stale bearer token must not retain a revoked grant.

### Canonical route context

Tenant scope is explicit and linkable:

- Platform Build AI: `/build-ai/...`
- School-scoped Build AI: `/schools/:schoolSlug/build-ai/...`
- Teach: `/schools/:schoolSlug/teach/...`
- Manage school: `/schools/:schoolSlug/manage/...`

A schoolless legacy link is a guarded resolver, not an unconditional redirect. If more than one authorized school is possible and no still-valid preference exists, it opens **Choose a school**. Switching school returns to the chosen workspace home and clears scoped object IDs, caches, breadcrumbs, previews, and drafts after any required unsaved-change confirmation.

### Global page inventory

| Page | Route | First takeaway | Primary destination |
| --- | --- | --- | --- |
| Sign in | `/` | Sign in to the school or platform workspace. | Authorized requested page or resolved workspace home. |
| Choose workspace | `/workspaces` | Choose what you need to do now. | Selected authorized workspace home. |
| Choose school | `/schools` | Choose the school context for this task. | Same page in the selected scope. |
| Access changed | `/access-changed` | Your access to this page changed. | An authorized workspace home. |
| No workspace access | `/no-workspace-access` | Your account is active but has no workspace grant. | Verified contact route or sign out. |
| Account | `/account` | Review identity and active sessions; no capability editing. | Session revocation or previous page. |

## Build AI

### Build AI purpose and navigation

Build AI moves an operator through one Tutor flow. The detailed content ownership, exclusions, and illustrative component acceptance criteria live in [`build-ai-page-content.md`](build-ai-page-content.md).

1. Overview
2. Curriculum
3. Teaching
4. Test tutor
5. Publish
6. Activity

Editing, live preview, testing, and educator review update readiness around the Draft but never make a version available. Only **Publish version** does that. A Published version remains unused by a class until an authorized teacher applies it in Teach.

Restricted platform runtime settings and access administration are contextual utilities linked from Overview only when authorized. They do not appear as build steps or expand the six-item navigation.

The active scope is always visible. Platform Build AI can manage platform-owned Tutor configuration. School-scoped Build AI can manage only school-owned configuration. Neither scope exposes learner identity or school-management records.

### Build AI page contracts

#### Overview — `/build-ai` or `/schools/:schoolSlug/build-ai`

**First visible:** Tutor name, Draft status, current Published version, and the next incomplete build step.

**Content order:**

1. Six-step progress with the current step and direct links.
2. Draft summary: curriculum, teaching changes, editor, and saved time.
3. Latest Test results and educator-review status.
4. Publish readiness summary with **Open Publish**.
5. Current Published version, Version history shortcut, and Used by classes count.
6. Privacy-safe health and recent activity.
7. Platform utilities when separately authorized.

The six steps use an ordered list and text states, not a decorative stepper that hides links or relies on color. The page has one primary action: the next safe incomplete step. It does not lead with metrics or a work queue.

**States:** a first-run explanation with **Set curriculum**; a loading skeleton preserving the same shape; partial-data warning; actionable retry; no Published version yet; Draft ahead of Published version; and access changed. New urgent incidents are announced politely without moving focus.

**Mobile:** sections stack in reading order. Step name, status, and action stay above supporting metadata.

#### Curriculum — `/build-ai/curriculum` or `/schools/:schoolSlug/build-ai/curriculum`

**First visible:** curriculum source, pinned immutable revision, validation state, coverage, and whether the Draft differs from the Published version.

**Content order:** source and revision; subject and level scope; language coverage; validation results; grounding examples and citations; Draft change summary.

**Actions:** **Edit curriculum**, **Save Draft**, **Discard changes**, **Open live preview**, and **Continue to Teaching**. Placeholders are examples, never labels. Saving updates only the Draft and announces “Draft saved. The Published version did not change.”

A source revision cannot silently follow an upstream branch. Production requires an approved immutable curriculum revision. Validation distinguishes errors, warnings, and incomplete coverage in text and icon. A change invalidates any Test results or educator review whose evidence no longer matches the Draft.

**Live preview:** on wide layouts, a resizable but bounded preview panel sits beside the curriculum form. It runs only approved synthetic scenarios and shows grounding citations and the Draft revision being previewed. On narrow layouts, **Open live preview** opens a full-screen route or Sheet with an explicit **Back to Curriculum** action. The preview never uses or retains a real learner conversation.

**Concurrency:** saving an old Draft opens a conflict dialog that identifies changed sections. Options are **Review latest Draft**, **Copy my changes to a new Draft**, and **Cancel**. Never silently overwrite.

#### Teaching — `/build-ai/teaching` or `/schools/:schoolSlug/build-ai/teaching`

**First visible:** the Tutor's teaching behavior and a live preview of the current Draft.

**Sections in reading order:**

1. Explanation and questioning approach
2. Hints, practice, and assessment behavior
3. Language and mixed-language support
4. Safety boundaries, uncertainty, and educator escalation
5. Channel behavior and response constraints
6. Class-adaptation bounds
7. Draft change summary

This is a structured settings page, not a free-form builder canvas. Each field explains the teaching effect. Operator-only settings and teacher-adaptable settings are named in text; lock state never relies on an icon alone.

**Actions:** **Edit teaching**, **Save Draft**, **Discard changes**, **Reset section**, **Run tests**, and **Compare with Published version**. Saving updates only the Draft and makes stale Test results or review states visible.

**GPT Builder-style live preview:** the preview remains beside the controls on wide layouts so the operator can send synthetic prompts and see Draft behavior while editing. It shows the scenario, language, curriculum citations, relevant safety behavior, and Draft freshness. Preview output is labeled synthetic and cannot be copied into learner records as evidence without an approved evidence flow. Changing a field marks the previous preview **Out of date** until rerun.

On narrow layouts the preview becomes a dedicated full-screen view with **Back to Teaching**, preserving unsaved local edits. Keyboard focus moves to the preview result heading after a run; messages are a semantic list; status is announced through a pre-existing polite live region.

#### Test tutor — `/build-ai/test` or `/schools/:schoolSlug/build-ai/test`

**First visible:** the exact Draft being tested, latest Test results, evidence freshness, and the next action.

**Content order:** required pilot suites; language and channel coverage; safety, privacy, grounding, and teaching-quality checks; optional approved scenarios; cost and time estimate; latest Test results. A suite cannot hide required safeguarding checks.

**Run flow:**

```text
Open Test tutor
  → review the exact Draft and required checks
  → Run tests
  → show durable progress by group
  → present passed, failed, needs-review, and inconclusive results
  → link each failed result to the relevant Curriculum or Teaching setting
```

Starting tests creates immutable internal evidence tied to the exact Draft, but the UI calls the outcome Test results. While running, show elapsed time and **Cancel tests** when safe. Motion is not the only progress cue.

Each result shows expected behavior, observed privacy-safe evidence, evaluator, freshness, and reproducibility details. Sensitive raw conversations are excluded; synthetic or otherwise approved evidence is clearly labeled. An unavailable or flaky evaluator is **Inconclusive**, never **Passed**. Partial results remain visible with a warning.

**Actions:** **Run tests**, **Rerun failed tests**, **Open failed result**, **Edit Curriculum**, **Edit Teaching**, and, when required checks pass, **Open Publish**. None publishes or changes a class.

Filters for result and suite update the URL. Search-empty copy names the query and offers **Clear filters**. Test results remain reachable by stable contextual URL, such as `/build-ai/test/results/:resultId` or `/schools/:schoolSlug/build-ai/test/results/:resultId`, without becoming a top-level navigation category.

#### Publish — `/build-ai/publish` or `/schools/:schoolSlug/build-ai/publish`

**First visible:** whether the exact Draft is ready to publish and the one readiness checklist.

**One readiness checklist:**

1. Draft saved and current
2. Approved curriculum revision pinned and validated
3. Required Test results passed and current
4. Required safety, privacy, language, and channel checks complete
5. Educator review complete for the exact Draft
6. Version note and recovery metadata recorded

Each row shows **Ready**, **Needs action**, **In progress**, or **Out of date** with text and icon, evidence time, and one direct repair action. The checklist is one ordered region; it is not split into separate technical stages or lanes. An edit invalidates only the evidence it actually changes, and the server recomputes readiness authoritatively.

**Educator-review flow:**

```text
Required Test results are current
  → Request educator review
  → select an authorized designated reviewer
  → review teaching and curriculum summary
  → Request educator review
  → reviewer opens the request in Teach
  → reviewer completes the review or requests changes
  → Publish checklist refreshes for the exact Draft
```

Requesting or completing review does not publish. The creator cannot review their own Draft. Expired or stale review is not ready.

**Publish flow:**

```text
All readiness items are ready
  → review Draft summary, version note, and current Published version
  → Publish version
  → server rechecks grant, Draft identity, evidence freshness, educator review, and idempotency
  → success: show the new Published version as available
  → failure: keep the previous Published version available and preserve the Draft
```

The final confirmation says: “Publish this version? It will become available to teachers, but no class will change until a teacher applies it.” The final button is **Publish version**. This is the only action that changes the available version.

After success, the page shows **Published version [identifier] is available. No classes changed.** It offers **View Used by classes**, **Open Version history**, and **Go to Activity**. It never offers an operator shortcut that silently applies the version to classes.

**Published version:** immutable identifier, published time and actor, curriculum revision, Test results summary, educator-review provenance, channels supported, and health. Internal digests may appear under technical details, not as the primary name.

**Version history:** ordered newest first with version note, published time, author, curriculum revision, comparison, availability, and authorized class-use count. A previously Published version can be selected as a recovery candidate, but making it the available version still goes through the Publish page and its current readiness policy.

**Used by classes:** authorized class name, school context, version in use, adaptation state, applied time, and teacher. Platform views use privacy-safe aggregates unless an explicit school grant permits class detail. The view never grants class access and never reveals out-of-scope schools or learners.

Stable contextual routes may include `/build-ai/publish/history` and `/build-ai/publish/used-by-classes`, with equivalent `/schools/:schoolSlug/build-ai/...` routes. Their page titles remain **Version history** and **Used by classes**.

#### Activity — `/build-ai/activity` or `/schools/:schoolSlug/build-ai/activity`

**First visible:** “Tutor is healthy” or the single most important incident. Usage charts are supporting evidence, not the lead.

**Content order:** active incidents; Published version health; safeguarding and answer-quality signals; curriculum grounding health; channel health; privacy-safe aggregate usage and budget; recent Draft, test, review, publish, and class-use events.

**Actions:** **Open incident**, **Open Published version**, **Review Test results**, **Review budget**, and **Open Used by classes**. Data uses tabular numbers and text labels. It never exposes a cross-school leaderboard, learner identity, or raw learner conversation. Platform operators see aggregate scope names only when authorized for that operational purpose.

#### Incident — `/build-ai/activity/incidents/:incidentId` or `/schools/:schoolSlug/build-ai/activity/incidents/:incidentId`

**First visible:** severity, current learner impact, affected Published version or channel, owner, and safest next action.

**Actions:** **Acknowledge incident**, **Pause Tutor service** when emergency policy allows, **Prepare recovery version**, **Add update**, and **Resolve incident**. Preparing a prior version routes through Publish; it does not silently change the available version or any class. Affected teachers receive actionable class notices and choose whether to apply an available recovery version. Resolution requires cause, privacy-safe evidence, and follow-up owner.

The timeline is ordered newest first visually but preserves a logical heading and list reading order.

#### Runtime settings — `/build-ai/settings/runtime`

Only `can_manage_ai_settings` may open or mutate this platform-only page. Provider, model, secret, and runtime flags are platform-global; the page says this before the first control. Secret values are never returned after save.

Settings are grouped by outcome rather than implementation jargon: default model route, credentials, Tutor features, and failure behavior. Each save names its scope and impact. High-risk changes require re-authentication, validation, and an audit record. A Build AI operator without this narrower grant sees no utility link and receives Access changed on direct navigation.

#### Platform access — `/build-ai/access`

This platform-only utility requires a separate grant-administration predicate. It lists active Build AI grants, scope, issuer, last use, and expiry. Possessing `can_build_ai` does not permit granting it. Grant, revoke, and last-builder recovery policies are explicit and audited.

## Teach

### Teach purpose and navigation

Teach moves an educator through **Today → Class → Learner or topic → Action → Adapt tutor → Preview**. Its navigation is:

1. Today
2. My classes
3. Educator reviews, only for designated reviewers

Learner access is always derived from an assigned class. Search cannot reveal learners outside those assignments. A teacher explicitly applies a Published version to one assigned class; no publish, school-management, or operator action silently changes that class.

### Teach page contracts

#### Today — `/schools/:schoolSlug/teach`

**First visible:** “2 things need your attention today”, followed by the safest next teaching action.

**Content order:** urgent safeguarding or access issue; Educator review request; learners needing support; topics needing attention; Published version available for a class; recent completions. Metrics sit below actionable items.

Each item states class, learner or topic when authorized, reason, evidence recency, suggested action, and destination. **Review learner**, **Review topic**, **Review Tutor**, **Review class update**, and **Open class** are links. Suggestions never claim to replace educator judgment.

Empty copy: “Nothing needs action today. Review your classes or preview the Tutor used by a class.” Actions: **Open my classes**, **Preview Tutor**.

#### My classes — `/schools/:schoolSlug/teach/classes`

**First visible:** assigned classes and their next action. Each class shows learners, Published version in use, whether a newer version is available, current or Draft adaptation, topics needing support, and last Tutor activity. **Open class** links to its workspace.

If no class is assigned, explain “No classes are assigned to you” and offer a real **Contact a school manager** destination. Class creation, join codes, roster membership, and closure belong to Manage school; Teach receives read-only roster context.

#### Class — `/schools/:schoolSlug/teach/classes/:classId`

**First visible:** class name, Published version in use, newer version availability, current adaptation, and the most important teaching need.

**Content order:** today's actions; learner needs; topic progress; Tutor version and adaptation; recent activity; read-only roster context. Tabs are used only if every panel is one view of the same class; otherwise these are nested routes with links and breadcrumbs.

**Actions:** **Review learner**, **Review topic**, **Plan class action**, **Review Tutor update**, **Adapt Tutor**, and **Preview Tutor**. The page never exposes roster mutations, join codes, provider, model, prompt, or platform safety controls.

A new Published version appears as **Version [identifier] is available** with summary and **Review Tutor update**. It never changes the class until the teacher completes the explicit application flow. An unassigned class returns Access changed without confirming that the class exists.

#### Apply Published version — `/schools/:schoolSlug/teach/classes/:classId/tutor/version`

**First visible:** current Published version, selected available version, affected class, and the statement “Nothing changes until you apply this version.”

**Content order:** teaching and curriculum change summary; current and proposed version comparison; Test results summary; educator-review provenance; curriculum compatibility; current adaptation compatibility; preview; final class effect.

**Flow:**

```text
Open Review Tutor update from an assigned class
  → compare current and available Published versions
  → preview an approved synthetic class scenario
  → review how the current Class adaptation will be preserved, changed, or blocked
  → Apply to [class]
  → server verifies assigned-class grant, available version, compatibility, and current class version
  → success: class uses the selected Published version
  → failure: class keeps its previous Published version and adaptation
```

The final button includes the class name, such as **Apply to 5 Amanah**. A conflict names the newer class state and offers **Review current class version**. Applying an older Published version uses the same explicit flow. The operation records teacher, class, previous version, new version, time, reason when required, and effective adaptation.

#### Learner — `/schools/:schoolSlug/teach/classes/:classId/learners/:learnerId`

**First visible:** learner name, class, consent and access state, current topics, and the next educator action. Data is minimized to teaching need: mastery evidence, recent approved session summaries, repeated misconceptions, and assigned work. Raw transcripts require a narrower safeguarded access policy and are never a default panel.

**Actions:** **Plan support**, **Assign practice**, **Open topic**, and **Preview Tutor for this learner** if a bounded learner-level preview exists. The interface distinguishes AI suggestion from educator judgment in text, icon, and provenance.

Empty and error states do not reveal whether an out-of-scope learner exists.

#### Topic — `/schools/:schoolSlug/teach/classes/:classId/topics/:topicId`

**First visible:** what the class understands, what needs support, and the next suggested teaching move. Evidence shows cohort distribution, misconceptions, curriculum source revision, and learners needing attention within the assigned class.

**Actions:** **Plan class support**, **Review learner**, **Adapt Tutor for this topic**, and **Preview teaching approach**. Avoid rank ordering or colored learner leaderboards.

#### Teaching action — `/schools/:schoolSlug/teach/classes/:classId/actions/new`

**First visible:** a small set of educator-owned actions: plan teacher-led support, assign practice, preview a check-in, adjust pacing, emphasize a topic, or add a teaching note. It never auto-sends or auto-applies from a suggestion.

The form starts from the selected class, learner, or topic and exposes the structured reason, evidence strength, and freshness that prompted it. Do not infer a misconception from a mastery score alone. Replace the current one-click **Nudge** with **Preview check-in**; only the reviewed preview may offer **Send check-in**, naming the learner and class. **Apply action** names the exact class and effect. Failure preserves the Draft, states that nothing was sent or applied, and explains recovery.

#### Adapt Tutor — `/schools/:schoolSlug/teach/classes/:classId/tutor/adapt`

**First visible:** Published version in use, allowed adaptation bounds, and current Class adaptation. The fixed scope message says “Changes affect [class] only.” A v1 adaptation targets exactly one authorized class; there is no whole-school, multi-class, or learner-specific adaptation.

Allowed controls come from the Published version. The bounded first set is: how to begin when a learner is stuck; language support; practice pace; and focus on topics already assigned to this class in the same approved curriculum revision. Each choice includes **Use Published version setting**. Educators cannot change curriculum source, safety or answer-check policy, provider, model, temperature, credentials, budget, channel infrastructure, or global runtime behavior.

Each control is a labeled fieldset that shows the inherited value, effective value, allowed range, teaching effect, and any “Locked by school or operator policy” reason. **Reset to Published version** is secondary and names what will be removed.

**Actions:** **Save Draft**, **Preview adaptation**, and **Discard changes**. Saving never activates. Any edit after preview marks it **Out of date** and blocks Apply until the exact Draft is previewed again. A stale Published version or concurrent adaptation opens a conflict state rather than silently rebasing.

#### Preview Tutor — `/schools/:schoolSlug/teach/classes/:classId/tutor/preview`

**First visible:** “This is a preview. No learner will receive these messages.”

The educator selects an approved synthetic scenario, language, topic, and learner support profile. The transcript compares the Published version and proposed Class adaptation when comparison helps. Grounding citations, safety behavior, and changed settings remain visible.

**Actions:** **Run preview**, **Edit adaptation**, and **Apply adaptation to [class]**. Applying opens a final summary and requires **Apply adaptation**. A failure never changes the Published version or adaptation used by the class.

Keyboard focus moves to the result heading after a run; the transcript is a semantic list, not an unlabeled chat-only visual.

#### Educator review — `/schools/:schoolSlug/teach/tutor-reviews/:reviewId`

Only the designated educator reviewer may complete the review. The Draft editor cannot review their own change even if the same account also has Teach access.

**First visible:** what will change for teaching, the curriculum scope, exact Draft, Test results freshness, and review deadline.

**Content order:** change summary; grounding revision and curriculum coverage; representative synthetic evidence; failed, needs-review, or inconclusive results; safety results; potential class impact; technical provenance under disclosure. Provider details that do not affect the teaching judgment stay disclosed, not dominant.

**Actions:** **Complete educator review** and **Request changes**. Every choice opens a consequence-specific form. Requesting changes requires actionable notes tied to Curriculum or Teaching. Completion attests to curriculum and teaching-quality review; it cannot waive required safety checks and remains tied to the exact Draft. It satisfies one Publish checklist item but does not publish or change a class.

## Manage school

### Manage school purpose and navigation

Manage school moves a school administrator through **Overview → People and access → Classes and rosters → Setup and channels → Budget → Records and audit**. It does not contain teaching queues or platform AI settings.

Navigation:

1. Overview
2. People and access
3. Classes and rosters
4. School setup
5. Budget
6. Channels
7. Records
8. Audit log

### Manage school page contracts

#### Overview — `/schools/:schoolSlug/manage`

**First visible:** the most important school setup or operational action.

**Content order:** blocked setup; access issue; expiring invite; budget risk; channel problem; consent and safeguarding readiness; recent audited changes. Small status summaries follow. **Resolve setup issue**, **Review access**, **Review budget**, and **Open channel** link to exact pages.

No learner progress, Tutor-quality work, or platform runtime control appears here.

#### People and access — `/schools/:schoolSlug/manage/people`

**First visible:** active staff and pending invitations in the selected school. Search, status, workspace grant, class assignment, and invitation filters update the URL.

Every row shows person, account status, persona label, tenant-scoped grants, class assignment count, last active time, and pending change. A display name never explains access.

**Actions:** **Invite staff member**, **Review access**, **Reissue invitation**, and **Cancel invitation**. Platform Build AI grants cannot be issued here. Search-empty copy names the query and offers **Clear filters**. A completely empty school explains why access is needed and offers **Invite staff member**.

#### Person access — `/schools/:schoolSlug/manage/people/:userId`

**First visible:** effective access for this school, how it was granted, and last use.

**Content order:** account state; Teach grant; Manage school grant; assigned classes; active sessions; audit history. Build AI grants are shown only as “Managed by platform access” without a mutation control.

**Actions:** **Save access**, **Disable account**, and **Revoke sessions**. Saving is a full replacement for this school's grants and class assignments. It never changes another school.

Removing the last active school manager is blocked transactionally, including concurrent requests. Self-removal requires another active manager and re-authentication. The repair message names the requirement: “Assign Manage school access to another active staff member before removing yours.”

#### Invite staff member — `/schools/:schoolSlug/manage/invites/new`

**First visible:** who is being invited, which school, and what they will be able to do.

**Fields:** email; persona label; Teach access; Manage school access; class assignments when Teach is selected; optional expiry. The school is read-only. The form shows an effective-access summary before sending. A school manager cannot invite or grant Build AI.

**Actions:** **Review invitation** then **Send invitation**. The final button repeats the consequence. Duplicate, expired-domain, and invalid-address errors explain repair next to the field. Submit remains enabled until the request starts.

Reissue preserves the scoped grant set and shows what changed. Cancel says **Cancel invitation** and records an audit event.

#### Classes and rosters — `/schools/:schoolSlug/manage/classes`

**First visible:** active classes, assigned educators, learner count, joining state, and setup issues for the selected school. Search and status filters update the URL.

**Actions:** **Create class**, **Open class administration**, and **Review educator access**. A completely empty school explains that classes hold rosters and use a Published version, then offers **Create class**. This page does not show learner progress, teaching recommendations, or Tutor configuration.

#### Class administration — `/schools/:schoolSlug/manage/classes/:classId`

**First visible:** class identity, active or closed state, assigned educators, roster count, joining method, and Published version in use as read-only context.

**Content order:** class details; educator assignments; learner roster membership; join code; Published version and adaptation link; audit history. **Add educator**, **Add learner**, **Move learner**, **Remove from class**, **Rotate join code**, and **Close class** are school-administration actions with object- and scope-specific confirmations.

Join codes are sensitive, hidden by default, and revealed or copied only through labeled actions. Rotating a code says the old code will stop working. Closing a class does not delete history and requires **Close class**. Teaching progress, Published version application, and adaptation remain in Teach; the page offers **Open class in Teach** only when the current person also has a scoped Teach grant for that class.

#### School setup — `/schools/:schoolSlug/manage/setup`

**First visible:** setup completion and the next incomplete requirement.

Sections: school identity and locale; curriculum sources; consent and safeguarding contacts; class setup ownership; channel readiness. Setup writes the same `groups` and membership domain used by classes and rosters; it must not create a parallel class model.

**Actions:** **Save and continue** consistently between sections and **Finish school setup** at the end. Completion routes an educator with Teach access to the created class or offers **Invite an educator**. Progress is text plus count, never color alone.

#### Budget — `/schools/:schoolSlug/manage/budget`

**First visible:** current allowance, window, used amount, remaining amount, and whether learning is at risk. Numbers are tabular. The page distinguishes observed usage from configured limits.

**Actions:** **Edit budget**, then **Save budget**. The confirmation names school, allowance, dates, and expected behavior at the limit. A manager without narrower budget mutation authority sees a read-only page with an explanation.

#### Channels — `/schools/:schoolSlug/manage/channels`

**First visible:** school channel readiness and next setup issue. Each channel shows status text, permitted origin or account, Published version in use, last check, and owner. **Configure website chat** and other configured channels open channel details.

A school manager can configure a school-owned channel only within the channel grant. Channel setup never edits the Tutor or publishes a version. Applying a Published version to a class remains a teacher action in Teach.

#### Website chat — `/schools/:schoolSlug/manage/channels/website`

**First visible:** whether chat is available, on which approved hosts, and which Published version it uses.

**Content order:** installation checklist; allowed hosts; appearance; read-only live preview; install snippet; Published version selection when separately authorized; health. Saving appearance, approving a host, selecting an available Published version for this non-class channel, and enabling chat are separate audited actions.

**Actions:** **Save appearance**, **Approve host**, **Copy install code**, **Use Published version**, **Enable website chat**, and **Disable website chat**, shown only when authorized. Enable and disable confirmations state learner impact. Copy success is announced politely. The preview is `aria-hidden` only if a complete accessible description is adjacent; otherwise its controls must be genuinely operable.

#### Records — `/schools/:schoolSlug/manage/records`

**First visible:** what can be downloaded, its school scope, sensitivity, retention, and last generated time. Exports use clear names such as **Student roster CSV**, **Conversation audit JSON**, and **Topic progress CSV**.

**Flow:** select export → review purpose, included fields, date range, and sensitivity → re-authenticate when required → **Generate [record type]** → background progress → expiring download link. Do not expose a permanent raw endpoint as the only UI. Conversation exports require the narrowest policy and explicit safeguarding purpose.

A generated export is an audit event. Errors explain whether to retry, narrow the date range, or request additional authority.

#### Audit log — `/schools/:schoolSlug/manage/audit`

**First visible:** security- and privacy-relevant changes in the selected school. Filters include date, actor, action, object, and result. Each entry states who did what, scope, timestamp, result, and correlation ID; sensitive before and after values are redacted.

Audit entries are immutable and downloadable only through the Records policy. “No activity in this period” offers **Clear filters**.

## Cross-workspace handoffs

### Build, review, publish, and apply Tutor behavior

```text
Build AI / Overview
  → Curriculum: save Draft and check grounding in live preview
  → Teaching: save Draft and check behavior in live preview
  → Test tutor: run required tests and inspect Test results
  → Publish: request educator review
Teach / Educator review
  → complete review or request changes for the exact Draft
Build AI / Publish
  → verify the single readiness checklist
  → Publish version
  → confirm that no class changed
Teach / Today or Class
  → review the available Published version
  → preview it for the assigned class
  → explicitly Apply to [class]
  → optionally adapt within bounds and preview before applying
Build AI / Activity
  → verify privacy-safe health
```

The handoff preserves stable references and back links. It never asks a reviewer to find the Draft again on a generic page, and it never applies a version to a class as a side effect of testing, review, or publishing.

### Respond to a Tutor incident

```text
Build AI / Activity
  → open incident
  → identify the affected Published version
  → pause Tutor service only when emergency policy allows
  → prepare a previously Published version through Publish
  → publish incident update
Teach / Today
  → show only actionable classroom impact
  → assigned teacher previews and explicitly applies an available recovery version when needed
Manage school / Overview
  → show only actionable school or channel impact
```

Internal recovery and rollback records preserve provenance. They do not replace the explicit Publish action or the teacher's explicit class application.

### Invite an educator and prepare a class

```text
Manage school / People and access
  → invite educator with Teach access for this school
  → assign one or more classes
  → invitation accepted
Teach / Today
  → open assigned class
  → review the Published version in use or available
  → explicitly apply, preview, or adapt within bounds
```

## Required state system

Every page must implement these states deliberately rather than through generic toasts:

| State | UX contract |
| --- | --- |
| Loading | Preserve page shape with labeled skeletons; do not replace the whole shell. |
| Empty | Say what the place is, why it is empty, and provide one forward action. |
| Filter empty | Name the search or filter and offer **Clear filters**. |
| Recoverable error | State what failed, how to retry, and preserve local input. |
| Permission denied | Do not confirm hidden object existence; route through Access changed. |
| Stale grant | Refresh once, disable mutations, preserve an unsent local Draft in memory, and offer authorized destinations. |
| Concurrent edit | Show Draft conflict and offer review or copy-to-new-Draft; never last-write-wins. |
| Stale evidence | Say which Draft edit made Test results or Educator review out of date and link to the repair action. |
| Preview | Distinguish loading, ready, out of date, failed, and privacy-blocked states without implying learner impact. |
| Publish | Recheck readiness server-side; on failure preserve the Draft and previous Published version. |
| Class application | Name class, current version, selected version, and adaptation effect; on failure preserve current behavior. |
| Offline | Keep unsent edits local, say they are not saved, and offer retry when online. |
| Success | Confirm the object and resulting state inline; live-announce routine status politely. |
| Destructive action | Name scope, object, consequence, and recovery; final button repeats the consequence. |
| Long-running work | Show durable progress, allow safe navigation, and provide a return destination. |
| Reduced motion | Use no slide or scale choreography; status still changes through text and icon. |
| Increased contrast | Reinforce borders, focus, and status without introducing new color meanings. |

## Interface quality contract

### Accessibility

- Native links navigate; native buttons act. Radix Dialog, AlertDialog, Sheet, Select, and Tabs are used only for their intended patterns.
- Every flow completes with keyboard only. Escape closes overlays; focus is trapped and restored; no positive `tabindex`.
- Focus uses `:focus-visible` with a visible 2px outline and offset. Interactive targets are at least 44×44px in touch contexts and 40×40px on desktop.
- Forms retain visible labels. Submit stays enabled until a request starts; errors use `aria-invalid`, `aria-describedby`, and focus the first invalid field.
- One `h1`, one `main`, ordered headings, descriptive links, and pre-existing live regions for dynamic status.
- Status never relies on color. Reflow works at 320px and 200% zoom without document-level horizontal scrolling.
- The live preview has a programmatic name, semantic message list, keyboard-operable actions, predictable focus, and a persistent statement that it does not affect learners.

### Layout

- Each page starts with takeaway, state, and one primary action. Supporting metrics follow.
- Use space before backgrounds and backgrounds before separators. Inter-group spacing is at least twice intra-group spacing.
- Use shared edges and logical properties. Text and controls remain inside safe margins; tables become cards or scroll within a labeled region.
- Components adapt with container queries where they are reused. Breakpoints occur when content stops fitting, not at arbitrary devices.
- Primary actions remain in stable chrome on long forms and are inset from viewport edges.
- Curriculum and Teaching use a two-region editor and preview layout only while both retain a readable measure; otherwise preview becomes an explicit full-screen view in the same reading order.

### Writing

- Plain, direct, calm, and sentence case. Address the reader as “you”; never “the user” or “we” in errors.
- One term per concept and one verb per action. Destructive confirmations repeat the consequence.
- Use Draft, Test results, Published version, Version history, and Used by classes in public copy; do not leak internal record names into headings.
- Errors explain repair next to the failure. Empty states orient and point forward.
- BM and English strings are complete localized messages, not concatenated fragments. Allow at least 40% text growth.
- Publishing copy always distinguishes “available to teachers” from “applied to a class.”

### Typography

- Use the existing legal system font stack until Pandai font reuse permission is confirmed. Do not copy Poppins files from the private mirror.
- Map Pandai type roles to semantic tokens: page title, section title, body, label, caption, and numeric data. Use no more than three type sizes and two weights on one screen where possible.
- Body copy uses generous leading and a 60–70 character measure. Numeric evidence uses tabular figures. Inputs render at 16px or larger on mobile.
- Headings descend by level; small labels never carry major hierarchy through uppercase tracking alone.
- Preview messages use the same body role and readable measure as the surrounding product; chat bubbles do not shrink evidence or citations into unreadable captions.

### Color

- `design.color.md` remains the value authority. Production code consumes Pandai semantic tokens; it does not add raw hex, HSL, or decorative color values.
- Preserve one meaning per color. Filled color is reserved for the single primary action; secondary controls remain neutral.
- Critical, warning, success, and informational states always pair color with text and icon.
- Verify rendered foreground and background pairs in every supported appearance and increased-contrast mode. Do not mechanically invert the palette for dark mode.
- Draft, stale Test results, ready to publish, Published version, and class-use states must remain understandable without color.

### UI polish and motion

- Pandai's flat, border-led surfaces override generic shadcn shadows and gradients. Borders communicate structure and state; decoration stays restrained.
- Nested radii are concentric. Icons use one Lucide set, `currentColor`, and stroke weight matched to adjacent text.
- Interactive transitions name exact properties; never `transition: all`. Routine feedback is instant or at most 150ms.
- Optional press feedback uses `scale(0.96)` only when reduced motion is not requested. No page-load choreography or repeated high-frequency animation.
- The live preview does not simulate typing for completed output, auto-scroll away from the operator's reading position, or animate stale results as if they were current.

## Route transition

Canonical routes should ship with replace redirects from current URLs after the corresponding production page exists. Search parameters and entity IDs are preserved.

| Current route | Canonical route |
| --- | --- |
| `/dashboard` | `/schools/:schoolSlug/teach` when authorized; otherwise resolved workspace home |
| `/dashboard/classes` | `/schools/:schoolSlug/teach/classes` for Teach, or `/schools/:schoolSlug/manage/classes` for Manage-only access |
| `/students/:id` | `/schools/:schoolSlug/teach/classes/:classId/learners/:id` after server-authoritative scope resolution |
| `/dashboard/ai-usage` | `/build-ai/activity` for platform Build AI, school-scoped Activity when explicitly granted, or school budget usage for Manage school; never Teach |
| `/dashboard/metrics` | Same guarded operational resolver as AI usage |
| `/settings/ai` | `/build-ai/settings/runtime` |
| `/settings/users` | `/schools/:schoolSlug/manage/people` |
| `/settings/budget` | `/schools/:schoolSlug/manage/budget` |
| `/settings/embed` | `/schools/:schoolSlug/manage/channels/website` |
| `/export` | `/schools/:schoolSlug/manage/records` |
| `/setup/onboard` | `/schools/:schoolSlug/manage/setup` |
| `/parents/:id` | Unchanged until the parent experience is separately redesigned. |

The first production shell may classify legacy routes into workspaces before every canonical page exists, but it must not show synthetic production data or imply that missing Tutor contracts exist. Generated TanStack route artifacts change only through route generation.

## Implementation contract gaps

The product and interaction direction is approved. These gaps block specific pages or states and must be resolved during implementation planning:

- Scoped grant persistence, active status, authorization version, live revocation, and grant-administration predicates
- Explicit school-route resolution, safe requested URLs, and server-validated last workspace and school preferences
- Assigned-class enforcement and designated educator-reviewer assignment
- Single-Tutor lookup per authorized scope and a migration path from any existing multiple-configuration assumptions
- Draft read/write, optimistic concurrency, immutable revision identity, comparison, and Version history APIs
- Approved curriculum revision, validation, coverage, provenance, and change-invalidation contracts
- Live preview isolation, synthetic scenario data, grounding citations, cancellation, cost controls, and retention boundaries
- Required test-suite execution, immutable Test results, pass criteria, cancellation, freshness, and evidence APIs
- Educator review separation of duties, expiry, requested changes, exact-Draft binding, and audit history
- Atomic and idempotent Publish operation, immutable Published version identity, readiness recheck, failure recovery, and recovery-version policy
- Used by classes queries with platform aggregate, school, and assigned-class privacy enforcement
- Teacher-controlled class-version application, current-version conflict detection, compatibility checks, audit trail, and failure atomicity
- Bounded Class adaptation schema tied to one class and one Published version, including preview freshness and version-change behavior
- Incident, emergency pause, notification, budget, and privacy-safe Activity APIs
- School-channel Published version selection and the authority boundary between Build AI and Manage school
- Export generation, expiry, sensitivity, and audit APIs
- One authoritative `groups` and membership model for onboarding, classes, and rosters
- Real support destination for a signed-in account with no workspace access
- Confirmation of which Pandai font and icon assets are cleared for use

## Approved scoped-grant migration (Option 1)

- Persist capability plus platform or explicit school scope.
- Put school scope in every school-scoped Build AI, Teach, and Manage school URL; use platform-only URLs for platform Build AI.
- Store the last successful target per workspace server-side and revalidate it on every resolution; never silently choose a first or default school.
- Derive Teach object access from assigned classes and Educator review assignment.
- Require step-up authentication for sensitive cross-school mutations rather than routine workspace navigation.
- Existing teacher → Teach for the current school.
- Existing administrator → Teach plus Manage school for the current school.
- Existing platform administrator → Build AI at platform scope only; no implicit access to any school's people, classes, learners, budget, or records.
- Grant Build AI to existing school administrators only through an explicit reviewed grant.
- School managers may grant Teach and Manage school inside their school. A separate platform authority grants Build AI. No one can self-grant.
- `can_manage_ai_settings` remains a narrower platform-only grant and never follows from `can_build_ai`.

This approved model keeps platform operation separate from school personally identifiable information and makes the visible scope agree with backend authority.
