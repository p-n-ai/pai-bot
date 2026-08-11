---
version: alpha
name: P&AI Operate
description: A calm, learner-aware operating system for school administrators and educators.
colors:
  primary: "oklch(0.406 0.073 181.541)"
  primary-hover: "oklch(0.745 0.17 159.438)"
  canvas: "oklch(0.989 0.013 145.472)"
  surface: "oklch(1 0 0)"
  surface-muted: "oklch(0.952 0.034 173.507)"
  line: "oklch(0.745 0.17 159.438)"
  control-line: "oklch(0.551 0.023 264.365)"
  focus-line: "oklch(0.406 0.073 181.541)"
  ink: "oklch(0.371 0 0)"
  ink-hover: "oklch(0.406 0.073 181.541)"
  ink-soft: "oklch(0.51 0 0)"
  muted: "oklch(0.551 0.023 264.365)"
  accent: "oklch(0.745 0.17 159.438)"
  accent-muted: "oklch(0.631 0.143 159.699)"
  nav-text: "oklch(1 0 0)"
  nav-muted: "oklch(0.879 0.09 169.866)"
  nav-label: "oklch(0.897 0.141 134.649)"
  success: "oklch(0.448 0.108 151.328)"
  warning: "oklch(0.473 0.125 46.2)"
  danger: "oklch(0.5 0.182 29.513)"
  informative: "oklch(0.676 0.15 238.112)"
typography:
  display:
    fontFamily: "Poppins, ui-sans-serif, system-ui, sans-serif"
    fontSize: 64px
    fontWeight: 600
    lineHeight: 60px
    letterSpacing: "-0.025em"
  page-title:
    fontFamily: "Poppins, ui-sans-serif, system-ui, sans-serif"
    fontSize: 40px
    fontWeight: 600
    lineHeight: 43px
    letterSpacing: "-0.035em"
  section-title:
    fontFamily: "Poppins, ui-sans-serif, system-ui, sans-serif"
    fontSize: 20px
    fontWeight: 600
    lineHeight: 24px
    letterSpacing: "-0.025em"
  body:
    fontFamily: "Poppins, ui-sans-serif, system-ui, sans-serif"
    fontSize: 15px
    fontWeight: 400
    lineHeight: 24px
  body-small:
    fontFamily: "Poppins, ui-sans-serif, system-ui, sans-serif"
    fontSize: 14px
    fontWeight: 400
    lineHeight: 20px
  label:
    fontFamily: "Poppins, ui-sans-serif, system-ui, sans-serif"
    fontSize: 14px
    fontWeight: 600
    lineHeight: 20px
  metadata:
    fontFamily: "Poppins, ui-sans-serif, system-ui, sans-serif"
    fontSize: 11px
    fontWeight: 600
    lineHeight: 16px
    letterSpacing: "0.14em"
  numeric:
    fontFamily: "Poppins, ui-sans-serif, system-ui, sans-serif"
    fontSize: 28px
    fontWeight: 600
    lineHeight: 28px
    letterSpacing: "-0.03em"
    fontFeature: "tnum"
rounded:
  control: 10px
  navigation: 12px
  surface: 16px
  sign-in: 20px
  feature: 28px
  pill: 999px
spacing:
  xs: 4px
  sm: 8px
  md: 12px
  lg: 16px
  xl: 20px
  2xl: 28px
  3xl: 40px
  4xl: 64px
