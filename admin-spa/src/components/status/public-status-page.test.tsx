/**
 * @vitest-environment jsdom
 */
import '@testing-library/jest-dom/vitest'

import { cleanup, render, screen } from '@testing-library/react'
import { afterEach, expect, it, vi } from 'vitest'

import { PublicStatusPage } from './public-status-page'

const readPublicStatus = vi.hoisted(() => vi.fn())

vi.mock('@/lib/public-status-client', () => ({
  readPublicStatus,
}))

afterEach(() => {
  cleanup()
  readPublicStatus.mockReset()
})

it('shows the application and AI availability without requiring admin auth', async () => {
  readPublicStatus.mockResolvedValue({
    status: 'ok',
    components: [
      { id: 'application', status: 'operational' },
      { id: 'ai_provider', status: 'operational' },
    ],
  })

  render(<PublicStatusPage />)

  expect(
    await screen.findByRole('heading', {
      level: 1,
      name: 'All systems operational',
    }),
  ).toBeInTheDocument()
  expect(screen.getByText('Application API')).toBeInTheDocument()
  expect(screen.getByText('AI provider')).toBeInTheDocument()
  expect(
    screen.getByText('Primary provider response health'),
  ).toBeInTheDocument()
  expect(screen.getAllByText('Operational')).toHaveLength(2)
  expect(screen.getByRole('link', { name: 'JSON status' })).toHaveAttribute(
    'href',
    '/health/status',
  )
})

it('does not claim an outage when the status service cannot be reached', async () => {
  readPublicStatus.mockRejectedValue(new Error('private detail'))

  render(<PublicStatusPage />)

  expect(
    await screen.findByRole('heading', {
      level: 1,
      name: 'Status currently unavailable',
    }),
  ).toBeInTheDocument()
  expect(screen.getAllByText('Unknown')).toHaveLength(2)
  expect(screen.queryByText('private detail')).not.toBeInTheDocument()
})

it('shows provider degradation separately from the application', async () => {
  readPublicStatus.mockResolvedValue({
    status: 'degraded',
    components: [
      { id: 'application', status: 'operational' },
      { id: 'ai_provider', status: 'unavailable' },
    ],
  })

  render(<PublicStatusPage />)

  expect(
    await screen.findByRole('heading', {
      level: 1,
      name: 'Some systems unavailable',
    }),
  ).toBeInTheDocument()
  expect(screen.getByText('Application API')).toBeInTheDocument()
  expect(screen.getByText('AI provider')).toBeInTheDocument()
  expect(screen.getByText('Operational')).toBeInTheDocument()
  expect(screen.getByText('Unavailable')).toBeInTheDocument()
})
