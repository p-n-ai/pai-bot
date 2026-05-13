import { describe, expect, it } from 'vitest'

import { requireAdminPath } from './router-guards'
import type { AuthState } from '@/auth-provider'

const anonymousAuth: AuthState = {
  status: 'anonymous',
  session: null,
  error: null,
}

const adminAuth: AuthState = {
  status: 'authenticated',
  session: {
    expires_at: '2026-05-08T00:00:00Z',
    user: {
      user_id: 'admin_1',
      role: 'admin',
    },
  },
  error: null,
}

describe('requireAdminPath', () => {
  it('redirects signed-out users to the root login page with the attempted path', () => {
    expect(() =>
      requireAdminPath(anonymousAuth, '/dashboard', '/dashboard'),
    ).toThrow(
      expect.objectContaining({
        options: expect.objectContaining({
          to: '/',
          search: {
            next: '/dashboard',
          },
        }),
      }),
    )
  })

  it('allows elevated admin routes', () => {
    expect(() =>
      requireAdminPath(adminAuth, '/dashboard', '/dashboard'),
    ).not.toThrow()
  })

  it('allows elevated dashboard child routes', () => {
    expect(() =>
      requireAdminPath(adminAuth, '/dashboard/classes', '/dashboard/classes'),
    ).not.toThrow()
  })
})
