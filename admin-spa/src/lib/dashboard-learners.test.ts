import { describe, expect, it } from 'vitest'

import { getDashboardLearners } from './dashboard-learners'
import type { ClassProgress } from './dashboard-types'

const progress: ClassProgress = {
  students: [
    {
      id: 'on_track',
      name: 'Alya',
      topics: { algebra: 0.8, geometry: 0.7 },
    },
    {
      id: 'unmeasured',
      name: 'Zara',
      topics: {},
    },
    {
      id: 'attention',
      name: 'Hakim',
      topics: { algebra: 0.4, geometry: 0.2 },
    },
  ],
  topic_ids: ['algebra', 'geometry'],
}

describe('getDashboardLearners', () => {
  it('orders attention first and gives each learner a contextual next action', () => {
    expect(
      getDashboardLearners(progress).map((learner) => ({
        id: learner.student.id,
        nextAction: learner.nextAction,
        status: learner.status,
      })),
    ).toEqual([
      {
        id: 'attention',
        nextAction: 'Review Geometry',
        status: 'attention',
      },
      {
        id: 'unmeasured',
        nextAction: 'Check in after their first activity',
        status: 'unmeasured',
      },
      {
        id: 'on_track',
        nextAction: 'Open progress and plan the next stretch',
        status: 'on-track',
      },
    ])
  })

  it('searches learner names and filters by progress status', () => {
    expect(getDashboardLearners(progress, '  HAK  ')[0]?.student.id).toBe(
      'attention',
    )
    expect(
      getDashboardLearners(progress, '', 'on-track').map(
        (learner) => learner.student.id,
      ),
    ).toEqual(['on_track'])
    expect(
      getDashboardLearners(progress, '', 'attention').map(
        (learner) => learner.student.id,
      ),
    ).toEqual(['attention'])
    expect(
      getDashboardLearners(progress, '', 'unmeasured').map(
        (learner) => learner.student.id,
      ),
    ).toEqual(['unmeasured'])
  })

  it('keeps the 50 percent boundary on track and preserves topic order on ties', () => {
    const boundaryProgress: ClassProgress = {
      students: [
        {
          id: 'boundary',
          name: 'Mira',
          topics: { algebra: 0.5, geometry: 0.5 },
        },
      ],
      topic_ids: ['geometry', 'algebra'],
    }

    expect(getDashboardLearners(boundaryProgress)[0]).toMatchObject({
      averageMastery: 50,
      status: 'on-track',
      weakestTopicID: 'geometry',
    })
  })
})
