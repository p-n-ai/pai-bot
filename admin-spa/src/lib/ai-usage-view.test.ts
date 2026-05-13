import { describe, expect, it } from 'vitest'

import { formatBudgetAmount, getAIUsageView } from './ai-usage-view'

describe('getAIUsageView', () => {
  it('builds the AI usage dashboard view model', () => {
    expect(
      getAIUsageView({
        total_messages: 10,
        total_input_tokens: 4000,
        total_output_tokens: 1000,
        budget_limit_tokens: 10000,
        budget_used_tokens: 5000,
        budget_remaining_tokens: 5000,
        per_student_average_tokens: 250,
        providers: [
          {
            provider: 'openai',
            model: 'gpt-4.1-mini',
            messages: 10,
            input_tokens: 4000,
            output_tokens: 1000,
            total_tokens: 5000,
          },
        ],
      }),
    ).toMatchObject({
      totalTokens: 5000,
      totalMessages: 10,
      budgetLabel: 'Within token budget',
      budgetRemaining: 5000,
      budgetLimit: 10000,
      averageTokensPerLearner: 250,
      topProvider: {
        provider: 'openai',
      },
    })
  })

  it('falls back to the USD budget status when token budget is absent', () => {
    expect(
      getAIUsageView({
        total_messages: 10,
        total_input_tokens: 4000,
        total_output_tokens: 1000,
        monthly_cost_usd: 85,
        budget_limit_usd: 100,
        providers: [],
      }),
    ).toMatchObject({
      budgetLabel: 'Near budget limit',
      budgetRemaining: 15,
      budgetLimit: 100,
      monthlyCost: 85,
      usdBudgetLimit: 100,
    })
  })

  it('keeps USD provider summary values even when token budget takes priority', () => {
    expect(
      getAIUsageView({
        total_messages: 10,
        total_input_tokens: 4000,
        total_output_tokens: 1000,
        budget_limit_tokens: 10000,
        budget_used_tokens: 5000,
        budget_remaining_tokens: 5000,
        monthly_cost_usd: 12.5,
        budget_limit_usd: 80,
        providers: [],
      }),
    ).toMatchObject({
      budgetLabel: 'Within token budget',
      budgetLimit: 10000,
      budgetUnit: 'tokens',
      monthlyCost: 12.5,
      usdBudgetLimit: 80,
    })
  })

  it('formats budget values by their source unit', () => {
    expect(formatBudgetAmount(1500, 'tokens')).toBe('1.5K')
    expect(formatBudgetAmount(85, 'usd')).toBe('$85.00')
    expect(formatBudgetAmount(null, 'none')).toBe('--')
  })

  it('builds source-admin daily trend metadata', () => {
    expect(
      getAIUsageView({
        total_messages: 3,
        total_input_tokens: 1200,
        total_output_tokens: 300,
        daily_usage: [
          {
            date: '2026-05-01T00:00:00Z',
            messages: 1,
            tokens: 600,
          },
          {
            date: '2026-05-02T00:00:00Z',
            messages: 2,
            tokens: 900,
          },
        ],
        providers: [],
      }),
    ).toMatchObject({
      dailyTrendPeak: 900,
      dailyUsage: [
        {
          date: '2026-05-01T00:00:00Z',
          tokens: 600,
        },
        {
          date: '2026-05-02T00:00:00Z',
          tokens: 900,
        },
      ],
      hasDailyTrend: true,
    })
  })

  it('builds the source-admin budget window label', () => {
    expect(
      getAIUsageView({
        total_messages: 3,
        total_input_tokens: 1200,
        total_output_tokens: 300,
        budget_period_start: '2026-05-01',
        budget_period_end: '2026-05-31',
        providers: [],
      }),
    ).toMatchObject({
      budgetWindowLabel: 'May 1 to May 31',
    })
  })
})
