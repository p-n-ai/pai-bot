import {
  Link,
  Outlet,
  createFileRoute,
  useRouterState,
} from '@tanstack/react-router'
import {
  DownloadIcon,
  GaugeIcon,
  LogOutIcon,
  SparklesIcon,
  UsersIcon,
  WalletCardsIcon,
} from 'lucide-react'
import { useCallback } from 'react'

import type { AuthUser } from '@/lib/auth-types'
import { useAuth } from '@/auth-provider'
import { Button } from '@/components/ui/button'
import {
  Sidebar,
  SidebarContent,
  SidebarFooter,
  SidebarGroup,
  SidebarGroupContent,
  SidebarGroupLabel,
  SidebarHeader,
  SidebarInset,
  SidebarMenu,
  SidebarMenuButton,
  SidebarMenuItem,
  SidebarProvider,
} from '@/components/ui/sidebar'
import { logout } from '@/lib/auth-client'
import { canAccessPath } from '@/lib/rbac'
import { requireAdminPath } from '@/lib/router-guards'

export const Route = createFileRoute('/_authenticated')({
  beforeLoad: ({ context, location }) => {
    requireAdminPath(context.auth, location.pathname, location.href)
  },
  component: AuthenticatedLayout,
})

function AuthenticatedLayout() {
  return (
    <SidebarProvider className='min-h-svh w-full bg-[#fafaf9] text-[#0c0a09]'>
      <AdminSidebar />
      <SidebarInset className='min-h-svh min-w-0 flex-1 bg-[#fafaf9]'>
        <Outlet />
      </SidebarInset>
    </SidebarProvider>
  )
}

const navigationGroups = [
  {
    label: 'Workspace',
    items: [
      {
        Icon: GaugeIcon,
        href: '/dashboard',
        label: 'Dashboard',
      },
    ],
  },
  {
    label: 'Manage',
    items: [
      {
        Icon: WalletCardsIcon,
        href: '/settings/budget',
        label: 'Budget',
      },
      {
        Icon: UsersIcon,
        href: '/settings/users',
        label: 'Users',
      },
      {
        // Visible only with can_manage_ai_settings (canAccessPath filter).
        Icon: SparklesIcon,
        href: '/settings/ai',
        label: 'AI',
      },
      {
        Icon: DownloadIcon,
        href: '/export',
        label: 'Export',
      },
    ],
  },
]

type NavigationItem = (typeof navigationGroups)[number]['items'][number]

function AdminSidebar() {
  const { auth, setAnonymousSession } = useAuth()
  const pathname = useRouterState({
    select: (state) => state.location.pathname,
  })
  const user = auth.session?.user ?? null
  const visibleGroups = getVisibleNavigationGroups(user)

  const handleLogout = useCallback(() => {
    logout()
      .catch(() => {})
      .finally(() => {
        setAnonymousSession()
        window.location.assign('/')
      })
  }, [setAnonymousSession])

  return (
    <Sidebar
      collapsible='none'
      className='sticky top-0 h-svh shrink-0 border-r border-[#e5e7eb] bg-white'
    >
      <SidebarHeader className='px-4 py-5'>
        <Link
          className='flex items-center gap-3 rounded-lg px-2 py-2 text-sm font-semibold text-[#0c0a09] no-underline'
          search={{ student: undefined }}
          to='/dashboard'
        >
          <span className='flex size-9 items-center justify-center rounded-lg bg-[#3ba6f1] text-sm font-semibold text-white shadow-[rgba(0,0,0,0.05)_0px_1px_2px_0px]'>
            P
          </span>
          <span>P&AI Bot</span>
        </Link>
      </SidebarHeader>

      <SidebarContent className='px-4'>
        <nav aria-label='Admin navigation'>
          {visibleGroups.map((group) => (
            <SidebarGroup className='px-0 py-2' key={group.label}>
              <SidebarGroupLabel className='h-5 px-3 text-[11px] tracking-[0.08em] text-[#a8a29e] uppercase'>
                {group.label}
              </SidebarGroupLabel>
              <SidebarGroupContent>
                <SidebarMenu>
                  {group.items.map((item) => (
                    <AdminNavigationLink
                      item={item}
                      key={item.href}
                      pathname={pathname}
                    />
                  ))}
                </SidebarMenu>
              </SidebarGroupContent>
            </SidebarGroup>
          ))}
        </nav>
      </SidebarContent>

      <SidebarFooter className='mt-auto px-4 py-5'>
        <Button
          className='w-full rounded-full'
          onClick={handleLogout}
          type='button'
          variant='outline'
        >
          <LogOutIcon aria-hidden='true' data-icon='inline-start' />
          Log out
        </Button>
      </SidebarFooter>
    </Sidebar>
  )
}

function isNavigationItemVisible(item: NavigationItem, user: AuthUser | null) {
  return canAccessPath(user, item.href)
}

function getVisibleNavigationGroups(user: AuthUser | null) {
  return navigationGroups
    .map((group) => createVisibleNavigationGroup(group, user))
    .filter(hasNavigationItems)
}

function createVisibleNavigationGroup(
  group: (typeof navigationGroups)[number],
  user: AuthUser | null,
) {
  return {
    label: group.label,
    items: group.items.filter((item) => isNavigationItemVisible(item, user)),
  }
}

function hasNavigationItems(group: { items: Array<NavigationItem> }) {
  return group.items.length > 0
}

function AdminNavigationLink({
  item,
  pathname,
}: {
  item: NavigationItem
  pathname: string
}) {
  const { Icon, href, label } = item
  const isActive = isNavigationItemActive(href, pathname)

  return (
    <SidebarMenuItem>
      <SidebarMenuButton
        asChild
        className='relative h-10 gap-3 rounded-lg px-3 text-[#78716c] hover:bg-[#fafaf9] hover:text-[#0c0a09] data-active:bg-[#e8f5fd] data-active:text-[#0c0a09]'
        isActive={isActive}
      >
        <Link to={href}>
          <span
            aria-hidden='true'
            className='absolute left-0 h-5 w-0.5 rounded-full bg-transparent group-data-[active=true]/menu-button:bg-[#3ba6f1]'
          />
          <Icon
            aria-hidden='true'
            className='text-[#a8a29e] group-data-[active=true]/menu-button:text-[#3ba6f1]'
          />
          <span>{label}</span>
        </Link>
      </SidebarMenuButton>
    </SidebarMenuItem>
  )
}

function isNavigationItemActive(href: string, pathname: string) {
  return (
    pathname === href ||
    (href !== '/dashboard' && pathname.startsWith(`${href}/`))
  )
}
