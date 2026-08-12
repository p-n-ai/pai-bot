import type { SVGProps } from 'react'

/** Icon names copied from the Pandai Design System 1.5 Student outline set. */
export type PandaiIconName =
  | 'activity'
  | 'alert-triangle'
  | 'arrow-left'
  | 'award'
  | 'bar-chart-3'
  | 'bell'
  | 'book-open'
  | 'calendar'
  | 'check'
  | 'check-circle'
  | 'chevron-down'
  | 'chevron-left'
  | 'chevron-right'
  | 'copy'
  | 'credit-card'
  | 'download'
  | 'edit-3'
  | 'external-link'
  | 'file-text'
  | 'globe'
  | 'heart'
  | 'home'
  | 'info'
  | 'layout'
  | 'layers'
  | 'log-out'
  | 'mail'
  | 'menu'
  | 'message-square'
  | 'minus'
  | 'more-horizontal'
  | 'mortar-board'
  | 'plus'
  | 'refresh-cw'
  | 'rotate-ccw'
  | 'search'
  | 'settings'
  | 'shield'
  | 'star'
  | 'trending-up'
  | 'users'
  | 'x'
  | 'x-circle'
  | 'zap'

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
