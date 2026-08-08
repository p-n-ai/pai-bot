# Build AI page content specification

**Status:** Approved design contract for the illustrative component. This document records the result of repeated Layers analysis and peer review across Overview, Curriculum and Teaching, Test tutor and Publish, Activity, and cross-page simplification.

## Decision

Keep the six approved Build AI destinations:

1. Overview
2. Curriculum
3. Teaching
4. Test tutor
5. Publish
6. Activity

They are not six completion steps. The build path is **Curriculum → Teaching → Test tutor → Publish**. Overview resumes the operator at the correct action. Activity monitors the Tutor after publication.

Do not collapse the information architecture to Home / Configure / Verify / Release / Monitor in this component pass. That remains a later usability hypothesis. Keep the route label **Activity** for now and make **Tutor status** its first section.

## Evidence and confidence

### Observed

- Direct product feedback says the current mixed admin experience is confusing.
- The approved direction is one Tutor with a private Draft, current Test results, independent Educator review, immutable Published versions, Version history, and explicit teacher application to a class.
- Current production APIs do not yet provide Draft, test evidence, review, publication, class-version, incident, or scoped-grant contracts.
- Current usage data cannot prove Tutor health.
- No analytics, task-frequency study, or usability recording has been provided.

### Inferred

- Operators need to distinguish private work from what is available to teachers, identify the first real blocker, inspect trustworthy evidence, publish deliberately, and notice exceptions without reading learner conversations.
- Simplicity comes from strict page ownership and direct repair paths, not from hiding non-pass states or duplicating summary cards.

### Assumed for the supervised pilot

- One Draft selects one approved curriculum bundle and immutable revision.
- Preview may use unsaved local values, but tests, review, and publication use the exact saved Draft.
- English, Bahasa Melayu, and supported mixed-language replies are the initial learner-language set.
- Class adaptations use bounded choices; there is no free-form instruction.
- A behavior cannot be presented as verified until implementation and a matching test exist.

The component must label all illustrative evidence as synthetic and must not imply that missing production contracts exist.

## Public conceptual model

Use only these user-facing nouns:

- **Tutor** — the one teaching AI being configured.
- **Draft** — private curriculum and Teaching changes.
- **Test results** — reproducible evidence for the exact saved Draft.
- **Educator review** — independent teaching and curriculum judgment.
- **Published version** — immutable version available to teachers.
- **Version history** — prior immutable Published versions and provenance.
- **Used by classes** — authorized class use of Published versions.
- **Incident** — operational problem requiring a response.

Draft and Published versions coexist. Starting, restoring, or editing a Draft never removes a Published version or its class-use history.

Do not use a single `published` boolean. The illustrative state model must separately represent:

```text
Draft
  saved revision
  unsaved changes
  selected curriculum revision
  Teaching settings
  Test-result binding and freshness
  Educator-review binding and freshness

Published versions[]
  immutable identifier
  version note
  publisher and time
  curriculum revision
  evidence/review provenance
  authorized class-use count
  latest-available relationship
```

Separate these three meanings of language:

1. Interface language — an account/application concern.
2. Curriculum language coverage — shown on Curriculum.
3. Tutor reply language — configured on Teaching and limited by curriculum coverage.

## Cross-page ownership

| Concern | Owning page | Summary elsewhere |
| --- | --- | --- |
| Next required action | Overview | Nowhere else duplicates the resume logic. |
| Curriculum source, revision, validation, coverage, citations | Curriculum | Overview shows one compact provenance line. |
| Tutor behavior, locked safeguards, class-adaptation bounds | Teaching | Overview shows saved state/change summary. |
| Exact-Draft reproducible evidence and repair | Test tutor | Overview and Publish show status/blockers only. |
| Readiness, Educator review, publication, Published version, Version history, Used by classes | Publish | Overview shows current relationship; Activity links back. |
| Serving status, incidents, recovery entry, privacy-safe operational context | Activity | Overview shows only the top supported exception. |
| Platform provider/runtime configuration | Runtime settings | Conditional link only with the narrower grant. |
| School channel setup and budget mutations | Manage school | Activity may link when the same scoped grant allows it. |
| Teacher application and class adaptation | Teach | Publish shows authorized aggregate use only. |

