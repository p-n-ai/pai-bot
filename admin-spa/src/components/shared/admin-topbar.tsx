import { useRouterState } from '@tanstack/react-router'

import { useAuth } from '@/auth-provider'
import { SidebarTrigger } from '@/components/ui/sidebar'
import { getAdminPageLabel } from '@/lib/admin-page-label'
import { getAdminUserInitials } from '@/lib/admin-user-label'

export function AdminTopbar() {
  const { auth } = useAuth()
  const pathname = useRouterState({
    select: (state) => state.location.pathname,
  })
  const user = auth.session?.user ?? null
  const pageLabel = getAdminPageLabel(pathname)

  return (
    <header className='sticky top-3 z-20 mx-3 mt-3 flex min-h-14 items-center justify-between gap-3 rounded-2xl border border-[var(--admin-line)] bg-[color:var(--admin-surface)]/90 px-3 shadow-[0_8px_30px_oklch(0.25_0.015_150/0.06)] backdrop-blur-xl sm:mx-5 sm:px-5 lg:mx-7 lg:px-6'>
      <div className='flex min-w-0 items-center gap-2 sm:gap-3'>
        <SidebarTrigger
          aria-label='Open navigation'
          className='size-11 shrink-0 text-[var(--admin-ink-soft)] focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-current focus-visible:ring-0 md:hidden'
        />
        <div className='min-w-0 leading-tight'>
          <p className='truncate text-[10px] font-semibold tracking-[0.12em] text-[var(--admin-muted)] uppercase sm:text-[11px]'>
            {user?.tenant_name ?? 'School workspace'}
          </p>
          <p className='truncate text-sm font-semibold text-[var(--admin-ink)] sm:text-[15px]'>
            {pageLabel}
          </p>
        </div>
      </div>
      <div className='flex min-h-10 min-w-0 items-center gap-2 rounded-full border border-[var(--admin-line)] bg-[var(--admin-surface-muted)] py-1 pr-3 pl-1'>
        <span className='flex size-8 shrink-0 items-center justify-center rounded-full bg-[var(--admin-ink)] text-[10px] font-semibold text-[var(--admin-accent)]'>
          {getAdminUserInitials(user)}
        </span>
        <span className='hidden max-w-40 truncate text-xs font-medium text-[var(--admin-ink-soft)] sm:block'>
          {user?.name ?? 'Administrator'}
        </span>
      </div>
    </header>
  )
}
