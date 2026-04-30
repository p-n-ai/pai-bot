---
summary: Presentation-ready development log for recent pai-bot work, grouped by week with impact notes and likely team Q&A.
owner: thor
read_when:
  - You need to present recent pai-bot progress to the team.
  - You need a human-readable summary of recent landing, admin, runtime, docs, test, and embed work.
  - You are updating development-timeline.md with weekly progress context.
---

# thor development log for pai-bot

Owner: thor.

This log summarizes recent work in presentation form. It is written for team readout, not as an implementation checklist. For the source timeline, see [development-timeline.md](development-timeline.md).

## Past Two Weeks + Today

Short version:
- Apr 20-24: product polish, landing/admin UX, test stability, and first embed-widget preview.
- Apr 27-30: tutor runtime structure, repo documentation, admin test stabilization, and safer embed settings groundwork.
- Today, Apr 30: shift priority away from admin polish and toward improving tutor prompts using GEPA-style prompt optimization.

Presentation framing:
Over the past two weeks, the work moved between two product needs: making the app easier to demo, and making the tutor runtime easier to inspect when answers go wrong. The landing/admin work helped with demo quality. The runtime/docs/test work made ownership and debugging paths clearer. The next focus is less admin surface area and more tutor-answer quality.

Main concern:
Student-facing quality still needs more real testing. A lot of confidence currently comes from code structure, tests, simulations, and local/manual verification. That is useful, but it is not the same as watching enough real students struggle, ask messy questions, switch language, abandon flows, or misunderstand explanations. The next improvement loop should include more real student conversations and prompt evaluation against those conversations.

Today plan:
- Drop admin polish as the main priority for now.
- Improve tutor prompts using GEPA-style prompt optimization: define examples, score outputs, compare prompt variants, and keep the version that improves tutor behavior.
- Use real or representative student conversations as evaluation material where available.
- Track whether prompt changes improve explanation clarity, curriculum alignment, recovery from confusion, and answer helpfulness.
- Avoid optimizing only for clean test prompts; include messy student language and partial understanding.

Potential Q&A:
- Why drop admin for now?
  Admin is useful, but the core product promise is the tutor helping students. If tutor quality is weak, admin polish will not save the experience.
- What is the biggest current risk?
  Not enough real student testing. The system may pass structured checks while still failing on messy real learning behavior.
- Why use GEPA?
  To make prompt improvement more systematic instead of only hand-tuning copy. The goal is to compare prompt variants against evaluation examples and keep changes that improve tutor behavior.
- What should count as success?
  Better student-facing explanations, fewer confused follow-ups, stronger curriculum grounding, better handling of weak/partial answers, and clearer next-step guidance.
- What should we avoid?
  Overfitting prompts to polished benchmark examples while ignoring real student phrasing and behavior.

## Apr 20-24, 2026

Theme: product polish, landing/admin UX, test stability, and first embed-widget preview.

Evidence snapshot:
- 9 non-merge commits directly attributed to Thoriq.
- Around 1,439 insertions and 408 deletions.
- Main work dates: Apr 20 and Apr 22, 2026.
- Embed preview work is on `origin/thor/embeddable-widget-preview`, not current `main`.

### 1. Landing Page Improvement

Improved the admin/root landing page so it looked and read more like a real product surface.

Key work:
- Reworked the landing page structure.
- Added audience-focused landing data.
- Added a live-demo style landing section.
- Added landing icons and copy draft material.
- Refined and shortened the landing copy after the first pass.

Why it matters:
The app became easier to explain visually. Instead of only showing admin screens, it started presenting who the product is for and what the first experience should communicate.

Potential Q&A:
- What changed on the landing page?
  The page got clearer product messaging, better audience sections, and a stronger visual/demo structure.
- Why spend time on landing copy?
  Because unclear copy makes the product harder to sell or demo, even if the backend works.
- Was this just cosmetic?
  No. It helped clarify who the product is for and what the admin/user experience is supposed to communicate.

### 2. Admin App Copy and Login Visual Polish

Tightened admin-facing text and aligned the login aurora visual treatment.

Key work:
- Tightened dashboard, parent, invite, account, and login copy.
- Adjusted login backdrop/color treatment.
- Kept tests aligned with the copy changes.

