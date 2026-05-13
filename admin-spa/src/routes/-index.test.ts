import { describe, expect, it } from 'vitest'

import type { AuthUser } from '@/lib/auth-types'
import { getAuthenticatedRootRedirectTarget } from '@/lib/root-redirect-target'

describe('getAuthenticatedRootRedirectTarget', () => {
  it('uses the safe next path when the signed-in user can access it', () => {
    expect(getAuthenticatedRootRedirectTarget(teacherUser, '/dashboard')).toBe(
      '/dashboard',
    )
    expect(
      getAuthenticatedRootRedirectTarget(teacherUser, '/dashboard/classes'),
    ).toBe('/dashboard/classes')
  })

  it('falls back to the role default when next points back to auth entry', () => {
    expect(getAuthenticatedRootRedirectTarget(teacherUser, '/')).toBe(
      '/dashboard',
    )
    expect(getAuthenticatedRootRedirectTarget(teacherUser, '/login')).toBe(
      '/dashboard',
    )
  })

  it('falls back to the role default when the user cannot access next', () => {
    expect(getAuthenticatedRootRedirectTarget(parentUser, '/dashboard')).toBe(
      '/parents/parent-1',
    )
  })
})

const teacherUser: AuthUser = {
  role: 'teacher',
  user_id: 'teacher-1',
}

const parentUser: AuthUser = {
  role: 'parent',
  user_id: 'parent-1',
}
