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
import { afterEach, describe, expect, it, vi } from 'vitest'
import type { ReactNode } from 'react'
import type * as TanStackRouter from '@tanstack/react-router'

import type { ClassProgress } from '@/lib/dashboard-types'
import { getNudgeSuccessMessage } from '@/lib/dashboard-nudge-copy'
import { DashboardPageView } from '@/components/dashboard/dashboard-page-view'
import { DashboardReady } from '@/components/dashboard/dashboard-ready'

vi.mock('@tanstack/react-router', async (importOriginal) => {
  const actual = await importOriginal<typeof TanStackRouter>()

  return {
    ...actual,
    Link: ({
      children,
      className,
      params,
      to,
    }: {
      children: ReactNode
      className?: string
      params?: { id?: string }
      to: string
    }) => (
      <a className={className} href={to.replace('$id', params?.id ?? '')}>
        {children}
      </a>
    ),
  }
})

afterEach(() => {
  cleanup()
})

const progress: ClassProgress = {
  students: [
    {
      id: 'student_1',
      name: 'Alya',
      topics: {
        algebra: 0.8,
      },
    },
  ],
  topic_ids: ['algebra'],
}

const progressWithAttention: ClassProgress = {
  students: [
    {
      id: 'student_1',
      name: 'Alya',
      topics: {
        algebra: 0.8,
        geometry: 0.9,
      },
    },
    {
      id: 'student_2',
      name: 'Hakim',
      topics: {
        algebra: 0.3,
        geometry: 0.4,
      },
    },
  ],
  topic_ids: ['algebra', 'geometry'],
}

const progressWithMissingScore: ClassProgress = {
  students: [
    {
      id: 'student_1',
      name: 'Alya',
      topics: {
        algebra: 0.8,
      },
    },
  ],
  topic_ids: ['algebra', 'geometry'],
}

const progressWithoutScores: ClassProgress = {
  students: [
    {
      id: 'student_1',
      name: 'Alya',
      topics: {},
    },
  ],
  topic_ids: ['algebra'],
}

const progressWithLongTopic: ClassProgress = {
  students: [
    {
      id: 'student_1',
      name: 'Alya',
      topics: {
        'form-one-linear-equations-with-two-variables': 0.8,
      },
    },
  ],
  topic_ids: ['form-one-linear-equations-with-two-variables'],
}

const emptyHeatmapProgress: ClassProgress = {
  students: [],
  topic_ids: [],
}

const readyDashboardState = {
  status: 'ready',
  result: {
    progress,
    source: 'live',
  },
  error: null,
} as const

const loadingDashboardState = {
  status: 'loading',
  progress: null,
  error: null,
} as const

const errorDashboardState = {
  status: 'error',
  progress: null,
  error: 'Backend offline',
} as const

const selectableClasses = [
  {
    id: 'class-1',
    name: 'Form 1 Algebra',
    type: 'class',
    join_code: 'ALG123',
    member_count: 2,
    closed: false,
  },
  {
    id: 'class-closed',
    name: 'Archived Algebra',
    type: 'class',
    join_code: 'OLD123',
    member_count: 1,
    closed: true,
  },
] as const

const readyLeaderboardState = {
  status: 'ready',
  entries: [
    {
      user_id: 'student_2',
      user_name: 'Hakim',
      mastery_gain: 0.12,
      rank: 1,
    },
    {
      user_id: 'student_1',
      user_name: 'Alya',
      mastery_gain: -0.03,
      rank: 2,
    },
  ],
} as const

const leaderboardErrorState = {
  status: 'error',
  message: 'Leaderboard offline',
} as const

