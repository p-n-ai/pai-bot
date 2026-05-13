/**
 * @vitest-environment jsdom
 */
import '@testing-library/jest-dom/vitest'

import {
  cleanup,
  fireEvent,
  render,
  screen,
  waitFor,
} from '@testing-library/react'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'

import { AIUsageBudgetEditor } from './ai-usage-budget-editor'
import type { AIUsageSummary } from '@/lib/ai-usage-types'
import type * as AdminAPI from '@/lib/admin-api'

const upsertTokenBudgetWindow = vi.hoisted(() => vi.fn())

vi.mock('@/lib/admin-api', async (importOriginal) => {
  const actual = await importOriginal<typeof AdminAPI>()

  return {
    ...actual,
    upsertTokenBudgetWindow,
  }
})

const usage: AIUsageSummary = {
  total_messages: 10,
  total_input_tokens: 4000,
  total_output_tokens: 1000,
  budget_limit_tokens: 250000,
  budget_period_start: '2026-04-01',
  budget_period_end: '2026-04-30',
  providers: [],
}

const usageWithoutBudget = {
  ...usage,
  budget_limit_tokens: null,
}

describe('AIUsageBudgetEditor', () => {
  beforeEach(() => {
    upsertTokenBudgetWindow.mockReset()
  })

  afterEach(() => {
    cleanup()
  })

  it('hides the editor from non-admin users', () => {
    render(
      <AIUsageBudgetEditor
        canManageBudget={false}
        onSaved={vi.fn()}
        usage={usage}
      />,
    )

    expect(
      screen.getByText('Budget changes require admin access.'),
    ).toBeInTheDocument()
    expect(
      screen.queryByRole('button', { name: 'Save token budget' }),
    ).not.toBeInTheDocument()
  })

  it('submits the token budget window and returns the refreshed usage summary', async () => {
    const updatedUsage = {
      ...usage,
      budget_limit_tokens: 300000,
      budget_remaining_tokens: 295000,
    }
    const onSaved = vi.fn()
    upsertTokenBudgetWindow.mockResolvedValue(updatedUsage)

    render(
      <AIUsageBudgetEditor
        canManageBudget
        onSaved={onSaved}
        usage={usageWithoutBudget}
      />,
    )

    fireEvent.change(screen.getByLabelText('Token limit'), {
      target: { value: '300000' },
    })
    fireEvent.click(screen.getByRole('button', { name: 'Save token budget' }))

    await waitFor(() => {
      expect(upsertTokenBudgetWindow).toHaveBeenCalledWith({
        budget_tokens: 300000,
        period_start: '2026-04-01',
        period_end: '2026-04-30',
      })
    })
    expect(onSaved).toHaveBeenCalledWith(updatedUsage)
  })
})
