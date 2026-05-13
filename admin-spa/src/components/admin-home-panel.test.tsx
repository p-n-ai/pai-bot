/**
 * @vitest-environment jsdom
 */
import '@testing-library/jest-dom/vitest'

import { cleanup, render, screen, within } from '@testing-library/react'
import { afterEach, describe, expect, it } from 'vitest'
import { AdminHomePanel } from './admin-home-panel'

import type { AuthState } from '@/auth-provider'

afterEach(() => {
  cleanup()
})

describe('AdminHomePanel', () => {
  it('turns the root page into a login-first admin entry', () => {
    render(<AdminHomePanel auth={anonymousAuth} />)

    const signIn = screen.getByLabelText('Admin sign in')
    expect(
      within(signIn).getByRole('heading', {
        level: 1,
      }),
    ).toBeInTheDocument()
    expect(
      within(signIn).getByRole('textbox', { name: /email/i }),
    ).toBeInTheDocument()
    expect(within(signIn).getByLabelText(/password/i)).toBeInTheDocument()
    expect(
      within(signIn).getByRole('button', { name: 'Sign in' }),
    ).toBeInTheDocument()
  })

  it('keeps the root page focused on sign in without secondary product previews', () => {
    render(<AdminHomePanel auth={anonymousAuth} />)

    expect(screen.queryByLabelText('Product proof')).not.toBeInTheDocument()
    expect(
      screen.queryByLabelText('Teacher-ready preview'),
    ).not.toBeInTheDocument()
  })

  it('does not render demo credentials or sign-in helper copy', () => {
    render(<AdminHomePanel auth={anonymousAuth} />)

    const signIn = screen.getByLabelText('Admin sign in')
    expect(
      within(signIn).queryByText('platform-admin@example.com'),
    ).not.toBeInTheDocument()
    expect(within(signIn).queryByText('demo-password')).not.toBeInTheDocument()
    expect(
      within(signIn).queryByText(/Use an invited admin account/u),
    ).not.toBeInTheDocument()
    expect(
      within(signIn).queryByText(/Class data is loaded/u),
    ).not.toBeInTheDocument()
  })

  it('keeps signed-out navigation free of duplicate sign in links', () => {
    render(<AdminHomePanel auth={anonymousAuth} nextPath='/dashboard' />)

    expect(
      screen.queryByRole('link', { name: /sign in/i }),
    ).not.toBeInTheDocument()
  })

  it('renders no signed-in interstitial because the route redirects instead', () => {
    render(<AdminHomePanel auth={teacherAuth} nextPath='/dashboard/classes' />)

    expect(screen.queryByLabelText('Signed-in account')).not.toBeInTheDocument()
    expect(
      screen.queryByRole('link', { name: /open workspace/i }),
    ).not.toBeInTheDocument()
  })
})

const anonymousAuth: AuthState = {
  error: null,
  session: null,
  status: 'anonymous',
}

const teacherAuth: AuthState = {
  error: null,
  session: {
    expires_at: '2026-05-09T00:00:00.000Z',
    user: {
      role: 'teacher',
      user_id: 'teacher-1',
    },
  },
  status: 'authenticated',
}