## Shared shell and page structure

- Stable context names the active workspace and explicit platform or school scope.
- Sidebar uses real links and route state. Browser Back, deep links, and focus restoration work.
- On mobile, use the existing off-canvas shell. Do not stack workspace selection plus all six navigation items ahead of the page.
- Every page has one `main`, one `h1`, one integrated Tutor-state sentence, and at most one filled primary action when work is genuinely required.
- Do not repeat the same state as a badge, banner, card, footer, and toast.
- Consequential success remains inline. Toasts are supplementary only.

## Overview

### Overview job

Answer in one scan:

1. What is private in the Draft?
2. Which Published version is available to teachers, and which authorized classes use it?
3. What is the next safe action?

Overview is read-only. It is not an editor, preview, metric dashboard, or release surface.

### Overview first visible content

- Breadcrumb: **Build AI / Overview**.
- `h1`: the Tutor’s actual name.
- One integrated state sentence combining Draft save state, latest Published version, authorized class use, the first blocker or supported incident, and freshness.
- One filled action only when the server-authoritative state requires work.

Examples:

- **Draft saved. Published version 3 remains available to teachers, and four authorized classes use it. Test results are out of date after Tutor reply language changed.** → **Run tests**
- **No Published version is available to teachers yet. Classes are unchanged. Set the approved curriculum first.** → **Set curriculum**
- **The Draft matches Published version 3. Four authorized classes use it. Tutor setup is up to date.** → no manufactured primary action

### Next-action precedence

1. Fresh supported urgent Incident affecting a Published version in use → **Open incident**.
2. Curriculum absent, unapproved for use, withdrawn, or invalid → **Set curriculum** or **Review curriculum**.
3. Teaching unsaved, invalid, or missing an explicitly required value → **Review teaching**.
4. Test results absent/out of date/cancelled → **Run tests**; running → **View test progress**; Failed/Needs review/Inconclusive → **Review Test results**.
5. Required tests current but review/readiness incomplete → **Open Publish**.
6. All readiness items satisfied → **Open Publish**.
7. No unpublished changes and no exception → no filled primary action.

The server supplies canonical blocker/action codes. The client does not infer readiness from timestamps or whether a control was touched.

### Overview content order

1. Integrated state and action.
2. Terse linked map of all six destinations. Overview is **Current**; Activity is **Monitor**. No percentage or second stepper.
3. One compact Tutor-state block:
   - Curriculum provenance and validation
   - Teaching saved state/change summary
   - Test-result state and invalidating edit
   - Educator-review and Publish-readiness blocker
4. Published-version relationship and neutral links to **Version history** and **Used by classes**.
5. One supported health sentence or top Incident, plus at most three significant privacy-safe changes.
6. Conditional Runtime settings and Platform access links.

### Overview exclusions

- Live preview or editable controls
- Publish, Apply, restore, pause, or recovery mutation
- Metrics wall, provider/model mix, token counts, per-learner cost, or budget cards
- Learner identity, raw conversations, out-of-scope class names, or cross-school detail
- Duplicate blocker cards, percentage progress, decorative stepper, or badge pile
- Unsupported Healthy, Complete, Ready, or current-state claims

## Curriculum

### Curriculum job

Select and understand the exact approved curriculum revision used by the Draft, confirm its coverage and provenance, and inspect grounding without editing source files.

### Curriculum first visible content

One sentence integrates source, aligned subject/level, immutable curriculum revision, approval-for-use, validation, and Draft-versus-Published difference.

Use **aligned to KSSM**, not **official KSSM source**, unless stronger provenance is verified.

### Curriculum content order

1. Source and curriculum revision
   - human source name
   - KSSM alignment, subject, level
   - friendly immutable revision label
   - approved-by/date
   - exact snapshot identifier under disclosure
2. Approved bundle scope; v1 does not cherry-pick individual objectives.
3. Curriculum language coverage.
4. Validation and coverage:
   - topics/objectives supported
   - missing/warning areas
   - teaching notes, examples, and assessments present
5. Grounding examples with synthetic scenario, Tutor excerpt, and descriptive citations.
6. Draft change summary and evidence made out of date.

### Curriculum actions

