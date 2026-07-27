import { describe, expect, it } from 'vitest'

import { buildStudentViewModel, getActivityTone } from './student-detail-view'
import {
  studentConversationFixture,
  studentDetailFixture,
} from './student-detail-types.test'

describe('buildStudentViewModel', () => {
  it('prioritizes the weakest struggle and summarizes recent activity', () => {
    const detail = {
      ...studentDetailFixture,
      progress: [
        ...studentDetailFixture.progress,
        {
          ease_factor: 2.1,
          interval_days: 1,
          last_studied_at: '2026-05-07T00:00:00Z',
          mastery_score: 0.42,
          next_review_at: '2026-05-09T00:00:00Z',
          topic_id: 'fractions',
        },
      ],
    }

    const view = buildStudentViewModel(
      detail,
      [...studentConversationFixture],
      new Date('2026-05-08T12:00:00Z'),
    )

    expect(view.status).toEqual({
      description: '1 topic needs targeted support.',
      label: 'Needs attention',
      tone: 'attention',
    })
    expect(view.struggleAreas[0]?.topic_id).toBe('fractions')
    expect(view.strengthAreas[0]?.topic_id).toBe('linear-equations')
    expect(view.recommendation.title).toBe('Review Fractions next')
    expect(view.recentMessageCount).toBe(2)
    expect(view.activeDays).toBe(1)
    expect(view.latestActivityAt).toBe('2026-05-08T00:01:00Z')
    expect(view.conversations.map((conversation) => conversation.id)).toEqual([
      'message_2',
      'message_1',
    ])
  })

  it('gives a useful starting action when no learning data exists', () => {
    const view = buildStudentViewModel(
      { ...studentDetailFixture, progress: [] },
      [],
    )

    expect(view.status.label).toBe('Getting started')
    expect(view.recommendation).toEqual({
      description:
        'Ask the learner to complete a short diagnostic conversation so strengths and support needs can be identified.',
      title: 'Start with a guided check-in',
    })
    expect(view.latestActivityAt).toBeNull()
    expect(view.recentMessageCount).toBe(0)
  })

  it('does not treat stale conversation history as recent activity', () => {
    const view = buildStudentViewModel(
      {
        ...studentDetailFixture,
        progress: [...studentDetailFixture.progress],
      },
      [...studentConversationFixture],
      new Date('2026-05-30T12:00:00Z'),
    )

    expect(view.recentMessageCount).toBe(0)
    expect(view.activeDays).toBe(0)
    expect(view.latestActivityAt).toBe('2026-05-08T00:01:00Z')
    expect(view.status).toEqual({
      description:
        'Mastery is developing, but no recent tutoring activity is recorded.',
      label: 'Check in',
      tone: 'neutral',
    })
  })

  it('provides contrasting text for each activity intensity', () => {
    for (const level of [0, 1, 2, 3, 4]) {
      expect(getActivityTone(level)).toContain('text-')
      expect(getActivityTone(level)).toContain('dark:text-')
    }
  })
})
