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

const loginWithPassword = vi.hoisted(() => vi.fn())
const readAuthCapabilities = vi.hoisted(() => vi.fn())
const buildGoogleLoginURL = vi.hoisted(() =>
  vi.fn(() => '/api/auth/google/start?next=%2Fdashboard'),
)

vi.mock('@/lib/auth-client', async (importOriginal) => {
  const actual = await importOriginal<typeof AuthClient>()

  return {
    ...actual,
    buildGoogleLoginURL,
    loginWithPassword,
    readAuthCapabilities,
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
    readAuthCapabilities.mockReset()
    readAuthCapabilities.mockResolvedValue({ google_login: false })
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

  it('shows the source-admin email divider when the server enables Google sign-in', async () => {
    readAuthCapabilities.mockResolvedValue({ google_login: true })

    render(<LoginForm onAuthenticated={vi.fn()} />)

    expect(
      await screen.findByRole('button', { name: 'Continue with Google' }),
    ).toBeInTheDocument()
    expect(screen.getByText('G')).toHaveAttribute('aria-hidden', 'true')
    expect(screen.getByText('or use email')).toBeInTheDocument()
  })

  it('keeps password sign-in available when capabilities cannot be loaded', async () => {
    readAuthCapabilities.mockRejectedValue(new Error('network unavailable'))

    render(<LoginForm onAuthenticated={vi.fn()} />)

    await waitFor(() => {
      expect(readAuthCapabilities).toHaveBeenCalledOnce()
    })
    expect(
      screen.queryByRole('button', { name: 'Continue with Google' }),
    ).not.toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'Sign in' })).toBeEnabled()
  })

  it('shows Google redirect progress and disables email submit while redirecting', async () => {
    readAuthCapabilities.mockResolvedValue({ google_login: true })
    const assign = vi.fn()
    vi.stubGlobal('location', { assign })

    render(<LoginForm onAuthenticated={vi.fn()} nextPath='/dashboard' />)

    fireEvent.click(
      await screen.findByRole('button', { name: 'Continue with Google' }),
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
