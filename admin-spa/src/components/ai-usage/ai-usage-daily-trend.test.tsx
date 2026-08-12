/**
 * @vitest-environment jsdom
 */
import '@testing-library/jest-dom/vitest'

import { cleanup, render, screen, within } from '@testing-library/react'
import { afterEach, describe, expect, it } from 'vitest'

import { AIUsageDailyTrend } from './ai-usage-daily-trend'
import type { AIUsageView } from '@/lib/ai-usage-view'

afterEach(() => {
  cleanup()
})

describe('AIUsageDailyTrend', () => {
  it('renders source-admin daily token trend rows', () => {
    render(<AIUsageDailyTrend view={viewWithDailyUsage} />)

    const section = screen.getByRole('region', { name: 'Daily token trend' })

    expect(
      within(section).getByRole('heading', { name: 'Daily token trend' }),
    ).toBeInTheDocument()
    expect(
      within(section).getByText(
        'Compare daily token use across the current reporting period.',
      ),
    ).toBeInTheDocument()
    expect(screen.getByText('May 1')).toBeInTheDocument()
    expect(screen.getByText('600 tokens')).toBeInTheDocument()
    expect(screen.getByText('May 2')).toBeInTheDocument()
    expect(screen.getByText('900 tokens')).toBeInTheDocument()
  })

  it('renders the source-admin empty state when no daily trend exists', () => {
    render(<AIUsageDailyTrend view={viewWithoutDailyUsage} />)

    const section = screen.getByRole('region', { name: 'Daily token trend' })

    expect(
      within(section).getByRole('heading', {
        name: 'No daily token trend yet',
      }),
    ).toBeInTheDocument()
    expect(
      screen.getByText(
        'Daily totals will appear after this school starts using AI features.',
      ),
    ).toBeInTheDocument()
  })
})

const viewWithDailyUsage: AIUsageView = {
  averageTokensPerLearner: null,
  budgetLabel: 'No token budget configured',
  budgetLimit: null,
  budgetRemaining: null,
  budgetWindowLabel: 'No active budget period',
  budgetUnit: 'none',
  dailyTrendPeak: 900,
  dailyUsage: [
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
  hasDailyTrend: true,
  monthlyCost: null,
  providers: [],
  topProvider: null,
  totalMessages: 3,
  totalTokens: 1500,
  usdBudgetLimit: null,
}

const viewWithoutDailyUsage: AIUsageView = {
  ...viewWithDailyUsage,
  dailyUsage: [],
  hasDailyTrend: false,
}
