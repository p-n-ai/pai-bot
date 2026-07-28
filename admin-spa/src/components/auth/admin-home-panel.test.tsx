/**
 * @vitest-environment jsdom
 */
import '@testing-library/jest-dom/vitest'

import { cleanup, render, screen } from '@testing-library/react'
import { afterEach, expect, it, vi } from 'vitest'
import { AdminHomePanel } from './admin-home-panel'

import type { AuthState } from '@/auth-provider'

vi.mock('@/lib/auth-client', () => ({
  buildGoogleLoginURL: vi.fn(),
  loginWithPassword: vi.fn(),
  readAuthCapabilities: vi.fn().mockResolvedValue({ google_login: false }),
}))

afterEach(() => {
  cleanup()
})

it('keeps signed-out navigation free of duplicate sign in links', () => {
  render(<AdminHomePanel auth={anonymousAuth} nextPath='/dashboard' />)

  expect(
    screen.getByRole('heading', { level: 1, name: 'Welcome back.' }),
  ).toBeInTheDocument()
  expect(
    screen.getByText(
      'Sign in to review progress and support the right students.',
    ),
  ).toBeInTheDocument()
  expect(screen.getAllByRole('heading', { level: 1 })).toHaveLength(1)
  expect(screen.getByText('See who needs help next.')).toBeInTheDocument()
  expect(screen.getByText('Guide the next lesson')).toBeInTheDocument()
  expect(screen.getByText('Run the whole workspace')).toBeInTheDocument()
  expect(screen.getByText('Keep access accountable')).toBeInTheDocument()
  expect(
    screen.queryByRole('link', { name: /sign in/i }),
  ).not.toBeInTheDocument()
  expect(screen.queryByRole('navigation')).not.toBeInTheDocument()
  expect(screen.getByRole('link', { name: 'P&AI Bot' })).toHaveAttribute(
    'href',
    '/',
  )
})

it('renders no signed-in interstitial because the route redirects instead', () => {
  render(<AdminHomePanel auth={teacherAuth} nextPath='/dashboard/classes' />)

  expect(screen.queryByLabelText('Signed-in account')).not.toBeInTheDocument()
  expect(
    screen.queryByRole('link', { name: /open workspace/i }),
  ).not.toBeInTheDocument()
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
