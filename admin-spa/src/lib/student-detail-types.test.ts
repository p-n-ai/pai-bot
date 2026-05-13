import { describe, expect, it } from 'vitest'

import { isStudentConversations, isStudentDetail } from './student-detail-types'

export const studentDetailFixture = {
  student: {
    id: 'student_1',
    name: 'Alya',
    external_id: 'telegram_1',
    channel: 'telegram',
    form: 'Form 1',
    created_at: '2026-05-08T00:00:00Z',
  },
  progress: [
    {
      topic_id: 'linear-equations',
      mastery_score: 0.72,
      ease_factor: 2.5,
      interval_days: 3,
      next_review_at: null,
      last_studied_at: '2026-05-08T00:00:00Z',
    },
  ],
  streak: {
    current: 3,
    longest: 8,
    total_xp: 120,
  },
} as const

export const studentConversationFixture = [
  {
    id: 'message_1',
    timestamp: '2026-05-08T00:00:00Z',
    role: 'student',
    text: 'How do I solve x + 2 = 5?',
  },
  {
    id: 'message_2',
    timestamp: '2026-05-08T00:01:00Z',
    role: 'assistant',
    text: 'Subtract 2 from both sides.',
  },
] as const

describe('student detail type guards', () => {
  it('accepts student detail and conversation contracts', () => {
    expect(isStudentDetail(studentDetailFixture)).toBe(true)
    expect(isStudentConversations(studentConversationFixture)).toBe(true)
  })

  it('rejects malformed student detail payloads', () => {
    expect(
      isStudentDetail({
        ...studentDetailFixture,
        streak: {
          current: '3',
        },
      }),
    ).toBe(false)
    expect(isStudentConversations([{ role: 'teacher' }])).toBe(false)
  })
})
