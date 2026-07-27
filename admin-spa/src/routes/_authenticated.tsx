import { Outlet, createFileRoute } from '@tanstack/react-router'

import { AdminSidebar } from '@/components/shared/admin-sidebar'
import { AdminTopbar } from '@/components/shared/admin-topbar'
import { SidebarInset, SidebarProvider } from '@/components/ui/sidebar'
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
      <a
        className='fixed top-3 left-3 z-50 -translate-y-20 rounded-lg bg-[#101828] px-4 py-3 text-sm font-semibold text-white shadow-[0_1px_2px_rgb(16_24_40/0.18),0_8px_20px_rgb(16_24_40/0.16)] transition-transform duration-150 ease-[cubic-bezier(0.23,1,0.32,1)] focus:translate-y-0 focus:ring-3 focus:ring-[#2f80ed]/40 focus:outline-none'
        href='#admin-content'
      >
        Skip to content
      </a>
      <AdminSidebar />
      <SidebarInset
        className='min-h-svh min-w-0 flex-1 bg-[#f7f8fa]'
        id='admin-content'
        tabIndex={-1}
      >
        <AdminTopbar />
        <Outlet />
      </SidebarInset>
    </SidebarProvider>
  )
}
