---
name: P&AI Bot admin workspace
description: A calm, capability-aware control plane for operating a school Tutor.
colors:
  canvas: "oklch(0.955 0.01 90)"
  surface: "oklch(0.995 0.006 90)"
  surface-muted: "oklch(0.968 0.009 90)"
  line: "oklch(0.87 0.012 85)"
  ink: "oklch(0.225 0.018 150)"
  ink-soft: "oklch(0.445 0.018 150)"
  muted: "oklch(0.52 0.012 150)"
  accent: "oklch(0.925 0.175 115)"
  nav-text: "oklch(0.935 0.01 145)"
typography:
  headline:
    fontFamily: "Geist, Inter, ui-sans-serif, system-ui, sans-serif"
    fontSize: "1.875rem"
    fontWeight: 600
    lineHeight: 1.2
    letterSpacing: "-0.025em"
  body:
    fontFamily: "Geist, Inter, ui-sans-serif, system-ui, sans-serif"
    fontSize: "1rem"
    fontWeight: 400
    lineHeight: 1.75
  label:
    fontFamily: "Geist, Inter, ui-sans-serif, system-ui, sans-serif"
    fontSize: "0.875rem"
    fontWeight: 500
    lineHeight: 1.5
rounded:
  control: "0.625rem"
  surface: "0.75rem"
  prominent: "1rem"
spacing:
  xs: "0.5rem"
  sm: "0.75rem"
  md: "1rem"
  lg: "1.5rem"
  xl: "2rem"
components:
  primary-button:
    backgroundColor: "{colors.ink}"
    textColor: "{colors.surface}"
    rounded: "{rounded.control}"
    padding: "0.625rem 1rem"
  operator-surface:
    backgroundColor: "{colors.surface}"
    textColor: "{colors.ink}"
    rounded: "{rounded.surface}"
    padding: "1.5rem"
---

## Overview

The admin SPA is an **Operate** interface: quiet, explicit, and task-first. Its visual character comes from dark botanical navigation, warm neutral work surfaces, lime emphasis, compact status language, and strong reading order. Build AI extends that world rather than introducing a separate identity. Pandai Design System 1.5 is the approved future authority, but the repository currently implements only the locally defined semantic tokens above; private Pandai assets and fonts have not been copied.

## Colors

Use `.admin-workspace` variables from `admin-spa/src/styles.css` as the implemented source of truth. Canvas and surfaces are warm near-neutrals; ink is a deep green-black; the lime accent identifies active navigation and focused operator context. Status meaning must include text or an icon, never color alone. Do not invent interaction colors or present synthetic state as production evidence.

## Typography

Use the installed Geist variable face with Inter and system fallbacks. Headings are compact, semibold, and slightly tightened; body copy remains comfortable at roughly 65–75 characters. Breadcrumbs provide context semantically and must not become decorative eyebrows. English, Bahasa Melayu, and mixed-language copy must grow without truncating essential actions.

## Layout

The authenticated shell owns the primary sidebar, top bar, skip link, and single content landmark. Feature routes stay inside that shell. Build AI uses one Tutor and six stable destinations: Overview, Curriculum, Teaching, Test tutor, Publish, and Activity. Desktop pairs a narrow local destination rail with content; narrow screens collapse navigation and open Curriculum or Teaching preview as a dedicated full-screen layer. Controls target at least 44px on touch layouts.

## Elevation & Depth

Prefer flat, border-led surfaces. Use elevation sparingly for the sticky top bar, overlays, and controls that must separate from moving content. Do not combine a strong border with a decorative shadow, add colored halos, or use glass effects without a functional reason.

## Shapes

Use 10–16px radii for controls and surfaces. Pills are reserved for small statuses, identities, and compact controls—not general content containers. Dividers and negative space should group related information before adding another card.

## Components

Compose the installed shadcn/Radix Nova primitives for keyboard and screen-reader behavior. Buttons name actions; links navigate. Active navigation is explicit with `aria-current`. Draft, Test results, educator review, Published versions, Version history, and Used by classes remain visually and conceptually distinct. Preview state is local and never becomes test, review, publication, or learner evidence.

## Do's and Don'ts

- Do keep scope, provenance, freshness, and consequences visible in the primary reading flow.
- Do show at most one authoritative next action for the current blocker.
- Do preserve immutable Published-version history and state that publishing changes no class automatically.
- Do label illustrative people, identifiers, and evidence as synthetic.
- Don't expose provider secrets, private learner content, or unsupported health claims.
- Don't use gradients, decorative glass, card grids as page scaffolding, or badge piles.
- Don't treat roles, hidden navigation, or display names as authorization.
