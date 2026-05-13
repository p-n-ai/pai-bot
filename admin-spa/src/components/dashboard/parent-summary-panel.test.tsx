/**
 * @vitest-environment jsdom
 */
import '@testing-library/jest-dom/vitest'

import { cleanup, render, screen } from '@testing-library/react'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'

import { ParentSummaryPanel } from './parent-summary-panel'
import type * as AdminAPI from '@/lib/admin-api'
import { parentSummaryFixture } from '@/lib/parent-summary-types.test'

const getParentSummary = vi.hoisted(() => vi.fn())

vi.mock('@/lib/admin-api', async (importOriginal) => {
  const actual = await importOriginal<typeof AdminAPI>()

  return {
    ...actual,
    getParentSummary,
  }
})

describe('ParentSummaryPanel', () => {
  beforeEach(() => {
    getParentSummary.mockReset()
  })

  afterEach(() => {
    cleanup()
  })

  it('renders weekly stats, mastery, and encouragement for a parent summary', async () => {
    getParentSummary.mockResolvedValue(parentSummaryWithNextReviewFixture)

    render(<ParentSummaryPanel parentID='parent_1' />)

    expect(await screen.findByText('Alya this week')).toBeInTheDocument()
    expect(screen.getByText('3 days')).toBeInTheDocument()
    expect(screen.getByText('8 days')).toBeInTheDocument()
    expect(screen.getByLabelText('Weekly summary stats')).toBeInTheDocument()
    expect(screen.getByText('Active days')).toBeInTheDocument()
    expect(screen.getByText('Longest streak')).toBeInTheDocument()
    expect(screen.getByText('Total XP')).toBeInTheDocument()
    expect(screen.getByText('Linear Equations')).toBeInTheDocument()
    expect(
      screen.getByText('Next review 09 May 2026, 00:00 UTC'),
    ).toBeInTheDocument()
    expect(screen.getByText('Alya kept the streak alive.')).toBeInTheDocument()
  })

  it('shows load errors without falling through to empty mastery state', async () => {
    getParentSummary.mockRejectedValue(new Error('offline'))

    render(<ParentSummaryPanel parentID='parent_1' />)

    expect(
      await screen.findByText('Parent summary unavailable'),
    ).toBeInTheDocument()
    expect(screen.queryByText('No mastery data yet')).not.toBeInTheDocument()
  })
})

const parentSummaryWithNextReviewFixture = {
  ...parentSummaryFixture,
  mastery: [
    {
      ...parentSummaryFixture.mastery[0],
      next_review_at: '2026-05-09T00:00:00Z',
    },
  ],
}
