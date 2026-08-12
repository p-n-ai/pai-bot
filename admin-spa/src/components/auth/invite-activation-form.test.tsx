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

    expect(
      screen.getByRole('heading', {
        name: 'This invite link is incomplete',
      }),
    ).toBeInTheDocument()
    expect(
      screen.getByText(/open the full invite link from your email/i),
    ).toBeInTheDocument()
    expect(screen.queryByLabelText('Full name')).not.toBeInTheDocument()
    expect(screen.queryByLabelText('Password')).not.toBeInTheDocument()
    expect(
      screen.queryByRole('button', { name: 'Accept invite' }),
    ).not.toBeInTheDocument()
    expect(
      screen.getByRole('link', { name: 'Back to sign in' }),
    ).toHaveAttribute('href', '/')
  })

  it('renders focused activation guidance and trust cues', () => {
    render(
      <InviteActivationForm onAuthenticated={vi.fn()} token='invite-token' />,
    )

    expect(screen.getByText('Invite activation')).toBeInTheDocument()
    expect(
      screen.getByRole('heading', { name: 'Accept your invite' }),
    ).toBeInTheDocument()
    expect(
      screen.getByRole('heading', { name: 'Set up your account' }),
    ).toBeInTheDocument()
    expect(screen.getByText('One-time link')).toBeInTheDocument()
    expect(
      screen.getByText('This link works once for one school account.'),
    ).toBeInTheDocument()
    expect(screen.getByText('Automatic sign-in')).toBeInTheDocument()
    expect(
      screen.getByText('After setup, you’ll continue to the right workspace.'),
    ).toBeInTheDocument()
    expect(screen.getByLabelText('Password')).toHaveAccessibleDescription(
      '12 characters minimum.',
    )
  })

  it('focuses and describes the first invalid field', () => {
    render(
      <InviteActivationForm onAuthenticated={vi.fn()} token='invite-token' />,
    )

    fireEvent.click(screen.getByRole('button', { name: 'Accept invite' }))

    const nameInput = screen.getByLabelText('Full name')
    expect(nameInput).toHaveFocus()
    expect(nameInput).toHaveAttribute('aria-invalid', 'true')
    expect(nameInput).toHaveAccessibleDescription('Enter your full name.')
    expect(acceptInvite).not.toHaveBeenCalled()
  })

  it('rejects and describes a password shorter than 12 characters', () => {
    render(
      <InviteActivationForm onAuthenticated={vi.fn()} token='invite-token' />,
    )

    fireEvent.change(screen.getByLabelText('Full name'), {
      target: { value: 'Parent One' },
    })
    fireEvent.change(screen.getByLabelText('Password'), {
      target: { value: 'short-pass' },
    })
    fireEvent.click(screen.getByRole('button', { name: 'Accept invite' }))

    const passwordInput = screen.getByLabelText('Password')
    expect(passwordInput).toHaveFocus()
    expect(passwordInput).toHaveAttribute('aria-invalid', 'true')
    expect(passwordInput).toHaveAccessibleDescription(
      '12 characters minimum. Use at least 12 characters for your password.',
    )
    expect(acceptInvite).not.toHaveBeenCalled()
  })

  it('clears a field error after the user edits that field', () => {
    render(
      <InviteActivationForm onAuthenticated={vi.fn()} token='invite-token' />,
    )

    fireEvent.click(screen.getByRole('button', { name: 'Accept invite' }))
    const nameInput = screen.getByLabelText('Full name')
    expect(nameInput).toHaveAttribute('aria-invalid', 'true')

    fireEvent.change(nameInput, { target: { value: 'Parent One' } })

    expect(nameInput).not.toHaveAttribute('aria-invalid')
    expect(screen.queryByText('Enter your full name.')).not.toBeInTheDocument()
  })

  it('keeps the submit label and disables the form while activation is pending', async () => {
    let resolveInvite: ((session: AuthSession) => void) | undefined
    acceptInvite.mockReturnValue(
      new Promise<AuthSession>((resolve) => {
        resolveInvite = resolve
      }),
    )
    const onAuthenticated = vi.fn()

    render(
      <InviteActivationForm
        onAuthenticated={onAuthenticated}
        token='invite-token'
      />,
    )

    const nameInput = screen.getByLabelText('Full name')
    const passwordInput = screen.getByLabelText('Password')
    const button = screen.getByRole('button', { name: 'Accept invite' })

    fireEvent.change(nameInput, { target: { value: 'Parent One' } })
    fireEvent.change(passwordInput, { target: { value: 'strong-pass-1' } })
    fireEvent.click(button)

    await waitFor(() => {
      expect(button.closest('form')).toHaveAttribute('aria-busy', 'true')
    })
    expect(nameInput).toBeDisabled()
    expect(passwordInput).toBeDisabled()
    expect(button).toBeDisabled()
    expect(button).toHaveAccessibleName('Accept invite')

    resolveInvite?.(parentSession)
    await waitFor(() => {
      expect(onAuthenticated).toHaveBeenCalledWith(parentSession)
    })
  })

  it('turns expired invite errors into a safe recovery instruction', async () => {
    acceptInvite.mockRejectedValue(new Error('invite expired'))

    render(
      <InviteActivationForm onAuthenticated={vi.fn()} token='invite-token' />,
    )

    fireEvent.change(screen.getByLabelText('Full name'), {
      target: { value: 'Parent One' },
    })
    fireEvent.change(screen.getByLabelText('Password'), {
      target: { value: 'strong-pass-1' },
    })
    fireEvent.click(screen.getByRole('button', { name: 'Accept invite' }))

    expect(await screen.findByRole('alert')).toHaveTextContent(
      'This invite link is invalid or has expired. Ask your school administrator for a new invite.',
    )
  })

  it('does not expose unexpected backend error details', async () => {
    acceptInvite.mockRejectedValue(
      new Error('postgres: relation invitations does not exist'),
    )

    render(
      <InviteActivationForm onAuthenticated={vi.fn()} token='invite-token' />,
    )

    fireEvent.change(screen.getByLabelText('Full name'), {
      target: { value: 'Parent One' },
    })
    fireEvent.change(screen.getByLabelText('Password'), {
      target: { value: 'strong-pass-1' },
    })
    fireEvent.click(screen.getByRole('button', { name: 'Accept invite' }))

    const alert = await screen.findByRole('alert')
    expect(alert).toHaveTextContent(
      'Unable to accept this invite. Check your connection and try again.',
    )
    expect(alert).not.toHaveTextContent('postgres')
  })
})
