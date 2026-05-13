import { describe, expect, it, vi } from 'vitest'

import {
  acceptInvite,
  buildGoogleLoginURL,
  loginWithPassword,
  logout,
  readAuthSession,
} from './auth-client'

describe('readAuthSession', () => {
  it('reads the cookie-backed admin session', async () => {
    const session = {
      expires_at: '2026-05-08T00:00:00Z',
      user: {
        user_id: 'user_1',
        role: 'admin',
      },
    }
    const fetcher = vi.fn().mockResolvedValue({
      ok: true,
      json: () => Promise.resolve(session),
    })

    await expect(readAuthSession(fetcher)).resolves.toEqual(session)

    expect(fetcher).toHaveBeenCalledWith('/api/auth/session', {
      credentials: 'include',
    })
  })

  it('treats non-ok session responses as signed out', async () => {
    const fetcher = vi.fn().mockResolvedValue({
      ok: false,
      json: () => Promise.reject(new Error('should not parse')),
    })

    await expect(readAuthSession(fetcher)).resolves.toBeNull()
  })

  it('fails closed when a successful response violates the session contract', async () => {
    const fetcher = vi.fn().mockResolvedValue({
      ok: true,
      json: () => Promise.resolve({ user: { role: 'admin' } }),
    })

    await expect(readAuthSession(fetcher)).rejects.toMatchObject({
      name: 'AuthContractError',
      message: 'Invalid auth session response',
    })
  })
})

describe('loginWithPassword', () => {
  it('posts credentials without exposing tokens to browser storage', async () => {
    const session = {
      expires_at: '2026-05-08T00:00:00Z',
      user: {
        user_id: 'admin_1',
        role: 'admin',
      },
    }
    const fetcher = vi.fn().mockResolvedValue(
      new Response(JSON.stringify(session), {
        status: 200,
      }),
    )

    await expect(
      loginWithPassword(
        {
          email: 'teacher@school.edu',
          password: 'secret',
        },
        fetcher,
      ),
    ).resolves.toEqual({
      kind: 'authenticated',
      session,
    })

    expect(fetcher).toHaveBeenCalledWith('/api/auth/login', {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
      },
      credentials: 'include',
      cache: 'no-store',
      body: JSON.stringify({
        email: 'teacher@school.edu',
        password: 'secret',
      }),
    })
  })

  it('returns tenant choices when the backend requires a school selection', async () => {
    const fetcher = vi.fn().mockResolvedValue(
      new Response(
        JSON.stringify({
          kind: 'tenant_required',
          error: 'tenant_required',
          tenant_choices: [
            {
              tenant_id: 'school_1',
              tenant_slug: 'alpha',
              tenant_name: 'Alpha School',
            },
          ],
        }),
        {
          status: 409,
        },
      ),
    )

    await expect(
      loginWithPassword(
        {
          email: 'teacher@school.edu',
          password: 'secret',
        },
        fetcher,
      ),
    ).resolves.toEqual({
      kind: 'tenant_required',
      message: 'tenant_required',
      tenant_choices: [
        {
          tenant_id: 'school_1',
          tenant_slug: 'alpha',
          tenant_name: 'Alpha School',
        },
      ],
    })
  })

  it('does not expose internal session persistence failures', async () => {
    const fetcher = vi.fn().mockResolvedValue(
      new Response('insert session: timeout: context deadline exceeded', {
        status: 400,
      }),
    )

    await expect(
      loginWithPassword(
        {
          email: 'teacher@school.edu',
          password: 'secret',
        },
        fetcher,
      ),
    ).rejects.not.toThrow(/insert session|context deadline/u)
  })

  it('builds the Google start URL with the current safe next path', () => {
    expect(buildGoogleLoginURL('/dashboard')).toBe(
      '/api/auth/google/start?next=%2Fdashboard',
    )
  })

  it('does not send external OAuth return targets to the backend start route', () => {
    expect(buildGoogleLoginURL('https://example.com')).toBe(
      '/api/auth/google/start',
    )
    expect(buildGoogleLoginURL('//example.com')).toBe('/api/auth/google/start')
    expect(buildGoogleLoginURL('/')).toBe('/api/auth/google/start')
    expect(buildGoogleLoginURL('/login')).toBe('/api/auth/google/start')
  })
})

describe('acceptInvite', () => {
  it('posts invite activation with cookie credentials and no browser token storage', async () => {
    const session = {
      expires_at: '2026-05-08T00:00:00Z',
      user: {
        user_id: 'parent_1',
        role: 'parent',
      },
    }
    const fetcher = vi.fn().mockResolvedValue(
      new Response(JSON.stringify(session), {
        status: 200,
      }),
    )

    await expect(
      acceptInvite(
        {
          token: 'invite-token',
          name: 'Parent One',
          password: 'strong-pass-1',
        },
        fetcher,
      ),
    ).resolves.toEqual(session)

    expect(fetcher).toHaveBeenCalledWith('/api/auth/invitations/accept', {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
      },
      credentials: 'include',
      cache: 'no-store',
      body: JSON.stringify({
        token: 'invite-token',
        name: 'Parent One',
        password: 'strong-pass-1',
      }),
    })
  })

  it('fails closed when activation returns a malformed session', async () => {
    const fetcher = vi.fn().mockResolvedValue(
      new Response(JSON.stringify({ user: { role: 'parent' } }), {
        status: 200,
      }),
    )

    await expect(
      acceptInvite(
        {
          token: 'invite-token',
          name: 'Parent One',
          password: 'strong-pass-1',
        },
        fetcher,
      ),
    ).rejects.toMatchObject({
      name: 'AuthContractError',
      message: 'Invalid invite activation response',
    })
  })
})

describe('logout', () => {
  it('clears the cookie-backed session through the auth API', async () => {
    const fetcher = vi
      .fn()
      .mockResolvedValue(new Response(null, { status: 204 }))

    await expect(logout(fetcher)).resolves.toBeUndefined()

    expect(fetcher).toHaveBeenCalledWith('/api/auth/logout', {
      method: 'POST',
      credentials: 'include',
      cache: 'no-store',
    })
  })

  it('surfaces logout failures', async () => {
    const fetcher = vi.fn().mockResolvedValue(
      new Response(JSON.stringify({ error: 'logout failed' }), {
        status: 500,
      }),
    )

    await expect(logout(fetcher)).rejects.toThrow('logout failed')
  })
})
