import { describe, expect, it } from 'vitest'

import { readPublicStatusSnapshot } from './public-status-types'

describe('readPublicStatusSnapshot', () => {
  it('parses a coherent operational response', () => {
    expect(
      readPublicStatusSnapshot({
        status: 'ok',
        components: [
          { id: 'application', status: 'operational' },
          { id: 'ai', status: 'operational' },
        ],
      }),
    ).toEqual({
      status: 'ok',
      components: [
        { id: 'application', status: 'operational' },
        { id: 'ai', status: 'operational' },
      ],
    })
  })

  it.each([
    {
      status: 'ok',
      components: [{ id: 'ai', status: 'operational' }],
    },
    {
      status: 'ok',
      components: [
        { id: 'application', status: 'operational' },
        { id: 'application', status: 'operational' },
      ],
    },
    {
      status: 'ok',
      components: [{ id: 'application', status: 'unavailable' }],
    },
    {
      status: 'degraded',
      components: [{ id: 'application', status: 'operational' }],
    },
  ])('rejects an invalid or contradictory response', (payload) => {
    expect(readPublicStatusSnapshot(payload)).toBeNull()
  })
})
