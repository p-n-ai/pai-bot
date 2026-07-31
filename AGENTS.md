# P&AI BOT

**Generated:** 2026-07-11
**Commit:** bdd0c16

K-12 proactive learning agent for Telegram, WhatsApp, WebSocket/embed chat, and admin dashboards. Go modular monolith with Vite and Astro frontends; current primary scope is backend Go and `admin-spa/`.

## STRUCTURE

```
pai-bot/
├── cmd/          # entrypoint binaries (AGENTS.md)
├── internal/     # backend domains (AGENTS.md)
├── admin-spa/    # Vite + TanStack Router admin SPA (AGENTS.md)
├── site/         # Astro site
├── migrations/   # SQL migrations
├── deploy/       # Docker image/runtime assets
├── terraform/    # infrastructure
├── scripts/      # dev/ops helpers
├── tools/        # repo tooling
└── oss/          # curriculum/content mirror
```

## WHERE TO LOOK

| Task | Location |
|------|----------|
| Server startup/routes | `internal/server`, `cmd/server/main.go` entrypoint wrapper, `internal/adminapi`, `internal/apidocs` |
| Tutor turn behavior | `internal/agent/engine.go`, `internal/agent/turn.go`, `internal/agent/prompt_builder.go` |
| Bot/chat channels | `internal/chat`, `internal/agent/dev_commands.go` |
| AI providers/routing | `internal/ai`, `internal/platform/airouter`, `internal/platform/config` |
| Auth/session/RBAC | `internal/auth`, `internal/adminapi`, migrations touching auth tables |
| Progress/mastery | `internal/progress`, `internal/agent/quiz_progress.go` |
| Curriculum/retrieval | `internal/curriculum`, `internal/retrieval`, `oss/` |
| Admin SPA API/client | `admin-spa/src/lib/admin-api.ts`, `admin-spa/src/lib/*-types.ts` |
| Admin SPA routes | `admin-spa/src/routes`, `admin-spa/src/routeTree.gen.ts` |
| Admin SPA UI | `admin-spa/src/components`, `admin-spa/src/components/ui` |
| Local seed/demo data | `cmd/seed`, `internal/platform/seed` |

## CONVENTIONS

- `cmd/*` parses flags/env and wires dependencies; reusable behavior belongs in `internal/*`.
- `cmd/server` stays a thin entrypoint; `internal/server` owns HTTP lifecycle, handlers, middleware, chat HTTP mounts, and server adapters.
- Backend I/O paths take `context.Context` first: DB, cache, AI, HTTP-ish orchestration.
- Postgres code uses `*_postgres.go`; integration tests are explicit with `_integration_test.go` naming/build tags where used.
- Tenant data paths preserve `tenant_id`; platform-admin/global access is explicit, not fallback.
- AI decisions route through `internal/ai`; tutor/chat packages stay provider-neutral.
- Admin SPA mirrors backend JSON shapes in `src/lib/*-types.ts` and keeps route files thin.

## COMMIT & PR CONVENTIONS

- No internal plan/sequence labels ("PR1", "PR3", "step 2", "phase 4") in commit messages, PR titles, or PR bodies. GitHub's own number is the only PR number.
- `.humanlayer/` and `.agents/` are local working artifacts: never commit them, never link or mention them in commit messages or PR descriptions.
- PR descriptions describe the code diff only — no walkthrough/artifact generation into `.humanlayer/`, no cloud artifact links, no deviation-from-plan sections referencing local plan files. Skip those steps when a PR-description skill asks for them.
- Branch, commit, and PR names say what the change does in plain words — no codenames or plan shorthand.
- Comments stay sparse everywhere: only tricky or bug-prone logic. `internal/llm` is comment-free (attribution in `NOTICE`).

## ANTI-PATTERNS

- No imports from `cmd/` into `internal/`.
- No provider-specific branches in `internal/agent` or `internal/chat`.
- No tenant-blind SQL on product tables.
- No broad env/secret dumps in logs, tests, or CLI debug output.
- No free-form quiz JSON parsing; use structured helpers in `internal/ai`.
- No hand-editing generated TanStack router artifacts without matching route changes.

## COMMANDS

```bash
just setup
just prepare-local-dev
just go
just admin-spa
just once-dev
just test
just test-integration
just test-all
just build-backend
just migration-create <name>
just migrate
just seed
just db-url-redacted
just db-seed-state
cd admin-spa && pnpm test
cd admin-spa && pnpm test:e2e
cd admin-spa && pnpm typecheck
cd admin-spa && pnpm build
```

## KEY CONFIGS

