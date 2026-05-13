import { Outlet, createRootRouteWithContext } from '@tanstack/react-router'

import type { AuthContextValue } from '@/auth-provider'
import { TooltipProvider } from '@/components/ui/tooltip'

export const Route = createRootRouteWithContext<AuthContextValue>()({
  component: RootLayout,
})

function RootLayout() {
  return (
    <TooltipProvider>
      <a className='skip-link' href='#main-content'>
        Skip to main content
      </a>
      <main id='main-content' className='app-shell'>
        <Outlet />
      </main>
    </TooltipProvider>
  )
}