Why it matters:
Admin surfaces became shorter and easier to scan. This reduces rough edges during demos.

Potential Q&A:
- What is login aurora?
  The visual background/style treatment on the login screen.
- Why tighten admin copy?
  Shorter, clearer copy makes the admin app easier to scan and more professional.

### 3. Test Stabilization Around Landing/Admin

Reduced fragile tests around copy and headings.

Key work:
- Decoupled landing tests from exact copy.
- Aligned landing smoke heading checks.
- Updated page/component tests after landing changes.

Why it matters:
Tests still protect important behavior, but they are less likely to fail from harmless wording changes.

Potential Q&A:
- Did test coverage become weaker?
  Not really. It became less fragile. The tests moved away from exact wording and toward meaningful behavior/structure.
- Why does that matter?
  It keeps CI useful: fewer false alarms, more confidence when tests fail.

### 4. CI E2E Gating

Adjusted CI so E2E tests are skipped on pull requests.

Key work:
- Updated CI/deploy workflow gating.
- Reduced PR friction from heavier E2E jobs.

Why it matters:
PRs become easier to iterate on while heavier verification stays available for controlled deploy/main flows.

Potential Q&A:
- Why skip E2E on PRs?
  To reduce slow or flaky PR feedback while keeping E2E available in controlled deploy/main flows.
- Is that risky?
  Slightly. It is a tradeoff: faster PR iteration now, stronger E2E reserved for release/deploy paths.

### 5. Embeddable Widget Preview

Added the first embeddable widget preview on `origin/thor/embeddable-widget-preview`.

Key work:
- Added `admin/public/embed.js`.
- Added a `/widget` page.
- Added widget chat UI.
- Added widget configuration helpers.

Why it matters:
This was early groundwork for letting the bot be embedded outside the main app.

Potential Q&A:
- What is the embed widget?
  A small chat experience that can be placed on another website.
- Was this already merged?
  From current evidence, this commit is on `origin/thor/embeddable-widget-preview`, not current `main`.

### Overall Impact

Last week made `pai-bot` more presentable and easier to iterate:
- Better landing/product story.
- Cleaner admin copy.
- More polished login visuals.
- Less brittle landing/admin tests.
- Faster PR feedback via CI gating.
- First embed-widget preview branch.

## Apr 27-30, 2026

Theme: tutor runtime structure, repo documentation, admin test stabilization, and safer embed settings groundwork.

Evidence snapshot:
- 26 non-merge commits across refs.
- Around 4,426 insertions and 1,145 deletions.
- Apr 27 work is on current `main`.
- Apr 30 embed work is on `feat/embed-ui` / `origin/feat/embed-ui`, not current `main`.

### 1. Tutor Runtime and Agent Architecture

Cleaned up the tutor agent runtime so the system has clearer boundaries between learner context, prompt construction, and model execution.

Key work:
- Added clearer turn packet / context packet boundaries for tutor messages.
- Refactored prompt-building logic so context is passed in a more structured way.
- Removed obsolete context prompt helpers.
- Fixed duplicated image prompt instructions.
- Added and refined documentation for the agent turn API.

Why it matters:
This makes the AI tutor flow easier to debug because learner context, prompt construction, and model execution are separated instead of being mixed into one ad-hoc prompt path.

Potential Q&A:
- What is tutor runtime?
  The backend flow that takes a student message, gathers relevant context, prepares the AI prompt, calls the model, and returns the tutor response.
- What are context packets?
  Structured pieces of information passed into one AI tutor turn, such as learner context, message history, curriculum data, and metadata.
- Why not just put everything into one prompt?
  Because that becomes hard to debug and easy to break. Structured context makes the tutor easier to test, inspect, and extend.
- What does this unlock later?
  Better personalization, safer memory boundaries, quiz-aware tutoring, progress-aware responses, and cleaner analytics.

### 2. Documentation and Codebase Organization

Did a major documentation cleanup so future work can start from accurate repo context instead of scattered assumptions.

