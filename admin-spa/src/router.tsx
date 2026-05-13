import { createRouter } from '@tanstack/react-router'

import { initialAuthContext } from './auth-provider'
import { routeTree } from './routeTree.gen'

export const router = createRouter({
  routeTree,
  context: initialAuthContext,
})

declare module '@tanstack/react-router' {
  interface Register {
    router: typeof router
  }
}
