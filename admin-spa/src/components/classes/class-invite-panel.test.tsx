/**
 * @vitest-environment jsdom
 */
import '@testing-library/jest-dom/vitest'

import { fireEvent, render, screen, waitFor } from '@testing-library/react'
import { describe, expect, it, vi } from 'vitest'
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
  it('issues an invite from the class management surface', async () => {
    issueInviteMock.mockResolvedValue({
      email: 'teacher@example.com',
      invite_token: 'invite-token',
      role: 'teacher',
    })

    render(<ClassInvitePanel />)

    fireEvent.change(screen.getByLabelText('Email'), {
      target: { value: 'teacher@example.com' },
    })
    fireEvent.click(screen.getByRole('button', { name: /send invite/i }))

    await waitFor(() =>
      expect(issueInviteMock).toHaveBeenCalledWith({
        email: 'teacher@example.com',
        role: 'teacher',
      }),
    )
    expect(screen.getByLabelText('Activation link')).toHaveValue(
      'http://localhost:3000/activate?token=invite-token',
    )
  })
})
