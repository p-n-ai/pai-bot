import { redirect } from '@tanstack/react-router'

import { canAccessPath } from './rbac'
import type { AuthState } from '@/auth-provider'

export function requireAdminPath(
  auth: AuthState,
  pathname: string,
  href: string,
) {
  const user = auth.status === 'authenticated' ? auth.session.user : null

  if (canAccessPath(user, pathname)) {
    return
  }

  throw redirect({
    to: '/',
    search: {
      next: href,
    },
  })
}
