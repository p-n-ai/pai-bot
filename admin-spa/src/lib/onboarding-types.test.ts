import { describe, expect, it } from 'vitest'

import { isOnboardingView, isSubmitOnboardingResult } from './onboarding-types'

describe('onboarding type guards', () => {
  it('accepts the onboarding view contract', () => {
    expect(
      isOnboardingView({
        tenant_id: 'tenant_1',
        tenant_name: 'Sekolah Harapan',
        onboarding: {
          school_name: 'Sekolah Harapan',
          curriculum: {
            syllabus_id: 'kssm-algebra',
            label: 'KSSM Algebra',
          },
          first_class: {
            name: 'Form 1 Mathematics',
            slug: 'form-1-mathematics',
          },
          bot_setup: {
            preset: 'guided-practice',
          },
          join_link: 'https://app.test/join/form-1-mathematics',
          save_status: 'saved',
          configured_at: '2026-05-08T00:00:00Z',
        },
      }),
    ).toBe(true)
  })

  it('rejects malformed onboarding results', () => {
    expect(
      isSubmitOnboardingResult({
        class_id: 'class_1',
        school_name: 'Sekolah Harapan',
        class_name: 'Form 1 Mathematics',
      }),
    ).toBe(false)
  })
})
