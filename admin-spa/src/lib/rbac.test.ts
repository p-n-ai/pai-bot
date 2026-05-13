import { describe, expect, it } from 'vitest'

import {
  canAccessPath,
  getDefaultRouteForUser,
  isSafeRedirectPath,
} from './rbac'
import type { AuthUser } from './auth-types'

const admin: AuthUser = {
  user_id: 'admin_1',
  role: 'admin',
}

const parent: AuthUser = {
  user_id: 'parent_1',
  role: 'parent',
}

const platformAdmin: AuthUser = {
  user_id: 'platform_admin_1',
  role: 'platform_admin',
}

const teacher: AuthUser = {
  user_id: 'teacher_1',
  role: 'teacher',
}

describe('admin SPA RBAC', () => {
  it('allows public routes without a session', () => {
    expect(canAccessPath(null, '/')).toBe(true)
    expect(canAccessPath(null, '/login')).toBe(false)
    expect(canAccessPath(null, '/join/invite_1')).toBe(true)
  })

  it('matches parent self-route access from the current admin app', () => {
    expect(canAccessPath(parent, '/parents/parent_1')).toBe(true)
    expect(canAccessPath(parent, '/parents/parent_1/activity')).toBe(true)
    expect(canAccessPath(parent, '/parents/other')).toBe(false)
    expect(getDefaultRouteForUser(parent)).toBe('/parents/parent_1')
  })

  it('matches elevated dashboard and setup access from the current admin app', () => {
    expect(canAccessPath(admin, '/dashboard')).toBe(true)
    expect(canAccessPath(admin, '/settings')).toBe(true)
    expect(canAccessPath(admin, '/setup/schools')).toBe(true)
    expect(getDefaultRouteForUser(admin)).toBe('/dashboard')
  })

  it('limits data export to admin roles', () => {
    expect(canAccessPath(admin, '/export')).toBe(true)
    expect(canAccessPath(platformAdmin, '/export')).toBe(true)
    expect(canAccessPath(teacher, '/export')).toBe(false)
  })

  it('limits user management to admin roles', () => {
    expect(canAccessPath(admin, '/settings/users')).toBe(true)
    expect(canAccessPath(teacher, '/settings/users')).toBe(false)
  })

  it('limits token budget management to admin roles', () => {
    expect(canAccessPath(admin, '/settings/budget')).toBe(true)
    expect(canAccessPath(platformAdmin, '/settings/budget')).toBe(true)
    expect(canAccessPath(teacher, '/settings/budget')).toBe(false)
  })

  it('limits WhatsApp setup to admin roles', () => {
    expect(canAccessPath(admin, '/settings/whatsapp')).toBe(true)
    expect(canAccessPath(platformAdmin, '/settings/whatsapp')).toBe(true)
    expect(canAccessPath(teacher, '/settings/whatsapp')).toBe(false)
  })

  it('limits embed setup to admin roles', () => {
    expect(canAccessPath(admin, '/settings/embed')).toBe(true)
    expect(canAccessPath(platformAdmin, '/settings/embed')).toBe(true)
    expect(canAccessPath(teacher, '/settings/embed')).toBe(false)
  })

  it('rejects redirect values that could escape the admin app or loop login', () => {
    expect(isSafeRedirectPath('/dashboard')).toBe(true)
    expect(isSafeRedirectPath('/')).toBe(false)
    expect(isSafeRedirectPath('https://example.com')).toBe(false)
    expect(isSafeRedirectPath('//example.com')).toBe(false)
    expect(isSafeRedirectPath('/login')).toBe(false)
  })
})
