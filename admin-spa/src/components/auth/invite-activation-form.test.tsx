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

import { InviteActivationForm } from './invite-activation-form'
import type { AuthSession } from '@/lib/auth-types'
import type * as AuthClient from '@/lib/auth-client'

const acceptInvite = vi.hoisted(() => vi.fn())

vi.mock('@/lib/auth-client', async (importOriginal) => {
  const actual = await importOriginal<typeof AuthClient>()

  return {
    ...actual,
    acceptInvite,
  }
})

const parentSession: AuthSession = {
  expires_at: '2026-05-08T00:00:00Z',
  user: {
    user_id: 'parent_1',
    role: 'parent',
  },
}

describe('InviteActivationForm', () => {
  afterEach(() => {
    cleanup()
  })

  beforeEach(() => {
    acceptInvite.mockReset()
  })

  it('submits trimmed name, password, and token to invite activation', async () => {
    acceptInvite.mockResolvedValue(parentSession)
    const onAuthenticated = vi.fn()

    render(
      <InviteActivationForm
        onAuthenticated={onAuthenticated}
        token='invite-token'
      />,
    )

    fireEvent.change(screen.getByLabelText('Full name'), {
      target: { value: ' Parent One ' },
    })
    fireEvent.change(screen.getByLabelText('Password'), {
      target: { value: 'strong-pass-1' },
    })
    fireEvent.click(screen.getByRole('button', { name: 'Accept invite' }))

    await waitFor(() => {
      expect(acceptInvite).toHaveBeenCalledWith({
        token: 'invite-token',
        name: 'Parent One',
        password: 'strong-pass-1',
      })
    })
    expect(onAuthenticated).toHaveBeenCalledWith(parentSession)
  })

  it('disables activation and explains when the token is missing', () => {
    render(<InviteActivationForm onAuthenticated={vi.fn()} token='' />)

    expect(screen.getByRole('alert')).toHaveTextContent('Invite token missing.')
    expect(screen.getByRole('button', { name: 'Accept invite' })).toBeDisabled()
  })

  it('renders the source-admin activation trust cues and password guidance', () => {
    render(
      <InviteActivationForm onAuthenticated={vi.fn()} token='invite-token' />,
    )

    const shell = screen.getByLabelText('Invite activation')
    const setupCard = screen.getByLabelText('Set up invite access')

    expect(shell).toBeInTheDocument()
    expect(setupCard).toBeInTheDocument()
    expect(screen.getByText('Invite Activation')).toBeInTheDocument()
    expect(
      screen.getByRole('heading', { name: 'Accept your invite' }),
    ).toBeInTheDocument()
    expect(
      screen.getByRole('heading', { name: 'Set up access' }),
    ).toBeInTheDocument()
    expect(screen.getByText('One-time activation')).toBeInTheDocument()
    expect(
      screen.getByText('This link activates one workspace account.'),
    ).toBeInTheDocument()
    expect(screen.getByText('Immediate sign-in')).toBeInTheDocument()
    expect(
      screen.getByText('Go straight to the right admin view.'),
    ).toBeInTheDocument()
    expect(screen.getByLabelText('Password')).toHaveAccessibleDescription(
      'Use a strong password for future sign-ins.',
    )
  })
})
