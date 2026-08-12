/**
 * @vitest-environment jsdom
 */
import '@testing-library/jest-dom/vitest'

import { useState } from 'react'
import {
  cleanup,
  fireEvent,
  render,
  screen,
  waitFor,
  within,
} from '@testing-library/react'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'

import { BuildAIPage } from './build-ai-page'
import type { BuildAIPageKey } from '@/lib/build-ai-search'

function Harness({ initialPage }: { initialPage: BuildAIPageKey }) {
  const [page, setPage] = useState(initialPage)
  return <BuildAIPage onPageChange={setPage} page={page} />
}

class ResizeObserverStub {
  disconnect() {}
  observe() {}
  unobserve() {}
}

beforeEach(() => vi.stubGlobal('ResizeObserver', ResizeObserverStub))
afterEach(() => {
  cleanup()
  vi.unstubAllGlobals()
})

describe('Build AI state transitions', () => {
  it('creates a character inside the Draft before continuing to Curriculum', () => {
    render(<Harness initialPage='character' />)
    const preview = screen.getByRole('img', {
      name: /P&AI Tutor character preview/i,
    })

    expect(preview).toHaveAttribute('data-silhouette', 'blob')
    expect(preview).toHaveAttribute('data-expression', 'attentive')
    fireEvent.click(screen.getByRole('button', { name: 'Bean' }))
    expect(preview).toHaveAttribute('data-silhouette', 'bean')
    expect(screen.getByTestId('build-ai-live-region')).toHaveTextContent(
      'Character silhouette changed. The Draft has unsaved changes.',
    )
    fireEvent.click(screen.getByRole('button', { name: 'Thinking' }))
    expect(preview).toHaveAttribute('data-expression', 'thoughtful')
    fireEvent.click(screen.getByRole('button', { name: 'Save Draft' }))
    fireEvent.click(
      screen.getByRole('button', { name: 'Continue to Curriculum' }),
    )
    expect(screen.getByRole('heading', { name: 'Curriculum' })).toBeVisible()
  })

  it('marks a Teaching preview stale after an edit and saves the Draft', () => {
    render(<Harness initialPage='teaching' />)
    fireEvent.click(screen.getByRole('button', { name: 'Run preview' }))
    expect(screen.getByText('Current local preview')).toBeVisible()
    fireEvent.click(screen.getByLabelText('Prefer English'))
    expect(screen.getByText('Out of date')).toBeVisible()
    const save = screen.getByRole('button', { name: /Save Draft/ })
    fireEvent.click(save)
    expect(save).toBeDisabled()
    expect(screen.getByTestId('build-ai-live-region')).toHaveTextContent(
      'Existing Published versions and classes are unchanged',
    )
  })

  it('shows honest non-pass states and advances a saved Draft through tests', () => {
    render(<Harness initialPage='test' />)
    for (const status of [
      'Needs review',
      'Inconclusive',
      'Failed',
      'Cancelled',
    ]) {
      expect(screen.getByText(status)).toBeVisible()
    }
    fireEvent.click(screen.getByRole('button', { name: 'Run tests' }))
    fireEvent.click(screen.getByRole('button', { name: 'Open Publish' }))
    expect(
      screen.getByRole('heading', { name: 'Publication readiness' }),
    ).toBeVisible()
    expect(screen.getAllByText('Ready').length).toBeGreaterThanOrEqual(5)
  })

  it('confirms the class consequence before publication and preserves history', () => {
    render(<Harness initialPage='test' />)
    fireEvent.click(screen.getByRole('button', { name: 'Run tests' }))
    fireEvent.click(screen.getByRole('button', { name: 'Open Publish' }))
    fireEvent.change(screen.getByLabelText('Version note'), {
      target: { value: 'Clearer fraction explanations' },
    })
    fireEvent.click(screen.getByRole('button', { name: 'Review publication' }))
    expect(
      screen.getByText(/no class will change until a teacher applies it/i),
    ).toBeVisible()
    fireEvent.click(screen.getByRole('button', { name: 'Publish version' }))
    expect(
      screen.getAllByText(
        'Published version 4 is available. No classes changed.',
      ),
    ).toHaveLength(2)
    expect(
      screen.getByText(/Published version 4 · Latest available/),
    ).toBeVisible()
    expect(screen.getByText(/Published version 3 · Earlier/)).toBeVisible()
    expect(
      screen.getByRole('heading', { name: 'Used by classes' }),
    ).toBeVisible()
    expect(screen.getByLabelText('Version note')).toBeDisabled()
    expect(
      screen.getByRole('button', { name: 'Review publication' }),
    ).toBeDisabled()
    fireEvent.change(screen.getByLabelText('Version note'), {
      target: { value: 'Mutated after publication' },
    })
    expect(
      screen.getByText(/Clearer fraction explanations · published/),
    ).toBeVisible()
    expect(
      screen.queryByText(/Mutated after publication/),
    ).not.toBeInTheDocument()
    fireEvent.click(
      screen.getByRole('button', { name: 'Start Draft from this version' }),
    )
    expect(
      screen.getByText(/replaces the existing private Draft/i),
    ).toBeVisible()
    fireEvent.click(screen.getByRole('button', { name: 'Replace Draft' }))
    expect(
      screen.getByText(/Published version 4 · Latest available/),
    ).toBeVisible()
    expect(screen.getByTestId('build-ai-live-region')).toHaveTextContent(
      'Published version 4 remains available and no classes changed',
    )
    fireEvent.click(
      within(
        screen.getByRole('navigation', { name: 'Build AI destinations' }),
      ).getByRole('link', { name: 'Teaching' }),
    )
    expect(
      screen.getByLabelText('Start with one guiding question'),
    ).toBeChecked()
    expect(screen.getByLabelText('Brief')).toBeChecked()
    expect(screen.getByLabelText('Prefer English')).toBeChecked()
  })

  it('offers the authoritative Overview test action only while results are stale', () => {
    render(<Harness initialPage='overview' />)
    fireEvent.click(screen.getByRole('button', { name: 'Run tests' }))
    expect(screen.getByRole('heading', { name: 'Test tutor' })).toBeVisible()
    fireEvent.click(screen.getByRole('button', { name: 'Run tests' }))
    fireEvent.click(
      within(
        screen.getByRole('navigation', { name: 'Build AI destinations' }),
      ).getByRole('link', { name: 'Overview' }),
    )
    expect(
      screen.queryByRole('button', { name: 'Run tests' }),
    ).not.toBeInTheDocument()
    expect(
      screen.getByText(/Passed and current for Draft revision/),
    ).toBeVisible()
    expect(
      screen.getByText('Complete for the exact saved Draft.'),
    ).toBeVisible()
  })

  it('opens a dedicated narrow-screen preview and preserves its state after Back', async () => {
    render(<Harness initialPage='curriculum' />)
    fireEvent.change(screen.getByLabelText('Approved scenario'), {
      target: { value: 'ratio' },
    })
    const previewTrigger = screen.getByRole('button', {
      name: 'Open full-screen preview',
    })
    fireEvent.click(previewTrigger)
    const preview = screen.getByTestId('narrow-screen-preview')
    expect(preview).toBeVisible()
    expect(preview).toHaveAttribute('role', 'dialog')
    expect(within(preview).getByRole('button', { name: 'Back' })).toHaveFocus()
    expect(within(preview).getByLabelText('Approved scenario')).toHaveValue(
      'ratio',
    )
    fireEvent.click(
      within(preview).getByRole('button', { name: 'Run preview' }),
    )
    fireEvent.click(within(preview).getByRole('button', { name: 'Back' }))
    expect(
      screen.queryByTestId('narrow-screen-preview'),
    ).not.toBeInTheDocument()
    await waitFor(() => expect(previewTrigger).toHaveFocus())
    fireEvent.click(previewTrigger)
    const reopenedPreview = screen.getByTestId('narrow-screen-preview')
    expect(
      within(reopenedPreview).getByText('Current local preview'),
    ).toBeVisible()
    fireEvent.keyDown(reopenedPreview, { key: 'Escape' })
    expect(
      screen.queryByTestId('narrow-screen-preview'),
    ).not.toBeInTheDocument()
    await waitFor(() => expect(previewTrigger).toHaveFocus())
  })

  it('keeps saved Teaching preferences across navigation and restores them on discard', () => {
    render(<Harness initialPage='teaching' />)
    fireEvent.click(screen.getByLabelText('Start with one guiding question'))
    fireEvent.click(screen.getByRole('button', { name: /Save Draft/ }))
    fireEvent.click(
      within(
        screen.getByRole('navigation', { name: 'Build AI destinations' }),
      ).getByRole('link', { name: 'Overview' }),
    )
    fireEvent.click(
      within(
        screen.getByRole('navigation', { name: 'Build AI destinations' }),
      ).getByRole('link', { name: 'Teaching' }),
    )
    expect(
      screen.getByLabelText('Start with one guiding question'),
    ).toBeChecked()

    fireEvent.click(
      screen.getByLabelText('Start with one small worked example'),
    )
    fireEvent.click(screen.getByRole('button', { name: 'Discard changes' }))
    expect(
      screen.getByLabelText('Start with one guiding question'),
    ).toBeChecked()
  })

  it('keeps row-level test actions from bypassing an unsaved Draft', () => {
    render(<Harness initialPage='teaching' />)
    fireEvent.click(screen.getByLabelText('Prefer English'))
    fireEvent.click(
      within(
        screen.getByRole('navigation', { name: 'Build AI destinations' }),
      ).getByRole('link', { name: 'Test tutor' }),
    )

    expect(
      screen.queryByRole('button', { name: 'Rerun' }),
    ).not.toBeInTheDocument()
    const saveActions = screen.getAllByRole('button', {
      name: 'Open Teaching to save',
    })
    const rerunAction = saveActions.at(-1)
    expect(rerunAction).toBeDefined()
    if (!rerunAction) return
    fireEvent.click(rerunAction)
    expect(screen.getByRole('heading', { name: 'Teaching' })).toBeVisible()
  })

  it('completes the selected in-workspace educator review flow after tests pass', () => {
    render(<Harness initialPage='teaching' />)
    fireEvent.click(screen.getByLabelText('Prefer English'))
    fireEvent.click(screen.getByRole('button', { name: /Save Draft/ }))
    fireEvent.click(
      screen.getByRole('button', { name: 'Continue to Test tutor' }),
    )
    fireEvent.click(screen.getByRole('button', { name: 'Run tests' }))
    fireEvent.click(screen.getByRole('button', { name: 'Open Publish' }))

    const review = screen.getByRole('button', {
      name: 'Complete educator review',
    })
    expect(review).toBeEnabled()
    fireEvent.click(review)
    fireEvent.change(screen.getByLabelText('Version note'), {
      target: { value: 'Reviewed Teaching preferences' },
    })
    expect(
      screen.getByRole('button', { name: 'Review publication' }),
    ).toBeEnabled()
  })

  it('updates Overview and Activity after publication', () => {
    render(<Harness initialPage='test' />)
    fireEvent.click(screen.getByRole('button', { name: 'Run tests' }))
    fireEvent.click(screen.getByRole('button', { name: 'Open Publish' }))
    fireEvent.change(screen.getByLabelText('Version note'), {
      target: { value: 'Current published snapshot' },
    })
    fireEvent.click(screen.getByRole('button', { name: 'Review publication' }))
    fireEvent.click(screen.getByRole('button', { name: 'Publish version' }))

    fireEvent.click(
      within(
        screen.getByRole('navigation', { name: 'Build AI destinations' }),
      ).getByRole('link', { name: 'Overview' }),
    )
    expect(
      screen.getByText(
        /Published version 4 remains available.*results are passed and current/i,
      ),
    ).toBeVisible()

    fireEvent.click(
      within(
        screen.getByRole('navigation', { name: 'Build AI destinations' }),
      ).getByRole('link', { name: 'Activity' }),
    )
    expect(
      screen.getByText(/Published version 4 remains available to teachers/i),
    ).toBeVisible()
  })

  it('requires a fresh version note after saving a new Draft revision', () => {
    render(<Harness initialPage='test' />)
    fireEvent.click(screen.getByRole('button', { name: 'Run tests' }))
    fireEvent.click(screen.getByRole('button', { name: 'Open Publish' }))
    fireEvent.change(screen.getByLabelText('Version note'), {
      target: { value: 'Version four note' },
    })
    fireEvent.click(screen.getByRole('button', { name: 'Review publication' }))
    fireEvent.click(screen.getByRole('button', { name: 'Publish version' }))
    fireEvent.click(
      within(
        screen.getByRole('navigation', { name: 'Build AI destinations' }),
      ).getByRole('link', { name: 'Teaching' }),
    )
    fireEvent.click(screen.getByLabelText('Prefer English'))
    fireEvent.click(screen.getByRole('button', { name: /Save Draft/ }))
    fireEvent.click(
      screen.getByRole('button', { name: 'Continue to Test tutor' }),
    )
    fireEvent.click(screen.getByRole('button', { name: 'Run tests' }))
    fireEvent.click(screen.getByRole('button', { name: 'Open Publish' }))
    fireEvent.click(
      screen.getByRole('button', { name: 'Complete educator review' }),
    )

    expect(screen.getByLabelText('Version note')).toHaveValue('')
    expect(
      screen.getByRole('button', { name: 'Review publication' }),
    ).toBeDisabled()
  })
})
