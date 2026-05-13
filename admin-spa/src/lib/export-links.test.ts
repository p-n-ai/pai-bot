import { describe, expect, it } from 'vitest'

import { adminExportLinks } from './export-links'

describe('adminExportLinks', () => {
  it('keeps exports on same-origin backend routes', () => {
    expect(adminExportLinks.map((link) => link.href)).toEqual([
      '/api/admin/export/students',
      '/api/admin/export/conversations',
      '/api/admin/export/progress',
    ])
  })
})