- **Change curriculum**
- **Save Draft**
- **Discard changes**
- **Run preview**
- **Continue to Teaching**
- **Compare with Published version N**, when available

Validation runs automatically when selecting or saving. **Run validation again** is a secondary repair action only when a real validation job exists.

### Curriculum exclusions

- OSS/YAML editing, file paths, hashes as primary content, Git controls, embeddings, retrieval weights, or indexing tools
- Teacher class resources or learner data
- Topic inclusion/exclusion without a separately approved model
- Provider/model/key/runtime controls
- A no-op **Validate curriculum** action beside already validated copy
- Validation presented as educator approval or official government endorsement

### Curriculum states

No selection; no approved revision; checking; validated; validated with warnings; invalid; incomplete; withdrawn; partial coverage; save conflict; offline; access changed. A withdrawn revision stays identifiable but blocks new publication and offers a repair path.

## Teaching

### Teaching job

Set bounded teaching preferences, understand locked behavior, define what teachers may adapt, and preview the exact local choices without exposing a prompt canvas.

### Editable v1 preferences

1. **How the Tutor starts**
   - Adapt to the learner
   - Start with one guiding question
   - Start with one small worked example
   - Start with a short explanation
2. **Typical response detail**
   - Adapt to the learner
   - Brief
   - Balanced
   - More detailed when useful
3. **Tutor reply language**
   - Follow the learner’s English, Bahasa Melayu, or natural mix
   - Prefer English
   - Prefer Bahasa Melayu
4. **Tone**
   - Calm and clear
   - Warm and encouraging
   - Direct and concise

Each setting is a labeled fieldset. Helpers state that explicit learner requests, demonstrated progress, evidence, and hard safeguards take priority.

### Visible locked behavior

- Hints, practice, and assessment
- AI identity and uncertainty
- Approved curriculum boundary
- Safety and escalation behavior only when an implemented contract and matching test exist
- Enabled-channel response constraints, with channel setup owned elsewhere

State who controls each locked item and why. Do not use a lock icon alone.

### Class-adaptation bounds

Operators may allow named choices for one assigned class:

- help-first preference
- pacing/detail
- reply-language preference within supported coverage
- current-topic emphasis inside the approved curriculum revision

There is no free-form class instruction in v1.

### Teaching exclusions

- System prompts or free-form behavior instructions
- Temperature, token, model, provider, credential, or budget controls
- Learner-specific behavior
- Safety-off toggles
- Arbitrary channel instructions
- Claims that wellbeing escalation or another safeguard is active without implementation and evidence

## Curriculum and Teaching preview

Persistent preview belongs only on Curriculum and Teaching.

### Preview interaction

1. Select an approved synthetic scenario, topic, and language.
2. Choose **Run preview**.
3. Preview uses current local values, including unsaved changes, without saving.
4. Header states the exact binding and consequence: **Synthetic preview using unsaved Teaching changes and curriculum revision 2026.08. Learners are not affected.**
5. A later field change keeps the result visible but marks it **Out of date**. It never silently rewrites the answer.
6. **Run preview again** produces a new explicit result.

Preview is exploratory. It never becomes Test results, Educator review evidence, or learner history.

Use approved scenarios until privacy detection and retention exist. Do not provide a fake free-form message field. Citations use descriptive text, not `[C1]` alone.

### Preview layout

- Wide: bounded editor plus preview at readable widths; optional sticky preview.
- Narrow: dedicated full-screen preview destination with **Back to Curriculum/Teaching**. Browser Back works and unsaved local edits remain in memory.
- Never append preview after the entire long mobile editor.

### Preview states

Not run; running; ready; out of date; failed; cancelled; privacy blocked; access changed. Failure keeps the previous result and states it is unchanged.

## Test tutor

### Test tutor job

Generate trustworthy, reproducible evidence for the exact saved Draft, make every non-pass state honest, and route directly to repair.

### Test tutor first visible content

Exact saved Draft summary, saved time/actor, curriculum revision, changed sections, result freshness, honest status counts, and one next action.

### Test-result states

- Running
- Passed
- Failed
- Needs review
- Inconclusive
- Out of date
- Cancelled

