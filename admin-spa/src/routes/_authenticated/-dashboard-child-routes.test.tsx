/**
 * @vitest-environment jsdom
 */
import '@testing-library/jest-dom/vitest'

import {
  RouterProvider,
  createMemoryHistory,
  createRouter,
} from '@tanstack/react-router'
import { cleanup, render, screen } from '@testing-library/react'
import { afterEach, describe, expect, it, vi } from 'vitest'

import type { AuthContextValue } from '@/auth-provider'
import type * as AuthProviderModule from '@/auth-provider'
import type * as AdminAPIModule from '@/lib/admin-api'
import { routeTree } from '@/routeTree.gen'

const authContext: AuthContextValue = {
  auth: {
    status: 'authenticated',
    session: {
      expires_at: '2099-01-01T00:00:00Z',
      user: {
        role: 'admin',
        tenant_id: 'tenant_1',
        tenant_name: 'Test School',
        user_id: 'admin_1',
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

vi.mock('@/components/shared/admin-sidebar', () => ({
  AdminSidebar: () => <nav>Admin navigation</nav>,
}))

vi.mock('@/hooks/use-mobile', () => ({
  useIsMobile: () => false,
}))

vi.mock('@/lib/admin-api', async (importOriginal) => {
  const actual = await importOriginal<typeof AdminAPIModule>()

  return {
    ...actual,
    getAIUsage: vi
      .fn()
      .mockRejectedValue(new Error('Not needed by route test')),
    listGroups: vi.fn().mockResolvedValue([]),
  }
})

afterEach(() => {
  cleanup()
})

describe('dashboard child routes', () => {
  it.each([
    ['/dashboard/classes', 'Class management'],
    ['/dashboard/metrics', 'AI usage'],
    ['/dashboard/ai-usage', 'AI usage'],
  ])('mounts the child page for %s', async (path, heading) => {
    window.scrollTo = vi.fn()
    const router = createRouter({
      context: authContext,
      history: createMemoryHistory({ initialEntries: [path] }),
      routeTree,
    })

    render(<RouterProvider context={authContext} router={router} />)

    expect(
      await screen.findByRole('heading', { name: heading }),
    ).toBeInTheDocument()
    expect(
      screen.queryByRole('heading', { name: 'Dashboard' }),
    ).not.toBeInTheDocument()
  })
})