Key work:
- Reorganized repo documentation into clearer sections: admin, runtime, ops, architecture, QA, and codebase maps.
- Added folder-level documentation for backend, frontend, data/ops, and overall repo structure.
- Documented Telegram and WhatsApp runtime behavior.
- Cleaned up audit residue and simplified site doc references.
- Preserved Akmal-owned planning documents while making the docs easier to navigate.

Why it matters:
This reduces repeated context discovery and gives engineers a faster way to find the package, route, or doc that owns a change.

Potential Q&A:
- Why spend time on docs?
  Because the project is growing. Clear docs help the team understand current behavior faster, avoid repeated discovery work, and make safer changes.
- What changed practically?
  Docs are now easier to navigate by area: backend, frontend, runtime, ops, QA, and admin.
- Are these docs replacing the code?
  No. The code is still the source of truth. The docs help people find and understand the code faster.
- Why preserve Akmal-owned planning docs?
  Because those documents carry product and planning context. The cleanup preserved ownership and intent instead of rewriting them.

### 3. Admin Panel and E2E Test Stabilization

Fixed several admin-facing and test stability issues.

Key work:
- Restored public page landmarks for admin login/invite surfaces.
- Aligned backend E2E redirects with the actual app behavior.
- Updated AI usage E2E expectations.
- Removed overly copy-sensitive E2E assertions.
- Focused admin auth E2E coverage on behavior rather than fragile text.

Why it matters:
The admin panel tests become less noisy while still protecting login, redirect, and public-entry behavior.

Potential Q&A:
- What are E2E tests?
  End-to-end tests simulate real user flows in the app, such as logging in, navigating admin pages, and checking that important screens still work.
- Why remove copy-sensitive assertions?
  Some tests were failing because exact wording changed, even when the feature still worked. The checks now focus on behavior instead of fragile text.
- Did this reduce test quality?
  No. The goal was to keep meaningful coverage while removing brittle checks that do not represent real product breakage.
- What does "restore public page landmarks" mean?
  It means public admin pages have the expected page structure so tests, accessibility, and navigation can recognize them properly.

### 4. Embed Settings and Chat Widget Work

Worked on the embeddable chat widget/admin settings surface on the `feat/embed-ui` branch.

Key work:
- Added the embed settings UI.
- Bound widget authentication to the parent origin for safer embed behavior.
- Refactored the admin embed settings UX to align with familiar Vercel-style patterns.
- Added a workflow benchmark for embed settings.
- Documented chat widget domain decisions.

Why it matters:
This moves the product closer to a configurable embeddable chat experience, with origin checks documented as part of the security boundary.

Potential Q&A:
- What is the embed settings UI?
  The admin interface for configuring an embeddable chat widget, so the bot can eventually be placed on another website or school page.
- What does "bind widget auth to parent origin" mean?
  It means the widget checks which website is hosting it and ties authentication to that allowed website origin.
- Why is origin binding important?
  It helps prevent the widget from being used from unauthorized websites.
- Is the embed work already on main?
  Not yet. The Apr 30 embed work is on `feat/embed-ui` / `origin/feat/embed-ui`.
- What is the user-facing value?
  Schools or partners get a path toward configuring the bot for their own site instead of needing one-off engineering setup for each embed.

### Overall Impact

This week made `pai-bot` easier to inspect, document, and extend:
- Cleaner AI tutor runtime.
- Better repo documentation.
- More stable admin tests.
- Safer embeddable chat groundwork.
- Clearer separation between shipped behavior, planning docs, and future work.

Extra presentation Q&A:
- What was the main theme of the week?
  Stabilization and structure. The work made the product easier to extend without making the system harder to understand.
- Was this mostly backend, frontend, or docs?
  Mixed: backend tutor runtime, repo documentation, admin test stability, and embed settings UI.
- What part is most important technically?
  The tutor runtime cleanup, because it affects how future AI behavior can be made easier to debug, personalize, and evaluate.
- What part is most visible to users?
  The embed settings UI and admin panel fixes.
- What part helps the team the most?
  The documentation and codebase organization, because it reduces onboarding friction and makes future changes easier to locate.
- What is still pending?
  The embed work is still on the feature branch and not yet merged into `main`.
- What is the next logical step?
  Review and merge the embed UI branch, then continue improving admin-facing configuration and widget behavior.
