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

import { BuildAIPage } from './build-ai-page'
import type { BuildAIPageKey } from '@/lib/build-ai-search'

afterEach(cleanup)

function renderPage(page: BuildAIPageKey = 'overview') {
  const onPageChange = vi.fn()
  const view = render(<BuildAIPage onPageChange={onPageChange} page={page} />)
  return { onPageChange, ...view }
}

describe('Build AI illustrative workspace', () => {
  it('labels synthetic evidence and exposes the seven creation destinations', () => {
    renderPage()

    expect(screen.getByText(/Synthetic illustrative data only/i)).toBeVisible()
    expect(screen.getByRole('heading', { name: 'P&AI Tutor' })).toBeVisible()
    const navigation = screen.getByRole('navigation', {
      name: 'Build AI destinations',
    })
    expect(
      within(navigation)
        .getAllByRole('link')
        .map((link) => link.textContent),
    ).toEqual([
      'Overview',
      'P-Bot character',
      'Curriculum',
      'Teaching',
      'Test tutor',
      'Publish',
      'ActivityMonitor',
    ])
  })

  it('requests query-backed navigation without taking over browser history', () => {
    const { onPageChange } = renderPage()
    const navigation = screen.getByRole('navigation', {
      name: 'Build AI destinations',
    })

    fireEvent.click(within(navigation).getByRole('link', { name: 'Teaching' }))

    expect(onPageChange).toHaveBeenCalledWith('teaching')
  })

  it('keeps monitoring honest when production telemetry is unavailable', () => {
    renderPage('activity')

    expect(screen.getByRole('heading', { name: 'Activity' })).toBeVisible()
    expect(screen.getByText('Monitoring incomplete')).toBeVisible()
    expect(screen.getByText(/Tutor health cannot be confirmed/i)).toBeVisible()
    expect(screen.queryByText(/^Healthy$/i)).not.toBeInTheDocument()
  })

  it('labels unavailable curriculum mutation instead of exposing a dead action', () => {
    renderPage('curriculum')

    expect(
      screen.getByRole('button', { name: 'Change curriculum unavailable' }),
    ).toBeDisabled()
    expect(
      screen.getByText(/does not connect to a curriculum-change contract/i),
    ).toBeVisible()
  })
})
