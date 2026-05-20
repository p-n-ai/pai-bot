/**
 * @vitest-environment jsdom
 */
import '@testing-library/jest-dom/vitest'

import { cleanup, render, screen } from '@testing-library/react'
import { afterEach, expect, it } from 'vitest'
import { AdminHomePanel } from './admin-home-panel'

import type { AuthState } from '@/auth-provider'

afterEach(() => {
  cleanup()
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
