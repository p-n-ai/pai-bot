import { describe, expect, it, vi } from 'vitest'

import { PublicStatusError, readPublicStatus } from './public-status-client'

describe('readPublicStatus', () => {
  it('reads the no-store public status contract', async () => {
    const fetcher = vi.fn().mockResolvedValue(
      new Response(
        JSON.stringify({
          status: 'ok',
          components: [
            { id: 'application', status: 'operational' },
            { id: 'ai_provider', status: 'operational' },
          ],
        }),
        {
          status: 200,
          headers: { 'Content-Type': 'application/json' },
        },
      ),
    )

    await expect(readPublicStatus(fetcher)).resolves.toEqual({
      status: 'ok',
      components: [
        { id: 'application', status: 'operational' },
        { id: 'ai_provider', status: 'operational' },
      ],
    })
    expect(fetcher).toHaveBeenCalledWith('/health/status', {
      cache: 'no-store',
      headers: { Accept: 'application/json' },
    })
  })

  it('returns a safe error for a failed response', async () => {
    const fetcher = vi
      .fn()
      .mockResolvedValue(new Response('private detail', { status: 503 }))

    await expect(readPublicStatus(fetcher)).rejects.toEqual(
      new PublicStatusError('Public status is unavailable'),
    )
  })
})
