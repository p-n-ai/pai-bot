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
      { id: 'ai', status: 'operational' },
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
  expect(screen.getByText('AI services')).toBeInTheDocument()
  expect(screen.getAllByText('Operational')).toHaveLength(2)
  expect(
    screen.getByRole('link', { name: 'View JSON status' }),
  ).toHaveAttribute('href', '/health/api')
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
  expect(screen.getByText('Unknown')).toBeInTheDocument()
  expect(screen.queryByText('private detail')).not.toBeInTheDocument()
})
