import { describe, expect, it } from 'vitest'

import { getDashboardSummary } from './dashboard-summary'

describe('getDashboardSummary', () => {
  it('computes the dashboard overview from class progress', () => {
    expect(
      getDashboardSummary({
        topic_ids: ['linear-equations', 'fractions'],
        students: [
          {
            id: 'student_1',
            name: 'Alya',
            topics: {
              'linear-equations': 0.8,
              fractions: 0.6,
            },
          },
        ],
      }),
    ).toEqual({
      attentionCount: 0,
      studentCount: 1,
      topicCount: 2,
      trackedScores: 2,
      averageMastery: 70,
      hasHeatmap: true,
      coveragePercent: 100,
      strongestTopic: {
        score: 80,
        topicID: 'linear-equations',
      },
      weakestTopic: {
        score: 60,
        topicID: 'fractions',
      },
    })
  })
})
