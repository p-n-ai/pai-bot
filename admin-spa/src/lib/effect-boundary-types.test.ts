import { describe, expect, it } from 'vitest'

import { isAIUsageSummary } from './ai-usage-types'
import { isGroupDetail, isGroupRecord } from './group-types'
import {
  isLearningStreak,
  isProgressItem,
  isStudentProfile,
} from './learner-types'

describe('Effect boundary contracts', () => {
  it('accepts complete AI usage with nullable optional metrics', () => {
    expect(
      isAIUsageSummary({
        total_messages: 3,
        total_input_tokens: 120,
        total_output_tokens: 45,
        providers: [
          {
            provider: 'openai',
            model: 'gpt-5',
            messages: 3,
            input_tokens: 120,
            output_tokens: 45,
            total_tokens: 165,
          },
        ],
        monthly_cost_usd: null,
      }),
    ).toBe(true)
  })

  it('rejects incomplete AI provider usage', () => {
    expect(
      isAIUsageSummary({
        total_messages: 3,
        total_input_tokens: 120,
        total_output_tokens: 45,
        providers: [{ provider: 'openai', model: 'gpt-5' }],
      }),
    ).toBe(false)
  })

  it('distinguishes group summaries from complete group details', () => {
    const group = {
      id: 'group_1',
      name: 'Math',
      type: 'class',
      join_code: 'ABC123',
      member_count: 1,
      closed: false,
    }

    expect(isGroupRecord(group)).toBe(true)
    expect(isGroupRecord({ ...group, closed: undefined })).toBe(false)
    expect(isGroupDetail(group)).toBe(false)
    expect(
      isGroupDetail({
        ...group,
        members: [
          {
            id: 'member_1',
            name: 'Student',
            role: 'member',
            channel: 'telegram',
            mastery: 0.5,
          },
        ],
      }),
    ).toBe(true)
  })

  it('accepts learner contracts including nullable review dates', () => {
    expect(
      isStudentProfile({
        id: 'student_1',
        name: 'Student',
        external_id: 'tg_1',
        channel: 'telegram',
        form: 'Form 1',
        created_at: '2026-07-28T00:00:00Z',
      }),
    ).toBe(true)
    expect(
      isProgressItem({
        topic_id: 'algebra',
        mastery_score: 0.5,
        ease_factor: 2.5,
        interval_days: 3,
        next_review_at: null,
        last_studied_at: null,
      }),
    ).toBe(true)
    expect(isLearningStreak({ current: 2, longest: 5, total_xp: 80 })).toBe(
      true,
    )
  })
})
