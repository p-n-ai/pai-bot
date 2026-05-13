import {
  canAccessPath,
  getDefaultRouteForUser,
  isSafeRedirectPath,
} from './rbac'
import type { AuthUser } from './auth-types'

export function getAuthenticatedRootRedirectTarget(
  user: AuthUser,
  nextPath: string | undefined,
) {
  if (isSafeRedirectPath(nextPath) && canAccessPath(user, nextPath)) {
    return nextPath
  }

  return getDefaultRouteForUser(user)
}
