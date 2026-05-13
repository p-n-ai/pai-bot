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

describe('DashboardReady', () => {
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

    expect(
      screen.getByRole('heading', { name: 'Dashboard' }),
    ).toBeInTheDocument()
    expect(
      screen.getByText('Track who needs support today across the class.'),
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
      'Preparing class snapshot',
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

    expect(screen.getByRole('heading', { name: 'Class data unavailable' }))
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

    fireEvent.click(screen.getByRole('button', { name: /Alya/ }))

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

    const heatmap = screen.getByLabelText('Mastery heatmap')
    const row = within(heatmap).getByRole('row', { name: /Alya/ })
    const button = within(row).getByRole('button', { name: /Alya/ })

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

    const hakim = screen.getByRole('button', { name: /Hakim/ })

    expect(within(hakim).getByRole('img')).toHaveAccessibleName(
      '35% average mastery',
    )
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

    const heatmap = screen.getByLabelText('Mastery heatmap')

    expect(
      within(heatmap).getByRole('heading', { name: 'Mastery heatmap' }),
    ).toBeInTheDocument()
    expect(
      within(heatmap).getByText(
        'Students by topic with direct navigation into detail views.',
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

    const heatmap = screen.getByLabelText('Mastery heatmap')

    expect(
      within(heatmap).getByRole('heading', { name: 'Mastery heatmap' }),
    ).toBeInTheDocument()
    expect(
      within(heatmap).getByRole('heading', { name: 'No class heatmap yet' }),
    ).toBeInTheDocument()
    expect(
      within(heatmap).getByText(
        'Progress appears after students start topics.',
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

    const heatmap = screen.getByLabelText('Mastery heatmap')
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

    const heatmap = screen.getByLabelText('Mastery heatmap')

    expect(within(heatmap).getByText('80%')).toBeInTheDocument()
    expect(within(heatmap).getByText('--')).toBeInTheDocument()
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
    expect(screen.getByText('4 of 4 score slots filled')).toBeInTheDocument()
    expect(screen.getByText('1 learner need attention')).toBeInTheDocument()
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
      within(learners).getByText('1 learner need attention'),
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

    const nudgeButton = screen.getByRole('button', { name: 'Nudge' })

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

    fireEvent.click(screen.getByRole('button', { name: 'Nudge' }))

    expect(onNudge).toHaveBeenCalledWith('student_1', 'Alya')
    expect(
      screen.getByText('Nudge sent to Alya on Telegram.'),
    ).toBeInTheDocument()
  })
})

describe('dashboard nudge copy', () => {
  it('matches the source-admin Telegram success message', () => {
    expect(getNudgeSuccessMessage('Alya')).toBe(
      'Nudge sent to Alya on Telegram.',
    )
  })
})
