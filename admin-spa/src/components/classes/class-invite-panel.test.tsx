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
import { ClassInvitePanel } from './class-invite-panel'
import type * as AdminApi from '@/lib/admin-api'

const issueInviteMock = vi.hoisted(() => vi.fn())

vi.mock('@/lib/admin-api', async (importOriginal) => {
  const actual = await importOriginal<typeof AdminApi>()

  return {
    ...actual,
    issueInvite: issueInviteMock,
  }
})

describe('ClassInvitePanel', () => {
  beforeEach(() => {
    issueInviteMock.mockReset()
  })

  afterEach(() => {
    cleanup()
    vi.unstubAllGlobals()
  })

  it('issues an invite from the class management surface', async () => {
    issueInviteMock.mockResolvedValue({
      email: 'teacher@example.com',
      invite_token: 'invite-token',
      role: 'teacher',
    })

    render(<ClassInvitePanel />)

    fireEvent.change(screen.getByLabelText('Email address'), {
      target: { value: 'teacher@example.com' },
    })
    fireEvent.click(screen.getByRole('button', { name: /create invite/i }))

    await waitFor(() =>
      expect(issueInviteMock).toHaveBeenCalledWith({
        email: 'teacher@example.com',
        role: 'teacher',
      }),
    )
    expect(screen.getByLabelText('Activation link')).toHaveValue(
      'http://localhost:3000/activate?token=invite-token',
    )
    expect(
      screen.getByText('Invite ready for teacher@example.com'),
    ).toBeInTheDocument()
  })

  it('gives copy-specific guidance when clipboard access fails', async () => {
    issueInviteMock.mockResolvedValue({
      email: 'teacher@example.com',
      invite_token: 'invite-token',
      role: 'teacher',
    })

    render(<ClassInvitePanel />)

    fireEvent.change(screen.getByLabelText('Email address'), {
      target: { value: 'teacher@example.com' },
    })
    fireEvent.click(screen.getByRole('button', { name: /create invite/i }))
    await screen.findByLabelText('Activation link')

    vi.stubGlobal('navigator', {
      clipboard: {
        writeText: vi.fn().mockRejectedValue(new Error('blocked')),
      },
    })
    fireEvent.click(screen.getByRole('button', { name: 'Copy link' }))

    expect(await screen.findByRole('alert')).toHaveTextContent(
      'Could not copy the activation link Try again.',
    )
    expect(screen.getByRole('alert')).not.toHaveTextContent('Check the email')
  })
})
