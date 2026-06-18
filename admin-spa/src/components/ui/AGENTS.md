# ADMIN SPA UI PRIMITIVES

**Generated:** 2026-06-04T16:28:07Z
**Commit:** bb3a740

shadcn/base UI primitives and low-level design-system wrappers used by feature components.

## WHERE TO LOOK

| Task | Location |
|------|----------|
| Buttons/inputs/forms | `button*.tsx`, `input*.tsx`, `field.tsx`, `label.tsx`, `textarea.tsx` |
| Overlays | `dialog.tsx`, `sheet.tsx`, `drawer.tsx`, `popover.tsx`, `tooltip.tsx` |
| Menus/selects | `dropdown-menu.tsx`, `select.tsx`, `command.tsx`, `combobox.tsx` |
| Data display | `table.tsx`, `card.tsx`, `badge.tsx`, `chart.tsx` |
| Navigation/layout | `sidebar.tsx`, `navigation-menu.tsx`, `breadcrumb.tsx`, `tabs.tsx` |
| Feedback | `alert.tsx`, `sonner.tsx`, `skeleton.tsx`, `spinner.tsx`, `empty.tsx` |

## CONVENTIONS

- Treat files as primitives: composable props, no product-specific copy or data fetching.
- Use installed shadcn/radix-nova patterns before creating variants.
- Keep imports aligned with `@/*` alias and `components.json` context.
- Accessibility behavior belongs here when it is primitive-level.

## ANTI-PATTERNS

- No school/admin business logic in `ui/`.
- No local styling fork when an existing primitive variant works.
- No breaking primitive API without checking all feature consumers.
