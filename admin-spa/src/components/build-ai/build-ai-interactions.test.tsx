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
  within,
} from '@testing-library/react'
import { afterEach, describe, expect, it } from 'vitest'

import { BuildAIPage } from './build-ai-page'
import type { BuildAIPageKey } from '@/lib/build-ai-search'

function Harness({ initialPage }: { initialPage: BuildAIPageKey }) {
  const [page, setPage] = useState(initialPage)
  return <BuildAIPage onPageChange={setPage} page={page} />
}

afterEach(cleanup)

describe('Build AI state transitions', () => {
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
      'Published version 3 remains available and no classes changed',
    )
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

  it('opens a dedicated narrow-screen preview and preserves its state after Back', () => {
    render(<Harness initialPage='curriculum' />)
    fireEvent.change(screen.getByLabelText('Approved scenario'), {
      target: { value: 'ratio' },
    })
    fireEvent.click(
      screen.getByRole('button', { name: 'Open full-screen preview' }),
    )
    const preview = screen.getByTestId('narrow-screen-preview')
    expect(preview).toBeVisible()
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
    fireEvent.click(
      screen.getByRole('button', { name: 'Open full-screen preview' }),
    )
    expect(
      within(screen.getByTestId('narrow-screen-preview')).getByText(
        'Current local preview',
      ),
    ).toBeVisible()
  })
})