A missing/flaky evaluator is Inconclusive, never Passed. Any required non-pass blocks Publish. Educator review cannot waive required safeguards.

### Required groups

1. Curriculum grounding
2. Teaching behavior
3. English, Bahasa Melayu, and supported mixed-language behavior
4. Safety and wellbeing only where implemented
5. Privacy boundaries
6. Enabled-channel constraints

Each group shows required/optional, counts by status, expected behavior, privacy-safe observed evidence, synthetic/approved source label, evaluator method, time/freshness, reproduction details under disclosure, and direct **Edit Curriculum**, **Edit Teaching**, or **Rerun** repair.

### Test tutor interaction

- Unsaved Draft → **Open Teaching to save**.
- Saved Draft without current results → **Run tests**.
- Running → durable group progress, elapsed time, optional **Cancel tests**, and safe navigation away/return.
- Partial failure → retain completed results and mark unfinished checks Inconclusive.
- Failed/stale → repair link is primary.
- Passed/current → **Open Publish** primary; **Run tests again** secondary.

### Test tutor exclusions

- Persistent chat preview
- Lone green score such as 12/12
- Real learner conversations or school records
- System prompts, provider/model, credentials, queue/run IDs, or hashes
- Manual mark-safety-passed control
- Test, review, and publish combined into one action

## Publish

### Publish job

Show one authoritative readiness gate, coordinate independent Educator review, publish deliberately, preserve prior versions, and make the no-class-change consequence unmistakable.

### Readiness checklist

One semantic ordered list. Each row has **Ready**, **Needs action**, **In progress**, or **Out of date**, an evidence time, short proof, and one repair destination.

1. **Draft saved and current**
2. **Curriculum revision approved and validated**
3. **Teaching and grounding Test results passed and current**
4. **Non-waivable safeguards and coverage passed and current**
5. **Educator review complete**
6. **Version note and recovery ready**

The operator supplies a plain Version note. Technical recovery provenance is recorded automatically and appears only on an actionable exception.

The server recomputes readiness. The client does not use a green score or local timestamps as authority.

### Educator review

- Request only after required Test results are current.
- Select an authorized designated reviewer; the Draft creator is excluded.
- Reviewer completes or requests changes in Teach on a stable exact-Draft link.
- Any Draft edit makes review Out of date.
- Educator review covers teaching and curriculum quality; it cannot waive non-waivable safeguards.
- Publish lets the operator request and track review, but never **Complete educator review**.

### Publication

When all rows are ready:

1. Add Version note.
2. **Review publication**.
3. Confirmation states: **Publish this version? It will become available to teachers, but no class will change until a teacher applies it.**
4. Final action: **Publish version**. Do not predict the version identifier.
5. Wait for authoritative success.

Success: **Published version [identifier] is available. No classes changed.**

Failure preserves the Draft and every prior Published version, resolves an unknown outcome before retry, and links to the exact repair.

### Version history and recovery

Newest first, with Published-version identifier, Latest/Earlier wording, Version note, publisher/time, curriculum revision, Test/review provenance, and authorized class-use count.

Routine action: **Start Draft from this version**. Confirmation states that the latest Published version remains available and no class changes. An existing Draft is never silently overwritten.

Incident action: **Prepare recovery Draft**. It creates a private Draft with Incident and prior-version provenance, then follows the same tests, review, and Publish gate. It never instantly rolls back or applies to a class.

## Activity

### Activity job

Answer whether the serving Tutor needs action, what is affected, and where the safe repair lives. Activity is a status and Incident page, not an analytics dashboard or raw event log.

### Activity first visible content

- `h1`: **Activity**
- First section: **Tutor status**
- One supported outcome with freshness and at most one primary **Open incident** action.

Status precedence:

1. Paused
2. Critical active Incident
3. Degraded
4. Monitoring incomplete or stale
5. Healthy only when every required signal is fresh and complete

No traffic is neutral. Missing or stale monitoring can never render Healthy.

### Activity content order

1. Tutor status
2. Active Incidents, only when present
3. Compact Published-version context with links back to Publish/Test
4. Only backed live checks:
   - safeguarding
   - answer quality
   - curriculum grounding
   - channel delivery
