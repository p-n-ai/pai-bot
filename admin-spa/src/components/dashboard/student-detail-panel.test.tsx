/**
 * @vitest-environment jsdom
 */
import '@testing-library/jest-dom/vitest'

import {
  cleanup,
  fireEvent,
  render,
  screen,
  within,
} from '@testing-library/react'
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

  it('puts an actionable learner summary before progress and conversations', async () => {
    getStudentDetail.mockResolvedValue(studentDetailWithStruggleFixture)
    getStudentConversations.mockResolvedValue(studentConversationFixture)

    render(<StudentDetailPanel studentID='student_1' />)

    expect(await screen.findByText('Alya')).toBeInTheDocument()
    expect(screen.getByText('Needs attention')).toBeInTheDocument()
    expect(screen.getByText('Review Fractions next')).toBeInTheDocument()
    expect(screen.getByText('Strengths & struggles')).toBeInTheDocument()
    expect(screen.getByText('Recent activity')).toBeInTheDocument()
    expect(screen.getByText('3 days')).toBeInTheDocument()
    expect(screen.getByText('8 days')).toBeInTheDocument()
    expect(screen.getByText('Linear Equations')).toBeInTheDocument()
    const snapshot = screen
      .getByText('Teacher snapshot')
      .closest('[data-slot="card"]')
    if (!(snapshot instanceof HTMLElement)) {
      throw new Error('Expected teacher snapshot to render')
    }
    expect(within(snapshot).getByText(/Fractions\s+42%/)).toBeInTheDocument()
    const snapshotHeading = screen.getByText('Teacher snapshot')
    const masteryHeading = screen.getByText('Mastery overview')
    const conversationHeading = screen.getByText('Recent conversations')
    expect(
      snapshotHeading.compareDocumentPosition(masteryHeading) &
        Node.DOCUMENT_POSITION_FOLLOWING,
    ).toBeTruthy()
    expect(
      snapshotHeading.compareDocumentPosition(conversationHeading) &
        Node.DOCUMENT_POSITION_FOLLOWING,
    ).toBeTruthy()
    expect(screen.getAllByText('72%').length).toBeGreaterThan(0)
    expect(screen.getByText('14-day activity')).toBeInTheDocument()
    expect(
      screen.getByRole('region', { name: '14-day tutoring activity' }),
    ).toHaveAttribute('tabindex', '0')
    expect(
      screen.getByRole('region', { name: 'Conversation history' }),
    ).toHaveAttribute('tabindex', '0')
    expect(
      screen.getAllByText('08 May 2026, 00:00 UTC').length,
    ).toBeGreaterThan(0)
    const latestMessage = screen.getByText('Subtract 2 from both sides.')
    const firstMessage = screen.getByText('How do I solve x + 2 = 5?')
    expect(
      latestMessage.compareDocumentPosition(firstMessage) &
        Node.DOCUMENT_POSITION_FOLLOWING,
    ).toBeTruthy()
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

  it('recovers from a load error when the teacher retries', async () => {
    getStudentDetail
      .mockRejectedValueOnce(new Error('offline'))
      .mockResolvedValueOnce(studentDetailFixture)
    getStudentConversations.mockResolvedValue([])

    render(<StudentDetailPanel studentID='student_1' />)

    fireEvent.click(await screen.findByRole('button', { name: 'Try again' }))

    expect(await screen.findByText('Alya')).toBeInTheDocument()
    expect(getStudentDetail).toHaveBeenCalledTimes(2)
    expect(getStudentConversations).toHaveBeenCalledTimes(2)
  })

  it('shows actionable empty states in the sheet layout', async () => {
    getStudentDetail.mockResolvedValue({
      ...studentDetailFixture,
      progress: [],
    })
    getStudentConversations.mockResolvedValue([])

    render(<StudentDetailPanel studentID='student_1' variant='sheet' />)

    expect(await screen.findByText('Teacher snapshot')).toBeInTheDocument()
    expect(screen.getByText('Getting started')).toBeInTheDocument()
    expect(screen.getByText('Start with a guided check-in')).toBeInTheDocument()
    expect(screen.getByText('No mastery overview yet')).toBeInTheDocument()
    expect(screen.getByText('No tutoring messages yet')).toBeInTheDocument()
    expect(
      screen.getByText('Mastery overview').closest('section'),
    ).not.toHaveClass('xl:grid-cols-[0.8fr_1.2fr]')
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
