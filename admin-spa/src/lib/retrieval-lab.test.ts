import { describe, expect, it, vi } from 'vitest'

import {
  buildRetrievalSearchPayload,
  getRetrievalRunCount,
  normalizeRetrievalError,
  searchRetrieval,
} from './retrieval-lab'

describe('buildRetrievalSearchPayload', () => {
  it('trims query fields and omits empty filters', () => {
    expect(
      buildRetrievalSearchPayload({
        collection_ids: ' curriculum:math-f1, curriculum:math-f2 ',
        include_inactive: true,
        kinds: '',
        limit: '7',
        metadata: '{"form":"1"}',
        query: ' linear equations ',
        repeats: '2',
        source_types: 'curriculum',
      }),
    ).toEqual({
      collection_ids: ['curriculum:math-f1', 'curriculum:math-f2'],
      include_inactive: true,
      kinds: undefined,
      limit: 7,
      metadata: { form: '1' },
      query: 'linear equations',
      source_types: ['curriculum'],
    })
  })

  it('rejects metadata that is not a string-valued object', () => {
    expect(() =>
      buildRetrievalSearchPayload({
        collection_ids: '',
        include_inactive: false,
        kinds: '',
        limit: '5',
        metadata: '{"form":1}',
        query: 'linear',
        repeats: '1',
        source_types: '',
      }),
    ).toThrow('Metadata must be a JSON object with string values')
  })
})

describe('getRetrievalRunCount', () => {
  it('keeps repeat runs at one or more', () => {
    expect(getRetrievalRunCount('0')).toBe(1)
    expect(getRetrievalRunCount('3')).toBe(3)
  })
})

describe('normalizeRetrievalError', () => {
  it('turns backend auth failures into user-facing copy', () => {
    expect(normalizeRetrievalError('401 Unauthorized: expired token')).toBe(
      'Your admin session expired. Sign in again to keep testing retrieval.',
    )
  })
})

describe('searchRetrieval', () => {
  it('posts typed retrieval payloads to the backend admin route', async () => {
    const fetcher = vi.fn().mockResolvedValue(
      new Response(
        JSON.stringify([
          {
            document: {
              id: 'doc_1',
              kind: 'topic_card',
              title: 'Linear equations',
            },
            excerpt: 'Solve linear equations one step at a time.',
            high_signal_terms: 3,
            matched_terms: 2,
            score: 12.5,
          },
        ]),
      ),
    )

    await expect(
      searchRetrieval(
        {
          include_inactive: false,
          limit: 5,
          query: 'linear',
        },
        fetcher,
      ),
    ).resolves.toHaveLength(1)

    expect(fetcher).toHaveBeenCalledWith('/api/admin/retrieval/search', {
      body: JSON.stringify({
        include_inactive: false,
        limit: 5,
        query: 'linear',
      }),
      cache: 'no-store',
      credentials: 'include',
      headers: {
        'Content-Type': 'application/json',
      },
      method: 'POST',
    })
  })
})
