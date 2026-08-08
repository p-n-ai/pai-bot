# Product

<!-- impeccable:product-schema 1 -->

## Platform

web

## Users

- **AI operators** are the primary admin-SPA users. They configure the Tutor's curriculum, teaching behavior, runtime, channels, safety policy, and budgets; test the complete Tutor; publish reviewed versions; and monitor live operation.
- **Teachers who use the Tutor with a class** are the secondary users. They need to understand learner progress, decide who needs help, explicitly apply a Published version to an assigned class, and adjust approved class-level teaching behavior without handling infrastructure or secrets.
- **School administrators** manage people, access, budgets, classes, channels, and records. They may also hold teacher or operator capabilities.
- **Parents** have a separate, limited view of their own learner. Students use the chat experiences rather than the admin workspace.

## Product purpose

Pai Bot provides curriculum-grounded AI tutoring through the chat tools learners already use. The admin SPA is its capability-adaptive control plane: operators build and run the learning AI, while teachers apply it safely to real classes.

The approved production experience uses a simple Intercom Fin-style setup flow with GPT Builder-style live preview inside one authenticated application with three separated workspaces:

- **Build AI:** **Overview → Curriculum → Teaching → Test tutor → Publish → Activity**
- **Teach:** **Today → Class → Learner or topic → Action → Adapt tutor → Preview**
- **Manage school:** **Overview → People and access → Classes and rosters → Setup and channels → Budget → Records and audit**

Build AI presents one **Tutor**, not a catalog of technical build, test, publication, or class-link records. Operators work with a **Draft**, inspect **Test results**, publish a **Published version**, and use **Version history** and **Used by classes** to understand its production state. Curriculum and Teaching use a live preview so operators can check the Tutor while they configure it.

## Positioning

Pai Bot is not a generic agent builder or a reporting dashboard. It connects approved curriculum, AI runtime, school workspace, class context, and learner progress in one controlled system. The same Tutor can be operated centrally and adapted locally without exposing provider secrets or weakening school safeguards.

The public experience stays simple while technical contracts preserve immutable revisions, evidence tied to the exact Draft, educator review, audit history, and recovery. These safeguards support the flow; they do not become navigation categories or page titles.

## Operating context

- Malaysian school workflows and KSSM curriculum are the current context.
- AI operators work across curriculum validation, teaching configuration, model and provider configuration, channel delivery, budgets, publish readiness, and incidents.
- Teachers work around the daily class rhythm: reviewing who needs attention, understanding why, acting, and checking whether the intervention helped.
- Pai Bot can run with cloud or local AI providers and supports self-hosted deployments.
- Curriculum comes from Open School Syllabus, and production must use an approved immutable revision.

## Capabilities and constraints

- The authorization model uses three explicit workspace capabilities: `can_build_ai`, `can_teach`, and `can_manage_school`. Roles describe account personas and migration defaults; authoritative grants carry platform or school scope, with assigned-class and designated-reviewer predicates where required.
- Scoped authoritative grants are the approved authorization direction. Platform scope never implies access to every school, and school context is explicit in tenant routes.
- `can_manage_ai_settings` remains a narrower platform-global permission inside Build AI and never follows from `can_build_ai` alone in a multi-tenant deployment.
- Current scoped grants determine the default workspace, visible navigation, actions, and data. Display names personalize copy only; they never grant access or choose permissions.
- AI operators open in Build AI, educators open in Teach, and school administrators open in Manage school. People with multiple scoped grants return to their last-used still-authorized workspace and target, then can switch explicitly.
- Operators own global or deployment-level models, providers, credentials, policies, publishing controls, and privacy-safe monitoring.
- Teachers receive bounded class-level controls and never handle provider secrets or platform-global settings.
- Tenant isolation is mandatory across every school, class, learner, configuration, metric, and export path.
- Tutor and chat behavior remain provider-neutral; provider details stay behind the AI platform boundary.
- Pai Bot is disclosed as an AI learning companion, never as a human, friend, or replacement for an educator.
- Curriculum changes require educator review, an immutable revision, validation, and explicit publication.
- Public or shared evidence must not contain learner, school, tenant, consent, credential, or private operational details.
- Editing, previewing, testing, or educator review changes only the Draft. **Publish** is the only action that makes a version available.
- Publishing never changes a class automatically. An assigned teacher explicitly reviews and applies a Published version to each class.
- Class adaptation is bounded to one authorized class and one Published version. It cannot change providers, secrets, safety policy, or the approved curriculum revision.

## Brand commitments

- Product names: **Pai Bot** in prose and **P&AI Bot** where the existing brand lockup is used.
- Pandai Design System 1.5 is the visual foundation for the overhaul. Internal source: private `p-n-ai/pandai.design` at `6ec54c7ed8facd4e7d1441b90db896883a5446b3`.
- Use the main Figma Design System 1.5 file `TLVKe3bgJTdVvuPAzgDq2f`; ignore WIP Backup, Zul's Dungeon, Nadia Exploration, and Syakila Components as visual authorities.
- Use the semantic token vocabulary from `design.color.md`. Do not invent visual tokens or derive interaction-state colors.
- Keep the private mirror and its fonts, icons, images, and other assets internal until license and reuse permissions are confirmed.
- Use the installed shadcn/Radix Nova primitives for accessible behavior and composition, adapted through approved Pandai tokens rather than replacing the design system with stock shadcn styling.

## Evidence on hand

- The current application has working routes and components for class progress, learner detail, onboarding, staff access, budgets, AI usage, global AI settings, website chat, exports, and parent summaries.
- The current navigation mixes Teaching, School administration, and Technical tools in one sidebar. All elevated roles default to `/dashboard`, and teachers can reach AI activity, which supports the reported confusion.
- The current visual implementation mixes admin semantic variables with hard-coded colors, shadows, and page-specific treatments. It does not consistently follow Pandai Design System rules.
- The private Pandai mirror includes `design.color.md`, role-specific design session files, and design principles. It has no confirmed license for external redistribution.
- No user analytics, recorded usability sessions, or role-specific task-frequency data have been provided. The strongest evidence is the current implementation and direct feedback that the experience is confusing.

## Product principles

1. **Separate jobs before simplifying screens.** Teaching, AI operation, and school administration must feel like distinct workspaces.
2. **Make the safe path obvious.** Build AI is one linear flow around one Tutor, with the next incomplete step visible on Overview.
3. **Test before publish.** The Publish page has one readiness checklist, including current Test results and educator review, before a version can become available.
4. **Keep class changes explicit.** A teacher chooses when to apply a Published version or bounded adaptation to an assigned class.
5. **Bound power by grant and scope.** Educators adapt approved behavior for assigned classes; operators control infrastructure and publishing policy only in explicitly granted scopes.
6. **Make safety and provenance understandable.** Show the Draft, curriculum source, Test results, Published version, Version history, and Used by classes without exposing technical record structure, secrets, or private learner content.

## Accessibility and inclusion

- Preserve and improve semantic HTML, keyboard navigation, visible focus, screen-reader labels, responsive behavior, and reduced-motion support already present in the app.
- Critical status and meaning must never rely on color alone.
- Teacher workflows must remain usable under time pressure and on narrow screens.
- Bahasa Melayu, English, and natural mixed-language content must fit without truncating essential actions or status.
