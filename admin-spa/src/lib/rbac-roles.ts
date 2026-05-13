import type { AuthUser } from './auth-types'

const elevatedRoles = new Set(['teacher', 'admin', 'platform_admin'])
const adminAccessRoles = new Set([
  'teacher',
  'parent',
  'admin',
  'platform_admin',
])

export function hasAdminUIAccess(user: AuthUser | null): user is AuthUser {
  return Boolean(user?.user_id && adminAccessRoles.has(user.role))
}

export function getDefaultRouteForUser(user: AuthUser | null): string {
  if (!hasAdminUIAccess(user)) {
    return '/'
  }

  if (user.role === 'parent') {
    return `/parents/${user.user_id}`
  }

  return '/dashboard'
}

export function isElevatedRole(user: AuthUser): boolean {
  return elevatedRoles.has(user.role)
}

export function canUseSetupRoutes(user: AuthUser): boolean {
  return user.role === 'admin' || user.role === 'platform_admin'
}

export function canUseAdminDataExports(user: AuthUser): boolean {
  return user.role === 'admin' || user.role === 'platform_admin'
}
