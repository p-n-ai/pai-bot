# ADMIN SPA COMPONENTS

**Generated:** 2026-07-11
**Commit:** bdd0c16

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
| Embed/AI/WhatsApp settings | `settings/` |
| Export workflows | `export/` |

## CONVENTIONS

- Feature components may own API effects and loading state; response parsing and deterministic calculations stay in `src/lib`.
- Tests cover visible behavior, form state, and loading/error branches.
- Prefer shared admin surfaces before inventing new wrappers.
- Component additions should use existing `ui/` primitives and shadcn project context.

## ANTI-PATTERNS

- No duplicate API orchestration across a route and its feature component; make one owner explicit.
- No one-off visual primitive that belongs in `shared/` or `ui/`.
- No inaccessible form/control composition; labels and focus states are required.
