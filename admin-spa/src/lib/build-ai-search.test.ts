import { describe, expect, it } from 'vitest'

import { parseBuildAISearch } from './build-ai-search'

describe('Build AI search state', () => {
  it.each([
    'overview',
    'curriculum',
    'teaching',
    'test',
    'publish',
    'activity',
  ])('keeps the %s destination', (page) => {
    expect(parseBuildAISearch({ page })).toEqual({ page })
  })

  it('falls back to Overview for unknown boundary input', () => {
    expect(parseBuildAISearch({ page: 'secrets' })).toEqual({
      page: 'overview',
    })
    expect(parseBuildAISearch({})).toEqual({ page: 'overview' })
  })
})