components:
  app-canvas:
    backgroundColor: "{colors.canvas}"
    textColor: "{colors.ink}"
    typography: "{typography.body}"
  primary-button:
    backgroundColor: "{colors.primary}"
    textColor: "{colors.nav-text}"
    typography: "{typography.label}"
    rounded: "{rounded.pill}"
    padding: "0 16px"
    height: "40px"
  primary-button-hover:
    backgroundColor: "{colors.primary-hover}"
    textColor: "{colors.ink}"
    typography: "{typography.label}"
    rounded: "{rounded.pill}"
    padding: "0 16px"
    height: "40px"
  outline-button:
    backgroundColor: "{colors.surface}"
    textColor: "{colors.ink}"
    typography: "{typography.label}"
    rounded: "{rounded.pill}"
    padding: "0 16px"
    height: "40px"
  surface-card:
    backgroundColor: "{colors.surface}"
    textColor: "{colors.ink}"
    typography: "{typography.body-small}"
    rounded: "{rounded.surface}"
    padding: "{spacing.xl}"
  sign-in-card:
    backgroundColor: "{colors.surface}"
    textColor: "{colors.ink}"
    typography: "{typography.body}"
    rounded: "{rounded.sign-in}"
    padding: "{spacing.2xl}"
  feature-surface:
    backgroundColor: "{colors.surface}"
    textColor: "{colors.ink}"
    typography: "{typography.body}"
    rounded: "{rounded.feature}"
    padding: "{spacing.2xl}"
  muted-surface:
    backgroundColor: "{colors.surface-muted}"
    textColor: "{colors.ink-soft}"
    typography: "{typography.body-small}"
    rounded: "{rounded.navigation}"
    padding: "{spacing.lg}"
  helper-copy:
    backgroundColor: "{colors.surface}"
    textColor: "{colors.muted}"
    typography: "{typography.body-small}"
  sidebar:
    backgroundColor: "{colors.primary}"
    textColor: "{colors.nav-text}"
    typography: "{typography.body-small}"
    padding: "{spacing.xl}"
  sidebar-item:
    backgroundColor: "{colors.primary}"
    textColor: "{colors.nav-muted}"
    typography: "{typography.label}"
    rounded: "{rounded.navigation}"
    padding: "{spacing.md}"
    height: "44px"
  sidebar-group-label:
    backgroundColor: "{colors.primary}"
    textColor: "{colors.nav-label}"
    typography: "{typography.metadata}"
    padding: "0 12px"
  sidebar-item-active:
    backgroundColor: "{colors.accent}"
    textColor: "{colors.ink}"
    typography: "{typography.label}"
    rounded: "{rounded.navigation}"
    padding: "{spacing.md}"
    height: "44px"
  divider:
    backgroundColor: "{colors.line}"
    height: "1px"
    width: "100%"
  progress-marker:
    backgroundColor: "{colors.accent-muted}"
    rounded: "{rounded.pill}"
    size: "6px"
  status-success:
    backgroundColor: "{colors.surface-muted}"
    textColor: "{colors.success}"
    typography: "{typography.metadata}"
    rounded: "{rounded.pill}"
    padding: "4px 8px"
  status-warning:
    backgroundColor: "{colors.surface-muted}"
    textColor: "{colors.warning}"
    typography: "{typography.metadata}"
    rounded: "{rounded.pill}"
    padding: "4px 8px"
  status-danger:
    backgroundColor: "{colors.surface-muted}"
    textColor: "{colors.danger}"
    typography: "{typography.metadata}"
    rounded: "{rounded.pill}"
    padding: "4px 8px"
  status-informative:
    backgroundColor: "{colors.surface-muted}"
    textColor: "{colors.informative}"
    typography: "{typography.metadata}"
    rounded: "{rounded.pill}"
    padding: "4px 8px"
---

## Overview

P&AI Operate is a school operations interface, not a generic analytics dashboard. It helps educators see a learner, understand the state, and take one responsible next action. The mood is calm, capable, and quietly optimistic. Pandai's light green surfaces, graphite text, and green accents create a clear operating hierarchy without turning every action into a celebration.

This document governs new and changed work in `admin-spa/`. It does not govern the learner application, the public Astro site, or third-party embeds. It describes the intended system and the implemented admin vocabulary together. Runtime source remains the final record of shipped behavior.

Use this authority order when evidence conflicts:

1. Accessible behavior, data integrity, and product truth.
2. Runtime tokens and primitives in `admin-spa/src/styles.css` and `admin-spa/src/components/ui/`.
3. Verified Pandai DS 1.5 semantic values recorded from private `p-n-ai/pandai.design` commit `6ec54c7ed8facd4e7d1441b90db896883a5446b3`.
4. This document for composition rules and intentional accessibility adaptations.
5. Live Pandai Figma `TLVKe3bgJTdVvuPAzgDq2f` after direct verification. It supersedes the pinned snapshot when values drift.

Pandai Design System 1.5 supplies the implemented Poppins typography and semantic color foundation. P&AI Operate keeps its own admin composition, Lucide icon library, and selected component geometry.

