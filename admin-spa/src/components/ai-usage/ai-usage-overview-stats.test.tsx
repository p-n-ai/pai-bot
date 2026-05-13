/**
 * @vitest-environment jsdom
 */
import '@testing-library/jest-dom/vitest'

import { render, screen } from '@testing-library/react'
import { describe, expect, it } from 'vitest'

import { AIUsageOverviewStats } from './ai-usage-overview-stats'
import type { AIUsageView } from '@/lib/ai-usage-view'

describe('AIUsageOverviewStats', () => {
  it('renders source-admin overview metric notes', () => {
    render(<AIUsageOverviewStats view={view} />)

    expect(screen.getByText('Tokens')).toBeInTheDocument()
    expect(screen.getByText('5K')).toBeInTheDocument()
    expect(screen.getByText('openai top source')).toBeInTheDocument()
    expect(screen.getByText('Messages')).toBeInTheDocument()
    expect(screen.getByText('Current aggregate window')).toBeInTheDocument()
    expect(screen.getByText('Budget')).toBeInTheDocument()
    expect(screen.getByText('Apr 1 to Apr 30')).toBeInTheDocument()
    expect(screen.getByText('Per learner')).toBeInTheDocument()
    expect(screen.getByText('Token average')).toBeInTheDocument()
  })
})

const view: AIUsageView = {
  averageTokensPerLearner: null,
  budgetLabel: 'Within token budget',
  budgetLimit: 250000,
  budgetRemaining: 200000,
  budgetWindowLabel: 'Apr 1 to Apr 30',
  budgetUnit: 'tokens',
  dailyTrendPeak: 0,
  dailyUsage: [],
  hasDailyTrend: false,
  monthlyCost: null,
  providers: [],
  topProvider: {
    provider: 'openai',
    model: 'gpt-4.1-mini',
    messages: 10,
    input_tokens: 4000,
    output_tokens: 1000,
    total_tokens: 5000,
  },
  totalMessages: 10,
  totalTokens: 5000,
  usdBudgetLimit: null,
}
