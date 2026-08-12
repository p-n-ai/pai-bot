import { canAccessPath } from './rbac'
import type { AuthUser } from './auth-types'
import type { PandaiIconName } from '@/components/ui/pandai-icon'

export type NavigationItem = {
  readonly href: string
  readonly icon: PandaiIconName
  readonly label: string
}

type NavigationGroup = {
  readonly items: ReadonlyArray<NavigationItem>
  readonly label: string
}

const navigationGroups: ReadonlyArray<NavigationGroup> = [
  {
    label: 'Teaching',
    items: [
      {
        href: '/dashboard',
        icon: 'home',
        label: 'Today',
      },
      {
        href: '/dashboard/classes',
        icon: 'book-open',
        label: 'Classes',
      },
    ],
  },
  {
    label: 'School administration',
    items: [
      {
        href: '/settings/users',
        icon: 'users',
        label: 'Staff access',
      },
      {
        href: '/settings/budget',
        icon: 'credit-card',
        label: 'AI budget',
      },
      {
        href: '/export',
        icon: 'download',
        label: 'Download records',
      },
    ],
  },
  {
    label: 'Technical tools',
    items: [
      {
        href: '/build-ai',
        icon: 'layers',
        label: 'Build AI',
      },
      {
        href: '/dashboard/ai-usage',
        icon: 'activity',
        label: 'AI usage',
      },
      {
        // Visible only with can_manage_ai_settings (canAccessPath filter).
        href: '/settings/ai',
        icon: 'settings',
        label: 'AI settings',
      },
      {
        href: '/settings/embed',
        icon: 'globe',
        label: 'Website chat',
      },
    ],
  },
]

export function getVisibleNavigationGroups(user: AuthUser | null) {
  return navigationGroups
    .map((group) => ({
      label: group.label,
      items: group.items.filter((item) => canAccessPath(user, item.href)),
    }))
    .filter(({ items }) => items.length > 0)
}

export function isNavigationItemActive(
  href: string,
  pathname: string,
): boolean {
  return (
    pathname === href ||
    (href !== '/dashboard' && pathname.startsWith(`${href}/`))
  )
}
