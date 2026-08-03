/**
 * @vitest-environment jsdom
 */
import '@testing-library/jest-dom/vitest'

import { cleanup, fireEvent, render, screen } from '@testing-library/react'
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
    vi.unstubAllGlobals()
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
        canInviteAdults
        onCreated={vi.fn()}
        onRetry={vi.fn()}
        onSelectClass={vi.fn()}
        selectedClassID=''
        state={readyState}
      />,
    )

    expect(await screen.findByText('Current class')).toBeInTheDocument()
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
        canInviteAdults
        onCreated={vi.fn()}
        onRetry={vi.fn()}
        onSelectClass={vi.fn()}
        selectedClassID='class-1'
        state={readyState}
      />,
    )

    expect(screen.getAllByText('Classes').length).toBeGreaterThan(0)
    expect(screen.getByText('Students')).toBeInTheDocument()
    expect(
      screen.getByRole('heading', { name: 'School classes' }),
    ).toBeInTheDocument()
    expect(
      screen.getByText('Choose a class to manage its roster.'),
    ).toBeInTheDocument()
    expect(
      screen.getAllByText('KSSM Form 1, 1 learner').length,
    ).toBeGreaterThan(0)
    expect(screen.getByText('Mathematics')).toBeInTheDocument()
    expect(screen.getAllByText('KSSM Form 1').length).toBeGreaterThan(0)
    expect(screen.getByText('Mon, Wed, Fri')).toBeInTheDocument()
    expect(
      screen.getByText(
        (_, element) =>
          element?.textContent === 'Students send /join ALG-F1A to the bot.',
      ),
    ).toBeInTheDocument()
    expect(await screen.findByText('Student roster')).toBeInTheDocument()
    expect(
      screen.getByText(
        'Students who joined this class and their current mastery.',
      ),
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

  it('reveals secondary actions only when requested', () => {
    getGroupDetailMock.mockResolvedValue({
      ...classGroup,
      members: [],
    })

    render(
      <ClassesContent
        canInviteAdults
        onCreated={vi.fn()}
        onRetry={vi.fn()}
        onSelectClass={vi.fn()}
        selectedClassID='class-1'
        state={readyState}
      />,
    )

    expect(
      screen.queryByRole('heading', { name: 'Create a class' }),
    ).not.toBeInTheDocument()
    fireEvent.click(screen.getByRole('button', { name: 'Create class' }))
    expect(
      screen.getByRole('heading', { name: 'Create a class' }),
    ).toBeInTheDocument()
    fireEvent.click(screen.getByRole('button', { name: 'Invite an adult' }))
    expect(
      screen.getByRole('heading', { name: 'Invite an adult' }),
    ).toBeInTheDocument()
    expect(
      screen.queryByRole('heading', { name: 'Create a class' }),
    ).not.toBeInTheDocument()
  })

  it('does not offer admin-only invitations to teachers', () => {
    getGroupDetailMock.mockResolvedValue({
      ...classGroup,
      members: [],
    })

    render(
      <ClassesContent
        canInviteAdults={false}
        onCreated={vi.fn()}
        onRetry={vi.fn()}
        onSelectClass={vi.fn()}
        selectedClassID='class-1'
        state={readyState}
      />,
    )

    expect(
      screen.queryByRole('button', { name: 'Invite an adult' }),
    ).not.toBeInTheDocument()
    expect(
      screen.getByRole('button', { name: 'Create class' }),
    ).toBeInTheDocument()
  })

  it('puts class creation directly in the empty state', () => {
    render(
      <ClassesContent
        canInviteAdults
        onCreated={vi.fn()}
        onRetry={vi.fn()}
        onSelectClass={vi.fn()}
        selectedClassID=''
        state={emptyState}
      />,
    )

    expect(
      screen.getByRole('heading', { name: 'Create your first class' }),
    ).toBeInTheDocument()
    expect(screen.getAllByLabelText('Class name')).toHaveLength(1)
    expect(
      screen.queryByRole('button', { name: 'Create class', expanded: false }),
    ).not.toBeInTheDocument()
    expect(
      screen.getByRole('button', { name: 'Invite an adult' }),
    ).toBeInTheDocument()
  })

  it('reports unavailable clipboard access instead of throwing', async () => {
    getGroupDetailMock.mockResolvedValue({
      ...classGroup,
      members: [],
    })
    vi.stubGlobal('navigator', {})

    render(
      <ClassesContent
        canInviteAdults
        onCreated={vi.fn()}
        onRetry={vi.fn()}
        onSelectClass={vi.fn()}
        selectedClassID='class-1'
        state={readyState}
      />,
    )

    fireEvent.click(screen.getByRole('button', { name: 'Copy join code' }))

    expect(
      await screen.findByText('Could not copy the join code'),
    ).toHaveAttribute('role', 'alert')
  })

  it('clears join-code copy feedback when the selected class changes', async () => {
    const writeText = vi.fn().mockResolvedValue(undefined)
    vi.stubGlobal('navigator', { clipboard: { writeText } })
    getGroupDetailMock.mockResolvedValue({
      ...classGroup,
      members: [],
    })

    const { rerender } = render(
      <ClassesContent
        canInviteAdults
        onCreated={vi.fn()}
        onRetry={vi.fn()}
        onSelectClass={vi.fn()}
        selectedClassID='class-1'
        state={readyStateWithTwoClasses}
      />,
    )

    fireEvent.click(screen.getByRole('button', { name: 'Copy join code' }))
    expect(await screen.findByText('Join code copied')).toBeInTheDocument()
    expect(writeText).toHaveBeenCalledWith('ALG-F1A')

    rerender(
      <ClassesContent
        canInviteAdults
        onCreated={vi.fn()}
        onRetry={vi.fn()}
        onSelectClass={vi.fn()}
        selectedClassID='class-2'
        state={readyStateWithTwoClasses}
      />,
    )

    expect(screen.queryByText('Join code copied')).not.toBeInTheDocument()
    expect(screen.getByText('SCI-F2B')).toBeInTheDocument()
  })

  it('offers a retry when classes cannot load', () => {
    const onRetry = vi.fn()

    render(
      <ClassesContent
        canInviteAdults
        onCreated={vi.fn()}
        onRetry={onRetry}
        onSelectClass={vi.fn()}
        selectedClassID=''
        state={errorState}
      />,
    )

    fireEvent.click(screen.getByRole('button', { name: 'Try again' }))
    expect(onRetry).toHaveBeenCalledOnce()
  })
})

const classGroup = {
  closed: false,
  id: 'class-1',
  join_code: 'ALG-F1A',
  member_count: 1,
  name: 'Form 1 Algebra A',
  cadence: 'Mon, Wed, Fri',
  subject: 'Mathematics',
  syllabus: 'KSSM Form 1',
  type: 'class',
} as const

const secondClassGroup = {
  ...classGroup,
  id: 'class-2',
  join_code: 'SCI-F2B',
  name: 'Form 2 Science B',
} as const

const readyState: ClassesState = {
  error: null,
  groups: [classGroup],
  status: 'ready',
}

const readyStateWithTwoClasses: ClassesState = {
  error: null,
  groups: [classGroup, secondClassGroup],
  status: 'ready',
}

const emptyState: ClassesState = {
  error: null,
  groups: [],
  status: 'ready',
}

const errorState: ClassesState = {
  error: 'Network unavailable',
  groups: [],
  status: 'error',
}
