import { describe, expect, it } from 'vitest'

import { readPublicStatusSnapshot } from './public-status-types'

describe('readPublicStatusSnapshot', () => {
  it('parses a coherent operational response', () => {
    expect(
      readPublicStatusSnapshot({
        status: 'ok',
        components: [
          { id: 'application', status: 'operational' },
          { id: 'ai_provider', status: 'operational' },
        ],
      }),
    ).toEqual({
      status: 'ok',
      components: [
        { id: 'application', status: 'operational' },
        { id: 'ai_provider', status: 'operational' },
      ],
    })
  })

  it.each([
    {
      status: 'ok',
      components: [{ id: 'ai_provider', status: 'operational' }],
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
      components: [
        { id: 'application', status: 'unavailable' },
        { id: 'ai_provider', status: 'operational' },
      ],
    },
    {
      status: 'degraded',
      components: [
        { id: 'application', status: 'operational' },
        { id: 'ai_provider', status: 'operational' },
      ],
    },
  ])('rejects an invalid or contradictory response', (payload) => {
    expect(readPublicStatusSnapshot(payload)).toBeNull()
  })
})
