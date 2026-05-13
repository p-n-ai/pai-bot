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
import { afterEach, describe, expect, it, vi } from 'vitest'

import { OnboardingResultActions } from './onboarding-result-actions'
import { issueInvite } from '@/lib/admin-api'

vi.mock('@/lib/admin-api', () => ({
  issueInvite: vi.fn(),
}))

const mockedIssueInvite = vi.mocked(issueInvite)

afterEach(() => {
  cleanup()
  vi.unstubAllGlobals()
  vi.clearAllMocks()
})

describe('OnboardingResultActions', () => {
  it('copies, opens, and allows editing the saved setup', async () => {
    const clipboard = vi.fn().mockResolvedValue(undefined)
    const open = vi.fn()
    const handleEdit = vi.fn()

    vi.stubGlobal('navigator', { clipboard: { writeText: clipboard } })
    vi.stubGlobal('open', open)

    render(
      <OnboardingResultActions onEditSetup={handleEdit} result={savedResult} />,
    )

    expect(
      screen.getByRole('heading', { name: 'Welcome to your classroom.' }),
    ).toBeInTheDocument()
    expect(
      screen.getByText('Your first class is ready to share.'),
    ).toBeInTheDocument()
    expect(screen.getByText('Form 1 Algebra A')).toBeInTheDocument()
    expect(screen.getByText('Sekolah Satu')).toBeInTheDocument()
    expect(screen.getByText('Students')).toBeInTheDocument()

    fireEvent.click(screen.getByRole('button', { name: 'Copy' }))
    await waitFor(() =>
      expect(clipboard).toHaveBeenCalledWith('https://pai.test/join/ALG-F1A'),
    )
    expect(screen.getByText('Copied')).toBeInTheDocument()

    fireEvent.click(screen.getByRole('button', { name: 'Open' }))
    expect(open).toHaveBeenCalledWith(
      'https://pai.test/join/ALG-F1A',
      '_blank',
      'noopener,noreferrer',
    )

    fireEvent.click(screen.getByRole('button', { name: 'Edit setup' }))
    expect(handleEdit).toHaveBeenCalledOnce()
  })

  it('dedupes teacher emails and shows invite outcomes', async () => {
    const clipboard = vi.fn().mockResolvedValue(undefined)
    const handleEdit = vi.fn()

    vi.stubGlobal('navigator', { clipboard: { writeText: clipboard } })
    vi.stubGlobal('location', { origin: 'https://pai.test' })
    mockedIssueInvite
      .mockResolvedValueOnce({
        delivery_status: 'sent',
        email: 'teacher@example.com',
        invite_token: 'teacher-token',
        role: 'teacher',
      })
      .mockResolvedValueOnce({
        activation_url: 'https://pai.test/activate?token=second-token',
        delivery_error: 'SMTP provider throttled',
        delivery_status: 'failed',
        email: 'second@example.com',
        invite_token: 'second-token',
        role: 'teacher',
      })

    render(
      <OnboardingResultActions onEditSetup={handleEdit} result={savedResult} />,
    )

    fireEvent.change(screen.getByLabelText('Teacher emails'), {
      target: {
        value: 'Teacher@Example.com; teacher@example.com\nsecond@example.com',
      },
    })
    fireEvent.click(screen.getByRole('button', { name: 'Send invites' }))

    await waitFor(() => expect(mockedIssueInvite).toHaveBeenCalledTimes(2))
    expect(mockedIssueInvite).toHaveBeenNthCalledWith(1, {
      email: 'teacher@example.com',
      role: 'teacher',
    })
    expect(mockedIssueInvite).toHaveBeenNthCalledWith(2, {
      email: 'second@example.com',
      role: 'teacher',
    })
    expect(
      screen.getByText('1 sent, 1 needing follow-up, 2 processed.'),
    ).toBeInTheDocument()
    expect(screen.getByText('SMTP provider throttled')).toBeInTheDocument()

    fireEvent.click(
      screen.getAllByRole('button', { name: 'Copy invite link' })[0],
    )
    await waitFor(() =>
      expect(clipboard).toHaveBeenCalledWith(
        'https://pai.test/activate?token=teacher-token',
      ),
    )
    expect(
      await screen.findByText('Copied link for teacher@example.com'),
    ).toBeInTheDocument()
  })

  it('shows the source-admin invite error title when no teacher emails are provided', () => {
    render(
      <OnboardingResultActions onEditSetup={vi.fn()} result={savedResult} />,
    )

    fireEvent.click(screen.getByRole('button', { name: 'Send invites' }))

    expect(screen.getByRole('alert')).toHaveTextContent(
      'We could not send the invite',
    )
    expect(screen.getByRole('alert')).toHaveTextContent(
      'Add at least one teacher email.',
    )
  })
})

const savedResult = {
  class_id: 'class-1',
  class_name: 'Form 1 Algebra A',
  join_link: 'https://pai.test/join/ALG-F1A',
  save_status: 'saved',
  school_name: 'Sekolah Satu',
} as const
