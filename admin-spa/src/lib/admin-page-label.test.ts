import { describe, expect, it } from 'vitest'

import { getAdminPageLabel } from './admin-page-label'

describe('admin topbar context', () => {
  it.each([
    ['/dashboard', 'Today'],
    ['/dashboard/classes', 'Classes'],
    ['/dashboard/classes/class_1', 'Classes'],
    ['/dashboard/metrics', 'Learning progress'],
    ['/settings/users', 'Staff access'],
    ['/settings/embed', 'Website chat'],
    ['/students/student_1', 'Learner profile'],
    ['/parents/parent_1', 'Family overview'],
    ['/export', 'Download records'],
  ])('labels %s as %s', (pathname, label) => {
    expect(getAdminPageLabel(pathname)).toBe(label)
  })

  it('uses a calm fallback for an unknown authenticated path', () => {
    expect(getAdminPageLabel('/unknown')).toBe('School')
  })

  it('does not restore context for the retired retrieval lab', () => {
    expect(getAdminPageLabel('/dashboard/retrieval-lab')).toBe('School')
  })
})
