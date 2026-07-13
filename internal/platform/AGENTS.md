# PLATFORM ADAPTERS

**Generated:** 2026-07-11
**Commit:** bdd0c16

Infrastructure wrappers for config, DB, cache, AI router setup, mailer, feature flags, and seed data.

## STRUCTURE

```
platform/
├── config/        # LEARN_* env loading and validation
├── database/      # pgxpool setup
├── cache/         # Redis/Dragonfly client
├── airouter/      # AI router setup from config
├── featureflags/  # runtime flags
├── mailer/        # outbound email adapter
├── settings/      # encrypted persisted runtime settings (AGENTS.md)
├── tenant/        # tenant context adapter
└── seed/          # demo/token-budget seed routines
```

## WHERE TO LOOK

| Task | Location |
|------|----------|
| Env/config defaults | `config/` and `.env.example` |
| DB pool setup | `database/` |
| Cache client | `cache/` |
| AI router from config | `airouter/` |
| Demo/token-budget seed | `seed/`, `cmd/seed` |
| Mail delivery | `mailer/` |
| Runtime AI/auth settings | `settings/` |
| Tenant context adapter | `tenant/` |

## CONVENTIONS

- Adapters stay thin where possible; `settings` owns persistence, encryption, and effective-value policy.
- Config defaults and validation stay in lockstep with `.env.example`.
- Connection constructors accept context and return closable clients/pools.
- Seed routines prefer idempotent inserts/upserts.

## ANTI-PATTERNS

- No importing `cmd/server` helpers here.
- No hardcoded local service URLs except documented dev defaults.
- No panics for config/runtime errors; return errors to callers.
- No destructive seed behavior without explicit mode/flag.