5. Supporting volume/budget context only when useful and scoped
6. Five to ten recent significant operational events

Do not render an unavailable card for every unsupported signal. When no trustworthy monitoring contract exists, show one honest **Monitoring is not connected** state.

### Incident detail

First visible: severity/state, privacy-safe aggregate impact, affected Published version/channel, owner, evidence freshness, and safest next action.

Permission-gated actions:

- **Acknowledge incident**
- **Pause Tutor service**, only under a narrow emergency policy and likely step-up authentication
- **Prepare recovery Draft**
- **Add update**
- **Resolve incident** with cause, sanitized evidence, and follow-up owner

Incident and service states remain independent: resolving does not resume; resuming does not resolve.

### Activity exclusions

- Duplicate Version history or full Test evidence
- Provider/model table, token taxonomy, top provider, or per-learner averages
- Learner identities, rosters, chat handles, raw prompts/replies/transcripts, or exact learner-linked timestamps
- Cross-school rankings or drilldowns
- Tiny-cell rates enabling re-identification
- Credentials, prompts, raw logs, trace IDs, stack traces, or request payloads
- Unsupported green health or synthetic production claims

## Teach and Manage school handoffs

- Teach receives a notice only when classroom action is required. It shows class impact, the available recovery Published version, and **Review recovery version** → compare/preview → **Apply to [class]**.
- Manage school receives only actionable school/channel/budget impact with direct destinations.
- Build AI does not apply a Published version to a class.
- Non-class channel version authority remains an implementation decision; do not imply a result in the component.

## Shared states

Every page deliberately implements or documents:

- shape-preserving loading
- first run and meaningful empty state
- partial data and per-section retry
- recoverable error with preserved input
- offline/unsent state
- stale evidence with invalidating edit
- concurrent Draft/version conflict
- access changed without hidden-object disclosure
- long-running progress and idempotent retry
- persistent consequential success
- reduced motion and increased contrast

## Interface requirements

- Links navigate; buttons act.
- Stable component state; never replace the whole `main` on every input.
- Preserve keyboard focus after field and scenario changes.
- One scoped polite live region; do not make the entire conversation `aria-live`.
- Visible `:focus-visible`, 44px touch targets, 40px desktop targets, 16px mobile input text.
- 320px and 200% zoom reflow without document-level horizontal scrolling.
- Allow at least 40% English/BM copy growth.
- Status always uses text and icon in addition to color.
- Flat Pandai semantic-token surfaces, no gradients or decorative shadows.
- Group with space before separators; reduce border-heavy card grids.
- Do not use white normal-size text on the current bright green primary fill without a measured accessible pairing. The illustrative component must use a token pairing that reaches WCAG AA.

## Illustrative component acceptance checks

The component may use synthetic data only when it visibly says so. It must demonstrate:

- explicit platform/school scope
- Draft and Published version coexisting
- Published history remaining after starting another Draft
- real linkable page state through `?page=` or routes
- Overview without preview and without a manufactured CTA
- Curriculum/Teaching-only explicit preview with Out of date state
- bounded Teaching fieldsets and no free-form prompt boxes
- Test states beyond Passed, direct repair, and no persistent chat
- Publish checklist, external Teach review handoff, consequence confirmation, and no predicted version number
- Publish success that changes availability but no class
- Version history and **Start Draft from this version** without overwriting
- Activity led by honest monitoring state, not a fake green or duplicate history
- navigation and field interactions that preserve focus
- mobile off-canvas navigation or equivalent content-first behavior
- full-screen narrow preview
- 44px scenario/interactive targets
- measured primary-action contrast
- keyboard operation, focus visibility, reduced motion, increased contrast, and no console errors

## Implementation contract gaps

Do not treat prototype state as implementation evidence. Production needs:

- scoped grants, active principal, authorization version, and explicit target scope
- Tutor Draft persistence and concurrency
- approved curriculum revision selection and provenance API
- bounded Teaching schema and preview isolation
- Test evidence and evaluator contracts
- designated Educator review and separation of duties
- idempotent publication, immutable Published versions, history, and recovery provenance
- explicit teacher class-version application and bounded adaptation
- privacy-safe monitoring and Incident contracts
- safe budget/channel handoff authority
