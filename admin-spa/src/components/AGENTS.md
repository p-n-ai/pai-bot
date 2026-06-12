# ADMIN SPA COMPONENTS

**Generated:** 2026-06-04T16:28:07Z
**Commit:** bb3a740

Feature, shared, and primitive React components for admin workflows and dashboards.

## STRUCTURE

```
components/
├── ui/           # shadcn/base primitives (AGENTS.md)
├── shared/       # admin layout/display building blocks
├── auth/         # login/join/activation panels
├── dashboard/    # dashboard and entity details
├── ai-usage/     # budget and provider usage views
├── onboarding/   # setup wizard
├── classes/      # class management
├── users/        # user management
├── retrieval/    # retrieval lab
├── settings/     # embed/WhatsApp settings
└── export/       # export panel
```

## WHERE TO LOOK

| Task | Location |
|------|----------|
| Shared admin surfaces | `shared/` |
| Design primitives | `ui/` |
| Auth screens | `auth/` |
| Dashboard panels | `dashboard/` |
| AI usage charts/tables | `ai-usage/` |
| Onboarding wizard | `onboarding/` |
| Classes/users management | `classes/`, `users/` |
| Retrieval lab UI | `retrieval/` |

## CONVENTIONS

- Feature components receive typed data/actions; parsing and calculations stay in `src/lib`.
- Tests cover visible behavior, form state, and loading/error branches.
- Prefer shared admin surfaces before inventing new wrappers.
- Component additions should use existing `ui/` primitives and shadcn project context.

## ANTI-PATTERNS

- No API fetching hidden in leaf components when route/provider layer can pass data.
- No one-off visual primitive that belongs in `shared/` or `ui/`.
- No inaccessible form/control composition; labels and focus states are required.