describe('DashboardReady', () => {
  it('selects active classes and keeps closed classes out of the dashboard scope', () => {
    const onSelectClass = vi.fn()

    render(
      <DashboardPageView
        classes={selectableClasses}
        nudgeMessage=''
        onCloseStudent={vi.fn()}
        onNudge={vi.fn()}
        onSelectClass={onSelectClass}
        onSelectStudent={vi.fn()}
        selectedStudentID={undefined}
        sendingStudentID=''
        state={readyDashboardState}
      />,
    )

    const selector = screen.getByLabelText('Class')
    expect(
      within(selector).getByRole('option', { name: 'All learners' }),
    ).toBeInTheDocument()
    expect(
      within(selector).getByRole('option', { name: 'Form 1 Algebra' }),
    ).toBeInTheDocument()
    expect(
      within(selector).queryByRole('option', { name: 'Archived Algebra' }),
    ).not.toBeInTheDocument()

    fireEvent.change(selector, { target: { value: 'class-1' } })
    expect(onSelectClass).toHaveBeenCalledWith('class-1')
  })

  it('renders weekly leaders and opens the existing learner detail action', () => {
    const onSelectStudent = vi.fn()

    render(
      <DashboardPageView
        leaderboardState={readyLeaderboardState}
        nudgeMessage=''
        onCloseStudent={vi.fn()}
        onNudge={vi.fn()}
        onSelectStudent={onSelectStudent}
        selectedStudentID={undefined}
        sendingStudentID=''
        state={readyDashboardState}
      />,
    )

    expect(screen.getByText('+12 pts')).toBeInTheDocument()
    expect(screen.getByText('-3 pts')).toBeInTheDocument()
    fireEvent.click(screen.getByRole('button', { name: 'Hakim' }))
    expect(onSelectStudent).toHaveBeenCalledWith('student_2')
  })

  it('keeps progress visible when leaderboard loading fails and retries locally', () => {
    const onRetryLeaderboard = vi.fn()

    render(
      <DashboardPageView
        leaderboardState={leaderboardErrorState}
        nudgeMessage=''
        onCloseStudent={vi.fn()}
        onNudge={vi.fn()}
        onRetryLeaderboard={onRetryLeaderboard}
        onSelectStudent={vi.fn()}
        selectedStudentID={undefined}
        sendingStudentID=''
        state={readyDashboardState}
      />,
    )

    expect(screen.getByText('Alya')).toBeInTheDocument()
    expect(screen.getByRole('alert')).toHaveTextContent('Leaderboard offline')
    fireEvent.click(screen.getByRole('button', { name: 'Try again' }))
    expect(onRetryLeaderboard).toHaveBeenCalledOnce()
  })

  it('renders the source-admin dashboard page heading', () => {
    render(
      <DashboardPageView
        nudgeMessage=''
        onCloseStudent={vi.fn()}
        onNudge={vi.fn()}
        onSelectStudent={vi.fn()}
        selectedStudentID={undefined}
        sendingStudentID=''
        state={readyDashboardState}
      />,
    )

    expect(screen.getByRole('heading', { name: 'Today' })).toBeInTheDocument()
    expect(
      screen.getByText(
        'Welcome back. Start with the learners who need you most today.',
      ),
    ).toBeInTheDocument()
  })

  it('renders source-admin dashboard loading skeleton and error copy', () => {
    const { rerender } = render(
      <DashboardPageView
        nudgeMessage=''
        onCloseStudent={vi.fn()}
        onNudge={vi.fn()}
        onSelectStudent={vi.fn()}
        selectedStudentID={undefined}
        sendingStudentID=''
        state={loadingDashboardState}
      />,
    )

    expect(screen.getByRole('status')).toHaveTextContent(
      'Loading class progress',
    )
    expect(screen.getByTestId('dashboard-skeleton')).toBeInTheDocument()

    rerender(
      <DashboardPageView
        nudgeMessage=''
        onCloseStudent={vi.fn()}
        onNudge={vi.fn()}
        onSelectStudent={vi.fn()}
        selectedStudentID={undefined}
        sendingStudentID=''
        state={errorDashboardState}
      />,
    )

    expect(screen.getByRole('heading', { name: 'Class progress unavailable' }))
    expect(screen.getByText('Backend offline')).toBeInTheDocument()
  })

  it('opens each student in the dashboard detail route', () => {
    const onSelectStudent = vi.fn()

    render(
      <DashboardReady
        nudgeMessage=''
        onNudge={vi.fn()}
        onSelectStudent={onSelectStudent}
        progress={progress}
        sendingStudentID=''
      />,
    )

    fireEvent.click(screen.getByRole('button', { name: 'Open Alya progress' }))

    expect(onSelectStudent).toHaveBeenCalledWith('student_1')
  })

  it('renders the source-admin heatmap row and student drilldown affordance', () => {
    render(
      <DashboardReady
        nudgeMessage=''
        onNudge={vi.fn()}
        progress={progress}
        sendingStudentID=''
      />,
    )

    const heatmap = screen.getByLabelText('Learner progress')
    const row = within(heatmap).getByRole('row', { name: /Alya/ })
    const button = within(row).getByRole('button', {
      name: 'Open Alya progress',
    })

    expect(row).toBeInTheDocument()
    expect(button.querySelector('svg')).toHaveAttribute('aria-hidden', 'true')
    expect(within(button).getByRole('img')).toHaveAccessibleName(
      'No attention flag',
    )
  })

  it('marks learners below attention threshold in the heatmap', () => {
    render(
      <DashboardReady
        nudgeMessage=''
        onNudge={vi.fn()}
        progress={progressWithAttention}
        sendingStudentID=''
      />,
    )

    const hakim = screen.getByRole('button', {
      name: 'Open Hakim progress',
    })

    expect(within(hakim).getByRole('img')).toHaveAccessibleName(
      '35% average mastery',
    )
  })

  it('orders attention first and shows contextual next actions', () => {
    render(
      <DashboardReady
        nudgeMessage=''
        onNudge={vi.fn()}
        progress={progressWithAttention}
        sendingStudentID=''
      />,
    )

    const heatmap = screen.getByLabelText('Learner progress')
    const learnerRows = within(heatmap).getAllByRole('row').slice(1)

    expect(learnerRows[0]).toHaveTextContent('Hakim')
    expect(learnerRows[0]).toHaveTextContent('Review Algebra')
    expect(learnerRows[1]).toHaveTextContent('Alya')
    expect(learnerRows[1]).toHaveTextContent(
      'Open progress and plan the next stretch',
    )
  })

  it('searches and filters learner progress', () => {
    render(
      <DashboardReady
        nudgeMessage=''
        onNudge={vi.fn()}
        progress={progressWithAttention}
        sendingStudentID=''
      />,
    )

    fireEvent.change(screen.getByLabelText('Search learners'), {
      target: { value: 'Alya' },
    })

    expect(screen.getByText('Showing 1 of 2')).toBeInTheDocument()
    expect(
      screen.queryByRole('button', { name: 'Open Hakim progress' }),
    ).not.toBeInTheDocument()

    fireEvent.change(screen.getByLabelText('Progress status'), {
      target: { value: 'attention' },
    })

    expect(
      screen.getByRole('heading', { name: 'No learners match your search' }),
    ).toBeInTheDocument()
    expect(
      screen.getByText(
        'Change the name or progress filter, or clear the search field.',
      ),
    ).toBeInTheDocument()
  })

  it('renders the source-admin mastery heatmap surface heading', () => {
    render(
      <DashboardReady
        nudgeMessage=''
        onNudge={vi.fn()}
        progress={progress}
        sendingStudentID=''
      />,
    )

    const heatmap = screen.getByLabelText('Learner progress')

    expect(
      within(heatmap).getByRole('heading', { name: 'Learner progress' }),
    ).toBeInTheDocument()
    expect(
      within(heatmap).getByText(
        'Start with learners who need support, then open progress or send a nudge.',
      ),
    ).toBeInTheDocument()
    expect(within(heatmap).getByRole('table')).toBeInTheDocument()
  })

  it('renders the source-admin heatmap surface for empty progress', () => {
    render(
      <DashboardReady
        nudgeMessage=''
        onNudge={vi.fn()}
        progress={emptyHeatmapProgress}
        sendingStudentID=''
      />,
    )

    const heatmap = screen.getByLabelText('Learner progress')

    expect(
      within(heatmap).getByRole('heading', { name: 'Learner progress' }),
    ).toBeInTheDocument()
    expect(
      within(heatmap).getByRole('heading', { name: 'No class progress yet' }),
    ).toBeInTheDocument()
    expect(
      within(heatmap).getByText(
        'Topic scores will appear after students begin practising.',
      ),
    ).toBeInTheDocument()
    expect(
      screen.getByText(
        'Learner activity will shape the next recommended review.',
      ),
    ).toBeInTheDocument()
  })

  it('keeps long source-admin topic headers bounded with full-label access', () => {
    render(
      <DashboardReady
        nudgeMessage=''
        onNudge={vi.fn()}
        progress={progressWithLongTopic}
        sendingStudentID=''
      />,
    )

    const heatmap = screen.getByLabelText('Learner progress')
    const topicHeader = within(heatmap).getByText(
      'Form One Linear Equations With Two Variables',
    )

    expect(topicHeader).toHaveAttribute('data-slot', 'tooltip-trigger')
    expect(topicHeader).toHaveAttribute('type', 'button')
  })

  it('renders source-admin mastery score chips and missing-score fallback', () => {
    render(
      <DashboardReady
        nudgeMessage=''
        onNudge={vi.fn()}
        progress={progressWithMissingScore}
        sendingStudentID=''
      />,
    )

    const heatmap = screen.getByLabelText('Learner progress')

    expect(within(heatmap).getAllByText('80%')).toHaveLength(2)
    expect(within(heatmap).getByText('--')).toBeInTheDocument()
  })

  it('keeps an unmeasured student average distinct from zero mastery', () => {
    render(
      <DashboardReady
        nudgeMessage=''
        onNudge={vi.fn()}
        progress={progressWithoutScores}
        sendingStudentID=''
      />,
    )

    const heatmap = screen.getByLabelText('Learner progress')

    expect(
      within(heatmap).getByLabelText('No average mastery data'),
    ).toHaveTextContent('--')
    expect(
      within(heatmap).getByRole('img', { name: 'No mastery data' }),
    ).toBeInTheDocument()
  })

  it('shows source-admin dashboard summary signals', () => {
    render(
      <DashboardReady
        nudgeMessage=''
        onNudge={vi.fn()}
        progress={progressWithAttention}
        sendingStudentID=''
      />,
    )

    expect(screen.getByText('Class grade')).toBeInTheDocument()
    expect(screen.getByText('60% average mastery')).toBeInTheDocument()
    expect(screen.getByText('4 of 4 topic scores recorded')).toBeInTheDocument()
    expect(screen.getByText('1 learner needs attention')).toBeInTheDocument()
    expect(screen.getByText('Weakest: Algebra')).toBeInTheDocument()
    expect(screen.getByText('Strongest: Geometry')).toBeInTheDocument()
  })

  it('renders summary signals as source-admin stat cards', () => {
    render(
      <DashboardReady
        nudgeMessage=''
        onNudge={vi.fn()}
        progress={progressWithAttention}
        sendingStudentID=''
      />,
    )

    const learners = screen.getByText('Learners').closest('[data-slot="card"]')
    const average = screen
      .getByText('Average mastery')
      .closest('[data-slot="card"]')

    if (!(learners instanceof HTMLElement)) {
      throw new Error('Expected learners stat card to render')
    }

    if (!(average instanceof HTMLElement)) {
      throw new Error('Expected average mastery stat card to render')
    }

    expect(within(learners).getByText('2')).toBeInTheDocument()
    expect(
      within(learners).getByText('1 learner needs attention'),
    ).toBeInTheDocument()
    expect(within(average).getByText('Weakest: Algebra')).toBeInTheDocument()
    expect(within(average).getByText('Strongest: Geometry')).toBeInTheDocument()
  })

  it('uses the source-admin compact primary nudge button treatment', () => {
    render(
      <DashboardReady
        nudgeMessage=''
        onNudge={vi.fn()}
        progress={progress}
        sendingStudentID=''
      />,
    )

    const nudgeButton = screen.getByRole('button', {
      name: 'Send nudge to Alya',
    })

    expect(nudgeButton).toHaveAttribute('data-size', 'sm')
    expect(nudgeButton).toHaveAttribute('data-variant', 'default')
  })

  it('reports source-admin Telegram copy after sending a nudge', () => {
    const onNudge = vi.fn()

    render(
      <DashboardReady
        nudgeMessage='Nudge sent to Alya on Telegram.'
        onNudge={onNudge}
        progress={progress}
        sendingStudentID=''
      />,
    )

    fireEvent.click(screen.getByRole('button', { name: 'Send nudge to Alya' }))

    expect(onNudge).toHaveBeenCalledWith('student_1', 'Alya')
    expect(screen.getByRole('status')).toHaveTextContent(
      'Nudge sent to Alya on Telegram.',
    )
  })
})

describe('dashboard nudge copy', () => {
  it('matches the source-admin Telegram success message', () => {
    expect(getNudgeSuccessMessage('Alya')).toBe(
      'Nudge sent to Alya on Telegram.',
    )
  })
})
