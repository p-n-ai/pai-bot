import { describe, expect, it } from 'vitest'

import { isParentSummary } from './parent-summary-types'

export const parentSummaryFixture = {
  parent: {
    id: 'parent_1',
    name: 'Parent One',
    email: 'parent@example.com',
    child_ids: ['student_1'],
    created_at: '2026-05-08T00:00:00Z',
  },
  child: {
    id: 'student_1',
    name: 'Alya',
    external_id: 'telegram_1',
    channel: 'telegram',
    form: 'Form 1',
    created_at: '2026-05-08T00:00:00Z',
  },
  streak: {
    current: 3,
    longest: 8,
    total_xp: 120,
  },
  weekly_stats: {
    days_active: 3,
    messages_exchanged: 18,
    quizzes_completed: 2,
    needs_review_count: 1,
  },
  mastery: [
    {
      topic_id: 'linear-equations',
      mastery_score: 0.72,
      ease_factor: 2.5,
      interval_days: 3,
      next_review_at: null,
      last_studied_at: '2026-05-08T00:00:00Z',
    },
  ],
  encouragement: {
    headline: 'Alya kept the streak alive.',
    text: 'Praise the consistency and ask for one short review.',
  },
} as const

describe('isParentSummary', () => {
  it('accepts the parent summary API contract', () => {
    expect(isParentSummary(parentSummaryFixture)).toBe(true)
  })

  it('rejects malformed parent summary payloads', () => {
    expect(
      isParentSummary({
        ...parentSummaryFixture,
        weekly_stats: {
          days_active: '3',
        },
      }),
    ).toBe(false)
  })
})
