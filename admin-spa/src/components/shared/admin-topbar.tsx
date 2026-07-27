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
    <header className='sticky top-0 z-20 flex min-h-16 items-center justify-between gap-3 border-b border-[#e6e9ef]/90 bg-white/95 px-3 backdrop-blur-xl sm:px-6 lg:px-10'>
      <div className='flex min-w-0 items-center gap-2 sm:gap-3'>
        <SidebarTrigger
          aria-label='Open navigation'
          className='size-11 shrink-0 text-[#475467] focus-visible:ring-[#2f80ed]/35 md:hidden'
        />
        <div className='min-w-0 leading-tight'>
          <p className='truncate text-[11px] font-medium text-[#98a2b3] sm:text-xs'>
            {user?.tenant_name ?? 'School workspace'}
          </p>
          <p className='truncate text-sm font-semibold text-[#101828] sm:text-[15px]'>
            {pageLabel}
          </p>
        </div>
      </div>
      <div className='flex min-h-9 min-w-0 items-center gap-2 rounded-full bg-white py-1 pr-2.5 pl-1 shadow-[0_0_0_1px_rgb(16_24_40/0.08),0_1px_2px_rgb(16_24_40/0.04)]'>
        <span className='flex size-7 shrink-0 items-center justify-center rounded-full bg-[#eaf2ff] text-[10px] font-semibold text-[#175cd3]'>
          {getAdminUserInitials(user)}
        </span>
        <span className='hidden max-w-40 truncate text-xs font-medium text-[#475467] sm:block'>
          {user?.name ?? 'Administrator'}
        </span>
      </div>
    </header>
  )
}
