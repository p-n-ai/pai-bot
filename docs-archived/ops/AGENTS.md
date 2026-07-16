# OPS DOCS

**Generated:** 2026-06-03T13:57:00Z
**Commit:** 08308df

Local setup, environment config, deployment, auth emulation, and support tooling docs.

## WHERE TO LOOK

| Task | Location |
|------|----------|
| Quickstart/prereqs | `setup.md` |
| Env variables/defaults | `config.md`, `.env.example` |
| Production deploy | `deployment.md`, `deploy/` |
| Local auth emulation | `local-auth-emulation.md` |
| Helper tools | `local-tools.md`, `tools/`, `scripts/` |

## CONVENTIONS

- Env var docs must match `internal/platform/config/config.go` and validation tests.
- Prefer `just` recipes where available; keep Docker fallback explicit.
- Deployment docs must name actual compose/Helm files.

## ANTI-PATTERNS

- No secret values or broad env dumps.
- No install steps for unused package managers.

## NOTES

- Local commands should favor `just` aliases already in the repo.
- Config docs should include default, requiredness, and failure behavior.
- Deployment changes often touch Docker, Helm, Caddy/Nginx, and env docs.
- Auth emulation docs must stay clearly local-only.
