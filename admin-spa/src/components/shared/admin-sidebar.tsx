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

const dashboardSearch = { class: undefined, student: undefined }

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
      className='border-r-0 bg-[var(--admin-navigation)] text-[var(--admin-nav-text)]'
    >
      <SidebarHeader className='px-4 pt-5 pb-4'>
        <div className='flex min-h-11 items-center gap-2'>
          <Link
            className='flex min-w-0 flex-1 items-center gap-3 rounded-xl px-2 py-1 text-sm font-semibold text-[var(--admin-nav-text)] no-underline transition-[background-color,box-shadow,transform] duration-150 ease-[cubic-bezier(0.23,1,0.32,1)] hover:bg-white/6 focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-current active:scale-[0.96] motion-reduce:transform-none motion-reduce:transition-none'
            onClick={handleCloseNavigation}
            preload='intent'
            search={dashboardSearch}
            to='/dashboard'
          >
            <span className='relative flex size-9 shrink-0 items-center justify-center overflow-hidden rounded-xl bg-[var(--admin-accent)] text-[var(--admin-ink)] ring-1 ring-white/30'>
              <SparklesIcon aria-hidden='true' className='size-[18px]' />
              <span className='absolute right-1.5 bottom-1.5 size-1.5 rounded-full bg-[var(--admin-ink)] ring-2 ring-[var(--admin-accent)]' />
            </span>
            <span className='min-w-0 leading-tight'>
              <span className='block truncate'>P&AI Bot</span>
              <span className='mt-0.5 block truncate text-xs font-medium tracking-[0.08em] text-[var(--admin-nav-muted)] uppercase'>
                Learning OS
              </span>
            </span>
          </Link>
          {isMobile && (
            <Button
              aria-label='Close navigation'
              className='size-11 shrink-0 text-[var(--admin-nav-muted)] hover:bg-white/8 hover:text-white focus-visible:ring-0 focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-current'
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

      <SidebarContent className='px-3'>
        <nav aria-label='Admin navigation'>
          {visibleGroups.map((group) => (
            <SidebarGroup className='px-0 py-2' key={group.label}>
              <SidebarGroupLabel className='h-7 px-3 text-xs font-semibold tracking-[0.1em] text-[var(--admin-nav-label)] uppercase'>
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

      <SidebarFooter className='mt-auto border-t border-white/8 px-3 py-4'>
        <div className='flex items-center gap-3 rounded-2xl border border-white/8 bg-white/5 p-2.5'>
          <span className='flex size-10 shrink-0 items-center justify-center rounded-full bg-[var(--admin-accent)] text-xs font-semibold text-[var(--admin-ink)]'>
            {getAdminUserInitials(user)}
          </span>
          <div className='min-w-0 flex-1'>
            <p className='truncate text-sm font-medium text-[var(--admin-nav-text)]'>
              {user?.name ?? 'Administrator'}
            </p>
            <p className='truncate text-xs text-[var(--admin-nav-muted)]'>
              {user?.tenant_name ?? formatRole(user?.role)}
            </p>
          </div>
          <Button
            aria-label='Log out'
            className='size-11 text-[var(--admin-nav-muted)] hover:bg-white/8 hover:text-white focus-visible:ring-0 focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-current'
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
        className='relative h-11 gap-3 rounded-xl px-3 text-[var(--admin-nav-muted)] transition-[background-color,color,box-shadow,transform] duration-150 ease-[cubic-bezier(0.23,1,0.32,1)] hover:bg-white/7 hover:text-white focus-visible:ring-0 focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-current active:scale-[0.96] motion-reduce:transform-none motion-reduce:transition-none data-active:bg-[var(--admin-accent)] data-active:font-semibold data-active:text-[var(--admin-ink)]'
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
            className='absolute left-0 h-5 w-0.5 rounded-full bg-transparent'
          />
          <Icon
            aria-hidden='true'
            className='text-[var(--admin-nav-label)] group-data-[active=true]/menu-button:text-[var(--admin-ink)]'
            strokeWidth={1.5}
          />
          <span>{label}</span>
        </Link>
      </SidebarMenuButton>
    </SidebarMenuItem>
  )
}

function formatRole(role: AuthUser['role'] | undefined): string {
  return role ? role.replaceAll('_', ' ') : 'Administrator'
}
