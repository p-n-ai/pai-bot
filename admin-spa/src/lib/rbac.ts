import { isPublicPath, isRoleRoute, isSafeRedirectPath } from './rbac-paths'
import { getDefaultRouteForUser, hasAdminUIAccess } from './rbac-roles'
import type { AuthUser } from './auth-types'

export function canAccessPath(
  user: AuthUser | null,
  pathname: string,
): boolean {
  if (isPublicPath(pathname)) {
    return true
  }

  if (!hasAdminUIAccess(user)) {
    return false
  }

  return isRoleRoute(user, pathname)
}

export { getDefaultRouteForUser, isSafeRedirectPath }
