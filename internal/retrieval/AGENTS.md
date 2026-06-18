# RETRIEVAL SERVICE

**Generated:** 2026-06-04T16:28:07Z
**Commit:** bb3a740

Curriculum retrieval/indexing facade used by tutor context and admin retrieval lab.

## WHERE TO LOOK

| Task | Location |
|------|----------|
| Service behavior | `service.go`, `service_test.go` |
| Platform adapters | `platform.go` |
| Curriculum seeding | `curriculum_seed.go` |
| Tutor context use | `internal/agent/curriculum_retriever.go`, `internal/agent/context_*` |
| Admin retrieval lab | `admin-spa/src/components/retrieval`, `admin-spa/src/lib/retrieval-lab*` |

## CONVENTIONS

- Return curriculum citations suitable for tutor prompts.
- Seed paths are idempotent and safe during server startup.
- Ranking/search stays deterministic in unit tests.
- Favor source-path-rich results over opaque snippets.

## ANTI-PATTERNS

- No prompt text assembly here; return context, let agent build prompts.
- No indexing that blocks healthz readiness indefinitely.
- No storage/index shape change without updating tutor and admin retrieval consumers.

## NOTES

- Retrieval lab and tutor context both depend on this package.
