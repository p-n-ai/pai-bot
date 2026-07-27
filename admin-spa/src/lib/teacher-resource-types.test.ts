import { describe, expect, it } from 'vitest'

import {
  isAllowedTeacherResourceFile,
  isTeacherResource,
} from './teacher-resource-types'

export const teacherResourceFixture = {
  id: 'resource-1',
  filename: 'algebra.pdf',
  title: 'Algebra revision',
  source_type: 'pdf',
  media_type: 'application/pdf',
  byte_size: 2048,
  chunk_count: 12,
  active: true,
  class_ids: ['class-1'],
  created_at: '2026-07-27T12:00:00Z',
  updated_at: '2026-07-27T12:00:00Z',
  uploader_name: 'Ms Lim',
} as const

describe('teacher resource types', () => {
  it('validates a teacher resource response', () => {
    expect(isTeacherResource(teacherResourceFixture)).toBe(true)
    expect(
      isTeacherResource({ ...teacherResourceFixture, class_ids: [42] }),
    ).toBe(false)
    expect(
      isTeacherResource({ ...teacherResourceFixture, source_type: 'txt' }),
    ).toBe(false)
  })

  it('accepts only PDF, DOCX, and PPTX filenames', () => {
    expect(isAllowedTeacherResourceFile(new File(['pdf'], 'LESSON.PDF'))).toBe(
      true,
    )
    expect(isAllowedTeacherResourceFile(new File(['doc'], 'lesson.docx'))).toBe(
      true,
    )
    expect(isAllowedTeacherResourceFile(new File(['ppt'], 'lesson.pptx'))).toBe(
      true,
    )
    expect(isAllowedTeacherResourceFile(new File(['text'], 'lesson.txt'))).toBe(
      false,
    )
  })
})