The current sign-in screen is more editorial than the authenticated workspace. Keep that difference controlled through Poppins, Pandai semantic colors, strong reading order, and one accessibility contract. The runtime loads Poppins globally through the shared sans token. Do not introduce page-local font families.

## Colors

The front matter records OKLCH conversions of the implemented Pandai DS 1.5 sRGB source values. `admin-spa/src/styles.css` defines the runtime semantic tokens and maps the `--admin-*` vocabulary onto them.

Author interface colors in OKLCH. Six-digit hex remains valid only where an external contract requires it, including Google brand artwork, Recharts-generated SVG selectors, the native color input, and the published embed configuration.

Use color by role:

- Canvas uses `Surface/secondary/default-hover oklch(0.989 0.013 145.472)` and never competes with content.
- Surface uses `Surface/general/default oklch(1 0 0)` for focused work.
- Surface muted uses `Surface/primary/default-subtle oklch(0.952 0.034 173.507)` for secondary information.
- Ink uses `Text/default/heading oklch(0.371 0 0)` for headings and primary reading order.
- Navigation uses `Surface/tertiary/default oklch(0.406 0.073 181.541)` as its stable frame.
- Pandai green uses `Surface/primary/default oklch(0.745 0.17 159.438)` for selection and orientation.
- Control boundaries use the P&AI adaptation `oklch(0.551 0.023 264.365)` so inputs remain visible at non-text contrast requirements.
- Secondary graphite carries supporting copy, never primary instructions.

Pandai green is the implemented accent and hover color. The shared primary action deliberately uses Pandai dark teal with white text at rest, then green with graphite text on hover. This is an accessibility adaptation: do not place white normal-size text directly on `oklch(0.745 0.17 159.438)`.

Success, warning, danger, and informative colors communicate state only with an adjacent label or icon. Do not use a colored dot as the complete message. Keep status colors out of navigation and brand surfaces.

Pandai student-context semantic colors are the current admin target and runtime truth. Do not introduce teacher pink, parent yellow, unverified dark-mode values, or derived state colors without a named product requirement and direct DS verification.

## Typography

Poppins is the implemented and required P&AI Operate typeface. It supports friendly product character while remaining clear for compact operations, large entry copy, and numeric data. Use one global font token and a system fallback.

The runtime imports Poppins weights 400–800 and maps both the root family and `--font-sans` to the same fallback stack. Keep new interface typography on that shared token. Reserve monospace for code-like or deliberately tabular technical content.

The sizes in this document are P&AI Operate admin adaptations. They do not claim complete compliance with Pandai's 21-style typography scale.

Use type by function:

- Display is rare. Reserve it for the sign-in promise or an equivalent single-message entry surface.
- Page titles name the current working context. They scale from 30px on small screens to 40px on wide screens.
- Section titles begin a real content region. They do not decorate a card.
- Body text uses 15/24 when reading matters and 14/20 for compact operational detail.
- Metadata is uppercase only for durable categories, workspace context, and status taxonomy.
- Numeric values use tabular figures and a tight line box.

Keep paragraphs near 65–75 characters. Use `text-wrap: balance` for short headings and `text-wrap: pretty` for prose. Tight letter spacing belongs to headings, not fields or body copy.

An eyebrow must carry information such as school, class, scope, or lifecycle state. Remove it when it repeats the title. Never truncate the only visible action, error, or consequence.

## Layout

The authenticated shell owns the off-canvas sidebar, sticky top bar, skip link, and one main content outlet. Feature routes compose inside that shell. Do not build a second page-level navigation system inside a route.

Standard pages use a 1240px maximum content width with 20px, 32px, and 40px horizontal padding across small, medium, and large layouts. Vertical page padding grows from 32px to 48px. Keep page headers within a readable 768px measure and descriptions within roughly 672px.

The sign-in surface uses a maximum 1180px frame. Wide layouts pair one product promise with one 360–432px authentication card. Narrow layouts remove the promise column and keep the form centered. The form remains the primary task at every size.

Use real content to determine height. Fixed heights belong to controls, icons, and intentionally bounded previews. Let cards, forms, tables, translations, and error messages grow. Prefer grid and flex reflow over duplicated mobile markup.

All touch controls are at least 44px when they stand alone. A compact 40px control is acceptable inside a desktop tool row when its focus and target remain clear. Preserve 320px minimum viewport support; do not copy Pandai's 390px design frame as a browser minimum.

