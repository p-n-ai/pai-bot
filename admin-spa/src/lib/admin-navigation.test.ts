import { describe, expect, it } from 'vitest'

import {
  getVisibleNavigationGroups,
  isNavigationItemActive,
} from './admin-navigation'
import type { AuthUser } from '@/lib/auth-types'

const teacher: AuthUser = {
  role: 'teacher',
  tenant_id: 'school_1',
  user_id: 'teacher_1',
}

describe('admin sidebar navigation', () => {
  it('keeps daily teaching separate from technical tools', () => {
    const groups = getVisibleNavigationGroups(teacher)

    expect(groups.map(({ label }) => label)).toEqual([
      'Teaching',
      'Technical tools',
    ])
    expect(groups[0]?.items.map(({ label }) => label)).toEqual([
      'Today',
      'My classes',
    ])
    expect(groups[1]?.items.map(({ label }) => label)).toEqual(['AI activity'])
    expect(
      groups.flatMap(({ items }) => items).map(({ href }) => href),
    ).not.toContain('/dashboard/retrieval-lab')
    expect(
      groups.flatMap(({ items }) => items).map(({ href }) => href),
    ).not.toContain('/dashboard/metrics')
  })

  it('shows school administration only to roles with access', () => {
    const adminGroups = getVisibleNavigationGroups({
      ...teacher,
      role: 'admin',
    })

    expect(
      adminGroups
        .find(({ label }) => label === 'School administration')
        ?.items.map(({ label }) => label),
    ).toEqual(['Staff access', 'AI budget', 'Download records'])

    expect(getVisibleNavigationGroups({ ...teacher, role: 'parent' })).toEqual(
      [],
    )
  })

  it('keeps AI settings capability-aware', () => {
    const withoutCapability = getVisibleNavigationGroups({
      ...teacher,
      role: 'platform_admin',
    })
    const withCapability = getVisibleNavigationGroups({
      ...teacher,
      can_manage_ai_settings: true,
      role: 'platform_admin',
    })

    expect(
      withoutCapability.flatMap(({ items }) => items).map(({ label }) => label),
    ).not.toContain('AI settings')
    expect(
      withoutCapability.flatMap(({ items }) => items).map(({ label }) => label),
    ).not.toContain('Build AI')
    expect(
      withCapability.flatMap(({ items }) => items).map(({ label }) => label),
    ).toContain('AI settings')
    expect(
      withCapability.flatMap(({ items }) => items).map(({ label }) => label),
    ).toContain('Build AI')
  })

  it('marks only the active destination and its descendants as current', () => {
    expect(
      isNavigationItemActive(
        '/dashboard/classes',
        '/dashboard/classes/class_1',
      ),
    ).toBe(true)
    expect(isNavigationItemActive('/dashboard', '/dashboard/classes')).toBe(
      false,
    )
  })
})
