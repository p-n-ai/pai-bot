import { describe, expect, it } from 'vitest'

import { isJoinClassView } from './join-types'

describe('isJoinClassView', () => {
  it('accepts public join class payloads', () => {
    expect(
      isJoinClassView({
        class_id: 'class_1',
        class_name: 'Form 1 Algebra',
        class_slug: 'form-1-algebra',
        curriculum_label: 'KSSM Form 1',
        school_name: 'Sekolah Harapan',
      }),
    ).toBe(true)
  })

  it('rejects malformed public join payloads', () => {
    expect(
      isJoinClassView({
        class_id: 'class_1',
        class_name: 'Form 1 Algebra',
      }),
    ).toBe(false)
  })
})
