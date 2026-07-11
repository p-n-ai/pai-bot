# SITE AND PUBLIC DOCS

**Generated:** 2026-07-11
**Commit:** bdd0c16

Astro package combining a custom landing page with Starlight documentation.

## WHERE TO LOOK

| Task | Location |
|------|----------|
| Landing page | `src/pages/index.astro` |
| Landing layout/components | `src/layouts`, `src/components` |
| Starlight docs | `src/content/docs` |
| Sidebar/base/site config | `astro.config.mjs`, `src/content.config.ts` |
| Landing theme styles | `src/styles/globals.css` |
| Starlight overrides | `src/styles/starlight.css` |

## CONVENTIONS

- Use `pnpm` in this package.
- Route landing links/assets through `withBase()` or `import.meta.env.BASE_URL`.
- Keep landing `.dark` state separate from Starlight's `data-theme` mechanism.
- Docs pages carry Starlight frontmatter, including title, description, and sidebar order where relevant.
- Verify runtime, command, version, and architecture claims against current code and `justfile`.

## ANTI-PATTERNS

- No assumption that landing and Starlight share routing or theme state.
- No stale setup/deployment claim copied from old docs without code verification.
- No committed generated output.

## COMMANDS

`pnpm dev`, `pnpm build`, and `pnpm preview` run from this directory.
