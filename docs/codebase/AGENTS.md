# CODEBASE MAP DOCS

**Generated:** 2026-06-03T13:57:00Z
**Commit:** 08308df

Agent/human navigation maps for backend, frontend, data, ops, and local tooling.

## WHERE TO LOOK

| Task | Location |
|------|----------|
| Top-level routing | `README.md` |
| Backend map | `backend.md` |
| Frontend map | `frontend.md` |
| Data/ops/scripts map | `data-and-ops.md` |

## CONVENTIONS

- This folder is factual inventory; update after moving files or changing ownership.
- Prefer concise path-to-purpose bullets over architecture prose.

## ANTI-PATTERNS

- No planned directories unless clearly labeled planned.
- No stale ownership claims after package moves.

## NOTES

- Re-run lightweight `find`/`rg` inventory before significant edits here.
- Keep maps task-oriented: where to change X, not what X means broadly.
- Mention generated/local-only agent files only when they affect workflow.
- Avoid copying full root AGENTS guidance into these docs.
