import { Link, useRouterState } from '@tanstack/react-router'
import { LogOutIcon, SparklesIcon, XIcon } from 'lucide-react'
import { useCallback } from 'react'

import type { AuthUser } from '@/lib/auth-types'
import type { NavigationItem } from '@/lib/admin-navigation'
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
  SidebarMenu,
  SidebarMenuButton,
  SidebarMenuItem,
  useSidebar,
} from '@/components/ui/sidebar'
import {
  getVisibleNavigationGroups,
  isNavigationItemActive,
} from '@/lib/admin-navigation'
import { logout } from '@/lib/auth-client'
import { getAdminUserInitials } from '@/lib/admin-user-label'

const dashboardSearch = { student: undefined }

/** Renders role-aware navigation and session controls for the admin workspace. */
export function AdminSidebar() {
  const { auth, setAnonymousSession } = useAuth()
  const { isMobile, setOpenMobile } = useSidebar()
  const pathname = useRouterState({
    select: (state) => state.location.pathname,
  })
  const user = auth.session?.user ?? null
  const visibleGroups = getVisibleNavigationGroups(user)
  const handleCloseNavigation = useCallback(() => {
    setOpenMobile(false)
  }, [setOpenMobile])

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
      <SidebarHeader className='px-4 pt-4 pb-3'>
        <div className='flex min-h-11 items-center gap-2'>
          <Link
            className='flex min-w-0 flex-1 items-center gap-3 rounded-xl px-2 py-1 text-sm font-semibold text-[#101828] no-underline transition-[background-color,box-shadow,transform] duration-150 ease-[cubic-bezier(0.23,1,0.32,1)] hover:bg-[#f8fafc] focus-visible:ring-3 focus-visible:ring-[#2f80ed]/35 focus-visible:outline-none active:scale-[0.96] motion-reduce:transform-none motion-reduce:transition-none'
            onClick={handleCloseNavigation}
            preload='intent'
            search={dashboardSearch}
            to='/dashboard'
          >
            <span className='relative flex size-9 shrink-0 items-center justify-center overflow-hidden rounded-xl bg-[#101828] text-white shadow-[0_1px_2px_rgb(16_24_40/0.15),0_4px_12px_rgb(16_24_40/0.12)]'>
              <SparklesIcon aria-hidden='true' className='size-[18px]' />
              <span className='absolute right-1.5 bottom-1.5 size-1.5 rounded-full bg-[#53d3a0] ring-2 ring-[#101828]' />
            </span>
            <span className='min-w-0 leading-tight'>
              <span className='block truncate'>P&AI Bot</span>
              <span className='mt-0.5 block truncate text-[10px] font-medium tracking-[0.08em] text-[#98a2b3] uppercase'>
                Learning OS
              </span>
            </span>
          </Link>
          {isMobile && (
            <Button
              aria-label='Close navigation'
              className='size-11 shrink-0 text-[#667085] focus-visible:ring-[#2f80ed]/35'
              onClick={handleCloseNavigation}
              size='icon'
              type='button'
              variant='ghost'
            >
              <XIcon aria-hidden='true' />
            </Button>
          )}
        </div>
      </SidebarHeader>

      <SidebarContent className='px-4'>
        <nav aria-label='Admin navigation'>
          {visibleGroups.map((group) => (
            <SidebarGroup className='px-0 py-1.5' key={group.label}>
              <SidebarGroupLabel className='h-6 px-3 text-[10px] font-semibold tracking-[0.12em] text-[#98a2b3] uppercase'>
                {group.label}
              </SidebarGroupLabel>
              <SidebarGroupContent>
                <SidebarMenu>
                  {group.items.map((item) => (
                    <AdminNavigationLink
                      item={item}
                      key={item.href}
                      onNavigate={handleCloseNavigation}
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
        <div className='flex items-center gap-3 rounded-2xl bg-[#f8fafc] p-2.5'>
          <span className='flex size-9 shrink-0 items-center justify-center rounded-lg bg-[#eaf2ff] text-xs font-semibold text-[#175cd3]'>
            {getAdminUserInitials(user)}
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
            className='size-11 text-[#667085] hover:bg-white hover:text-[#101828] focus-visible:ring-[#2f80ed]/35 md:size-8'
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

function AdminNavigationLink({
  item,
  onNavigate,
  pathname,
}: {
  item: NavigationItem
  onNavigate: () => void
  pathname: string
}) {
  const { Icon, href, label } = item
  const isActive = isNavigationItemActive(href, pathname)

  return (
    <SidebarMenuItem>
      <SidebarMenuButton
        asChild
        className='relative h-11 gap-3 rounded-lg px-3 text-[#667085] transition-[background-color,color,box-shadow,transform] duration-150 ease-[cubic-bezier(0.23,1,0.32,1)] hover:bg-[#f4f7fb] hover:text-[#101828] focus-visible:ring-3 focus-visible:ring-[#2f80ed]/35 active:scale-[0.96] motion-reduce:transform-none motion-reduce:transition-none data-active:bg-[#eaf2ff] data-active:font-semibold data-active:text-[#175cd3]'
        isActive={isActive}
      >
        <Link
          aria-current={isActive ? 'page' : undefined}
          onClick={onNavigate}
          preload='intent'
          search={href === '/dashboard' ? dashboardSearch : undefined}
          to={href}
        >
          <span
            aria-hidden='true'
            className='absolute left-0 h-5 w-0.5 rounded-full bg-transparent group-data-[active=true]/menu-button:bg-[#2f80ed]'
          />
          <Icon
            aria-hidden='true'
            className='text-[#98a2b3] group-data-[active=true]/menu-button:text-[#2f80ed]'
            strokeWidth={1.5}
          />
          <span>{label}</span>
        </Link>
      </SidebarMenuButton>
    </SidebarMenuItem>
  )
}

function formatRole(role: AuthUser['role'] | undefined): string {
  return role ? role.replaceAll('_', ' ') : 'Admin workspace'
}
