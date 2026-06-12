# ARCHITECTURE DOCS

**Generated:** 2026-06-03T13:57:00Z
**Commit:** 08308df

Source for domain boundaries, AI provider behavior, and curriculum contracts.

## WHERE TO LOOK

| Task | Location |
|------|----------|
| Modular monolith/domain map | `architecture.md` |
| AI providers/fallback/budget | `ai-providers.md` |
| Curriculum YAML/schema/loading | `curriculum.md` |

## CONVENTIONS

- Architecture docs must distinguish current packages from planned splits.
- Provider additions/removals require README provider table alignment.
- Curriculum schema claims must match `internal/curriculum/types.go` and fixtures.

## ANTI-PATTERNS

- No new domain names unless `internal/` actually has or will get the boundary.
- No provider-specific tutor behavior here; keep that in runtime/prompt docs.

## NOTES

- Use current/planned labels aggressively.
- Architecture diagrams must match text and code boundaries.
- AI cost/budget claims should cite provider routing behavior in code.
- Curriculum docs are contract docs; update tests/fixtures with schema edits.
