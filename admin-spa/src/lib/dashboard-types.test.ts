import { describe, expect, it } from 'vitest'

import { readClassProgress } from './dashboard-types'

describe('readClassProgress', () => {
  it('reads class progress payloads', () => {
    const progress = {
      students: [
        {
          id: 'student_1',
          name: 'Alya',
          topics: {
            algebra: 0.8,
          },
        },
      ],
      topic_ids: ['algebra'],
    }

    expect(readClassProgress(progress)).toEqual(progress)
  })

  it('normalizes empty backend slices', () => {
    expect(
      readClassProgress({
        students: null,
        topic_ids: null,
      }),
    ).toEqual({
      students: [],
      topic_ids: [],
    })
  })

  it('rejects malformed progress payloads', () => {
    expect(readClassProgress({ students: {}, topic_ids: [] })).toBeNull()
    expect(readClassProgress({ students: [], topic_ids: [123] })).toBeNull()
  })
})
