/**
 * @vitest-environment jsdom
 */
import '@testing-library/jest-dom/vitest'

import { cleanup, render, screen, within } from '@testing-library/react'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'

import { StudentDetailPanel } from './student-detail-panel'
import type * as AdminAPI from '@/lib/admin-api'
import {
  studentConversationFixture,
  studentDetailFixture,
} from '@/lib/student-detail-types.test'

const getStudentDetail = vi.hoisted(() => vi.fn())
const getStudentConversations = vi.hoisted(() => vi.fn())

vi.mock('@/lib/admin-api', async (importOriginal) => {
  const actual = await importOriginal<typeof AdminAPI>()

  return {
    ...actual,
    getStudentConversations,
    getStudentDetail,
  }
})

describe('StudentDetailPanel', () => {
  beforeEach(() => {
    getStudentDetail.mockReset()
    getStudentConversations.mockReset()
  })

  afterEach(() => {
    cleanup()
  })

  it('renders profile, progress, and recent conversations', async () => {
    getStudentDetail.mockResolvedValue(studentDetailWithStruggleFixture)
    getStudentConversations.mockResolvedValue(studentConversationFixture)

    render(<StudentDetailPanel studentID='student_1' />)

    expect(await screen.findByText('Alya')).toBeInTheDocument()
    expect(screen.getByText('3 days')).toBeInTheDocument()
    expect(screen.getByText('8 days')).toBeInTheDocument()
    expect(screen.getByText('Linear Equations')).toBeInTheDocument()
    expect(screen.getByText('Struggle areas')).toBeInTheDocument()
    const strugglePanel = screen
      .getByText('Struggle areas')
      .closest('[data-slot="card"]')
    if (!(strugglePanel instanceof HTMLElement)) {
      throw new Error('Expected struggle areas panel to render')
    }
    expect(
      within(strugglePanel).getByText(/Fractions\s+42%/),
    ).toBeInTheDocument()
    expect(screen.getByText('Mastery radar')).toBeInTheDocument()
    expect(screen.getByText('72% mastery')).toBeInTheDocument()
    expect(screen.getByText('Activity grid')).toBeInTheDocument()
    expect(screen.getByTitle('May 8: 2 messages')).toBeInTheDocument()
    expect(screen.getByLabelText('May 8: 2 messages')).toBeInTheDocument()
    expect(
      screen.getAllByText('08 May 2026, 00:00 UTC').length,
    ).toBeGreaterThan(0)
    expect(screen.getByText('How do I solve x + 2 = 5?')).toBeInTheDocument()
  })

  it('shows a hard load error without empty-state fallthrough', async () => {
    getStudentDetail.mockRejectedValue(new Error('offline'))
    getStudentConversations.mockResolvedValue([])

    render(<StudentDetailPanel studentID='student_1' />)

    expect(
      await screen.findByText('Student detail unavailable'),
    ).toBeInTheDocument()
    expect(screen.queryByText('No topic progress yet')).not.toBeInTheDocument()
  })
})

const studentDetailWithStruggleFixture = {
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
