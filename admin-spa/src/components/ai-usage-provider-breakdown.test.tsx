/**
 * @vitest-environment jsdom
 */
import '@testing-library/jest-dom/vitest'

import { cleanup, render, screen, within } from '@testing-library/react'
import { afterEach, describe, expect, it } from 'vitest'

import { AIUsageProviderBreakdown } from './ai-usage-provider-breakdown'
import type { AIUsageView } from '@/lib/ai-usage-view'

afterEach(() => {
  cleanup()
})

describe('AIUsageProviderBreakdown', () => {
  it('renders the source-admin provider breakdown surface with summary panels', () => {
    render(<AIUsageProviderBreakdown view={view} />)

    const section = screen.getByRole('region', { name: 'Provider breakdown' })

    expect(
      within(section).getByRole('heading', { name: 'Provider breakdown' }),
    ).toBeInTheDocument()
    expect(
      within(section).getByText(
        'Provider and model mix for the recorded AI traffic in this workspace.',
      ),
    ).toBeInTheDocument()
    expect(within(section).getByRole('table')).toBeInTheDocument()
    expect(within(section).getByText('Monthly cost')).toBeInTheDocument()
    expect(within(section).getByText('$12.50')).toBeInTheDocument()
    expect(within(section).getByText('Budget cap (USD)')).toBeInTheDocument()
    expect(within(section).getByText('$80.00')).toBeInTheDocument()
    expect(within(section).getByText('Top provider')).toBeInTheDocument()
    expect(within(section).getAllByText('openai').length).toBeGreaterThan(0)
  })

  it('keeps the source-admin empty provider copy inside the breakdown surface', () => {
    render(<AIUsageProviderBreakdown view={emptyView} />)

    const section = screen.getByRole('region', { name: 'Provider breakdown' })

    expect(
      within(section).getByText('No provider traffic recorded'),
    ).toBeInTheDocument()
    expect(
      within(section).getByText(
        'Provider rows will populate after the first successful AI requests for this tenant.',
      ),
    ).toBeInTheDocument()
    expect(within(section).getByText('None yet')).toBeInTheDocument()
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

const emptyView: AIUsageView = {
  ...view,
  monthlyCost: null,
  providers: [],
  topProvider: null,
  usdBudgetLimit: null,
}
