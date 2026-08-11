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
    <SidebarProvider className='admin-workspace min-h-svh w-full bg-[var(--admin-canvas)] text-[var(--admin-ink)] [--sidebar-foreground:var(--admin-nav-text)] [--sidebar:var(--admin-navigation)]'>
      <a
        className='fixed top-3 left-3 z-50 -translate-y-20 rounded-full bg-[var(--admin-navigation)] px-4 py-3 text-sm font-semibold text-[var(--admin-nav-text)] transition-transform duration-150 ease-[cubic-bezier(0.23,1,0.32,1)] focus:translate-y-0 focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-current motion-reduce:transition-none'
        href='#admin-content'
      >
        Skip to content
      </a>
      <AdminSidebar />
      <SidebarInset
        className='relative min-h-svh min-w-0 flex-1 overflow-x-hidden bg-transparent'
        id='admin-content'
        tabIndex={-1}
      >
        <AdminTopbar />
        <Outlet />
      </SidebarInset>
    </SidebarProvider>
  )
}
