# DATABASE MIGRATIONS

**Generated:** 2026-07-11
**Commit:** bdd0c16

Timestamped Goose SQL migrations for product schema, constraints, and required data transitions.

## CONVENTIONS

- Create files with `just migration-create <name>`; keep Goose Up/Down markers valid.
- Treat applied migrations as append-only; add a corrective migration instead of rewriting history.
- Product tables carry `tenant_id` unless the global/platform exception is explicit in schema and code.
- Cross-table relationships preserve tenant consistency with constraints or triggers where plain FKs cannot.
- Data backfills are deterministic, bounded, and safe against partially populated databases.
- Keep Down ordering dependency-safe; document deliberate irreversibility in the migration itself.
- Use `StatementBegin`/`StatementEnd` for trigger/function bodies when Goose parsing requires it.

## VERIFICATION

- Never point `GOOSE_DSN` or `LEARN_DATABASE_URL` at a remote database for local validation.
- Run the smallest owning integration tests after schema changes; auth and runtime-settings tests load named migrations.
- Keep demo/product seed behavior in `cmd/seed` and `internal/platform/seed` unless schema bootstrap requires it.

## ANTI-PATTERNS

- No tenant-blind product schema or data rewrite.
- No second migration runner against the same long-lived database.
- No rename/removal of a migration referenced directly by integration tests without updating the contract.

## NOTES

- Runtime settings are a deliberate platform-global schema exception; keep exceptions explicit.
