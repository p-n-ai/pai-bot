# Domain Docs

How the engineering skills should consume this repo's domain documentation when exploring the codebase.

## Layout

This repo uses a single-context layout.

- Root context: `CONTEXT.md` if it exists
- Architecture decisions: `docs/adr/` if it exists
- No `CONTEXT-MAP.md` is configured

## Before exploring, read these

- `CONTEXT.md` at the repo root
- ADRs under `docs/adr/` that touch the area you're about to work in
- Existing docs under `docs/architecture/`, `docs/runtime/`, `docs/codebase/`, `docs/admin/`, and `docs/ops/` when relevant

If any of these files don't exist, proceed silently. Do not create context docs upfront; add them only when a task needs resolved project language or decisions.

## Use project vocabulary

When your output names a domain concept, use the term as defined in `CONTEXT.md` or the existing docs. If the concept is missing, note the gap instead of inventing a new source of truth.
