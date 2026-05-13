/**
 * @vitest-environment jsdom
 */
import '@testing-library/jest-dom/vitest'

import { cleanup, render, screen, within } from '@testing-library/react'
import { afterEach, describe, expect, it, vi } from 'vitest'

import { AIUsageBudgetSection } from './ai-usage-budget-section'
import type { AIUsageSummary } from '@/lib/ai-usage-types'
import type { AIUsageView } from '@/lib/ai-usage-view'

afterEach(() => {
  cleanup()
})

describe('AIUsageBudgetSection', () => {
  it('renders the source-admin token budget surface with window panels and editor', () => {
    render(
      <AIUsageBudgetSection
        canManageBudget
        onSaved={vi.fn()}
        usage={usage}
        view={view}
      />,
    )

    const section = screen.getByRole('region', { name: 'Token budget window' })

    expect(
      within(section).getByRole('heading', { name: 'Token budget window' }),
    ).toBeInTheDocument()
    expect(
      within(section).getByText(
        'Budget thresholds and remaining allowance for the active tenant-wide token window.',
      ),
    ).toBeInTheDocument()
    expect(within(section).getByText('Window')).toBeInTheDocument()
    expect(within(section).getByText('Apr 1 to Apr 30')).toBeInTheDocument()
    expect(within(section).getByText('Limit')).toBeInTheDocument()
    expect(within(section).getByText('250K')).toBeInTheDocument()
    expect(within(section).getByText('Remaining')).toBeInTheDocument()
    expect(within(section).getByText('200K')).toBeInTheDocument()
    expect(
      within(section).getByRole('button', { name: 'Save token budget' }),
    ).toBeInTheDocument()
  })

  it('keeps source-admin locked-budget copy inside the budget surface', () => {
    render(
      <AIUsageBudgetSection
        canManageBudget={false}
        onSaved={vi.fn()}
        usage={usage}
        view={view}
      />,
    )

    const section = screen.getByRole('region', { name: 'Token budget window' })

    expect(
      within(section).getByText('Budget changes require admin access.'),
    ).toBeInTheDocument()
    expect(
      within(section).queryByRole('button', { name: 'Save token budget' }),
    ).not.toBeInTheDocument()
  })
})

const usage: AIUsageSummary = {
  total_messages: 10,
  total_input_tokens: 4000,
  total_output_tokens: 1000,
  budget_limit_tokens: 250000,
  budget_remaining_tokens: 200000,
  budget_period_start: '2026-04-01',
  budget_period_end: '2026-04-30',
  providers: [],
}

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
  topProvider: null,
  totalMessages: 10,
  totalTokens: 5000,
  usdBudgetLimit: null,
}
