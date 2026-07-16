# PROJECT DOCUMENTATION

**Generated:** 2026-06-03T13:57:00Z
**Commit:** 08308df

Cross-cutting product, architecture, runtime, ops, QA, and implementation notes.

## STRUCTURE

```
docs/
├── architecture/  # backend/domain/provider/curriculum contracts
├── runtime/       # live bot/chat/runtime behavior contracts
├── ops/           # setup, config, deployment, local tools
├── admin/         # admin routes/auth-specific docs
├── codebase/      # repo maps for agents/humans
├── qa/            # pilot scripts and test artifacts
└── planning/      # proposals, not current-state docs
```

## WHERE TO LOOK

| Task | Location |
|------|----------|
| Current code map | `codebase/README.md`, `codebase/backend.md`, `codebase/frontend.md` |
| Architecture claims | `architecture/architecture.md`, `technical-plan.md` |
| AI provider docs | `architecture/ai-providers.md` |
| Curriculum docs | `architecture/curriculum.md`, `curriculum-oss.md` |
| Runtime behavior | `runtime/*.md` |
| Env/local setup/deploy | `ops/*.md` |
| Admin product/API docs | `admin-panel.md`, `admin-panel-uiux.md`, `admin/routes.md` |

## CONVENTIONS

- Mark planned work as planned; do not claim unimplemented endpoints/features.
- If code changes env/config/provider/domain boundaries, update docs in the same task.
- Keep `docs/codebase/*` factual and derived from repo shape.

## ANTI-PATTERNS

- No stale day/status claims without checking `docs/development-timeline.md` and code.
- No duplicating long API specs in multiple docs; link or summarize.
