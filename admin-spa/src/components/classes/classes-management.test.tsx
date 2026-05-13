/**
 * @vitest-environment jsdom
 */
import '@testing-library/jest-dom/vitest'

import { cleanup, render, screen } from '@testing-library/react'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'

import { ClassesContent } from './classes-management'
import type { ClassesState } from './classes-management'
import type * as AdminApi from '@/lib/admin-api'

const getGroupDetailMock = vi.hoisted(() => vi.fn())

vi.mock('@/lib/admin-api', async (importOriginal) => {
  const actual = await importOriginal<typeof AdminApi>()

  return {
    ...actual,
    getGroupDetail: getGroupDetailMock,
  }
})

describe('ClassesContent', () => {
  afterEach(() => {
    cleanup()
  })

  beforeEach(() => {
    getGroupDetailMock.mockReset()
  })

  it('auto-selects the first loaded class when no class is selected', async () => {
    getGroupDetailMock.mockResolvedValue({
      ...classGroup,
      members: [],
    })

    render(
      <ClassesContent
        onCreated={vi.fn()}
        onSelectClass={vi.fn()}
        selectedClassID=''
        state={readyState}
      />,
    )

    expect(await screen.findByText('Selected class')).toBeInTheDocument()
    expect(screen.getAllByText('Form 1 Algebra A').length).toBeGreaterThan(0)
    expect(getGroupDetailMock).toHaveBeenCalledWith('class-1')
  })

  it('shows assigned topic progress for the selected class', async () => {
    getGroupDetailMock.mockResolvedValue({
      ...classGroup,
      members: [],
    })

    render(
      <ClassesContent
        onCreated={vi.fn()}
        onSelectClass={vi.fn()}
        selectedClassID='class-1'
        state={readyState}
      />,
    )

    expect(screen.getAllByText('Classes').length).toBeGreaterThan(0)
    expect(screen.getByText('Members')).toBeInTheDocument()
    expect(screen.getByText('Active learners')).toBeInTheDocument()
    expect(screen.getByText('Avg mastery')).toBeInTheDocument()
    expect(screen.getByRole('heading', { name: 'Classes' })).toBeInTheDocument()
    expect(
      screen.getByText('Select a class to view its roster and join code.'),
    ).toBeInTheDocument()
    expect(
      screen.getAllByText('KSSM Form 1, 1 learner').length,
    ).toBeGreaterThan(0)
    expect(screen.getByText('Mathematics')).toBeInTheDocument()
    expect(screen.getAllByText('KSSM Form 1').length).toBeGreaterThan(0)
    expect(screen.getByText('Mon, Wed, Fri')).toBeInTheDocument()
    expect(
      screen.getByText(
        'Share this code with students. They join via /join ALG-F1A in the bot.',
      ),
    ).toBeInTheDocument()
    expect(await screen.findByText('Member roster')).toBeInTheDocument()
    expect(
      screen.getByText('Class membership and mastery overview.'),
    ).toBeInTheDocument()
    expect(await screen.findByText('Assigned topics')).toBeInTheDocument()
    expect(screen.getByText('Algebraic Expressions')).toBeInTheDocument()
    expect(screen.getByText('76%')).toBeInTheDocument()
    expect(
      screen.getByRole('button', {
        name: 'Assign topics to class (coming soon)',
      }),
    ).toBeDisabled()
  })
})

const classGroup = {
  id: 'class-1',
  join_code: 'ALG-F1A',
  member_count: 1,
  name: 'Form 1 Algebra A',
  cadence: 'Mon, Wed, Fri',
  subject: 'Mathematics',
  syllabus: 'KSSM Form 1',
  type: 'class',
} as const

const readyState: ClassesState = {
  error: null,
  groups: [classGroup],
  status: 'ready',
}
