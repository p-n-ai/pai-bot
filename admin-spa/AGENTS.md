<!-- intent-skills:start -->

## Skill Loading

Before substantial work:

- Skill check: run `pnpm dlx @tanstack/intent@latest list`, or use skills already listed in context.
- Skill guidance: if one local skill clearly matches the task, run `pnpm dlx @tanstack/intent@latest load <package>#<skill>` and follow the returned `SKILL.md`.
- Vite/router config: run `pnpm run intent:vite` and keep `tanstackRouter(...)` before `react()` in `vite.config.ts`.
- Component work: use `building-components`; prefer semantic HTML, accessible defaults, composable props, visible focus states, and lightweight component APIs.
- UI review: use `web-design-guidelines`; fetch the latest Vercel Web Interface Guidelines before review and check changed UI files against accessibility, focus, form, and animation rules.
- shadcn/ui: use `shadcn` before adding or composing UI. Run `pnpm dlx shadcn@latest info --json` for project context, use `pnpm dlx shadcn@latest docs <component>` before component work, prefer installed components, and keep imports aligned with aliases in `components.json`.
- shadcn project context: Vite SPA, Tailwind v4, radix-nova, lucide icons, `@/*` alias, UI components under `src/components/ui`.
- Monorepos: when working across packages, run the skill check from the workspace root and prefer the local skill for the package being changed.
- Multiple matches: prefer the most specific local skill for the package or concern you are changing; load additional skills only when the task spans multiple packages or concerns.
<!-- intent-skills:end -->
