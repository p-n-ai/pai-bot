import {
  canUseAdminDataExports,
  canUseSetupRoutes,
  isElevatedRole,
} from './rbac-roles'
import type { AuthUser } from './auth-types'

const elevatedPrefixes = ['/dashboard', '/settings', '/students', '/parents']
const publicExactPaths = new Set(['/', '/activate'])

export function isPublicPath(pathname: string): boolean {
  return publicExactPaths.has(pathname) || pathname.startsWith('/join/')
}

export function isSafeRedirectPath(
  pathname: string | undefined,
): pathname is string {
  return Boolean(
    pathname && isLocalPath(pathname) && !isAuthEntryPath(pathname),
  )
}

export function isRoleRoute(user: AuthUser, pathname: string): boolean {
  if (!isElevatedRole(user)) {
    return isParentSelfRoute(user, pathname)
  }

  return isElevatedRoute(user, pathname)
}

function isLocalPath(pathname: string): boolean {
  return pathname.startsWith('/') && !pathname.startsWith('//')
}

function isAuthEntryPath(pathname: string): boolean {
  return pathname === '/' || pathname.startsWith('/login')
}

function isParentSelfRoute(user: AuthUser, pathname: string): boolean {
  const parentPath = `/parents/${user.user_id}`

  return pathname === parentPath || pathname.startsWith(`${parentPath}/`)
}

// fallow-ignore-next-line complexity
function isElevatedRoute(user: AuthUser, pathname: string): boolean {
  if (pathname.startsWith('/settings/whatsapp')) {
    return canUseSetupRoutes(user)
  }

  if (pathname.startsWith('/settings/users')) {
    return canUseSetupRoutes(user)
  }

  if (pathname.startsWith('/settings/budget')) {
    return canUseSetupRoutes(user)
  }

  if (pathname.startsWith('/settings/embed')) {
    return canUseSetupRoutes(user)
  }

  return (
    elevatedPrefixes.some((prefix) => pathname.startsWith(prefix)) ||
    isDataExportRoute(user, pathname) ||
    isSetupRoute(user, pathname)
  )
}

function isDataExportRoute(user: AuthUser, pathname: string): boolean {
  return canUseAdminDataExports(user) && pathname.startsWith('/export')
}

function isSetupRoute(user: AuthUser, pathname: string): boolean {
  return canUseSetupRoutes(user) && pathname.startsWith('/setup')
}
