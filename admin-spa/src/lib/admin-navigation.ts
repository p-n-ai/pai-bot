import {
  BarChart3Icon,
  BookOpenCheckIcon,
  BotIcon,
  CableIcon,
  DownloadIcon,
  GaugeIcon,
  MessageCircleMoreIcon,
  Settings2Icon,
  UsersIcon,
  WalletCardsIcon,
} from 'lucide-react'

import { canAccessPath } from './rbac'
import type { AuthUser } from './auth-types'

const navigationGroups = [
  {
    label: 'Teaching',
    items: [
      {
        Icon: GaugeIcon,
        href: '/dashboard',
        label: 'Today',
      },
      {
        Icon: BookOpenCheckIcon,
        href: '/dashboard/classes',
        label: 'My classes',
      },
      {
        Icon: BarChart3Icon,
        href: '/dashboard/metrics',
        label: 'Learning progress',
      },
    ],
  },
  {
    label: 'School administration',
    items: [
      {
        Icon: UsersIcon,
        href: '/settings/users',
        label: 'Staff access',
      },
      {
        Icon: WalletCardsIcon,
        href: '/settings/budget',
        label: 'AI budget',
      },
      {
        Icon: DownloadIcon,
        href: '/export',
        label: 'Download records',
      },
    ],
  },
  {
    label: 'Technical tools',
    items: [
      {
        Icon: BotIcon,
        href: '/dashboard/ai-usage',
        label: 'AI activity',
      },
      {
        // Visible only with can_manage_ai_settings (canAccessPath filter).
        Icon: Settings2Icon,
        href: '/settings/ai',
        label: 'AI settings',
      },
      {
        Icon: MessageCircleMoreIcon,
        href: '/settings/whatsapp',
        label: 'WhatsApp setup',
      },
      {
        Icon: CableIcon,
        href: '/settings/embed',
        label: 'Website chat',
      },
    ],
  },
]

export type NavigationItem = (typeof navigationGroups)[number]['items'][number]

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
