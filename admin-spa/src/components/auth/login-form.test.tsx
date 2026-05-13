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

import { LoginForm } from './login-form'
import type { AuthSession } from '@/lib/auth-types'
import type * as AuthClient from '@/lib/auth-client'
import type * as LoginSettings from '@/lib/login-settings'

const loginWithPassword = vi.hoisted(() => vi.fn())
const buildGoogleLoginURL = vi.hoisted(() =>
  vi.fn(() => '/api/auth/google/start?next=%2Fdashboard'),
)
const isGoogleLoginEnabled = vi.hoisted(() => vi.fn(() => false))

vi.mock('@/lib/auth-client', async (importOriginal) => {
  const actual = await importOriginal<typeof AuthClient>()

  return {
    ...actual,
    buildGoogleLoginURL,
    loginWithPassword,
  }
})

vi.mock('@/lib/login-settings', async (importOriginal) => {
  const actual = await importOriginal<typeof LoginSettings>()

  return {
    ...actual,
    isGoogleLoginEnabled,
  }
})

const adminSession: AuthSession = {
  expires_at: '2026-05-08T00:00:00Z',
  user: {
    user_id: 'admin_1',
    role: 'admin',
  },
}

describe('LoginForm', () => {
  beforeEach(() => {
    isGoogleLoginEnabled.mockReturnValue(false)
    loginWithPassword.mockReset()
    window.HTMLElement.prototype.scrollIntoView = vi.fn()
  })

  afterEach(() => {
    cleanup()
    vi.unstubAllGlobals()
  })

  it('submits trimmed email and password to the backend auth route', async () => {
    loginWithPassword.mockResolvedValue({
      kind: 'authenticated',
      session: adminSession,
    })
    const onAuthenticated = vi.fn()

    render(
      <LoginForm onAuthenticated={onAuthenticated} nextPath='/dashboard' />,
    )

    fireEvent.change(screen.getByLabelText('Email'), {
      target: { value: ' teacher@school.edu ' },
    })
    fireEvent.change(screen.getByLabelText('Password'), {
      target: { value: 'secret' },
    })
    fireEvent.click(screen.getByRole('button', { name: 'Sign in' }))

    await waitFor(() => {
      expect(loginWithPassword).toHaveBeenCalledWith({
        email: 'teacher@school.edu',
        password: 'secret',
      })
    })
    expect(onAuthenticated).toHaveBeenCalledWith(adminSession)
  })

  it('maps google auth error codes to an accessible alert', () => {
    render(<LoginForm authError='link_required' onAuthenticated={vi.fn()} />)

    expect(screen.getByRole('alert')).toHaveTextContent(
      'We found no Google-linked admin account yet.',
    )
  })

  it('shows the source-admin email divider when Google sign-in is enabled', () => {
    isGoogleLoginEnabled.mockReturnValue(true)

    render(<LoginForm onAuthenticated={vi.fn()} />)

    expect(
      screen.getByRole('button', { name: 'Continue with Google' }),
    ).toBeInTheDocument()
    expect(screen.getByText('G')).toHaveAttribute('aria-hidden', 'true')
    expect(screen.getByText('or use email')).toBeInTheDocument()
  })

  it('shows Google redirect progress and disables email submit while redirecting', () => {
    isGoogleLoginEnabled.mockReturnValue(true)
    const assign = vi.fn()
    vi.stubGlobal('location', { assign })

    render(<LoginForm onAuthenticated={vi.fn()} nextPath='/dashboard' />)

    fireEvent.click(
      screen.getByRole('button', { name: 'Continue with Google' }),
    )

    expect(buildGoogleLoginURL).toHaveBeenCalledWith('/dashboard')
    expect(assign).toHaveBeenCalledWith(
      '/api/auth/google/start?next=%2Fdashboard',
    )
    expect(
      screen.getByRole('button', { name: 'Redirecting to Google...' }),
    ).toBeDisabled()
    expect(screen.getByRole('button', { name: 'Sign in' })).toBeDisabled()
  })

  it('retries password login with the selected school when tenant selection is required', async () => {
    loginWithPassword
      .mockResolvedValueOnce({
        kind: 'tenant_required',
        tenant_choices: [
          {
            tenant_id: 'school-1',
            tenant_name: 'Sekolah Merdeka',
          },
        ],
      })
      .mockResolvedValueOnce({
        kind: 'authenticated',
        session: adminSession,
      })
    const onAuthenticated = vi.fn()

    render(<LoginForm onAuthenticated={onAuthenticated} />)

    fireEvent.change(screen.getByLabelText('Email'), {
      target: { value: 'teacher@school.edu' },
    })
    fireEvent.change(screen.getByLabelText('Password'), {
      target: { value: 'secret' },
    })
    fireEvent.click(screen.getByRole('button', { name: 'Sign in' }))

    expect(
      await screen.findByText('Choose a school to continue signing in.'),
    ).toBeInTheDocument()

    fireEvent.click(screen.getByRole('combobox', { name: 'School' }))
    fireEvent.click(
      await screen.findByRole('option', { name: 'Sekolah Merdeka' }),
    )
    fireEvent.click(screen.getByRole('button', { name: 'Sign in' }))

    await waitFor(() => {
      expect(loginWithPassword).toHaveBeenLastCalledWith({
        email: 'teacher@school.edu',
        password: 'secret',
        tenant_id: 'school-1',
      })
    })
    expect(onAuthenticated).toHaveBeenCalledWith(adminSession)
  })
})
