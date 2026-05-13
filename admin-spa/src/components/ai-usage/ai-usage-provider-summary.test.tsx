/**
 * @vitest-environment jsdom
 */
import '@testing-library/jest-dom/vitest'

import { render, screen } from '@testing-library/react'
import { describe, expect, it } from 'vitest'

import { AIUsageProviderSummary } from './ai-usage-provider-summary'
import type { AIUsageView } from '@/lib/ai-usage-view'

describe('AIUsageProviderSummary', () => {
  it('renders source-admin provider summary panels', () => {
    render(<AIUsageProviderSummary view={view} />)

    expect(screen.getByText('Monthly cost')).toBeInTheDocument()
    expect(screen.getByText('$12.50')).toBeInTheDocument()
    expect(screen.getByText('Budget cap (USD)')).toBeInTheDocument()
    expect(screen.getByText('$80.00')).toBeInTheDocument()
    expect(screen.getByText('Top provider')).toBeInTheDocument()
    expect(screen.getByText('openai')).toBeInTheDocument()
  })

  it('renders source-admin empty provider summary fallbacks', () => {
    render(<AIUsageProviderSummary view={emptyProviderSummaryView} />)

    expect(screen.getByText('--')).toBeInTheDocument()
    expect(screen.getByText('Not set')).toBeInTheDocument()
    expect(screen.getByText('None yet')).toBeInTheDocument()
  })
})

const view: AIUsageView = {
  averageTokensPerLearner: null,
  budgetLabel: 'Within token budget',
  budgetLimit: 10000,
  budgetRemaining: 5000,
  budgetWindowLabel: 'May 1 to May 31',
  budgetUnit: 'tokens',
  dailyTrendPeak: 0,
  dailyUsage: [],
  hasDailyTrend: false,
  monthlyCost: 12.5,
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
  usdBudgetLimit: 80,
}

const emptyProviderSummaryView: AIUsageView = {
  ...view,
  monthlyCost: null,
  topProvider: null,
  usdBudgetLimit: null,
}
