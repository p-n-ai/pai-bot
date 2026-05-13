import { describe, expect, it } from 'vitest'

import {
  buildInitialOnboardingInput,
  prepareOnboardingInput,
} from './onboarding-form'

describe('onboarding form helpers', () => {
  it('builds a form from the current onboarding view', () => {
    expect(
      buildInitialOnboardingInput({
        tenant_id: 'tenant_1',
        tenant_name: 'Sekolah Harapan',
        onboarding: null,
      }),
    ).toMatchObject({
      school_name: 'Sekolah Harapan',
      first_class: {
        name: '',
      },
      bot_setup: {
        preset: 'guided-practice',
      },
    })
  })

  it('normalizes class names into join-safe slugs before submit', () => {
    expect(
      prepareOnboardingInput({
        school_name: ' Sekolah Harapan ',
        curriculum: {
          syllabus_id: 'kssm-algebra',
          label: 'KSSM Algebra',
        },
        first_class: {
          name: ' Form 1 Algebra ',
          slug: '',
        },
        bot_setup: {
          preset: 'guided-practice',
        },
      }),
    ).toMatchObject({
      school_name: 'Sekolah Harapan',
      first_class: {
        name: 'Form 1 Algebra',
        slug: 'form-1-algebra',
      },
    })
  })
})
