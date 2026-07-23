import {
  Link,
  Outlet,
  createFileRoute,
  useRouterState,
} from '@tanstack/react-router'
import {
  BarChart3Icon,
  BookOpenCheckIcon,
  BotIcon,
  CableIcon,
  DownloadIcon,
  GaugeIcon,
  LogOutIcon,
  MessageCircleMoreIcon,
  SearchCodeIcon,
  Settings2Icon,
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
  SidebarTrigger,
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

const dashboardSearch = { student: undefined }

function AuthenticatedLayout() {
  return (
    <SidebarProvider className='min-h-svh w-full bg-[#f7f8fa] text-[#101828]'>
      <AdminSidebar />
      <SidebarInset className='min-h-svh min-w-0 flex-1 bg-[#f7f8fa]'>
        <AdminTopbar />
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
      {
        Icon: BookOpenCheckIcon,
        href: '/dashboard/classes',
        label: 'Classes',
      },
      {
        Icon: BarChart3Icon,
        href: '/dashboard/metrics',
        label: 'Learning metrics',
      },
      {
        Icon: BotIcon,
        href: '/dashboard/ai-usage',
        label: 'AI usage',
      },
      {
        Icon: SearchCodeIcon,
        href: '/dashboard/retrieval-lab',
        label: 'Retrieval lab',
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
        Icon: Settings2Icon,
        href: '/settings/ai',
        label: 'AI settings',
      },
      {
        Icon: MessageCircleMoreIcon,
        href: '/settings/whatsapp',
        label: 'WhatsApp',
      },
      {
        Icon: CableIcon,
        href: '/settings/embed',
        label: 'Embed',
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
      collapsible='offcanvas'
      className='border-r border-[#e6e9ef] bg-white'
    >
      <SidebarHeader className='px-4 pt-5 pb-4'>
        <Link
          className='flex items-center gap-3 rounded-xl px-2 py-2 text-sm font-semibold text-[#101828] no-underline focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-[#2f80ed]'
          search={dashboardSearch}
          to='/dashboard'
        >
          <span className='relative flex size-9 items-center justify-center overflow-hidden rounded-xl bg-[#101828] text-white shadow-[0_1px_2px_rgb(16_24_40/0.15),0_4px_12px_rgb(16_24_40/0.12)]'>
            <SparklesIcon aria-hidden='true' className='size-[18px]' />
            <span className='absolute right-1.5 bottom-1.5 size-1.5 rounded-full bg-[#53d3a0] ring-2 ring-[#101828]' />
          </span>
          <span className='leading-tight'>
            <span className='block'>P&AI Bot</span>
            <span className='mt-0.5 block text-[10px] font-medium tracking-[0.08em] text-[#98a2b3] uppercase'>
              Learning OS
            </span>
          </span>
        </Link>
      </SidebarHeader>

      <SidebarContent className='px-4'>
        <nav aria-label='Admin navigation'>
          {visibleGroups.map((group) => (
            <SidebarGroup className='px-0 py-2' key={group.label}>
              <SidebarGroupLabel className='h-6 px-3 text-[10px] font-semibold tracking-[0.12em] text-[#98a2b3] uppercase'>
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

      <SidebarFooter className='mt-auto border-t border-[#eef0f3] px-4 py-4'>
        <div className='flex items-center gap-3 rounded-xl bg-[#f8fafc] p-2.5'>
          <span className='flex size-9 shrink-0 items-center justify-center rounded-lg bg-[#eaf2ff] text-xs font-semibold text-[#175cd3]'>
            {getUserInitials(user)}
          </span>
          <div className='min-w-0 flex-1'>
            <p className='truncate text-sm font-medium text-[#101828]'>
              {user?.name ?? 'Administrator'}
            </p>
            <p className='truncate text-xs text-[#667085]'>
              {user?.tenant_name ?? formatRole(user?.role)}
            </p>
          </div>
          <Button
            aria-label='Log out'
            className='text-[#667085] hover:bg-white hover:text-[#101828]'
            onClick={handleLogout}
            size='icon-sm'
            type='button'
            variant='ghost'
          >
            <LogOutIcon aria-hidden='true' />
          </Button>
        </div>
      </SidebarFooter>
    </Sidebar>
  )
}

function AdminTopbar() {
  const { auth } = useAuth()
  const user = auth.session?.user ?? null

  return (
    <header className='sticky top-0 z-20 flex h-14 items-center justify-between border-b border-[#e6e9ef]/90 bg-white/90 px-4 backdrop-blur-xl sm:px-6 lg:px-10'>
      <div className='flex min-w-0 items-center gap-3'>
        <SidebarTrigger className='-ml-1 size-11 md:hidden' />
        <div className='min-w-0'>
          <p className='truncate text-sm font-medium text-[#344054]'>
            {user?.tenant_name ?? 'Learning operations'}
          </p>
        </div>
      </div>
      <div className='flex items-center gap-2 rounded-full border border-[#e6e9ef] bg-white py-1 pr-2.5 pl-1 shadow-[0_1px_2px_rgb(16_24_40/0.04)]'>
        <span className='flex size-6 items-center justify-center rounded-full bg-[#eaf2ff] text-[10px] font-semibold text-[#175cd3]'>
          {getUserInitials(user)}
        </span>
        <span className='hidden max-w-40 truncate text-xs font-medium text-[#475467] sm:block'>
          {user?.name ?? 'Administrator'}
        </span>
      </div>
    </header>
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
        className='relative h-10 gap-3 rounded-lg px-3 text-[#667085] transition-[background-color,color,transform] duration-150 ease-[cubic-bezier(0.23,1,0.32,1)] hover:bg-[#f4f7fb] hover:text-[#101828] active:scale-[0.98] data-active:bg-[#eaf2ff] data-active:font-semibold data-active:text-[#175cd3]'
        isActive={isActive}
      >
        <Link to={href}>
          <span
            aria-hidden='true'
            className='absolute left-0 h-5 w-0.5 rounded-full bg-transparent group-data-[active=true]/menu-button:bg-[#2f80ed]'
          />
          <Icon
            aria-hidden='true'
            className='text-[#98a2b3] group-data-[active=true]/menu-button:text-[#2f80ed]'
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

function getUserInitials(user: AuthUser | null): string {
  const label = user?.name ?? user?.email ?? 'Administrator'
  const initials = label
    .split(/\s+/)
    .filter(Boolean)
    .slice(0, 2)
    .map((part) => part[0].toUpperCase())
    .join('')

  return initials || 'A'
}

function formatRole(role: AuthUser['role'] | undefined): string {
  return role ? role.replaceAll('_', ' ') : 'Admin workspace'
}
