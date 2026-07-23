import type { AuthUser } from './auth-types'

/** Returns a compact, deterministic label for an admin avatar. */
export function getAdminUserInitials(user: AuthUser | null): string {
  const label = user?.name ?? user?.email ?? 'Administrator'
  const initials = label
    .split(/\s+/)
    .filter(Boolean)
    .slice(0, 2)
    .map((part) => part[0].toUpperCase())
    .join('')

  return initials || 'A'
}
