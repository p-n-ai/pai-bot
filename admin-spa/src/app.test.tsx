/**
 * @vitest-environment jsdom
 */
import '@testing-library/jest-dom/vitest'

import { cleanup, fireEvent, render, screen } from '@testing-library/react'
import { afterEach, describe, expect, it, vi } from 'vitest'

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
  afterEach(() => {
    cleanup()
  })

  it('renders a quiet admin shell skeleton while the session is loading', () => {
    render(<AdminApp />)

    expect(
      screen.getByRole('status', { name: 'Preparing admin workspace' }),
    ).toBeInTheDocument()
    expect(
      screen.queryByText('Checking admin session...'),
    ).not.toBeInTheDocument()
  })

  it('allows the loading shell to be closed', () => {
    render(<AdminApp />)

    fireEvent.click(
      screen.getByRole('button', { name: 'Close loading screen' }),
    )

    expect(authContext.setAnonymousSession).toHaveBeenCalled()
  })
})
