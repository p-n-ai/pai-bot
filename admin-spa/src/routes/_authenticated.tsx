import { Outlet, createFileRoute } from '@tanstack/react-router'

import { useAuth } from '@/auth-provider'
import { AdminSidebar } from '@/components/shared/admin-sidebar'
import {
  SidebarInset,
  SidebarProvider,
  SidebarTrigger,
} from '@/components/ui/sidebar'
import { getAdminUserInitials } from '@/lib/admin-user-label'
import { requireAdminPath } from '@/lib/router-guards'

export const Route = createFileRoute('/_authenticated')({
  beforeLoad: ({ context, location }) => {
    requireAdminPath(context.auth, location.pathname, location.href)
  },
  component: AuthenticatedLayout,
})

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
          {getAdminUserInitials(user)}
        </span>
        <span className='hidden max-w-40 truncate text-xs font-medium text-[#475467] sm:block'>
          {user?.name ?? 'Administrator'}
        </span>
      </div>
    </header>
  )
}
