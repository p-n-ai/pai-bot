/**
 * @vitest-environment jsdom
 */
import '@testing-library/jest-dom/vitest'

import {
  cleanup,
  fireEvent,
  render,
  screen,
  waitFor,
} from '@testing-library/react'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'

import { OnboardingPanel } from './onboarding-panel'
import type * as AdminAPI from '@/lib/admin-api'
import type {
  OnboardingView,
  SubmitOnboardingResult,
} from '@/lib/onboarding-types'

const getOnboarding = vi.hoisted(() => vi.fn())
const submitOnboarding = vi.hoisted(() => vi.fn())

vi.mock('@/lib/admin-api', async (importOriginal) => {
  const actual = await importOriginal<typeof AdminAPI>()

  return {
    ...actual,
    getOnboarding,
    submitOnboarding,
  }
})

describe('OnboardingPanel', () => {
  beforeEach(() => {
    getOnboarding.mockResolvedValue(onboardingView)
    submitOnboarding.mockResolvedValue(onboardingResult)
  })

  afterEach(() => {
    cleanup()
    vi.clearAllMocks()
  })

  it('walks the setup wizard and submits the selected class and tutor style', async () => {
    render(<OnboardingPanel />)

    expect(await screen.findByText('Curriculum')).toBeInTheDocument()
    expect(screen.getByText('Choose the starting syllabus')).toBeInTheDocument()
    expect(
      screen.getByRole('progressbar', { name: 'Setup progress' }),
    ).toHaveAttribute('aria-valuenow', '25')
    expect(screen.getByRole('button', { name: 'Curriculum' })).toHaveAttribute(
      'aria-current',
      'step',
    )

    fireEvent.click(screen.getByRole('button', { name: 'Next' }))
    expect(screen.getByText('Name the class')).toBeInTheDocument()
    expect(
      screen.getByRole('progressbar', { name: 'Setup progress' }),
    ).toHaveAttribute('aria-valuenow', '50')
    expect(screen.getByRole('button', { name: 'First class' })).toHaveAttribute(
      'aria-current',
      'step',
    )

    fireEvent.change(screen.getByLabelText('First class name'), {
      target: { value: '' },
    })
    expect(screen.getByRole('button', { name: 'Next' })).toBeDisabled()

    fireEvent.change(screen.getByLabelText('First class name'), {
      target: { value: 'Form 1 Amanah' },
    })
    fireEvent.click(screen.getByRole('button', { name: 'Next' }))

    expect(screen.getByText('Choose how the tutor begins')).toBeInTheDocument()
    fireEvent.click(screen.getByRole('button', { name: 'Revision boost' }))
    expect(
      screen.getByRole('button', { name: 'Revision boost' }),
    ).toHaveAttribute('aria-pressed', 'true')
    fireEvent.click(screen.getByRole('button', { name: 'Next' }))

    expect(screen.getByText('Review and save')).toBeInTheDocument()
    fireEvent.change(screen.getByLabelText('School name (optional)'), {
      target: { value: 'Sekolah Amanah' },
    })
    fireEvent.click(screen.getByRole('button', { name: 'Save setup' }))

    await waitFor(() => {
      expect(submitOnboarding).toHaveBeenCalledWith({
        school_name: 'Sekolah Amanah',
        curriculum: {
          label: 'KSSM Algebra',
          syllabus_id: 'kssm-algebra',
        },
        first_class: {
          name: 'Form 1 Amanah',
          slug: 'form-1-amanah',
        },
        bot_setup: {
          preset: 'revision-boost',
        },
      })
    })
    expect(
      await screen.findByRole('heading', {
        name: 'Welcome to your classroom.',
      }),
    ).toBeInTheDocument()
    expect(screen.getByText('Form 1 Amanah')).toBeInTheDocument()
  })
})

const onboardingView: OnboardingView = {
  tenant_id: 'tenant-1',
  tenant_name: 'Sekolah Satu',
  onboarding: null,
}

const onboardingResult: SubmitOnboardingResult = {
  class_id: 'class-1',
  class_name: 'Form 1 Amanah',
  join_link: 'https://pai.test/join/form-1-amanah',
  save_status: 'saved',
  school_name: 'Sekolah Amanah',
}