| Tool | Entry | Notes |
|------|-------|-------|
| Go module | `go.mod` | Go 1.25 module `github.com/p-n-ai/pai-bot` |
| Recipes | `justfile` | Preferred local workflow surface |
| Env template | `.env.example` | `LEARN_*`; auth root secret `PAI_AUTH_SECRET` |
| Local runtime | `docker-compose.yml` | Postgres + Dragonfly |
| Production runtime | `docker-compose.prod.yml`, `deploy/docker/Dockerfile` | Container packaging |
| Admin SPA | `admin-spa/package.json`, `admin-spa/vite.config.ts` | Vite, TanStack Router, React, pnpm |
| Migrations | `migrations/` | Create with `just migration-create <name>` |

## NOTES

- Local runtime should stay local: never run migrations with `GOOSE_DSN`/`LEARN_DATABASE_URL` aimed at remote DBs.
- `just admin-spa` starts the Vite admin SPA and attempts to boot the Go backend when needed.

## Agent skills

### Issue tracker

Issues are tracked in GitHub Issues using the `gh` CLI. See `docs/agents/issue-tracker.md`.

### Triage labels

Triage uses the default five GitHub labels. See `docs/agents/triage-labels.md`.

### Domain docs

Domain docs use a single-context layout. See `docs/agents/domain.md`.

<!-- codex-lean-guardrails:start -->
# Codex working agreement

## Scope

- Implement only the user's explicit request.
- Do not invent acceptance criteria, speculative abstractions, future-proofing, unrelated cleanup, or follow-up features.
- Report newly discovered unrelated issues instead of fixing them.
- Prefer the smallest coherent diff that satisfies the request.

## Test authoring

- Tests are evidence for changed behavior, not a second implementation project.
- For a bug fix, add one minimal regression test when one case proves the bug. Use the smallest coherent set only when distinct changed branches genuinely need separate proof.
- For a feature or behavior change, extend the nearest existing test at the cheapest deterministic layer that proves the externally observable behavior.
- Do not duplicate the same behavior across unit, integration, browser, and E2E layers. Add a higher layer only for confidence the lower layer cannot provide.
- For refactors, documentation, formatting, comments, or other behavior-preserving changes, do not add tests unless an existing test must be adjusted for a real contract change.
- Follow the repository's existing test architecture, naming, helpers, and fixture style. Do not introduce a new test framework, runner configuration, setup subsystem, first test suite, helper abstraction, test-only production API, snapshot system, or fixture system during a normal coding task.
- Test public or user-observable behavior. Do not lock in private calls, internal ordering, incidental markup, or other implementation details.
- Do not test trivial constants, static mappings, removed behavior, framework behavior, or hypothetical edge cases unrelated to the requested change or reproduced failure.
- Avoid broad parameter matrices. Select representative equivalence classes and the boundaries changed by this task.
- Prefer an existing test file. Snapshots, fixtures, and golden files are allowed only when that artifact is the established contract and the change intentionally affects it.
- Never weaken assertions, rewrite expected output, or update snapshots merely to make the implementation pass.
- Keep test changes within `.codex/test-policy.json`. The default `focused` profile is a circuit breaker, not a coverage target. Do not fill its allowance unnecessarily.
- When a legitimate task exceeds the focused profile, stop and report why it needs `CODEX_TEST_PROFILE=expanded`, `CODEX_TEST_PROFILE=tests-only`, or explicit human ownership. Do not silently broaden scope.

## Validation

- The only local validation command available to the agent is `./scripts/agent-check changed`.
- Run it once after a coherent edit batch, in the foreground, and only rerun after a relevant code change.
- Do not run raw repository-wide test, lint, typecheck, build, coverage, E2E, benchmark, CI, or release commands.
- Full validation belongs to CI or an explicit human request.
- Stop after the requested change and bounded validation. State exactly what was not checked.

## Guardrail integrity

- Do not edit `.codex/agent-check.json`, `.codex/test-policy.json`, `.codex/hooks.json`, files under `.codex/hooks/`, `scripts/agent-check`, `scripts/test-policy`, `scripts/test-policy-ci`, `.github/workflows/codex-test-policy.yml`, or `.github/CODEOWNERS` from a normal Codex session.

## Efficient inspection

- In Code Mode, batch independent read-only tool calls within one bounded stage. Use `Promise.allSettled` when partial results remain useful and `Promise.all` when any failure should abort.
- Keep dependent, adaptive, approval-sensitive, conflicting, waiting, and mutation steps sequential.
- Do not expand investigation scope merely because more calls can be batched.
<!-- codex-lean-guardrails:end -->
