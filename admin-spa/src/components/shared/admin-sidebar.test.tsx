/**
 * @vitest-environment jsdom
 */
import '@testing-library/jest-dom/vitest'

import { cleanup, render, screen } from '@testing-library/react'
import { afterEach, describe, expect, it, vi } from 'vitest'
import { AdminSidebar } from './admin-sidebar'
import type { MouseEventHandler, ReactNode } from 'react'
import type * as TanStackRouter from '@tanstack/react-router'
import type { AuthContextValue } from '@/auth-provider'
import type * as AuthProviderModule from '@/auth-provider'
import { SidebarProvider } from '@/components/ui/sidebar'

const authContext: AuthContextValue = {
  auth: {
    status: 'authenticated',
    session: {
      expires_at: '2099-01-01T00:00:00Z',
      user: {
        role: 'teacher',
        tenant_id: 'school_1',
        user_id: 'teacher_1',
      },
    },
    error: null,
  },
  setAnonymousSession: vi.fn(),
  setAuthenticatedSession: vi.fn(),
}

vi.mock('@/auth-provider', async (importOriginal) => {
  const actual = await importOriginal<typeof AuthProviderModule>()

  return {
    ...actual,
    useAuth: () => authContext,
  }
})

vi.mock('@/hooks/use-mobile', () => ({
  useIsMobile: () => false,
}))

vi.mock('@tanstack/react-router', async (importOriginal) => {
  const actual = await importOriginal<typeof TanStackRouter>()

  return {
    ...actual,
    Link: ({
      'aria-current': ariaCurrent,
      children,
      className,
      onClick,
      to,
    }: {
      'aria-current'?: 'page'
      children: ReactNode
      className?: string
      onClick?: MouseEventHandler<HTMLAnchorElement>
      to: string
    }) => (
      <a
        aria-current={ariaCurrent}
        className={className}
        href={to}
        onClick={onClick}
      >
        {children}
      </a>
    ),
    useRouterState: ({
      select,
    }: {
      select: (state: { location: { pathname: string } }) => unknown
    }) => select({ location: { pathname: '/dashboard/classes' } }),
  }
})

afterEach(() => {
  cleanup()
})

describe('admin sidebar', () => {
  it('announces the current page without linking to retired tools', () => {
    render(
      <SidebarProvider>
        <AdminSidebar />
      </SidebarProvider>,
    )

    expect(screen.getByRole('link', { name: 'My classes' })).toHaveAttribute(
      'aria-current',
      'page',
    )
    expect(screen.getByRole('link', { name: 'Today' })).not.toHaveAttribute(
      'aria-current',
    )
    expect(
      screen.queryByRole('link', { name: 'Content search' }),
    ).not.toBeInTheDocument()
  })
})
