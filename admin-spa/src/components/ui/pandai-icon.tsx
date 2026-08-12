import type { SVGProps } from 'react'

/** Icon names copied from the Pandai Design System 1.5 Student outline set. */
export type PandaiIconName =
  | 'activity'
  | 'alert-triangle'
  | 'bell'
  | 'book-open'
  | 'check-circle'
  | 'chevron-down'
  | 'chevron-right'
  | 'credit-card'
  | 'download'
  | 'globe'
  | 'home'
  | 'layout'
  | 'log-out'
  | 'menu'
  | 'search'
  | 'settings'
  | 'star'
  | 'trending-up'
  | 'users'
  | 'x'

/** Renders an exact Pandai DS outline icon from the shared external sprite. */
export function PandaiIcon({
  'aria-hidden': ariaHidden = true,
  name,
  ...props
}: Omit<SVGProps<SVGSVGElement>, 'children'> & { name: PandaiIconName }) {
  return (
    <svg
      aria-hidden={ariaHidden}
      data-slot='pandai-icon'
      focusable='false'
      {...props}
    >
      <use href={`/icons/pandai.svg#ic-${name}`} />
    </svg>
  )
}