## Elevation & Depth

Depth communicates hierarchy through space, tone, and a one-pixel semantic line. Standard cards, forms, page heroes, and containers do not use decorative elevation.

Approved uses are:

- A restrained shadow on an overlay when it must separate from moving content.
- An inset or outside shadow only when it reproduces a required stroke without changing geometry.

Do not stack a strong border, ring, and shadow on one surface. Do not add gradients, glass panels, colored halos, or decorative glow. Backdrop blur belongs only to an overlay with content moving behind it.

## Shapes

Shape follows interaction:

- Inputs and compact controls use 10px corners.
- Navigation items and grouped secondary surfaces use 12px corners.
- Standard cards and the top bar use 16px corners.
- The sign-in card uses 20px corners.
- Feature workspaces may use 28px corners when one large surface owns the page.
- Buttons, avatars, status pills, and progress tracks may use a full pill.

Do not turn every container into a pill. Do not nest three different radii without hierarchy. Child surfaces normally step down one radius from their parent.

Use Lucide for interface icons because it is the installed P&AI library. Keep the 1.5px stroke used in navigation and preserve the library view box. Use exact brand assets for provider or company marks. Never hand-draw a replacement icon because it is convenient.

## Components

Buttons perform actions; links navigate. The shared primary button uses Pandai dark teal, a white label, and a full pill. Hover changes to Pandai green with graphite text. This state pairing preserves readable contrast while retaining Pandai's color identity. The sign-in submit intentionally follows the form's 10px control geometry. Pressed feedback may scale to 96% for 150ms. Focus uses a visible three-pixel ring or a two-pixel outline with offset. Disabled state removes interaction and reduces emphasis without hiding the label.

Outline buttons use a surface fill, a semantic line, and ink text. Use ghost buttons only for low-risk actions inside an already visible context. Destructive actions use the destructive semantic color and require explicit copy. Do not make two adjacent actions look primary.

Cards group one coherent task or dataset. Default cards use a white surface, one semantic green line, and 16px corners. Use a muted green fill to group secondary facts before adding another nested card. A card is not clickable unless the whole card is one named control.

Navigation uses Pandai dark teal as the stable frame. Resting items use light mint text. Hover raises contrast without changing geometry. The active page uses Pandai green with graphite text and `aria-current="page"`. Dropdown triggers expose `aria-haspopup` and `aria-expanded`. Mobile navigation closes after successful navigation.

Fields keep labels visible above controls. Use 44px height, 10px corners, surface fill, and a semantic border. Focus changes both border and ring. Errors appear near the field or form summary with a specific recovery action. Placeholder text never replaces a label.

Status chips are compact summaries, not the only explanation. Pair their color with text or an icon. Keep lifecycle concepts distinct: draft, review, published, failed, and synthetic are not interchangeable.

Motion is direct and brief. Use the existing 150ms transition and ease curve for hover, focus, and press feedback. Respect `prefers-reduced-motion`; remove scale and spatial movement while preserving state changes. Do not animate routine data refreshes.

Every component must preserve keyboard access, visible focus, an accessible name, and correct semantic ownership. Hidden navigation is not authorization. Preview data is not test evidence. Synthetic people, identifiers, and learning outcomes must remain visibly synthetic.

## Do's and Don'ts

- Do keep the learner and the operational consequence visible in the main reading flow.
- Do provide one authoritative next action for the current blocker.
- Do reuse `AdminPageSection`, `AdminSurface`, shared primitives, and semantic tokens.
- Do inspect the exact parent component before changing nested type, spacing, or icon behavior.
- Do keep English, Bahasa Melayu, and mixed-language copy able to grow.
- Do use tabular figures for metrics and stable alignment for changing values.
- Don't introduce page-local font families that bypass the global Poppins token.
- Do treat Poppins and the verified Pandai student-context semantic colors as implemented runtime truth.
- Don't claim teacher pink, parent yellow, dark mode, or unverified Pandai component geometry is implemented.
- Don't introduce a new visual dialect through page-local hardcoded colors.
- Don't use gradients or decorative glass in the authenticated workspace.
- Don't use badge piles, generic card grids, or decorative eyebrows as page structure.
- Don't encode status, permission, freshness, or risk through color alone.
- Don't present preview, cached, or synthetic state as current production evidence.
