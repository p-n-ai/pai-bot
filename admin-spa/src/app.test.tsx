/**
 * @vitest-environment jsdom
 */
import '@testing-library/jest-dom/vitest'

import { cleanup, fireEvent, render, screen } from '@testing-library/react'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'

import { AdminApp } from './app'
import type { AuthContextValue } from './auth-provider'
import type * as AuthProviderModule from './auth-provider'

const authContext: AuthContextValue = {
  auth: {
    status: 'pending',
    session: null,
    error: null,
  },
  setAnonymousSession: vi.fn(),
  setAuthenticatedSession: vi.fn(),
}

vi.mock('./auth-provider', async (importOriginal) => {
  const actual = await importOriginal<typeof AuthProviderModule>()

  return {
    ...actual,
    useAuth: () => authContext,
  }
})

describe('AdminApp', () => {
  beforeEach(() => {
    authContext.auth = {
      status: 'pending',
      session: null,
      error: null,
    }
  })

  afterEach(() => {
    cleanup()
  })

  it('renders a quiet admin shell skeleton while the session is loading', () => {
    render(<AdminApp />)

    expect(
      screen.getByRole('status', { name: 'Loading P&AI Bot' }),
    ).toBeInTheDocument()
    expect(
      screen.queryByText('Checking admin session...'),
    ).not.toBeInTheDocument()
  })

  it('allows the loading shell to be closed', () => {
    render(<AdminApp />)

    fireEvent.click(screen.getByRole('button', { name: 'Continue to sign in' }))

    expect(authContext.setAnonymousSession).toHaveBeenCalled()
  })

  it('gives safe recovery guidance when session verification fails', () => {
    authContext.auth = {
      status: 'error',
      session: null,
      error: new Error('private service detail'),
    }

    render(<AdminApp />)

    expect(
      screen.getByRole('heading', { name: 'Unable to verify your session' }),
    ).toBeInTheDocument()
    expect(
      screen.getByText('Check your connection and reload the page.'),
    ).toBeInTheDocument()
    expect(screen.queryByText('private service detail')).not.toBeInTheDocument()
  })
})
